import { KeyStatus } from '../models/key'
import { DeviceStatus, DeviceEvent } from '../models/device'
import { BorrowRecord, BorrowRecordStatus, isRecordOverdue } from '../models/borrow-record'
import { ReservationStatus } from '../models/reservation'
import { KeyPresenceState } from '../models/key-presence'
import { DeviceOperationStatus } from '../models/device-operation'

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

export const KEY_PRESENCE_LABEL: Record<KeyPresenceState, string> = {
  [KeyPresenceState.PRESENT]: '在柜',
  [KeyPresenceState.ABSENT]: '离柜',
  [KeyPresenceState.UNKNOWN]: '未知',
  [KeyPresenceState.FAULT]: '传感器异常',
}

export const RESERVATION_STATUS_LABEL: Record<ReservationStatus, string> = {
  [ReservationStatus.PENDING]: '待审批',
  [ReservationStatus.APPROVED]: '已批准',
  [ReservationStatus.ACTIVE]: '待取钥',
  [ReservationStatus.USED]: '已取走',
  [ReservationStatus.REJECTED]: '已拒绝',
  [ReservationStatus.CANCELLED]: '已取消',
  [ReservationStatus.EXPIRED]: '已过期',
}

export const RESERVATION_STATUS_TONE: Record<ReservationStatus, StatusTone> = {
  [ReservationStatus.PENDING]: 'orange',
  [ReservationStatus.APPROVED]: 'blue',
  [ReservationStatus.ACTIVE]: 'blue',
  [ReservationStatus.USED]: 'green',
  [ReservationStatus.REJECTED]: 'red',
  [ReservationStatus.CANCELLED]: 'gray',
  [ReservationStatus.EXPIRED]: 'gray',
}

export const BORROW_STATUS_LABEL: Record<BorrowRecordStatus, string> = {
  [BorrowRecordStatus.BORROWING]: '出柜中',
  [BorrowRecordStatus.BORROWED]: '借用中',
  [BorrowRecordStatus.RETURNING]: '入柜中',
  [BorrowRecordStatus.COMPLETED]: '已归还',
  [BorrowRecordStatus.EXCEPTION]: '异常',
}

export function getBorrowRecordDisplayStatus(record: BorrowRecord): { label: string; tone: StatusTone } {
  if (record.status === BorrowRecordStatus.COMPLETED) {
    return { label: '已归还', tone: 'green' }
  }
  if (record.status === BorrowRecordStatus.EXCEPTION) {
    return { label: '异常', tone: 'red' }
  }
  if (record.status === BorrowRecordStatus.BORROWING) {
    return { label: '出柜中', tone: 'blue' }
  }
  if (record.status === BorrowRecordStatus.RETURNING) {
    return { label: '入柜中', tone: 'blue' }
  }
  if (isRecordOverdue(record)) {
    return { label: '已逾期', tone: 'red' }
  }
  return { label: '借用中', tone: 'orange' }
}

export const DEVICE_STATUS_LABEL: Record<DeviceStatus, string> = {
  [DeviceStatus.ONLINE]: '在线',
  [DeviceStatus.OFFLINE]: '离线',
  [DeviceStatus.BUSY]: '执行中',
  [DeviceStatus.FAULT]: '故障',
  [DeviceStatus.MAINTENANCE]: '维护中',
}

export const DEVICE_EVENT_LABEL: Record<DeviceEvent, string> = {
  [DeviceEvent.RECEIVED]: '收到操作指令',
  [DeviceEvent.AUTH_CONFIRMED]: '身份验证成功',
  [DeviceEvent.POSITIONING]: '正在定位钥匙槽位',
  [DeviceEvent.POSITIONED]: '槽位定位完成',
  [DeviceEvent.DOOR_OPEN]: '安全柜门已打开',
  [DeviceEvent.WAITING_REMOVE]: '等待取走钥匙',
  [DeviceEvent.KEY_REMOVED]: '钥匙已被取走',
  [DeviceEvent.KEY_RETURNED]: '检测到钥匙放入归还口',
  [DeviceEvent.RFID_CONFIRMED]: 'RFID 身份校验通过',
  [DeviceEvent.DOOR_CLOSED]: '柜门已关闭锁止',
  [DeviceEvent.HOMING]: '转盘复位归零中',
  [DeviceEvent.SUCCESS]: '操作圆满完成',
  [DeviceEvent.FAILED]: '操作异常终止',
}

export const OPERATION_STATUS_LABEL: Record<DeviceOperationStatus, string> = {
  [DeviceOperationStatus.CREATED]: '已创建',
  [DeviceOperationStatus.AUTHORIZED]: '已授权',
  [DeviceOperationStatus.SENT]: '已下发指令',
  [DeviceOperationStatus.EXECUTING]: '执行中',
  [DeviceOperationStatus.SUCCESS]: '操作成功',
  [DeviceOperationStatus.FAILED]: '操作失败',
  [DeviceOperationStatus.TIMEOUT]: '操作超时',
  [DeviceOperationStatus.CANCELLED]: '已取消',
}
