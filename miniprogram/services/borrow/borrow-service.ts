import { BorrowRecord, BorrowRecordStatus } from '../../models/borrow-record'

export interface BorrowService {
  /** 获取用户的借还记录 */
  getUserBorrowRecords(userId: string): Promise<BorrowRecord[]>

  /** 获取用户当前进行中的借用（BORROWED / BORROWING / RETURNING） */
  getCurrentBorrows(userId: string): Promise<BorrowRecord[]>

  /** 获取借还记录详情 */
  getBorrowRecordById(id: string): Promise<BorrowRecord | null>

  /** 根据钥匙ID获取当前处于活动状态的借用记录 */
  getActiveBorrowByKey(keyId: string): Promise<BorrowRecord | null>

  /** 创建借还记录（由设备取钥操作触发） */
  createBorrowRecord(
    userId: string,
    keyId: string,
    deviceId: string,
    slotId?: string,
    reservationId?: string,
    purpose?: string,
    expectedReturnAt?: number,
  ): Promise<BorrowRecord>

  /** 更新借还记录状态 */
  updateBorrowStatus(id: string, status: BorrowRecordStatus, notes?: string): Promise<void>

  /** 完成归还（由归还操作成功触发） */
  completeBorrowRecord(id: string): Promise<void>

  /** 检查逾期（更新逾期标记） */
  checkOverdue(): Promise<void>
}
