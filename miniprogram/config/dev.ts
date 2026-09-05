import { AppConfig, baseConfig } from './base'

/** 开发者工具默认使用 Mock，避免本地数据库状态污染演示数据。 */
export const devConfig: AppConfig = {
  ...baseConfig,
  apiBaseURL: 'http://localhost:8080',
  dataMode: 'mock',
  environment: 'devtools-mock',
}
