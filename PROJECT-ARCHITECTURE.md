# 智能钥匙自助借还系统：项目架构与目标

> 文档版本：V1.0  
> 项目名称：钥匙自助借还系统设计  
> 适用仓库：`zhouwu97/key-cabinet` 及后续设备端/后台端仓库  
> 需求基线：2026 年大学生创新训练计划申报书  
> 参考原型：2022 年《钥匙借还系统设计》  
> 当前阶段：系统架构冻结与小程序 V1 开发前准备

---

## 1. 项目背景与定位

本项目面向高校实验室、办公室、企业等存在大量钥匙借还需求的场景，目标是实现钥匙借还过程的自动化、无人化、可追溯和可监管。

2022 年旧版钥匙借还系统主要采用单片机、矩阵键盘、LCD1602、步进电机、电磁锁、红外定位和 EEPROM，完成“输入身份信息和房间号 → 转盘定位 → 开锁取钥 → 归位 → 记录”的基本闭环。

2026 年本项目在上述机械存取思想基础上进行升级，引入：

- 人脸识别与活体检测
- 触摸屏交互
- ESP32 网络通信与设备控制
- RFID 钥匙身份校验
- 后台数据管理
- 微信小程序
- 钥匙状态查询
- 手机预约
- 借还记录追溯
- 超时未归还提醒
- Wi-Fi / 4G 网络通信

因此，本项目不再是单纯的“单片机钥匙柜”，而是一套由 **移动端、业务后台、边缘计算、人脸识别、设备控制、RFID 与机械执行机构** 共同组成的智能物联网系统。

---

## 2. 总体建设目标

### 2.1 总目标

完成一套能够稳定运行的智能钥匙自助借还系统，实现：

1. 合法用户身份核验；
2. 钥匙状态查询和预约；
3. 根据房间号准确匹配钥匙；
4. 自动完成钥匙转盘定位与取钥；
5. RFID 校验归还钥匙身份；
6. 自动生成借还记录；
7. 实时同步钥匙状态；
8. 对超时未归还行为进行提醒；
9. 管理员能够查询和追溯完整操作记录；
10. 小程序与设备端相互解耦，可独立开发和后续扩展。

### 2.2 小程序目标

微信小程序定位为：

> **智能钥匙自助借还系统的移动端业务与管理入口。**

小程序负责：

- 用户登录与身份信息展示
- 钥匙列表与搜索
- 钥匙实时状态查询
- 钥匙预约
- 当前预约查看
- 当前借用查看
- 借还历史记录
- 超时状态与提醒展示
- 设备在线状态展示
- 管理员入口（后续阶段）
- 与业务后台的数据同步
- 调试阶段通过 DeviceService 对接 Mock / MQTT

小程序原则上不负责：

- 步进电机脉冲计算
- GPIO 控制
- RFID 底层驱动
- 人脸识别算法
- 活体检测算法
- RK3588/K230 Linux 驱动
- ESP32 外设控制
- 具体钥匙转盘角度计算

---

## 3. 系统总体架构

```text
┌──────────────────────────────────────────────────────────────┐
│                        微信小程序                            │
│                                                              │
│  登录 / 钥匙查询 / 预约 / 当前借用 / 记录 / 提醒 / 管理入口   │
└──────────────────────────────┬───────────────────────────────┘
                               │
                        HTTPS / WebSocket
                               │
                               ▼
┌──────────────────────────────────────────────────────────────┐
│                        业务后台                              │
│                                                              │
│  Auth 用户认证                                               │
│  User 用户管理                                               │
│  Key 钥匙管理                                                │
│  Reservation 预约                                            │
│  BorrowRecord 借还记录                                       │
│  Device 设备状态                                             │
│  Reminder 超时提醒                                           │
│  DeviceEvent 设备事件                                        │
│                                                              │
│  Database                                                    │
└──────────────────────────────┬───────────────────────────────┘
                               │
                          MQTT Broker
                               │
                         Wi-Fi / 4G
                               │
                               ▼
┌──────────────────────────────────────────────────────────────┐
│                           ESP32                              │
│                                                              │
│  MQTT 通信                                                   │
│  步进电机控制                                                │
│  电磁锁控制                                                  │
│  RFID 通信                                                   │
│  设备状态上报                                                │
│  异常状态上报                                                │
└───────────────┬──────────────────────────────┬───────────────┘
                │                              │
                ▼                              ▼
          步进电机 / 转盘                  RFID / 电磁锁
                ▲
                │ UART / 串口
                │
┌───────────────┴──────────────────────────────────────────────┐
│                    RK3588 / K230                            │
│                                                              │
│  Linux / 边缘计算                                            │
│  摄像头采集                                                  │
│  人脸检测                                                    │
│  人脸识别                                                    │
│  活体检测                                                    │
│  本地特征库                                                  │
│  触摸屏 UI                                                   │
│  房间号输入 / 选择                                           │
│  本地身份核验                                                │
└──────────────────────────────────────────────────────────────┘
```

