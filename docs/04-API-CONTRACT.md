# 智能钥匙自助借还系统：后台 RESTful API 契约 V1

> 文档版本：V1.1 (契约对齐版本)  
> 状态：正式冻结 (Frozen)  
> 基础路径：`/api/v1`  
> 数据格式：`application/json;charset=UTF-8`  
> 鉴权方式：`Authorization: Bearer <token>`

---

## 1. 统一响应格式与错误规范

### 1.1 成功响应 (HTTP 200/201)

```json
{
  "code": 0,
  "message": "success",
  "data": {}
}
```

### 1.2 业务失败响应

使用标准 HTTP 状态码配合业务错误体：

**HTTP 状态码规范**:
- `200/201`: 操作成功
- `400`: 参数错误 (缺少必填字段、格式错误)
- `401`: 未登录 / Token 失效
- `403`: 无权限 (角色不符、信用分不足等)
- `404`: 实体不存在 (钥匙、预约、借还记录不存在)
- `409`: 状态冲突 / 预约时间冲突 / 设备忙
- `422`: 当前业务条件不满足 (预约窗口未开始、钥匙不可借等)
- `500`: 服务端内部异常
- `503`: 设备离线 / 下游服务不可用

**业务错误响应体** (HTTP 4xx/5xx 时返回):
```json
{
  "code": 40901,
  "errorCode": "RESERVATION_CONFLICT",
  "message": "该时间段已被预约",
  "data": null,
  "timestamp": "2026-09-02T20:00:00.000Z"
}
```

**常见错误码对照** (详见 `07-ERROR-CODES.md`):
- `TIME_CONFLICT` (409): 预约时间冲突
- `DEVICE_OFFLINE` (503): 设备离线
- `DEVICE_BUSY` (409): 设备正在执行其他操作
- `RESERVATION_NOT_ACTIVE` (422): 预约未生效或已过期
- `KEY_NOT_AVAILABLE` (422): 钥匙当前不可借用

---

## 2. 接口列表与参数规范

### 2.1 认证与授权 (Auth)

#### 1. 微信小程序登录
- **Method**: `POST`
- **Path**: `/auth/wechat-login`
- **Request Body**:
```json
{
  "code": "wx.login() 返回的临时 code"
}
```
- **Response** (HTTP 200):
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "accessToken": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "expiresIn": 7200,
    "user": {
      "id": "U001",
      "name": "张三",
      "studentId": "20230001",
      "role": "USER",
      "department": "信息科学与工程学院",
      "creditScore": 100
    }
  }
}
```

**说明**:
- 后台使用微信 `code` 换取 `openid` 和 `session_key`
- 首次登录自动创建用户记录
- 返回的 `accessToken` 用于后续请求的 `Authorization: Bearer <token>` 头
- `expiresIn` 单位为秒，建议前端在过期前 5 分钟刷新

---

### 2.2 用户与个人中心 (User & Profile)

#### 1. 获取当前登录用户信息
- **Method**: `GET`
- **Path**: `/me`
- **Response**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": "U001",
    "name": "张三",
    "studentId": "20230001",
    "role": "USER",
    "department": "信息科学与工程学院",
    "phone": "13800138000",
    "creditScore": 100
  }
}
```

#### 2. 获取当前用户借用与履约统计看板
- **Method**: `GET`
- **Path**: `/me/stats`
- **Response**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "currentBorrowCount": 1,
    "activeReservationCount": 1,
    "totalBorrowCount": 12,
    "onTimeReturnRate": "100%",
    "hasOverdue": false
  }
}
```

---

### 2.2 钥匙管理 (Key)

#### 1. 查询钥匙列表
- **Method**: `GET`
- **Path**: `/keys`
- **Query Parameters**:
  - `keyword` (optional): 房间号或名称模糊搜索 (如 "103")
  - `deviceId` (optional): 柜体 ID (如 "CAB001")
  - `status` (optional): 业务状态过滤 (`AVAILABLE`, `RESERVED`, `BORROWED`, `MAINTENANCE`, `DISABLED`)
  - `enabled` (optional): 是否在用 (true/false)
- **Response**:
```json
{
  "code": 0,
  "message": "success",
  "data": [
    {
      "id": "KEY103",
      "name": "103 实验室钥匙",
      "roomNo": "103",
      "building": "信息楼",
      "description": "主要用于人工智能与嵌入式课题实验",
      "deviceId": "CAB001",
      "deviceName": "1号钥匙柜 (信息楼一楼大厅)",
      "status": "AVAILABLE",
      "enabled": true,
      "requiresApproval": false
    }
  ]
}
```

#### 2. 查询单个钥匙详情
- **Method**: `GET`
- **Path**: `/keys/{id}`
- **Response**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": "KEY103",
    "name": "103 实验室钥匙",
    "roomNo": "103",
    "building": "信息楼",
    "description": "主要用于人工智能与嵌入式课题实验",
    "deviceId": "CAB001",
    "deviceName": "1号钥匙柜 (信息楼一楼大厅)",
    "status": "AVAILABLE",
    "enabled": true,
    "requiresApproval": false,
    "pickupWindowRule": "预约成功后 2 小时内有效",
    "maxBorrowDurationHours": 12
  }
}
```

