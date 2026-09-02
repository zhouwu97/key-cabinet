# 智能钥匙自助借还系统：开发计划

> 文档版本：V1.0  
> 当前重点：微信小程序优先独立开发，同时冻结跨端接口  
> 原则：先业务闭环，后真实硬件；先 Mock，后 MQTT；先协议，后联调

---

## 1. 开发总原则

本项目由多个模块并行开发：

```text
微信小程序
业务后台
RK3588/K230
人脸识别
ESP32
RFID
机械结构
电路硬件
```

为了避免互相等待，所有模块必须遵循：

> **接口先行、Mock 先行、状态机先行、联调最后进行。**

小程序开发不得等待：

- ESP32 完成
- RK3588/K230 完成
- 人脸识别完成
- RFID 完成
- 机械结构完成

小程序先使用 Mock 数据完成完整业务流程。

---

# 2. 总体阶段划分

```text
阶段 0：需求与架构冻结
阶段 1：小程序业务数据层
阶段 2：小程序核心业务闭环
阶段 3：小程序 UI 完整化
阶段 4：后台 API
阶段 5：MQTT 与 ESP32 联调
阶段 6：RK3588/K230 + 人脸联调
阶段 7：RFID + 归还闭环
阶段 8：全系统测试
阶段 9：结题与成果整理
```

---

# 3. 阶段 0：需求与架构冻结

## 目标

先把所有跨模块依赖写清楚，避免后续推倒重来。

## 必须确定

- [ ] 系统总架构
- [ ] 五名成员职责
- [ ] 小程序职责边界
- [ ] RK3588/K230 职责
- [ ] ESP32 职责
- [ ] RFID 职责
- [ ] 后台负责人
- [ ] 数据模型
- [ ] MQTT Topic
- [ ] MQTT JSON 协议
- [ ] ESP32 ↔ RK3588 串口协议
- [ ] 预约状态机
- [ ] 借还状态机
- [ ] 设备状态机
- [ ] “钥匙在位”检测方案

## 产物

```text
docs/
├── PROJECT-ARCHITECTURE.md
├── PLAN.md
├── MQTT-PROTOCOL.md
├── API-CONTRACT.md
└── STATE-MACHINE.md
```

## 完成条件

所有成员都能回答：

> 我的模块接收什么输入，输出什么结果，出错时返回什么。

---

# 4. 阶段 1：小程序业务数据层

## 当前优先级

**立即开始。**

目前 `key-cabinet` 已有：

- `User`
- `Key`
- `Device`
- `BorrowOrder`
- `DeviceService`
- `MockDeviceService`
- 4 Tab 骨架
- Mock 设备事件流

下一步不是 MQTT，而是补业务层。

## 4.1 调整现有模型

### Key

拆出物理状态：

```text
Key
KeyPhysicalState
```

不要把：

```text
AVAILABLE
BORROWED
MOVING
```

全部放在一个枚举中。

### 新增

```text
Reservation
BorrowRecord
Reminder
```

---

## 4.2 新增 Service

建议：

```text
services/
├── key/
├── reservation/
├── borrow/
├── user/
├── device/
└── reminder/
```

统一模式：

```ts
interface KeyService {}
interface ReservationService {}
interface BorrowService {}
```

第一版全部实现：

```text
MockKeyService
MockReservationService
MockBorrowService
```

---

## 4.3 Mock 数据

至少准备：

### 用户

```text
U001 普通用户
A001 管理员
```

### 柜子

```text
CAB001 在线
CAB002 离线
```

### 钥匙

至少 10 把：

```text
KEY101 在柜
KEY102 在柜
KEY103 在柜
KEY104 借出
KEY105 已预约
KEY106 逾期
KEY107 维护
KEY108 在柜
KEY109 在柜
KEY110 停用
```

### 借还记录

至少：

```text
1 条当前借用
2 条已完成
1 条逾期
```

---

## 4.4 本地持久化

开发阶段可使用：

```ts
wx.setStorageSync()
wx.getStorageSync()
```

用于：

- 模拟预约
- 模拟借出
- 模拟归还
- 模拟记录

