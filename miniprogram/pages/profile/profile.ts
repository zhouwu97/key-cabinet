import { userService, borrowService, reservationService, deviceService } from '../../services/index'
import { User } from '../../models/user'
import { BorrowRecordStatus } from '../../models/borrow-record'
import { ReservationStatus } from '../../models/reservation'

Page({
  data: {
    user: null as User | null,
    currentBorrowCount: 0,
    activeReservationCount: 0,
    totalBorrowCount: 0,
    loading: true,
  },

  onShow() {
    this.loadUserData()
  },

  async loadUserData() {
    try {
      this.setData({ loading: true })
      const user = await userService.getCurrentUser()
      if (!user) {
        this.setData({ loading: false })
        return
      }

      const [borrows, reservations] = await Promise.all([
        borrowService.getUserBorrowRecords(user.id).catch(() => []),
        reservationService.getUserReservations(user.id).catch(() => []),
      ])

      const currentBorrowCount = borrows.filter(
        b =>
          b.status === BorrowRecordStatus.BORROWED ||
          b.status === BorrowRecordStatus.BORROWING ||
          b.status === BorrowRecordStatus.RETURNING,
      ).length

      const activeReservationCount = reservations.filter(
        r =>
          r.status === ReservationStatus.ACTIVE ||
          r.status === ReservationStatus.APPROVED,
      ).length

      const totalBorrowCount = borrows.length

      this.setData({
        user,
        currentBorrowCount,
        activeReservationCount,
        totalBorrowCount,
        loading: false,
      })
    } catch (e) {
      console.error('加载用户数据失败', e)
      this.setData({ loading: false })
    }
  },

  goCurrentBorrows() {
    try {
      wx.setStorageSync('kcab_records_initial_tab', 'CURRENT')
    } catch (e) {}
    wx.switchTab({ url: '/pages/records/records' })
  },

  goReservations() {
    try {
      wx.setStorageSync('kcab_records_initial_tab', 'RESERVATIONS')
    } catch (e) {}
    wx.switchTab({ url: '/pages/records/records' })
  },

  goHistory() {
    try {
      wx.setStorageSync('kcab_records_initial_tab', 'HISTORY')
    } catch (e) {}
    wx.switchTab({ url: '/pages/records/records' })
  },

  showHelpModal() {
    wx.showModal({
      title: '借还流程说明',
      content:
        '1. 找钥匙选择目标房间提交预约。\n2. 在取钥窗口期内前往钥匙柜点击取钥。\n3. 使用完毕后将钥匙插入归还口，通过 RFID 芯片自动识别完成结算。',
      showCancel: false,
      confirmText: '我知道了',
    })
  },

	async showLocationsModal() {
		try {
			const devices = await deviceService.listDevices()
			const content = devices.length
				? devices
					.map(device => `● ${device.name || device.id}：${device.location || '位置未提供'}（${device.status === 'ONLINE' ? '在线' : '不可用'}）`)
					.join('\n')
				: '暂未获取到钥匙柜信息'
			wx.showModal({ title: '钥匙柜分布位置', content, showCancel: false, confirmText: '我知道了' })
		} catch (error) {
			console.error('加载钥匙柜位置失败', error)
			wx.showToast({ title: '暂时无法获取柜机信息', icon: 'none' })
		}
  },

  showRulesModal() {
    wx.showModal({
      title: '实验室钥匙借用守则',
      content:
        '1. 钥匙仅限本人使用，严禁私自转借。\n2. 请于预约应还时间内归还，逾期将扣减信用分。\n3. 如遇钥匙丢失或损坏，请立即前往管理员控制台报障。',
      showCancel: false,
      confirmText: '遵守守则',
    })
  },

  showAbout() {
    wx.showModal({
      title: '智能钥匙自助借还系统',
      content:
		'阶段：真实系统收口\n架构：WeChat MiniProgram + Go Gin Backend + PostgreSQL\n状态：软件借还闭环已完成，MQTT 实体柜机网关待接入',
      showCancel: false,
      confirmText: '确定',
    })
  },

  goIdentityBind() {
    wx.navigateTo({ url: '/pages/identity-bind/identity-bind' })
  },

  goAdmin() {
    if (this.data.user?.role !== 'ADMIN') {
      wx.showToast({ title: '无权访问管理中心', icon: 'none' })
      return
    }
    wx.navigateTo({ url: '/pages/admin/admin' })
  },
})
