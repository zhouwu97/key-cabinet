import { USE_MOCK } from '../../constants/config'
import { DeviceService } from './device-service'
import { MockDeviceService } from './mock-device-service'

let instance: DeviceService | null = null

export function getDeviceService(): DeviceService {
  if (instance === null) {
    // 单点切换：阶段五在此替换为 MQTT 实现（PRD 第三十节）
    instance = USE_MOCK ? new MockDeviceService() : createRealService()
  }
  return instance
}

function createRealService(): DeviceService {
  throw new Error('MQTT DeviceService 尚未接入（PRD 阶段五）')
}
