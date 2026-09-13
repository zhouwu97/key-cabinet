# RK3588 边缘端人脸识别与触控交互工程

本工程为《基于微信小程序与人脸识别的智能钥匙自助存取柜》中部署于 **瑞芯微 RK3588 边缘计算主板** 的现场人脸识别、活体防攻击与触控屏幕交互应用。

---

## 1. 核心技术特性

### 1.1 生物特征隐私保护规范
- **严禁长期存储原始人脸图像**：系统仅在首次录入时将面部提取归一化为 **512 维生物特征向量**，结合设备安全通信密钥采用 **AES-256-GCM** 强加密持久化为 `.enc` 二进制密文存储在本地 `templates/` 目录，杜绝明文向量或照片泄露风险。
- 比对全程在内存中进行余弦距离计算（Cosine Similarity），未检测到有效人脸时严格返回空，严禁使用画面中央区域降级。

### 1.2 静默活体防攻击算法 (Presentation Attack Detection, PAD)
- **拉普拉斯纹理高频损失分析**：独立阈值（默认 85.0）辨识彩色/黑白打印纸张与翻印材质的边缘散焦虚化与模糊降质。
- **频域 FFT 摩尔尖峰峰均功率比 (PAPR) 分析**：通过中高频带峰均功率比检验，精准捕捉手机、平板、液晶屏幕翻拍产生的点阵周期性网格尖峰。
- **色彩空间反光与高光溢出分析**：检测电子屏幕玻璃保护膜与相纸对环境光的强反光特性，多维加权概率打分融合（通过阈值 0.85），有效抵御假体攻击。

### 1.3 柜机设备安全认证 (HMAC-SHA256)
- 与后端通信强制启用硬件 HMAC-SHA256 鉴权。
- 每次通信自动携带请求时间戳（±300秒防过期）与动态 Nonce（防重放攻击）。
- 只有经过本地活体与 1:N 识别通过后，向服务端请求临时有效（5分钟）的 `FaceSessionToken`，现场出钥严格绑定该会话令牌。

---

## 2. 目录结构

```text
edge/rk3588/
├── config.yaml               # 终端配置 (服务端地址、机柜密钥、摄像头编号、阈值)
├── requirements.txt          # Python 依赖清单
├── README.md                 # 开发与部署说明
└── face_app/
    ├── __init__.py
    ├── api_client.py         # HMAC-SHA256 签名与后端接口通信
    ├── face_engine.py        # 人脸检测、512-d 特征提取与 1:N 余弦比对
    ├── liveness_detector.py  # 静默活体防攻击检测算法
    ├── ui.py                 # 触控交互虚拟数字键盘与取景界面
    └── app.py                # 终端主循环调度状态机
```

---

## 3. 运行指南

### 3.1 环境安装
```bash
cd edge/rk3588
pip install -r requirements.txt
```

### 3.2 人脸特征模板录入
在柜机现场或管理终端执行人脸录入（正对摄像头，检测到人脸绿框后按键盘 `s` 保存模板）：
```bash
python face_app/app.py --enroll 20230001
```

### 3.3 启动自助刷脸取钥终端
```bash
python face_app/app.py --config config.yaml
```
用户正对摄像头，活体与比对通过后，可选择本柜的有效预约取钥、选择借用记录归还，或输入房间号借用免审批且无预约冲突的钥匙。终端先恢复未完成操作，指令受理后持续查询设备状态，只有 `SUCCESS` 才显示成功；网络中断时沿用原请求编号核实受理结果。

### 3.4 柜端接口

以下接口均需 `X-Cabinet-ID`、`X-Timestamp`、`X-Nonce`、`X-Signature` 设备签名。除人脸认证与房间查询外，还需 `Authorization: Bearer <faceSessionToken>`；令牌绑定用户和柜机，有效期由认证响应的 `expiresIn` 给出，当前为 300 秒。

| 方法与路径（前缀 `/api/v1/cabinet`） | 请求与用途 |
| --- | --- |
| `POST /auth/face` | `studentNo`、`confidence`、`livenessPassed`；返回本柜当前可取预约、可还记录和人脸令牌 |
| `GET /keys/match-room?roomNo=101` | 查询本柜房间钥匙 |
| `POST /device-operations/pickup` | `reservationId`、`clientRequestId`；按预约取钥并沿用应还时间 |
| `POST /device-operations/return` | `borrowRecordId`、`clientRequestId`；向会话绑定柜机归还 |
| `GET /device-operations/active` | 恢复当前用户在本柜的未完成操作，无记录返回 `data: null` |
| `GET /device-operations/:id` | 获取操作状态及设备事件，不能查询其他柜机的操作 |
| `POST /device-operations/:id/cancel` | 请求安全取消；拒绝时应继续查询进度 |
| `POST /direct-dispense` | `requestId`、`roomNo` 或 `keyId`；现场免审批直借 |

每次重试生成新的签名 nonce，但同一取还任务必须复用 `clientRequestId`（直借使用 `requestId`），防止网络超时后重复发指令。`202` 仅代表受理，机械动作结果以进度接口为准。柜机密钥不能下发到微信小程序；本地录入仍使用上面的 `--enroll` 命令，小程序身份资料核验与人脸模板录入是两个步骤。
