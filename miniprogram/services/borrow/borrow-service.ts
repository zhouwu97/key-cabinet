import { BorrowRecord } from '../../models/borrow-record'

export interface BorrowService {
  /** 获取用户的借还记录 */
  getUserBorrowRecords(userId: string): Promise<BorrowRecord[]>

  /** 获取用户当前借用 */
  getCurrentBorrows(userId: string): Promise<BorrowRecord[]>

  /** 获取借还记录详情 */
  getBorrowRecordById(id: string): Promise<BorrowRecord | null>

  /** 创建借还记录（由设备操作触发） */
  createBorrowRecord(
    userId: string,
    keyId: string,
    deviceId: string,
    reservationId?: string,
  ): Promise<BorrowRecord>

  /** 完成归还 */
  completeBorrowRecord(id: string): Promise<void>

  /** 检查逾期 */
  checkOverdue(): Promise<void>
}
