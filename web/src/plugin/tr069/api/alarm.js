import service from '@/utils/request'

export const getActiveAlarms = (params) => {
  return service({
    url: '/tr069/alarm/list',
    method: 'get',
    params
  })
}

export const getHistoryAlarms = (params) => {
  return service({
    url: '/tr069/alarm/history',
    method: 'get',
    params
  })
}

export const getAlarmStats = () => {
  return service({
    url: '/tr069/alarm/stats',
    method: 'get'
  })
}
