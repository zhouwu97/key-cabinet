import { Key, KeyStatus } from '../../models/key'
import { KeySlot } from '../../models/key-slot'

export interface KeyService {
  /** 获取所有钥匙 */
  getKeys(): Promise<Key[]>

  /** 根据 ID 获取钥匙 */
  getKeyById(keyId: string): Promise<Key | null>

  /** 搜索钥匙（按房间号或名称） */
  searchKeys(keyword: string): Promise<Key[]>

  /** 按状态筛选钥匙 */
  filterKeysByStatus(status: KeyStatus): Promise<Key[]>

  /** 获取钥匙对应的物理槽位信息 */
  getKeySlot(slotIdOrKeyId: string): Promise<KeySlot | null>

  /** 获取指定设备下的所有槽位 */
  getDeviceSlots(deviceId: string): Promise<KeySlot[]>

  /** 更新钥匙业务状态 */
  updateKeyStatus(keyId: string, status: KeyStatus): Promise<void>
}
