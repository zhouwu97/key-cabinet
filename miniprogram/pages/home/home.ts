import { getDeviceService } from '../../services/device'
import { DEVICE_STATUS_LABEL } from '../../constants/labels'
import { DeviceStatus } from '../../models/device'

Page({
  data: {
    deviceName: '一号钥匙柜',
    deviceLabel: '检测中',
    tone: 'gray',
  },

  onLoad() {
    getDeviceService()
      .getDeviceStatus('CAB001')
      .then(device => {
        const tone =
          device.status === DeviceStatus.ONLINE
            ? 'green'
            : device.status === DeviceStatus.FAULT
              ? 'red'
              : 'gray'
        this.setData({
          deviceName: device.name,
          deviceLabel: DEVICE_STATUS_LABEL[device.status],
          tone,
        })
      })
  },

  goScan() {
    wx.navigateTo({ url: '/pages/scan/scan' })
  },

  goKeys() {
    wx.switchTab({ url: '/pages/keys/keys' })
  },

  goRecords() {
    wx.switchTab({ url: '/pages/records/records' })
  },
})
