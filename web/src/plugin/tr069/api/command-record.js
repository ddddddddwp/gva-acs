import service from '@/utils/request'

export const getCommandRecordList = (params) => service({
  url: '/tr069/command-record/list',
  method: 'get',
  params
})

export const getCommandRecordDetail = (commandId) => service({
  url: `/tr069/command-record/${commandId}`,
  method: 'get'
})

export const retryCommandRecord = (commandId) => service({
  url: `/tr069/command-record/${commandId}/retry`,
  method: 'post'
})
