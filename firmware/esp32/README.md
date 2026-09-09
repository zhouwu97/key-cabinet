# ESP32 钥匙柜嵌入式控制器固件工程

本工程为立项书《基于微信小程序与人脸识别的智能钥匙自助存取柜》中底层机电一体化控制器的**生产级 ESP-IDF C 语言工程源码**。

---

## 1. 硬件引脚映射与外围电路

| 模块 | 功能引脚 | ESP32 GPIO | 硬件电气设计规范 |
| :--- | :--- | :--- | :--- |
| **RC522 (13.56MHz RFID)** | SCK | GPIO 18 | SPI 总线时钟 |
| | MISO | GPIO 19 | SPI 总线主入从出 |
| | MOSI | GPIO 23 | SPI 总线主出从入 |
| | CS / SDA | GPIO 5 | 片选 (低电平选通) |
| | RST | GPIO 22 | 射频芯片硬件复位 |
| **步进电机驱动 (A4988 / TMC2209)** | STEP | GPIO 26 | 步进脉冲输出 |
| | DIR | GPIO 27 | 方向控制 (HIGH=推出, LOW=收回) |
| | EN | GPIO 14 | 使能脚 (LOW=使能, HIGH=休眠防发热) |
| **限位与在位微动** | 原点限位微动 | GPIO 32 | 下拉输入，推杆归位闭合触发 |
| | 槽位 1 在位检测 | GPIO 33 | 内部上拉输入，钥匙插入接地 |
| | 槽位 2 在位检测 | GPIO 25 | 内部上拉输入，钥匙插入接地 |
| | 安全柜门磁传感器 | GPIO 35 | **外部上拉输入** (ESP32 GPI 34-39 无内部上下拉，需在 PCB 外接 10kΩ 上拉电阻至 3.3V) |
| **指示与蜂鸣器** | 状态指示蓝灯 | GPIO 2 | 高电平点亮 |
| | 有源蜂鸣器 | GPIO 4 | 操作成功/错卡报警提示 |

---

## 2. 软件模块架构 (FreeRTOS)

- `main.c`: 固件主入口，初始化 NVS、SPI、GPIO 并创建各个工作任务。
- `rc522.c / rc522.h`: 13.56MHz 高频 ISO14443A 寻卡与防冲突驱动，读取 4/7 字节钥匙 UID。
- `motor_task.c / motor_task.h`: 步进机构原点限位归零校准与槽位精确定位推杆控制。
- `sensor_task.c / sensor_task.h`: 钥匙在位微动消抖、门磁检测，以及周期性向服务端主动上报 `status/inventory` 物理状态快照。
- `rfid_task.c / rfid_task.h`: 归还流程防错还闭环：插入检测 $\to$ RFID 寻卡 $\to$ UID 与 `expectedRfidTag` 比对 $\to$ 上报 `event/rfid_scanned`。
- `protocol.c / protocol.h`: cJSON 协议编解码（对接服务端 `docs/05-MQTT-PROTOCOL.md`）。
- `mqtt_client_task.c / mqtt_client_task.h`: 维持 MQTT 长连接、LWT 遗嘱消息（`status/online`）、心跳广播（`status/heartbeat`）与指令队列调度。

---

## 3. 固件编译与烧录指南

### 环境要求
- ESP-IDF v5.0 或更高版本
- CMake 3.16+
- Python 3.8+

### 编译与烧录命令
```bash
# 1. 切换至固件工程目录
cd firmware/esp32

# 2. 设置目标芯片架构
idf.py set-target esp32

# 3. 编译工程
idf.py build

# 4. 烧录固件并开启串口监控 (将 COM3 替换为实际串口号)
idf.py -p COM3 flash monitor
```
