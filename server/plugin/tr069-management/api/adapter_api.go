package api

import (
	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	"github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-management/model/request"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type AdapterApi struct{}

// GetDeviceStatus 获取设备在线状态
// @Tags TR069Adapter
// @Summary 获取设备在线状态
// @Security ApiKeyAuth
// @accept application/json
// @Produce application/json
// @Param data body request.DeviceSerialNumberRequest true "设备序列号"
// @Success 200 {object} response.Response{data=bool} "设备在线状态"
// @Router /tr069-management/adapter/deviceStatus [post]
func (a *AdapterApi) GetDeviceStatus(c *gin.Context) {
	var req request.DeviceSerialNumberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage("参数错误", c)
		return
	}
	
	online, err := adapterService.GetDeviceStatus(req.SerialNumber)
	if err != nil {
		global.GVA_LOG.Error("获取设备状态失败", zap.Error(err))
		response.FailWithMessage("获取设备状态失败", c)
		return
	}
	
	response.OkWithData(online, c)
}

// TriggerDeviceAction 触发设备操作
// @Tags TR069Adapter
// @Summary 触发设备操作（如重启、恢复出厂设置等）
// @Security ApiKeyAuth
// @accept application/json
// @Produce application/json
// @Param data body request.DeviceActionRequest true "设备操作请求"
// @Success 200 {object} response.Response{} "操作结果"
// @Router /tr069-management/adapter/triggerAction [post]
func (a *AdapterApi) TriggerDeviceAction(c *gin.Context) {
	var req request.DeviceActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage("参数错误", c)
		return
	}
	
	err := adapterService.TriggerDeviceAction(req.SerialNumber, req.Action)
	if err != nil {
		global.GVA_LOG.Error("触发设备操作失败", zap.Error(err))
		response.FailWithMessage("触发设备操作失败: "+err.Error(), c)
		return
	}
	
	response.OkWithMessage("操作已提交", c)
}

// GetDeviceEvents 获取设备事件记录
// @Tags TR069Adapter
// @Summary 获取设备事件记录
// @Security ApiKeyAuth
// @accept application/json
// @Produce application/json
// @Param data body request.DeviceEventsRequest true "设备事件请求"
// @Success 200 {object} response.Response{data=response.PageResult} "事件列表"
// @Router /tr069-management/adapter/deviceEvents [post]
func (a *AdapterApi) GetDeviceEvents(c *gin.Context) {
	var req request.DeviceEventsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage("参数错误", c)
		return
	}
	
	events, total, err := adapterService.GetDeviceEvents(req.SerialNumber, req.Page, req.PageSize)
	if err != nil {
		global.GVA_LOG.Error("获取设备事件记录失败", zap.Error(err))
		response.FailWithMessage("获取设备事件记录失败", c)
		return
	}
	
	response.OkWithDetailed(response.PageResult{
		List:     events,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}, "获取成功", c)
}

// GetDeviceSessions 获取设备会话记录
// @Tags TR069Adapter
// @Summary 获取设备会话记录
// @Security ApiKeyAuth
// @accept application/json
// @Produce application/json
// @Param data body request.DeviceSessionsRequest true "设备会话请求"
// @Success 200 {object} response.Response{data=response.PageResult} "会话列表"
// @Router /tr069-management/adapter/deviceSessions [post]
func (a *AdapterApi) GetDeviceSessions(c *gin.Context) {
	var req request.DeviceSessionsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage("参数错误", c)
		return
	}
	
	sessions, total, err := adapterService.GetDeviceSessions(req.SerialNumber, req.Page, req.PageSize)
	if err != nil {
		global.GVA_LOG.Error("获取设备会话记录失败", zap.Error(err))
		response.FailWithMessage("获取设备会话记录失败", c)
		return
	}
	
	response.OkWithDetailed(response.PageResult{
		List:     sessions,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}, "获取成功", c)
}

// GetDeviceOperationLogs 获取设备操作日志
// @Tags TR069Adapter
// @Summary 获取设备操作日志
// @Security ApiKeyAuth
// @accept application/json
// @Produce application/json
// @Param data body request.DeviceLogsRequest true "设备日志请求"
// @Success 200 {object} response.Response{data=response.PageResult} "日志列表"
// @Router /tr069-management/adapter/deviceLogs [post]
func (a *AdapterApi) GetDeviceOperationLogs(c *gin.Context) {
	var req request.DeviceLogsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage("参数错误", c)
		return
	}
	
	logs, total, err := adapterService.GetDeviceOperationLogs(req.SerialNumber, req.Page, req.PageSize)
	if err != nil {
		global.GVA_LOG.Error("获取设备操作日志失败", zap.Error(err))
		response.FailWithMessage("获取设备操作日志失败", c)
		return
	}
	
	response.OkWithDetailed(response.PageResult{
		List:     logs,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}, "获取成功", c)
}