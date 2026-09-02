import {
  Device,
  DeviceCommandMessage,
  DeviceEvent,
  DeviceEventMessage,
  DeviceStatus,
} from '../../models/device'
import { OperationErrorCode } from '../../models/operation-error'
import { MockScenario } from '../../mocks/mock-scenarios'
import { DeviceEventListener, DeviceService } from './device-service'
import { MOCK_DEVICES, STORAGE_KEYS } from '../../mocks/mock-data'

const PROTOCOL_VERSION = '1.0'

export class MockDeviceService implements DeviceService {
  private listeners = new Map<string, Set<DeviceEventListener>>()
  private busyDevices = new Set<string>()
  private globalScenario: MockScenario = MockScenario.SUCCESS
  private runningTimers = new Map<string, Array<ReturnType<typeof setTimeout>>>()

  constructor() {
    this.loadScenarioFromStorage()
  }

  private loadScenarioFromStorage(): void {
    try {
      const stored = wx.getStorageSync(STORAGE_KEYS.MOCK_SCENARIO)
      if (stored && Object.values(MockScenario).includes(stored)) {
        this.globalScenario = stored
      }
    } catch {
      this.globalScenario = MockScenario.SUCCESS
    }
  }

  setGlobalScenario(scenario: MockScenario): void {
    this.globalScenario = scenario
    try {
      wx.setStorageSync(STORAGE_KEYS.MOCK_SCENARIO, scenario)
    } catch (e) {
      console.error('保存 Mock 场景失败', e)
    }
  }

  getGlobalScenario(): MockScenario {
    return this.globalScenario
  }

  isDeviceBusy(deviceId: string): boolean {
    return this.busyDevices.has(deviceId)
  }

  async getDeviceStatus(deviceId: string): Promise<Device> {
    const devices: Device[] = wx.getStorageSync(STORAGE_KEYS.DEVICES) || MOCK_DEVICES
    const target = devices.find(d => d.id === deviceId) ?? {
      id: deviceId,
      name: deviceId,
      status: DeviceStatus.OFFLINE,
      lastHeartbeat: 0,
    }

    // 若当前正在执行任务且设备在线，动态显示 BUSY
    const status = this.busyDevices.has(deviceId)
      ? DeviceStatus.BUSY
      : target.status

    return new Promise(resolve => {
      setTimeout(() => resolve({ ...target, status, lastHeartbeat: Date.now() }), 50)
    })
  }

  subscribeDevice(deviceId: string, listener: DeviceEventListener): void {
    if (!this.listeners.has(deviceId)) {
      this.listeners.set(deviceId, new Set())
    }
    this.listeners.get(deviceId)!.add(listener)
  }

  unsubscribeDevice(deviceId: string, listener: DeviceEventListener): void {
    this.listeners.get(deviceId)?.delete(listener)
  }

  async executeCommand(command: DeviceCommandMessage, scenario?: MockScenario): Promise<void> {
    const activeScenario = scenario || this.globalScenario
    const { deviceId, action, operationId, requestId } = command

    // 场景校验：设备离线
    if (activeScenario === MockScenario.DEVICE_OFFLINE || deviceId === 'CAB002') {
      const errorMsg: DeviceEventMessage = {
        version: PROTOCOL_VERSION,
        operationId,
        requestId,
        eventId: `EV_${Date.now()}_ERR`,
        seq: 1,
        deviceId,
        event: DeviceEvent.FAILED,
        errorCode: OperationErrorCode.DEVICE_OFFLINE,
        errorMessage: '设备处于离线状态，通信未响应',
        timestamp: Date.now(),
      }
      setTimeout(() => this.emit(deviceId, errorMsg), 200)
      return
    }

    // 场景校验：设备繁忙
    if (this.busyDevices.has(deviceId) || activeScenario === MockScenario.DEVICE_BUSY) {
      const errorMsg: DeviceEventMessage = {
        version: PROTOCOL_VERSION,
        operationId,
        requestId,
        eventId: `EV_${Date.now()}_ERR`,
        seq: 1,
        deviceId,
        event: DeviceEvent.FAILED,
        errorCode: OperationErrorCode.DEVICE_BUSY,
        errorMessage: '设备当前正忙，正在执行其他任务',
        timestamp: Date.now(),
      }
      setTimeout(() => this.emit(deviceId, errorMsg), 200)
      return
    }

    // 锁定当前设备（多设备隔离）
    this.busyDevices.add(deviceId)
    this.runningTimers.set(operationId, [])

    // 构建事件序列
    const timeline = this.buildTimeline(action, activeScenario)

    let seq = 1
    timeline.forEach(({ event, delay, errorCode, errorMessage }) => {
      const timer = setTimeout(() => {
        const msg: DeviceEventMessage = {
          version: PROTOCOL_VERSION,
          operationId,
          requestId,
          eventId: `EV_${Date.now()}_${seq}`,
          seq: seq++,
          deviceId,
          event,
          errorCode,
          errorMessage,
          timestamp: Date.now(),
        }

        this.emit(deviceId, msg)

        // 操作结束（成功或失败），释放设备占用
        if (event === DeviceEvent.SUCCESS || event === DeviceEvent.FAILED) {
          this.busyDevices.delete(deviceId)
          this.runningTimers.delete(operationId)
        }
      }, delay)

      this.runningTimers.get(operationId)?.push(timer)
    })
  }

