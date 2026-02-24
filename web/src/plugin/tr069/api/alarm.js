import service from '@/utils/request'

/**
 * 获取当前告警列表
 * @param {Object} params 查询参数
 * @param {number} params.page 页码
 * @param {number} params.pageSize 每页数量
 * @param {string} params.severity 告警级别（可选）
 * @returns {Promise} 当前告警列表
 */
export const getActiveAlarms = (params) => {
  return service({
    url: '/tr069/alarm/list',
    method: 'get',
    params
  })
}

/**
 * 获取历史告警列表
 * @param {Object} params 查询参数
 * @param {number} params.page 页码
 * @param {number} params.pageSize 每页数量
 * @param {string} params.severity 告警级别（可选）
 * @param {string} params.startTime 开始时间（可选）
 * @param {string} params.endTime 结束时间（可选）
 * @returns {Promise} 历史告警列表
 */
export const getHistoryAlarms = (params) => {
  return service({
    url: '/tr069/alarm/history',
    method: 'get',
    params
  })
}

/**
 * 获取告警统计信息
 * @returns {Promise} 告警统计数据
 */
export const getAlarmStats = () => {
  return service({
    url: '/tr069/alarm/stats',
    method: 'get'
  })
}
