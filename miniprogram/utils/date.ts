export function formatDateTime(timestamp?: number): string {
  if (!timestamp) return '-'
  const date = new Date(timestamp)
  const m = (date.getMonth() + 1).toString().padStart(2, '0')
  const d = date.getDate().toString().padStart(2, '0')
  const h = date.getHours().toString().padStart(2, '0')
  const min = date.getMinutes().toString().padStart(2, '0')
  return `${m}-${d} ${h}:${min}`
}

export function formatTime(timestamp?: number): string {
  if (!timestamp) return '--:--'
  const date = new Date(timestamp)
  const h = date.getHours().toString().padStart(2, '0')
  const min = date.getMinutes().toString().padStart(2, '0')
  return `${h}:${min}`
}

export function formatDate(timestamp: number): string {
  const date = new Date(timestamp)
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}`
}

/** 按本地日历构造时间，避免小程序平台对日期字符串的解析差异。 */
export function parseLocalDateTime(date: string, time: string): number {
  if (!/^\d{4}-\d{2}-\d{2}$/.test(date) || !/^\d{2}:\d{2}$/.test(time)) return NaN
  const [year, month, day] = date.split('-').map(Number)
  const [hour, minute] = time.split(':').map(Number)
  const value = new Date(year, month - 1, day, hour, minute, 0, 0)
  if (formatDate(value.getTime()) !== date || formatTime(value.getTime()) !== time) return NaN
  return value.getTime()
}

export function formatPickupWindow(start: number, end: number): string {
  return `${formatDateTime(start)} - ${formatDate(start) === formatDate(end) ? formatTime(end) : formatDateTime(end)}`
}
