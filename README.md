# Key Cabinet - 智能钥匙自助借还系统

## 项目概述

智能钥匙自助借还系统（Key Cabinet）是一个面向校园/企业的钥匙管理系统，支持微信小程序预约、自助取还钥匙、设备联动控制。

**当前阶段**: Sprint 6.1 实机验收修正收口（软硬件全闭环完成度 ~75%，推进实机验收）

## 最新进展

### ✅ Sprint 6.1 实机验收修正收口（2026-09-09）

针对实机真实性和物理闭环关键点进行了深度硬化，彻底解决文档比代码领先的问题：
1. **强制真实深度人脸模型与禁用伪特征**：默认部署并加载真实 512 维 MobileFaceNet ONNX 深度模型与 YuNet 检测器，生产模式 (`allow_handcrafted_fallback=false`) 下无模型严禁启动与人脸认证；仅在本地离线调试时允许手工 HOG 梯度回退。
2. **真实 5 点人脸仿射对齐**：YuNet 输出提取 [右眼, 左眼, 鼻尖, 右嘴角, 左嘴角] 5 关键点，通过 `cv2.estimateAffinePartial2D` 计算相似变换矩阵并经 `cv2.warpAffine` 旋转平移生成标准 112×112 对齐人脸。
3. **安全柜门未关绝不上报 SUCCESS**：出钥与归还流程等待用户关门 30 秒超时，未闭合时必须上报 `DOOR_OPEN_TIMEOUT` 并标记 `FAILED`，杜绝假闭环。
4. **ABORT 指令实时硬件抢占**：下行 `CMD_TYPE_ABORT` 在 MQTT 事件回调中立即拦截并调用 `motor_request_abort()` 断电失能电机，跳过阻塞的业务 worker 队列并立即广播终态；等待循环全面支持抢占；新任务启动时才重置 abort 标志。
5. **Inventory 物理盘点与数据库对账器**：构建 `InventoryReconciler` 直接注入 `MQTTDeviceGateway`，实时把微动在位同步到数据库，并比对实读 RFID 与绑定钥匙 `RFIDTag`，告警 `WRONG_KEY_IN_SLOT`、`MISSING_KEY`、`UNEXPECTED_KEY` 异常。
6. **真实 CSPRNG 密钥重置**：数据库迁移升级采用 PostgreSQL `pgcrypto` 扩展的 `gen_random_bytes(32)`，生成真正不可预测的高熵 64 位十六进制设备密钥。
7. **ESP32 硬件引脚冲突消除**：将 4G 模组电源使能引脚从 GPIO 4 调整为 GPIO 13，彻底消除与蜂鸣器 GPIO 4 的引脚冲突。
8. **微信服务通知模板配置统一**：小程序前端配置与服务端环境变量统一切换至标准配置体系。

## 技术栈

### 前端（微信小程序）
- TypeScript
- WXML/WXSS
- Domain-Driven Design
- Mock / 真实 HTTP API 双模式
- 微信服务通知订阅授权 (`wx.requestSubscribeMessage`)

### 后端（Go）
- **语言**: Go 1.26.2
- **Web 框架**: Gin
- **ORM**: GORM
- **数据库**: PostgreSQL 14+ (pgcrypto)
- **认证**: JWT (HS256) + 设备 HMAC-SHA256 防重放签名
- **物联网**: MQTT v3.1.1 QoS 1（TLS 待实机部署挂载 CA 证书）
- **迁移工具**: golang-migrate

### 边缘计算与固件
- **ESP32**: ESP-IDF v5.0+ FreeRTOS C 固件 (RC522 RFID, A4988 步进推杆, 槽位微动, 门磁, Wi-Fi STA, SNTP, 抢占式 Abort)
- **RK3588**: Python 3.8+ 边缘终端 (MobileFaceNet 512-d Embedding, YuNet 5点仿射对齐, AES-256-GCM 模板加密, 静默活体 PAD, Tkinter 触屏 Kiosk UI)

