# Key Cabinet - 智能钥匙自助借还系统

## 项目概述

智能钥匙自助借还系统（Key Cabinet）是一个面向校园/企业的钥匙管理系统，支持微信小程序预约、自助取还钥匙、设备联动控制。

**当前阶段**: 真实系统收口（Real-flow Hardening）

## 最新进展

### ✅ Sprint 4.9 / 4.10 / 5.0 核心收口（2026-09-08）

系统已打通微信登录、钥匙/设备/槽位、预约、取钥、借用、归还和操作事件主链。本轮完成了设备操作终态保护、安全取消、借还准备事务、操作超时收敛、登录 single-flight、结构化 API 错误、真实 API 去假数据，以及预约审批和学校身份核验闭环。

首版 `MQTTDeviceGateway` 已支持连接与重连、心跳/LWT、取还命令、ACK、成功/失败事件、QoS 1 去重、安全中止和超时离线判定。当前最大缺口转为实体柜机联调与断网、重复消息、晚到消息、柜门未关、RFID 不一致等故障注入验收。

## 技术栈

### 前端（微信小程序）
- TypeScript
- WXML/WXSS
- Domain-Driven Design
- Mock / 真实 HTTP API 双模式

### 后端（Go）
- **语言**: Go 1.26.2
- **Web 框架**: Gin
- **ORM**: GORM
- **数据库**: PostgreSQL 14+
- **认证**: JWT (HS256)
- **迁移工具**: golang-migrate

### 硬件（v0.5+）
- ESP32/ESP8266 设备控制器
- MQTT 通信协议
- RK3588 人脸识别（v0.6+）

## 项目结构

```
key-cabinet/
├── miniprogram/          # 微信小程序前端
│   ├── components/       # 自定义组件
│   ├── models/           # 领域模型
│   ├── services/         # 业务服务层
│   ├── pages/            # 页面
│   └── app.ts            # 小程序入口
│
├── server/               # Go 后端服务
│   ├── cmd/              # 命令行入口
│   │   ├── api/          # API 服务器
│   │   └── migrate/      # 数据库迁移
│   ├── internal/         # 内部包
│   │   ├── config/       # 配置管理
│   │   ├── domain/       # 领域模型
│   │   ├── service/      # 应用服务
│   │   ├── repository/   # 数据访问
│   │   ├── transport/    # HTTP 传输层
│   │   ├── infrastructure/ # 基础设施
│   │   └── platform/     # 平台层
│   ├── migrations/       # SQL 迁移文件
│   └── tests/            # 测试
│
└── docs/                 # 文档
    ├── sprints/          # Sprint 计划和总结
    ├── troubleshooting/  # 故障排查
    └── *.md              # 各类文档
```

## 快速开始

### 前端（微信小程序）

1. 安装微信开发者工具
2. 导入项目（选择 `miniprogram/` 目录）
3. 编译运行

详见：[小程序 README](miniprogram/README.md)

### 后端（Go 服务）

```bash
# 1. 安装依赖
cd server
go mod download

# 2. 创建数据库
createdb keycabinet

# 3. 配置
cp internal/config/config.example.yaml internal/config/config.yaml
# 编辑 config.yaml，配置数据库连接

# 4. 运行迁移
go run cmd/migrate/main.go -command up

# 5. 启动服务器
go run cmd/api/main.go

# 6. 验证
curl http://localhost:8080/health
```

详见：[后端 README](server/README.md)

## 核心功能

### v0.3.1 - 产品级小程序（已完成）
- ✅ 微信小程序完整 UI/UX
- ✅ 钥匙浏览和搜索
- ✅ 预约创建和管理
- ✅ 取钥/还钥操作流程
- ✅ 借用历史记录
- ✅ 用户个人中心
- ✅ 完整的 Mock Service

### v0.4 - 软件闭环（已完成）
- ✅ 微信登录、JWT、用户资料与身份核验
- ✅ Key / Slot / Device 查询
- ✅ 预约冲突控制、审批、拒绝与自动过期
- ✅ BorrowRecord 和 DeviceOperation 事务落账
- ✅ 预约 → 取钥 → 借用 → 归还完整主链
- ✅ 小程序 Mock / API 模式切换与真实错误语义
- ✅ 取消、迟到事件和操作超时安全收敛

### v0.5 - 真实设备（进行中）
- ✅ MQTT 设备通信网关
- 📋 ESP32/ESP8266 控制器
- 📋 真实电机和 RFID
- ✅ 心跳、LWT 与超时离线监控

### v0.6 - 人脸识别（计划中）
- 📋 RK3588 人脸识别
- 📋 现场快速取钥
- 📋 多种认证方式

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
