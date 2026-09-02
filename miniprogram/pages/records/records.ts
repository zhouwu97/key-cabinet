import {
  reservationService,
  borrowService,
  userService,
  keyService,
  operationService,
} from '../../services'
import { Reservation, ReservationStatus } from '../../models/reservation'
import { BorrowRecord, BorrowRecordStatus, isRecordOverdue } from '../../models/borrow-record'
import { Key } from '../../models/key'
import {
  RESERVATION_STATUS_LABEL,
  RESERVATION_STATUS_TONE,
  getBorrowRecordDisplayStatus,
} from '../../constants/labels'
import { DeviceOperationAction } from '../../models/device-operation'

type TabType = 'CURRENT' | 'RESERVATIONS' | 'HISTORY' | 'EXCEPTION'

function formatDateTime(timestamp?: number): string {
  if (!timestamp) return '-'
  const date = new Date(timestamp)
  const m = (date.getMonth() + 1).toString().padStart(2, '0')
  const d = date.getDate().toString().padStart(2, '0')
  const h = date.getHours().toString().padStart(2, '0')
  const min = date.getMinutes().toString().padStart(2, '0')
  return `${m}-${d} ${h}:${min}`
}

function formatTime(timestamp?: number): string {
  if (!timestamp) return '--:--'
  const date = new Date(timestamp)
  const h = date.getHours().toString().padStart(2, '0')
  const min = date.getMinutes().toString().padStart(2, '0')
  return `${h}:${min}`
}

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
    reservations: [] as ReservationViewModel[],
    currentBorrows: [] as BorrowViewModel[],
    historyBorrows: [] as BorrowViewModel[],
    exceptionBorrows: [] as BorrowViewModel[],
    keyMap: {} as Record<string, Key>,
    loading: true,
  },

  onLoad() {
    this.checkInitialTab()
    this.loadData()
  },

  onShow() {
    this.checkInitialTab()
    this.loadData()
  },

  checkInitialTab() {
    try {
      const initialTab = wx.getStorageSync('kcab_records_initial_tab')
      if (initialTab) {
        this.setData({ activeTab: initialTab as TabType })
        wx.removeStorageSync('kcab_records_initial_tab')
      }
    } catch (e) {}
  },

  async loadData() {
    try {
      this.setData({ loading: true })

      const user = await userService.getCurrentUser()
      if (!user) {
        this.setData({ loading: false })
        return
      }

      const [rawReservations, rawBorrowRecords, keys] = await Promise.all([
        reservationService.getUserReservations(user.id),
        borrowService.getUserBorrowRecords(user.id),
        keyService.getKeys(),
      ])

      const keyMap: Record<string, Key> = {}
      keys.forEach(key => {
        keyMap[key.id] = key
      })

      const getKeyName = (keyId: string) => {
        const key = keyMap[keyId]
        return key ? `${key.roomNo} ${key.name}` : keyId
      }

      // 格式化预约 ViewModel
      const reservations: ReservationViewModel[] = rawReservations.map(r => ({
        ...r,
        keyName: getKeyName(r.keyId),
        pickupWindowText: `${formatDateTime(r.pickupWindowStart)} - ${formatTime(r.pickupWindowEnd)}`,
        expectedReturnText: formatDateTime(r.expectedReturnAt),
        statusLabel: RESERVATION_STATUS_LABEL[r.status] || '未知',
        statusTone: RESERVATION_STATUS_TONE[r.status] || 'gray',
        canPickup: r.status === ReservationStatus.ACTIVE || r.status === ReservationStatus.APPROVED,
        canCancel: r.status === ReservationStatus.ACTIVE || r.status === ReservationStatus.PENDING,
      }))

      // 格式化借还 ViewModel
      const mapBorrowViewModel = (b: BorrowRecord): BorrowViewModel => {
        const statusInfo = getBorrowRecordDisplayStatus(b)
        const canReturn =
          b.status === BorrowRecordStatus.BORROWED ||
          b.status === BorrowRecordStatus.BORROWING ||
          isRecordOverdue(b)

        return {
          ...b,
          keyName: getKeyName(b.keyId),
          borrowedAtText: formatDateTime(b.borrowedAt),
          expectedReturnText: formatDateTime(b.expectedReturnAt),
          returnedAtText: formatDateTime(b.returnedAt),
          statusLabel: statusInfo.label,
          statusTone: statusInfo.tone,
          canReturn,
        }
      }

      const currentBorrows: BorrowViewModel[] = rawBorrowRecords
        .filter(
          r =>
            r.status === BorrowRecordStatus.BORROWED ||
            r.status === BorrowRecordStatus.BORROWING ||
            r.status === BorrowRecordStatus.RETURNING,
        )
        .map(mapBorrowViewModel)

      const historyBorrows: BorrowViewModel[] = rawBorrowRecords
        .filter(r => r.status === BorrowRecordStatus.COMPLETED)
        .map(mapBorrowViewModel)

      const exceptionBorrows: BorrowViewModel[] = rawBorrowRecords
        .filter(r => isRecordOverdue(r) || r.status === BorrowRecordStatus.EXCEPTION)
        .map(mapBorrowViewModel)

      this.setData({
        reservations,
        currentBorrows,
        historyBorrows,
        exceptionBorrows,
        keyMap,
        loading: false,
      })
    } catch (e) {
      console.error('加载记录数据失败', e)
      this.setData({ loading: false })
      wx.showToast({ title: '加载失败', icon: 'none' })
    }
  },

  onTabChange(e: any) {
    const tab = e.currentTarget.dataset.tab as TabType
    this.setData({ activeTab: tab })
  },

  goKeys() {
    wx.switchTab({ url: '/pages/keys/keys' })
  },

  async onReservationPickup(e: any) {
    const { id: rsvId, keyId } = e.detail
    try {
      const user = await userService.getCurrentUser()
      if (!user) return

      const key = this.data.keyMap[keyId]
      if (!key) return

      wx.showLoading({ title: '正在建立会话...' })
      const op = await operationService.startOperation({
        action: DeviceOperationAction.PICKUP,
        userId: user.id,
        keyId,
        deviceId: key.deviceId,
        slotId: key.slotId,
        reservationId: rsvId,
      })
      wx.hideLoading()

      wx.navigateTo({
        url: `/pages/operation/operation?operationId=${op.id}`,
      })
    } catch (e: any) {
      wx.hideLoading()
      wx.showToast({ title: e.message || '发起取钥失败', icon: 'none' })
    }
  },

  async onReservationCancel(e: any) {
    const { id: rsvId } = e.detail
    try {
      const res = await wx.showModal({
        title: '取消预约',
        content: '确认取消该预约吗？取消后钥匙将重新对他人开放。',
      })

      if (res.confirm) {
        await reservationService.cancelReservation(rsvId)
        wx.showToast({ title: '已取消预约', icon: 'success' })
        this.loadData()
      }
    } catch (e: any) {
      console.error('取消预约失败', e)
      wx.showToast({ title: '取消失败', icon: 'none' })
    }
  },

  async onBorrowReturn(e: any) {
    const { id: borrowId, keyId } = e.detail
    try {
      const user = await userService.getCurrentUser()
      if (!user) return

      const key = this.data.keyMap[keyId]
      if (!key) return

      wx.showLoading({ title: '正在开启归还口...' })
      const op = await operationService.startOperation({
        action: DeviceOperationAction.RETURN,
        userId: user.id,
        keyId,
        deviceId: key.deviceId,
        slotId: key.slotId,
        borrowRecordId: borrowId,
      })
      wx.hideLoading()

      wx.navigateTo({
        url: `/pages/operation/operation?operationId=${op.id}`,
      })
    } catch (e: any) {
      wx.hideLoading()
      wx.showToast({ title: e.message || '发起归还失败', icon: 'none' })
    }
  },
})
