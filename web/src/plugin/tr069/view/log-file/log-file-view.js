export const formatBytes = (value) => {
  const bytes = Number(value)
  if (!Number.isFinite(bytes) || bytes <= 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  const index = Math.min(Math.floor(Math.log(bytes) / Math.log(1024)), units.length - 1)
  const amount = bytes / (1024 ** index)
  return `${Number(amount.toFixed(index === 0 ? 0 : 2))} ${units[index]}`
}

export const shortSHA256 = (value) => {
  const digest = String(value || '')
  if (!digest) return '-'
  if (digest.length <= 13) return digest
  return `${digest.slice(0, 8)}…${digest.slice(-4)}`
}

export const sourceView = (source) => ({
  ACTIVE: { label: '主动采集', type: 'primary' },
  PERIODIC: { label: '周期上传', type: 'success' }
}[source] || { label: source || '未知', type: 'info' })

export const statusView = (status) => ({
  AVAILABLE: { label: '可下载', type: 'success' },
  RECEIVING: { label: '接收中', type: 'primary' },
  FAILED: { label: '接收失败', type: 'danger' },
  DELETING: { label: '清理中', type: 'warning' },
  DELETED: { label: '已清理', type: 'info' }
}[status] || { label: status || '未知', type: 'info' })

export const canDownload = row => row?.status === 'AVAILABLE'

export const filenameFromDisposition = (disposition, fallback = 'artifact.bin') => {
  const value = String(disposition || '')
  const encoded = value.match(/filename\*\s*=\s*UTF-8''([^;]+)/i)?.[1]
  if (encoded) {
    try {
      return decodeURIComponent(encoded.replace(/^"|"$/g, ''))
    } catch {
      // Fall through to the ordinary filename and then the fallback.
    }
  }
  const ordinary = value.match(/filename\s*=\s*(?:"([^"]+)"|([^;]+))/i)
  return (ordinary?.[1] || ordinary?.[2] || fallback).trim()
}

export const triggerBlobDownload = (blob, filename, urlAPI = URL, documentAPI = document) => {
  const objectURL = urlAPI.createObjectURL(blob)
  const anchor = documentAPI.createElement('a')
  try {
    anchor.href = objectURL
    anchor.download = filename
    documentAPI.body.appendChild(anchor)
    anchor.click()
  } finally {
    anchor.remove()
    urlAPI.revokeObjectURL(objectURL)
  }
}
