# key-cabinet — 智能钥匙借还微信小程序

微信小程序作为智能钥匙借还系统的用户交互端：用户身份绑定、钥匙查询、借用申请、扫码取还、设备状态反馈、借还记录及管理员审批。设备通信统一走 DeviceService 封装，第一阶段用 Mock 独立开发，后续接入 MQTT over WebSocket 与 ESP 设备通信，业务层不依赖具体硬件实现。

## 文档

- [V1 产品设计方案（PRD）](docs/PRD-V1.md) — 页面结构、状态机、消息协议、异常场景的唯一依据

## 目录结构

```text
miniprogram/
├── pages/          # home 首页 / keys 钥匙 / records 记录 / profile 我的（4 Tab）
│                   # key-detail 详情 / borrow 申请 / scan 扫码 / operation 设备操作 / admin 管理中心
├── services/
│   └── device/     # DeviceService 接口 + Mock 实现（MQTT 实现阶段五替换）
├── models/         # User / Key / Device / BorrowOrder 与状态枚举
├── constants/      # USE_MOCK、MQTT topic、状态中文标签
└── utils/          # requestId 生成等
```

## 快速开始

```bash
npm install
npm run check   # tsc 类型检查 + app.json 页面完整性断言
```

微信开发者工具导入项目根目录（appid 已配置在 project.config.json），编译即可预览。设备操作页内置「Mock 事件流演示」按钮，可在无硬件情况下走完取钥事件时序。

## Mock 模式

`miniprogram/constants/config.ts` 中 `USE_MOCK = true` 时所有设备交互走 Mock（事件时序见 PRD 第三十五节）。接真实设备时在 `miniprogram/services/device/index.ts` 单点替换为 MQTT 实现，页面层不动。

## 与设备组的接口边界

```text
输入：deviceId / keyId / action / requestId
输出：requestId / event / errorCode / timestamp
Topic：keybox/{deviceId}/command|event|status|heartbeat
```

设备内部实现（ESP32 / ESP→RK3588→STM32 等）与小程序无关。

## 开发顺序（PRD 第四十三节）

1. 数据与状态 — 已完成（models / constants）
2. Mock DeviceService — 已完成（services/device）
3. 核心页面 — 当前为占位骨架，待实现
4. 扫码
5. 接真实 MQTT
6. 接后端 API
