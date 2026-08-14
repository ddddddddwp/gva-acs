export const canIssueDeviceCommand = (row) => row?.online === true

export const deviceParameterSyncPayload = () => ({ paths: ['Device.'] })

export const RPC_ACTION_GROUPS = [
  {
    label: '查询',
    actions: [
      { key: 'getRPCMethods', label: '查询设备能力', method: 'GetRPCMethods', confirm: 'none' },
      { key: 'getParameterValues', label: '获取参数', method: 'GetParameterValues', confirm: 'none' },
      { key: 'getParameterNames', label: '获取参数名称', method: 'GetParameterNames', confirm: 'none' },
      { key: 'getParameterAttributes', label: '获取参数属性', method: 'GetParameterAttributes', confirm: 'none' }
    ]
  },
  {
    label: '配置',
    actions: [
      { key: 'setParameterValues', label: '配置参数', method: 'SetParameterValues', confirm: 'normal' },
      { key: 'setParameterAttributes', label: '配置参数属性', method: 'SetParameterAttributes', confirm: 'normal' },
      { key: 'addObject', label: '添加对象', method: 'AddObject', confirm: 'normal' },
      { key: 'deleteObject', label: '删除对象实例', method: 'DeleteObject', confirm: 'danger' }
    ]
  },
  {
    label: '文件',
    actions: [
      { key: 'download', label: '下载文件', method: 'Download', confirm: 'normal' },
      { key: 'upload', label: '上传文件', method: 'Upload', confirm: 'normal' }
    ]
  },
  {
    label: '维护',
    actions: [
      { key: 'reboot', label: '重启设备', method: 'Reboot', confirm: 'danger' },
      { key: 'factoryReset', label: '恢复出厂设置', method: 'FactoryReset', confirm: 'danger' }
    ]
  }
]

export const RPC_ACTION_MENUS = [
  {
    key: 'query-config',
    label: '查询与配置',
    groups: RPC_ACTION_GROUPS.slice(0, 2)
  },
  {
    key: 'file-maintenance',
    label: '文件与维护',
    groups: RPC_ACTION_GROUPS.slice(2, 4)
  }
]

const RPC_ACTIONS = RPC_ACTION_GROUPS.flatMap(group => group.actions)

export const findRPCAction = (key) => RPC_ACTIONS.find(action => action.key === key)

export const canIssueRPCAction = (row, action) => {
  if (!canIssueDeviceCommand(row) || !action) return false
  if (action.method === 'GetRPCMethods') return true
  return Array.isArray(row.rpcMethods) && row.rpcMethods.includes(action.method)
}
