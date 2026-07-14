import service from '@/utils/request'
import { deviceParameterSyncPayload } from '../utils/device-actions'

/**
 * 获取设备支持的RPC方法列表
 * @param {number} deviceId 设备ID
 * @returns {Promise} RPC方法列表
 */
export const getRPCMethods = (deviceId) => {
  return service({
    url: `/tr069/command/${deviceId}/getRPCMethods`,
    method: 'post'
  })
}

/**
 * 获取设备参数值 (GetParameterValues)
 * @param {number} deviceId 设备ID
 * @param {Object} data 请求参数
 * @param {string[]} data.paths 参数路径列表
 * @returns {Promise} 参数值结果
 */
export const getParameterValues = (deviceId, data) => {
  return service({
    url: `/tr069/command/${deviceId}/getParameterValues`,
    method: 'post',
    data
  })
}

/**
 * 设置设备参数值 (SetParameterValues)
 * @param {number} deviceId 设备ID
 * @param {Object} data 设置参数
 * @param {string} data.parameterKey 参数键
 * @param {Array} data.parameters 参数列表
 * @returns {Promise} 设置结果
 */
export const setParameterValues = (deviceId, data) => {
  return service({
    url: `/tr069/command/${deviceId}/setParameterValues`,
    method: 'post',
    data
  })
}

/**
 * 全量数据模型同步
 * @param {number} deviceId 设备ID
 * @returns {Promise} 同步结果
 */
export const fullDataModelSync = (deviceId) => {
  return service({
    url: `/tr069/datamodel/${deviceId}/sync`,
    method: 'post',
    data: deviceParameterSyncPayload()
  })
}
