export enum DeviceStatus {
  ONLINE = 'ONLINE',
  OFFLINE = 'OFFLINE',
  BUSY = 'BUSY',
  FAULT = 'FAULT',
}

export enum DeviceEvent {
  RECEIVED = 'RECEIVED',
  POSITIONING = 'POSITIONING',
  POSITIONED = 'POSITIONED',
  DOOR_OPEN = 'DOOR_OPEN',
  WAITING_REMOVE = 'WAITING_REMOVE',
  KEY_REMOVED = 'KEY_REMOVED',
  KEY_RETURNED = 'KEY_RETURNED',
  DOOR_CLOSED = 'DOOR_CLOSED',
  HOMING = 'HOMING',
  SUCCESS = 'SUCCESS',
}

export enum DeviceErrorCode {
  DEVICE_OFFLINE = 'DEVICE_OFFLINE',
  DEVICE_BUSY = 'DEVICE_BUSY',
  MOTOR_ERROR = 'MOTOR_ERROR',
  POSITION_ERROR = 'POSITION_ERROR',
  DOOR_ERROR = 'DOOR_ERROR',
  KEY_NOT_REMOVED = 'KEY_NOT_REMOVED',
  KEY_NOT_RETURNED = 'KEY_NOT_RETURNED',
  TIMEOUT = 'TIMEOUT',
}

export type DeviceAction = 'BORROW' | 'RETURN'

export interface Device {
  id: string
  name: string
  status: DeviceStatus
  lastHeartbeat: number
}

export interface DeviceCommandMessage {
  version: string
  requestId: string
  action: DeviceAction
  keyId: string
  timestamp: number
}

export interface DeviceEventMessage {
  version: string
  requestId: string
  event: DeviceEvent
  errorCode?: DeviceErrorCode
  timestamp: number
}
