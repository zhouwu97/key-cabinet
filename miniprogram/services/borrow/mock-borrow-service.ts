import { BorrowRecord, BorrowRecordStatus } from '../../models/borrow-record'
import { BorrowService } from './borrow-service'
import { MOCK_BORROW_RECORDS, STORAGE_KEYS } from '../../mocks/mock-data'
import { KeyStatus } from '../../models/key'
import { KeyPresenceState } from '../../models/key-presence'

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
    return `BR${Date.now()}${Math.random().toString(36).substring(2, 7).toUpperCase()}`
  }

  async getUserBorrowRecords(userId: string): Promise<BorrowRecord[]> {
    this.loadFromStorage()
    return new Promise(resolve => {
      setTimeout(() => {
        const results = this.borrowRecords
          .filter(r => r.userId === userId)
          .sort((a, b) => (b.borrowedAt || 0) - (a.borrowedAt || 0))
        resolve(results.map(r => ({ ...r })))
      }, 100)
    })
  }

  async getCurrentBorrows(userId: string): Promise<BorrowRecord[]> {
    this.loadFromStorage()
    return new Promise(resolve => {
      setTimeout(() => {
        const results = this.borrowRecords.filter(
          r =>
            r.userId === userId &&
            (r.status === BorrowRecordStatus.BORROWED ||
              r.status === BorrowRecordStatus.BORROWING ||
              r.status === BorrowRecordStatus.RETURNING),
        )
        resolve(results.map(r => ({ ...r })))
      }, 100)
    })
  }

  async getBorrowRecordById(id: string): Promise<BorrowRecord | null> {
    this.loadFromStorage()
    return new Promise(resolve => {
      setTimeout(() => {
        const record = this.borrowRecords.find(r => r.id === id)
        resolve(record ? { ...record } : null)
      }, 100)
    })
  }

  async getActiveBorrowByKey(keyId: string): Promise<BorrowRecord | null> {
    this.loadFromStorage()
    return new Promise(resolve => {
      setTimeout(() => {
        const record = this.borrowRecords.find(
          r =>
            r.keyId === keyId &&
            (r.status === BorrowRecordStatus.BORROWED ||
              r.status === BorrowRecordStatus.BORROWING ||
              r.status === BorrowRecordStatus.RETURNING),
        )
        resolve(record ? { ...record } : null)
      }, 50)
    })
  }

  async createBorrowRecord(
    userId: string,
    keyId: string,
    deviceId: string,
    slotId?: string,
    reservationId?: string,
    purpose?: string,
    expectedReturnAt?: number,
  ): Promise<BorrowRecord> {
    this.loadFromStorage()
    return new Promise(resolve => {
      setTimeout(() => {
        const now = Date.now()
        const defaultDuration = 7200000 // 2小时

        // 查找槽位
        let targetSlotId: string = slotId || ''
        if (!targetSlotId) {
          const keys = wx.getStorageSync(STORAGE_KEYS.KEYS) || []
          const keyObj = keys.find((k: any) => k.id === keyId)
          targetSlotId = keyObj?.slotId || 'SLOT01'
        }

        const record: BorrowRecord = {
          id: this.generateId(),
          userId,
          keyId,
          slotId: targetSlotId,
          deviceId,
          reservationId,
          purpose: purpose || '实验教学与研讨',
          status: BorrowRecordStatus.BORROWING,
          borrowedAt: now,
          expectedReturnAt: expectedReturnAt || (now + defaultDuration),
        }

        this.borrowRecords.unshift(record)
        this.saveToStorage()

        resolve({ ...record })
      }, 100)
    })
  }

  async updateBorrowStatus(id: string, status: BorrowRecordStatus, notes?: string): Promise<void> {
    this.loadFromStorage()
    return new Promise(resolve => {
      setTimeout(() => {
        const record = this.borrowRecords.find(r => r.id === id)
        if (record) {
          record.status = status
          if (notes) record.notes = notes
          if (status === BorrowRecordStatus.BORROWED && !record.borrowedAt) {
            record.borrowedAt = Date.now()
          }
          this.saveToStorage()
        }
        resolve()
      }, 50)
    })
  }

  async completeBorrowRecord(id: string): Promise<void> {
    this.loadFromStorage()
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

            // 更新槽位在位状态为 PRESENT
            const slots = wx.getStorageSync(STORAGE_KEYS.KEY_SLOTS) || []
            const slot = slots.find((s: any) => s.id === record.slotId || s.keyId === record.keyId)
            if (slot) {
              slot.presence = KeyPresenceState.PRESENT
              slot.lastUpdated = now
              wx.setStorageSync(STORAGE_KEYS.KEY_SLOTS, slots)
            }
          } catch (e) {
            console.error('更新钥匙及槽位状态失败', e)
          }
        }
        resolve()
      }, 100)
    })
  }

  async checkOverdue(): Promise<void> {
    this.loadFromStorage()
    return new Promise(resolve => {
      setTimeout(() => {
        const now = Date.now()
        let updated = false

        this.borrowRecords.forEach(record => {
          if (
            record.status === BorrowRecordStatus.BORROWED &&
            now > record.expectedReturnAt &&
            !record.overdueAt
          ) {
            record.overdueAt = now
            updated = true
          }
        })

        if (updated) {
          this.saveToStorage()
        }

        resolve()
      }, 50)
    })
  }
}
