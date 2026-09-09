import os
import sys
import time
import argparse
import yaml
import cv2
import logging
import tkinter as tk
from typing import Optional, Dict, Any

from api_client import CabinetApiClient
from liveness_detector import LivenessDetector
from face_engine import FaceEngine
from ui import TouchscreenKioskUI

logging.basicConfig(level=logging.INFO, format="%(asctime)s [%(levelname)s] %(name)s: %(message)s")
logger = logging.getLogger("RK3588EdgeApp")

class RK3588EdgeApplication:
    def __init__(self, config_path: str = "config.yaml"):
        # 优先检测 config.local.yaml
        local_cfg_path = os.path.join(os.path.dirname(config_path), "config.local.yaml")
        if os.path.exists(local_cfg_path):
            config_path = local_cfg_path
            logger.info(f"加载本地优先配置: {local_cfg_path}")
        else:
            logger.info(f"加载主配置文件: {config_path}")

        with open(config_path, "r", encoding="utf-8") as f:
            self.cfg = yaml.safe_load(f) or {}

        cab_cfg = self.cfg.get("cabinet", {})
        # 支持环境变量覆写真实机柜安全通信密钥
        device_secret = os.environ.get("CABINET_DEVICE_SECRET") or cab_cfg.get("device_secret", "")
        self.device_id = cab_cfg.get("device_id", "CAB001")
        self.device_secret = device_secret

        self.api_client = CabinetApiClient(
            server_url=cab_cfg.get("server_url", "http://127.0.0.1:8080"),
            device_id=self.device_id,
            device_secret=self.device_secret,
            timeout=cab_cfg.get("timeout_seconds", 5)
        )

        rec_cfg = self.cfg.get("recognition", {})
        live_cfg = self.cfg.get("liveness", {})

        self.liveness_detector = LivenessDetector(
            laplacian_threshold=live_cfg.get("laplacian_threshold", 85.0),
            accept_score=live_cfg.get("accept_score", 0.85),
            fft_high_freq_threshold=live_cfg.get("fft_high_freq_threshold", 0.80),
            glare_ratio_threshold=live_cfg.get("glare_ratio_threshold", 0.15)
        )

        self.face_engine = FaceEngine(
            template_dir=rec_cfg.get("template_dir", "templates"),
            device_secret=self.device_secret,
            model_path=rec_cfg.get("model_path")
        )

        self.conf_threshold = rec_cfg.get("confidence_threshold", 0.80)
        self.face_session_token: Optional[str] = None
        self.authenticated_user: Optional[Dict[str, Any]] = None

        # 状态机：SCANNING -> AUTHENTICATED -> DISPENSING -> RESULT
        self.state = "SCANNING"
        self.state_expire_time = 0.0
        self.last_face_check_time = 0.0

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
            on_cancel=self._on_cancel
        )

        self._reset_to_scanning()
        self.root.after(30, self._video_loop)

        # 优雅关闭捕获
        self.root.protocol("WM_DELETE_WINDOW", self._on_window_close)
        self.root.mainloop()

    def _on_window_close(self):
        if self.cap:
            self.cap.release()
        if self.root:
            self.root.destroy()
        logger.info("终端应用已退出")

    def _reset_to_scanning(self):
        self.state = "SCANNING"
        self.face_session_token = None
        self.authenticated_user = None
        if self.ui:
            self.ui.clear_room_input()
            self.ui.set_status("等待人脸进入取景框...", "#4ade80")
            self.ui.set_user_info("身份：未认证")

    def _on_cancel(self):
        logger.info("用户取消操作，重置为等待刷脸状态")
        self._reset_to_scanning()

    def _on_room_submit(self, room_no: str):
        """用户在虚拟触摸键盘输入房间号后点击确定"""
        if self.state != "AUTHENTICATED" or not self.face_session_token:
            if self.ui:
                self.ui.set_status("请先完成人脸认证再选择房间", "#facc15")
            return

        logger.info(f"用户提交取钥房间号: {room_no}，向服务端发送取钥请求...")
        self.state = "DISPENSING"
        if self.ui:
            self.ui.set_status(f"正在申请出钥 [{room_no}] ...", "#38bdf8")

        res = self.api_client.direct_dispense(room_no=room_no, key_id="", face_session_token=self.face_session_token)
        if res and res.get("operationId"):
            key_name = res.get("keyName", room_no)
            slot_no = res.get("slotNo", 1)
            logger.info(f"✅ 服务端已受理出钥: 槽位={slot_no}, 钥匙={key_name}")
            if self.ui:
                self.ui.set_status(f"出钥成功！请在 #{slot_no} 号槽位取走钥匙 [{key_name}]", "#4ade80")
            self.state = "RESULT"
            self.state_expire_time = time.time() + 6.0
        else:
            err_msg = "无可用钥匙或未授权借用该房间"
            logger.warning(f"❌ 出钥失败: {err_msg}")
            if self.ui:
                self.ui.set_status(f"出钥失败: {err_msg}", "#ef4444")
            self.state = "RESULT"
            self.state_expire_time = time.time() + 4.0

    def _video_loop(self):
        if not self.cap or not self.cap.isOpened():
            return

        ret, frame = self.cap.read()
        if ret and frame is not None:
            now = time.time()
            display = frame.copy()

            if self.state == "SCANNING":
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
                                auth_res = self.api_client.auth_face(
                                    student_no=student_no,
                                    confidence=conf,
                                    liveness_passed=True
                                )
                                if auth_res and auth_res.get("faceSessionToken"):
                                    self.face_session_token = auth_res.get("faceSessionToken")
                                    self.authenticated_user = auth_res.get("user", {})
                                    self.state = "AUTHENTICATED"
                                    self.state_expire_time = now + 45.0  # 45秒内需完成选房输入
                                    name = self.authenticated_user.get("name", student_no)
                                    logger.info(f"✅ 用户 [{name}] 认证通过，等待触屏选房")
                                    if self.ui:
                                        self.ui.set_status("认证成功！请在右侧键盘输入房间号", "#4ade80")
                                        self.ui.set_user_info(f"欢迎: {name} ({student_no})")
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
