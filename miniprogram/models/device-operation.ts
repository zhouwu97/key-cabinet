import { OperationErrorCode } from './operation-error'

/**
 * 设备操作类型
 */
export enum DeviceOperationAction {
  /** 取钥匙 */
  PICKUP = 'PICKUP',
  /** 归还钥匙 */
  RETURN = 'RETURN',
}

/**
 * 设备操作生命周期状态
 */
export enum DeviceOperationStatus {
  /** 已创建 */
  CREATED = 'CREATED',
  /** 身份/预约验证通过，已授权 */
  AUTHORIZED = 'AUTHORIZED',
  /** 已向设备发送控制指令 */
  SENT = 'SENT',
  /** 设备正在执行机械动作（定位/开门/等待） */
  EXECUTING = 'EXECUTING',
  /** 操作成功完成 */
  SUCCESS = 'SUCCESS',
  /** 操作失败 */
  FAILED = 'FAILED',
  /** 操作超时 */
  TIMEOUT = 'TIMEOUT',
  /** 操作被取消 */
  CANCELLED = 'CANCELLED',
}

/**
 * 设备操作实体（可持久化、用于跨页面/断线恢复）
 */
export interface DeviceOperation {
  /** 操作唯一标识，例如 'OP202609020001' */
  id: string
  /** 一次操作对应的防重请求ID */
  requestId: string

  /** 操作类型 */
  action: DeviceOperationAction

  /** 操作人用户ID */
  userId: string
  /** 目标钥匙ID */
  keyId: string
  /** 关联槽位ID */
  slotId: string
  /** 关联设备ID */
  deviceId: string

  /** 关联预约ID（取钥时必填） */
  reservationId?: string
  /** 关联借用记录ID（取钥成功后生成/归还时传入） */
  borrowRecordId?: string

  /** 当前操作状态 */
  status: DeviceOperationStatus

  /** 失败或异常错误码 */
  errorCode?: OperationErrorCode | string

  /** 错误描述信息 */
  errorMessage?: string

  /** 创建时间戳 */
  createdAt: number
  /** 开始执行时间戳 */
  startedAt?: number
  /** 完成时间戳 */
  finishedAt?: number
}

/**
 * 发起设备操作输入参数
 */
export interface StartOperationInput {
  action: DeviceOperationAction
  userId: string
  keyId: string
  slotId?: string
  deviceId: string
  reservationId?: string
  borrowRecordId?: string
}
