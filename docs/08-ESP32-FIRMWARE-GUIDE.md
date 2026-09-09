# ESP32 嵌入式控制器固件开发与机电控制规范

> 文档版本：V1.0  
> 目标平台：ESP32-WROOM-32 / ESP32-S3  
> 固件框架：ESP-IDF v5.x 或 Arduino-ESP32 (FreeRTOS)  
> 核心任务：实现钥匙柜步进推杆机构驱动、RC522 钥匙 RFID 校验、微动在位检测与 MQTT 协议双向通信。

---

## 1. 硬件引脚映射规划 (Hardware Pinout)

| 外设模块 | 功能管脚 | ESP32 GPIO | 说明 |
| :--- | :--- | :--- | :--- |
| **RC522 (RFID)** | SCK | GPIO 18 | SPI 总线时钟 |
| | MISO | GPIO 19 | SPI 总线主入从出 |
| | MOSI | GPIO 23 | SPI 总线主出从入 |
| | SS / SDA | GPIO 5 | 硬件片选 (CS) |
| | RST | GPIO 22 | 复位脚 |
| **步进电机驱动 (A4988/TMC2209)** | STEP | GPIO 26 | 脉冲步进信号 (PWM/定时器) |
| | DIR | GPIO 27 | 方向控制 (HIGH=推入, LOW=收回) |
| | EN | GPIO 14 | 使能 (LOW=有效) |
| **限位与在位传感器** | 机构原点限位 | GPIO 32 | 下拉输入，推杆归位微动开关 |
| | 槽位 1 在位微动 | GPIO 33 | 上拉输入，检测钥匙是否插入槽位 |
| | 槽位 2 在位微动 | GPIO 25 | 上拉输入，槽位2在位检测 |
| | 安全柜门磁传感器 | GPIO 35 | 上拉输入，检测柜门开闭状态 |
| **通信与状态指示** | 状态指示 LED | GPIO 2 | 板载蓝色 LED (网络/操作指示) |
| | 蜂鸣器 (可选) | GPIO 4 | 操作成功/错卡报警提示 |

---

## 2. FreeRTOS 多任务架构

```
┌───────────────────────────────────────────────────────────┐
│                      ESP32 FreeRTOS                       │
├───────────────┬───────────────┬──────────────┬────────────┤
│   MQTT 任务   │   电机机构任务 │  RFID 读卡任务│ 传感器任务 │
│  (task_mqtt)  │ (task_motor)  │ (task_rfid)  │(task_sens) │
│               │               │              │            │
│ • MQTT 心跳   │ • 归零校准    │ • RC522 寻卡 │ • 钥匙在位 │
│ • 命令解析    │ • 推杆出钥    │ • 读 UID     │ • 门磁状态 │
│ • 状态上报    │ • 超时保护    │ • 防错还校验 │ • 按钮防抖 │
└───────▲───────────────▲──────────────▲──────────────▲─────┘
        │               │              │              │
        └───────────────┴── 事件队列 ───┴──────────────┘
```

---

## 3. 关键业务流程：RFID 防错还闭环时序

为满足立项书**“防错取、防错还”**核心指标，归还操作严禁在未确认 RFID 的情况下直接结束，必须遵循以下状态机时序：

```text
[云端下发归还指令] (cmd/return, 附带 expectedRfid)
      │
      ▼
[ESP32 回复 ACK] (event/ack, 提示设备已就绪)
      │
      ▼
[用户将钥匙插入槽位] (传感器任务检测到在位微动闭合)
      │
      ▼
[RC522 射频天线寻卡读卡]
      │
      ├── 读不到卡 ──> 上报 event/rfid_scanned (isMatch=false, stage=rfid_not_found)
      │                提示“未检测到钥匙标签”，蜂鸣器短鸣报警
      │
      ├── 读到标签 UID ──> 比对 UID 是否与 expectedRfid 一致
      │        │
      │        ├── 不匹配 ──> 上报 event/rfid_scanned (isMatch=false, scannedRfid=UID)
      │        │             提示“钥匙错误”，拒绝闭门完成，服务端阻断
      │        │
      │        └── 匹配 ───> 上报 event/rfid_scanned (isMatch=true, scannedRfid=UID)
      │                        │
      ▼                        ▼
[步进机构复位/电磁门锁闭合] ───> [上报 RETURN_SUCCESS] ───> [服务端落账完成]
```

---

## 4. MQTT 报文规范与时序定义

### 4.1 心跳广播 (`status/heartbeat`)
* **Topic**: `kcab/cab/{deviceId}/status/heartbeat`
* **周期**: 20~30 秒一次
* **Payload**:
  ```json
  {
    "deviceId": "CAB001",
    "status": "ONLINE",
    "timestamp": 1773024800000
  }
  ```

### 4.2 归还时 RFID 扫描上报 (`event/rfid_scanned`)
* **Topic**: `kcab/cab/{deviceId}/event/rfid_scanned`
* **Payload (标签匹配正常)**:
  ```json
  {
    "msgId": "evt_rfid_101",
    "deviceId": "CAB001",
    "timestamp": 1773024850000,
    "data": {
      "operationId": "op_98a72b",
      "stage": "rfid_scanned",
      "isMatch": true,
      "scannedRfid": "E28011606000021A",
      "uid": "E28011606000021A"
    }
  }
  ```
* **Payload (错放其它钥匙)**:
  ```json
  {
    "msgId": "evt_rfid_102",
    "deviceId": "CAB001",
    "timestamp": 1773024850000,
    "data": {
      "operationId": "op_98a72b",
      "stage": "rfid_scanned",
      "isMatch": false,
      "scannedRfid": "E280999999999999",
      "uid": "E280999999999999"
    }
  }
  ```

### 4.3 终态履约完成 (`event/operation_progress`)
* **Topic**: `kcab/cab/{deviceId}/event/operation_progress`
* **Payload**:
  ```json
  {
    "msgId": "evt_ret_succ_103",
    "deviceId": "CAB001",
    "status": "SUCCESS",
    "timestamp": 1773024855000,
    "data": {
      "operationId": "op_98a72b",
      "stage": "RETURN_SUCCESS",
      "slotNo": 1
    }
  }
  ```

---

## 5. 本地联调与自测工具

若当前尚未将硬件电路板焊接连接，开发人员可直接在项目根目录下运行硬件模拟器：

```bash
# 启动正常模拟模式
node scripts/mqtt-cabinet-simulator.mjs --device CAB001

# 模拟错卡归还（注入 RFID 错位故障）
node scripts/mqtt-cabinet-simulator.mjs --device CAB001 --fail-rfid

# 模拟电机卡阻故障
node scripts/mqtt-cabinet-simulator.mjs --device CAB001 --fail-motor
```
通过上述命令可完整模拟物理时序与 MQTT 报文交互，验证服务端与小程序的鲁棒性。
