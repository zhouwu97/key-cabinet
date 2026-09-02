import { Key, KeyStatus } from '../../models/key'
import { KeyLocation } from '../../models/key-physical-state'
import { KeyService } from './key-service'
import {
  MOCK_KEYS,
  MOCK_KEY_LOCATIONS,
  STORAGE_KEYS,
} from '../../mocks/mock-data'

export class MockKeyService implements KeyService {
  private keys: Key[] = []
  private keyLocations: KeyLocation[] = []

  constructor() {
    this.loadFromStorage()
  }

  private loadFromStorage(): void {
    try {
      const storedKeys = wx.getStorageSync(STORAGE_KEYS.KEYS)
      const storedLocations = wx.getStorageSync(STORAGE_KEYS.KEY_LOCATIONS)

      this.keys = storedKeys && storedKeys.length > 0 ? storedKeys : [...MOCK_KEYS]
      this.keyLocations =
        storedLocations && storedLocations.length > 0
          ? storedLocations
          : [...MOCK_KEY_LOCATIONS]

      this.saveToStorage()
    } catch (e) {
      console.error('加载钥匙数据失败', e)
      this.keys = [...MOCK_KEYS]
      this.keyLocations = [...MOCK_KEY_LOCATIONS]
    }
  }

  private saveToStorage(): void {
    try {
      wx.setStorageSync(STORAGE_KEYS.KEYS, this.keys)
      wx.setStorageSync(STORAGE_KEYS.KEY_LOCATIONS, this.keyLocations)
    } catch (e) {
      console.error('保存钥匙数据失败', e)
    }
  }

  async getKeys(): Promise<Key[]> {
    return new Promise(resolve => {
      setTimeout(() => resolve([...this.keys]), 200)
    })
  }

  async getKeyById(keyId: string): Promise<Key | null> {
    return new Promise(resolve => {
      setTimeout(() => {
        const key = this.keys.find(k => k.id === keyId)
        resolve(key ? { ...key } : null)
      }, 150)
    })
  }

  async searchKeys(keyword: string): Promise<Key[]> {
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
      }, 200)
    })
  }

  async filterKeysByStatus(status: KeyStatus): Promise<Key[]> {
    return new Promise(resolve => {
      setTimeout(() => {
        const results = this.keys.filter(k => k.status === status)
        resolve(results)
      }, 150)
    })
  }

  async getKeyLocation(keyId: string): Promise<KeyLocation | null> {
    return new Promise(resolve => {
      setTimeout(() => {
        const location = this.keyLocations.find(loc => loc.keyId === keyId)
        resolve(location ? { ...location } : null)
      }, 100)
    })
  }

  async updateKeyStatus(keyId: string, status: KeyStatus): Promise<void> {
    return new Promise(resolve => {
      setTimeout(() => {
        const key = this.keys.find(k => k.id === keyId)
        if (key) {
          key.status = status
          this.saveToStorage()
        }
        resolve()
      }, 100)
    })
  }
}
