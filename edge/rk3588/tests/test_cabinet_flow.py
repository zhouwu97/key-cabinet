import hashlib
import hmac
import json
import os
import sys
import time
import unittest
from concurrent.futures import Future
from unittest.mock import Mock, patch

import requests

sys.path.insert(0, os.path.abspath(os.path.join(os.path.dirname(__file__), "..", "face_app")))
from api_client import CabinetApiClient
from app import RK3588EdgeApplication


class ImmediateExecutor:
    def submit(self, callback):
        future = Future()
        future.set_result(callback())
        return future


class CabinetFlowTest(unittest.TestCase):
    def make_app(self):
        # 仅隔离摄像头和模型初始化，实际运行柜端状态机及网络回调。
        app = RK3588EdgeApplication.__new__(RK3588EdgeApplication)
        app.api_client = Mock(last_error="", last_error_code="")
        app.ui = Mock()
        app._network = ImmediateExecutor()
        app._pending_network = None
        app._session_version = 0
        app._retry_start = None
        app.face_session_token = "face-token"
        app.state = "AUTHENTICATED"
        app.session_expire_time = time.time() + 300
        app.state_expire_time = time.time() + 45
        app.next_operation_check = time.time() + 1.5
        app.operation_id = ""
        app.operation_action = ""
        app.active_reservations = [{"id": "r1", "roomNo": "101"}]
        app.active_borrows = [{"id": "b1"}]
        return app

    @patch("api_client.requests.request")
    def test_signed_pickup_and_return_contract(self, request):
        request.return_value = Mock(status_code=202, json=lambda: {"code": 0, "data": {"id": "op1", "status": "EXECUTING"}})
        client = CabinetApiClient("http://localhost:8080", "CAB001", "local-test-secret")
        result = client.pickup_reservation("r1", "face-token", "stable-request")
        self.assertEqual(result["status"], "EXECUTING")
        method, url = request.call_args.args
        options = request.call_args.kwargs
        self.assertEqual(method, "POST")
        self.assertTrue(url.endswith("/api/v1/cabinet/device-operations/pickup"))
        self.assertEqual(json.loads(options["data"]), {"reservationId": "r1", "clientRequestId": "stable-request"})
        headers = options["headers"]
        self.assertEqual(headers["Authorization"], "Bearer face-token")
        canonical = "\n".join([method, "/api/v1/cabinet/device-operations/pickup", headers["X-Timestamp"], headers["X-Nonce"], hashlib.sha256(options["data"]).hexdigest()])
        self.assertEqual(headers["X-Signature"], hmac.new(b"local-test-secret", canonical.encode(), hashlib.sha256).hexdigest())
        client.return_key("b1", "face-token", "return-request")
        self.assertEqual(json.loads(request.call_args.kwargs["data"])["deviceId"], "CAB001")
        client.get_operation("op1", "face-token")
        self.assertTrue(request.call_args.args[1].endswith("/device-operations/op1"))
        client.cancel_operation("op1", "face-token")
        self.assertTrue(request.call_args.args[1].endswith("/op1/cancel"))

    @patch("api_client.requests.request")
    def test_invalid_face_proof_does_not_request_authentication(self, request):
        client = CabinetApiClient("http://localhost", "CAB001", "local-test")
        for confidence in [float("nan"), float("inf"), 0.79, 1.1]:
            self.assertIsNone(client.auth_face("20230001", confidence, True))
        self.assertIsNone(client.auth_face("20230001", 0.95, False))
        request.assert_not_called()

    @patch("api_client.requests.request")
    def test_network_error_and_server_rejection_are_distinct(self, request):
        client = CabinetApiClient("http://localhost", "CAB001", "local-test")
        request.side_effect = requests.Timeout()
        self.assertIsNone(client.get_operation("op1", "token"))
        self.assertEqual(client.last_error_code, "NETWORK_ERROR")
        request.side_effect = None
        request.return_value = Mock(status_code=403, json=lambda: {"code": 400, "errorCode": "FORBIDDEN", "message": "身份未核验"})
        self.assertIsNone(client.pickup_reservation("r1", "token", "request"))
        self.assertEqual(client.last_error, "身份未核验")
        self.assertEqual(client.last_error_code, "FORBIDDEN")

    def test_accepted_operation_waits_for_physical_terminal_state(self):
        app = self.make_app()
        app.api_client.pickup_reservation.return_value = {"id": "op1", "status": "EXECUTING"}
        app.api_client.get_operation.return_value = {"id": "op1", "status": "EXECUTING"}
        app._on_room_submit("101")
        app._network_tick()
        self.assertEqual(app.state, "OPERATING")
        self.assertNotIn("成功", app.ui.set_status.call_args.args[0])
        app._on_operation_snapshot({"status": "EXECUTING", "events": [{"seq": 4, "type": "KEY_REMOVED"}]}, "", "")
        self.assertIn("关闭柜门", app.ui.set_status.call_args.args[0])
        self.assertEqual(app.state, "OPERATING")
        app._on_operation_snapshot({"status": "SUCCESS"}, "", "")
        self.assertEqual(app.state, "RESULT")
        self.assertIn("取钥成功", app.ui.set_status.call_args.args[0])
        app._on_operation_snapshot({"status": "EXECUTING"}, "", "")
        self.assertIn("取钥成功", app.ui.set_status.call_args.args[0])

    def test_return_rejects_double_click_and_displays_failure(self):
        app = self.make_app()
        app.api_client.return_key.return_value = {"id": "op2", "status": "EXECUTING"}
        app._on_borrow_return("b1")
        app._on_borrow_return("b1")
        self.assertEqual(app.api_client.return_key.call_count, 1)
        app._on_operation_started({"id": "op2", "status": "EXECUTING"}, "", "")
        app._on_cancel()
        self.assertEqual(app.state, "OPERATING")
        app._on_operation_snapshot({"status": "FAILED", "errorMessage": "RFID 不匹配"}, "", "")
        self.assertEqual(app.state, "RESULT")
        self.assertEqual(app.ui.set_status.call_args.args[0], "RFID 不匹配")

    def test_retry_reuses_business_request_id(self):
        app = self.make_app()
        app.api_client.last_error = "网络断开"
        app.api_client.last_error_code = "NETWORK_ERROR"
        app.api_client.pickup_reservation.return_value = None
        app._on_reservation_pickup("r1")
        app._network_tick()
        self.assertEqual(app.state, "STARTING")
        app.next_operation_check = 0
        app._network_tick()
        self.assertEqual(app.api_client.pickup_reservation.call_count, 2)
        self.assertEqual(app.api_client.pickup_reservation.call_args_list[0].args, app.api_client.pickup_reservation.call_args_list[1].args)

    def test_reauthentication_recovers_existing_operation(self):
        app = self.make_app()
        app.api_client.get_active_operation.return_value = {"id": "existing", "action": "RETURN", "status": "EXECUTING"}
        app._on_face_authenticated({"faceSessionToken": "new-token", "expiresIn": 300, "user": {"name": "用户"}, "activeBorrows": [{"id": "b1"}]}, "", "")
        self.assertEqual(app.state, "RECOVERING")
        app._network_tick()
        app._network_tick()
        self.assertEqual(app.operation_id, "existing")
        self.assertEqual(app.operation_action, "RETURN")
        self.assertEqual(app.state, "OPERATING")

    def test_expired_session_and_late_auth_response_do_not_start_new_operation(self):
        app = self.make_app()
        app.session_expire_time = time.time() - 1
        app._on_reservation_pickup("r1")
        app.api_client.pickup_reservation.assert_not_called()
        app.state = "AUTHENTICATING"
        future = Future()
        app._pending_network = (future, app._on_face_authenticated, app._session_version)
        app._on_cancel()
        future.set_result(({"faceSessionToken": "old-token"}, "", ""))
        app._network_tick()
        self.assertEqual(app.state, "SCANNING")
        self.assertIsNone(app.face_session_token)


if __name__ == "__main__":
    unittest.main()
