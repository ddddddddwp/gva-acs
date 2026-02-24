import service from '@/utils/request'

/**
 * 获取基站无线参数 (TR-196)
 * @param {number} deviceId 设备ID
 * @returns {Promise} 基站无线参数信息
 */
export const getFAPInfo = (deviceId) => {
  return service({
    url: `/tr069/fap/${deviceId}`,
    method: 'get'
  })
}

/**
 * 同步基站参数（下发任务）
 * @param {number} deviceId 设备ID
 * @returns {Promise} 同步任务结果
 */
export const syncFAPInfo = (deviceId) => {
  return service({
    url: `/tr069/fap/${deviceId}/sync`,
    method: 'post'
  })
}

/**
 * 配置基站参数
 * @param {number} deviceId 设备ID
 * @param {Object} data 配置参数
 * @param {number} data.pci 物理小区ID
 * @param {string} data.txPower 发射功率
 * @returns {Promise} 配置结果
 */
export const configureFAP = (deviceId, data) => {
  return service({
    url: `/tr069/fap/${deviceId}`,
    method: 'put',
    data
  })
}