---

## 4. 各模块职责边界

### 4.1 微信小程序

#### 核心职责

- 展示用户相关业务状态
- 查询钥匙
- 提交预约
- 查看预约
- 查看借用状态
- 查看借还记录
- 展示超时提醒
- 展示设备在线/异常状态
- 后续提供管理员功能

#### 不允许依赖

页面层不得直接依赖：

- 电机角度
- 转盘槽位
- RFID UID 到底层驱动的映射
- ESP32 GPIO
- RK3588/K230 算法内部实现

页面只依赖业务数据：

```text
User
Key
Reservation
BorrowRecord
Device
Reminder
```

---

### 4.2 业务后台

后台是整个系统的 **业务事实源（Single Source of Truth）**。

后台负责：

- 用户身份映射
- 钥匙信息维护
- 钥匙当前业务状态
- 预约管理
- 借还记录生成
- 设备状态记录
- 超时判断
- 消息提醒
- 接收 MQTT 设备事件
- 向 ESP32 下发设备业务命令
- 保存完整可追溯日志

正式系统中，钥匙是否“已借出”“已归还”不能仅由小程序本地状态决定。

---

### 4.3 RK3588 / K230

定位：

> **人脸识别 + 活体检测 + 柜体本地交互终端。**

主要负责：

- Linux 环境运行
- 摄像头图像采集
- 人脸识别
- 活体检测
- 本地人脸特征管理
- 触摸屏 UI
- 房间号输入与选择
- 将经过身份核验的业务请求发送给设备控制端

推荐原则：

- 不承担步进电机实时脉冲控制；
- 不承担全部云端业务数据；
- 不把人脸识别算法暴露给小程序。

---

### 4.4 ESP32

定位：

> **联网设备控制器。**

主要负责：

- MQTT / 网络通信
- 与 RK3588/K230 串口通信
- 步进电机控制
- 门锁控制
- RFID 模块通信
- 设备心跳
- 设备错误上报
- 业务命令幂等处理
- 硬件执行状态反馈

ESP32 不负责完整用户业务和长期借还历史。

---

### 4.5 RFID

定位：

> **钥匙身份校验机制。**

至少用于：

- 归还时确认实际放入的钥匙 ID；
- 防止错还、乱还；
- 作为借还状态确认链路的一部分。

需要后续硬件组进一步明确：

- RFID 是“归还口单点读取”，还是“全部在位检测”；
- 如果不是全槽位 RFID，则“实时在位状态”需要额外在位传感器，或明确由借还事件推导。

---

### 4.6 机械与电控部分

继续参考旧版机械思想：

- 钥匙固定于圆形转盘
- 步进电机驱动转盘
- 指定钥匙转到取钥口
- 电磁锁控制取还门
- 设置归零/定位机制
- 取钥结束后归位

旧版红外对射归零机制可以继续作为参考，但最终硬件方案由硬件组确定。

---

## 5. 用户角色

### 5.1 普通用户

可执行：

- 查看钥匙
- 搜索房间
- 查看钥匙是否在柜
- 预约钥匙
- 查看预约
- 查看当前借用
- 查看历史记录
- 查看超时状态
- 接收提醒

### 5.2 管理员

后续阶段增加：

- 查看所有钥匙
- 查看所有预约
- 审批预约（如启用）
- 查看借还记录
- 查看逾期
- 查看设备状态
- 禁用钥匙
- 修改钥匙信息
- 异常处理

---

## 6. 正式业务流程

## 6.1 预约流程

```text
用户打开小程序
    ↓
搜索房间 / 钥匙
    ↓
查看钥匙状态
    ↓
选择可预约钥匙
    ↓
提交预约
    ↓
后台检查是否可预约
    ↓
生成 Reservation
    ↓
返回预约成功
```

预约状态：

```text
ACTIVE
USED
CANCELLED
EXPIRED
```

---

## 6.2 取钥流程

正式主流程以柜体端为主：

```text
用户到达钥匙柜
    ↓
摄像头采集人脸
    ↓
RK3588/K230 人脸识别
    ↓
活体检测
    ↓
确认用户身份
    ↓
查询预约 / 权限
    ↓
触摸屏选择房间 / 钥匙
    ↓
向 ESP32 发送取钥请求
    ↓
ESP32 控制转盘
    ↓
定位钥匙
    ↓
打开取钥口
    ↓
用户取走钥匙
    ↓
设备确认执行结果
    ↓
MQTT 上报后台
    ↓
生成 / 更新 BorrowRecord
    ↓
小程序同步为“借用中”
```

