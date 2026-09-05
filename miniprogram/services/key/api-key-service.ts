import { httpClient } from '../../api/http-client'
import { toQueryString, toTimestamp } from '../../api/serializers'
import { Key, KeyStatus } from '../../models/key'
import { KeySlot } from '../../models/key-slot'
import { KeyService } from './key-service'

function normalizeKey(data: Key): Key {
  return { ...data }
}

function normalizeSlot(data: KeySlot & { lastUpdatedAt?: string | number }): KeySlot {
  return {
    ...data,
    lastUpdated: toTimestamp(data.lastUpdatedAt ?? data.lastUpdated),
  }
}

/** 真实后端钥匙服务；API 模式下不读取或写入 Mock 存储。 */
export class ApiKeyService implements KeyService {
  async getKeys(): Promise<Key[]> {
    const keys = await httpClient.request<Key[]>({ url: '/api/v1/keys' })
    return keys.map(normalizeKey)
  }

  async getKeyById(keyId: string): Promise<Key | null> {
    try {
      const key = await httpClient.request<Key>({
        url: `/api/v1/keys/${encodeURIComponent(keyId)}`,
      })
      return normalizeKey(key)
    } catch (error) {
      console.error(`Failed to get key ${keyId}:`, error)
      return null
    }
  }

  async searchKeys(keyword: string): Promise<Key[]> {
    const keys = await httpClient.request<Key[]>({
      url: `/api/v1/keys${toQueryString({ keyword })}`,
    })
    return keys.map(normalizeKey)
  }

  async filterKeysByStatus(status: KeyStatus): Promise<Key[]> {
    const keys = await httpClient.request<Key[]>({
      url: `/api/v1/keys${toQueryString({ status })}`,
    })
    return keys.map(normalizeKey)
  }

  async getKeySlot(slotIdOrKeyId: string): Promise<KeySlot | null> {
    try {
      const slot = await httpClient.request<KeySlot & { lastUpdatedAt?: string | number }>({
        url: `/api/v1/keys/${encodeURIComponent(slotIdOrKeyId)}/slot`,
      })
      return normalizeSlot(slot)
    } catch (error) {
      console.error(`Failed to get slot for ${slotIdOrKeyId}:`, error)
      return null
    }
  }

  async getDeviceSlots(deviceId: string): Promise<KeySlot[]> {
    const slots = await httpClient.request<Array<KeySlot & { lastUpdatedAt?: string | number }>>({
      url: `/api/v1/devices/${encodeURIComponent(deviceId)}/slots`,
    })
    return slots.map(normalizeSlot)
  }

  async updateKeyStatus(keyId: string, status: KeyStatus): Promise<void> {
    await httpClient.request<void>({
      url: `/api/v1/keys/${encodeURIComponent(keyId)}/status`,
      method: 'PATCH',
      data: { status },
    })
  }
}
