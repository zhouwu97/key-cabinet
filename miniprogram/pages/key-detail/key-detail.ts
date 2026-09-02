import { keyService, reservationService, userService } from '../../services'
import { Key } from '../../models/key'
import { KeySlot } from '../../models/key-slot'
import { KEY_STATUS_LABEL, KEY_STATUS_TONE, KEY_PRESENCE_LABEL } from '../../constants/labels'

Page({
  data: {
    keyId: '',
    key: null as Key | null,
    slot: null as KeySlot | null,
    statusLabel: '',
    statusTone: 'gray',
    presenceLabel: '未知',
    loading: true,
    canReserve: false,
    isAdminOrDev: false,
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
      const [key, user] = await Promise.all([
        keyService.getKeyById(keyId),
        userService.getCurrentUser(),
      ])

      if (!key) {
        wx.showToast({ title: '钥匙不存在', icon: 'none' })
        setTimeout(() => wx.navigateBack(), 1500)
        return
      }

      const slot = await keyService.getKeySlot(key.slotId || key.id)
      const canReserve = await reservationService.canReserveKey(keyId)
      const isAdminOrDev = user?.role === 'ADMIN'

      const statusLabel = KEY_STATUS_LABEL[key.status] || '未知'
      const statusTone = KEY_STATUS_TONE[key.status] || 'gray'
      const presenceLabel = slot ? (KEY_PRESENCE_LABEL[slot.presence] || '未知') : '未知'

      this.setData({
        key,
        slot,
        statusLabel,
        statusTone,
        presenceLabel,
        canReserve,
        isAdminOrDev,
        loading: false,
      })
    } catch (e) {
      console.error('加载钥匙详情失败', e)
      this.setData({ loading: false })
      wx.showToast({ title: '加载失败', icon: 'none' })
    }
  },

  toggleDevMode() {
    this.setData({ isAdminOrDev: !this.data.isAdminOrDev })
  },

  async goReserve() {
    const { key, canReserve } = this.data
    if (!key) return

    if (!canReserve) {
      wx.showToast({ title: '该钥匙当前不可预约', icon: 'none' })
      return
    }

    wx.navigateTo({
      url: `/pages/reservation-create/reservation-create?keyId=${key.id}`,
    })
  },
})
