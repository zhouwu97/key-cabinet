let seq = 0

export function generateRequestId(): string {
  const now = new Date()
  const date = [
    now.getFullYear(),
    String(now.getMonth() + 1).padStart(2, '0'),
    String(now.getDate()).padStart(2, '0'),
  ].join('')
  seq = (seq + 1) % 100000
  return `REQ-${date}-${String(seq).padStart(5, '0')}`
}
