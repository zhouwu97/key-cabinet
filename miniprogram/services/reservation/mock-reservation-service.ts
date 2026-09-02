import { Reservation, ReservationStatus } from '../../models/reservation'
import {
  CreateReservationParams,
  ReservationService,
} from './reservation-service'
import { MOCK_RESERVATIONS, STORAGE_KEYS } from '../../mocks/mock-data'
import { KeyStatus } from '../../models/key'
import { BorrowRecordStatus } from '../../models/borrow-record'
import { DeviceStatus } from '../../models/device'
import { OperationErrorCode } from '../../models/operation-error'

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
    return `RSV${Date.now()}${Math.random().toString(36).substring(2, 7).toUpperCase()}`
  }

  async createReservation(
    params: CreateReservationParams,
  ): Promise<Reservation> {
    this.loadFromStorage()
    return new Promise((resolve, reject) => {
      setTimeout(async () => {
        const now = Date.now()
        const start = params.pickupWindowStart || now
        const end = params.pickupWindowEnd || (start + 1800000) // 默认窗口 30 分钟
        const duration = params.expectedDuration || 7200000 // 默认借用 2 小时
        const expectedReturn = params.expectedReturnAt || (start + duration)

        // 1. 检查钥匙基础状态
        const keys = wx.getStorageSync(STORAGE_KEYS.KEYS) || []
        const key = keys.find((k: any) => k.id === params.keyId)
        if (!key || !key.enabled) {
          reject(new Error(OperationErrorCode.KEY_NOT_AVAILABLE))
          return
        }

        // 2. 检查设备状态
        const devices = wx.getStorageSync(STORAGE_KEYS.DEVICES) || []
        const device = devices.find((d: any) => d.id === key.deviceId)
        if (device && (device.status === DeviceStatus.OFFLINE || device.status === DeviceStatus.FAULT)) {
          reject(new Error(OperationErrorCode.DEVICE_OFFLINE))
          return
        }

        // 3. 检查钥匙当前是否存在借出中借还记录
        const borrows = wx.getStorageSync(STORAGE_KEYS.BORROW_RECORDS) || []
        const activeBorrow = borrows.find(
          (b: any) =>
            b.keyId === params.keyId &&
            (b.status === BorrowRecordStatus.BORROWING ||
              b.status === BorrowRecordStatus.BORROWED ||
              b.status === BorrowRecordStatus.RETURNING),
        )
        if (activeBorrow && now < (activeBorrow.expectedReturnAt || now)) {
          // 当前若已被借出且还在借期内
          reject(new Error(OperationErrorCode.KEY_ALREADY_BORROWED))
          return
        }

        // 4. 检查用户本人是否已有该钥匙的活跃预约
        const userActive = this.reservations.find(
          r =>
            r.userId === params.userId &&
            r.keyId === params.keyId &&
            (r.status === ReservationStatus.ACTIVE || r.status === ReservationStatus.APPROVED),
        )
        if (userActive) {
          reject(new Error(OperationErrorCode.RESERVATION_CONFLICT))
          return
        }

        // 5. 检查时间冲突 (new.start < old.end && new.end > old.start)
        const hasConflict = this.reservations.some(r => {
          if (r.keyId !== params.keyId) return false
          if (r.status !== ReservationStatus.ACTIVE && r.status !== ReservationStatus.APPROVED) {
            return false
          }
          // 比较预约覆盖的完整周期 (从取钥窗口起到预计归还)
          const existingStart = r.pickupWindowStart || r.createdAt
          const existingEnd = r.expectedReturnAt || (existingStart + 7200000)
          return start < existingEnd && expectedReturn > existingStart
        })

        if (hasConflict) {
          reject(new Error(OperationErrorCode.RESERVATION_CONFLICT))
          return
        }

        // 6. 创建预约对象（自动确定 ACTIVE 或 APPROVED）
        const isCurrentlyActive = now >= (start - 300000) && now <= end
        const status = isCurrentlyActive ? ReservationStatus.ACTIVE : ReservationStatus.APPROVED

        const reservation: Reservation = {
          id: this.generateId(),
          userId: params.userId,
          keyId: params.keyId,
          status,
          purpose: params.purpose || '科研实验借用',
          createdAt: now,
          pickupWindowStart: start,
          pickupWindowEnd: end,
          expectedReturnAt: expectedReturn,
          approvedAt: now,
        }

        this.reservations.push(reservation)
        this.saveToStorage()

        // 7. 更新钥匙状态为 RESERVED（如果是当前活跃或钥匙原本可用）
        try {
          if (key && key.status === KeyStatus.AVAILABLE) {
            key.status = KeyStatus.RESERVED
            wx.setStorageSync(STORAGE_KEYS.KEYS, keys)
          }
        } catch (e) {
          console.error('更新钥匙状态失败', e)
        }

        resolve({ ...reservation })
      }, 150)
    })
  }

  async getUserReservations(userId: string): Promise<Reservation[]> {
    this.loadFromStorage()
    return new Promise(resolve => {
      setTimeout(() => {
        const results = this.reservations
          .filter(r => r.userId === userId)
          .sort((a, b) => b.createdAt - a.createdAt)
        resolve(results.map(r => ({ ...r })))
      }, 100)
    })
  }

  async getReservationById(id: string): Promise<Reservation | null> {
    this.loadFromStorage()
    return new Promise(resolve => {
      setTimeout(() => {
        const reservation = this.reservations.find(r => r.id === id)
        resolve(reservation ? { ...reservation } : null)
      }, 100)
    })
  }

  async cancelReservation(id: string): Promise<void> {
    this.loadFromStorage()
    return new Promise(resolve => {
      setTimeout(() => {
        const reservation = this.reservations.find(r => r.id === id)
        if (
          reservation &&
          (reservation.status === ReservationStatus.ACTIVE ||
            reservation.status === ReservationStatus.APPROVED ||
            reservation.status === ReservationStatus.PENDING)
        ) {
          reservation.status = ReservationStatus.CANCELLED
          reservation.cancelledAt = Date.now()
          this.saveToStorage()

          // 检查该钥匙是否还有其他活跃预约，若无则恢复 AVAILABLE
          try {
            const keys = wx.getStorageSync(STORAGE_KEYS.KEYS) || []
            const key = keys.find((k: any) => k.id === reservation.keyId)
            const otherActive = this.reservations.some(
              r =>
                r.keyId === reservation.keyId &&
                r.id !== id &&
                (r.status === ReservationStatus.ACTIVE || r.status === ReservationStatus.APPROVED),
            )
            if (key && !otherActive && key.status === KeyStatus.RESERVED) {
              key.status = KeyStatus.AVAILABLE
              wx.setStorageSync(STORAGE_KEYS.KEYS, keys)
            }
          } catch (e) {
            console.error('更新钥匙状态失败', e)
          }
        }
        resolve()
      }, 100)
    })
  }

  async canReserveKey(keyId: string, windowStart?: number, windowEnd?: number): Promise<boolean> {
    this.loadFromStorage()
    return new Promise(resolve => {
      setTimeout(() => {
        try {
          const keys = wx.getStorageSync(STORAGE_KEYS.KEYS) || []
          const key = keys.find((k: any) => k.id === keyId)
          if (!key || !key.enabled || key.status === KeyStatus.DISABLED || key.status === KeyStatus.MAINTENANCE) {
            resolve(false)
            return
          }

          const borrows = wx.getStorageSync(STORAGE_KEYS.BORROW_RECORDS) || []
          const activeBorrow = borrows.find(
            (b: any) =>
              b.keyId === keyId &&
              (b.status === BorrowRecordStatus.BORROWING ||
                b.status === BorrowRecordStatus.BORROWED ||
                b.status === BorrowRecordStatus.RETURNING),
          )
          if (activeBorrow) {
            resolve(false)
            return
          }

          const start = windowStart || Date.now()
          const end = windowEnd || (start + 7200000)

          const hasConflict = this.reservations.some(r => {
            if (r.keyId !== keyId) return false
            if (r.status !== ReservationStatus.ACTIVE && r.status !== ReservationStatus.APPROVED) {
              return false
            }
            const existingStart = r.pickupWindowStart || r.createdAt
            const existingEnd = r.expectedReturnAt || (existingStart + 7200000)
            return start < existingEnd && end > existingStart
          })

          resolve(!hasConflict)
        } catch (e) {
          console.error('检查钥匙状态失败', e)
          resolve(false)
        }
      }, 50)
    })
  }

  async getActiveReservation(
    userId: string,
    keyId?: string,
  ): Promise<Reservation | null> {
    this.loadFromStorage()
    return new Promise(resolve => {
      setTimeout(() => {
        const now = Date.now()
        const reservation = this.reservations.find(
          r =>
            r.userId === userId &&
            (!keyId || r.keyId === keyId) &&
            (r.status === ReservationStatus.ACTIVE ||
              (r.status === ReservationStatus.APPROVED && now >= (r.pickupWindowStart - 300000))),
        )
        resolve(reservation ? { ...reservation } : null)
      }, 50)
    })
  }

  async markReservationUsed(id: string): Promise<void> {
    this.loadFromStorage()
    return new Promise(resolve => {
      setTimeout(() => {
        const reservation = this.reservations.find(r => r.id === id)
        if (reservation) {
          reservation.status = ReservationStatus.USED
          reservation.usedAt = Date.now()
          this.saveToStorage()
        }
        resolve()
      }, 50)
    })
  }
}