---

## 6.3 归还流程

```text
用户到钥匙柜
    ↓
进入归还流程
    ↓
放入钥匙
    ↓
RFID 读取钥匙身份
    ↓
确认钥匙与借还记录匹配
    ↓
ESP32 执行归还机构动作
    ↓
确认归还完成
    ↓
MQTT 上报后台
    ↓
BorrowRecord = COMPLETED
    ↓
钥匙恢复“在柜”
    ↓
小程序同步显示
```

---

## 6.4 超时提醒流程

```text
BorrowRecord = BORROWED
    ↓
当前时间 > expectedReturnAt
    ↓
后台判断 OVERDUE
    ↓
生成 Reminder
    ↓
小程序展示 / 微信消息提醒
    ↓
用户归还
    ↓
Reminder 关闭
```

---

## 7. 小程序产品结构

底部 Tab 固定：

```text
首页
钥匙
记录
我的
```

### 7.1 首页

重点展示：

- 当前预约
- 当前借用
- 即将逾期 / 已逾期
- 快捷查询
- 设备状态

不把管理统计作为首页主体。

---

### 7.2 钥匙页

功能：

- 搜索房间号
- 搜索钥匙名称
- 按状态筛选
- 查看钥匙详情
- 查看所在钥匙柜
- 查看当前是否在柜
- 预约

示例状态：

```text
在柜
已预约
借出
逾期
维护
停用
异常
```

---

### 7.3 记录页

建议分类：

```text
预约
借用中
已完成
异常 / 超时
```

---

### 7.4 我的

包含：

- 用户资料
- 身份绑定
- 我的预约
- 我的借用
- 消息提醒
- 使用帮助
- 管理员入口

---

## 8. 小程序软件架构

当前仓库建议维持以下分层：

```text
miniprogram/
├── pages/
├── components/
├── services/
│   ├── device/
│   ├── key/
│   ├── reservation/
│   ├── borrow/
│   ├── user/
│   └── api/
├── models/
├── constants/
├── stores/
├── mocks/
└── utils/
```

原则：

```text
Page
 ↓
Service
 ↓
Mock / HTTP / MQTT
```

页面禁止直接：

```text
mqtt.publish(...)
wx.request(...)
```

统一通过 Service 调用。

---

## 9. 核心数据模型

## 9.1 User

```ts
interface User {
  id:string
  name:string
  studentNo?:string
  employeeNo?:string
  phone?:string
  role:'USER'|'ADMIN'
  status:'ACTIVE'|'DISABLED'
}
```

---

## 9.2 Key

`Key` 只表示钥匙本体和绑定关系。

```ts
interface Key {
  id:string
  name:string
  roomNo:string
  deviceId:string
  rfidTag?:string
  enabled:boolean
}
```

---

## 9.3 KeyPhysicalState

物理状态和业务状态必须拆开：

```text
IN_CABINET
MOVING
AT_PICKUP
OUT
RETURN_CHECK
FAULT
UNKNOWN
```

---

## 9.4 Reservation

```ts
interface Reservation {
  id:string
  userId:string
  keyId:string
  status:'ACTIVE'|'USED'|'CANCELLED'|'EXPIRED'
  reservedAt:number
  expiresAt:number
}
```

---

## 9.5 BorrowRecord

```text
BORROWING
BORROWED
OVERDUE
RETURNING
COMPLETED
EXCEPTION
```

```ts
interface BorrowRecord {
  id:string
  userId:string
  keyId:string
  deviceId:string
  status:BorrowStatus
  borrowedAt?:number
  expectedReturnAt?:number
  returnedAt?:number
}
```

---

## 9.6 Device

```text
ONLINE
OFFLINE
BUSY
FAULT
MAINTENANCE
```

---

## 9.7 DeviceEvent

建议保留以下事件：

```text
RECEIVED
AUTH_CONFIRMED
POSITIONING
POSITIONED
DOOR_OPEN
KEY_REMOVED
KEY_RETURNED
RFID_CONFIRMED
DOOR_CLOSED
HOMING
SUCCESS
FAILED
```

---

## 10. MQTT 通信边界

建议 Topic：

```text
keybox/{deviceId}/command
keybox/{deviceId}/event
keybox/{deviceId}/status
keybox/{deviceId}/heartbeat
```

命令示例：

