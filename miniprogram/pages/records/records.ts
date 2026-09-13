import {
  reservationService,
  borrowService,
  userService,
  keyService,
  operationService,
} from '../../services/index'
import { Reservation, canCancelReservation, canPickupReservation } from '../../models/reservation'
import { BorrowRecord, BorrowRecordStatus, isRecordOverdue, canReturnBorrow } from '../../models/borrow-record'
import { DeviceOperation } from '../../models/device-operation'
import { Key } from '../../models/key'
import {
  RESERVATION_STATUS_LABEL,
  RESERVATION_STATUS_TONE,
  getBorrowRecordDisplayStatus,
} from '../../constants/labels'
import { formatDateTime, formatPickupWindow } from '../../utils/date'

type TabType = 'CURRENT' | 'HISTORY' | 'EXCEPTION'

export interface ReservationViewModel extends Reservation {
  keyName: string
  pickupWindowText: string
  expectedReturnText: string
  statusLabel: string
  statusTone: string
  canPickup: boolean
  canCancel: boolean
}

export interface BorrowViewModel extends BorrowRecord {
  isOverdue: boolean
  keyName: string
  borrowedAtText: string
  expectedReturnText: string
  returnedAtText: string
  statusLabel: string
  statusTone: string
  canReturn: boolean
}

