# 智能钥匙自助借还系统：后台 RESTful API 契约 V1

> 文档版本：V1.0  
> 状态：正式冻结 (Frozen)  
> 基础路径：`/api/v1`  
> 数据格式：`application/json;charset=UTF-8`  
> 鉴权方式：`Authorization: Bearer <token>`

---

## 1. 统一响应格式与错误规范

### 1.1 成功响应 (HTTP 200)

```json
{
  "code": 0,
  "message": "success",
  "data": {}
}
```

### 1.2 业务失败响应 (HTTP 200 或 4xx/5xx)

```json
{
  "code": 102,
  "errorCode": "TIME_CONFLICT",
  "message": "所选时间段与已有预约冲突，请重新选择",
  "data": null,
  "timestamp": "2026-09-02T20:00:00.000Z"
}
```

---

## 2. 接口列表与参数规范

### 2.1 用户与个人中心 (User & Profile)

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
  - `status` (optional): 业务状态过滤 (`AVAILABLE`, `BORROWED` 等)
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

#### 2. 查询当前用户预约列表
- **Method**: `GET`
- **Path**: `/me/reservations`
- **Query Parameters**:
  - `status` (optional): `ACTIVE`, `USED`, `CANCELLED`, `EXPIRED`
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
      "expectedReturnAt": "2026-09-03T08:00:00.000Z"
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
  - `status` (optional): `BORROWED`, `RETURNED_NORMAL`, `OVERDUE`, `RETURNED_OVERDUE`
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
      "purpose": "项目组周会",
      "isOverdue": false
    }
  ]
}
```

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
    "status": "INIT",
    "currentStep": 1,
    "totalSteps": 6,
    "stepDescription": "正在建立设备会话，准备取钥..."
  }
}
```

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
    "status": "INIT",
    "currentStep": 1,
    "totalSteps": 6,
    "stepDescription": "正在打开归还口，请稍候..."
  }
}
```

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
    "status": "DOOR_OPENED",
    "currentStep": 4,
    "totalSteps": 6,
    "stepDescription": "柜门已开启，请取走钥匙并在 60 秒内关闭柜门",
    "events": [
      { "seq": 1, "timestamp": "2026-09-02T20:00:01.000Z", "type": "AUTH_SUCCESS" },
      { "seq": 2, "timestamp": "2026-09-02T20:00:02.000Z", "type": "DEVICE_ACCEPTED" },
      { "seq": 3, "timestamp": "2026-09-02T20:00:04.000Z", "type": "POSITIONED" },
      { "seq": 4, "timestamp": "2026-09-02T20:00:06.000Z", "type": "DOOR_OPENED" }
    ]
  }
}
```

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
    "isOnline": true,
    "isBusy": false,
    "totalSlots": 10,
    "availableSlots": 7,
    "lastHeartbeat": "2026-09-02T20:00:00.000Z"
  }
}
```
