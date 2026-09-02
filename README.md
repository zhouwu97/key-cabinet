# 智能钥匙自助借还系统 (Key Cabinet Self-Service System)

## 📌 项目概述

- **项目名称**: 智能钥匙自助借还系统
- **技术栈**: 微信小程序 + TypeScript + SCSS
- **架构主线**: 微信小程序 ➔ 业务后台 REST API ➔ MQTT Broker ➔ 柜控硬件 (RK3588/K230/ESP32)
- **当前发布版本**: `v0.3.0-product-ready` (已推送到远端并在微信模拟器中完成全流程闭环验证)

---

## 🏆 阶段里程碑与演进路线

| 版本 Tag | 阶段代号 | 核心交付内容 | 验收状态 |
| :--- | :--- | :--- | :---: |
| `v0.1.0` | **Stage 1** | 数据模型定义、Service 接口层与 Mock 仓库持久化 | ✅ 已完成 |
| `v0.2.0-mock-closed-loop` | **Stage 2** | Mock 设备服务闭环、预约/取钥/借用/归还完整流转、24 项自动化测试 | ✅ 已完成 |
| `v0.2.1-runtime-verified` | **Stage 2.1** | WXML 运行时 ViewModel 预计算重构、WXML 语法与非小程序标签 Linter | ✅ 已完成 |
| **`v0.3.0-product-ready`** | **Stage 3 & 4** | **9 大复用组件沉淀、全页面 6 状态完整化、Operation 旗舰展示流、5 份核心后台与 MQTT 协议规范冻结** | ✅ **已完成** |
| `v0.4.0-backend-integrated` | **Stage 5** | 替换 Mock 服务为真实后端 RESTful API（小程序形态与行为保持稳定） | 🚀 下一阶段 |
| `v0.5.0-device-connected` | **Stage 6** | 业务后台接入真实 MQTT Broker 与 ESP32 柜控硬件 | 规划中 |
| `v0.6.0-identity-integrated` | **Stage 7** | RK3588 / K230 人脸识别与现场屏端交互联调 | 规划中 |
| `v0.7.0-rfid-closed-loop` | **Stage 8** | RFID 实物天线读取与在位检测物理闭环 | 规划中 |
| `v1.0.0` | **Stage 9** | 生产就绪与结题交付 | 规划中 |

---

## 📚 冻结的系统级规范文档 (`docs/`)

- [02-DATA-MODEL.md](docs/02-DATA-MODEL.md) - 完整领域数据模型定义与 SQL DDL 字典
- [03-STATE-MACHINE.md](docs/03-STATE-MACHINE.md) - 钥匙、预约、借还与设备操作 4 套核心状态机及状态转移矩阵
- [04-API-CONTRACT.md](docs/04-API-CONTRACT.md) - 后台 RESTful API 契约 V1 (`/me`, `/keys`, `/reservations`, `/borrow-records`, `/device-operations`)
- [05-MQTT-PROTOCOL.md](docs/05-MQTT-PROTOCOL.md) - 柜机 MQTT Topic 树与 JSON Payload 报文协议
- [06-DEVICE-OPERATION.md](docs/06-DEVICE-OPERATION.md) - 6 阶段设备操作时序契约、超时保护与现场恢复机制
- [07-ERROR-CODES.md](docs/07-ERROR-CODES.md) - 全系统错误码对齐字典
- [PRD-V1.md](docs/PRD-V1.md) - 智能钥匙借还系统需求规格说明书

---

## 🧩 9 大公共组件库 (`miniprogram/components/`)

1. `status-badge`：统一状态标签（主题色彩、圆点动效）；
2. `section-header`：统一模块标题与副标题，支持右侧操作插槽；
3. `empty-state`：通用空状态插图、文案与引导行动按钮；
4. `error-state`：网络/接口异常展示与「重新加载」重试按钮；
5. `device-status`：钥匙柜硬件健康、在位统计及物理位置卡片；
6. `key-card`：统一钥匙卡片（房间号、名称、所在柜、审批规则、点击交互）；
7. `reservation-card`：统一预约卡片（取钥窗口倒计时、预计归还、现场取钥/取消）；
8. `borrow-card`：统一借还卡片（借出时间、应还时间、逾期警示高亮、一键归还）；
9. `operation-stepper`：旗舰级竖向步进器（进度计数 `3 / 6`、呼吸动效、全中文友好流转）。

---

## 🧪 自动化测试与静态验证

```bash
# 1. 静态检查 (TypeScript 编译 + 21 个 JSON 组件路径校验 + 19 个 WXML 模板语法分析)
npm run check

# 2. Stage 3 验收清单核对 (核对 6 份协议 + 9 个组件 + 8 大页面状态)
npm run checklist

# 3. 业务闭环与故障注入自动化测试套件 (24/24 项用例)
npm test
```

---

**最后更新**: 2026-09-02 (v0.3.0-product-ready)
