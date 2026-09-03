# 契约对齐报告 (Contract Alignment Report)

> **版本**: v0.3.1  
> **日期**: 2026-09-03  
> **状态**: 执行中

---

## 1. 问题背景

`v0.3.0-product-ready` 已完成小程序产品形态闭环，并通过 24 项自动化测试。但在准备进入 `v0.4.0-backend-integrated` 之前，发现**冻结的后台协议文档与实际前端代码存在多处实质性不一致**。

如果现在直接按文档开发后台，将导致：
- 前后端状态机不匹配
- API 返回的枚举值前端无法识别
- 数据库设计与领域模型脱节
- 后期需要大量返工

因此插入 `v0.3.1-contract-alignment` 阶段，统一所有跨端契约。

---

## 2. 发现的主要不一致

### 2.1 BorrowRecord 状态机 ⚠️ P0

**当前代码** (miniprogram/models/borrow-record.ts):
```typescript
enum BorrowRecordStatus {
  BORROWING = 'BORROWING',    // 取钥出柜中
  BORROWED = 'BORROWED',      // 已借出
  RETURNING = 'RETURNING',    // 归还入柜中
  COMPLETED = 'COMPLETED',    // 已完成归还
  EXCEPTION = 'EXCEPTION',    // 异常状态
}
```

**设计哲学**:
- `OVERDUE` 从主状态机分离
- 通过 `status === BORROWED && now > expectedReturnAt` 计算判断
- 派生字段: `overdueAt?`, `returnedAt?`

**文档定义** (02-DATA-MODEL.md, 03-STATE-MACHINE.md):
```typescript
status: 'BORROWED' | 'RETURNED_NORMAL' | 'OVERDUE' | 'RETURNED_OVERDUE' | 'LOST'
```

**影响**:
- API 查询参数 `?status=OVERDUE` 在当前模型中无意义
- `RETURNED_NORMAL` vs `RETURNED_OVERDUE` 分裂了归还状态
- 前端无法直接消费 API 返回的这些状态值

**决议**: ✅ 以代码为准，文档改为:
```typescript
BORROWING | BORROWED | RETURNING | COMPLETED | EXCEPTION
```

附加派生逻辑:
```typescript
isOverdue = status === BORROWED && now > expectedReturnAt
wasOverdue = status === COMPLETED && overdueAt != null
```

---

### 2.2 DeviceOperation 状态机 ⚠️ P0

**当前代码** (miniprogram/models/device-operation.ts):
```typescript
enum DeviceOperationStatus {
  CREATED = 'CREATED',
  AUTHORIZED = 'AUTHORIZED',
  SENT = 'SENT',
  EXECUTING = 'EXECUTING',
  SUCCESS = 'SUCCESS',
  FAILED = 'FAILED',
  TIMEOUT = 'TIMEOUT',
  CANCELLED = 'CANCELLED',
}
```

**文档定义** (02-DATA-MODEL.md, 03-STATE-MACHINE.md, 04-API-CONTRACT.md):
```typescript
status: 'INIT' | 'PREPARING' | 'DOOR_OPENED' | 'VERIFYING' | 'COMPLETED' | 'FAILED'
```

**问题根源**: 文档混淆了两个概念:

1. **Operation 生命周期** (应该是状态):
   - CREATED → AUTHORIZED → SENT → EXECUTING → SUCCESS/FAILED/TIMEOUT/CANCELLED

2. **当前设备步骤/事件** (应该是事件流):
   - POSITIONING → POSITIONED → DOOR_OPEN → KEY_REMOVED → RFID_CONFIRMED → HOMING

**决议**: ✅ 以代码为准:
- `DeviceOperationStatus`: 生命周期状态
- `DeviceEvent` (已存在): 设备事件流
- 文档中 `DOOR_OPENED` 应该是事件，不是状态

---

### 2.3 Reservation 状态机 ⚠️ P0

**当前代码** (miniprogram/models/reservation.ts):
```typescript
enum ReservationStatus {
  PENDING = 'PENDING',      // 待审批
  APPROVED = 'APPROVED',    // 已审批通过
  ACTIVE = 'ACTIVE',        // 处于可取钥窗口
  USED = 'USED',           // 已取钥完成
  REJECTED = 'REJECTED',    // 审批拒绝
  CANCELLED = 'CANCELLED',  // 已取消
  EXPIRED = 'EXPIRED',      // 超期未取
}
```

