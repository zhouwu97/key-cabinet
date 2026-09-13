import {
  keyService,
  deviceService,
  userService,
  operationService,
  reservationService,
  borrowService,
} from '../../services/index'
import { DeviceOperationAction, DeviceOperationStatus } from '../../models/device-operation'
import { canPickupReservation } from '../../models/reservation'
import { canReturnBorrow } from '../../models/borrow-record'
import { Key } from '../../models/key'
import { Device } from '../../models/device'
import { currentConfig } from '../../config/index'

Page({
  data: {
    mode: 'PICKUP' as 'PICKUP' | 'RETURN',
    reservationId: '',
    borrowRecordId: '',
    keyId: '',
    expectedDeviceId: '',
    key: null as Key | null,
    device: null as Device | null,
    userName: '',
    keyDisplayName: '',
	cabinetName: '设备信息待获取',
	cabinetLocation: '位置未提供',
	isOnline: false,
	showMockTools: currentConfig.dataMode === 'mock',
    scannedCabinetId: '',
    verified: false,
    verifyError: '',
    starting: false,
    verifying: false,
    operationSlotId: '',
    activeOperationId: '',
  },

  async onLoad(options: any) {
    const mode = (options.mode === 'RETURN' ? 'RETURN' : 'PICKUP') as 'PICKUP' | 'RETURN'
    const reservationId = options.reservationId || ''
    const borrowRecordId = options.borrowRecordId || ''
    const keyId = options.keyId || ''
	const expectedDeviceId = options.expectedDeviceId || ''

    this.setData({
      mode,
      reservationId,
      borrowRecordId,
      keyId,
      expectedDeviceId,
    })

    await this.loadContext()
  },

  async loadContext() {
    try {
		const [user, key] = await Promise.all([
        userService.getCurrentUser(),
        this.data.keyId ? keyService.getKeyById(this.data.keyId) : Promise.resolve(null),
      ])
		const expectedDeviceId = this.data.expectedDeviceId || key?.deviceId || ''
		const device = expectedDeviceId
			? await deviceService.getDeviceStatus(expectedDeviceId)
			: null

      const keyDisplayName = key
        ? (!key.roomNo || key.name.includes(key.roomNo) ? key.name : `${key.roomNo}室 · ${key.name}`)
        : ''

      this.setData({
			expectedDeviceId,
        userName: user?.name || '',
        key: key || null,
        keyDisplayName,
        device: device || null,
		cabinetName: device?.name || '设备信息待获取',
		cabinetLocation: device?.location || '位置未提供',
        isOnline: device?.status === 'ONLINE',
			verifyError: expectedDeviceId ? '' : '当前任务没有绑定柜机，无法开始现场操作',
      })
    } catch (e) {
      console.error('加载核验上下文失败', e)
      this.setData({ verifyError: '任务信息加载失败，请重新扫码核验', verified: false })
    }
  },

  // 启动微信扫码
  startScan() {
    if (this.data.starting || this.data.verifying) return
    this.setData({ verifyError: '', verified: false, scannedCabinetId: '' })
    wx.scanCode({
      onlyFromCamera: true,
      scanType: ['qrCode'],
      success: res => {
        this.handleScanRawResult(res.result)
      },
      fail: err => {
        if (err.errMsg && !err.errMsg.includes('cancel')) {
          this.setData({ verifyError: '调用扫码失败，请重试或开启相机权限' })
        }
      },
    })
  },

	// 模拟入口只在 Mock 环境显示，API 模式不注入固定设备编号。
  simulateScanSuccess() {
		if (!this.data.showMockTools || !this.data.expectedDeviceId) return
		this.handleScanRawResult(JSON.stringify({ cabinetId: this.data.expectedDeviceId }))
  },

  // 解析扫码内容
  handleScanRawResult(raw: string) {
    if (this.data.starting || this.data.verifying) return
    let cabinetId = ''
    try {
      const parsed = JSON.parse(raw)
      const candidate = typeof parsed === 'string' ? parsed : parsed?.cabinetId || parsed?.deviceId
      cabinetId = typeof candidate === 'string' ? candidate.trim() : ''
    } catch {
		// 永久二维码只用于选择设备；最终授权始终由服务端按预约、用户和槽位关系判断。
      cabinetId = typeof raw === 'string' ? raw.trim() : ''
    }

    if (!cabinetId) {
      this.setData({
        verifyError: '二维码格式无效，未识别到钥匙柜编号',
        verified: false,
      })
      return
    }

    return this.verifyCabinet(cabinetId)
  },

  async verifyTask(scannedCabinetId: string) {
    const user = await userService.getCurrentUser()
    if (!user) throw new Error('用户登录态已失效，请重新登录')
    if (this.data.mode === 'PICKUP') {
      if (!this.data.reservationId) throw new Error('缺少有效预约，请从借还记录重新进入')
      if (!user.identityVerified) throw new Error('身份资料尚未通过核验，暂时无法取钥')
      const reservations = await reservationService.getUserReservations(user.id)
      const reservation = reservations.find(r => r.id === this.data.reservationId)
      if (!reservation || reservation.keyId !== this.data.keyId) throw new Error('预约与当前钥匙不匹配，请从借还记录重新进入')
      if (!canPickupReservation(reservation)) throw new Error('预约尚未获批或不在取钥窗口内，请查看最新预约状态')
      const key = await keyService.getKeyById(reservation.keyId)
      if (!key || key.deviceId !== scannedCabinetId || !key.slotId) throw new Error('预约钥匙与柜机不匹配，请检查柜机编号')
      this.setData({ key, operationSlotId: key.slotId })
    } else {
      if (!this.data.borrowRecordId) throw new Error('缺少借用记录，请从借还记录重新进入')
      const borrows = await borrowService.getUserBorrowRecords(user.id)
      const record = borrows.find(b => b.id === this.data.borrowRecordId)
      if (!record || record.keyId !== this.data.keyId) throw new Error('借用记录与当前钥匙不匹配')
      if (!canReturnBorrow(record)) throw new Error('当前记录不可重复归还，请查看已有操作进度或联系管理员')
      // 归还沿用借出时的柜机和槽位，避免钥匙配置变化后归还到错误位置。
      if (record.deviceId !== scannedCabinetId || !record.slotId) throw new Error('请前往借出记录指定的柜机归还')
      this.setData({ operationSlotId: record.slotId })
    }
  },

  // 核心核验逻辑
  async verifyCabinet(scannedCabinetId: string) {
    if (this.data.verifying) return false
    wx.showLoading({ title: '正在核验设备...' })
    this.setData({ verifyError: '', verified: false, verifying: true, activeOperationId: '' })

    try {
      // 已有会话优先于设备忙碌提示，确保用户始终可以恢复进度。
      const activeOp = await operationService.getActiveOperation()
      if (activeOp && [DeviceOperationStatus.CREATED, DeviceOperationStatus.AUTHORIZED,
        DeviceOperationStatus.SENT, DeviceOperationStatus.EXECUTING].includes(activeOp.status)) {
        this.setData({ activeOperationId: activeOp.id })
        throw new Error('检测到您当前有正在进行的钥匙柜操作，请先恢复并完成当前会话')
      }
      // 1. 验证设备是否在线
      const device = await deviceService.getDeviceStatus(scannedCabinetId)
      if (!device) {
        throw new Error(`未找到编号为 [${scannedCabinetId}] 的钥匙柜设备`)
      }
      if (device.status !== 'ONLINE') {
        throw new Error(`钥匙柜 [${device.name || scannedCabinetId}] 当前处于离线状态，暂时无法提供自助借还服务`)
      }

      // 2. 验证扫码设备 == 当前钥匙所在柜
		const expected = this.data.expectedDeviceId || this.data.key?.deviceId || ''
		if (!expected) {
			throw new Error('当前任务没有绑定目标柜机，请联系管理员检查钥匙配置')
		}
      if (scannedCabinetId !== expected) {
        throw new Error(`设备不匹配！目标钥匙存放在 [${expected}]，您扫描的是 [${scannedCabinetId}]，请前往指定钥匙柜扫码`)
      }

      await this.verifyTask(scannedCabinetId)

      // 核验通过，展示确认卡片
      this.setData({
        scannedCabinetId,
		cabinetName: device.name || scannedCabinetId,
		cabinetLocation: device.location || '位置未提供',
        isOnline: true,
        verified: true,
        verifyError: '',
      })
      return true
    } catch (err: any) {
      this.setData({
        verified: false,
        verifyError: err.message || '核验失败，请重试',
      })
      return false
    } finally {
      wx.hideLoading()
      this.setData({ verifying: false })
    }
  },

  // 重新扫码
  reScan() {
    if (this.data.starting || this.data.verifying) return
    this.setData({
      verified: false,
      verifyError: '',
    })
    this.startScan()
  },

  // 用户确认开始取钥 / 归还
  async onConfirmOperation() {
    if (this.data.starting || this.data.verifying) return
    if (!this.data.verified || !this.data.scannedCabinetId) {
      wx.showToast({ title: '请先扫描并核验目标柜机', icon: 'none' })
      return
    }
    this.setData({ starting: true })

    try {
      // 扫码与点击确认之间可能发生审批、超时或归还状态变化，发指令前重新核验。
      if (!await this.verifyCabinet(this.data.scannedCabinetId)) {
        this.setData({ starting: false })
        return
      }
      const user = await userService.getCurrentUser()
      if (!user) {
        wx.showToast({ title: '登录态失效', icon: 'none' })
        this.setData({ starting: false })
        return
      }

      const key = this.data.key
		const keyId = this.data.keyId || key?.id || ''
      const deviceId = this.data.scannedCabinetId
      const slotId = this.data.operationSlotId
		if (!keyId || !deviceId || !slotId) {
			throw new Error('钥匙、柜机或槽位信息不完整，无法发起设备操作')
		}
		if (this.data.mode === 'PICKUP' && !this.data.reservationId) {
			throw new Error('缺少有效预约，无法发起取钥操作')
		}
		if (this.data.mode === 'RETURN' && !this.data.borrowRecordId) {
			throw new Error('缺少借用记录，无法发起归还操作')
		}

      wx.showLoading({ title: '正在建立柜机连接...' })

      let op
      if (this.data.mode === 'PICKUP') {
        op = await operationService.startOperation({
          action: DeviceOperationAction.PICKUP,
          userId: user.id,
          keyId,
          deviceId,
          slotId,
          reservationId: this.data.reservationId || undefined,
        })
      } else {
        op = await operationService.startOperation({
          action: DeviceOperationAction.RETURN,
          userId: user.id,
          keyId,
          deviceId,
          slotId,
          borrowRecordId: this.data.borrowRecordId || undefined,
        })
      }

      wx.hideLoading()

      // 跳转至设备操作进度页面
      wx.redirectTo({
        url: `/pages/operation/operation?operationId=${op.id}`,
        fail: () => wx.switchTab({ url: '/pages/records/records' }),
      })
    } catch (err: any) {
      wx.hideLoading()
      this.setData({ starting: false })
      wx.showToast({ title: err.message || '发起操作失败，请重试', icon: 'none' })
    }
  },

  resumeOperation() {
    if (this.data.activeOperationId) {
      wx.redirectTo({ url: `/pages/operation/operation?operationId=${encodeURIComponent(this.data.activeOperationId)}` })
    }
  },
})
