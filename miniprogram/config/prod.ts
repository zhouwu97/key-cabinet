import { AppConfig, baseConfig } from './base'

/** 正式版只允许访问正式 API，并由服务端继续执行生产安全校验。 */
export const prodConfig: AppConfig = {
  ...baseConfig,
  apiBaseURL: 'https://api.yourdomain.com',
  dataMode: 'api',
  environment: 'prod',
}
