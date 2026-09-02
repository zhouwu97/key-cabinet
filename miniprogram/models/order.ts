export enum BorrowOrderStatus {
  PENDING_APPROVAL = 'PENDING_APPROVAL',
  APPROVED = 'APPROVED',
  WAITING_PICKUP = 'WAITING_PICKUP',
  PICKING = 'PICKING',
  BORROWED = 'BORROWED',
  RETURNING = 'RETURNING',
  COMPLETED = 'COMPLETED',
  OVERDUE = 'OVERDUE',
  CANCELLED = 'CANCELLED',
  REJECTED = 'REJECTED',
  EXCEPTION = 'EXCEPTION',
}

export interface BorrowOrder {
  id: string
  userId: string
  keyId: string
  deviceId: string
  status: BorrowOrderStatus
  purpose?: string
  borrowedAt?: number
  expectedReturnAt: number
  returnedAt?: number
}
