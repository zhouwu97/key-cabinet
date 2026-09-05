import { AppConfig, baseConfig } from './base'

/** 体验版使用独立测试服务，避免与正式数据混用。 */
export const testConfig: AppConfig = {
  ...baseConfig,
  apiBaseURL: 'https://test-api.yourdomain.com',
  dataMode: 'api',
  environment: 'test',
}
