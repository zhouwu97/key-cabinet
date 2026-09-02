import { Key, KeyStatus } from '../../models/key'
import { KeySlot } from '../../models/key-slot'
import { KeyService } from './key-service'
import {
  MOCK_KEYS,
  MOCK_KEY_SLOTS,
  STORAGE_KEYS,
} from '../../mocks/mock-data'

export class MockKeyService implements KeyService {
  private keys: Key[] = []
  private keySlots: KeySlot[] = []

  constructor() {
    this.loadFromStorage()
  }

  private loadFromStorage(): void {
    try {
      const storedKeys = wx.getStorageSync(STORAGE_KEYS.KEYS)
      const storedSlots = wx.getStorageSync(STORAGE_KEYS.KEY_SLOTS)

      this.keys = storedKeys && storedKeys.length > 0 ? storedKeys : [...MOCK_KEYS]
      this.keySlots = storedSlots && storedSlots.length > 0 ? storedSlots : [...MOCK_KEY_SLOTS]

      this.saveToStorage()
    } catch (e) {
      console.error('加载钥匙数据失败', e)
      this.keys = [...MOCK_KEYS]
      this.keySlots = [...MOCK_KEY_SLOTS]
    }
  }

  private saveToStorage(): void {
    try {
      wx.setStorageSync(STORAGE_KEYS.KEYS, this.keys)
      wx.setStorageSync(STORAGE_KEYS.KEY_SLOTS, this.keySlots)
    } catch (e) {
      console.error('保存钥匙数据失败', e)
    }
  }

  async getKeys(): Promise<Key[]> {
    this.loadFromStorage()
    return new Promise(resolve => {
      setTimeout(() => resolve([...this.keys]), 100)
    })
  }

  async getKeyById(keyId: string): Promise<Key | null> {
    this.loadFromStorage()
    return new Promise(resolve => {
      setTimeout(() => {
        const key = this.keys.find(k => k.id === keyId)
        resolve(key ? { ...key } : null)
      }, 100)
    })
  }

  async searchKeys(keyword: string): Promise<Key[]> {
    this.loadFromStorage()
    return new Promise(resolve => {
      setTimeout(() => {
        const lowerKeyword = keyword.toLowerCase().trim()
        if (!lowerKeyword) {
          resolve([...this.keys])
          return
        }

        const results = this.keys.filter(
          key =>
            key.roomNo.toLowerCase().includes(lowerKeyword) ||
            key.name.toLowerCase().includes(lowerKeyword) ||
            key.description?.toLowerCase().includes(lowerKeyword),
        )
        resolve(results)
      }, 100)
    })
  }

  async filterKeysByStatus(status: KeyStatus): Promise<Key[]> {
    this.loadFromStorage()
    return new Promise(resolve => {
      setTimeout(() => {
        const results = this.keys.filter(k => k.status === status)
        resolve(results)
      }, 100)
    })
  }

  async getKeySlot(slotIdOrKeyId: string): Promise<KeySlot | null> {
    this.loadFromStorage()
    return new Promise(resolve => {
      setTimeout(() => {
        const slot = this.keySlots.find(
          s => s.id === slotIdOrKeyId || s.keyId === slotIdOrKeyId,
        )
        resolve(slot ? { ...slot } : null)
      }, 100)
    })
  }

  async getDeviceSlots(deviceId: string): Promise<KeySlot[]> {
    this.loadFromStorage()
    return new Promise(resolve => {
      setTimeout(() => {
        const slots = this.keySlots.filter(s => s.deviceId === deviceId)
        resolve([...slots])
      }, 100)
    })
  }

  async updateKeyStatus(keyId: string, status: KeyStatus): Promise<void> {
    this.loadFromStorage()
    return new Promise(resolve => {
      setTimeout(() => {
        const key = this.keys.find(k => k.id === keyId)
        if (key) {
          key.status = status
          this.saveToStorage()
        }
        resolve()
      }, 50)
    })
  }
}
