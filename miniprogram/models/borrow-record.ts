/**
 * 借还记录生命周期状态
 * 注意：OVERDUE 从主状态机分离，通过 status === BORROWED 且 now > expectedReturnAt 进行计算判断
 */
export enum BorrowRecordStatus {
  /** 取钥出柜中（设备正在执行操作） */
  BORROWING = 'BORROWING',
  /** 已借出（钥匙已被用户取走，处于借用期） */
  BORROWED = 'BORROWED',
  /** 归还入柜中（设备正在执行归还检测操作） */
  RETURNING = 'RETURNING',
  /** 已完成归还 */
  COMPLETED = 'COMPLETED',
  /** 异常状态（如还错钥匙、设备卡死等） */
  EXCEPTION = 'EXCEPTION',
}

export interface BorrowRecord {
  id: string
  userId: string
  keyId: string
  slotId: string
  deviceId: string
  reservationId?: string
  status: BorrowRecordStatus
  purpose?: string
  borrowedAt?: number
  expectedReturnAt: number
  /** 逾期触发时间戳（若发生逾期） */
  overdueAt?: number
  returnedAt?: number
  notes?: string
}

/**
 * 判断借还记录当前是否处于逾期状态
 */
export function isRecordOverdue(record: BorrowRecord, now: number = Date.now()): boolean {
  if (record.status === BorrowRecordStatus.COMPLETED) {
    return false
  }
  return now > record.expectedReturnAt
}
