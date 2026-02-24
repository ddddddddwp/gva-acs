import service from '@/utils/request'

/**
 * 获取设备数据模型结构
 * @param {number} deviceId 设备ID
 * @returns {Promise} 数据模型结构
 */
export const getDataModelStructure = (deviceId) => {
  return service({
    url: `/tr069/datamodel/${deviceId}/structure`,
    method: 'get',
    params: { _t: new Date().getTime() }
  })
}

/**
 * 获取设备数据模型列表
 * @param {number} deviceId 设备ID
 * @param {Object} params 查询参数
 * @param {string} params.path 路径过滤（可选）
 * @param {string} params.search 搜索关键字（可选）
 * @returns {Promise} 数据模型列表
 */
export const getDataModelList = (deviceId, params) => {
  return service({
    url: `/tr069/datamodel/${deviceId}/list`,
    method: 'get',
    params: { ...params, _t: new Date().getTime() }
  })
}