**文档定义** (02-DATA-MODEL.md):
```typescript
status: 'PENDING_APPROVAL' | 'ACTIVE' | 'USED' | 'CANCELLED' | 'EXPIRED' | 'REJECTED'
```

**问题**:
- 文档缺少 `APPROVED` 状态
- `PENDING` vs `PENDING_APPROVAL` 命名不一致
- 缺少 `APPROVED` 导致无法表达"审批通过但窗口未开始"

**语义**:
```text
PENDING: 等待管理员审批
APPROVED: 审批通过但取钥窗口尚未开始
ACTIVE: 当前已经进入可取钥窗口
USED: 完成取钥
```

**决议**: ✅ 以代码为准，保留全部 7 个状态

---

### 2.4 KeyPresence 状态枚举 ⚠️ P1

**当前代码** (miniprogram/models/key-presence.ts):
```typescript
enum KeyPresenceState {
  PRESENT = 'PRESENT',  // 钥匙在柜
  ABSENT = 'ABSENT',    // 钥匙已离柜
  UNKNOWN = 'UNKNOWN',  // 无法确认
  FAULT = 'FAULT',      // 传感器异常
}
```

**文档定义** (02-DATA-MODEL.md):
```typescript
presence: 'PRESENT' | 'ABSENT'
```

**问题**: 文档缺少 `UNKNOWN` 和 `FAULT`

**决议**: ✅ 以代码为准。真实硬件必须能表达"不知道"和"传感器坏了"

---

### 2.5 DeviceStatus 状态枚举 ⚠️ P1

**当前代码** (miniprogram/models/device.ts):
```typescript
enum DeviceStatus {
  ONLINE = 'ONLINE',
  OFFLINE = 'OFFLINE',
  BUSY = 'BUSY',
  FAULT = 'FAULT',
  MAINTENANCE = 'MAINTENANCE',
}
```

**文档定义** (02-DATA-MODEL.md):
```typescript
status: 'ONLINE' | 'OFFLINE' | 'BUSY' | 'ERROR'
```

**问题**:
- `ERROR` vs `FAULT` 语义模糊
- 缺少 `MAINTENANCE` 状态

**决议**: ✅ 以代码为准:
- `FAULT`: 当前存在故障
- `MAINTENANCE`: 已人为进入维护状态

---

### 2.6 Key.status 业务状态 ⚠️ P1

**文档定义** (02-DATA-MODEL.md):
```typescript
status: 'AVAILABLE' | 'RESERVED' | 'BORROWED' | 'OVERDUE' | 'MAINTENANCE' | 'DISABLED'
```

**问题**: `OVERDUE` 不应该是 Key 的状态，应该从 BorrowRecord 派生

**决议**: ✅ 删除 `OVERDUE`，保留:
```typescript
status: 'AVAILABLE' | 'RESERVED' | 'BORROWED' | 'MAINTENANCE' | 'DISABLED'
```

---

### 2.7 V1 是否允许"无预约现场直借" ⚠️ P0 架构决策

**文档定义** (03-STATE-MACHINE.md):
```text
AVAILABLE → BORROWED (现场即时取钥)
```

并明确描述:
> 现场扫码/人脸验证成功即可直接取钥。

**当前测试与代码**:
- 所有取钥操作都要求有 `reservationId`
- 测试用例也验证必须有预约

**决议**: ✅ **V1 阶段暂不支持无预约直借**

理由:
1. 权限逻辑简化
2. 预约时间窗口机制已验证
3. 未来 v1.1 或 V2 可扩展

文档改为:
```text
V1: Reservation ACTIVE + Identity verified → 允许 PICKUP
V2 (规划): 支持无预约现场直借
```

---

### 2.8 Auth API 缺失 ⚠️ P0

**现状**: `04-API-CONTRACT.md` 没有定义登录接口

**后台开发必需**:
```http
POST /api/v1/auth/wechat-login
```

Request:
```json
{
  "code": "wx.login() 返回的临时 code"
}
```