正式接后台后替换。

## 阶段 1 完成标准

- [ ] 数据不再写死在 WXML
- [ ] 刷新后 Mock 数据仍能恢复
- [ ] 所有页面通过 Service 获取数据
- [ ] 页面层不直接管理业务数据库

---

# 5. 阶段 2：小程序核心业务闭环

这一阶段是当前最重要阶段。

目标：

> **完全没有真实硬件，也能走通预约 → 取钥 → 借用 → 归还 → 完成。**

---

## 5.1 首页

实现：

- [ ] 当前预约
- [ ] 当前借用
- [ ] 超时提示
- [ ] 设备状态
- [ ] 快捷入口

删除“纯占位文本”。

---

## 5.2 钥匙页

实现：

- [ ] 动态钥匙列表
- [ ] 房间号搜索
- [ ] 名称搜索
- [ ] 状态筛选
- [ ] 点击进入详情
- [ ] 显示所在柜
- [ ] 显示在位状态

URL：

```text
/pages/key-detail/key-detail?keyId=KEY103
```

禁止详情页写死 KEY103。

---

## 5.3 钥匙详情

显示：

```text
名称
房间号
钥匙编号
所在柜
物理状态
业务状态
是否可预约
```

操作按钮根据状态动态变化。

---

## 5.4 预约

新增：

```text
/pages/reservation-create/
```

或者改造现有 `borrow` 页面为“预约申请”。

实现：

- [ ] 选择预计使用时间
- [ ] 填写用途
- [ ] 检查钥匙是否允许预约
- [ ] 创建 Reservation
- [ ] 防止重复预约
- [ ] 提交成功后更新首页

---

## 5.5 记录

记录页改成：

```text
预约
借用中
已完成
异常
```

实现：

- [ ] Reservation 列表
- [ ] BorrowRecord 列表
- [ ] 状态标签
- [ ] 详情

---

## 5.6 Mock 取钥

现有 `operation` 页面保留为：

> 设备操作/开发联调页。

流程：

```text
Reservation ACTIVE
 ↓
模拟身份通过
 ↓
DeviceService.borrowKey
 ↓
RECEIVED
 ↓
POSITIONING
 ↓
DOOR_OPEN
 ↓
KEY_REMOVED
 ↓
SUCCESS
 ↓
Reservation USED
 ↓
BorrowRecord BORROWED
 ↓
KeyPhysicalState OUT
```

---

## 5.7 Mock 归还

```text
BorrowRecord BORROWED
 ↓
returnKey()
 ↓
POSITIONING
 ↓
DOOR_OPEN
 ↓
KEY_RETURNED
 ↓
RFID_CONFIRMED
 ↓
DOOR_CLOSED
 ↓
SUCCESS
 ↓
BorrowRecord COMPLETED
 ↓
KeyPhysicalState IN_CABINET
```

---

## 阶段 2 完成标准

可以完整演示：

```text
搜索 103
↓
预约
↓
首页出现待使用预约
↓
模拟柜机取钥
↓
首页出现借用中
↓
记录页出现借用记录
↓
模拟归还
↓
记录完成
↓
钥匙重新显示在柜
```

---

# 6. 阶段 3：小程序 UI 完整化

业务闭环跑通后再精修 UI。

## 设计目标

风格：

```text
工具型
简洁
可靠
清晰
弱装饰
```

建议：

- 白色主背景
- 浅灰页面底色
- 蓝色主操作色
- 绿色正常
- 橙色等待/即将逾期
- 红色异常/超时

避免：

- 复杂赛博风
- 过度玻璃拟态
- 大量渐变
- 过量动画

---

## 页面状态必须覆盖

每个主要页面都要有：

- [ ] Loading
- [ ] Empty
- [ ] Error
- [ ] Normal
- [ ] Offline
- [ ] Disabled

---

# 7. 阶段 4：业务后台

只有小程序 Mock 闭环稳定后，再接后台。

## 最低 API

### 用户

```http
GET /api/me
```

### 钥匙

```http
GET /api/keys
GET /api/keys/{id}
```

