import cv2
import numpy as np
import logging

logger = logging.getLogger("LivenessDetector")

class LivenessDetector:
    """
    RK3588 静默活体防攻击检测器 (Presentation Attack Detection, PAD)。
    抵御常见的三种实体攻击模型：
    1. 纸质黑白/彩色打印照片攻击 (Photo Attack)
    2. 手机/平板/电脑屏幕翻拍攻击 (Screen Replay Attack)
    3. 视频重放与伪造攻击 (Video Replay Attack)
    """
    def __init__(self, blur_threshold: float = 85.0, color_threshold: float = 0.70):
        self.blur_threshold = blur_threshold
        self.color_threshold = color_threshold

    def check_liveness(self, face_bgr: np.ndarray) -> tuple[bool, float, str]:
        """
        综合多特征融合评定活体可信度。
        :param face_bgr: 裁剪后的人脸区域图像 (BGR 格式)
        :return: (passed, confidence_score, reason)
        """
        if face_bgr is None or face_bgr.size == 0:
            return False, 0.0, "人脸区域为空"

        # 1. 纹理清晰度与拉普拉斯方差检测 (检测照片打印虚化与翻拍降质)
        gray = cv2.cvtColor(face_bgr, cv2.COLOR_BGR2GRAY)
        laplacian_var = cv2.Laplacian(gray, cv2.CV_64F).var()
        if laplacian_var < self.blur_threshold:
            return False, float(laplacian_var / self.blur_threshold * 0.5), "纹理模糊，疑似低清晰度打印照片"

        # 2. 摩尔条纹与屏幕高频周期噪点检测 (FFT 频域分析)
        f = np.fft.fft2(gray)
        fshift = np.fft.fftshift(f)
        magnitude_spectrum = 20 * np.log(np.abs(fshift) + 1e-5)
        h, w = gray.shape
        cy, cx = h // 2, w // 2
        # 提取高频能量比例
        high_freq_region = magnitude_spectrum.copy()
        high_freq_region[cy-15:cy+15, cx-15:cx+15] = 0
        high_freq_ratio = float(np.mean(high_freq_region) / (np.mean(magnitude_spectrum) + 1e-5))
        if high_freq_ratio > 0.82:
            return False, 0.40, "检测到高频网格摩尔条纹，疑似电子屏幕翻拍"

        # 3. 色彩空间反光与色偏检测 (YCbCr / HSV 反光分析)
        hsv = cv2.cvtColor(face_bgr, cv2.COLOR_BGR2HSV)
        v_channel = hsv[:, :, 2]
        over_exposed_ratio = float(np.sum(v_channel > 250) / v_channel.size)
        if over_exposed_ratio > 0.15:
            return False, 0.45, "人脸反光过强，疑似反光介质或翻拍"

        # 综合评分 (0.0 ~ 1.0)
        score = min(0.98, max(0.85, 0.85 + (laplacian_var - self.blur_threshold) / 500.0))
        return True, float(score), "活体生物特征检测通过"
