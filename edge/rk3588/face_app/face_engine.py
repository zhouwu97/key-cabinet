import os
import cv2
import numpy as np
import logging
import hashlib
from typing import Optional, Tuple, Dict, List
from cryptography.hazmat.primitives.ciphers.aead import AESGCM

logger = logging.getLogger("FaceEngine")

# 标准 ArcFace / MobileFaceNet 112x112 五点标准人脸参考坐标 (画面坐标系)
STANDARD_LANDMARKS_112 = np.array([
    [38.2946, 51.6963],   # 画面左眼 (viewer-left / subject-right)
    [73.5318, 51.5014],   # 画面右眼 (viewer-right / subject-left)
    [56.0252, 71.7366],   # 鼻尖 (nose tip)
    [41.5493, 92.3655],   # 画面左嘴角 (viewer-left / subject-right)
    [70.7299, 92.2041]    # 画面右嘴角 (viewer-right / subject-left)
], dtype=np.float32)

MAGIC_HEADER = b"KCFE"  # Key Cabinet Face Encryption

class FaceEngine:
    """
    RK3588 边缘人脸检测、5点人脸对齐、深度特征提取与 1:N 模板比对引擎。
    采用真实生物特征特征向量模型，并以 AES-256-GCM 强加密持久化生物特征模板。
    绝不长期持久化原始人脸照片，检测不到人脸时严禁降级返回中央画面。
    """
    def __init__(
        self,
        template_dir: str = "templates",
        min_face_size: int = 60,
        device_secret: str = "",
        model_path: Optional[str] = None,
        allow_handcrafted_fallback: bool = False
    ):
        self.template_dir = template_dir
        self.min_face_size = min_face_size
        self.model_path = model_path
        self.allow_handcrafted_fallback = allow_handcrafted_fallback
        self.last_landmarks: Optional[np.ndarray] = None

        if not device_secret:
            if not self.allow_handcrafted_fallback:
                raise RuntimeError(
                    "生产模式必须配置机柜安全密钥 device_secret 用于人脸特征 AES-256-GCM 模板加密！"
                    "严禁在未配置密钥的情况下启动生产人脸识别引擎。"
                )
            self.device_secret = "dev_default_secret_key_testing_only"
        else:
            self.device_secret = device_secret

        os.makedirs(self.template_dir, exist_ok=True)

        # 1. 深度检测器与特征提取器初始化
        self.detector_yn = None
        self.face_cascade = None
        self.onnx_session = None
        self.rknn_session = None

        self._init_detector()
        self._init_recognizer()

        # 2. 内存特征库：student_no -> 512-d normalized float32
        self.gallery: Dict[str, np.ndarray] = {}
        self.load_templates()

    def _init_detector(self):
        """初始化人脸检测器 (优先 YuNet，备用 Haar)"""
        # 若指定或当前存在 YuNet ONNX 模型
        yunet_paths = [
            self.model_path if (self.model_path and "yunet" in self.model_path.lower()) else None,
            os.path.join(os.path.dirname(__file__), "models", "face_detection_yunet_2023mar.onnx"),
            "face_detection_yunet_2023mar.onnx",
            "face_detection_yunet.onnx"
        ]
        for p in yunet_paths:
            if p and os.path.exists(p) and hasattr(cv2, 'FaceDetectorYN'):
                try:
                    self.detector_yn = cv2.FaceDetectorYN.create(p, "", (320, 320))
                    logger.info(f"✅ 成功加载 YuNet 深度人脸检测与关键点模型: {p}")
                    return
                except Exception as e:
                    logger.warning(f"加载 YuNet 模型失败: {e}")

        # 备用 Haar Cascade (仅做无模型情况下的兜底检测)
        if hasattr(cv2, 'CascadeClassifier') and hasattr(cv2, 'data'):
            cascade_path = getattr(cv2.data, 'haarcascades', '') + 'haarcascade_frontalface_default.xml'
            if os.path.exists(cascade_path):
                self.face_cascade = cv2.CascadeClassifier(cascade_path)
                logger.info("已初始化 Haar 人脸检测兜底器")

    def _init_recognizer(self):
        """初始化 512 维生物特征提取模型 (RKNN / ONNXRuntime)"""
        # 1. 尝试 RK3588 NPU 原生加速 (rknn-toolkit-lite2)
        try:
            from rknnlite.api import RKNNLite
            self.rknn_session = RKNNLite()
            if self.model_path and self.model_path.endswith(".rknn") and os.path.exists(self.model_path):
                ret = self.rknn_session.load_rknn(self.model_path)
                if ret == 0:
                    ret = self.rknn_session.init_runtime(core_mask=RKNNLite.NPU_CORE_0)
                    if ret == 0:
                        logger.info("✅ RK3588 NPU 人脸特征提取模型初始化成功")
                        return
        except ImportError:
            pass
        except Exception as e:
            logger.warning(f"RKNN 加载异常: {e}")

        # 2. 尝试 ONNX Runtime (MobileFaceNet / ArcFace)
        try:
            import onnxruntime as ort
            if self.model_path:
                onnx_paths = [self.model_path]
            else:
                onnx_paths = [
                    os.path.join(os.path.dirname(__file__), "models", "mobilefacenet.onnx"),
                    os.path.join(os.path.dirname(__file__), "models", "arcface_resnet100.onnx")
                ]
            for op in onnx_paths:
                if op and os.path.exists(op) and op.endswith(".onnx"):
                    self.onnx_session = ort.InferenceSession(op, providers=['CPUExecutionProvider'])
                    logger.info(f"✅ ONNX Runtime 加载人脸特征提取模型成功: {op}")
                    return
        except Exception as e:
            logger.warning(f"ONNXRuntime 加载异常: {e}")

        # 3. 若均未载入有效深度神经网络
        if self.rknn_session is None and self.onnx_session is None:
            if not self.allow_handcrafted_fallback:
                raise RuntimeError(
                    "生产模式严禁无模型启动：未检测到有效 ArcFace/MobileFaceNet 模型 (RKNN/ONNX)。"
                    "请配置有效的 recognition.model_path 或部署 models/mobilefacenet.onnx 模型文件。"
                    "如仅在研发测试环境下运行，请显式配置 allow_handcrafted_fallback: true。"
                )
            logger.warning("⚠️ 警告：当前以研发模式启动，未加载深度神经网络模型，回退至手工梯度拓扑特征 (HOG 512D)！")
        else:
            logger.info("✅ 深度人脸识别模型已就绪 (512-d Biometric Embedding)")

    def detect_face(self, frame: np.ndarray) -> Optional[Tuple[int, int, int, int]]:
        """
        检测画面中的最大人脸并记录 5 点面部关键点坐标。
        重要：若未检测到有效人脸，必须严格返回 None，绝不把画面中央当作人脸！
        :return: (x, y, w, h) 或 None
        """
        self.last_landmarks = None
        if frame is None or frame.size == 0:
            return None

        h, w = frame.shape[:2]

        # 1. YuNet 深度检测
        if self.detector_yn is not None:
            self.detector_yn.setInputSize((w, h))
            _, faces = self.detector_yn.detect(frame)
            if faces is not None and len(faces) > 0:
                # 选取置信度与面积综合最大的人脸
                best_face = max(faces, key=lambda f: f[2] * f[3] * f[14])
                fx, fy, fw, fh = int(best_face[0]), int(best_face[1]), int(best_face[2]), int(best_face[3])
                # 提取 YuNet 5 点关键点并显式按画面左右坐标重排：
                # [画面左眼 (viewer-left), 画面右眼 (viewer-right), 鼻尖, 画面左嘴角, 画面右嘴角]
                pt_eye1 = np.array([best_face[4], best_face[5]], dtype=np.float32)
                pt_eye2 = np.array([best_face[6], best_face[7]], dtype=np.float32)
                if pt_eye1[0] <= pt_eye2[0]:
                    left_eye, right_eye = pt_eye1, pt_eye2
                else:
                    left_eye, right_eye = pt_eye2, pt_eye1

                nose = np.array([best_face[8], best_face[9]], dtype=np.float32)

                pt_m1 = np.array([best_face[10], best_face[11]], dtype=np.float32)
                pt_m2 = np.array([best_face[12], best_face[13]], dtype=np.float32)
                if pt_m1[0] <= pt_m2[0]:
                    left_mouth, right_mouth = pt_m1, pt_m2
                else:
                    left_mouth, right_mouth = pt_m2, pt_m1

                landmarks = np.array([
                    left_eye,
                    right_eye,
                    nose,
                    left_mouth,
                    right_mouth
                ], dtype=np.float32)
                self.last_landmarks = landmarks

                # 边界保护
                fx = max(0, fx)
                fy = max(0, fy)
                fw = min(w - fx, fw)
                fh = min(h - fy, fh)
                if fw >= self.min_face_size and fh >= self.min_face_size:
                    return (fx, fy, fw, fh)

        # 2. Haar 级联检测器 (备用兜底)
        if self.face_cascade is not None:
            gray = cv2.cvtColor(frame, cv2.COLOR_BGR2GRAY)
            faces = self.face_cascade.detectMultiScale(
                gray,
                scaleFactor=1.15,
                minNeighbors=5,
                minSize=(self.min_face_size, self.min_face_size)
            )
            if len(faces) > 0:
                largest_face = max(faces, key=lambda rect: rect[2] * rect[3])
                return tuple(int(v) for v in largest_face)

        # 严格返回 None，杜绝中央 ROI 虚假假人脸！
        return None

    def detect_face_with_landmarks(self, frame: np.ndarray) -> Tuple[Optional[Tuple[int, int, int, int]], Optional[np.ndarray]]:
        """检测人脸并显式返回 (bbox, 5-landmarks)"""
        bbox = self.detect_face(frame)
        return bbox, self.last_landmarks

    def align_face(
        self,
        frame: np.ndarray,
        bbox: Optional[Tuple[int, int, int, int]] = None,
        landmarks: Optional[np.ndarray] = None
    ) -> np.ndarray:
        """
        根据 5 点面部关键点执行相似变换仿射对齐 (cv2.estimateAffinePartial2D + warpAffine)，
        生成标准 112×112 ArcFace / MobileFaceNet 对齐人脸；若无关键点则回退扩边裁剪。
        关键点映射基准为：
          [0]: 画面左眼 (viewer-left eye, x 较小)
          [1]: 画面右眼 (viewer-right eye, x 较大)
          [2]: 鼻尖 (nose)
          [3]: 画面左嘴角 (viewer-left mouth, x 较小)
          [4]: 画面右嘴角 (viewer-right mouth, x 较大)
        """
        lm = landmarks if landmarks is not None else self.last_landmarks
        if lm is not None and len(lm) == 5:
            try:
                lm = np.array(lm, dtype=np.float32).copy()
                # 几何拓扑防御保护：若双眼或双嘴角的 x 坐标逆序，自动按画面左右重排，杜绝仿射矩阵反向镜像畸变
                if lm[0, 0] > lm[1, 0]:
                    lm[[0, 1]] = lm[[1, 0]]
                if lm[3, 0] > lm[4, 0]:
                    lm[[3, 4]] = lm[[4, 3]]

                M, _ = cv2.estimateAffinePartial2D(lm, STANDARD_LANDMARKS_112)
                if M is not None:
                    aligned = cv2.warpAffine(frame, M, (112, 112), flags=cv2.INTER_LINEAR, borderMode=cv2.BORDER_REFLECT)
                    return aligned
            except Exception as e:
                logger.warning(f"5 点仿射对齐异常，降级至 bbox 裁剪: {e}")

        # 降级模式：无关键点时的平移与缩放裁剪
        if bbox is None:
            bbox = self.detect_face(frame)
        if bbox is None:
            return cv2.resize(frame, (112, 112))

        x, y, w, h = bbox
        img_h, img_w = frame.shape[:2]

        # 扩展边缘防止贴边截断
        pad_x = int(w * 0.1)
        pad_y = int(h * 0.1)
        x1 = max(0, x - pad_x)
        y1 = max(0, y - pad_y)
        x2 = min(img_w, x + w + pad_x)
        y2 = min(img_h, y + h + pad_y)

        face_roi = frame[y1:y2, x1:x2]
        if face_roi.size == 0:
            return cv2.resize(frame, (112, 112))
        aligned = cv2.resize(face_roi, (112, 112), interpolation=cv2.INTER_AREA)
        return aligned

    def extract_features(self, face_bgr: np.ndarray) -> np.ndarray:
        """
        提取人脸的 512 维深度生物特征向量（单位 L2 归一化）。
        彻底告别伪特征/颜色直方图！
        :param face_bgr: 裁剪对齐后的 112x112 人脸图像
        :return: 512 维 float32 向量 (norm=1.0)
        """
        if face_bgr is None or face_bgr.size == 0:
            return np.zeros(512, dtype=np.float32)

        aligned = cv2.resize(face_bgr, (112, 112))

        # 1. 若载入了 RKNN NPU 引擎
        if self.rknn_session is not None:
            try:
                rgb = cv2.cvtColor(aligned, cv2.COLOR_BGR2RGB)
                input_data = np.expand_dims(rgb, 0)
                outputs = self.rknn_session.inference(inputs=[input_data])
                feat = outputs[0].flatten().astype(np.float32)[:512]
                norm = np.linalg.norm(feat)
                if norm > 1e-6:
                    return feat / norm
            except Exception as e:
                logger.warning(f"RKNN 推理失败: {e}")

        # 2. 若载入了 ONNX Runtime 引擎 (MobileFaceNet / ArcFace)
        if self.onnx_session is not None:
            try:
                rgb = cv2.cvtColor(aligned, cv2.COLOR_BGR2RGB).astype(np.float32)
                # 归一化 (x - 127.5) / 128.0
                norm_img = (rgb - 127.5) / 128.0
                norm_img = np.transpose(norm_img, (2, 0, 1))  # (3, 112, 112)
                input_blob = np.expand_dims(norm_img, 0)      # (1, 3, 112, 112)
                input_name = self.onnx_session.get_inputs()[0].name
                outputs = self.onnx_session.run(None, {input_name: input_blob})
                feat = outputs[0].flatten().astype(np.float32)[:512]
                norm = np.linalg.norm(feat)
                if norm > 1e-6:
                    return feat / norm
            except Exception as e:
                logger.warning(f"ONNX 推理失败: {e}")

        # 3. 兜底手工特征模式 (仅当显式允许 allow_handcrafted_fallback=True 时生效)
        if not self.allow_handcrafted_fallback:
            raise RuntimeError("未加载有效深度人脸识别模型，生产环境严禁提取手工梯度伪特征！")

        # 基于梯度方向、分块局部二值模式与面部拓扑特征 (Deterministic 512-d Biometric Embedding)
        gray = cv2.cvtColor(aligned, cv2.COLOR_BGR2GRAY)
        norm_gray = cv2.equalizeHist(gray).astype(np.float32) / 255.0

        # 计算 Sobel 空间梯度拓扑
        gx = cv2.Sobel(norm_gray, cv2.CV_32F, 1, 0, ksize=3)
        gy = cv2.Sobel(norm_gray, cv2.CV_32F, 0, 1, ksize=3)
        mag, ang = cv2.cartToPolar(gx, gy, angleInDegrees=True)

        # 8x8 空间网格特征聚合 (64 个单元格，每个提取 8 方向梯度能量 = 512 维)
        cell_h = 112 // 8
        cell_w = 112 // 8
        feature_vector = np.zeros(512, dtype=np.float32)

        for r in range(8):
            for c in range(8):
                cell_mag = mag[r*cell_h:(r+1)*cell_h, c*cell_w:(c+1)*cell_w]
                cell_ang = ang[r*cell_h:(r+1)*cell_h, c*cell_w:(c+1)*cell_w]
                idx = (r * 8 + c) * 8
                # 8 个方向分箱 (0~45, 45~90, ..., 315~360)
                for b in range(8):
                    lower = b * 45.0
                    upper = (b + 1) * 45.0
                    mask = (cell_ang >= lower) & (cell_ang < upper)
                    feature_vector[idx + b] = np.sum(cell_mag[mask])

        # L2 范数归一化为单位球面超向量
        norm = np.linalg.norm(feature_vector)
        if norm > 1e-6:
            feature_vector = feature_vector / norm
        else:
            feature_vector = np.zeros(512, dtype=np.float32)

        return feature_vector

    def _derive_key(self, salt: bytes) -> bytes:
        """从 device_secret 与盐值通过 PBKDF2 派生 256 位 AES-GCM 密钥"""
        return hashlib.pbkdf2_hmac(
            'sha256',
            self.device_secret.encode('utf-8'),
            salt,
            iterations=100000,
            dklen=32
        )

    def encrypt_feature(self, vector: np.ndarray) -> bytes:
        """使用 AES-256-GCM 加密 512-d 特征向量"""
        salt = os.urandom(16)
        nonce = os.urandom(12)
        key = self._derive_key(salt)
        aesgcm = AESGCM(key)

        data_bytes = vector.astype(np.float32).tobytes()
        ciphertext = aesgcm.encrypt(nonce, data_bytes, MAGIC_HEADER)

        # 封包：[MAGIC 4B][SALT 16B][NONCE 12B][CIPHERTEXT + 16B TAG]
        return MAGIC_HEADER + salt + nonce + ciphertext

    def decrypt_feature(self, enc_bytes: bytes) -> np.ndarray:
        """使用 AES-256-GCM 解密特征模板"""
        if len(enc_bytes) < 48 or not enc_bytes.startswith(MAGIC_HEADER):
            raise ValueError("无效的人脸模板加密格式或魔数不匹配")

        salt = enc_bytes[4:20]
        nonce = enc_bytes[20:32]
        ciphertext = enc_bytes[32:]

        key = self._derive_key(salt)
        aesgcm = AESGCM(key)

        decrypted = aesgcm.decrypt(nonce, ciphertext, MAGIC_HEADER)
        vector = np.frombuffer(decrypted, dtype=np.float32).copy()
        if vector.shape[0] != 512:
            raise ValueError(f"解密特征维度异常: {vector.shape[0]}, 期望 512")
        return vector

    def load_templates(self):
        """从本地模板目录载入并解密全部已登记用户的 512-d 特征向量"""
        self.gallery.clear()
        count = 0
        if not os.path.exists(self.template_dir):
            return

        for filename in os.listdir(self.template_dir):
            filepath = os.path.join(self.template_dir, filename)
            if filename.endswith(".enc"):
                student_no = os.path.splitext(filename)[0]
                try:
                    with open(filepath, "rb") as f:
                        enc_bytes = f.read()
                    feat = self.decrypt_feature(enc_bytes)
                    self.gallery[student_no] = feat
                    count += 1
                except Exception as e:
                    logger.error(f"加载解密特征模板失败: {filepath}, {e}")
            elif filename.endswith(".npy"):
                logger.warning(f"检测到未加密的不安全模板 {filename}，根据安全规范拒绝直接加载")

        logger.info(f"✅ 成功加载并解密 {count} 位用户的生物特征比对模板")

    def enroll_face(self, student_no: str, frame: np.ndarray) -> bool:
        """
        人脸录入：提取 512-d 向量并以 AES-256-GCM 加密保存为 .enc 文件。
        绝不长期持久化原始照片！
        """
        bbox = self.detect_face(frame)
        if bbox is None:
            logger.warning("未检测到有效人脸，录入终止")
            return False

        aligned_face = self.align_face(frame, bbox)
        feature = self.extract_features(aligned_face)

        enc_payload = self.encrypt_feature(feature)
        target_path = os.path.join(self.template_dir, f"{student_no}.enc")
        with open(target_path, "wb") as f:
            f.write(enc_payload)

        self.gallery[student_no] = feature
        logger.info(f"✅ 用户 [{student_no}] 生物特征模板加密存储成功: {target_path}")
        return True

    def match_1_n(self, face_bgr: np.ndarray, threshold: float = 0.80) -> Tuple[Optional[str], float]:
        """
        1:N 人脸余弦相似度比对。
        :return: (matched_student_no, confidence)
        """
        if len(self.gallery) == 0:
            return None, 0.0

        feat = self.extract_features(face_bgr)
        best_student = None
        best_similarity = -1.0

        for student_no, template_feat in self.gallery.items():
            # 余弦相似度 = dot(u, v) (因已做 L2 归一化)
            similarity = float(np.dot(feat, template_feat))
            if similarity > best_similarity:
                best_similarity = similarity
                best_student = student_no

        if best_similarity >= threshold:
            return best_student, best_similarity
        return None, max(0.0, best_similarity)
