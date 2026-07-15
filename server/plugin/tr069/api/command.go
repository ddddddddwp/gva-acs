package api

import (
	"errors"
	"strconv"

	"github.com/ddddddddwp/gva-acs/server/model/common/response"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/adapter"
	req "github.com/ddddddddwp/gva-acs/server/plugin/tr069/model/request"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/service"
	"github.com/gin-gonic/gin"
)

type CommandApi struct{}

var commandService = service.NewCommandService(service.NewCommandManager(nil, adapter.EnqueueImmediate))

func commandFailureMessage(err error) string {
	switch {
	case errors.Is(err, service.ErrDeviceOffline):
		return "设备离线，无法下发任务"
	case errors.Is(err, service.ErrCommandQueueUnavailable):
		return "命令队列不可用"
	default:
		return "任务下发失败: " + err.Error()
	}
}

// SyncRPCMethods
// @Tags TR069
// @Summary 下发 GetRPCMethods
// @Security ApiKeyAuth
// @accept application/json
// @Produce application/json
// @Param deviceId path int true "设备ID"
// @Success 200 {object} response.Response{data=map[string]string,msg=string} "下发成功"
// @Router /tr069/command/{deviceId}/getRPCMethods [post]
func (a *CommandApi) SyncRPCMethods(c *gin.Context) {
	deviceId, _ := strconv.Atoi(c.Param("deviceId"))
	result, err := commandService.Submit(c.Request.Context(), uint(deviceId), "GetRPCMethods", nil)
	if err != nil {
		response.FailWithMessage(commandFailureMessage(err), c)
		return
	}
	response.OkWithDetailed(result, "任务已下发", c)
}

// GetParameterValues
// @Tags TR069
// @Summary 下发 GetParameterValues
// @Security ApiKeyAuth
// @accept application/json
// @Produce application/json
// @Param deviceId path int true "设备ID"
// @Param data body request.GetParameterValuesRequest true "查询参数列表"
// @Success 200 {object} response.Response{data=map[string]string,msg=string} "下发成功"
// @Router /tr069/command/{deviceId}/getParameterValues [post]
func (a *CommandApi) GetParameterValues(c *gin.Context) {
	deviceId, _ := strconv.Atoi(c.Param("deviceId"))
	var in req.GetParameterValuesRequest
	if err := c.ShouldBindJSON(&in); err != nil {
		response.FailWithMessage("参数错误", c)
		return
	}
	result, err := commandService.Submit(c.Request.Context(), uint(deviceId), "GetParameterValues", in)
	if err != nil {
		response.FailWithMessage(commandFailureMessage(err), c)
		return
	}
	response.OkWithDetailed(result, "任务已下发", c)
}

// SetParameterValues
// @Tags TR069
// @Summary 下发 SetParameterValues
// @Security ApiKeyAuth
// @accept application/json
// @Produce application/json
// @Param deviceId path int true "设备ID"
// @Param data body request.SetParameterValuesRequest true "设置参数列表"
// @Success 200 {object} response.Response{data=map[string]string,msg=string} "下发成功"
// @Router /tr069/command/{deviceId}/setParameterValues [post]
func (a *CommandApi) SetParameterValues(c *gin.Context) {
	deviceId, _ := strconv.Atoi(c.Param("deviceId"))
	var in req.SetParameterValuesRequest
	if err := c.ShouldBindJSON(&in); err != nil {
		response.FailWithMessage("参数错误", c)
		return
	}
	result, err := commandService.Submit(c.Request.Context(), uint(deviceId), "SetParameterValues", in)
	if err != nil {
		response.FailWithMessage(commandFailureMessage(err), c)
		return
	}
	response.OkWithDetailed(result, "任务已下发", c)
}
