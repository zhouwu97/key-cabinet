import {
  deviceService,
  keyService,
  reservationService,
  borrowService,
  userService,
  operationService,
} from '../../services/index'
import { Key } from '../../models/key'
import { DeviceOperation, DeviceOperationStatus } from '../../models/device-operation'
import { canCancelReservation, canPickupReservation } from '../../models/reservation'
import { BorrowRecord, canReturnBorrow, isRecordOverdue } from '../../models/borrow-record'
import { formatDateTime, formatTime, formatPickupWindow } from '../../utils/date'
import { getBorrowRecordDisplayStatus, RESERVATION_STATUS_LABEL, RESERVATION_STATUS_TONE } from '../../constants/labels'
import { User } from '../../models/user'

interface ReservationViewModel {
  id: string
  keyId: string
  keyName: string
  roomNo: string
	deviceId: string
  pickupWindowStartText: string
  pickupWindowEndText: string
  statusLabel: string
  statusTone: string
  canPickup: boolean
  canCancel: boolean
}

interface BorrowViewModel extends BorrowRecord {
  keyName: string
  roomNo: string
  isOverdue: boolean
  expectedReturnText: string
  borrowedAtText: string
  statusLabel: string
  statusTone: string
  canReturn: boolean
}

function getGreeting(): string {
  const hour = new Date().getHours()
  if (hour < 6) return '凌晨好'
  if (hour < 12) return '早上好'
  if (hour < 14) return '中午好'
  if (hour < 19) return '下午好'
  return '晚上好'
}

function formatKeyDisplayName(key: Key | null | undefined, fallback: string): string {
  if (!key) return fallback
  const name = key.name || fallback
  const room = key.roomNo || ''
  if (!room || name.includes(room)) {
    return name
  }
  return `${room} 室 · ${name}`
}

