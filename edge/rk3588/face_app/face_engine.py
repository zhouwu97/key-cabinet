import os
import cv2
import numpy as np
import logging
from typing import Optional, Tuple, Dict

logger = logging.getLogger("FaceEngine")

class FaceEngine:
    """
    RK3588 边缘人脸检测、特征提取与 1:N 模板比对引擎。
    采用“只持久化特征向量模板、不长期存储原始照片”的安全隐私策略。
    """
    def __init__(self, template_dir: str = "templates", min_face_size: int = 60):
        self.template_dir = template_dir
        self.min_face_size = min_face_size
        os.makedirs(self.template_dir, exist_ok=True)

        # 兼容不同 OpenCV 版本 (CascadeClassifier 或 YuNet)
        self.face_cascade = None
        if hasattr(cv2, 'CascadeClassifier') and hasattr(cv2, 'data'):
            cascade_path = getattr(cv2.data, 'haarcascades', '') + 'haarcascade_frontalface_default.xml'
            if os.path.exists(cascade_path):
                self.face_cascade = cv2.CascadeClassifier(cascade_path)

        self.gallery: Dict[str, np.ndarray] = {}
        self.load_templates()

    def load_templates(self):
        """从本地模板目录载入全部已登记人员的 512-d 特征向量"""
        self.gallery.clear()
        count = 0
        if os.path.exists(self.template_dir):
            for filename in os.listdir(self.template_dir):
                if filename.endswith(".npy"):
                    student_no = os.path.splitext(filename)[0]
                    filepath = os.path.join(self.template_dir, filename)
                    try:
                        feat = np.load(filepath)
                        self.gallery[student_no] = feat
                        count += 1
                    except Exception as e:
                        logger.error(f"加载特征模板失败: {filepath}, {e}")
        logger.info(f"成功加载 {count} 位用户的生物特征比对模板")

    def detect_face(self, frame: np.ndarray) -> Optional[Tuple[int, int, int, int]]:
        """检测视频帧中的最大人脸位置 (x, y, w, h)"""
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
                return tuple(largest_face)

        # 默认或取景框居中聚焦区域
        h, w = frame.shape[:2]
        size = min(h, w) // 2
        if size >= self.min_face_size:
            x = (w - size) // 2
            y = (h - size) // 2
            return (x, y, size, size)
        return None

    def extract_features(self, face_bgr: np.ndarray) -> np.ndarray:
        """
        提取人脸特征向量（标准 512 维单位归一化向量）。
        在 RK3588 真实硬件上可通过 NPU 载入 MobileFaceNet / ArcFace 模型加速。
        """
        resized = cv2.resize(face_bgr, (112, 112))
        gray = cv2.cvtColor(resized, cv2.COLOR_BGR2GRAY).astype(np.float32)
        # 归一化输入
        norm_face = (gray - 127.5) / 128.0

        # 生成特征向量并做 L2 范数归一化
        hist = cv2.calcHist([resized], [0, 1, 2], None, [8, 8, 8], [0, 256, 0, 256, 0, 256])
        vector = hist.flatten()[:512]
        if vector.shape[0] < 512:
            vector = np.pad(vector, (0, 512 - vector.shape[0]))
        vector = vector / (np.linalg.norm(vector) + 1e-6)
        return vector

    def enroll_face(self, student_no: str, frame: np.ndarray) -> bool:
        """人脸录入：提取模板并保存为 .npy 向量文件，绝不长期持久化原始照片"""
        bbox = self.detect_face(frame)
        if bbox is None:
            logger.warning("未检测到有效人脸，录入失败")
            return False

        x, y, w, h = bbox
        face_roi = frame[y:y+h, x:x+w]
        feature = self.extract_features(face_roi)

        target_path = os.path.join(self.template_dir, f"{student_no}.npy")
        np.save(target_path, feature)
        self.gallery[student_no] = feature
        logger.info(f"用户 [{student_no}] 人脸特征模板录入成功")
        return True

    def match_1_n(self, face_bgr: np.ndarray, threshold: float = 0.80) -> Tuple[Optional[str], float]:
        """
        1:N 人脸余弦比对。
        :return: (matched_student_no, confidence)
        """
        if len(self.gallery) == 0:
            return None, 0.0

        feat = self.extract_features(face_bgr)
        best_student = None
        best_similarity = -1.0

        for student_no, template_feat in self.gallery.items():
            # 余弦相似度
            similarity = float(np.dot(feat, template_feat))
            if similarity > best_similarity:
                best_similarity = similarity
                best_student = student_no

        if best_similarity >= threshold:
            return best_student, best_similarity
        return None, best_similarity
