Page({
  onScan() {
    wx.scanCode({
      success: res => {
        wx.showToast({ title: `扫码结果：${res.result}`, icon: 'none' })
      },
      fail: () => {
        wx.showToast({ title: '已取消扫码', icon: 'none' })
      },
    })
  },
})
