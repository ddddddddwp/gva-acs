import service from '@/utils/request'

export const getLogArtifactList = (params) => service({
  url: '/tr069/artifact/list',
  method: 'get',
  params
})

export const downloadLogArtifact = (artifactId) => service({
  url: `/tr069/artifact/${encodeURIComponent(artifactId)}/download`,
  method: 'get',
  responseType: 'blob',
  donNotShowLoading: true
})