Page({
  data: {
    activeTab: 'CURRENT' as TabType,
    activeReservations: [] as ReservationViewModel[],
    currentBorrows: [] as BorrowViewModel[],
    currentTotalCount: 0,
    historyBorrows: [] as BorrowViewModel[],
    historyReservations: [] as ReservationViewModel[],
    historyTotalCount: 0,
    exceptionBorrows: [] as BorrowViewModel[],
    activeOperation: null as DeviceOperation | null,
    keyMap: {} as Record<string, Key>,
    loading: true,
    loadError: '',
    loggedIn: false,
    cancellingId: '',
  },

  _loadVersion: 0,

  onShow() {
    this.checkInitialTab()
    this.loadData()
  },

  async onPullDownRefresh() {
    try {
      await this.loadData()
    } finally {
      wx.stopPullDownRefresh()
    }
  },

  checkInitialTab() {
    try {
      let initialTab = wx.getStorageSync('kcab_records_initial_tab')
      if (initialTab) {
        if (initialTab === 'RESERVATIONS') initialTab = 'CURRENT'
        if (['CURRENT', 'HISTORY', 'EXCEPTION'].includes(initialTab)) {
          this.setData({ activeTab: initialTab as TabType })
        }
        wx.removeStorageSync('kcab_records_initial_tab')
      }
    } catch (e) {}
  },

  async loadData() {
    const version = ++this._loadVersion
    try {
      this.setData({ loading: true, loadError: '' })

      const user = await userService.getCurrentUser()
      if (version !== this._loadVersion) return
      if (!user) {
        this.setData({
          loading: false, loggedIn: false, activeReservations: [], currentBorrows: [],
          historyBorrows: [], historyReservations: [], exceptionBorrows: [],
          currentTotalCount: 0, historyTotalCount: 0, keyMap: {}, activeOperation: null,
        })
        return
      }

      const [rawReservations, rawBorrowRecords, keys, activeOperation] = await Promise.all([
        reservationService.getUserReservations(user.id),
        borrowService.getUserBorrowRecords(user.id),
        keyService.getKeys(),
        operationService.getActiveOperation(),
      ])
      if (version !== this._loadVersion) return

      const keyMap: Record<string, Key> = {}
      keys.forEach(key => {
        keyMap[key.id] = key
      })

      const getKeyName = (keyId: string) => {
        const key = keyMap[keyId]
        if (!key) return keyId
        if (!key.roomNo || key.name.includes(key.roomNo)) {
          return key.name
        }
        return `${key.roomNo}室 · ${key.name}`
      }

      const now = Date.now()
      const reservations: ReservationViewModel[] = rawReservations
        .sort((a, b) => b.createdAt - a.createdAt)
        .map(r => ({
          ...r,
          keyName: getKeyName(r.keyId),
          pickupWindowText: formatPickupWindow(r.pickupWindowStart, r.pickupWindowEnd),
          expectedReturnText: formatDateTime(r.expectedReturnAt),
          statusLabel: RESERVATION_STATUS_LABEL[r.status] || '待取钥',
          statusTone: RESERVATION_STATUS_TONE[r.status] || 'blue',
          canPickup: Boolean(user.identityVerified) && canPickupReservation(r, now),
          canCancel: canCancelReservation(r),
        }))
      const activeReservations = reservations.filter(canCancelReservation)
      const historyReservations = reservations.filter(r => !canCancelReservation(r))

      // 格式化借还 ViewModel
      const mapBorrowViewModel = (b: BorrowRecord): BorrowViewModel => {
        const statusInfo = getBorrowRecordDisplayStatus(b)
        const canReturn = canReturnBorrow(b)

        return {
          ...b,
          keyName: getKeyName(b.keyId),
          borrowedAtText: formatDateTime(b.borrowedAt),
          expectedReturnText: formatDateTime(b.expectedReturnAt),
          returnedAtText: b.returnedAt ? formatDateTime(b.returnedAt) : '',
          isOverdue: isRecordOverdue(b),
          statusLabel: statusInfo.label,
          statusTone: statusInfo.tone,
          canReturn,
        }
      }

      // 借用中与归还中
      const currentBorrows: BorrowViewModel[] = rawBorrowRecords
        .filter(
          r =>
            r.status === BorrowRecordStatus.BORROWED ||
            r.status === BorrowRecordStatus.BORROWING ||
            r.status === BorrowRecordStatus.RETURNING,
        )
        .map(mapBorrowViewModel)

      // 历史完成
      const historyBorrows: BorrowViewModel[] = rawBorrowRecords
        .filter(r => r.status === BorrowRecordStatus.COMPLETED)
        .map(mapBorrowViewModel)

      // 异常与逾期
      const exceptionBorrows: BorrowViewModel[] = rawBorrowRecords
        .filter(r => isRecordOverdue(r) || r.status === BorrowRecordStatus.EXCEPTION)
        .map(mapBorrowViewModel)

      this.setData({
        activeReservations,
        currentBorrows,
        currentTotalCount: activeReservations.length + currentBorrows.length,
        historyBorrows,
        historyReservations,
        historyTotalCount: historyReservations.length + historyBorrows.length,
        exceptionBorrows,
        activeOperation,
        loggedIn: true,
        keyMap,
        loading: false,
      })
    } catch (e) {
      if (version !== this._loadVersion) return
      console.error('加载记录数据失败', e)
      this.setData({ loading: false, loadError: '记录同步失败，请检查网络后重试' })
    }
  },

  onTabChange(e: any) {
    const tab = e.currentTarget.dataset.tab as TabType
    this.setData({ activeTab: tab })
  },

  // 预约卡片「现场取钥」 -> 统一路由至扫码核验页
  onReservationPickup(e: any) {
    const { id: rsvId, keyId } = e.detail
    const reservation = this.data.activeReservations.find(r => r.id === rsvId)
    if (!reservation?.canPickup || !canPickupReservation(reservation)) {
      wx.showToast({ title: '请在审批通过后的取钥窗口内操作', icon: 'none' })
      this.loadData()
      return
    }
    const key = this.data.keyMap[keyId]
	const deviceId = key?.deviceId
	if (!deviceId) {
		wx.showToast({ title: '钥匙尚未绑定柜机', icon: 'none' })
		return
	}
    wx.navigateTo({
      url: `/pages/scan/scan?mode=PICKUP&reservationId=${rsvId}&keyId=${keyId}&expectedDeviceId=${deviceId}`,
    })
  },

  // 取消预约
  async onReservationCancel(e: any) {
    const { id: rsvId } = e.detail
    if (this.data.cancellingId) return
    this.setData({ cancellingId: rsvId })
    try {
      const res = await wx.showModal({
        title: '取消预约',
        content: '确认取消该预约吗？取消后钥匙将重新对他人开放。',
      })

      if (res.confirm) {
        await reservationService.cancelReservation(rsvId)
        wx.showToast({ title: '已取消预约', icon: 'success' })
        await this.loadData()
      }
    } catch (e: any) {
      console.error('取消预约失败', e)
      wx.showToast({ title: e.message || '取消失败', icon: 'none' })
    } finally {
      this.setData({ cancellingId: '' })
    }
  },

  // 借用卡片「归还」 -> 统一路由至扫码核验页
  onBorrowReturn(e: any) {
    const { id: borrowId, keyId } = e.detail
    const record = [...this.data.currentBorrows, ...this.data.exceptionBorrows].find(b => b.id === borrowId)
    if (!record || !canReturnBorrow(record)) return
    const deviceId = record.deviceId
	if (!deviceId) {
		wx.showToast({ title: '钥匙尚未绑定柜机', icon: 'none' })
		return
	}
    wx.navigateTo({
      url: `/pages/scan/scan?mode=RETURN&borrowRecordId=${borrowId}&keyId=${keyId}&expectedDeviceId=${deviceId}`,
    })
  },

  goKeys() {
    wx.switchTab({ url: '/pages/keys/keys' })
  },

  goProfile() {
    wx.switchTab({ url: '/pages/profile/profile' })
  },

  resumeOperation() {
    const operation = this.data.activeOperation
    if (operation) wx.navigateTo({ url: `/pages/operation/operation?operationId=${encodeURIComponent(operation.id)}` })
  },
})