Page({
  data: {
    user: null as User | null,
    userName: '同学',
    greetingText: '下午好',
    pendingTaskCount: 0,
    loading: true,

    // P0: 未完成中断操作
    activeOperation: null as DeviceOperation | null,
    activeOperationKey: null as Key | null,
    activeOperationKeyName: '',

    // P1: 逾期借用
    overdueBorrows: [] as BorrowViewModel[],

    // P2: 待取钥预约
    activeReservations: [] as ReservationViewModel[],

    // P3: 正常借用中
    normalBorrows: [] as BorrowViewModel[],

    // P6: 设备状态
	deviceName: '设备信息待获取',
	deviceLocation: '',
	deviceOnline: false,
	deviceStatsKnown: false,
	availableSlotCount: 0,
	totalSlotCount: 0,
  },

  onShow() {
    this.loadData()
  },

  async loadData() {
    try {
      this.setData({ loading: true })

      const greetingText = getGreeting()

      // 1. 获取当前用户
      const user = await userService.getCurrentUser()

      // 2. 检查未完成操作 (P0)
      const activeOp = await operationService.getActiveOperation()
      const isOpInProgress =
        activeOp &&
        [
          DeviceOperationStatus.CREATED,
          DeviceOperationStatus.AUTHORIZED,
          DeviceOperationStatus.SENT,
          DeviceOperationStatus.EXECUTING,
        ].includes(activeOp.status)

      let activeOpKey: Key | null = null
      let activeOperationKeyName = ''
      if (isOpInProgress && activeOp) {
        activeOpKey = await keyService.getKeyById(activeOp.keyId)
        activeOperationKeyName = formatKeyDisplayName(activeOpKey, '钥匙操作')
      } else {
        this.setData({ activeOperation: null, activeOperationKey: null, activeOperationKeyName: '' })
      }

      if (user) {
		const [reservations, borrows, allKeys, devices] = await Promise.all([
          reservationService.getUserReservations(user.id),
          borrowService.getCurrentBorrows(user.id),
          keyService.getKeys(),
			deviceService.listDevices(),
        ])

        const keyMap = new Map<string, Key>()
        allKeys.forEach(k => keyMap.set(k.id, k))

        // 待审批也保留在首页，避免提交后找不到当前预约。
        const activeReservations: ReservationViewModel[] = reservations
          .filter(canCancelReservation)
          .map(r => {
            const key = keyMap.get(r.keyId)
            return {
              ...r,
              keyName: formatKeyDisplayName(key, r.keyId),
              roomNo: key?.roomNo || '',
				deviceId: key?.deviceId || '',
              pickupWindowStartText: formatTime(r.pickupWindowStart),
              pickupWindowEndText: formatTime(r.pickupWindowEnd),
              pickupWindowText: formatPickupWindow(r.pickupWindowStart, r.pickupWindowEnd),
              expectedReturnText: formatDateTime(r.expectedReturnAt),
              statusLabel: RESERVATION_STATUS_LABEL[r.status],
              statusTone: RESERVATION_STATUS_TONE[r.status],
              canPickup: Boolean(user.identityVerified) && canPickupReservation(r),
              canCancel: canCancelReservation(r),
            }
          })

        // 处理借用记录（拆分逾期与正常借用）
        const overdueBorrows: BorrowViewModel[] = []
        const normalBorrows: BorrowViewModel[] = []

        borrows.forEach(b => {
          const key = keyMap.get(b.keyId)
          const overdue = isRecordOverdue(b)
          const status = getBorrowRecordDisplayStatus(b)
          const vm: BorrowViewModel = {
            ...b,
            keyName: formatKeyDisplayName(key, b.keyId),
            roomNo: key?.roomNo || '',
            isOverdue: overdue,
            expectedReturnText: formatDateTime(b.expectedReturnAt),
            borrowedAtText: b.borrowedAt ? formatDateTime(b.borrowedAt) : '',
            statusLabel: status.label,
            statusTone: status.tone,
            canReturn: canReturnBorrow(b),
          }
          if (overdue) {
            overdueBorrows.push(vm)
          } else {
            normalBorrows.push(vm)
          }
        })

        const pendingTaskCount =
          overdueBorrows.length +
          activeReservations.length +
          normalBorrows.length +
          (isOpInProgress ? 1 : 0)
		const preferredDeviceId =
			activeOp?.deviceId ||
			activeReservations[0]?.deviceId ||
			overdueBorrows[0]?.deviceId ||
			normalBorrows[0]?.deviceId ||
			devices[0]?.id ||
			''
		const device = devices.find(item => item.id === preferredDeviceId) || null
		const relatedKeys = preferredDeviceId
			? allKeys.filter(key => key.deviceId === preferredDeviceId)
			: []
		const availableSlotCount =
			device?.availableSlots ?? relatedKeys.filter(key => key.status === 'AVAILABLE').length
		const totalSlotCount = device?.totalSlots ?? relatedKeys.length

        this.setData({
          user,
          userName: user.name || '师生',
          greetingText,
          pendingTaskCount,
			deviceName: device?.name || '设备信息待获取',
			deviceLocation: device?.location || '',
			deviceOnline: device?.status === 'ONLINE',
			deviceStatsKnown: Boolean(device),
			availableSlotCount,
			totalSlotCount,
          activeOperation: isOpInProgress ? activeOp : null,
          activeOperationKey: activeOpKey,
          activeOperationKeyName,
          overdueBorrows,
          activeReservations,
          normalBorrows,
          loading: false,
        })
      } else {
        this.setData({
          user: null,
          userName: '访客',
          greetingText,
          pendingTaskCount: 0,
			deviceName: '设备信息待获取',
			deviceLocation: '',
			deviceOnline: false,
			deviceStatsKnown: false,
          activeOperation: null,
          activeOperationKey: null,
          activeOperationKeyName: '',
          overdueBorrows: [],
          activeReservations: [],
          normalBorrows: [],
          loading: false,
        })
      }
    } catch (e) {
      console.error('加载首页数据失败', e)
      this.setData({ loading: false })
    }
  },

  // 恢复未完成操作
  resumeOperation() {
    if (this.data.activeOperation) {
      wx.navigateTo({
        url: `/pages/operation/operation?operationId=${this.data.activeOperation.id}`,
      })
    }
  },

  // 首页主扫码行动入口
  onMainScanTap() {
    if (this.data.activeOperation) {
      this.resumeOperation()
      return
    }
    const overdue = this.data.overdueBorrows.find(b => b.canReturn)
    const reservation = this.data.activeReservations.find(r => r.canPickup)
    const borrow = this.data.normalBorrows.find(b => b.canReturn)
    if (overdue) {
      const b = overdue
      wx.navigateTo({
		url: `/pages/scan/scan?mode=RETURN&borrowRecordId=${b.id}&keyId=${b.keyId}&expectedDeviceId=${b.deviceId}`,
      })
    } else if (reservation) {
      const r = reservation
      wx.navigateTo({
		url: `/pages/scan/scan?mode=PICKUP&reservationId=${r.id}&keyId=${r.keyId}&expectedDeviceId=${r.deviceId}`,
      })
    } else if (borrow) {
      const b = borrow
      wx.navigateTo({
		url: `/pages/scan/scan?mode=RETURN&borrowRecordId=${b.id}&keyId=${b.keyId}&expectedDeviceId=${b.deviceId}`,
      })
    } else {
		wx.showToast({ title: '暂无可执行的借还任务', icon: 'none' })
    }
  },

  // 取约卡片「现场取钥」 -> 路由至扫码页核验
  async onReservationCardPickup(e: any) {
    const { id: rsvId, keyId } = e.detail
    try {
      const key = await keyService.getKeyById(keyId)
		const deviceId = key?.deviceId
		if (!deviceId) throw new Error('钥匙尚未绑定可用柜机')
      wx.navigateTo({
        url: `/pages/scan/scan?mode=PICKUP&reservationId=${rsvId}&keyId=${keyId}&expectedDeviceId=${deviceId}`,
      })
    } catch (err: any) {
      wx.showToast({ title: err.message || '进入扫码核验失败', icon: 'none' })
    }
  },

  // 取消预约
  async onReservationCardCancel(e: any) {
    const { id: rsvId } = e.detail
    wx.showModal({
      title: '取消预约',
      content: '确定要取消该笔钥匙预约吗？',
      success: async res => {
        if (res.confirm) {
          try {
            await reservationService.cancelReservation(rsvId)
            wx.showToast({ title: '已取消预约', icon: 'success' })
            this.loadData()
          } catch (err: any) {
            wx.showToast({ title: err.message || '取消失败', icon: 'none' })
          }
        }
      },
    })
  },

  // 借用卡片「归还」 -> 路由至扫码页核验
  async onBorrowCardReturn(e: any) {
    const { id: borrowId, keyId } = e.detail
    try {
      const borrow = [...this.data.normalBorrows, ...this.data.overdueBorrows].find(b => b.id === borrowId)
      if (!borrow || !canReturnBorrow(borrow)) return
      const deviceId = borrow.deviceId
		if (!deviceId) throw new Error('钥匙尚未绑定可用柜机')
      wx.navigateTo({
        url: `/pages/scan/scan?mode=RETURN&borrowRecordId=${borrowId}&keyId=${keyId}&expectedDeviceId=${deviceId}`,
      })
    } catch (err: any) {
      wx.showToast({ title: err.message || '进入扫码核验失败', icon: 'none' })
    }
  },

  goIdentityBind() {
    wx.navigateTo({ url: '/pages/identity-bind/identity-bind' })
  },

  goKeys() {
    wx.switchTab({ url: '/pages/keys/keys' })
  },

  goMyReservations() {
    try {
      wx.setStorageSync('kcab_records_initial_tab', 'CURRENT')
    } catch (e) {}
    wx.switchTab({ url: '/pages/records/records' })
  },

  goRecords() {
    wx.switchTab({ url: '/pages/records/records' })
  },

  goHelp() {
    wx.switchTab({ url: '/pages/profile/profile' })
  },
})
