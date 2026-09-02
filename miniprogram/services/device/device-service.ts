import { Device, DeviceEventMessage } from '../../models/device'

export type DeviceEventListener = (message: DeviceEventMessage) => void

export interface DeviceService {
  /** 发起取钥，返回 requestId；设备忙时抛错 */
  borrowKey(deviceId: string, keyId: string): string
  /** 发起归还，返回 requestId；设备忙时抛错 */
  returnKey(deviceId: string, keyId: string): string
  getDeviceStatus(deviceId: string): Promise<Device>
  subscribeDevice(deviceId: string, listener: DeviceEventListener): void
  unsubscribeDevice(deviceId: string, listener: DeviceEventListener): void
}
