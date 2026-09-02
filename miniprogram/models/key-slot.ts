import { KeyPresenceState } from './key-presence'

/**
 * 钥匙柜物理槽位实体
 * 隔离业务层与机械电机层参数（脉冲数/角度/GPIO 等均不暴露给小程序业务）
 */
export interface KeySlot {
  /** 槽位全局唯一标识，例如 'CAB001_SLOT01' 或 'SLOT01' */
  id: string
  /** 所属设备ID，例如 'CAB001' */
  deviceId: string
  /** 物理槽位序号 (1, 2, 3...) */
  slotNo: number
  /** 当前绑定的钥匙ID（空槽位为 undefined） */
  keyId?: string
  /** 槽位钥匙在位状态 */
  presence: KeyPresenceState
  /** 槽位是否启用 */
  enabled: boolean
  /** 状态最后更新时间戳 */
  lastUpdated: number
}
