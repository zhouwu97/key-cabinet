import { Reservation } from '../../models/reservation'

export interface CreateReservationParams {
  userId: string
  keyId: string
  purpose?: string
  expectedDuration: number // 预计使用时长（毫秒）
}

export interface ReservationService {
  /** 创建预约 */
  createReservation(params: CreateReservationParams): Promise<Reservation>

  /** 获取用户的预约列表 */
  getUserReservations(userId: string): Promise<Reservation[]>

  /** 获取预约详情 */
  getReservationById(id: string): Promise<Reservation | null>

  /** 取消预约 */
  cancelReservation(id: string): Promise<void>

  /** 检查钥匙是否可预约 */
  canReserveKey(keyId: string): Promise<boolean>

  /** 获取用户当前活跃预约 */
  getActiveReservation(userId: string, keyId: string): Promise<Reservation | null>
}
