# 智能钥匙自助借还系统 错误码与异常状态规范

> 版本：V2.0
> 生效阶段：阶段 2 冻结

---

## 一、错误码分类规范

所有系统错误码均采用大写下划线命名，按业务层与硬件层模块化划分：

| 模块分类 | 前缀 | 描述 |
| :--- | :--- | :--- |
| 认证与身份 | `AUTH_` | 人脸识别、活体检测、微信认证、操作权限 |
| 预约服务 | `RESERVATION_` | 预约创建、时间窗冲突、过期与状态异常 |
| 钥匙与槽位 | `KEY_` / `SLOT_` | 钥匙在位、状态可用性、槽位状态 |
| 设备通信 | `DEVICE_` | 设备心跳、在线状态、并发独占锁 |
| 机械执行 | `MOTOR_` / `DOOR_` | 电机定位、归零、柜门开闭、取钥超时 |
| 射频识别 | `RFID_` | 标签读取、钥匙比对错误 |
| 操作生命周期 | `OPERATION_` | 操作幂等、中断恢复、超时与冲突 |
| 系统网络 | `SYSTEM_` / `NETWORK_` | 网络异常、数据持久化异常 |

---

## 二、详细错误码列表

### 1. 认证与权限 (AUTH)
- `AUTH_FAILED`: 身份认证失败
- `AUTH_FACE_FAILED`: 人脸特征未匹配到合法用户
- `AUTH_LIVENESS_FAILED`: 活体检测未通过
- `AUTH_UNAUTHORIZED`: 用户无权执行该操作

### 2. 预约业务 (RESERVATION)
- `RESERVATION_NOT_FOUND`: 预约单不存在
- `RESERVATION_EXPIRED`: 预约时间窗口已过，无法用于取钥
- `RESERVATION_CONFLICT`: 所选时间段与其他有效预约存在时间交叉
- `RESERVATION_NOT_ACTIVE`: 预约单非 ACTIVE 状态
- `RESERVATION_ALREADY_USED`: 预约单已被使用

### 3. 钥匙与槽位 (KEY / SLOT)
- `KEY_NOT_FOUND`: 指定钥匙不存在
- `KEY_NOT_AVAILABLE`: 钥匙已被借出、停用或处于维护状态
- `KEY_NOT_PRESENT`: 槽位传感器显示钥匙物理不在位
- `KEY_ALREADY_BORROWED`: 钥匙已被其他借用记录占用
- `KEY_WRONG_RETURN`: 归还钥匙与目标钥匙不匹配
- `KEY_DISABLED`: 钥匙已停用
- `SLOT_NOT_FOUND`: 指定槽位不存在
- `SLOT_DISABLED`: 槽位已停用

### 4. 设备状态 (DEVICE)
- `DEVICE_NOT_FOUND`: 设备不存在
- `DEVICE_OFFLINE`: 钥匙柜设备处于离线状态
- `DEVICE_BUSY`: 钥匙柜正在执行其他任务，设备忙
- `DEVICE_FAULT`: 设备存在机械或电气故障

### 5. 机械与执行机构 (MOTOR / DOOR)
- `MOTOR_POSITION_FAILED`: 电机旋转或托盘定位失败
- `MOTOR_HOME_FAILED`: 电机归零复位失败
- `DOOR_OPEN_FAILED`: 柜门打开失败（电磁锁或行程开关异常）
- `DOOR_CLOSE_FAILED`: 柜门关闭失败
- `DOOR_NOT_CLOSED`: 用户操作完毕后柜门未能在超时前关好
- `KEY_NOT_REMOVED`: 柜门打开后用户超时未取走钥匙

### 6. 射频识别 (RFID)
- `RFID_NOT_FOUND`: 未能在规定时间内感应到 RFID 标签
- `RFID_WRONG_KEY`: 检测到的 RFID 标签与当前归还记录的钥匙 ID 不一致
- `RFID_READ_TIMEOUT`: RFID 读取超时

### 7. 操作生命周期 (OPERATION)
- `OPERATION_NOT_FOUND`: 指定操作 ID 不存在
- `OPERATION_TIMEOUT`: 设备操作整体流程超时
- `OPERATION_DUPLICATED`: 重复提交同一操作请求
- `OPERATION_STATE_CONFLICT`: 操作状态流转冲突
- `OPERATION_USER_MISMATCH`: 试图操作非当前用户的借还记录
