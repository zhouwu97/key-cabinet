/**
 * API 配置
 */
export interface ApiConfig {
  apiBaseURL: string
}

/**
 * 开发环境配置
 */
const devConfig: ApiConfig = {
  apiBaseURL: 'http://localhost:8080',
}

/**
 * 生产环境配置
 */
const prodConfig: ApiConfig = {
  apiBaseURL: 'https://api.yourdomain.com',
}

/**
 * 当前配置
 * TODO: 根据编译环境自动切换
 */
export const currentConfig: ApiConfig = devConfig
