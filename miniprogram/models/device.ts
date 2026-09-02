import { OperationErrorCode } from './operation-error'

export enum DeviceStatus {
  ONLINE = 'ONLINE',
  OFFLINE = 'OFFLINE',
  BUSY = 'BUSY',
  FAULT = 'FAULT',
  MAINTENANCE = 'MAINTENANCE',
}

export enum DeviceEvent {
  /** 收到控制指令 */
  RECEIVED = 'RECEIVED',
  /** 用户身份认证通过 */
  AUTH_CONFIRMED = 'AUTH_CONFIRMED',
  /** 电机开始定位槽位 */
  POSITIONING = 'POSITIONING',
  /** 电机定位到位 */
  POSITIONED = 'POSITIONED',
  /** 柜门打开 */
  DOOR_OPEN = 'DOOR_OPEN',
  /** 等待用户取走钥匙 */
  WAITING_REMOVE = 'WAITING_REMOVE',
  /** 传感器检测到钥匙已被取出 */
  KEY_REMOVED = 'KEY_REMOVED',
  /** 归还口检测到钥匙放入 */
  KEY_RETURNED = 'KEY_RETURNED',
  /** RFID 读取并确认钥匙身份正确 */
  RFID_CONFIRMED = 'RFID_CONFIRMED',
  /** 柜门关闭 */
  DOOR_CLOSED = 'DOOR_CLOSED',
  /** 机构归零复位中 */
  HOMING = 'HOMING',
  /** 操作成功完成 */
  SUCCESS = 'SUCCESS',
  /** 操作失败 */
  FAILED = 'FAILED',
}

export type DeviceAction = 'PICKUP' | 'RETURN'

export interface Device {
  id: string
  name: string
  status: DeviceStatus
  lastHeartbeat: number
}

export interface DeviceCommandMessage {
  version: string
  operationId: string
  requestId: string
  action: DeviceAction
  keyId: string
  slotId: string
  deviceId: string
  timestamp: number
}

export interface DeviceEventMessage {
  version: string
  operationId: string
  requestId: string
  eventId: string
  seq: number
  deviceId: string
  event: DeviceEvent
  errorCode?: OperationErrorCode | string
  errorMessage?: string
  timestamp: number
}
