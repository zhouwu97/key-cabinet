import {
  keyService,
  reservationService,
  userService,
} from '../../services/index'
import { Key } from '../../models/key'
import { CreateReservationParams } from '../../services/reservation/index'
import { OperationErrorCode } from '../../models/operation-error'
import { formatTime } from '../../utils/date'

Page({
  data: {
    keyId: '',
    key: null as Key | null,
    purposeTags: ['实验教学', '设备调试', '会议/答辩', '自习开发'],
    selectedTag: '实验教学',
    purpose: '实验教学：课程专项实验上机使用',
    pickupWindowText: '',
    returnDateText: '今天',
    returnTime: '18:00',
    agreedRules: true,
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

    const now = Date.now()
    const pickupStart = formatTime(now)
    const pickupEnd = formatTime(now + 1800000)

    // 默认预计归还时间为当前时间后 3 小时
    const defaultReturn = new Date(now + 3 * 3600000)
    const defHour = defaultReturn.getHours().toString().padStart(2, '0')
    const defMin = defaultReturn.getMinutes().toString().padStart(2, '0')

    this.setData({
      keyId,
      pickupWindowText: `${pickupStart} - ${pickupEnd}`,
      returnTime: `${defHour}:${defMin}`,
    })

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

  onSelectTag(e: any) {
    const tag = e.currentTarget.dataset.tag
    let text = this.data.purpose
    if (tag === '实验教学') text = '实验教学：课程专项实验上机使用'
    else if (tag === '设备调试') text = '设备调试：硬件与网络调试运维'
    else if (tag === '会议/答辩') text = '会议/答辩：科研研讨与论文答辩'
    else if (tag === '自习开发') text = '自习开发：自主创新训练与项目开发'

    this.setData({
      selectedTag: tag,
      purpose: text,
    })
  },

  onPurposeInput(e: any) {
    this.setData({ purpose: e.detail.value })
  },

  onReturnTimeChange(e: any) {
    this.setData({ returnTime: e.detail.value })
  },

  toggleRulesAgree() {
    this.setData({ agreedRules: !this.data.agreedRules })
  },

  async submitReservation() {
    const { keyId, purpose, returnTime, agreedRules, submitting } = this.data

    if (submitting) return

    if (!agreedRules) {
      wx.showToast({ title: '请先阅读并同意借用规则', icon: 'none' })
      return
    }

    if (!purpose.trim()) {
      wx.showToast({ title: '请填写钥匙用途', icon: 'none' })
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
      const [hStr, mStr] = returnTime.split(':')
      const targetDate = new Date()
      targetDate.setHours(parseInt(hStr, 10), parseInt(mStr, 10), 0, 0)

      let expectedReturnAt = targetDate.getTime()
      if (expectedReturnAt <= now) {
        // 如果选择的时间小于当前时间，推迟到第二天
        expectedReturnAt += 24 * 3600000
      }

      const params: CreateReservationParams = {
        userId: user.id,
        keyId,
        purpose: purpose.trim(),
        pickupWindowStart: now,
        pickupWindowEnd: now + 1800000, // 30分钟取钥窗口
        expectedDuration: expectedReturnAt - now,
        expectedReturnAt,
      }

      await reservationService.createReservation(params)

      wx.showToast({ title: '预约成功', icon: 'success' })

      setTimeout(() => {
        wx.switchTab({ url: '/pages/home/home' })
      }, 1200)
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
