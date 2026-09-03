import {
  keyService,
  reservationService,
  userService,
} from '../../services/index'
import { Key } from '../../models/key'
import { CreateReservationParams } from '../../services/reservation/index'
import { OperationErrorCode } from '../../models/operation-error'

Page({
  data: {
    keyId: '',
    key: null as Key | null,
    purpose: '科研实验与教学研讨',
    duration: 2, // 默认2小时
    loading: true,
    submitting: false,
  },

  onLoad(options: any) {
    const keyId = options.keyId
    if (!keyId) {
      wx.showToast({ title: '参数错误', icon: 'none' })
      setTimeout(() => wx.navigateBack(), 1500)
      return
    }
    this.setData({ keyId })
    this.loadKeyInfo(keyId)
  },

  async loadKeyInfo(keyId: string) {
    try {
      this.setData({ loading: true })
      const key = await keyService.getKeyById(keyId)

      if (!key) {
        wx.showToast({ title: '钥匙不存在', icon: 'none' })
        setTimeout(() => wx.navigateBack(), 1500)
        return
      }

      this.setData({ key, loading: false })
    } catch (e) {
      console.error('加载钥匙信息失败', e)
      this.setData({ loading: false })
      wx.showToast({ title: '加载失败', icon: 'none' })
    }
  },

  onPurposeInput(e: any) {
    this.setData({ purpose: e.detail.value })
  },

  onDurationChange(e: any) {
    this.setData({ duration: parseInt(e.detail.value) })
  },

  async submitReservation() {
    const { keyId, purpose, duration, submitting } = this.data

    if (submitting) return

    if (!purpose.trim()) {
      wx.showToast({ title: '请填写用途', icon: 'none' })
      return
    }

    try {
      this.setData({ submitting: true })

      const user = await userService.getCurrentUser()
      if (!user) {
        wx.showToast({ title: '请先登录', icon: 'none' })
        this.setData({ submitting: false })
        return
      }

      const now = Date.now()
      const params: CreateReservationParams = {
        userId: user.id,
        keyId,
        purpose: purpose.trim(),
        pickupWindowStart: now,
        pickupWindowEnd: now + 1800000, // 30分钟取钥窗口
        expectedDuration: duration * 3600000,
        expectedReturnAt: now + duration * 3600000,
      }

      await reservationService.createReservation(params)

      wx.showToast({ title: '预约成功', icon: 'success' })

      setTimeout(() => {
        wx.switchTab({ url: '/pages/home/home' })
      }, 1500)
    } catch (e: any) {
      console.error('预约失败', e)
      let msg = '预约失败'
      if (e.message === OperationErrorCode.RESERVATION_CONFLICT) {
        msg = '所选时间段该钥匙已被他人预约'
      } else if (e.message === OperationErrorCode.KEY_ALREADY_BORROWED) {
        msg = '该钥匙当前已被借出'
      } else if (e.message === OperationErrorCode.KEY_NOT_AVAILABLE) {
        msg = '该钥匙当前不可预约'
      } else if (e.message === OperationErrorCode.DEVICE_OFFLINE) {
        msg = '所属钥匙柜离线，暂时无法预约'
      } else if (e.message) {
        msg = e.message
      }
      wx.showToast({ title: msg, icon: 'none', duration: 2500 })
      this.setData({ submitting: false })
    }
  },
})
