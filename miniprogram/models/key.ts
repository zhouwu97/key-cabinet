/**
 * 钥匙业务状态（与物理状态 KeyPhysicalState 分离）
 */
export enum KeyStatus {
  /** 可用（在柜且未被预约/借出） */
  AVAILABLE = 'AVAILABLE',
  /** 已预约 */
  RESERVED = 'RESERVED',
  /** 借出中 */
  BORROWED = 'BORROWED',
  /** 逾期 */
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
  status: KeyStatus
  rfidTag?: string
  enabled: boolean
  requiresApproval: boolean
  description?: string
}
