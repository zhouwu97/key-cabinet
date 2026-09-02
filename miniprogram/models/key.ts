/**
 * 钥匙业务状态（与物理在位状态 KeyPresenceState 分离）
 */
export enum KeyStatus {
  /** 可用（在柜且未被借出/未在有效预约中） */
  AVAILABLE = 'AVAILABLE',
  /** 已预约 */
  RESERVED = 'RESERVED',
  /** 借出中 */
  BORROWED = 'BORROWED',
  /** 逾期未还 */
  OVERDUE = 'OVERDUE',
  /** 维护中 */
  MAINTENANCE = 'MAINTENANCE',
  /** 已停用 */
  DISABLED = 'DISABLED',
}

export interface Key {
  id: string
  name: string
  roomNo: string
  deviceId: string
  slotId: string
  status: KeyStatus
  rfidTag?: string
  enabled: boolean
  requiresApproval: boolean
  description?: string
}
