/** 将后端返回的 ISO 时间或毫秒时间戳统一转换为小程序使用的毫秒时间戳。 */
export function toTimestamp(value: unknown, fallback = Date.now()): number {
  if (typeof value === 'number' && Number.isFinite(value)) {
    return value
  }
  if (typeof value === 'string') {
    const timestamp = Date.parse(value)
    if (!Number.isNaN(timestamp)) {
      return timestamp
    }
  }
  return fallback
}

/** 构造稳定且经过编码的查询字符串，避免业务服务重复拼接参数。 */
export function toQueryString(
  params: Record<string, string | number | boolean | undefined>,
): string {
  const query = Object.entries(params)
    .filter(([, value]) => value !== undefined && value !== '')
    .map(([key, value]) => `${encodeURIComponent(key)}=${encodeURIComponent(String(value))}`)
    .join('&')
  return query ? `?${query}` : ''
}

export function toISOTime(value: number | undefined): string | undefined {
  return value === undefined ? undefined : new Date(value).toISOString()
}