```json
{
  "version":"1.0",
  "requestId":"REQ202609020001",
  "action":"BORROW",
  "keyId":"KEY103",
  "timestamp":1788342000
}
```

设备事件示例：

```json
{
  "version":"1.0",
  "requestId":"REQ202609020001",
  "event":"POSITIONING",
  "timestamp":1788342001
}
```

---

## 11. MQTT 与后台的原则

开发期允许：

```text
小程序
 ↓ WSS
MQTT Broker
 ↑
ESP32
```

用于设备联调。

正式业务建议：

```text
小程序
 ↓ HTTPS
后台
 ↓ MQTT
ESP32
```

原因：

- 预约不能依赖用户手机持续在线；
- 借还记录必须服务端持久化；
- 超时提醒必须服务端执行；
- 权限不能交给客户端决定；
- 设备最终状态必须可审计；
- 避免小程序长期持有高权限 MQTT 凭证。

---

## 12. 小程序 DeviceService 设计

保留当前仓库中的 `DeviceService` 思路。

统一提供：

```ts
getDeviceStatus(deviceId)
subscribeDevice(deviceId, listener)
unsubscribeDevice(deviceId, listener)

borrowKey(deviceId, keyId)
returnKey(deviceId, keyId)
```

开发阶段：

```text
DeviceService → MockDeviceService
```

联调阶段：

```text
DeviceService → MQTTDeviceService
```

正式阶段：

```text
业务操作 → Backend API
实时状态 → Backend WebSocket / MQTT Gateway
```

页面层不随底层实现改变。

---

## 13. 项目必须冻结的接口边界

小程序只允许关心：

```text
userId
keyId
roomNo
deviceId
reservationId
borrowRecordId
status
event
errorCode
timestamp
```

小程序不得关心：

```text
motorPulse
angle
DIR
GPIO
RFID register
camera driver
Linux device node
```

---

## 14. 当前必须明确的技术问题

以下问题必须由团队共同确认，否则后续会产生返工：

### P0

1. “实时在位状态”如何获得：
   - RFID 全柜读取？
   - 每槽位传感器？
   - 借还事件推导？

2. 正式业务后台由谁负责？

3. 后台部署位置：
   - 云服务器
   - 校内服务器
   - RK3588 本地服务器
   - 混合部署

4. 人脸身份与微信用户如何绑定？

5. 无预约是否允许现场直接借钥匙？

6. 是否存在管理员审批？

7. RFID 只负责归还还是借出也校验？

### P1

8. ESP32 与 RK3588/K230 串口协议；
9. MQTT Broker 由谁部署；
10. 设备离线后的业务策略；
11. Wi-Fi 与 4G 的切换策略；
12. 设备重复命令幂等策略；
13. 操作中断/断电恢复策略。

---

## 15. 非功能目标

### 15.1 稳定性

- 重复点击不能重复执行借还；
- MQTT 重连不能重复开锁；
- 网络断开后业务状态必须可恢复；
- 设备事件必须带 requestId；
- 借还记录必须可追踪。

### 15.2 安全性

- 小程序不保存设备超级权限；
- 正式开锁必须经过服务端/柜体身份验证；
- 人脸特征数据不暴露给小程序；
- 日志记录用户、时间、钥匙、设备和结果。

### 15.3 可扩展性

系统设计需支持：

- 多钥匙柜
- 多钥匙
- 多用户
- 多管理员
- 多校区/多实验室
- 更换 ESP32/STM32 等设备控制器
- K230 与 RK3588 互换
- Wi-Fi/4G 网络切换

---

## 16. V1 成功标准

V1 不要求所有硬件同时完成。

软件 V1 至少应做到：

1. 小程序可以独立运行；
2. 使用 Mock 数据完成钥匙查询；
3. 可以完成预约；
4. 可以查看预约；
5. 可以模拟借出；
6. 可以模拟归还；
7. 借还记录状态正确；
8. 当前借用正确展示；
9. 超时状态可以模拟；
10. DeviceService 可以切换 Mock / MQTT；
11. UI 不依赖 ESP/RK3588 具体实现；
12. 全流程有明确状态机。

---

## 17. 项目最终验收目标

整个项目最终应能演示：

```text
手机预约
 ↓
柜机人脸认证
 ↓
选择目标房间
 ↓
ESP32 驱动转盘
 ↓
指定钥匙送出
 ↓
设备上报借出状态
 ↓
小程序实时看到“借用中”
 ↓
归还钥匙
 ↓
RFID 校验
 ↓
归还成功
 ↓
借还记录完成
 ↓
超时情况能够主动提醒
```

这一闭环即为本项目最核心的最终成果。