---

### 2.3 预约管理 (Reservation)

#### 1. 创建钥匙预约
- **Method**: `POST`
- **Path**: `/reservations`
- **Request Body**:
```json
{
  "keyId": "KEY103",
  "pickupWindowStart": "2026-09-02T20:00:00.000Z",
  "pickupWindowEnd": "2026-09-02T22:00:00.000Z",
  "expectedReturnAt": "2026-09-03T08:00:00.000Z",
  "purpose": "深度学习大作业调试"
}
```
- **Response**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": "RES202609020001",
    "keyId": "KEY103",
    "status": "ACTIVE",
    "pickupWindowStart": "2026-09-02T20:00:00.000Z",
    "pickupWindowEnd": "2026-09-02T22:00:00.000Z",
    "expectedReturnAt": "2026-09-03T08:00:00.000Z",
    "purpose": "深度学习大作业调试"
  }
}
```

**说明**:
- 若钥匙 `requiresApproval=true`，返回的 `status` 为 `PENDING`
- 若钥匙 `requiresApproval=false`，返回的 `status` 为 `ACTIVE` (直接生效)
- V1 阶段暂不支持无预约直接取钥，所有取钥操作必须先创建预约

#### 2. 查询当前用户预约列表
- **Method**: `GET`
- **Path**: `/me/reservations`
- **Query Parameters**:
  - `status` (optional): `PENDING`, `APPROVED`, `ACTIVE`, `USED`, `REJECTED`, `CANCELLED`, `EXPIRED`
- **Response**:
```json
{
  "code": 0,
  "message": "success",
  "data": [
    {
      "id": "RES202609020001",
      "keyId": "KEY103",
      "keyName": "103 实验室钥匙",
      "roomNo": "103",
      "deviceId": "CAB001",
      "status": "ACTIVE",
      "purpose": "深度学习大作业调试",
      "pickupWindowStart": "2026-09-02T20:00:00.000Z",
      "pickupWindowEnd": "2026-09-02T22:00:00.000Z",
      "expectedReturnAt": "2026-09-03T08:00:00.000Z",
      "createdAt": "2026-09-02T18:00:00.000Z"
    }
  ]
}
```

#### 3. 取消预约
- **Method**: `POST`
- **Path**: `/reservations/{id}/cancel`
- **Response**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": "RES202609020001",
    "status": "CANCELLED"
  }
}
```

---

### 2.4 借还台账与记录 (Borrow Records)

#### 1. 查询当前用户借还记录列表
- **Method**: `GET`
- **Path**: `/me/borrow-records`
- **Query Parameters**:
  - `status` (optional): `BORROWING`, `BORROWED`, `RETURNING`, `COMPLETED`, `EXCEPTION`
- **Response**:
```json
{
  "code": 0,
  "message": "success",
  "data": [
    {
      "id": "BOR202609020001",
      "keyId": "KEY104",
      "keyName": "104 会议室钥匙",
      "roomNo": "104",
      "deviceId": "CAB001",
      "status": "BORROWED",
      "borrowedAt": "2026-09-02T16:00:00.000Z",
      "expectedReturnAt": "2026-09-02T21:00:00.000Z",
      "purpose": "项目组周会"
    }
  ]
}
```

**说明**:
- `isOverdue` 字段由前端根据 `status === 'BORROWED' && now > expectedReturnAt` 计算
- 历史记录中若 `overdueAt` 不为空，表示曾经逾期
- 前端可根据 `overdueAt` 显示"逾期归还"或"按时归还"

---

### 2.5 设备操作事务 (Device Operation)

