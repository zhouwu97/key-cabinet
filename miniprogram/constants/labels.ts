import { KeyStatus } from '../models/key'
import { BorrowOrderStatus } from '../models/order'
import { DeviceStatus, DeviceEvent } from '../models/device'

export type StatusTone = 'blue' | 'green' | 'orange' | 'red' | 'gray'

export const KEY_STATUS_LABEL: Record<KeyStatus, string> = {
  [KeyStatus.AVAILABLE]: '可借',
  [KeyStatus.RESERVED]: '已预约',
  [KeyStatus.PENDING]: '审批中',
  [KeyStatus.BORROWED]: '借出',
  [KeyStatus.OVERDUE]: '逾期',
  [KeyStatus.MAINTENANCE]: '维护',
  [KeyStatus.LOST]: '遗失',
  [KeyStatus.DISABLED]: '停用',
}

export const KEY_STATUS_TONE: Record<KeyStatus, StatusTone> = {
  [KeyStatus.AVAILABLE]: 'green',
  [KeyStatus.RESERVED]: 'blue',
  [KeyStatus.PENDING]: 'orange',
  [KeyStatus.BORROWED]: 'orange',
  [KeyStatus.OVERDUE]: 'red',
  [KeyStatus.MAINTENANCE]: 'gray',
  [KeyStatus.LOST]: 'red',
  [KeyStatus.DISABLED]: 'gray',
}

export const ORDER_STATUS_LABEL: Record<BorrowOrderStatus, string> = {
  [BorrowOrderStatus.PENDING_APPROVAL]: '待审批',
  [BorrowOrderStatus.APPROVED]: '已批准',
  [BorrowOrderStatus.WAITING_PICKUP]: '待取钥',
  [BorrowOrderStatus.PICKING]: '取钥中',
  [BorrowOrderStatus.BORROWED]: '借用中',
  [BorrowOrderStatus.RETURNING]: '归还中',
  [BorrowOrderStatus.COMPLETED]: '已完成',
  [BorrowOrderStatus.OVERDUE]: '已逾期',
  [BorrowOrderStatus.CANCELLED]: '已取消',
  [BorrowOrderStatus.REJECTED]: '已拒绝',
  [BorrowOrderStatus.EXCEPTION]: '异常',
}

export const DEVICE_STATUS_LABEL: Record<DeviceStatus, string> = {
  [DeviceStatus.ONLINE]: '在线',
  [DeviceStatus.OFFLINE]: '离线',
  [DeviceStatus.BUSY]: '执行中',
  [DeviceStatus.FAULT]: '故障',
}

export const DEVICE_EVENT_LABEL: Record<DeviceEvent, string> = {
  [DeviceEvent.RECEIVED]: '收到命令',
  [DeviceEvent.POSITIONING]: '正在定位钥匙',
  [DeviceEvent.POSITIONED]: '定位完成',
  [DeviceEvent.DOOR_OPEN]: '柜门打开',
  [DeviceEvent.WAITING_REMOVE]: '等待取走钥匙',
  [DeviceEvent.KEY_REMOVED]: '钥匙已取',
  [DeviceEvent.KEY_RETURNED]: '已检测到钥匙归还',
  [DeviceEvent.DOOR_CLOSED]: '柜门关闭',
  [DeviceEvent.HOMING]: '设备正在归位',
  [DeviceEvent.SUCCESS]: '操作完成',
}
