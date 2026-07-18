const STATUS_VIEWS = {
  QUEUED: { label: '排队中', type: 'info' },
  WAITING_DEVICE: { label: '等待设备', type: 'info' },
  BUILDING: { label: '构造中', type: 'warning' },
  SENT: { label: '已发送', type: 'warning' },
  WAITING_TRANSFER: { label: '等待传输完成', type: 'warning' },
  WAITING_REBOOT: { label: '设备已受理，等待重启', type: 'warning' },
  COMPLETED: { label: '完成', type: 'success' },
  FAILED: { label: '失败', type: 'danger' },
  TIMEOUT: { label: '超时', type: 'danger' }
}

const OPERATION_LABELS = {
  GetRPCMethods: '查询设备能力',
  GetParameterValues: '获取参数',
  GetParameterNames: '获取参数名称',
  GetParameterAttributes: '获取参数属性',
  SetParameterValues: '配置参数',
  SetParameterAttributes: '配置参数属性',
  AddObject: '添加对象',
  DeleteObject: '删除对象',
  Download: '下载文件',
  Upload: '上传文件',
  Reboot: '重启设备',
  FactoryReset: '恢复出厂设置'
}

export const statusView = (status) => STATUS_VIEWS[status] || { label: status || '未知', type: 'info' }

export const operationLabel = (operation) => OPERATION_LABELS[operation] || operation || '-'

export const operationOptions = Object.entries(OPERATION_LABELS).map(([value, label]) => ({ value, label }))

export const statusOptions = Object.entries(STATUS_VIEWS).map(([value, view]) => ({ value, label: view.label }))

export const canRetryRecord = (record) => ['FAILED', 'TIMEOUT'].includes(record?.status)

export const isDangerousOperation = (operation) => ['DeleteObject', 'Reboot', 'FactoryReset'].includes(operation)

export const cwmpIDDisplay = (command) => {
  if (command?.cwmpId) return command.cwmpId
  if (command?.status === 'FAILED' && command?.failureStage === 'core.build') {
    return '未生成（构造失败）'
  }
  return '尚未生成'
}

export const showsCommandKey = (operation) => ['Reboot', 'Download', 'Upload'].includes(operation)

export const formatDuration = (start, end) => {
  if (!start || !end) return '-'
  const duration = Math.max(0, Math.floor((new Date(end).getTime() - new Date(start).getTime()) / 1000))
  if (!Number.isFinite(duration)) return '-'
  if (duration < 60) return `${duration}秒`
  const minutes = Math.floor(duration / 60)
  const seconds = duration % 60
  if (minutes < 60) return `${minutes}分${seconds ? `${seconds}秒` : ''}`
  const hours = Math.floor(minutes / 60)
  const remainingMinutes = minutes % 60
  return `${hours}时${remainingMinutes ? `${remainingMinutes}分` : ''}`
}

export const formatJSON = (value) => {
  if (value === null || value === undefined || value === '') return '-'
  try {
    const parsed = typeof value === 'string' ? JSON.parse(value) : value
    return JSON.stringify(parsed, null, 2)
  } catch {
    return String(value)
  }
}

export const directionLabel = (direction) => direction === 'outbound' ? 'ACS → 设备' : direction === 'inbound' ? '设备 → ACS' : direction
