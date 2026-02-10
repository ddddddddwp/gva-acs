package api

import (
	"strconv"

	"github.com/ddddddddwp/gva-acs/server/model/common/response"
	req "github.com/ddddddddwp/gva-acs/server/plugin/tr069/model/request"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/service"
	"github.com/gin-gonic/gin"
)

type CommandApi struct{}

var commandService = new(service.CommandService)

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
	cmdID, err := commandService.EnqueueGetRPCMethods(uint(deviceId))
	if err != nil {
		response.FailWithMessage("任务下发失败", c)
		return
	}
	response.OkWithDetailed(map[string]string{"commandId": cmdID}, "任务已下发", c)
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
	cmdID, err := commandService.EnqueueGetParameterValues(uint(deviceId), in)
	if err != nil {
		response.FailWithMessage("任务下发失败", c)
		return
	}
	response.OkWithDetailed(map[string]string{"commandId": cmdID}, "任务已下发", c)
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
	cmdID, err := commandService.EnqueueSetParameterValues(uint(deviceId), in)
	if err != nil {
		response.FailWithMessage("任务下发失败", c)
		return
	}
	response.OkWithDetailed(map[string]string{"commandId": cmdID}, "任务已下发", c)
}

