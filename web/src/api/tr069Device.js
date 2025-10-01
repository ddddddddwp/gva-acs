import service from '@/utils/request'

// 获取TR069设备列表
export const getDeviceList = (data) => {
  return service({
    url: '/tr069-adapter/device/list',
    method: 'get',
    params: data
  })
}

// 根据ID获取TR069设备详情
export const getDeviceById = (id) => {
  return service({
    url: `/tr069-adapter/device/${id}`,
    method: 'get'
  })
}

// 根据序列号获取TR069设备详情
export const getDeviceBySerialNumber = (serialNumber) => {
  return service({
    url: '/tr069-adapter/device/search',
    method: 'get',
    params: { serialNumber }
  })
}

// 删除TR069设备
export const deleteDevice = (id) => {
  return service({
    url: `/tr069-adapter/device/${id}`,
    method: 'delete'
  })
}

// 获取TR069设备事件列表
export const getDeviceEvents = (id, data) => {
  return service({
    url: `/tr069-adapter/device/${id}/events`,
    method: 'get',
    params: data
  })
}

// 设置TR069设备参数
export const setDeviceParameters = (data) => {
  return service({
    url: '/tr069-adapter/device/param',
    method: 'post',
    data
  })
}

// 获取TR069设备统计信息
export const getDeviceStats = (id) => {
  return service({
    url: `/tr069-adapter/device/${id}/stats`,
    method: 'get'
  })
}