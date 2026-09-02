import { Key, KeyStatus } from '../../models/key'
import { KeyLocation } from '../../models/key-physical-state'

export interface KeyService {
  /** 获取所有钥匙 */
  getKeys(): Promise<Key[]>

  /** 根据 ID 获取钥匙 */
  getKeyById(keyId: string): Promise<Key | null>

  /** 搜索钥匙（按房间号或名称） */
  searchKeys(keyword: string): Promise<Key[]>

  /** 按状态筛选钥匙 */
  filterKeysByStatus(status: KeyStatus): Promise<Key[]>

  /** 获取钥匙物理位置 */
  getKeyLocation(keyId: string): Promise<KeyLocation | null>

  /** 更新钥匙状态 */
  updateKeyStatus(keyId: string, status: KeyStatus): Promise<void>
}
