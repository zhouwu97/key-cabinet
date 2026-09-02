import { KeyStatus } from '../models/key'
import { DeviceStatus, DeviceEvent } from '../models/device'
import { BorrowRecordStatus } from '../models/borrow-record'

export type StatusTone = 'blue' | 'green' | 'orange' | 'red' | 'gray'

export const KEY_STATUS_LABEL: Record<KeyStatus, string> = {
  [KeyStatus.AVAILABLE]: '可借用',
  [KeyStatus.RESERVED]: '已预约',
  [KeyStatus.BORROWED]: '借出中',
  [KeyStatus.OVERDUE]: '逾期未还',
  [KeyStatus.MAINTENANCE]: '维护中',
  [KeyStatus.DISABLED]: '已停用',
}

export const KEY_STATUS_TONE: Record<KeyStatus, StatusTone> = {
  [KeyStatus.AVAILABLE]: 'green',
  [KeyStatus.RESERVED]: 'blue',
  [KeyStatus.BORROWED]: 'orange',
  [KeyStatus.OVERDUE]: 'red',
  [KeyStatus.MAINTENANCE]: 'gray',
  [KeyStatus.DISABLED]: 'gray',
}

export const BORROW_STATUS_LABEL: Record<BorrowRecordStatus, string> = {
  [BorrowRecordStatus.BORROWING]: '借用中',
  [BorrowRecordStatus.BORROWED]: '借用中',
  [BorrowRecordStatus.OVERDUE]: '逾期',
  [BorrowRecordStatus.RETURNING]: '归还中',
  [BorrowRecordStatus.COMPLETED]: '已完成',
  [BorrowRecordStatus.EXCEPTION]: '异常',
}

export const DEVICE_STATUS_LABEL: Record<DeviceStatus, string> = {
  [DeviceStatus.ONLINE]: '在线',
  [DeviceStatus.OFFLINE]: '离线',
  [DeviceStatus.BUSY]: '执行中',
  [DeviceStatus.FAULT]: '故障',
  [DeviceStatus.MAINTENANCE]: '维护中',
}

export const DEVICE_EVENT_LABEL: Record<DeviceEvent, string> = {
  [DeviceEvent.RECEIVED]: '收到命令',
  [DeviceEvent.AUTH_CONFIRMED]: '身份验证',
  [DeviceEvent.POSITIONING]: '正在定位钥匙',
  [DeviceEvent.POSITIONED]: '定位完成',
  [DeviceEvent.DOOR_OPEN]: '柜门打开',
  [DeviceEvent.WAITING_REMOVE]: '等待取走钥匙',
  [DeviceEvent.KEY_REMOVED]: '钥匙已取',
  [DeviceEvent.KEY_RETURNED]: '已检测到钥匙归还',
  [DeviceEvent.RFID_CONFIRMED]: 'RFID确认',
  [DeviceEvent.DOOR_CLOSED]: '柜门关闭',
  [DeviceEvent.HOMING]: '设备正在归位',
  [DeviceEvent.SUCCESS]: '操作完成',
  [DeviceEvent.FAILED]: '操作失败',
}
