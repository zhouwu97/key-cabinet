import { httpClient } from '../../api/http-client'
import { toTimestamp } from '../../api/serializers'
import { Device, DeviceCommandMessage, DeviceEventMessage } from '../../models/device'
import { MockScenario } from '../../mocks/mock-scenarios'
import { DeviceEventListener, DeviceService } from './device-service'

type ApiDevice = Device & {
  lastHeartbeatAt?: string | number
  lastHeartbeat?: string | number
}

function normalizeDevice(data: ApiDevice): Device {
  return {
    ...data,
    lastHeartbeat: toTimestamp(data.lastHeartbeatAt ?? data.lastHeartbeat),
  }
}

/** API 模式下设备状态只从后台读取；硬件指令由后台操作事务统一调度。 */
export class ApiDeviceService implements DeviceService {
  private readonly listeners = new Map<string, Set<DeviceEventListener>>()

  async getDeviceStatus(deviceId: string): Promise<Device> {
    const device = await httpClient.request<ApiDevice>({
      url: `/api/v1/devices/${encodeURIComponent(deviceId)}/status`,
    })
    return normalizeDevice(device)
  }

  isDeviceBusy(_deviceId: string): boolean {
    // API 模式的忙闲状态由后台操作实体返回，客户端不维护第二份状态机。
    return false
  }

  async executeCommand(_command: DeviceCommandMessage): Promise<void> {
    throw new Error('API 模式必须通过 operationService 发起设备操作')
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

  setGlobalScenario(_scenario: MockScenario): void {
    throw new Error('Mock 故障场景仅在 mock 数据模式可用')
  }

  getGlobalScenario(): MockScenario {
    throw new Error('Mock 故障场景仅在 mock 数据模式可用')
  }

  /** 保留事件分发入口，供后续 WebSocket/SSE 适配器接入，不在 API 模式伪造设备事件。 */
  protected emitDeviceEvent(deviceId: string, message: DeviceEventMessage): void {
    this.listeners.get(deviceId)?.forEach(listener => listener(message))
  }
}
