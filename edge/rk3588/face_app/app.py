import os
import sys
import time
import argparse
import yaml
import cv2
import logging
import tkinter as tk
import uuid
from concurrent.futures import ThreadPoolExecutor
from typing import Optional, Dict, Any

# 兼容直接执行脚本和由服务管理器按 face_app 包导入两种启动方式。
try:
    from .api_client import CabinetApiClient
    from .liveness_detector import LivenessDetector
    from .face_engine import FaceEngine
    from .ui import TouchscreenKioskUI
except ImportError:
    from api_client import CabinetApiClient
    from liveness_detector import LivenessDetector
    from face_engine import FaceEngine
    from ui import TouchscreenKioskUI

logging.basicConfig(level=logging.INFO, format="%(asctime)s [%(levelname)s] %(name)s: %(message)s")
logger = logging.getLogger("RK3588EdgeApp")

class RK3588EdgeApplication:
    def __init__(self, config_path: str = "config.yaml"):
        # 统一解析配置文件绝对路径与配置所在目录，彻底消除由于工作目录不同导致的寻址异常
        config_abs = os.path.abspath(config_path)
        config_dir = os.path.dirname(config_abs)

        local_cfg_path = os.path.join(config_dir, "config.local.yaml")
        if os.path.exists(local_cfg_path):
            config_path = local_cfg_path
            config_abs = os.path.abspath(config_path)
            config_dir = os.path.dirname(config_abs)
            logger.info(f"加载本地优先配置: {local_cfg_path}")
        else:
            logger.info(f"加载主配置文件: {config_path}")

        with open(config_path, "r", encoding="utf-8") as f:
            self.cfg = yaml.safe_load(f) or {}

        cab_cfg = self.cfg.get("cabinet", {})
        rec_cfg = self.cfg.get("recognition", {})
        live_cfg = self.cfg.get("liveness", {})

        # 环境变量优先，未配置时读取配置项
        device_secret = os.environ.get("CABINET_DEVICE_SECRET") or cab_cfg.get("device_secret", "")
        self.device_id = cab_cfg.get("device_id", "CAB001")
        self.device_secret = device_secret
        allow_fallback = rec_cfg.get("allow_handcrafted_fallback", False)

        # 生产模式强制要求机柜通信密钥 (Fail-Fast)
        if not self.device_secret and not allow_fallback:
            raise RuntimeError(
                "生产模式必须配置机柜安全密钥 CABINET_DEVICE_SECRET (通过环境变量或 cabinet.device_secret)！"
                "严禁使用空密钥或默认密钥启动生产人脸认证与模板加密。"
            )

        self.api_client = CabinetApiClient(
            server_url=cab_cfg.get("server_url", "http://127.0.0.1:8080"),
            device_id=self.device_id,
            device_secret=self.device_secret,
            timeout=cab_cfg.get("timeout_seconds", 5)
        )

        self.liveness_detector = LivenessDetector(
            laplacian_threshold=live_cfg.get("laplacian_threshold", 85.0),
            accept_score=live_cfg.get("accept_score", 0.85),
            fft_high_freq_threshold=live_cfg.get("fft_high_freq_threshold", 0.80),
            glare_ratio_threshold=live_cfg.get("glare_ratio_threshold", 0.15)
        )

        # 路径解析：若为相对路径，一律按配置文件所在目录 (config_dir) 解析
        raw_model_path = rec_cfg.get("model_path")
        if raw_model_path:
            if not os.path.isabs(raw_model_path):
                model_path = os.path.normpath(os.path.join(config_dir, raw_model_path))
            else:
                model_path = raw_model_path
        else:
            model_path = None

        raw_template_dir = rec_cfg.get("template_dir", "face_app/templates")
        if raw_template_dir:
            if not os.path.isabs(raw_template_dir):
                template_dir = os.path.normpath(os.path.join(config_dir, raw_template_dir))
            else:
                template_dir = raw_template_dir
        else:
            template_dir = os.path.normpath(os.path.join(config_dir, "face_app", "templates"))

        self.face_engine = FaceEngine(
            template_dir=template_dir,
            device_secret=self.device_secret,
            model_path=model_path,
            allow_handcrafted_fallback=allow_fallback
        )

        self.conf_threshold = rec_cfg.get("confidence_threshold", 0.80)
        self.face_session_token: Optional[str] = None
        self.authenticated_user: Optional[Dict[str, Any]] = None

        # 状态机：刷脸认证后先恢复旧操作，再允许发起取还并等待设备终态。
        self.state = "SCANNING"
        self.state_expire_time = 0.0
        self.last_face_check_time = 0.0
        self.active_reservations = []
        self.active_borrows = []
        self.operation_id = ""
        self.operation_action = ""
        self.session_expire_time = 0.0
        self.next_operation_check = 0.0
        self._network = ThreadPoolExecutor(max_workers=1, thread_name_prefix="cabinet-api")
        self._pending_network = None
        self._session_version = 0
        self._retry_start = None

        # UI 与视频句柄
        self.root: Optional[tk.Tk] = None
        self.ui: Optional[TouchscreenKioskUI] = None
        self.cap: Optional[cv2.VideoCapture] = None

    def run_enrollment(self, student_no: str, cam_idx: int = 0) -> bool:
        """管理员模式：录入指定学号人员的面部生物特征模板并进行 AES-GCM 加密存储"""
        logger.info(f"开启摄像头 (index={cam_idx}) 进行生物特征模板录入...")
        cap = cv2.VideoCapture(cam_idx)
        if not cap.isOpened():
            logger.error("无法打开摄像头设备")
            return False

        logger.info("请正对摄像头，按 's' 采集录入，按 'q' 退出")
        success = False
        while True:
            ret, frame = cap.read()
            if not ret:
                break

            bbox = self.face_engine.detect_face(frame)
            display = frame.copy()
            if bbox:
                x, y, w, h = bbox
                cv2.rectangle(display, (x, y), (x+w, y+h), (0, 255, 0), 2)
                cv2.putText(display, f"Face Ready: Press 's' to enroll [{student_no}]", (x, max(20, y-10)),
                            cv2.FONT_HERSHEY_SIMPLEX, 0.6, (0, 255, 0), 2)
            else:
                cv2.putText(display, "No Face Detected", (20, 40),
                            cv2.FONT_HERSHEY_SIMPLEX, 0.7, (0, 0, 255), 2)

            cv2.imshow("RK3588 Face Enrollment", display)
            key = cv2.waitKey(1) & 0xFF
            if key == ord('s') and bbox:
                success = self.face_engine.enroll_face(student_no, frame)
                if success:
                    logger.info(f"✅ 学号 [{student_no}] 生物特征模板录入成功!")
                break
            elif key == ord('q'):
                break

        cap.release()
        cv2.destroyAllWindows()
        return success

    def run_touchscreen_kiosk(self, cam_idx: int = 0):
        """启动现场触摸屏 Kiosk UI 与人脸检测业务主循环"""
        logger.info(f"启动 RK3588 智能存取柜触屏终端 (Camera={cam_idx}, DeviceID={self.device_id})...")
        self.cap = cv2.VideoCapture(cam_idx)
        if not self.cap.isOpened():
            logger.error("无法初始化摄像头视频流")
            return

        self.root = tk.Tk()
        self.ui = TouchscreenKioskUI(
            self.root,
            on_room_submit=self._on_room_submit,
            on_cancel=self._on_cancel,
            on_pickup=self._on_reservation_pickup,
            on_return=self._on_borrow_return,
        )

        self._reset_to_scanning()
        self.root.after(30, self._video_loop)

        # 优雅关闭捕获
        self.root.protocol("WM_DELETE_WINDOW", self._on_window_close)
        self.root.mainloop()

    def _on_window_close(self):
        self._session_version += 1
        self._network.shutdown(wait=False, cancel_futures=True)
        if self.cap:
            self.cap.release()
        if self.root:
            self.root.destroy()
        logger.info("终端应用已退出")

    def _reset_to_scanning(self):
        self.state = "SCANNING"
        self.face_session_token = None
        self.authenticated_user = None
        self._session_version += 1
        self.active_reservations = []
        self.active_borrows = []
        self.operation_id = ""
        self._retry_start = None
        if self.ui:
            self.ui.set_tasks([], [])
            self.ui.clear_room_input()
            self.ui.set_status("等待人脸进入取景框...", "#4ade80")
            self.ui.set_user_info("身份：未认证")

    def _on_cancel(self):
        if self.state in ("STARTING", "OPERATING", "RECOVERING"):
            if self.ui:
                self.ui.set_status("正在核实设备状态，请等待完成；请勿重复取还", "#facc15")
            return
        logger.info("用户取消操作，重置为等待刷脸状态")
        self._reset_to_scanning()

    def _on_room_submit(self, room_no: str):
        """用户在虚拟触摸键盘输入房间号后点击确定"""
        if not self._can_start_operation():
            if self.ui:
                self.ui.set_status("请先完成人脸认证再选择房间", "#facc15")
            return

        # 输入已预约房间时直接使用预约取钥，保持预约用途和归还期限。
        matching = [r for r in self.active_reservations if r.get("roomNo") == room_no]
        if matching:
            self._on_reservation_pickup(matching[0]["id"])
            return
        token, request_id = self.face_session_token, uuid.uuid4().hex
        self._start_operation(lambda: self.api_client.direct_dispense(room_no, "", token, request_id=request_id), "PICKUP")

    def _can_start_operation(self):
        return self.state == "AUTHENTICATED" and self._pending_network is None and bool(self.face_session_token) and time.time() < self.session_expire_time

    def _on_reservation_pickup(self, reservation_id: str):
        if not self._can_start_operation() or not any(r.get("id") == reservation_id for r in self.active_reservations):
            return
        token, request_id = self.face_session_token, uuid.uuid4().hex
        self._start_operation(lambda: self.api_client.pickup_reservation(reservation_id, token, request_id), "PICKUP")

    def _on_borrow_return(self, borrow_id: str):
        if not self._can_start_operation() or not any(b.get("id") == borrow_id for b in self.active_borrows):
            return
        token, request_id = self.face_session_token, uuid.uuid4().hex
        self._start_operation(lambda: self.api_client.return_key(borrow_id, token, request_id), "RETURN")

    def _submit_network(self, request, callback):
        if self._pending_network is not None:
            return
        def run():
            result = request()
            return result, self.api_client.last_error, self.api_client.last_error_code
        self._pending_network = (self._network.submit(run), callback, self._session_version)

    def _network_tick(self):
        # Future 只执行网络请求；所有 Tk 控件更新仍在主线程完成。
        pending = self._pending_network
        if pending is not None and pending[0].done():
            self._pending_network = None
            if pending[2] == self._session_version:
                try:
                    result, error, code = pending[0].result()
                except Exception:
                    logger.exception("柜端网络任务异常")
                    result, error, code = None, "请求异常，请核实设备状态", "NETWORK_ERROR"
                pending[1](result, error, code)
        if self.state in ("STARTING", "OPERATING", "RECOVERING"):
            if time.time() >= self.session_expire_time:
                self._show_result("人脸会话已过期，请重新刷脸并核实借还记录；当前结果尚未确认", False)
                return
            if self._pending_network is None and time.time() >= self.next_operation_check:
                self.next_operation_check = time.time() + 1.5
                if self.state == "STARTING" and self._retry_start:
                    self._submit_network(self._retry_start, self._on_operation_started)
                elif self.state == "RECOVERING":
                    self._submit_network(lambda: self.api_client.get_active_operation(self.face_session_token), self._on_operation_recovered)
                elif self.state == "OPERATING":
                    self._submit_network(lambda: self.api_client.get_operation(self.operation_id, self.face_session_token), self._on_operation_snapshot)

    def _start_operation(self, request, action):
        self.state = "STARTING"
        self.operation_action = action
        # 未收到受理响应时保留同一个业务请求编号，重试只更新设备验签 nonce。
        self._retry_start = request
        self.next_operation_check = time.time() + 1.5
        if self.ui:
            self.ui.set_status("正在申请取还操作，请稍候...", "#38bdf8")
        self._submit_network(request, self._on_operation_started)

    def _on_operation_started(self, result, error, code):
        if isinstance(result, dict) and (result.get("id") or result.get("operationId")):
            self.operation_id = result.get("id") or result["operationId"]
            self._retry_start = None
            self.state = "OPERATING"
            self.next_operation_check = 0.0
            self._on_operation_snapshot(result, "", "")
        elif code in ("NETWORK_ERROR", "INVALID_RESPONSE", "SERVER_ERROR"):
            if self.ui:
                self.ui.set_status("受理结果尚未确认，正在重试核实，请勿重复操作", "#facc15")
        else:
            self._show_result(error or "服务端未返回有效操作，请重新刷脸核实", False)

    def _on_face_authenticated(self, result, error, code):
        if not isinstance(result, dict) or not result.get("faceSessionToken"):
            self._show_result(error or "人脸认证未获授权，请联系管理员核实身份", False)
            return
        self.face_session_token = result["faceSessionToken"]
        self.authenticated_user = result.get("user") or {}
        self.active_reservations = result.get("activeReservations") or []
        self.active_borrows = result.get("activeBorrows") or []
        self.session_expire_time = time.time() + min(int(result.get("expiresIn", 300)), 300) - 5
        self.state = "RECOVERING"
        self.next_operation_check = 0.0
        if self.ui:
            self.ui.set_user_info(f"欢迎：{self.authenticated_user.get('name', '用户')}")
            self.ui.set_tasks(self.active_reservations, self.active_borrows)
            self.ui.set_status("认证通过，正在检查未完成操作...", "#38bdf8")

    def _on_operation_recovered(self, result, error, code):
        if error:
            if self.ui:
                self.ui.set_status("暂时无法核实已有操作，正在重试...", "#facc15")
            return
        if isinstance(result, dict) and result.get("id"):
            self.operation_id = result["id"]
            self.operation_action = result.get("action", "PICKUP")
            self.state = "OPERATING"
            self._on_operation_snapshot(result, "", "")
        else:
            self.state = "AUTHENTICATED"
            self.state_expire_time = min(time.time() + 45.0, self.session_expire_time)
            if self.ui:
                self.ui.set_status("请选择预约取钥、归还，或输入房间号借钥", "#4ade80")

    def _on_operation_snapshot(self, result, error, code):
        if self.state != "OPERATING":
            return
        if not isinstance(result, dict):
            if self.ui:
                self.ui.set_status(error or "正在核实设备状态，请稍候", "#facc15")
            return
        status = result.get("status")
        if status == "SUCCESS":
            self._show_result("归还成功，借还记录已结算" if self.operation_action == "RETURN" else "取钥成功，请按预约时间归还", True)
        elif status in ("FAILED", "TIMEOUT", "CANCELLED"):
            self._show_result(result.get("errorMessage") or {"FAILED": "操作失败，请查看记录或联系管理员", "TIMEOUT": "设备操作超时，请核实钥匙状态", "CANCELLED": "操作已取消"}[status], False)
        elif self.ui:
            events = result.get("events") or []
            latest = max(events, key=lambda event: event.get("seq", 0), default={})
            prompt = {
                "DOOR_OPEN": "柜门已打开，请完成取还并关闭柜门",
                "WAITING_REMOVE": "请取走钥匙，然后关闭柜门",
                "KEY_REMOVED": "检测到钥匙离柜，请关闭柜门并等待确认",
                "KEY_RETURNED": "检测到钥匙放入，正在核对 RFID",
                "RFID_CONFIRMED": "钥匙已核对，请关闭柜门并等待归还确认",
                "HOMING": "柜机复位中，正在等待最终结果",
            }.get(latest.get("type") or latest.get("event"), "指令已受理，正在等待设备执行结果")
            self.ui.set_status(prompt, "#38bdf8")

    def _show_result(self, message, success):
        self.state = "RESULT"
        self.state_expire_time = time.time() + 6.0
        self._retry_start = None
        if self.ui:
            self.ui.set_status(message, "#4ade80" if success else "#ef4444")

    def _video_loop(self):
        self._network_tick()
        if not self.cap or not self.cap.isOpened():
            if self.root:
                self.root.after(100, self._video_loop)
            return

        ret, frame = self.cap.read()
        if ret and frame is not None:
            now = time.time()
            display = frame.copy()

            if self.state == "SCANNING" and self._pending_network is None:
                bbox = self.face_engine.detect_face(frame)
                if bbox is not None:
                    x, y, w, h = bbox
                    cv2.rectangle(display, (x, y), (x+w, y+h), (0, 255, 255), 2)
                    cv2.putText(display, "Face Detected", (x, max(20, y-10)),
                                cv2.FONT_HERSHEY_SIMPLEX, 0.6, (0, 255, 255), 2)

                    # 0.8 秒防抖执行一次活体与比对
                    if now - self.last_face_check_time > 0.8:
                        self.last_face_check_time = now
                        aligned_face = self.face_engine.align_face(frame, bbox)

                        # 1. 活体防攻击
                        passed, live_score, reason = self.liveness_detector.check_liveness(aligned_face)
                        if not passed:
                            logger.warning(f"活体检测拦截: {reason} (置信度={live_score})")
                            if self.ui:
                                self.ui.set_status(f"活体拦截: {reason}", "#ef4444")
                        else:
                            # 2. 1:N 生物特征比对
                            student_no, conf = self.face_engine.match_1_n(aligned_face, threshold=self.conf_threshold)
                            if student_no:
                                logger.info(f"人脸比对成功: 学号={student_no}, 置信度={conf:.2f}")
                                if self.ui:
                                    self.ui.set_status("人脸命中，正在换取机柜授权会话...", "#38bdf8")

                                # 3. HMAC 上报申请 FaceSessionToken
                                self.state = "AUTHENTICATING"
                                self._submit_network(
                                    lambda no=student_no, score=float(conf): self.api_client.auth_face(no, score, True),
                                    self._on_face_authenticated,
                                )
                            else:
                                if self.ui:
                                    self.ui.set_status("未登记的人脸，请先在系统录入", "#facc15")

            elif self.state == "AUTHENTICATED":
                if now > self.state_expire_time:
                    logger.info("认证会话超时未操作，自动重置")
                    self._reset_to_scanning()

            elif self.state == "RESULT":
                if now > self.state_expire_time:
                    self._reset_to_scanning()

            if self.ui:
                self.ui.update_frame(display)

        if self.root:
            self.root.after(33, self._video_loop)

if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="RK3588 边缘人脸识别与触屏存取柜终端")
    parser.add_argument("--config", default="config.yaml", help="配置文件路径")
    parser.add_argument("--enroll", help="录入指定学号人员的生物特征模板")
    parser.add_argument("--cam", type=int, default=0, help="摄像头设备编号")
    args = parser.parse_args()

    app = RK3588EdgeApplication(config_path=args.config)
    if args.enroll:
        app.run_enrollment(args.enroll, cam_idx=args.cam)
    else:
        app.run_touchscreen_kiosk(cam_idx=args.cam)
