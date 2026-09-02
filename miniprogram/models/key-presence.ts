/**
 * 钥匙物理在位状态
 * 纯粹表达钥匙在槽位/柜内的物理存在情况，不掺杂操作过程临时状态（如 MOVING、AT_PICKUP 等）
 */
export enum KeyPresenceState {
  /** 钥匙在柜内/槽位中 */
  PRESENT = 'PRESENT',
  /** 钥匙已离柜（已借出或取走） */
  ABSENT = 'ABSENT',
  /** 设备无法确认（检测中或传感器数据不确定） */
  UNKNOWN = 'UNKNOWN',
  /** 检测机构或槽位传感器异常 */
  FAULT = 'FAULT',
}
