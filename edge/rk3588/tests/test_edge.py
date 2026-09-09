import os
import shutil
import sys
import unittest
import numpy as np

sys.path.insert(0, os.path.abspath(os.path.join(os.path.dirname(__file__), "..", "face_app")))

from api_client import CabinetApiClient
from liveness_detector import LivenessDetector
from face_engine import FaceEngine

class TestRK3588Edge(unittest.TestCase):
    def setUp(self):
        self.test_dir = os.path.join(os.path.dirname(__file__), "test_templates")
        os.makedirs(self.test_dir, exist_ok=True)

    def tearDown(self):
        if os.path.exists(self.test_dir):
            shutil.rmtree(self.test_dir)

    def test_signed_headers(self):
        client = CabinetApiClient("http://127.0.0.1:8080", "CAB001", "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
        headers = client._build_signed_headers("POST", "/api/v1/cabinet/auth/face", b'{"test":1}')
        self.assertEqual(headers["X-Cabinet-ID"], "CAB001")
        self.assertIn("X-Timestamp", headers)
        self.assertIn("X-Nonce", headers)
        self.assertIn("X-Signature", headers)
        self.assertEqual(len(headers["X-Signature"]), 64) # SHA256 hex string

    def test_strict_face_detection_no_fake_center_roi(self):
        """严格测试：无有效人脸时必须返回 None，严禁把画面中央截取为假人脸！"""
        engine = FaceEngine(template_dir=self.test_dir, allow_handcrafted_fallback=True)
        # 全黑画面
        black_frame = np.zeros((480, 640, 3), dtype=np.uint8)
        self.assertIsNone(engine.detect_face(black_frame))

        # 纯高斯白噪点画面 (无有效面部五官)
        noise_frame = np.random.randint(0, 256, (480, 640, 3), dtype=np.uint8)
        self.assertIsNone(engine.detect_face(noise_frame))

    def test_liveness_thresholds_and_spoof_rejection(self):
        """测试活体防攻击指标：模糊虚化拦截、屏幕摩尔条纹拦截与正常样本通过"""
        detector = LivenessDetector(laplacian_threshold=85.0, accept_score=0.85, moire_papr_threshold=35.0)

        # 1. 空人脸
        passed, score, reason = detector.check_liveness(np.zeros((100, 100, 3), dtype=np.uint8))
        self.assertFalse(passed)
        self.assertEqual(score, 0.0)

        # 2. 严重虚焦/低清晰度打印人脸 (Laplacian 方差接近 0)
        blurry_face = np.full((112, 112, 3), 128, dtype=np.uint8)
        passed, score, reason = detector.check_liveness(blurry_face)
        self.assertFalse(passed)
        self.assertIn("纹理模糊", reason)

        # 3. 电子屏幕翻拍特征 (周期性网格/扫描线产生的高频摩尔尖峰)
        y, x = np.ogrid[:112, :112]
        moire_pattern = 128 + 50 * np.sin(2 * np.pi * x / 4.0) + 50 * np.cos(2 * np.pi * y / 4.0)
        screen_face = np.stack([moire_pattern]*3, axis=-1).astype(np.uint8)
        passed, score, reason = detector.check_liveness(screen_face)
        self.assertFalse(passed)
        self.assertIn("摩尔尖峰", reason)

        # 4. 具有丰富自然五官梯度的清晰人脸样本
        natural_face = np.full((112, 112, 3), (180, 200, 220), dtype=np.uint8)
        import cv2
        cv2.circle(natural_face, (56, 56), 40, (140, 170, 205), -1)
        cv2.ellipse(natural_face, (38, 48), (8, 4), 0, 0, 360, (50, 40, 30), -1)
        cv2.ellipse(natural_face, (74, 48), (8, 4), 0, 0, 360, (50, 40, 30), -1)
        cv2.line(natural_face, (56, 55), (56, 72), (100, 120, 160), 2)
        cv2.ellipse(natural_face, (56, 88), (14, 6), 0, 0, 360, (80, 90, 180), -1)
        # 添加高频微纹理提升清晰度
        for i in range(0, 112, 4):
            for j in range(0, 112, 4):
                natural_face[i, j] = np.clip(natural_face[i, j].astype(int) + (i*j)%30 - 15, 0, 255)

        passed, score, reason = detector.check_liveness(natural_face)
        self.assertTrue(passed)
        self.assertGreaterEqual(score, 0.85)

    def test_production_mode_requires_model(self):
        """测试生产模式强制要求深度人脸模型，严禁静默回退手工伪特征"""
        with self.assertRaises(RuntimeError):
            # 无模型且禁用回退，必须抛出 RuntimeError
            FaceEngine(template_dir=self.test_dir, model_path="non_existent_model.onnx", allow_handcrafted_fallback=False)

    def test_deep_model_onnx_inference(self):
        """测试加载真实 MobileFaceNet ONNX 深度模型提取 512 维特征"""
        mfn_path = os.path.join(os.path.dirname(__file__), "..", "face_app", "models", "mobilefacenet.onnx")
        if os.path.exists(mfn_path):
            engine = FaceEngine(template_dir=self.test_dir, model_path=mfn_path, allow_handcrafted_fallback=False)
            self.assertIsNotNone(engine.onnx_session)
            face_img = np.random.randint(0, 256, (112, 112, 3), dtype=np.uint8)
            feat = engine.extract_features(face_img)
            self.assertEqual(feat.shape, (512,))
            self.assertAlmostEqual(np.linalg.norm(feat), 1.0, places=4)

    def test_5_point_affine_alignment(self):
        """测试 5 点面部关键点相似变换仿射对齐到标准 112x112"""
        engine = FaceEngine(template_dir=self.test_dir, allow_handcrafted_fallback=True)
        frame = np.zeros((300, 300, 3), dtype=np.uint8)
        # 构造旋转/缩放后的 5 个关键点
        pts = np.array([
            [80.0, 100.0],  # 右眼
            [150.0, 98.0],  # 左眼
            [115.0, 140.0], # 鼻尖
            [88.0, 180.0],  # 右嘴角
            [145.0, 178.0]  # 左嘴角
        ], dtype=np.float32)
        aligned = engine.align_face(frame, landmarks=pts)
        self.assertEqual(aligned.shape, (112, 112, 3))

    def test_face_features_512d_normalization(self):
        """测试真实 512 维生物特征向量与单位球 L2 范数归一化"""
        engine = FaceEngine(template_dir=self.test_dir, allow_handcrafted_fallback=True)
        img = np.random.randint(0, 256, (112, 112, 3), dtype=np.uint8)
        feat = engine.extract_features(img)
        self.assertEqual(feat.shape[0], 512)
        norm = np.linalg.norm(feat)
        self.assertAlmostEqual(norm, 1.0, places=4)

    def test_aes_gcm_template_encryption_roundtrip(self):
        """测试人脸特征模板 AES-256-GCM 加密与解密完整闭环"""
        secret = "super_secure_device_secret_32_bytes_csprng_random_hex"
        engine = FaceEngine(template_dir=self.test_dir, device_secret=secret, allow_handcrafted_fallback=True)

        # 随机生成一个归一化 512-d 特征向量
        raw_vec = np.random.randn(512).astype(np.float32)
        raw_vec /= np.linalg.norm(raw_vec)

        # 加密
        enc_bytes = engine.encrypt_feature(raw_vec)
        self.assertTrue(enc_bytes.startswith(b"KCFE"))
        # 确保密文不包含明文浮点数
        self.assertNotEqual(enc_bytes, raw_vec.tobytes())

        # 解密
        dec_vec = engine.decrypt_feature(enc_bytes)
        self.assertEqual(dec_vec.shape, (512,))
        np.testing.assert_allclose(raw_vec, dec_vec, rtol=1e-5, atol=1e-6)

        # 篡改密文应触发 AEAD Tag 认证失败
        tampered = bytearray(enc_bytes)
        tampered[-1] ^= 0xFF
        with self.assertRaises(Exception):
            engine.decrypt_feature(bytes(tampered))

    def test_enroll_and_match_1_n(self):
        """测试加密模板入库与 1:N 余弦比对"""
        engine = FaceEngine(template_dir=self.test_dir, device_secret="test_secret", allow_handcrafted_fallback=True)

        # 构造模拟人脸图像
        test_img = np.zeros((112, 112, 3), dtype=np.uint8)
        test_img[30:80, 30:80] = 200

        # 直接模拟特征入库
        feat = engine.extract_features(test_img)
        enc_data = engine.encrypt_feature(feat)
        with open(os.path.join(self.test_dir, "S2026001.enc"), "wb") as f:
            f.write(enc_data)

        # 重新载入模板库
        engine.load_templates()
        self.assertIn("S2026001", engine.gallery)

        # 比对同一人脸
        match_id, conf = engine.match_1_n(test_img, threshold=0.80)
        self.assertEqual(match_id, "S2026001")
        self.assertGreater(conf, 0.95)

if __name__ == "__main__":
    unittest.main()
