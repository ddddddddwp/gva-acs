package api

import (
	"github.com/ddddddddwp/gva-acs/server/model/common/response"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/model"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/service"
	"github.com/gin-gonic/gin"
	"strconv"
)

type FAPApi struct{}

var fapService = new(service.FAPService)

// GetFAPInfo
// @Tags TR069
// @Summary 获取基站无线参数
// @Security ApiKeyAuth
// @accept application/json
// @Produce application/json
// @Param deviceId path int true "设备ID"
// @Success 200 {object} response.Response{data=model.FAPService,msg=string} "获取成功"
// @Router /tr069/fap/{deviceId} [get]
func (a *FAPApi) GetFAPInfo(c *gin.Context) {
	deviceIdStr := c.Param("deviceId")
	deviceId, _ := strconv.Atoi(deviceIdStr)

	data, err := fapService.GetFAPInfo(uint(deviceId))
	if err != nil {
		response.FailWithMessage("获取失败", c)
		return
	}
	response.OkWithData(data, c)
}

// SyncFAPInfo
// @Tags TR069
// @Summary 同步基站参数(向设备发请求)
// @Security ApiKeyAuth
// @accept application/json
// @Produce application/json
// @Param deviceId path int true "设备ID"
// @Success 200 {object} response.Response{msg=string} "任务下发成功"
// @Router /tr069/fap/{deviceId}/sync [post]
func (a *FAPApi) SyncFAPInfo(c *gin.Context) {
	deviceIdStr := c.Param("deviceId")
	deviceId, _ := strconv.Atoi(deviceIdStr)

	err := fapService.SyncFAPInfo(uint(deviceId))
	if err != nil {
		response.FailWithMessage("同步任务创建失败", c)
		return
	}
	response.OkWithMessage("同步任务已下发，请稍后刷新", c)
}

// ConfigureFAP
// @Tags TR069
// @Summary 配置基站参数
// @Security ApiKeyAuth
// @accept application/json
// @Produce application/json
// @Param deviceId path int true "设备ID"
// @Param data body model.FAPService true "配置参数"
// @Success 200 {object} response.Response{msg=string} "配置下发成功"
// @Router /tr069/fap/{deviceId} [put]
func (a *FAPApi) ConfigureFAP(c *gin.Context) {
	deviceIdStr := c.Param("deviceId")
	deviceId, _ := strconv.Atoi(deviceIdStr)

	var config model.FAPService
	_ = c.ShouldBindJSON(&config)

	err := fapService.UpdateFAPConfig(uint(deviceId), config)
	if err != nil {
		response.FailWithMessage("配置下发失败", c)
		return
	}
	response.OkWithMessage("配置任务已下发", c)
}
