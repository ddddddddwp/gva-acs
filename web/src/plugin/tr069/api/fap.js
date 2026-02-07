import service from '@/utils/request'

// 获取基站无线参数
export const getFAPInfo = (deviceId) => {
  return service({
    url: `/tr069/fap/${deviceId}`,
    method: 'get'
  })
}

// 同步基站参数 (下发任务)
export const syncFAPInfo = (deviceId) => {
  return service({
    url: `/tr069/fap/${deviceId}/sync`,
    method: 'post'
  })
}

// 配置基站参数
export const configureFAP = (deviceId, data) => {
  return service({
    url: `/tr069/fap/${deviceId}`,
    method: 'put',
    data
  })
}
