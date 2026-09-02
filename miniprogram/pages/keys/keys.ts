import { keyService } from '../../services'
import { Key, KeyStatus } from '../../models/key'

type FilterType = 'ALL' | 'AVAILABLE' | 'BORROWED' | 'MY'

Page({
  data: {
    keys: [] as Key[],
    filteredKeys: [] as Key[],
    searchKeyword: '',
    activeFilter: 'ALL' as FilterType,
    loading: true,
  },

  onLoad() {
    this.loadKeys()
  },

  onShow() {
    // 每次显示时刷新数据
    this.loadKeys()
  },

  async loadKeys() {
    try {
      this.setData({ loading: true })
      const keys = await keyService.getKeys()
      this.setData({
        keys,
        filteredKeys: keys,
        loading: false,
      })
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
      // TODO: 过滤当前用户借用的钥匙
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

  getStatusLabel(status: KeyStatus): string {
    const labels: Record<KeyStatus, string> = {
      [KeyStatus.AVAILABLE]: '可借',
      [KeyStatus.RESERVED]: '已预约',
      [KeyStatus.BORROWED]: '借出',
      [KeyStatus.OVERDUE]: '逾期',
      [KeyStatus.MAINTENANCE]: '维护',
      [KeyStatus.DISABLED]: '停用',
    }
    return labels[status] || '未知'
  },

  getStatusTone(status: KeyStatus): string {
    const tones: Record<KeyStatus, string> = {
      [KeyStatus.AVAILABLE]: 'green',
      [KeyStatus.RESERVED]: 'blue',
      [KeyStatus.BORROWED]: 'orange',
      [KeyStatus.OVERDUE]: 'red',
      [KeyStatus.MAINTENANCE]: 'gray',
      [KeyStatus.DISABLED]: 'gray',
    }
    return tones[status] || 'gray'
  },
})
