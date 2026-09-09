import os
import sys
import time
import argparse
import yaml
import cv2
import logging

from api_client import CabinetApiClient
from liveness_detector import LivenessDetector
from face_engine import FaceEngine

logging.basicConfig(level=logging.INFO, format="%(asctime)s [%(levelname)s] %(name)s: %(message)s")
logger = logging.getLogger("RK3588EdgeApp")

class RK3588EdgeApplication:
    def __init__(self, config_path: str = "config.yaml"):
        with open(config_path, "r", encoding="utf-8") as f:
            self.cfg = yaml.safe_load(f)

        cab_cfg = self.cfg.get("cabinet", {})
        self.api_client = CabinetApiClient(
            server_url=cab_cfg.get("server_url", "http://127.0.0.1:8080"),
            device_id=cab_cfg.get("device_id", "CAB001"),
            device_secret=cab_cfg.get("device_secret", "cab_sec_CAB001"),
            timeout=cab_cfg.get("timeout_seconds", 5)
        )

        rec_cfg = self.cfg.get("recognition", {})
        self.liveness_detector = LivenessDetector(
            blur_threshold=rec_cfg.get("liveness_threshold", 85.0)
        )
        self.face_engine = FaceEngine(
            template_dir=rec_cfg.get("template_dir", "templates")
        )

        self.conf_threshold = rec_cfg.get("confidence_threshold", 0.80)
        self.face_session_token = None
        self.authenticated_user = None

    def run_enrollment(self, student_no: str, cam_idx: int = 0):
        """录入指定学号的人脸特征模板"""
        logger.info(f"开启摄像头 (index={cam_idx}) 进行人脸录入...")
        cap = cv2.VideoCapture(cam_idx)
        if not cap.isOpened():
            logger.error("无法打开摄像头")
            return False

        logger.info("请正对摄像头，按 's' 采集保存，按 'q' 退出")
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
                cv2.putText(display, "Face Ready: Press 's'", (x, y-10),
                            cv2.FONT_HERSHEY_SIMPLEX, 0.6, (0, 255, 0), 2)

            cv2.imshow("Face Enrollment - RK3588", display)
            key = cv2.waitKey(1) & 0xFF
            if key == ord('s') and bbox:
                success = self.face_engine.enroll_face(student_no, frame)
                break
            elif key == ord('q'):
                break

        cap.release()
        cv2.destroyAllWindows()
        return success

    def run_live_terminal(self, cam_idx: int = 0):
        """运行现场刷脸自助取钥业务主循环"""
        logger.info(f"启动 RK3588 边缘识别终端... (Camera Index={cam_idx})")
        cap = cv2.VideoCapture(cam_idx)
        if not cap.isOpened():
            logger.error("无法打开摄像头外设")
            return

        state = "SCANNING"  # SCANNING -> AUTHENTICATED -> DISPENSING
        last_check_time = 0

        while True:
            ret, frame = cap.read()
            if not ret:
                time.sleep(0.05)
                continue

            display = frame.copy()
            now = time.time()

            if state == "SCANNING":
                bbox = self.face_engine.detect_face(frame)
                if bbox:
                    x, y, w, h = bbox
                    face_roi = frame[y:y+h, x:x+w]

                    # 1秒执行一次防抖识别
                    if now - last_check_time > 1.0:
                        last_check_time = now

                        # 1. 活体防攻击检测
                        passed, live_score, reason = self.liveness_detector.check_liveness(face_roi)
                        if not passed:
                            logger.warning(f"活体检测拦截: {reason} (得分={live_score:.2f})")
                            cv2.rectangle(display, (x, y), (x+w, y+h), (0, 0, 255), 2)
                            cv2.putText(display, f"Liveness Failed: {reason}", (x, y-10),
                                        cv2.FONT_HERSHEY_SIMPLEX, 0.5, (0, 0, 255), 2)
                        else:
                            # 2. 1:N 余弦特征比对
                            student_no, conf = self.face_engine.match_1_n(face_roi, threshold=self.conf_threshold)
                            if student_no:
                                logger.info(f"人脸比对命中: 学号={student_no}, 置信度={conf:.2f}")

                                # 3. 向服务端上报带 HMAC 签名的认证请求，换取 FaceSession
                                auth_res = self.api_client.auth_face(
                                    student_no=student_no,
                                    confidence=conf,
                                    liveness_passed=True
                                )
                                if auth_res and auth_res.get("faceSessionToken"):
                                    self.face_session_token = auth_res.get("faceSessionToken")
                                    self.authenticated_user = auth_res.get("user")
                                    state = "AUTHENTICATED"
                                    logger.info(f"✅ 认证成功: {self.authenticated_user.get('name')}，进入选房取钥状态")

                    cv2.rectangle(display, (x, y), (x+w, y+h), (255, 200, 0), 2)
                    cv2.putText(display, "Checking...", (x, y-10), cv2.FONT_HERSHEY_SIMPLEX, 0.5, (255, 200, 0), 2)

            elif state == "AUTHENTICATED":
                cv2.putText(display, f"Auth OK: {self.authenticated_user.get('name')} ({self.authenticated_user.get('studentNo')})",
                            (20, 40), cv2.FONT_HERSHEY_SIMPLEX, 0.8, (0, 255, 0), 2)
                cv2.putText(display, "Press '1' for Room 101, 'q' to cancel",
                            (20, 80), cv2.FONT_HERSHEY_SIMPLEX, 0.6, (255, 255, 255), 2)

                key = cv2.waitKey(1) & 0xFF
                if key == ord('1'):
                    logger.info("用户选择房间号 101，发起出钥指令...")
                    res = self.api_client.direct_dispense("101", "", self.face_session_token)
                    if res:
                        logger.info(f"出钥成功: 槽位={res.get('slotNo')}, 钥匙={res.get('keyName')}")
                    state = "SCANNING"
                    self.face_session_token = None
                elif key == ord('q'):
                    state = "SCANNING"
                    self.face_session_token = None

            cv2.imshow("RK3588 Face Kiosk Terminal", display)
            if cv2.waitKey(1) & 0xFF == 27: # ESC 退出
                break

        cap.release()
        cv2.destroyAllWindows()

if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="RK3588 边缘人脸识别与机柜终端应用")
    parser.add_argument("--config", default="config.yaml", help="配置文件路径")
    parser.add_argument("--enroll", help="录入指定人员学号的人脸特征模板")
    parser.add_argument("--cam", type=int, default=0, help="摄像头设备编号")
    args = parser.parse_args()

    app = RK3588EdgeApplication(config_path=args.config)
    if args.enroll:
        app.run_enrollment(args.enroll, cam_idx=args.cam)
    else:
        app.run_live_terminal(cam_idx=args.cam)
