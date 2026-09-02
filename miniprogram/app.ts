App({
  onLaunch() {
    // 登录态换取 openid 待后端接入后实现（PRD 第二十四节）
    wx.login({
      success: res => {
        console.log('login code', res.code)
      },
    })
  },
})
