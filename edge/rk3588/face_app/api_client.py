import time
import uuid
import hmac
import hashlib
import json
import logging
import requests
from typing import Dict, Any, Optional

logger = logging.getLogger("CabinetApiClient")

class CabinetApiClient:
    """
    柜机终端专属 HTTP API 客户端。
    严格按照服务端 CabinetAuthMiddleware 规范生成 HMAC-SHA256 设备签名与防重放 Nonce。
    """
    def __init__(self, server_url: str, device_id: str, device_secret: str, timeout: int = 5):
        self.server_url = server_url.rstrip('/')
        self.device_id = device_id
        self.device_secret = device_secret
        self.timeout = timeout

    def _build_signed_headers(self, method: str, path: str, body_bytes: bytes = b"") -> Dict[str, str]:
        timestamp = str(int(time.time()))
        nonce = str(uuid.uuid4())
        body_hash = hashlib.sha256(body_bytes).hexdigest()

        canonical_string = f"{method.upper()}\n{path}\n{timestamp}\n{nonce}\n{body_hash}"
        signature = hmac.new(
            self.device_secret.encode('utf-8'),
            canonical_string.encode('utf-8'),
            hashlib.sha256
        ).hexdigest()

        return {
            "Content-Type": "application/json",
            "X-Cabinet-ID": self.device_id,
            "X-Timestamp": timestamp,
            "X-Nonce": nonce,
            "X-Signature": signature
        }

    def auth_face(self, student_no: str, confidence: float, liveness_passed: bool) -> Optional[Dict[str, Any]]:
        """
        向服务端上报 RK3588 本地人脸特征比对与活体检测结果，换取短期 FaceSessionToken。
        """
        path = "/api/v1/cabinet/auth/face"
        url = f"{self.server_url}{path}"
        payload = {
            "deviceId": self.device_id,
            "studentNo": student_no,
            "confidence": float(confidence),
            "livenessPassed": bool(liveness_passed)
        }
        body_bytes = json.dumps(payload, separators=(',', ':')).encode('utf-8')
        headers = self._build_signed_headers("POST", path, body_bytes)

        try:
            resp = requests.post(url, data=body_bytes, headers=headers, timeout=self.timeout)
            resp_data = resp.json()
            if resp.status_code == 200 and resp_data.get("code") == 0:
                logger.info("人脸认证成功，获得短期 FaceSession 令牌")
                return resp_data.get("data")
            else:
                logger.warning(f"人脸认证失败: HTTP {resp.status_code}, {resp_data.get('message')}")
                return None
        except Exception as e:
            logger.error(f"连接服务端异常: {e}")
            return None

    def match_room(self, room_no: str) -> Optional[list]:
        """
        根据房间号查询指定柜机中可借出的匹配钥匙。
        """
        path = "/api/v1/cabinet/keys/match-room"
        url = f"{self.server_url}{path}?roomNo={room_no}&deviceId={self.device_id}"
        headers = self._build_signed_headers("GET", path, b"")

        try:
            resp = requests.get(url, headers=headers, timeout=self.timeout)
            resp_data = resp.json()
            if resp.status_code == 200 and resp_data.get("code") == 0:
                return resp_data.get("data", [])
            return None
        except Exception as e:
            logger.error(f"查询房间号钥匙异常: {e}")
            return None

    def direct_dispense(self, room_no: str, key_id: str, face_session_token: str, purpose: str = "现场刷脸出钥") -> Optional[Dict[str, Any]]:
        """
        携带 FaceSessionToken 触发柜机出钥。
        """
        path = "/api/v1/cabinet/direct-dispense"
        url = f"{self.server_url}{path}"
        payload = {
            "requestId": f"req_{int(time.time() * 1000)}_{uuid.uuid4().hex[:6]}",
            "deviceId": self.device_id,
            "roomNo": room_no,
            "keyId": key_id,
            "purpose": purpose
        }
        body_bytes = json.dumps(payload, separators=(',', ':')).encode('utf-8')
        headers = self._build_signed_headers("POST", path, body_bytes)
        # 挂载 FaceSession Token
        headers["Authorization"] = f"Bearer {face_session_token}"

        try:
            resp = requests.post(url, data=body_bytes, headers=headers, timeout=self.timeout)
            resp_data = resp.json()
            if resp.status_code in (200, 202) and resp_data.get("code") == 0:
                logger.info(f"出钥指令已受理: {resp_data.get('data')}")
                return resp_data.get("data")
            else:
                logger.warning(f"出钥拒绝: HTTP {resp.status_code}, {resp_data.get('message')}")
                return None
        except Exception as e:
            logger.error(f"出钥网络调用失败: {e}")
            return None
