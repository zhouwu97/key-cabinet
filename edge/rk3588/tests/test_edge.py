import os
import sys
import unittest
import numpy as np

sys.path.insert(0, os.path.abspath(os.path.join(os.path.dirname(__file__), "..", "face_app")))

from api_client import CabinetApiClient
from liveness_detector import LivenessDetector
from face_engine import FaceEngine

class TestRK3588Edge(unittest.TestCase):
    def test_signed_headers(self):
        client = CabinetApiClient("http://127.0.0.1:8080", "CAB001", "cab_sec_CAB001")
        headers = client._build_signed_headers("POST", "/api/v1/cabinet/auth/face", b'{"test":1}')
        self.assertEqual(headers["X-Cabinet-ID"], "CAB001")
        self.assertIn("X-Timestamp", headers)
        self.assertIn("X-Nonce", headers)
        self.assertIn("X-Signature", headers)
        self.assertEqual(len(headers["X-Signature"]), 64) # SHA256 hex string

    def test_liveness_blank_frame(self):
        detector = LivenessDetector()
        passed, score, reason = detector.check_liveness(np.zeros((100, 100, 3), dtype=np.uint8))
        self.assertFalse(passed)

    def test_face_features_cosine(self):
        engine = FaceEngine(template_dir="test_templates")
        img1 = np.random.randint(0, 256, (112, 112, 3), dtype=np.uint8)
        feat1 = engine.extract_features(img1)
        self.assertEqual(feat1.shape[0], 512)
        norm = np.linalg.norm(feat1)
        self.assertAlmostEqual(norm, 1.0, places=4)

if __name__ == "__main__":
    unittest.main()
