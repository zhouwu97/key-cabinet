/**
 * 钥匙物理状态
 * 与业务状态（KeyStatus）分离
 */
export enum KeyPhysicalState {
  /** 在柜中 */
  IN_CABINET = 'IN_CABINET',
  /** 转盘移动中 */
  MOVING = 'MOVING',
  /** 在取钥口 */
  AT_PICKUP = 'AT_PICKUP',
  /** 已取出 */
  OUT = 'OUT',
  /** 归还检查中 */
  RETURN_CHECK = 'RETURN_CHECK',
  /** 故障 */
  FAULT = 'FAULT',
  /** 未知 */
  UNKNOWN = 'UNKNOWN',
}

export interface KeyLocation {
  keyId: string
  deviceId: string
  physicalState: KeyPhysicalState
  slotPosition?: number
  lastUpdated: number
}
