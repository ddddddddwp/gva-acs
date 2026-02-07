import service from '@/utils/request'

// 获取设备列表
export const getDeviceList = (params) => {
  return service({
    url: '/tr069/device/list',
    method: 'get',
    params
  })
}

// 录入设备
export const createDevice = (data) => {
  return service({
    url: '/tr069/device',
    method: 'post',
    data
  })
}

// 删除设备
export const deleteDevice = (deviceId) => {
  return service({
    url: `/tr069/device/${deviceId}`,
    method: 'delete'
  })
}
