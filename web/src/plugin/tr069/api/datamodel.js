import service from '@/utils/request'

export const getDataModelStructure = (deviceId) => {
  return service({
    url: `/tr069/datamodel/${deviceId}/structure`,
    method: 'get',
    params: { _t: new Date().getTime() } // Prevent caching
  })
}

export const getDataModelList = (deviceId, params) => {
  return service({
    url: `/tr069/datamodel/${deviceId}/list`,
    method: 'get',
    params: { ...params, _t: new Date().getTime() } // Prevent caching
  })
}
