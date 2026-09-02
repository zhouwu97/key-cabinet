import { keyService, deviceService } from '../../services'
import { Key, KeyStatus } from '../../models/key'
import { KEY_STATUS_LABEL, KEY_STATUS_TONE } from '../../constants/labels'
import { DeviceStatus } from '../../models/device'

type FilterType = 'ALL' | 'AVAILABLE' | 'BORROWED'

export interface KeyViewModel extends Key {
  statusLabel: string
  statusTone: string
  deviceName: string
}

Page({
  data: {
    keys: [] as KeyViewModel[],
    filteredKeys: [] as KeyViewModel[],
    searchKeyword: '',
    activeFilter: 'ALL' as FilterType,
    availableCount: 0,
    borrowedCount: 0,
    loading: true,
    hasError: false,
    isCabinetOffline: false,
  },

  onLoad() {
    this.loadKeys()
  },

  onShow() {
    this.loadKeys()
  },

  async loadKeys() {
    try {
      this.setData({ loading: true, hasError: false })
      
      const [rawKeys, device] = await Promise.all([
        keyService.getKeys(),
        deviceService.getDeviceStatus('CAB001').catch(() => null),
      ])

      const isCabinetOffline = device ? device.status === DeviceStatus.OFFLINE : false

      const keys: KeyViewModel[] = rawKeys.map(k => ({
        ...k,
        deviceName: k.deviceId === 'CAB001' ? '1号钥匙柜 (信息楼)' : k.deviceId,
        statusLabel: KEY_STATUS_LABEL[k.status] || '未知',
        statusTone: KEY_STATUS_TONE[k.status] || 'gray',
      }))

      const availableCount = keys.filter(k => k.status === KeyStatus.AVAILABLE).length
      const borrowedCount = keys.filter(
        k => k.status === KeyStatus.BORROWED || k.status === KeyStatus.OVERDUE,
      ).length

      this.setData({
        keys,
        availableCount,
        borrowedCount,
        isCabinetOffline,
        loading: false,
        hasError: false,
      })
      this.applyFilter()
    } catch (e) {
      console.error('加载钥匙列表失败', e)
      this.setData({ loading: false, hasError: true })
    }
  },

  onSearchInput(e: any) {
    const keyword = e.detail.value
    this.setData({ searchKeyword: keyword })
    this.applyFilter()
  },

  clearSearch() {
    this.setData({ searchKeyword: '' })
    this.applyFilter()
  },

  onFilterTap(e: any) {
    const filter = e.currentTarget.dataset.filter as FilterType
    this.setData({ activeFilter: filter })
    this.applyFilter()
  },

  applyFilter() {
    const { keys, searchKeyword, activeFilter } = this.data
    let results = [...keys]

    // 搜索过滤
    if (searchKeyword.trim()) {
      const keyword = searchKeyword.toLowerCase().trim()
      results = results.filter(
        key =>
          key.roomNo.toLowerCase().includes(keyword) ||
          key.name.toLowerCase().includes(keyword) ||
          key.description?.toLowerCase().includes(keyword),
      )
    }

    // 状态过滤
    if (activeFilter === 'AVAILABLE') {
      results = results.filter(key => key.status === KeyStatus.AVAILABLE)
    } else if (activeFilter === 'BORROWED') {
      results = results.filter(
        key =>
          key.status === KeyStatus.BORROWED || key.status === KeyStatus.OVERDUE,
      )
    }

    this.setData({ filteredKeys: results })
  },

  onKeyCardTap(e: any) {
    const { keyId } = e.detail
    wx.navigateTo({ url: `/pages/key-detail/key-detail?keyId=${keyId}` })
  },
})
