export enum ReservationStatus {
  /** 待审批（开启审批流程时使用） */
  PENDING = 'PENDING',
  /** 已审批通过 */
  APPROVED = 'APPROVED',
  /** 处于可取钥匙的有效窗口内 */
  ACTIVE = 'ACTIVE',
  /** 已取钥履约完成 */
  USED = 'USED',
  /** 审批拒绝 */
  REJECTED = 'REJECTED',
  /** 已取消 */
  CANCELLED = 'CANCELLED',
  /** 超期未取已过期 */
  EXPIRED = 'EXPIRED',
}

export interface Reservation {
  id: string
  userId: string
  keyId: string
  status: ReservationStatus
  purpose?: string
  createdAt: number

  /** 取钥时间窗口开始时间戳 */
  pickupWindowStart: number
  /** 取钥时间窗口截止时间戳 */
  pickupWindowEnd: number
  /** 预计借用归还截止时间戳 */
  expectedReturnAt: number

  approvedAt?: number
  usedAt?: number
  cancelledAt?: number
	reviewedAt?: number
	reviewedBy?: string
	rejectionReason?: string
	userName?: string
	studentNo?: string
	keyName?: string
	roomNo?: string
	deviceId?: string
	deviceName?: string
}

/** 预约状态与取钥窗口同时满足，才允许进入现场操作。 */
export function canPickupReservation(reservation: Reservation, now: number = Date.now()): boolean {
  return (
    (reservation.status === ReservationStatus.ACTIVE || reservation.status === ReservationStatus.APPROVED) &&
    now >= reservation.pickupWindowStart && now <= reservation.pickupWindowEnd
  )
}

export function canCancelReservation(reservation: Reservation): boolean {
  return [ReservationStatus.PENDING, ReservationStatus.ACTIVE, ReservationStatus.APPROVED].includes(reservation.status)
}