## 项目结构

```
key-cabinet/
├── miniprogram/          # 微信小程序前端
├── server/               # Go 后端服务
├── firmware/             # ESP32 机电一体化固件工程 (ESP-IDF C)
│   └── esp32/
│       ├── main/         # 步进电机、RC522、传感器微动、网络管理与 MQTT
│       └── CMakeLists.txt
├── edge/                 # 瑞芯微 RK3588 边缘计算工程 (Python)
│   └── rk3588/
│       ├── face_app/     # 人脸识别、活体防攻击、触屏 Kiosk UI 与签名通信
│       │   └── models/   # MobileFaceNet 与 YuNet ONNX 深度模型
│       └── tests/        # 边缘端自动化测试
└── docs/                 # 协议设计与架构文档
```

## 核心功能进展

### v0.3.1 - 产品级小程序（已完成）
- ✅ 微信小程序完整 UI/UX
- ✅ 钥匙浏览和搜索
- ✅ 预约创建和管理
- ✅ 取钥/还钥操作流程
- ✅ 借用历史记录
- ✅ 用户个人中心
- ✅ 微信服务通知订阅授权

### v0.4 - 软件闭环（已完成）
- ✅ 微信登录、JWT、用户资料与身份核验
- ✅ Key / Slot / Device 查询
- ✅ 预约冲突控制、审批、拒绝与自动过期
- ✅ BorrowRecord 和 DeviceOperation 事务落账
- ✅ 预约 → 取钥 → 借用 → 归还完整主链
- ✅ 取消、迟到事件和操作超时安全收敛

### v0.5 - 实体机电与网络固件（代码实现完成，待实体硬件验收）
- ✅ MQTT 设备通信网关 (QoS 1 保证、实时 Inventory 数据库对账器注入)
- ✅ ESP32 固件工程 (FreeRTOS 模块化架构)
- ✅ Wi-Fi STA 联网与 SNTP 毫秒授时
- 📐 4G 模组串口接口规划与引脚定义 (GPIO13 独立使能，待实插 SIM 拨号)
- ✅ RC522 13.56MHz 4 字节标准 UID 防错还
- ✅ 步进电机原点限位与 MQTT 抢占式异步硬件失能中止 (Abort Preemption)
- ✅ 物理全闭环保障：取还未确认关门 (DoorClosed) 绝不上报 SUCCESS，超时报告 DOOR_OPEN_TIMEOUT (FAILED)
- ✅ `status/inventory` 槽位物理状态上报与 `InventoryReconciler` 自动对账 (RFID 错还检测、失位检测)

### v0.6 - 边缘人脸识别与触屏终端（边缘终端框架完成，待真实模型与真人 PAD 验收）
- ✅ 强制部署真实深度人脸模型 (MobileFaceNet ONNX 512-d)，生产模式禁止静默回退手工梯度
- ✅ YuNet 5 点面部关键点仿射相似变换对齐 (`cv2.estimateAffinePartial2D` + `cv2.warpAffine` 映射至 112×112)
- ✅ AES-256-GCM 模板强加密持久化存储 (`.enc`)
- ✅ 静默活体防攻击 (PAD: Laplacian 散焦 + FFT 频域摩尔尖峰 PAPR + HSV 过曝分析)
- ✅ 柜机设备 HMAC-SHA256 防篡改防重放签名通信
- ✅ FaceSessionToken 现场授权与跨柜绑定强校验
- ✅ Tkinter 触控屏幕虚拟键盘与全流程业务串联

## 技术亮点

### 1. PostgreSQL 排他约束解决预约冲突

```sql
ALTER TABLE reservations
ADD CONSTRAINT reservations_no_overlap
EXCLUDE USING gist (
  key_id WITH =,
  tstzrange(pickup_window_start, expected_return_at) WITH &&
);
```

**优势**：
- 数据库层面保证时间冲突检测
- 并发安全，无法绕过
- 性能优于应用层锁