  private buildTimeline(
    action: 'PICKUP' | 'RETURN',
    scenario: MockScenario,
  ): Array<{ event: DeviceEvent; delay: number; errorCode?: OperationErrorCode | string; errorMessage?: string }> {
    if (action === 'PICKUP') {
      if (scenario === MockScenario.POSITION_ERROR) {
        return [
          { event: DeviceEvent.RECEIVED, delay: 300 },
          { event: DeviceEvent.AUTH_CONFIRMED, delay: 800 },
          { event: DeviceEvent.POSITIONING, delay: 1500 },
          {
            event: DeviceEvent.FAILED,
            delay: 2500,
            errorCode: OperationErrorCode.MOTOR_POSITION_FAILED,
            errorMessage: '电机定位超时，槽位对准失败',
          },
        ]
      }

      if (scenario === MockScenario.DOOR_ERROR) {
        return [
          { event: DeviceEvent.RECEIVED, delay: 300 },
          { event: DeviceEvent.AUTH_CONFIRMED, delay: 800 },
          { event: DeviceEvent.POSITIONING, delay: 1500 },
          { event: DeviceEvent.POSITIONED, delay: 2200 },
          {
            event: DeviceEvent.FAILED,
            delay: 3000,
            errorCode: OperationErrorCode.DOOR_OPEN_FAILED,
            errorMessage: '电磁锁开启异常，安全门未能打开',
          },
        ]
      }

      if (scenario === MockScenario.TIMEOUT) {
        return [
          { event: DeviceEvent.RECEIVED, delay: 300 },
          { event: DeviceEvent.AUTH_CONFIRMED, delay: 800 },
          { event: DeviceEvent.POSITIONING, delay: 1500 },
          {
            event: DeviceEvent.FAILED,
            delay: 4000,
            errorCode: OperationErrorCode.OPERATION_TIMEOUT,
            errorMessage: '设备执行操作响应超时',
          },
        ]
      }

      // 正常取钥时序
      return [
        { event: DeviceEvent.RECEIVED, delay: 300 },
        { event: DeviceEvent.AUTH_CONFIRMED, delay: 800 },
        { event: DeviceEvent.POSITIONING, delay: 1600 },
        { event: DeviceEvent.POSITIONED, delay: 2400 },
        { event: DeviceEvent.DOOR_OPEN, delay: 3200 },
        { event: DeviceEvent.WAITING_REMOVE, delay: 3600 },
        { event: DeviceEvent.KEY_REMOVED, delay: 5000 },
        { event: DeviceEvent.DOOR_CLOSED, delay: 6200 },
        { event: DeviceEvent.HOMING, delay: 7000 },
        { event: DeviceEvent.SUCCESS, delay: 8000 },
      ]
    }

    // 归还操作 RETURN
    if (scenario === MockScenario.RFID_WRONG_KEY) {
      return [
        { event: DeviceEvent.RECEIVED, delay: 300 },
        { event: DeviceEvent.POSITIONING, delay: 1200 },
        { event: DeviceEvent.POSITIONED, delay: 2000 },
        { event: DeviceEvent.DOOR_OPEN, delay: 2800 },
        { event: DeviceEvent.KEY_RETURNED, delay: 4200 },
        {
          event: DeviceEvent.FAILED,
          delay: 5500,
          errorCode: OperationErrorCode.RFID_WRONG_KEY,
          errorMessage: '检测到归还钥匙 RFID 不匹配，请放入正确的钥匙！',
        },
      ]
    }

    if (scenario === MockScenario.RFID_NOT_FOUND) {
      return [
        { event: DeviceEvent.RECEIVED, delay: 300 },
        { event: DeviceEvent.POSITIONING, delay: 1200 },
        { event: DeviceEvent.POSITIONED, delay: 2000 },
        { event: DeviceEvent.DOOR_OPEN, delay: 2800 },
        { event: DeviceEvent.KEY_RETURNED, delay: 4200 },
        {
          event: DeviceEvent.FAILED,
          delay: 5500,
          errorCode: OperationErrorCode.RFID_NOT_FOUND,
          errorMessage: '未能读取到钥匙 RFID 标签，请重新放置',
        },
      ]
    }

    if (scenario === MockScenario.DOOR_ERROR) {
      return [
        { event: DeviceEvent.RECEIVED, delay: 300 },
        { event: DeviceEvent.POSITIONING, delay: 1200 },
        {
          event: DeviceEvent.FAILED,
          delay: 2200,
          errorCode: OperationErrorCode.DOOR_OPEN_FAILED,
          errorMessage: '归还仓门打开失败',
        },
      ]
    }

    // 正常归还时序
    return [
      { event: DeviceEvent.RECEIVED, delay: 300 },
      { event: DeviceEvent.POSITIONING, delay: 1200 },
      { event: DeviceEvent.POSITIONED, delay: 2000 },
      { event: DeviceEvent.DOOR_OPEN, delay: 2800 },
      { event: DeviceEvent.KEY_RETURNED, delay: 4200 },
      { event: DeviceEvent.RFID_CONFIRMED, delay: 5400 },
      { event: DeviceEvent.DOOR_CLOSED, delay: 6600 },
      { event: DeviceEvent.HOMING, delay: 7400 },
      { event: DeviceEvent.SUCCESS, delay: 8200 },
    ]
  }

  private emit(deviceId: string, message: DeviceEventMessage): void {
    this.listeners.get(deviceId)?.forEach(listener => {
      try {
        listener(message)
      } catch (e) {
        console.error('设备事件分发异常', e)
      }
    })
  }
}
