import {
  Device,
  DeviceAction,
  DeviceCommandMessage,
  DeviceEvent,
  DeviceEventMessage,
  DeviceStatus,
} from '../../models/device'
import { generateRequestId } from '../../utils/request-id'
import { DeviceEventListener, DeviceService } from './device-service'

const PROTOCOL_VERSION = '1.0'

// PRD 第三十五节：Mock 事件时序（毫秒偏移）
const BORROW_TIMELINE: Array<[DeviceEvent, number]> = [
  [DeviceEvent.RECEIVED, 500],
  [DeviceEvent.POSITIONING, 1500],
  [DeviceEvent.DOOR_OPEN, 3000],
  [DeviceEvent.KEY_REMOVED, 5000],
  [DeviceEvent.SUCCESS, 6500],
]

const RETURN_TIMELINE: Array<[DeviceEvent, number]> = [
  [DeviceEvent.RECEIVED, 500],
  [DeviceEvent.POSITIONING, 1500],
  [DeviceEvent.DOOR_OPEN, 3000],
  [DeviceEvent.KEY_RETURNED, 4500],
  [DeviceEvent.DOOR_CLOSED, 5500],
  [DeviceEvent.SUCCESS, 6500],
]

const MOCK_DEVICES: Record<string, Device> = {
  CAB001: { id: 'CAB001', name: '一号钥匙柜', status: DeviceStatus.ONLINE, lastHeartbeat: 0 },
  CAB002: { id: 'CAB002', name: '二号钥匙柜', status: DeviceStatus.OFFLINE, lastHeartbeat: 0 },
}

export class MockDeviceService implements DeviceService {
  private listeners = new Map<string, Set<DeviceEventListener>>()
  private busy = false

  borrowKey(deviceId: string, keyId: string): string {
    return this.run(deviceId, keyId, 'BORROW', BORROW_TIMELINE)
  }

  returnKey(deviceId: string, keyId: string): string {
    return this.run(deviceId, keyId, 'RETURN', RETURN_TIMELINE)
  }

  getDeviceStatus(deviceId: string): Promise<Device> {
    const base = MOCK_DEVICES[deviceId] ?? {
      id: deviceId,
      name: deviceId,
      status: DeviceStatus.OFFLINE,
      lastHeartbeat: 0,
    }
    return new Promise(resolve => {
      setTimeout(() => resolve({ ...base, lastHeartbeat: Date.now() }), 200)
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

  private run(
    deviceId: string,
    keyId: string,
    action: DeviceAction,
    timeline: Array<[DeviceEvent, number]>,
  ): string {
    if (this.busy) {
      throw new Error('DEVICE_BUSY')
    }
    this.busy = true
    const command: DeviceCommandMessage = {
      version: PROTOCOL_VERSION,
      requestId: generateRequestId(),
      action,
      keyId,
      timestamp: Date.now(),
    }
    timeline.forEach(([event, delay]) =>
      setTimeout(() => {
        this.emit(deviceId, {
          version: PROTOCOL_VERSION,
          requestId: command.requestId,
          event,
          timestamp: Date.now(),
        })
        if (event === DeviceEvent.SUCCESS) {
          this.busy = false
        }
      }, delay),
    )
    return command.requestId
  }

  private emit(deviceId: string, message: DeviceEventMessage): void {
    this.listeners.get(deviceId)?.forEach(listener => listener(message))
  }
}
