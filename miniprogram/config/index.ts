import { AppConfig, RuntimeEnvironment } from './base'
import { devConfig } from './dev'
import { localConfig } from './local'
import { prodConfig } from './prod'
import { testConfig } from './test'

export type { AppConfig, DataMode, RuntimeEnvironment } from './base'
export { baseConfig } from './base'
export { devConfig } from './dev'
export { localConfig } from './local'
export { testConfig } from './test'
export { prodConfig } from './prod'

const ENVIRONMENT_OVERRIDE_KEY = 'key-cabinet.config.environment'

function isRuntimeEnvironment(value: unknown): value is RuntimeEnvironment {
  return (
    value === 'devtools-mock' ||
    value === 'devtools-local' ||
    value === 'test' ||
    value === 'prod'
  )
}

function readWxEnvironmentVersion(): 'develop' | 'trial' | 'release' | undefined {
  if (typeof wx === 'undefined') return undefined

  try {
    const runtime = wx as unknown as {
      getAccountInfoSync?: () => { miniProgram?: { envVersion?: string } }
    }
    const envVersion = runtime.getAccountInfoSync?.().miniProgram?.envVersion
    if (envVersion === 'develop' || envVersion === 'trial' || envVersion === 'release') {
      return envVersion
    }
  } catch (error) {
    console.warn('读取小程序运行环境失败，将使用开发 Mock 配置', error)
  }
  return undefined
}

function readEnvironmentOverride(): RuntimeEnvironment | undefined {
  if (typeof wx === 'undefined') return undefined

  try {
    const runtime = wx as unknown as {
      getStorageSync?: (key: string) => unknown
    }
    const override = runtime.getStorageSync?.(ENVIRONMENT_OVERRIDE_KEY)
    return isRuntimeEnvironment(override) ? override : undefined
  } catch (error) {
    console.warn('读取小程序环境覆盖失败', error)
    return undefined
  }
}

/** 供开发者工具切换本地 API；正式版不应调用该方法。 */
export function setEnvironmentOverride(environment: RuntimeEnvironment | null): void {
  if (typeof wx === 'undefined') return

  const runtime = wx as unknown as {
    setStorageSync?: (key: string, value: string) => void
    removeStorageSync?: (key: string) => void
  }
  if (environment === null) {
    runtime.removeStorageSync?.(ENVIRONMENT_OVERRIDE_KEY)
    return
  }
  runtime.setStorageSync?.(ENVIRONMENT_OVERRIDE_KEY, environment)
}

export function resolveEnvironment(): RuntimeEnvironment {
  const wxEnvironment = readWxEnvironmentVersion()

  // release 永远绑定 prod，防止遗留本地覆盖把正式版导向测试或 Mock。
  if (wxEnvironment === 'release') return 'prod'

	const override = readEnvironmentOverride()
	if (override && override !== 'prod') return override

  if (wxEnvironment === 'trial') return 'test'
  return 'devtools-mock'
}

const configs: Record<RuntimeEnvironment, AppConfig> = {
  'devtools-mock': devConfig,
  'devtools-local': localConfig,
  test: testConfig,
  prod: prodConfig,
}

export const currentConfig: AppConfig = configs[resolveEnvironment()]
