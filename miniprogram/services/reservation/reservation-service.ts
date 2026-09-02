import { Reservation } from '../../models/reservation'

export interface CreateReservationParams {
  userId: string
  keyId: string
  purpose?: string
  pickupWindowStart?: number // 预约取钥窗口开始（毫秒时间戳，默认当前时间）
  pickupWindowEnd?: number   // 预约取钥窗口截止（毫秒时间戳，默认开始+30分钟）
  expectedDuration?: number  // 预计使用时长（毫秒，默认2小时）
  expectedReturnAt?: number  // 预计归还时间戳
}

export interface ReservationService {
  /** 创建预约（含冲突检测） */
  createReservation(params: CreateReservationParams): Promise<Reservation>

  /** 获取用户的预约列表 */
  getUserReservations(userId: string): Promise<Reservation[]>

  /** 获取预约详情 */
  getReservationById(id: string): Promise<Reservation | null>

  /** 取消预约 */
  cancelReservation(id: string): Promise<void>

  /** 检查钥匙在指定时间段是否可预约 */
  canReserveKey(keyId: string, windowStart?: number, windowEnd?: number): Promise<boolean>

  /** 获取用户指定钥匙或任意钥匙的当前活跃预约 */
  getActiveReservation(userId: string, keyId?: string): Promise<Reservation | null>

  /** 将预约标记为已使用（取钥成功后调用） */
  markReservationUsed(id: string): Promise<void>
}