### 2. 事件驱动的设备网关

```go
type DeviceGateway interface {
    StartPickup(ctx, cmd) error          // 发送命令
	AbortOperation(ctx, cmd) error        // 安全中止执行中操作
    RegisterEventHandler(handler)        // 注册回调
}

type DeviceEventHandler interface {
    OnPickupSuccess(ctx, event) error    // 异步回调
    OnPickupFailed(ctx, event) error
}
```

**优势**：
- Mock 和 MQTT 实现一致
- 支持长时间异步操作
- 业务代码无需改动

### 3. 多认证方式支持

```
User (U001)
  ├─ UserIdentity (WECHAT, openid_xxx)    # 微信登录
  └─ UserIdentity (FACE, face_profile_001) # 人脸识别
```

**优势**：
- 灵活扩展登录方式
- 多种方式指向同一用户
- 便于后续添加新认证

### 4. Domain-Driven Design

前后端统一采用 DDD 架构：
- **Domain**: 领域模型 + 业务规则
- **Service**: 应用服务编排
- **Repository**: 数据访问抽象
- **Infrastructure**: 基础设施实现

## 文档

### 架构设计
- [后端架构](docs/BACKEND-ARCHITECTURE.md)
- [Sprint 4 总览](docs/sprints/SPRINT-4-OVERVIEW.md)
- [v0.4 完成报告](docs/v0.4-COMPLETION-REPORT.md)

### 开发指南
- [后端 README](server/README.md)
- [小程序 README](miniprogram/README.md)
- [下一步指引](NEXT-STEPS.md)

### 故障排查
- [白屏问题修复](docs/troubleshooting/WHITE-SCREEN-FIX.md)

### Sprint 计划
- [Sprint 4.1 计划](docs/sprints/SPRINT-4.1-PLAN.md)
- [Sprint 4.1 总结](docs/sprints/SPRINT-4.1-SUMMARY.md)

## 开发进度

| Sprint | 状态 | 完成日期 | 说明 |
|--------|------|---------|------|
| v0.3.0 | ✅ | 2026-08 | 产品级小程序 UI/UX |
| v0.3.1 | ✅ | 2026-08 | 契约对齐和文档完善 |
| Sprint 4.1 | ✅ | 2026-09-03 | Backend Foundation |
| Sprint 4.2-4.8 | ✅ | 2026-09 | Auth、业务实体、完整借还链路与前端 API 接入 |
| Sprint 4.9 | ✅ | 2026-09-08 | 状态机、事务、超时、登录与真实数据收口 |
| Sprint 4.10 | ✅ 核心完成 | 2026-09-08 | 预约审批、身份核验与禁用用户登录拦截 |
| Sprint 5.0 | ✅ | 2026-09-08 | MQTT DeviceGateway、心跳、事件去重与安全中止 |
| Sprint 5.1 | 📋 | - | 真实柜机联调与故障注入验收 |

## 测试

### 后端测试
```bash
cd server

# 单元测试（不需要数据库）
go test ./... -short

# 所有测试（需要 PostgreSQL）
go test ./...

# 带覆盖率
go test ./... -cover
```

### 前端检查
```bash
npm run check
```

实体扫码、柜机动作和传感器链路仍需在微信开发者工具与真实硬件上验收。

## 提交规范

遵循 Conventional Commits：

```
feat(scope): 新功能
fix(scope): 问题修复
docs(scope): 文档更新
refactor(scope): 代码重构
test(scope): 测试相关
chore(scope): 构建/工具相关
```

示例：
```
feat(backend): complete Sprint 4.1 - Backend Foundation
fix(miniprogram): remove global component registration
docs: add Sprint 4.1 completion report
```

## 贡献者

- 项目负责人：XIAOcx

## 许可证

MIT

## 联系方式

有问题或建议？
- 查看文档：`docs/` 目录
- 查看故障排查：`docs/troubleshooting/`
- 查看下一步：`NEXT-STEPS.md`
