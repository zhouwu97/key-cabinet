import { keyService } from '../../services'
import { Key, KeyStatus } from '../../models/key'
import { KEY_STATUS_LABEL, KEY_STATUS_TONE } from '../../constants/labels'

type FilterType = 'ALL' | 'AVAILABLE' | 'BORROWED' | 'MY'

export interface KeyViewModel extends Key {
  statusLabel: string
  statusTone: string
}

Page({
  data: {
    keys: [] as KeyViewModel[],
    filteredKeys: [] as KeyViewModel[],
    searchKeyword: '',
    activeFilter: 'ALL' as FilterType,
    loading: true,
  },

  onLoad() {
    this.loadKeys()
  },

  onShow() {
    this.loadKeys()
  },

  async loadKeys() {
    try {
      this.setData({ loading: true })
      const rawKeys = await keyService.getKeys()
      const keys: KeyViewModel[] = rawKeys.map(k => ({
        ...k,
        statusLabel: KEY_STATUS_LABEL[k.status] || '未知',
        statusTone: KEY_STATUS_TONE[k.status] || 'gray',
      }))

      this.setData({
        keys,
        loading: false,
      })
      this.applyFilter()
    } catch (e) {
      console.error('加载钥匙列表失败', e)
      this.setData({ loading: false })
      wx.showToast({ title: '加载失败', icon: 'none' })
    }
  },

  onSearchInput(e: any) {
    const keyword = e.detail.value
    this.setData({ searchKeyword: keyword })
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
    } else if (activeFilter === 'MY') {
      results = results.filter(
        key =>
          key.status === KeyStatus.BORROWED || key.status === KeyStatus.OVERDUE,
      )
    }

    this.setData({ filteredKeys: results })
  },

  goDetail(e: any) {
    const keyId = e.currentTarget.dataset.id
    wx.navigateTo({ url: `/pages/key-detail/key-detail?keyId=${keyId}` })
  },
})
