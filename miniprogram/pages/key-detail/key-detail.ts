import { keyService, reservationService } from '../../services'
import { Key, KeyStatus } from '../../models/key'
import { KeyLocation, KeyPhysicalState } from '../../models/key-physical-state'

Page({
  data: {
    keyId: '',
    key: null as Key | null,
    location: null as KeyLocation | null,
    loading: true,
    canReserve: false,
  },

  onLoad(options: any) {
    const keyId = options.keyId
    if (!keyId) {
      wx.showToast({ title: '参数错误', icon: 'none' })
      setTimeout(() => wx.navigateBack(), 1500)
      return
    }
    this.setData({ keyId })
    this.loadKeyDetail(keyId)
  },

  async loadKeyDetail(keyId: string) {
    try {
      this.setData({ loading: true })
      const [key, location] = await Promise.all([
        keyService.getKeyById(keyId),
        keyService.getKeyLocation(keyId),
      ])

      if (!key) {
        wx.showToast({ title: '钥匙不存在', icon: 'none' })
        setTimeout(() => wx.navigateBack(), 1500)
        return
      }

      const canReserve = await reservationService.canReserveKey(keyId)

      this.setData({
        key,
        location,
        canReserve,
        loading: false,
      })
    } catch (e) {
      console.error('加载钥匙详情失败', e)
      this.setData({ loading: false })
      wx.showToast({ title: '加载失败', icon: 'none' })
    }
  },

  async goReserve() {
    const { key, canReserve } = this.data

    if (!key) return

    if (!canReserve) {
      wx.showToast({ title: '该钥匙当前不可预约', icon: 'none' })
      return
    }

    // 跳转到预约页面
    wx.navigateTo({
      url: `/pages/reservation-create/reservation-create?keyId=${key.id}`,
    })
  },

  getStatusLabel(status: KeyStatus): string {
    const labels: Record<KeyStatus, string> = {
      [KeyStatus.AVAILABLE]: '可借用',
      [KeyStatus.RESERVED]: '已预约',
      [KeyStatus.BORROWED]: '借出中',
      [KeyStatus.OVERDUE]: '逾期未还',
      [KeyStatus.MAINTENANCE]: '维护中',
      [KeyStatus.DISABLED]: '已停用',
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

  getPhysicalStateLabel(state: KeyPhysicalState): string {
    const labels: Record<KeyPhysicalState, string> = {
      [KeyPhysicalState.IN_CABINET]: '在柜中',
      [KeyPhysicalState.MOVING]: '移动中',
      [KeyPhysicalState.AT_PICKUP]: '取钥口',
      [KeyPhysicalState.OUT]: '已取出',
      [KeyPhysicalState.RETURN_CHECK]: '归还检查',
      [KeyPhysicalState.FAULT]: '故障',
      [KeyPhysicalState.UNKNOWN]: '未知',
    }
    return labels[state] || '未知'
  },
})
