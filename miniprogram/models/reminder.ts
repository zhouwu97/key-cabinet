export enum ReminderType {
  /** 即将逾期 */
  APPROACHING_OVERDUE = 'APPROACHING_OVERDUE',
  /** 已逾期 */
  OVERDUE = 'OVERDUE',
  /** 预约即将过期 */
  RESERVATION_EXPIRING = 'RESERVATION_EXPIRING',
}

export enum ReminderStatus {
  ACTIVE = 'ACTIVE',
  READ = 'READ',
  RESOLVED = 'RESOLVED',
}

export interface Reminder {
  id: string
  userId: string
  type: ReminderType
  status: ReminderStatus
  title: string
  message: string
  relatedId: string // borrowRecordId or reservationId
  createdAt: number
  readAt?: number
}
