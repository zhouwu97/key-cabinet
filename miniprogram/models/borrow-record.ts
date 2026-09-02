export enum BorrowRecordStatus {
  /** 借用中（设备正在处理） */
  BORROWING = 'BORROWING',
  /** 已借出 */
  BORROWED = 'BORROWED',
  /** 逾期 */
  OVERDUE = 'OVERDUE',
  /** 归还中（设备正在处理） */
  RETURNING = 'RETURNING',
  /** 已完成 */
  COMPLETED = 'COMPLETED',
  /** 异常 */
  EXCEPTION = 'EXCEPTION',
}

export interface BorrowRecord {
  id: string
  userId: string
  keyId: string
  deviceId: string
  reservationId?: string
  status: BorrowRecordStatus
  purpose?: string
  borrowedAt?: number
  expectedReturnAt: number
  returnedAt?: number
  notes?: string
}
