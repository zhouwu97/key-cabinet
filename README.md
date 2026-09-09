# Key Cabinet - 智能钥匙自助借还系统

## 项目概述

智能钥匙自助借还系统（Key Cabinet）是一个面向校园/企业的钥匙管理系统，支持微信小程序预约、自助取还钥匙、设备联动控制。

**当前阶段**: 真实系统收口（Real-flow Hardening）

## 最新进展

### ✅ Sprint 6 实体硬件与边缘真实性收口（2026-09-09）

本轮针对真实硬件环境完成了关键收口，杜绝“代码写直方图却声称深度特征”、“未连网却声称固件完成”、“未检测微动却上报成功”等脱节问题：
1. **RK3588 真实生物人脸识别**：接入深度模型接口与 5 点人脸对齐，提取标准 512 维 L2 归一化特征向量，彻底剔除画面中央假人脸降级；生物特征模板结合设备密钥采用 **AES-256-GCM** 算法加密持久化存储 (`.enc`)。
2. **静默活体防攻击 (PAD)**：分离清晰度阈值 (`laplacian_threshold: 85.0`) 与综合通过置信度 (`accept_score: 0.85`)，引入 FFT 频域中高频段峰均功率比 (PAPR) 摩尔条纹尖峰检验与 HSV 反光过曝分析，多维加权连续打分。
3. **ESP32 固件真正联网与校时**：实现 Wi-Fi STA 联网流程与网络就绪前置阻塞；启动 SNTP 授时同步 RTC 时间（未授时返回 0 由服务端接收时间兜底）；提供 4G Modem (esp_modem/PPP) 串口抽象。
4. **全物理传感器闭环与紧急中止**：出钥必须检测到槽位微动从 `PRESENT -> ABSENT`（钥匙被拔出）且安全柜门闭合；归还必须检测微动闭合 + RFID 匹配 + 柜门闭合；支持下行 `CMD_TYPE_ABORT` 实时切断步进电机脉冲并硬件失能。
5. **服务端 Inventory 消费与对账**：Go MQTT 网关订阅 `status/inventory`，解析物理槽位快照并与数据库槽位状态进行审计对账。
6. **安全收口**：设备通信密钥全量采用 32-byte CSPRNG 随机十六进制字符串，移除可预测 fallback；`FaceSessionMiddleware` 强校验现场认证柜机 ID 与 Token 绑定一致性；小程序接入 `wx.requestSubscribeMessage` 授权。
7. **RK3588 触屏 UI 业务串联**：`TouchscreenKioskUI` 接入主事件循环，支持刷脸认证通过后在虚拟触控键盘上输入任意房间号现场申请出钥。

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
- **数据库**: PostgreSQL 14+
- **认证**: JWT (HS256) + 设备 HMAC-SHA256 防重放签名
- **物联网**: MQTT v3.1.1 (TLS/QoS 1)
- **迁移工具**: golang-migrate

### 边缘计算与固件
- **ESP32**: ESP-IDF v5.0+ FreeRTOS C 固件 (RC522 RFID, A4988 步进推杆, 槽位微动, 门磁, Wi-Fi STA, SNTP)
- **RK3588**: Python 3.8+ 边缘终端 (512-d Biometric Embedding, AES-256-GCM 模板加密, 静默活体 PAD, Tkinter 触屏 Kiosk UI)

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

### v0.5 - 实体机电与网络固件（已完成）
- ✅ MQTT 设备通信网关 (QoS 1 保证、状态对账)
- ✅ ESP32 生产级固件 (FreeRTOS)
- ✅ Wi-Fi STA 联网与 SNTP 毫秒授时
- ✅ 4G Modem (esp_modem/PPP) 架构抽象
- ✅ RC522 13.56MHz 4 字节标准 UID 防错还
- ✅ 步进电机原点限位与 Abort 紧急切断
- ✅ 槽位微动与柜门磁全物理传感器闭环
- ✅ `status/inventory` 槽位物理状态上报与服务端对账

### v0.6 - 边缘人脸识别与触屏终端（已完成）
- ✅ RK3588 512 维生物特征特征提取
- ✅ AES-256-GCM 模板强加密持久化存储 (`.enc`)
- ✅ 静默活体防攻击 (PAD: Laplacian 散焦 + FFT 频域摩尔尖峰 PAPR + HSV 过曝)
- ✅ 柜机设备 HMAC-SHA256 防篡改防重放签名
- ✅ FaceSessionToken 现场授权与跨柜绑定强校验
- ✅ Tkinter 触控屏幕数字键盘与全流程业务串联

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
