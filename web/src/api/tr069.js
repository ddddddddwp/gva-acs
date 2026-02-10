import service from '@/utils/request'

// @Tags TR069
// @Summary 分页获取TR069设备列表
// @Security ApiKeyAuth
// @accept application/json
// @Produce application/json
// @Param page query int true "页码"
// @Param pageSize query int true "每页数量"
// @Param serialNumber query string false "序列号"
// @Router /tr069/device/list [get]
export const getTR069DeviceList = (params) => {
  return service({
    url: '/tr069/device/list',
    method: 'get',
    params
  })
}

// @Tags TR069
// @Summary 下发 GetRPCMethods
// @Security ApiKeyAuth
// @accept application/json
// @Produce application/json
// @Param deviceId path int true "设备ID"
// @Router /tr069/command/{deviceId}/getRPCMethods [post]
export const tr069GetRPCMethods = (deviceId) => {
  return service({
    url: `/tr069/command/${deviceId}/getRPCMethods`,
    method: 'post'
  })
}

// @Tags TR069
// @Summary 下发 GetParameterValues
// @Security ApiKeyAuth
// @accept application/json
// @Produce application/json
// @Param deviceId path int true "设备ID"
// @Param data body {paths:["string"]} true "参数路径列表"
// @Router /tr069/command/{deviceId}/getParameterValues [post]
export const tr069GetParameterValues = (deviceId, data) => {
  return service({
    url: `/tr069/command/${deviceId}/getParameterValues`,
    method: 'post',
    data
  })
}

// @Tags TR069
// @Summary 下发 SetParameterValues
// @Security ApiKeyAuth
// @accept application/json
// @Produce application/json
// @Param deviceId path int true "设备ID"
// @Param data body {parameterKey:"string",parameters:[{name:"string",value:"any",type:"string"}]} true "设置参数列表"
// @Router /tr069/command/{deviceId}/setParameterValues [post]
export const tr069SetParameterValues = (deviceId, data) => {
  return service({
    url: `/tr069/command/${deviceId}/setParameterValues`,
    method: 'post',
    data
  })
}

// @Tags TR069
// @Summary 全量同步数据模型
// @Security ApiKeyAuth
// @accept application/json
// @Produce application/json
// @Param deviceId path int true "设备ID"
// @Param data body {maxDepth:16} false "同步参数"
// @Router /tr069/datamodel/{deviceId}/sync [post]
export const tr069FullDataModelSync = (deviceId, data) => {
  return service({
    url: `/tr069/datamodel/${deviceId}/sync`,
    method: 'post',
    data
  })
}

// @Tags TR069
// @Summary 查询设备的数据模型结构（仅对象路径）
// @Security ApiKeyAuth
// @accept application/json
// @Produce application/json
// @Param deviceId path int true "设备ID"
// @Router /tr069/datamodel/{deviceId}/structure [get]
export const tr069GetDataModelStructure = (deviceId) => {
  return service({
    url: `/tr069/datamodel/${deviceId}/structure`,
    method: 'get'
  })
}

// @Tags TR069
// @Summary 查询设备指定路径下的参数值
// @Security ApiKeyAuth
// @accept application/json
// @Produce application/json
// @Param deviceId path int true "设备ID"
// @Param prefix query string false "参数名前缀"
// @Router /tr069/datamodel/{deviceId}/list [get]
export const tr069GetDataModelList = (deviceId, params) => {
  return service({
    url: `/tr069/datamodel/${deviceId}/list`,
    method: 'get',
    params
  })
}