### 预约

```http
POST /api/reservations
GET /api/me/reservations
POST /api/reservations/{id}/cancel
```

### 借还记录

```http
GET /api/me/borrow-records
```

### 设备

```http
GET /api/devices
GET /api/devices/{id}
```

---

## 后台最低数据表

```text
users
keys
key_slots
reservations
borrow_records
devices
device_events
reminders
```

---

## 阶段 4 完成标准

- [ ] 小程序关闭 MockKeyService
- [ ] 小程序关闭 MockReservationService
- [ ] 小程序使用 API 获取业务数据
- [ ] 预约数据服务端保存
- [ ] 记录服务端保存

---

# 8. 阶段 5：MQTT 与 ESP32 联调

此阶段再开始真实设备通信。

## 8.1 Broker

先确定：

- Broker 地址
- 端口
- TLS
- WebSocket/WSS 是否开启
- 用户名密码
- Topic 权限

---

## 8.2 Topic

```text
keybox/{deviceId}/command
keybox/{deviceId}/event
keybox/{deviceId}/status
keybox/{deviceId}/heartbeat
```

---

## 8.3 联调顺序

不要一上来控制电机。

### Step 1

```text
ESP32 上线
↓
MQTT heartbeat
↓
后台显示 ONLINE
↓
小程序看到 ONLINE
```

### Step 2

```text
发送 TEST/PING
↓
ESP 返回 RECEIVED
```

### Step 3

```text
发送 BORROW KEY103
↓
ESP 只打印日志，不驱动电机
```

### Step 4

加入电机定位。

### Step 5

加入门锁。

### Step 6

加入完整设备事件。

---

## 阶段 5 完成标准

- [ ] ESP 自动重连
- [ ] 心跳正常
- [ ] requestId 回传正确
- [ ] 重复命令不重复执行
- [ ] 小程序能实时看到设备事件
- [ ] 设备失败有错误码

---

# 9. 阶段 6：RK3588/K230 + 人脸联调

## 目标

完成柜机身份认证与触摸屏主流程。

## 主要任务

- [ ] Linux 环境
- [ ] 摄像头
- [ ] 人脸检测
- [ ] 特征提取
- [ ] 人脸识别
- [ ] 活体检测
- [ ] 本地用户身份映射
- [ ] 触摸屏 UI
- [ ] 房间号输入
- [ ] 串口协议

---

## 推荐联调顺序

### 1

假人脸：

```text
按钮点击
↓
模拟 USER001 验证成功
```

先测后续串口。

### 2

加入真实人脸识别。

### 3

加入活体。

### 4

加入后台用户同步。

---

# 10. 阶段 7：RFID 与归还闭环

## 必须先确认

“钥匙在位”到底怎么判断。

### 方案 A

每个槽位传感器。

### 方案 B

全柜 RFID。

### 方案 C

仅归还口 RFID + 状态事件推导。

团队必须明确选一种。

---

## RFID 联调流程

```text
归还 KEY103
 ↓
RFID TAG103
 ↓
映射 KEY103
 ↓
查询当前 BorrowRecord
 ↓
匹配
 ↓
允许归还
```

错误情况：

```text
用户应还 KEY103
但读到 TAG104
```

系统必须：

```text
拒绝完成归还
上报 WRONG_KEY
提示用户
```

---

# 11. 阶段 8：全系统测试

## 11.1 正常链路

- [ ] 预约成功
- [ ] 人脸认证成功
- [ ] 指定钥匙定位
- [ ] 钥匙取出
- [ ] 小程序状态同步
- [ ] RFID 正确归还
- [ ] 记录闭环

---

## 11.2 异常链路

至少测试：

1. ESP32 离线
2. MQTT 断开
3. RK3588/K230 离线
4. 人脸认证失败
5. 活体失败
6. 钥匙不存在
7. 钥匙已借出
8. 电机定位失败
9. 电机卡住
10. 门锁打不开
11. 用户未取钥
12. 归还错误钥匙
13. RFID 读取失败
14. 柜门未关闭
15. 设备断电
16. 重复操作
17. 网络中断
18. 借用超时
19. 小程序退出后重新打开
20. 数据库与设备状态不一致

