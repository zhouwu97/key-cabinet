import {
  deviceService,
  keyService,
  userService,
  operationService,
} from '../../services/index'
import { currentConfig } from '../../config/index'
import { KeySlot } from '../../models/key-slot'
import { Key } from '../../models/key'
import { MockScenario, MOCK_SCENARIO_LABEL } from '../../mocks/mock-scenarios'
import { KEY_PRESENCE_LABEL } from '../../constants/labels'
import { KeyPresenceState } from '../../models/key-presence'
import { DeviceOperationAction } from '../../models/device-operation'
import { Reservation } from '../../models/reservation'
import { User } from '../../models/user'
import { Device } from '../../models/device'
import { httpClient } from '../../api/http-client'
import {
  MOCK_KEYS,
  MOCK_KEY_SLOTS,
  MOCK_RESERVATIONS,
  MOCK_BORROW_RECORDS,
  STORAGE_KEYS,
} from '../../mocks/mock-data'

Page({
  data: {
    slots: [] as KeySlot[],
    keys: [] as Key[],
    scenarios: Object.entries(MOCK_SCENARIO_LABEL).map(([key, label]) => ({ key, label })),
    currentScenario: MockScenario.SUCCESS,
    scenarioIndex: 0,
    isMockMode: currentConfig.dataMode === 'mock',
	devices: [] as Device[],
	selectedDeviceName: '设备信息待获取',
	pendingReservations: [] as Reservation[],
	pendingUsers: [] as User[],
	borrowedKeyCount: 0,
	availableKeyCount: 0,
    loading: true,
  },

  async onShow() {
    try {
      const user = await userService.getCurrentUser()
      if (!user || user.role !== 'ADMIN') {
        wx.showToast({ title: '无管理员访问权限', icon: 'none' })
        setTimeout(() => {
          wx.switchTab({ url: '/pages/home/home' })
        }, 1200)
        return
      }
      this.loadAdminData()
    } catch (e) {
      wx.switchTab({ url: '/pages/home/home' })
    }
  },

  async loadAdminData() {
    try {
      this.setData({ loading: true })
		const [devices, keys] = await Promise.all([
			deviceService.listDevices(),
			keyService.getKeys(),
		])
		const selectedDevice = devices[0] || null
		const slots = selectedDevice
			? await keyService.getDeviceSlots(selectedDevice.id)
			: []
		const [pendingReservations, pendingUsers] = this.data.isMockMode
			? [[], []]
			: await Promise.all([
				httpClient.request<Reservation[]>({ url: '/admin/reservations/pending' }),
				httpClient.request<User[]>({ url: '/admin/users/pending-verification' }),
			])

      const currentScenario = this.data.isMockMode
        ? deviceService.getGlobalScenario()
        : MockScenario.SUCCESS
      const scenarioIndex = this.data.scenarios.findIndex(s => s.key === currentScenario)

      this.setData({
        slots,
        keys,
		devices,
		selectedDeviceName: selectedDevice?.name || selectedDevice?.id || '设备信息待获取',
		pendingReservations,
		pendingUsers,
		borrowedKeyCount: keys.filter(key => key.status === 'BORROWED' || key.status === 'OVERDUE').length,
		availableKeyCount: keys.filter(key => key.status === 'AVAILABLE').length,
        currentScenario,
        scenarioIndex: scenarioIndex >= 0 ? scenarioIndex : 0,
        loading: false,
      })
    } catch (e) {
      console.error('加载管理员数据失败', e)
      this.setData({ loading: false })
    }
  },

	async approveReservation(e: any) {
		const id = e.currentTarget.dataset.id
		try {
			await httpClient.request({ url: `/admin/reservations/${encodeURIComponent(id)}/approve`, method: 'POST' })
			wx.showToast({ title: '已批准预约', icon: 'success' })
			await this.loadAdminData()
		} catch (error: any) {
			wx.showToast({ title: error.message || '审批失败', icon: 'none' })
		}
	},

	async rejectReservation(e: any) {
		const id = e.currentTarget.dataset.id
		const result = await wx.showModal({
			title: '拒绝预约',
			content: '',
			editable: true,
			placeholderText: '请输入拒绝原因',
		})
		if (!result.confirm || !result.content?.trim()) return
		try {
			await httpClient.request({
				url: `/admin/reservations/${encodeURIComponent(id)}/reject`,
				method: 'POST',
				data: { reason: result.content.trim() },
			})
			wx.showToast({ title: '已拒绝预约', icon: 'success' })
			await this.loadAdminData()
		} catch (error: any) {
			wx.showToast({ title: error.message || '审批失败', icon: 'none' })
		}
	},

	async verifyIdentity(e: any) {
		const id = e.currentTarget.dataset.id
		try {
			await httpClient.request({ url: `/admin/users/${encodeURIComponent(id)}/verify-identity`, method: 'POST' })
			wx.showToast({ title: '身份已核验', icon: 'success' })
			await this.loadAdminData()
		} catch (error: any) {
			wx.showToast({ title: error.message || '核验失败', icon: 'none' })
		}
	},

  onScenarioChange(e: any) {
    if (!this.data.isMockMode) return
    const idx = parseInt(e.detail.value)
    const item = this.data.scenarios[idx]
    if (item) {
      this.setData({
        scenarioIndex: idx,
        currentScenario: item.key as MockScenario,
      })
      deviceService.setGlobalScenario(item.key as MockScenario)
      wx.showToast({ title: `已设置场景：${item.label}`, icon: 'none' })
    }
  },

  async triggerDemoPickup() {
    if (!this.data.isMockMode) return
    try {
      const user = await userService.getCurrentUser()
      if (!user) return

      wx.showLoading({ title: '正在发起取钥...' })
      const op = await operationService.startOperation(
        {
          action: DeviceOperationAction.PICKUP,
          userId: user.id,
          keyId: 'KEY103',
          deviceId: 'CAB001',
          slotId: 'SLOT03',
        },
        this.data.currentScenario,
      )
      wx.hideLoading()

      wx.navigateTo({
        url: `/pages/operation/operation?operationId=${op.id}`,
      })
    } catch (e: any) {
      wx.hideLoading()
      wx.showToast({ title: e.message || '发起取钥失败', icon: 'none' })
    }
  },

  async triggerDemoReturn() {
    if (!this.data.isMockMode) return
    try {
      const user = await userService.getCurrentUser()
      if (!user) return

      wx.showLoading({ title: '正在发起归还...' })
      const op = await operationService.startOperation(
        {
          action: DeviceOperationAction.RETURN,
          userId: user.id,
          keyId: 'KEY104',
          deviceId: 'CAB001',
          slotId: 'SLOT04',
        },
        this.data.currentScenario,
      )
      wx.hideLoading()

      wx.navigateTo({
        url: `/pages/operation/operation?operationId=${op.id}`,
      })
    } catch (e: any) {
      wx.hideLoading()
      wx.showToast({ title: e.message || '发起归还失败', icon: 'none' })
    }
  },

  async resetMockData() {
    if (!this.data.isMockMode) return
    try {
      const res = await wx.showModal({
        title: '重置 Mock 数据',
        content: '确定重置所有钥匙、槽位、预约与借还记录到初始测试状态吗？',
      })

      if (res.confirm) {
        wx.setStorageSync(STORAGE_KEYS.KEYS, [...MOCK_KEYS])
        wx.setStorageSync(STORAGE_KEYS.KEY_SLOTS, [...MOCK_KEY_SLOTS])
        wx.setStorageSync(STORAGE_KEYS.RESERVATIONS, [...MOCK_RESERVATIONS])
        wx.setStorageSync(STORAGE_KEYS.BORROW_RECORDS, [...MOCK_BORROW_RECORDS])
        wx.setStorageSync(STORAGE_KEYS.OPERATIONS, [])
        wx.removeStorageSync(STORAGE_KEYS.ACTIVE_OPERATION_ID)

        deviceService.setGlobalScenario(MockScenario.SUCCESS)

        wx.showToast({ title: '数据已重置', icon: 'success' })
        this.loadAdminData()
      }
    } catch (e) {
      console.error('重置失败', e)
    }
  },

  getPresenceText(presence: KeyPresenceState) {
    return KEY_PRESENCE_LABEL[presence] || presence
  },
})
