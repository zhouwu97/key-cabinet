/** 所有环境共享的配置形状，业务服务只依赖这一层契约。 */
export type DataMode = 'mock' | 'api'

export type RuntimeEnvironment = 'devtools-mock' | 'devtools-local' | 'test' | 'prod'

export interface SubscriptionTemplates {
  returnReminder: string
  overdueAlert: string
}

export interface AppConfig {
  apiBaseURL: string
  dataMode: DataMode
  environment: RuntimeEnvironment
  subscriptionTemplates: SubscriptionTemplates
}

export const baseConfig: Pick<AppConfig, 'apiBaseURL' | 'dataMode' | 'subscriptionTemplates'> = {
  apiBaseURL: '',
  dataMode: 'api',
  subscriptionTemplates: {
    returnReminder: 'kcab_tmpl_return_reminder_v1',
    overdueAlert: 'kcab_tmpl_overdue_alert_v1',
  },
}
