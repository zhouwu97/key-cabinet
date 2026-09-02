import { getDeviceService } from '../../services/device'
import { DEVICE_EVENT_LABEL } from '../../constants/labels'
import { DeviceEvent, DeviceEventMessage } from '../../models/device'

const DEVICE_ID = 'CAB001'

Page({
  data: {
    latest: '尚未发起',
    log: [] as string[],
  },

  demoBorrow() {
    const service = getDeviceService()
    const onEvent = (message: DeviceEventMessage) => {
      const text = DEVICE_EVENT_LABEL[message.event]
      this.setData({ latest: text, log: [...this.data.log, text] })
      if (message.event === DeviceEvent.SUCCESS) {
        service.unsubscribeDevice(DEVICE_ID, onEvent)
      }
    }
    service.subscribeDevice(DEVICE_ID, onEvent)
    try {
      service.borrowKey(DEVICE_ID, 'KEY103')
      this.setData({ latest: '已发送取钥指令', log: [] })
    } catch {
      service.unsubscribeDevice(DEVICE_ID, onEvent)
      wx.showToast({ title: '设备忙，请稍后再试', icon: 'none' })
    }
  },
})
