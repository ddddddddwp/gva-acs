import service from '@/utils/request'

/**
 * 获取设备列表
 * @param {Object} params 查询参数
 * @param {number} params.page 页码
 * @param {number} params.pageSize 每页数量
 * @param {string} params.serialNumber 序列号（可选）
 * @returns {Promise} 设备列表数据
 */
export const getDeviceList = (params) => {
  return service({
    url: '/tr069/device/list',
    method: 'get',
    params
  })
}

/**
 * 录入设备（白名单）
 * @param {Object} data 设备信息
 * @param {string} data.serialNumber 序列号
 * @param {string} data.oui OUI
 * @param {string} data.remark 备注（可选）
 * @returns {Promise} 创建结果
 */
export const createDevice = (data) => {
  return service({
    url: '/tr069/device',
    method: 'post',
    data
  })
}

/**
 * 删除设备
 * @param {number} deviceId 设备ID
 * @returns {Promise} 删除结果
 */
export const deleteDevice = (deviceId) => {
  return service({
    url: `/tr069/device/${deviceId}`,
    method: 'delete'
  })
}
