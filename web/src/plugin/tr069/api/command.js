import service from '@/utils/request'
import { deviceParameterSyncPayload } from '../utils/device-actions'

const postCommand = (deviceId, operation, data) => service({
  url: `/tr069/command/${deviceId}/${operation}`,
  method: 'post',
  ...(data === undefined ? {} : { data })
})

/**
 * 获取设备支持的RPC方法列表
 * @param {number} deviceId 设备ID
 * @returns {Promise} RPC方法列表
 */
export const getRPCMethods = (deviceId) => {
  return postCommand(deviceId, 'getRPCMethods')
}

/**
 * 获取设备参数值 (GetParameterValues)
 * @param {number} deviceId 设备ID
 * @param {Object} data 请求参数
 * @param {string[]} data.paths 参数路径列表
 * @returns {Promise} 参数值结果
 */
export const getParameterValues = (deviceId, data) => {
  return postCommand(deviceId, 'getParameterValues', data)
}

export const getParameterNames = (deviceId, data) => postCommand(deviceId, 'getParameterNames', data)

export const getParameterAttributes = (deviceId, data) => postCommand(deviceId, 'getParameterAttributes', data)

/**
 * 设置设备参数值 (SetParameterValues)
 * @param {number} deviceId 设备ID
 * @param {Object} data 设置参数
 * @param {string} data.parameterKey 参数键
 * @param {Array} data.parameters 参数列表
 * @returns {Promise} 设置结果
 */
export const setParameterValues = (deviceId, data) => {
  return postCommand(deviceId, 'setParameterValues', data)
}

export const setParameterAttributes = (deviceId, data) => postCommand(deviceId, 'setParameterAttributes', data)

export const addObject = (deviceId, data) => postCommand(deviceId, 'addObject', data)

export const deleteObject = (deviceId, data) => postCommand(deviceId, 'deleteObject', data)

export const downloadFile = (deviceId, data) => postCommand(deviceId, 'download', data)

export const uploadFile = (deviceId, data) => postCommand(deviceId, 'upload', data)

export const rebootDevice = (deviceId) => postCommand(deviceId, 'reboot')

export const factoryResetDevice = (deviceId) => postCommand(deviceId, 'factoryReset')

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
