import { Reservation, ReservationStatus } from '../../models/reservation'
import {
  CreateReservationParams,
  ReservationService,
} from './reservation-service'
import { MOCK_RESERVATIONS, STORAGE_KEYS } from '../../mocks/mock-data'
import { KeyStatus } from '../../models/key'

export class MockReservationService implements ReservationService {
  private reservations: Reservation[] = []

  constructor() {
    this.loadFromStorage()
  }

  private loadFromStorage(): void {
    try {
      const stored = wx.getStorageSync(STORAGE_KEYS.RESERVATIONS)
      this.reservations =
        stored && stored.length > 0 ? stored : [...MOCK_RESERVATIONS]
      this.saveToStorage()
    } catch (e) {
      console.error('加载预约数据失败', e)
      this.reservations = [...MOCK_RESERVATIONS]
    }
  }

  private saveToStorage(): void {
    try {
      wx.setStorageSync(STORAGE_KEYS.RESERVATIONS, this.reservations)
    } catch (e) {
      console.error('保存预约数据失败', e)
    }
  }

  private generateId(): string {
    return `RSV${Date.now()}${Math.random().toString(36).substring(2, 9)}`
  }

  async createReservation(
    params: CreateReservationParams,
  ): Promise<Reservation> {
    return new Promise((resolve, reject) => {
      setTimeout(async () => {
        // 检查是否可预约
        const canReserve = await this.canReserveKey(params.keyId)
        if (!canReserve) {
          reject(new Error('该钥匙当前不可预约'))
          return
        }

        // 检查用户是否已有该钥匙的活跃预约
        const existing = await this.getActiveReservation(
          params.userId,
          params.keyId,
        )
        if (existing) {
          reject(new Error('您已预约过该钥匙'))
          return
        }

        const now = Date.now()
        const reservation: Reservation = {
          id: this.generateId(),
          userId: params.userId,
          keyId: params.keyId,
          status: ReservationStatus.ACTIVE,
          purpose: params.purpose,
          reservedAt: now,
          expiresAt: now + params.expectedDuration,
        }

        this.reservations.push(reservation)
        this.saveToStorage()

        // 更新钥匙状态为 RESERVED
        try {
          const keys = wx.getStorageSync(STORAGE_KEYS.KEYS) || []
          const key = keys.find((k: any) => k.id === params.keyId)
          if (key) {
            key.status = KeyStatus.RESERVED
            wx.setStorageSync(STORAGE_KEYS.KEYS, keys)
          }
        } catch (e) {
          console.error('更新钥匙状态失败', e)
        }

        resolve({ ...reservation })
      }, 300)
    })
  }

  async getUserReservations(userId: string): Promise<Reservation[]> {
    return new Promise(resolve => {
      setTimeout(() => {
        const results = this.reservations
          .filter(r => r.userId === userId)
          .sort((a, b) => b.reservedAt - a.reservedAt)
        resolve(results)
      }, 200)
    })
  }

  async getReservationById(id: string): Promise<Reservation | null> {
    return new Promise(resolve => {
      setTimeout(() => {
        const reservation = this.reservations.find(r => r.id === id)
        resolve(reservation ? { ...reservation } : null)
      }, 150)
    })
  }

  async cancelReservation(id: string): Promise<void> {
    return new Promise(resolve => {
      setTimeout(() => {
        const reservation = this.reservations.find(r => r.id === id)
        if (reservation && reservation.status === ReservationStatus.ACTIVE) {
          reservation.status = ReservationStatus.CANCELLED
          reservation.cancelledAt = Date.now()
          this.saveToStorage()

          // 更新钥匙状态回 AVAILABLE
          try {
            const keys = wx.getStorageSync(STORAGE_KEYS.KEYS) || []
            const key = keys.find((k: any) => k.id === reservation.keyId)
            if (key && key.status === KeyStatus.RESERVED) {
              key.status = KeyStatus.AVAILABLE
              wx.setStorageSync(STORAGE_KEYS.KEYS, keys)
            }
          } catch (e) {
            console.error('更新钥匙状态失败', e)
          }
        }
        resolve()
      }, 200)
    })
  }

  async canReserveKey(keyId: string): Promise<boolean> {
    return new Promise(resolve => {
      setTimeout(() => {
        try {
          const keys = wx.getStorageSync(STORAGE_KEYS.KEYS) || []
          const key = keys.find((k: any) => k.id === keyId)

          if (!key || !key.enabled) {
            resolve(false)
            return
          }

          // 只有 AVAILABLE 状态才能预约
          const canReserve = key.status === KeyStatus.AVAILABLE
          resolve(canReserve)
        } catch (e) {
          console.error('检查钥匙状态失败', e)
          resolve(false)
        }
      }, 100)
    })
  }

  async getActiveReservation(
    userId: string,
    keyId: string,
  ): Promise<Reservation | null> {
    return new Promise(resolve => {
      setTimeout(() => {
        const reservation = this.reservations.find(
          r =>
            r.userId === userId &&
            r.keyId === keyId &&
            r.status === ReservationStatus.ACTIVE,
        )
        resolve(reservation ? { ...reservation } : null)
      }, 100)
    })
  }
}