---

# 12. 测试记录格式

建立统一表格：

| ID | 场景 | 前置条件 | 操作 | 小程序 | 后台 | ESP | RK3588/K230 | 结果 |
|---|---|---|---|---|---|---|---|---|
| T001 | 正常预约 | KEY103 在柜 | 预约 | 成功 | ACTIVE | - | - | PASS |
| T002 | 设备离线 | CAB001 OFFLINE | 借用 | 禁止 | 拒绝 | 离线 | - | PASS |
| T003 | 错误归还 | 应还103 | 放入104 | 提示错误 | 不完成 | WRONG_KEY | - | PASS |

---

# 13. 当前 `key-cabinet` 仓库最近任务

按优先级：

## P0

- [ ] 修改 PRD，使“小程序预约/查询”为主，扫码开锁降级为调试功能
- [ ] 新增 `Reservation`
- [ ] 将 `BorrowOrder` 调整为 `BorrowRecord` 或明确其语义
- [ ] 拆分 `KeyPhysicalState`
- [ ] 新增 `KeyService`
- [ ] 新增 `ReservationService`
- [ ] 新增 `BorrowService`
- [ ] 建立 Mock 数据仓库
- [ ] 完成钥匙列表动态化
- [ ] 完成钥匙详情动态化
- [ ] 完成预约业务
- [ ] 完成首页状态联动
- [ ] 完成记录页

## P1

- [ ] 模拟柜机取钥
- [ ] 模拟 RFID 归还
- [ ] 模拟超时
- [ ] 操作中断恢复
- [ ] 全局业务状态持久化

## P2

- [ ] 管理员页面
- [ ] 故障上报
- [ ] 消息中心
- [ ] UI 精修

---

# 14. 小程序开发建议分支

建议：

```text
main
  └── develop
       ├── feature/data-layer
       ├── feature/key-list
       ├── feature/reservation
       ├── feature/borrow-record
       ├── feature/device-mock
       └── feature/ui-polish
```

个人开发也可以简化为：

```text
main
develop
```

每个阶段完成后再合并 `main`。

---

# 15. 每次提交最低要求

提交前运行：

```bash
npm run check
```

并确认：

- [ ] TypeScript 无错误
- [ ] app.json 页面存在
- [ ] 微信开发者工具编译正常
- [ ] 核心页面无控制台错误
- [ ] 状态切换符合文档
- [ ] 无硬编码 KEY103 依赖（除 Mock 示例）
- [ ] 无页面直接 MQTT 调用

---

# 16. 近期 7 天建议

## Day 1

- 重构数据模型
- 增加 Reservation
- 拆 KeyPhysicalState

## Day 2

- Mock 数据仓库
- KeyService

## Day 3

- 钥匙列表动态化
- 搜索/筛选

## Day 4

- 钥匙详情
- ReservationService

## Day 5

- 预约创建
- 首页预约展示

## Day 6

- BorrowService
- Mock 取钥

## Day 7

- Mock 归还
- 记录页
- 全流程测试

一周目标：

> 不接任何真实硬件，完整跑通 V1 业务闭环。

---

# 17. 中期目标

当小程序 Mock V1 完成后：

```text
Mock业务闭环
↓
真实后台
↓
MQTT心跳
↓
ESP控制
↓
人脸柜机
↓
RFID归还
↓
完整实物
```

禁止顺序反过来。

---

# 18. 最终成果目标

结题时至少准备：

- 一套可正常运行的钥匙存取装置
- 微信小程序
- 人脸识别与活体检测模块
- ESP32 设备控制程序
- RFID 校验模块
- 业务后台
- 借还记录
- 超时提醒
- 项目技术报告
- 系统测试报告
- 软件著作权或论文等成果材料

最终现场演示应能连续完成：

```text
手机预约
→ 人脸身份验证
→ 选择房间
→ 自动取钥
→ 手机状态更新
→ RFID归还
→ 手机记录完成
→ 超时提醒演示
```

