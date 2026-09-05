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
import { ReservationStatus } from '../../models/reservation'
import { BorrowRecord } from '../../models/borrow-record'
import { formatTime } from '../../utils/date'
import { User } from '../../models/user'

interface ReservationViewModel {
  id: string
  keyId: string
  keyName: string
  roomNo: string
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

function isRecordOverdue(record: BorrowRecord): boolean {
  if (record.returnedAt) return false
  return Date.now() > record.expectedReturnAt
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

    // P1: 逾期借用
    overdueBorrows: [] as BorrowViewModel[],

    // P2: 待取钥预约
    activeReservations: [] as ReservationViewModel[],

    // P3: 正常借用中
    normalBorrows: [] as BorrowViewModel[],

    // P6: 设备状态
    deviceName: '1号钥匙柜',
    deviceOnline: true,
    availableSlotCount: 7,
    totalSlotCount: 10,
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
      if (isOpInProgress && activeOp) {
        activeOpKey = await keyService.getKeyById(activeOp.keyId)
      } else {
        this.setData({ activeOperation: null, activeOperationKey: null })
      }

      // 3. 检查设备状态 (P6)
      const device = await deviceService.getDeviceStatus('CAB001')
      const isOnline = device ? device.status === 'ONLINE' : true

      if (user) {
        const [reservations, borrows, allKeys] = await Promise.all([
          reservationService.getUserReservations(user.id),
          borrowService.getCurrentBorrows(user.id),
          keyService.getKeys(),
        ])

        const keyMap = new Map<string, Key>()
        allKeys.forEach(k => keyMap.set(k.id, k))

        // 筛选可取钥的预约 (ACTIVE / APPROVED)
        const activeReservations: ReservationViewModel[] = reservations
          .filter(
            r =>
              r.status === ReservationStatus.ACTIVE ||
              r.status === ReservationStatus.APPROVED,
          )
          .map(r => {
            const key = keyMap.get(r.keyId)
            return {
              ...r,
              keyName: key?.name || r.keyId,
              roomNo: key?.roomNo || '',
              pickupWindowStartText: formatTime(r.pickupWindowStart),
              pickupWindowEndText: formatTime(r.pickupWindowEnd),
              statusLabel: r.status === ReservationStatus.APPROVED ? '已审批通过' : '待现场取钥',
              statusTone: 'blue',
              canPickup: true,
              canCancel: true,
            }
          })

        // 处理借用记录（拆分逾期与正常借用）
        const overdueBorrows: BorrowViewModel[] = []
        const normalBorrows: BorrowViewModel[] = []

        borrows.forEach(b => {
          const key = keyMap.get(b.keyId)
          const overdue = isRecordOverdue(b)
          const vm: BorrowViewModel = {
            ...b,
            keyName: key?.name || b.keyId,
            roomNo: key?.roomNo || '',
            isOverdue: overdue,
            expectedReturnText: formatTime(b.expectedReturnAt),
            borrowedAtText: formatTime(b.borrowedAt),
            statusLabel: overdue ? '已逾期' : '借用中',
            statusTone: overdue ? 'red' : 'green',
            canReturn: true,
          }
          if (overdue) {
            overdueBorrows.push(vm)
          } else {
            normalBorrows.push(vm)
          }
        })

        const availableSlotCount = allKeys.filter(k => k.status === 'AVAILABLE').length
        const pendingTaskCount =
          overdueBorrows.length +
          activeReservations.length +
          normalBorrows.length +
          (isOpInProgress ? 1 : 0)

        this.setData({
          user,
          userName: user.name || '师生',
          greetingText,
          pendingTaskCount,
          deviceName: device?.name || '1号钥匙柜 (信息楼)',
          deviceOnline: isOnline,
          availableSlotCount: availableSlotCount || 7,
          totalSlotCount: allKeys.length || 10,
          activeOperation: isOpInProgress ? activeOp : null,
          activeOperationKey: activeOpKey,
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
          deviceName: device?.name || '1号钥匙柜 (信息楼)',
          deviceOnline: isOnline,
          activeOperation: null,
          activeOperationKey: null,
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
    // 智能选择首要任务进入核验
    if (this.data.overdueBorrows.length > 0) {
      const b = this.data.overdueBorrows[0]
      wx.navigateTo({
        url: `/pages/scan/scan?mode=RETURN&borrowRecordId=${b.id}&keyId=${b.keyId}&expectedDeviceId=${b.deviceId || 'CAB001'}`,
      })
    } else if (this.data.activeReservations.length > 0) {
      const r = this.data.activeReservations[0]
      wx.navigateTo({
        url: `/pages/scan/scan?mode=PICKUP&reservationId=${r.id}&keyId=${r.keyId}&expectedDeviceId=CAB001`,
      })
    } else if (this.data.normalBorrows.length > 0) {
      const b = this.data.normalBorrows[0]
      wx.navigateTo({
        url: `/pages/scan/scan?mode=RETURN&borrowRecordId=${b.id}&keyId=${b.keyId}&expectedDeviceId=${b.deviceId || 'CAB001'}`,
      })
    } else {
      wx.navigateTo({
        url: '/pages/scan/scan?mode=PICKUP&expectedDeviceId=CAB001',
      })
    }
  },

  // 取约卡片「现场取钥」 -> 路由至扫码页核验
  async onReservationCardPickup(e: any) {
    const { id: rsvId, keyId } = e.detail
    try {
      const key = await keyService.getKeyById(keyId)
      const deviceId = key?.deviceId || 'CAB001'
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
      const key = await keyService.getKeyById(keyId)
      const deviceId = key?.deviceId || 'CAB001'
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