#### 1. 发起取钥操作 (Pickup)
- **Method**: `POST`
- **Path**: `/device-operations/pickup`
- **Request Body**:
```json
{
  "reservationId": "RES202609020001",
  "clientRequestId": "req_1725278400000"
}
```
- **Response**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": "OP20260902120001",
    "action": "PICKUP",
    "deviceId": "CAB001",
    "keyId": "KEY103",
    "status": "CREATED",
    "createdAt": "2026-09-02T20:00:00.000Z"
  }
}
```

**说明**:
- 操作创建后状态为 `CREATED`，随后快速流转为 `AUTHORIZED` → `SENT` → `EXECUTING`
- 前端应立即开始 Polling 查询操作进度 (`GET /device-operations/{id}`)
- V1 阶段使用 HTTP Polling，每 1~2 秒查询一次
- V2 阶段将升级为 WebSocket 推送

#### 2. 发起归还操作 (Return)
- **Method**: `POST`
- **Path**: `/device-operations/return`
- **Request Body**:
```json
{
  "borrowRecordId": "BOR202609020001",
  "deviceId": "CAB001",
  "clientRequestId": "req_1725278400001"
}
```
- **Response**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": "OP20260902120002",
    "action": "RETURN",
    "deviceId": "CAB001",
    "keyId": "KEY104",
    "status": "CREATED",
    "createdAt": "2026-09-02T21:00:00.000Z"
  }
}
```

**说明**:
- 同样需要前端 Polling 查询进度
- 归还操作核心在于 RFID 校验，若校验失败会返回 `status: FAILED` 及错误码

#### 3. 查询操作进度 / 获取当前进行中操作
- **Method**: `GET`
- **Path**: `/device-operations/{id}` 或 `/device-operations/active`
- **Response**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": "OP20260902120001",
    "action": "PICKUP",
    "deviceId": "CAB001",
    "keyId": "KEY103",
    "status": "EXECUTING",
    "createdAt": "2026-09-02T20:00:00.000Z",
    "startedAt": "2026-09-02T20:00:01.000Z",
    "events": [
      { "seq": 1, "timestamp": "2026-09-02T20:00:01.000Z", "type": "RECEIVED" },
      { "seq": 2, "timestamp": "2026-09-02T20:00:02.000Z", "type": "AUTH_CONFIRMED" },
      { "seq": 3, "timestamp": "2026-09-02T20:00:03.000Z", "type": "POSITIONING" },
      { "seq": 4, "timestamp": "2026-09-02T20:00:05.000Z", "type": "POSITIONED" },
      { "seq": 5, "timestamp": "2026-09-02T20:00:06.000Z", "type": "DOOR_OPEN" }
    ]
  }
}
```

**说明**:
- `status` 表示操作生命周期状态: `CREATED` → `AUTHORIZED` → `SENT` → `EXECUTING` → `SUCCESS`/`FAILED`/`TIMEOUT`
- `events` 数组展示设备执行的具体步骤事件流，用于前端实时进度展示
- 前端根据 `events` 最新事件类型显示当前步骤描述
- 操作完成后 `status` 为 `SUCCESS`、`FAILED`、`TIMEOUT` 或 `CANCELLED`，停止 Polling

**V1 实时更新策略**:
- 使用 HTTP Polling: 前端每 1~2 秒调用一次此接口
- 当 `status` 为终态 (`SUCCESS`/`FAILED`/`TIMEOUT`/`CANCELLED`) 时停止轮询
- V2 阶段将升级为 WebSocket 推送，无需轮询

---

### 2.6 设备状态 (Device Status)

#### 1. 获取指定钥匙柜运行状态
- **Method**: `GET`
- **Path**: `/devices/{id}/status`
- **Response**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": "CAB001",
    "name": "1号钥匙柜 (信息楼一楼大厅)",
    "location": "信息楼 1F 门厅东侧",
    "status": "ONLINE",
    "totalSlots": 10,
    "availableSlots": 7,
    "lastHeartbeatAt": "2026-09-02T20:00:00.000Z"
  }
}
```

**说明**:
- `status` 枚举值: `ONLINE`, `OFFLINE`, `BUSY`, `FAULT`, `MAINTENANCE`
- `FAULT`: 当前存在故障
- `MAINTENANCE`: 已人为进入维护状态
    "isOnline": true,
    "isBusy": false,
    "totalSlots": 10,
    "availableSlots": 7,
    "lastHeartbeat": "2026-09-02T20:00:00.000Z"
  }
}
```
