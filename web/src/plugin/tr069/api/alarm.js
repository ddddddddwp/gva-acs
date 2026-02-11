import service from '@/utils/request'

// @Tags TR069
// @Summary 分页获取告警列表
// @Security ApiKeyAuth
// @accept application/json
// @Produce application/json
// @Param page query int true "页码"
// @Param pageSize query int true "每页数量"
// @Param serialNumber query string false "序列号"
// @Param status query string false "状态"
// @Param severity query string false "级别"
// @Router /tr069/alarm/list [get]
export const getAlarmList = (params) => {
  return service({
    url: '/tr069/alarm/list',
    method: 'get',
    params
  })
}
