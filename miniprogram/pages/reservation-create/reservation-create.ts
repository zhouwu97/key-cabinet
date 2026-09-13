import {
  keyService,
  reservationService,
  userService,
	deviceService,
} from '../../services/index'
import { Key } from '../../models/key'
import { CreateReservationParams } from '../../services/reservation/index'
import { OperationErrorCode } from '../../models/operation-error'
import { formatDate, formatDateTime, formatPickupWindow, parseLocalDateTime } from '../../utils/date'
import { Reservation, ReservationStatus } from '../../models/reservation'
import { ApiException } from '../../api/http-client'
import { currentConfig } from '../../config/index'

Page({
  data: {
    keyId: '',
    key: null as Key | null,
	deviceName: '设备信息待获取',
	deviceLocation: '位置未提供',
    purposeTags: ['实验教学', '设备调试', '会议/答辩', '自习开发'],
    selectedTag: '实验教学',
    purpose: '实验教学：课程专项实验上机使用',
    returnDate: '',
    minReturnDate: '',
    returnTime: '18:00',
    agreedRules: true,
    loading: true,
    submitting: false,
    loadError: '',
    createdReservation: null as Reservation | null,
    createdPickupWindowText: '',
    createdReturnText: '',
  },

  onLoad(options: any) {
    const keyId = options.keyId
    if (!keyId) {
      wx.showToast({ title: '参数错误', icon: 'none' })
      setTimeout(() => wx.navigateBack(), 1500)
      return
    }

    const now = Date.now()

    // 默认预计归还时间为当前时间后 3 小时
    const defaultReturn = new Date(now + 3 * 3600000)
    const defHour = defaultReturn.getHours().toString().padStart(2, '0')
    const defMin = defaultReturn.getMinutes().toString().padStart(2, '0')

    this.setData({
      keyId,
      returnDate: formatDate(defaultReturn.getTime()),
      minReturnDate: formatDate(now),
      returnTime: `${defHour}:${defMin}`,
    })

    this.loadKeyInfo(keyId)
  },

  async loadKeyInfo(keyId: string) {
    try {
      this.setData({ loading: true, loadError: '' })
      const key = await keyService.getKeyById(keyId)

      if (!key) {
        this.setData({ key: null, loading: false, loadError: '钥匙不存在或已下架，请返回重新选择' })
        return
      }

		const device = key.deviceId
			? await deviceService.getDeviceStatus(key.deviceId).catch(() => null)
			: null
		const keyDisplayName = (!key.roomNo || key.name.includes(key.roomNo)) ? key.name : `${key.roomNo}室 · ${key.name}`
		this.setData({
			key,
			keyDisplayName,
			deviceName: device?.name || key.deviceId || '未绑定柜机',
			deviceLocation: device?.location || '位置未提供',
			loading: false,
		})
    } catch (e) {
      console.error('加载钥匙信息失败', e)
      this.setData({ loading: false, key: null, loadError: '无法加载钥匙信息，请检查网络后重试' })
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

  onReturnDateChange(e: any) {
    this.setData({ returnDate: e.detail.value })
  },

  retryLoad() {
    this.loadKeyInfo(this.data.keyId)
  },

  goRecords() {
    wx.setStorageSync('kcab_records_initial_tab', 'CURRENT')
    wx.switchTab({ url: '/pages/records/records' })
  },

  toggleRulesAgree() {
    this.setData({ agreedRules: !this.data.agreedRules })
  },

  async submitReservation() {
    const { keyId, key, purpose, returnDate, returnTime, agreedRules, submitting } = this.data

    if (submitting || this.data.createdReservation || !key || this.data.loading || this.data.loadError) return

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
		if (!user.identityVerified) {
			wx.showToast({
				title: user.profileCompleted ? '身份资料尚待管理员核验' : '请先完善身份资料',
				icon: 'none',
			})
			this.setData({ submitting: false })
			return
		}

      const now = Date.now()
      const expectedReturnAt = parseLocalDateTime(returnDate, returnTime)
      // 与服务端一致：归还时间必须晚于完整的 30 分钟取钥窗口。
      if (!Number.isFinite(expectedReturnAt) || expectedReturnAt <= now + 1800000) {
        wx.showToast({ title: '归还时间须晚于取钥窗口结束时间，请调整日期或时间', icon: 'none', duration: 3000 })
        this.setData({ submitting: false, minReturnDate: formatDate(now) })
        return
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

      const reservation = await reservationService.createReservation(params)
      this.setData({
        createdReservation: reservation,
        createdPickupWindowText: formatPickupWindow(reservation.pickupWindowStart, reservation.pickupWindowEnd),
        createdReturnText: formatDateTime(reservation.expectedReturnAt),
        submitting: false,
      })

      // 引导用户授权微信归还与逾期服务通知
      const tmplIds = [...new Set(Object.values(currentConfig.subscriptionTemplates))]
        .filter(id => id && !id.startsWith('kcab_tmpl_'))
      if (tmplIds.length > 0 && typeof wx.requestSubscribeMessage === 'function') {
        try {
          await new Promise<void>((resolve) => {
            wx.requestSubscribeMessage({
              tmplIds,
              success: (res) => {
                console.log('微信服务通知订阅成功:', res)
                resolve()
              },
              fail: (err) => {
                console.warn('微信服务通知订阅未授权或跳过:', err)
                resolve()
              },
            })
          })
        } catch {
          // 忽略订阅拒绝，不阻塞核心业务闭环
        }
      }

      wx.showToast({ title: reservation.status === ReservationStatus.PENDING ? '已提交，待审批' : '预约成功', icon: 'success' })
    } catch (e: any) {
      console.error('预约失败', e)
      let msg = '预约失败'
		const errorCode = e instanceof ApiException ? e.errorCode : e.message
		if (errorCode === OperationErrorCode.RESERVATION_CONFLICT) {
        msg = '所选时间段该钥匙已被他人预约'
		} else if (errorCode === OperationErrorCode.KEY_ALREADY_BORROWED) {
        msg = '该钥匙当前已被借出'
		} else if (errorCode === OperationErrorCode.KEY_NOT_AVAILABLE) {
        msg = '该钥匙当前不可预约'
		} else if (errorCode === OperationErrorCode.DEVICE_OFFLINE) {
        msg = '所属钥匙柜离线，暂时无法预约'
      } else if (e.message) {
        msg = e.message
      }
      wx.showToast({ title: msg, icon: 'none', duration: 2500 })
      this.setData({ submitting: false })
    }
  },
})
