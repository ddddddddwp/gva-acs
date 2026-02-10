import service from '@/utils/request'

export const getRPCMethods = (deviceId) => {
  return service({
    url: `/tr069/command/${deviceId}/getRPCMethods`,
    method: 'post'
  })
}

export const getParameterValues = (deviceId, data) => {
  return service({
    url: `/tr069/command/${deviceId}/getParameterValues`,
    method: 'post',
    data
  })
}

export const setParameterValues = (deviceId, data) => {
  return service({
    url: `/tr069/command/${deviceId}/setParameterValues`,
    method: 'post',
    data
  })
}

export const fullDataModelSync = (deviceId, data) => {
  return service({
    url: `/tr069/datamodel/${deviceId}/sync`,
    method: 'post',
    data
  })
}

