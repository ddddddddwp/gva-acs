import service from '@/utils/request'

export const getLogArtifactList = (params) => service({
  url: '/tr069/artifact/list',
  method: 'get',
  params
})

export const downloadLogArtifact = (fileId) => service({
  url: `/tr069/artifact/${encodeURIComponent(fileId)}/download`,
  method: 'get',
  responseType: 'blob',
  donNotShowLoading: true
})
