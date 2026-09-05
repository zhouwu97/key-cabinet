import { currentConfig } from './config/index'
import { authService } from './services/auth/index'

App({
  onLaunch() {
    if (currentConfig.dataMode !== 'api') {
      console.log('应用使用 Mock 数据模式')
      return
    }

    // API 模式在启动时建立真实登录态，失败只记录日志，由后续页面按需重试。
    void authService
      .login()
      .then(() => console.log('用户登录成功'))
      .catch(error => console.error('用户登录失败:', error))
  },
})
