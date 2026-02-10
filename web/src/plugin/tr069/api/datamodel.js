import service from '@/utils/request'

export const getDataModelStructure = (deviceId) => {
  return service({
    url: `/tr069/datamodel/${deviceId}/structure`,
    method: 'get'
  })
}

export const getDataModelList = (deviceId, params) => {
  return service({
    url: `/tr069/datamodel/${deviceId}/list`,
    method: 'get',
    params
  })
}
