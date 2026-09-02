export enum ReservationStatus {
  ACTIVE = 'ACTIVE',
  USED = 'USED',
  CANCELLED = 'CANCELLED',
  EXPIRED = 'EXPIRED',
}

export interface Reservation {
  id: string
  userId: string
  keyId: string
  status: ReservationStatus
  purpose?: string
  reservedAt: number
  expiresAt: number
  usedAt?: number
  cancelledAt?: number
}
