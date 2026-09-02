# 智能钥匙自助借还系统：柜机 MQTT 通信协议规范 V1

> 文档版本：V1.0  
> 状态：正式冻结 (Frozen)  
> 适用通信端：业务后台 ↔ 钥匙柜柜控主机 (RK3588 / K230 / ESP32)  
> 编码：`UTF-8 JSON`  
> 默认 QoS：`QoS 1` (保证至少送达一次，配合 msgId 去重)

---

## 1. Topic 命名空间与主题树设计

```text
kcab/
└── cab/
    └── {deviceId}/                      # 单机命名空间 (如 kcab/cab/CAB001/...)
        ├── cmd/                         # 【下行指令】业务后台 -> 柜控硬件
        │   ├── pickup                   # 发起取钥定位与开门指令
        │   ├── return                   # 发起归还开启与 RFID 校验指令
        │   ├── query_slots              # 主动查询所有物理槽位在位状态
        │   ├── open_door_admin          # 管理员应急/维护开门指令
        │   └── reboot                   # 远程重启设备指令
        ├── event/                       # 【上行事件】柜控硬件 -> 业务后台
        │   ├── operation_progress       # 操作过程中关键动作状态上报
        │   ├── slot_change              # 槽位传感器状态变化上报 (插拔检测)
        │   ├── rfid_scanned             # RFID 读头扫描到芯片事件
        │   └── alarm                    # 硬件故障/长时间未关门告警
        └── status/                      # 【设备状态】柜控硬件 -> 业务后台
            ├── heartbeat                # 周期心跳保活包 (30s / 次)
            └── online                   # LWT 遗嘱与上线通知 (Retained)
```

---

## 2. 统一报文结构体

### 2.1 下行指令报文格式 (Command)

```json
{
  "msgId": "cmd_20260902200001_abc123",
  "timestamp": 1788350400000,
  "deviceId": "CAB001",
  "action": "PICKUP",
  "data": {}
}
```

### 2.2 上行事件与响应报文格式 (Event / Ack)

```json
{
  "msgId": "evt_20260902200002_xyz789",
  "replyMsgId": "cmd_20260902200001_abc123",
  "timestamp": 1788350402000,
  "deviceId": "CAB001",
  "status": "SUCCESS",
  "errorCode": null,
  "data": {}
}
```

---

## 3. 核心指令与事件 Payload 详解

### 3.1 取钥指令与时序事件 (Pickup)

#### 1. 后台下发取钥指令
- **Topic**: `kcab/cab/CAB001/cmd/pickup`
- **Payload**:
```json
{
  "msgId": "cmd_pickup_001",
  "timestamp": 1788350400000,
  "deviceId": "CAB001",
  "action": "PICKUP",
  "data": {
    "operationId": "OP20260902120001",
    "slotNo": 3,
    "keyId": "KEY103",
    "timeoutSeconds": 60
  }
}
```

#### 2. 硬件执行阶段进度上报
- **Topic**: `kcab/cab/CAB001/event/operation_progress`
- **Payload**:
```json
{
  "msgId": "evt_op_001_pos",
  "timestamp": 1788350403000,
  "deviceId": "CAB001",
  "data": {
    "operationId": "OP20260902120001",
    "stage": "POSITIONED",
    "slotNo": 3,
    "description": "电机已转动至 3 号槽位"
  }
}
```

#### 3. 柜门开启与用户取钥上报
- **Topic**: `kcab/cab/CAB001/event/operation_progress`
- **Payload**:
```json
{
  "msgId": "evt_op_001_door",
  "timestamp": 1788350405000,
  "deviceId": "CAB001",
  "data": {
    "operationId": "OP20260902120001",
    "stage": "DOOR_OPEN",
    "slotNo": 3,
    "description": "柜门已开启，等待用户取钥"
  }
}
```

#### 4. 钥匙已取出、柜门已关、取钥完成上报
- **Topic**: `kcab/cab/CAB001/event/operation_progress`
- **Payload**:
```json
{
  "msgId": "evt_op_001_finish",
  "timestamp": 1788350412000,
  "deviceId": "CAB001",
  "data": {
    "operationId": "OP20260902120001",
    "stage": "SUCCESS",
    "slotNo": 3,
    "finalPresence": "ABSENT",
    "description": "钥匙已取走，柜门已关闭，取钥成功"
  }
}
```

---

### 3.2 归还指令与 RFID 校验事件 (Return)

#### 1. 后台下发归还开启指令
- **Topic**: `kcab/cab/CAB001/cmd/return`
- **Payload**:
```json
{
  "msgId": "cmd_return_002",
  "timestamp": 1788350500000,
  "deviceId": "CAB001",
  "action": "RETURN",
  "data": {
    "operationId": "OP20260902120002",
    "targetSlotNo": 3,
    "expectedRfidTag": "E200001A9903",
    "expectedKeyId": "KEY103",
    "timeoutSeconds": 60
  }
}
```

#### 2. RFID 扫描结果上报
- **Topic**: `kcab/cab/CAB001/event/rfid_scanned`
- **Payload (校验通过)**:
```json
{
  "msgId": "evt_rfid_001",
  "timestamp": 1788350510000,
  "deviceId": "CAB001",
  "data": {
    "operationId": "OP20260902120002",
    "targetSlotNo": 3,
    "scannedRfidTag": "E200001A9903",
    "isMatch": true
  }
}
```

- **Payload (错误钥匙拒绝)**:
```json
{
  "msgId": "evt_rfid_002",
  "timestamp": 1788350510000,
  "deviceId": "CAB001",
  "data": {
    "operationId": "OP20260902120002",
    "targetSlotNo": 3,
    "scannedRfidTag": "E200001A9999",
    "isMatch": false,
    "errorCode": "E208",
    "errorMessage": "检测到非本槽位钥匙，请重新放入正确钥匙"
  }
}
```

---

### 3.3 设备心跳与上线保活 (Heartbeat & LWT)

#### 1. 上线/遗嘱消息 (Last Will and Testament)
- **Topic**: `kcab/cab/CAB001/status/online`
- **Retain**: `true`
- **上线 Payload**:
```json
{
  "deviceId": "CAB001",
  "status": "ONLINE",
  "ip": "192.168.1.105",
  "firmwareVersion": "v1.2.0-esp32",
  "timestamp": 1788350000000
}
```
- **离线 LWT Payload**:
```json
{
  "deviceId": "CAB001",
  "status": "OFFLINE",
  "timestamp": 1788350000000
}
```

#### 2. 周期心跳 (Heartbeat)
- **Topic**: `kcab/cab/CAB001/status/heartbeat`
- **周期**: 每 30 秒发送一次
- **Payload**:
```json
{
  "deviceId": "CAB001",
  "timestamp": 1788350430000,
  "slotPresenceMask": "1101111111",
  "isBusy": false,
  "tempCpu": 42.5,
  "freeHeap": 128400
}
```
