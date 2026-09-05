import { AppConfig, baseConfig } from './base'

/** 开发者工具本地 API 环境，通过运行时环境覆盖显式启用。 */
export const localConfig: AppConfig = {
  ...baseConfig,
  apiBaseURL: 'http://localhost:8080',
  dataMode: 'api',
  environment: 'devtools-local',
}
