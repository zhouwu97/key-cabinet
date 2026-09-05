import { currentConfig } from '../../config/index'
import { ApiDeviceService } from './api-device-service'
import { DeviceService } from './device-service'
import { MockDeviceService } from './mock-device-service'

let instance: DeviceService | null = null

export function getDeviceService(): DeviceService {
  if (instance === null) {
    instance = currentConfig.dataMode === 'api' ? new ApiDeviceService() : new MockDeviceService()
  }
  return instance
}

export { ApiDeviceService } from './api-device-service'
export { MockDeviceService } from './mock-device-service'
