import service from '@/utils/request'

// 设备管理API
export const getDeviceList = (params) => {
  return service({
    url: '/tr069-management/device/list',
    method: 'get',
    params
  })
}

export const getDeviceDetail = (id) => {
  return service({
    url: `/tr069-management/device/detail/${id}`,
    method: 'get'
  })
}

export const createDevice = (data) => {
  return service({
    url: '/tr069-management/device/create',
    method: 'post',
    data
  })
}

export const updateDevice = (data) => {
  return service({
    url: '/tr069-management/device/update',
    method: 'put',
    data
  })
}

export const deleteDevice = (id) => {
  return service({
    url: `/tr069-management/device/delete/${id}`,
    method: 'delete'
  })
}

export const batchDeleteDevices = (data) => {
  return service({
    url: '/tr069-management/device/batchDelete',
    method: 'post',
    data
  })
}

export const getDeviceStats = () => {
  return service({
    url: '/tr069-management/device/stats',
    method: 'get'
  })
}

export const updateDeviceStatus = (data) => {
  return service({
    url: '/tr069-management/device/status',
    method: 'put',
    data
  })
}

// 参数管理API
export const getParametersByDevice = (params) => {
  return service({
    url: '/tr069-management/parameter/list',
    method: 'get',
    params
  })
}

export const getParameterDetail = (id) => {
  return service({
    url: `/tr069-management/parameter/detail/${id}`,
    method: 'get'
  })
}

export const setDeviceParameters = (data) => {
  return service({
    url: '/tr069-management/parameter/set',
    method: 'post',
    data
  })
}

export const getParameterCategories = (params) => {
  return service({
    url: '/tr069-management/parameter/categories',
    method: 'get',
    params
  })
}

// 设备组管理API
export const getGroupList = (params) => {
  return service({
    url: '/tr069-management/group/list',
    method: 'get',
    params
  })
}

export const getGroupDetail = (id) => {
  return service({
    url: `/tr069-management/group/detail/${id}`,
    method: 'get'
  })
}

export const createGroup = (data) => {
  return service({
    url: '/tr069-management/group/create',
    method: 'post',
    data
  })
}

export const updateGroup = (data) => {
  return service({
    url: '/tr069-management/group/update',
    method: 'put',
    data
  })
}

export const deleteGroup = (id) => {
  return service({
    url: `/tr069-management/group/delete/${id}`,
    method: 'delete'
  })
}

export const addDeviceToGroup = (params) => {
  return service({
    url: '/tr069-management/group/addDevice',
    method: 'post',
    params
  })
}

export const removeDeviceFromGroup = (params) => {
  return service({
    url: '/tr069-management/group/removeDevice',
    method: 'delete',
    params
  })
}

// 配置文件管理API
export const getConfigProfileList = (params) => {
  return service({
    url: '/tr069-management/config/list',
    method: 'get',
    params
  })
}

export const getConfigProfileDetail = (id) => {
  return service({
    url: `/tr069-management/config/detail/${id}`,
    method: 'get'
  })
}

export const createConfigProfile = (data) => {
  return service({
    url: '/tr069-management/config/create',
    method: 'post',
    data
  })
}

export const updateConfigProfile = (data) => {
  return service({
    url: '/tr069-management/config/update',
    method: 'put',
    data
  })
}

export const deleteConfigProfile = (id) => {
  return service({
    url: `/tr069-management/config/delete/${id}`,
    method: 'delete'
  })
}

export const applyConfigProfile = (params) => {
  return service({
    url: '/tr069-management/config/apply',
    method: 'post',
    params
  })
}

// 固件管理API
export const getFirmwareList = (params) => {
  return service({
    url: '/tr069-management/firmware/list',
    method: 'get',
    params
  })
}

export const getFirmwareDetail = (id) => {
  return service({
    url: `/tr069-management/firmware/detail/${id}`,
    method: 'get'
  })
}

export const createFirmware = (data) => {
  return service({
    url: '/tr069-management/firmware/create',
    method: 'post',
    data
  })
}

export const updateFirmware = (data) => {
  return service({
    url: '/tr069-management/firmware/update',
    method: 'put',
    data
  })
}

export const deleteFirmware = (id) => {
  return service({
    url: `/tr069-management/firmware/delete/${id}`,
    method: 'delete'
  })
}

export const uploadFirmware = (data) => {
  return service({
    url: '/tr069-management/firmware/upload',
    method: 'post',
    data
  })
}

export const upgradeFirmware = (params) => {
  return service({
    url: '/tr069-management/firmware/upgrade',
    method: 'post',
    params
  })
}

// 设备状态和操作API
export const getDeviceStatus = (id) => {
  return service({
    url: `/tr069-management/device/status/${id}`,
    method: 'get'
  })
}

export const triggerDeviceAction = (data) => {
  return service({
    url: '/tr069-management/device/action',
    method: 'post',
    data
  })
}

export const getDeviceEvents = (params) => {
  return service({
    url: '/tr069-management/device/events',
    method: 'get',
    params
  })
}

export const getDeviceSessions = (params) => {
  return service({
    url: '/tr069-management/device/sessions',
    method: 'get',
    params
  })
}

export const getDeviceOperationLogs = (params) => {
  return service({
    url: '/tr069-management/device/logs',
    method: 'get',
    params
  })
}

// 参数管理API
export const getDeviceParameters = (params) => {
  return service({
    url: '/tr069-management/parameter/list',
    method: 'get',
    params
  })
}

export const setDeviceParameter = (data) => {
  return service({
    url: '/tr069-management/parameter/set',
    method: 'post',
    data
  })
}

// 分组管理API补充
export const getGroupByID = (id) => {
  return service({
    url: `/tr069-management/group/detail/${id}`,
    method: 'get'
  })
}

// 配置管理API补充
export const getConfigProfileByID = (id) => {
  return service({
    url: `/tr069-management/config/detail/${id}`,
    method: 'get'
  })
}