Response:
```json
{
  "code": 0,
  "data": {
    "accessToken": "...",
    "expiresIn": 7200,
    "user": {
      "id": "U001",
      "name": "张三",
      "role": "USER"
    }
  }
}
```

**决议**: ✅ 补充完整 Auth 章节

---

### 2.9 HTTP 状态码规范 ⚠️ P1

**现状**: 文档写"HTTP 200 或 4xx/5xx"

**问题**: 后台开发者不知道该返回哪个

**决议**: ✅ 明确规定:
```text
200/201   成功
400       参数错误
401       未登录/Token 失效
403       无权限
404       实体不存在
409       状态冲突/预约冲突/设备忙
422       当前业务条件不满足
500       服务端异常
503       设备/下游服务不可用
```

业务错误体:
```json
{
  "code": 40901,
  "errorCode": "RESERVATION_CONFLICT",
  "message": "该时间段已被预约"
}
```

---

### 2.10 实时更新策略 ⚠️ P1

**现状**: 文档写"SSE / Polling"，未明确

**决议**: ✅ v0.4 使用 HTTP Polling:
```http
GET /api/v1/device-operations/{id}
```

每 1~2 秒查询一次

优点:
- 简单可靠
- 容易调试
- 与 Mock 行为对齐

v0.5 再升级 WebSocket

---

## 3. 对齐后的统一规范

### 3.1 状态枚举对照表

| 实体 | 统一枚举值 | 来源 |
|:---|:---|:---|
| `BorrowRecordStatus` | `BORROWING`, `BORROWED`, `RETURNING`, `COMPLETED`, `EXCEPTION` | 代码 |
| `DeviceOperationStatus` | `CREATED`, `AUTHORIZED`, `SENT`, `EXECUTING`, `SUCCESS`, `FAILED`, `TIMEOUT`, `CANCELLED` | 代码 |
| `ReservationStatus` | `PENDING`, `APPROVED`, `ACTIVE`, `USED`, `REJECTED`, `CANCELLED`, `EXPIRED` | 代码 |
| `KeyPresenceState` | `PRESENT`, `ABSENT`, `UNKNOWN`, `FAULT` | 代码 |
| `DeviceStatus` | `ONLINE`, `OFFLINE`, `BUSY`, `FAULT`, `MAINTENANCE` | 代码 |
| `Key.status` | `AVAILABLE`, `RESERVED`, `BORROWED`, `MAINTENANCE`, `DISABLED` | 修正 |

### 3.2 派生逻辑约定

**逾期判断**:
```typescript
// BorrowRecord
isOverdue = status === 'BORROWED' && now > expectedReturnAt
wasOverdue = status === 'COMPLETED' && overdueAt != null

// 前端显示
归还结果 = wasOverdue ? "逾期归还" : "按时归还"
```

**预约窗口**:
```typescript
// Reservation
isInWindow = status === 'ACTIVE' && pickupWindowStart <= now <= pickupWindowEnd
canPickup = isInWindow && deviceStatus === 'ONLINE'
```

---

## 4. 执行清单

- [ ] 更新 `02-DATA-MODEL.md` 所有枚举定义
- [ ] 更新 `03-STATE-MACHINE.md` 状态转移图和规则表
- [ ] 更新 `04-API-CONTRACT.md` 所有接口响应示例
- [ ] 补充 `04-API-CONTRACT.md` Auth 章节
- [ ] 明确 HTTP 状态码规范
- [ ] 明确 v0.4 使用 Polling 策略
- [ ] 创建 `api/dto/` 设计指南文档
- [ ] 更新 README.md 里程碑描述
- [ ] 提交并打 tag `v0.3.1-contract-alignment`

---

## 5. 下一阶段 v0.4.0 开发顺序

| Sprint | 内容 | 验收 |
|:---|:---|:---|
| 4.1 | Backend skeleton + DB migration | 服务启动、数据库初始化 |
| 4.2 | Auth + User | 微信登录、Token、`/me` |
| 4.3 | Key + Device + Slot | 小程序钥匙页读取真实 API |
| 4.4 | Reservation | 创建/冲突/取消/过期 |
| 4.5 | BorrowRecord | 当前借用、历史、逾期计算 |
| 4.6 | DeviceOperation | 后台 Mock Device Gateway + Polling |

---

**最后更新**: 2026-09-03
