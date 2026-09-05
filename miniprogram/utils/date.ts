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
