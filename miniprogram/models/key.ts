export enum KeyStatus {
  AVAILABLE = 'AVAILABLE',
  RESERVED = 'RESERVED',
  PENDING = 'PENDING',
  BORROWED = 'BORROWED',
  OVERDUE = 'OVERDUE',
  MAINTENANCE = 'MAINTENANCE',
  LOST = 'LOST',
  DISABLED = 'DISABLED',
}

export interface Key {
  id: string
  name: string
  roomNo: string
  deviceId: string
  status: KeyStatus
  requiresApproval: boolean
}
