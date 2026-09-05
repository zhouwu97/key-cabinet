import {
  reservationService,
  borrowService,
  userService,
  keyService,
} from '../../services/index'
import { Reservation, ReservationStatus } from '../../models/reservation'
import { BorrowRecord, BorrowRecordStatus, isRecordOverdue } from '../../models/borrow-record'
import { Key } from '../../models/key'
import {
  RESERVATION_STATUS_LABEL,
  RESERVATION_STATUS_TONE,
  getBorrowRecordDisplayStatus,
} from '../../constants/labels'
import { formatDateTime, formatTime } from '../../utils/date'

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
      let initialTab = wx.getStorageSync('kcab_records_initial_tab')
      if (initialTab) {
        if (initialTab === 'RESERVATIONS') initialTab = 'CURRENT'
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

      // 格式化当前有效待履约预约
      const activeReservations: ReservationViewModel[] = rawReservations
        .filter(
          r =>
            r.status === ReservationStatus.ACTIVE ||
            r.status === ReservationStatus.APPROVED ||
            r.status === ReservationStatus.PENDING,
        )
        .map(r => ({
          ...r,
          keyName: getKeyName(r.keyId),
          pickupWindowText: `${formatDateTime(r.pickupWindowStart)} - ${formatTime(r.pickupWindowEnd)}`,
          expectedReturnText: formatDateTime(r.expectedReturnAt),
          statusLabel: RESERVATION_STATUS_LABEL[r.status] || '待取钥',
          statusTone: RESERVATION_STATUS_TONE[r.status] || 'blue',
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

  // 预约卡片「现场取钥」 -> 统一路由至扫码核验页
  onReservationPickup(e: any) {
    const { id: rsvId, keyId } = e.detail
    const key = this.data.keyMap[keyId]
    const deviceId = key?.deviceId || 'CAB001'
    wx.navigateTo({
      url: `/pages/scan/scan?mode=PICKUP&reservationId=${rsvId}&keyId=${keyId}&expectedDeviceId=${deviceId}`,
    })
  },

  // 取消预约
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

  // 借用卡片「归还」 -> 统一路由至扫码核验页
  onBorrowReturn(e: any) {
    const { id: borrowId, keyId } = e.detail
    const key = this.data.keyMap[keyId]
    const deviceId = key?.deviceId || 'CAB001'
    wx.navigateTo({
      url: `/pages/scan/scan?mode=RETURN&borrowRecordId=${borrowId}&keyId=${keyId}&expectedDeviceId=${deviceId}`,
    })
  },

  goKeys() {
    wx.switchTab({ url: '/pages/keys/keys' })
  },
})
