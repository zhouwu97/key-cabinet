/**
 * 应用数据模式
 */
export type DataMode = 'mock' | 'api'

/**
 * 应用配置接口
 */
export interface AppConfig {
  apiBaseURL: string
  dataMode: DataMode
}

/**
 * 开发环境配置（默认使用 Mock，可切换为 'api' 连接本地 Go 后端）
 */
export const devConfig: AppConfig = {
  apiBaseURL: 'http://localhost:8080',
  dataMode: 'mock',
}

/**
 * 生产环境配置
 */
export const prodConfig: AppConfig = {
  apiBaseURL: 'https://api.yourdomain.com',
  dataMode: 'api',
}

/**
 * 当前生效配置
 */
export const currentConfig: AppConfig = devConfig
