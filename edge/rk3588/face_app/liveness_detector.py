import cv2
import numpy as np
import logging
from typing import Tuple

logger = logging.getLogger("LivenessDetector")

class LivenessDetector:
    """
    RK3588 静默活体防攻击检测器 (Presentation Attack Detection, PAD)。
    针对三类典型物理假体攻击进行多维特征融合鉴别：
    1. 纸质打印照片攻击：低清晰度、边缘散焦虚化 (Laplacian 方差检验)
    2. 电子屏幕翻拍攻击：屏幕像素点阵周期性摩尔条纹 (FFT 频域峰均功率比 PAPR 尖峰检测)
    3. 反光介质/屏幕过曝攻击：翻拍玻璃介质高光死白 (HSV 色彩空间 V 通道过曝分析)
    """
    def __init__(
        self,
        laplacian_threshold: float = 85.0,
        accept_score: float = 0.85,
        moire_papr_threshold: float = 35.0,
        glare_ratio_threshold: float = 0.15,
        **kwargs
    ):
        self.laplacian_threshold = float(laplacian_threshold)
        self.accept_score = float(accept_score)
        # 兼容配置项名称
        self.moire_papr_threshold = float(kwargs.get("fft_high_freq_threshold") or moire_papr_threshold)
        if self.moire_papr_threshold <= 1.0:
            # 若误配为百分比，映射到 PAPR 经验阈值
            self.moire_papr_threshold = 35.0
        self.glare_ratio_threshold = float(glare_ratio_threshold)

    def check_liveness(self, face_bgr: np.ndarray) -> Tuple[bool, float, str]:
        """
        综合多特征融合评定活体可信度。
        :param face_bgr: 裁剪后的人脸区域图像 (BGR 格式)
        :return: (passed, confidence_score, reason)
        """
        if face_bgr is None or face_bgr.size == 0:
            return False, 0.0, "人脸区域为空"

        # 1. 纹理清晰度与拉普拉斯方差检测 (检测照片打印虚化与翻拍降质)
        gray = cv2.cvtColor(face_bgr, cv2.COLOR_BGR2GRAY)
        laplacian_var = float(cv2.Laplacian(gray, cv2.CV_64F).var())
        if laplacian_var < self.laplacian_threshold:
            fail_score = min(0.50, float(laplacian_var / max(1.0, self.laplacian_threshold) * 0.5))
            return False, fail_score, f"纹理模糊 (方差={laplacian_var:.1f} < {self.laplacian_threshold:.1f})，疑似低清打印照片"

        # 归一化纹理得分 (0.0 ~ 1.0)
        s_tex = min(1.0, laplacian_var / (self.laplacian_threshold * 1.5))

        # 2. 屏幕周期网格与摩尔条纹检测 (基于傅里叶频谱频带峰均功率比 PAPR 分析)
        f = np.fft.fft2(gray.astype(np.float32))
        fshift = np.fft.fftshift(f)
        power_spectrum = np.abs(fshift)

        h, w = gray.shape
        cy, cx = h // 2, w // 2
        Y, X = np.ogrid[:h, :w]
        dist = np.sqrt((X - cx)**2 + (Y - cy)**2)

        # 关注中高频特征环带 (避开直流零频与极限边缘噪点)
        band_mask = (dist >= 12) & (dist <= min(cy, cx) - 4)
        if np.any(band_mask):
            band_vals = power_spectrum[band_mask]
            mean_power = float(np.mean(band_vals)) + 1e-5
            max_power = float(np.max(band_vals))
            papr = max_power / mean_power
        else:
            papr = 10.0

        if papr > self.moire_papr_threshold:
            return False, 0.40, f"检测到高频网格摩尔尖峰 (PAPR={papr:.1f} > {self.moire_papr_threshold:.1f})，疑似电子屏幕翻拍"

        # 正常人脸频域平滑衰减，PAPR 较小
        s_freq = max(0.0, min(1.0, 1.0 - max(0.0, papr - 10.0) / max(1.0, self.moire_papr_threshold - 10.0) * 0.3))

        # 3. 色彩空间反光与色偏检测 (HSV 反光过曝分析)
        hsv = cv2.cvtColor(face_bgr, cv2.COLOR_BGR2HSV)
        v_channel = hsv[:, :, 2]
        over_exposed_ratio = float(np.sum(v_channel > 250) / v_channel.size)

        if over_exposed_ratio > self.glare_ratio_threshold:
            return False, 0.45, f"人脸反光过强 (高光占比={over_exposed_ratio:.2f} > {self.glare_ratio_threshold:.2f})，疑似反光介质或翻拍"

        s_glare = max(0.0, min(1.0, 1.0 - (over_exposed_ratio / self.glare_ratio_threshold) * 0.3))

        # 4. 多特征连续融合评分 (0.0 ~ 1.0)
        composite_score = 0.40 * s_tex + 0.35 * s_freq + 0.25 * s_glare
        composite_score = round(min(0.99, max(0.0, composite_score)), 4)

        if composite_score < self.accept_score:
            return False, composite_score, f"活体综合置信度不足 ({composite_score:.2f} < {self.accept_score:.2f})"

        return True, composite_score, "活体生物特征检测通过"
