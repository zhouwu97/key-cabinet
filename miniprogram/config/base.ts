/** 所有环境共享的配置形状，业务服务只依赖这一层契约。 */
export type DataMode = 'mock' | 'api'

export type RuntimeEnvironment = 'devtools-mock' | 'devtools-local' | 'test' | 'prod'

export interface AppConfig {
  apiBaseURL: string
  dataMode: DataMode
  environment: RuntimeEnvironment
}

export const baseConfig: Pick<AppConfig, 'apiBaseURL' | 'dataMode'> = {
  apiBaseURL: '',
  dataMode: 'api',
}
