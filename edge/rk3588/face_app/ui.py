import tkinter as tk
from tkinter import messagebox
from PIL import Image, ImageTk
import cv2
from typing import Callable, Optional

class TouchscreenKioskUI:
    """
    RK3588 触摸屏交互界面。
    包含人脸取景框、认证状态提示、虚拟数字键盘与出钥状态反馈。
    """
    def __init__(self, root: tk.Tk, on_room_submit: Callable[[str], None], on_cancel: Callable[[], None]):
        self.root = root
        self.root.title("智能钥匙柜现场取钥终端 (RK3588)")
        self.root.geometry("800x480")
        self.root.configure(bg="#1e1e2d")

        self.on_room_submit = on_room_submit
        self.on_cancel = on_cancel
        self.current_room = ""

        self._build_ui()

    def _build_ui(self):
        # 1. 顶部标题栏
        header = tk.Label(self.root, text="智能钥匙自助存取终端 - 请正对摄像头刷脸",
                          font=("Helvetica", 16, "bold"), fg="#ffffff", bg="#1e1e2d")
        header.pack(pady=10)

        # 2. 中部主体区 (左：视频流取景框；右：键盘与操作区)
        content_frame = tk.Frame(self.root, bg="#1e1e2d")
        content_frame.pack(fill=tk.BOTH, expand=True, padx=20, pady=5)

        # 视频预览区
        self.video_label = tk.Label(content_frame, bg="#000000", width=360, height=320)
        self.video_label.pack(side=tk.LEFT, padx=10)

        # 交互面板
        right_panel = tk.Frame(content_frame, bg="#27293d", padx=15, pady=15)
        right_panel.pack(side=tk.RIGHT, fill=tk.BOTH, expand=True, padx=10)

        # 状态提示
        self.status_label = tk.Label(right_panel, text="等待人脸进入取景框...",
                                     font=("Helvetica", 12), fg="#4ade80", bg="#27293d")
        self.status_label.pack(pady=5)

        # 用户信息
        self.user_label = tk.Label(right_panel, text="身份：未认证",
                                   font=("Helvetica", 11), fg="#94a3b8", bg="#27293d")
        self.user_label.pack(pady=2)

        # 房间号输入回显
        tk.Label(right_panel, text="请输入房间号:", font=("Helvetica", 10), fg="#cbd5e1", bg="#27293d").pack(anchor="w")
        self.room_display = tk.Label(right_panel, text="---", font=("Helvetica", 18, "bold"),
                                     fg="#38bdf8", bg="#1e1e2d", width=12, height=1)
        self.room_display.pack(pady=5)

        # 虚拟数字小键盘 (触屏专用)
        keypad_frame = tk.Frame(right_panel, bg="#27293d")
        keypad_frame.pack(pady=5)

        buttons = [
            ('1', 0, 0), ('2', 0, 1), ('3', 0, 2),
            ('4', 1, 0), ('5', 1, 1), ('6', 1, 2),
            ('7', 2, 0), ('8', 2, 1), ('9', 2, 2),
            ('退格', 3, 0), ('0', 3, 1), ('确定', 3, 2),
        ]

        for text, r, c in buttons:
            action = lambda val=text: self._on_key_press(val)
            color = "#3b82f6" if text == "确定" else ("#ef4444" if text == "退格" else "#334155")
            btn = tk.Button(keypad_frame, text=text, font=("Helvetica", 12, "bold"),
                            width=4, height=1, fg="#ffffff", bg=color, relief=tk.FLAT,
                            command=action)
            btn.grid(row=r, column=c, padx=3, pady=3)

    def _on_key_press(self, key: str):
        if key == "退格":
            self.current_room = self.current_room[:-1]
        elif key == "确定":
            if self.current_room and self.on_room_submit:
                self.on_room_submit(self.current_room)
        else:
            if len(self.current_room) < 6:
                self.current_room += key
        self.room_display.config(text=self.current_room if self.current_room else "---")

    def update_frame(self, frame_bgr):
        """刷新视频帧显示"""
        rgb = cv2.cvtColor(frame_bgr, cv2.COLOR_BGR2RGB)
        resized = cv2.resize(rgb, (360, 320))
        img = Image.fromarray(resized)
        imgtk = ImageTk.PhotoImage(image=img)
        self.video_label.imgtk = imgtk
        self.video_label.configure(image=imgtk)

    def set_status(self, text: str, color: str = "#4ade80"):
        self.status_label.config(text=text, fg=color)

    def set_user_info(self, text: str):
        self.user_label.config(text=text)

    def clear_room_input(self):
        self.current_room = ""
        self.room_display.config(text="---")
