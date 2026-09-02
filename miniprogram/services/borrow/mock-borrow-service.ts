import { BorrowRecord, BorrowRecordStatus } from '../../models/borrow-record'
import { BorrowService } from './borrow-service'
import { MOCK_BORROW_RECORDS, STORAGE_KEYS } from '../../mocks/mock-data'
import { KeyStatus } from '../../models/key'
import { KeyPhysicalState } from '../../models/key-physical-state'

export class MockBorrowService implements BorrowService {
  private borrowRecords: BorrowRecord[] = []

  constructor() {
    this.loadFromStorage()
  }

  private loadFromStorage(): void {
    try {
      const stored = wx.getStorageSync(STORAGE_KEYS.BORROW_RECORDS)
      this.borrowRecords =
        stored && stored.length > 0 ? stored : [...MOCK_BORROW_RECORDS]
      this.saveToStorage()
    } catch (e) {
      console.error('加载借还记录失败', e)
      this.borrowRecords = [...MOCK_BORROW_RECORDS]
    }
  }

  private saveToStorage(): void {
    try {
      wx.setStorageSync(STORAGE_KEYS.BORROW_RECORDS, this.borrowRecords)
    } catch (e) {
      console.error('保存借还记录失败', e)
    }
  }

  private generateId(): string {
    return `BR${Date.now()}${Math.random().toString(36).substring(2, 9)}`
  }

  async getUserBorrowRecords(userId: string): Promise<BorrowRecord[]> {
    return new Promise(resolve => {
      setTimeout(() => {
        const results = this.borrowRecords
          .filter(r => r.userId === userId)
          .sort((a, b) => (b.borrowedAt || 0) - (a.borrowedAt || 0))
        resolve([...results])
      }, 200)
    })
  }

  async getCurrentBorrows(userId: string): Promise<BorrowRecord[]> {
    return new Promise(resolve => {
      setTimeout(() => {
        const results = this.borrowRecords.filter(
          r =>
            r.userId === userId &&
            (r.status === BorrowRecordStatus.BORROWED ||
              r.status === BorrowRecordStatus.OVERDUE),
        )
        resolve([...results])
      }, 150)
    })
  }

  async getBorrowRecordById(id: string): Promise<BorrowRecord | null> {
    return new Promise(resolve => {
      setTimeout(() => {
        const record = this.borrowRecords.find(r => r.id === id)
        resolve(record ? { ...record } : null)
      }, 150)
    })
  }

  async createBorrowRecord(
    userId: string,
    keyId: string,
    deviceId: string,
    reservationId?: string,
  ): Promise<BorrowRecord> {
    return new Promise(resolve => {
      setTimeout(() => {
        const now = Date.now()
        const twoHours = 7200000

        const record: BorrowRecord = {
          id: this.generateId(),
          userId,
          keyId,
          deviceId,
          reservationId,
          status: BorrowRecordStatus.BORROWING,
          borrowedAt: now,
          expectedReturnAt: now + twoHours,
        }

        this.borrowRecords.push(record)
        this.saveToStorage()

        // 更新钥匙状态
        try {
          const keys = wx.getStorageSync(STORAGE_KEYS.KEYS) || []
          const key = keys.find((k: any) => k.id === keyId)
          if (key) {
            key.status = KeyStatus.BORROWED
            wx.setStorageSync(STORAGE_KEYS.KEYS, keys)
          }

          // 更新钥匙物理状态
          const locations = wx.getStorageSync(STORAGE_KEYS.KEY_LOCATIONS) || []
          const location = locations.find((loc: any) => loc.keyId === keyId)
          if (location) {
            location.physicalState = KeyPhysicalState.OUT
            location.lastUpdated = now
            wx.setStorageSync(STORAGE_KEYS.KEY_LOCATIONS, locations)
          }
        } catch (e) {
          console.error('更新钥匙状态失败', e)
        }

        resolve({ ...record })
      }, 300)
    })
  }

  async completeBorrowRecord(id: string): Promise<void> {
    return new Promise(resolve => {
      setTimeout(() => {
        const record = this.borrowRecords.find(r => r.id === id)
        if (record) {
          const now = Date.now()
          record.status = BorrowRecordStatus.COMPLETED
          record.returnedAt = now
          this.saveToStorage()

          // 更新钥匙状态回 AVAILABLE
          try {
            const keys = wx.getStorageSync(STORAGE_KEYS.KEYS) || []
            const key = keys.find((k: any) => k.id === record.keyId)
            if (key) {
              key.status = KeyStatus.AVAILABLE
              wx.setStorageSync(STORAGE_KEYS.KEYS, keys)
            }

            // 更新钥匙物理状态
            const locations = wx.getStorageSync(STORAGE_KEYS.KEY_LOCATIONS) || []
            const location = locations.find(
              (loc: any) => loc.keyId === record.keyId,
            )
            if (location) {
              location.physicalState = KeyPhysicalState.IN_CABINET
              location.lastUpdated = now
              wx.setStorageSync(STORAGE_KEYS.KEY_LOCATIONS, locations)
            }
          } catch (e) {
            console.error('更新钥匙状态失败', e)
          }
        }
        resolve()
      }, 200)
    })
  }

  async checkOverdue(): Promise<void> {
    return new Promise(resolve => {
      setTimeout(() => {
        const now = Date.now()
        let updated = false

        this.borrowRecords.forEach(record => {
          if (
            record.status === BorrowRecordStatus.BORROWED &&
            now > record.expectedReturnAt
          ) {
            record.status = BorrowRecordStatus.OVERDUE
            updated = true

            // 更新钥匙状态
            try {
              const keys = wx.getStorageSync(STORAGE_KEYS.KEYS) || []
              const key = keys.find((k: any) => k.id === record.keyId)
              if (key) {
                key.status = KeyStatus.OVERDUE
                wx.setStorageSync(STORAGE_KEYS.KEYS, keys)
              }
            } catch (e) {
              console.error('更新钥匙状态失败', e)
            }
          }
        })

        if (updated) {
          this.saveToStorage()
        }

        resolve()
      }, 100)
    })
  }
}
