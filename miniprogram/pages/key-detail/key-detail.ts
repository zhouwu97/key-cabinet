import { keyService, reservationService, userService } from '../../services/index'
import { Key } from '../../models/key'
import { KeySlot } from '../../models/key-slot'
import { KEY_STATUS_LABEL, KEY_STATUS_TONE, KEY_PRESENCE_LABEL } from '../../constants/labels'

Page({
  data: {
    keyId: '',
    key: null as Key | null,
    slot: null as KeySlot | null,
    deviceName: '1号钥匙柜 (信息楼一楼大厅)',
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
        this.setData({ key: null, loading: false })
        return
      }

      const slot = await keyService.getKeySlot(key.slotId || key.id).catch(() => null)
      const canReserve = await reservationService.canReserveKey(keyId).catch(() => false)
      const isAdminOrDev = user?.role === 'ADMIN'

      const statusLabel = KEY_STATUS_LABEL[key.status] || '未知'
      const statusTone = KEY_STATUS_TONE[key.status] || 'gray'
      const presenceLabel = slot ? (KEY_PRESENCE_LABEL[slot.presence] || '未知') : '离柜'
      const deviceName = key.deviceId === 'CAB001' ? '1号钥匙柜 (信息楼一楼大厅)' : key.deviceId

      this.setData({
        key,
        slot,
        deviceName,
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

  goBack() {
    wx.navigateBack()
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
