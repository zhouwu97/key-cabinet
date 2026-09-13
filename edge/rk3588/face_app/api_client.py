import time
import uuid
import hmac
import hashlib
import json
import logging
import math
from urllib.parse import quote, urlencode
from typing import Dict, Any, Optional

import requests

logger = logging.getLogger("CabinetApiClient")


class CabinetApiClient:
    """柜机签名客户端；设备密钥只驻留柜端，现场借还另需短期人脸会话。"""

    def __init__(self, server_url: str, device_id: str, device_secret: str, timeout: int = 5):
        self.server_url = server_url.rstrip('/')
        self.device_id = device_id
        self.device_secret = device_secret
        self.timeout = timeout
        self.last_error = ""
        self.last_error_code = ""

    def _build_signed_headers(self, method: str, path: str, body_bytes: bytes = b"") -> Dict[str, str]:
        timestamp = str(int(time.time()))
        nonce = str(uuid.uuid4())
        body_hash = hashlib.sha256(body_bytes).hexdigest()
        canonical_string = f"{method.upper()}\n{path}\n{timestamp}\n{nonce}\n{body_hash}"
        signature = hmac.new(self.device_secret.encode('utf-8'), canonical_string.encode('utf-8'), hashlib.sha256).hexdigest()
        return {
            "Content-Type": "application/json", "X-Cabinet-ID": self.device_id,
            "X-Timestamp": timestamp, "X-Nonce": nonce, "X-Signature": signature,
        }

    def _request(self, method: str, path: str, payload=None, token: str = "", query=None):
        self.last_error = ""
        self.last_error_code = ""
        body = json.dumps(payload, separators=(',', ':'), allow_nan=False).encode('utf-8') if payload is not None else b""
        headers = self._build_signed_headers(method, path, body)
        if token:
            headers["Authorization"] = f"Bearer {token}"
        url = self.server_url + path
        if query:
            url += "?" + urlencode(query)
        try:
            response = requests.request(method, url, data=body, headers=headers, timeout=self.timeout, allow_redirects=False)
            result = response.json()
            if not isinstance(result, dict):
                raise ValueError("服务端响应格式无效")
            if response.status_code in (200, 202) and result.get("code") == 0:
                return result.get("data")
            self.last_error = str(result.get("message") or f"服务端拒绝请求（HTTP {response.status_code}）")
            self.last_error_code = str(result.get("errorCode") or result.get("code") or response.status_code)
            if response.status_code >= 500:
                self.last_error_code = "SERVER_ERROR"
        except requests.RequestException:
            self.last_error = "网络暂时不可用，正在核实操作状态"
            self.last_error_code = "NETWORK_ERROR"
        except (ValueError, TypeError):
            self.last_error = "服务端响应无法解析，请稍后重试"
            self.last_error_code = "INVALID_RESPONSE"
        logger.warning("柜机接口 %s %s 失败: %s", method, path, self.last_error)
        return None

    def auth_face(self, student_no: str, confidence: float, liveness_passed: bool) -> Optional[Dict[str, Any]]:
        if not math.isfinite(confidence) or not 0.8 <= confidence <= 1 or liveness_passed is not True:
            self.last_error = "人脸比对或活体检测未通过"
            self.last_error_code = "FACE_VERIFICATION_FAILED"
            return None
        return self._request("POST", "/api/v1/cabinet/auth/face", {
            "deviceId": self.device_id, "studentNo": student_no,
            "confidence": float(confidence), "livenessPassed": True,
        })

    def match_room(self, room_no: str) -> Optional[list]:
        return self._request("GET", "/api/v1/cabinet/keys/match-room", query={"roomNo": room_no, "deviceId": self.device_id})

    def direct_dispense(self, room_no: str, key_id: str, face_session_token: str,
                        purpose: str = "现场刷脸出钥", request_id: str = "") -> Optional[Dict[str, Any]]:
        return self._request("POST", "/api/v1/cabinet/direct-dispense", {
            "requestId": request_id or uuid.uuid4().hex, "deviceId": self.device_id,
            "roomNo": room_no, "keyId": key_id, "purpose": purpose,
        }, token=face_session_token)

    def pickup_reservation(self, reservation_id: str, face_session_token: str, request_id: str):
        return self._request("POST", "/api/v1/cabinet/device-operations/pickup", {
            "reservationId": reservation_id, "clientRequestId": request_id,
        }, token=face_session_token)

    def return_key(self, borrow_record_id: str, face_session_token: str, request_id: str):
        return self._request("POST", "/api/v1/cabinet/device-operations/return", {
            "borrowRecordId": borrow_record_id, "deviceId": self.device_id, "clientRequestId": request_id,
        }, token=face_session_token)

    def get_operation(self, operation_id: str, face_session_token: str):
        return self._request("GET", f"/api/v1/cabinet/device-operations/{quote(operation_id, safe='')}", token=face_session_token)

    def get_active_operation(self, face_session_token: str):
        return self._request("GET", "/api/v1/cabinet/device-operations/active", token=face_session_token)

    def cancel_operation(self, operation_id: str, face_session_token: str):
        return self._request("POST", f"/api/v1/cabinet/device-operations/{quote(operation_id, safe='')}/cancel", token=face_session_token)
