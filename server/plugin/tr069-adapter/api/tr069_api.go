package api

import (
	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	"github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-adapter/model/request"
	"github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-adapter/service"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"io/ioutil"
	"strconv"
)

type TR069Api struct{}

// GetDeviceList 获取设备列表
// @Tags     TR069
// @Summary  获取TR069设备列表
// @Security ApiKeyAuth
// @accept   application/json
// @Produce  application/json
// @Param    data query request.TR069DeviceSearch true "分页、筛选条件"
// @Success  200  {object} response.Response{data=response.PageResult,msg=string} "成功"
// @Router   /tr069-adapter/device/list [get]
func (api *TR069Api) GetDeviceList(c *gin.Context) {
	var pageInfo request.TR069DeviceSearch
	err := c.ShouldBindQuery(&pageInfo)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if pageInfo.Page == 0 {
		pageInfo.Page = 1
	}
	if pageInfo.PageSize == 0 {
		pageInfo.PageSize = 10
	}
	list, total, err := service.ServiceGroupApp.GetDeviceList(pageInfo)
	if err != nil {
		global.GVA_LOG.Error("获取设备列表失败!", zap.Error(err))
		response.FailWithMessage("获取设备列表失败", c)
		return
	}
	response.OkWithDetailed(response.PageResult{
		List:     list,
		Total:    total,
		Page:     pageInfo.Page,
		PageSize: pageInfo.PageSize,
	}, "获取成功", c)
}

// GetDeviceByID 根据ID获取设备
// @Tags     TR069
// @Summary  根据ID获取TR069设备
// @Security ApiKeyAuth
// @accept   application/json
// @Produce  application/json
// @Param    id path int true "设备ID"
// @Success  200  {object} response.Response{data=model.TR069Device,msg=string} "成功"
// @Router   /tr069-adapter/device/{id} [get]
func (api *TR069Api) GetDeviceByID(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	device, err := service.ServiceGroupApp.GetDeviceByID(uint(id))
	if err != nil {
		global.GVA_LOG.Error("获取设备失败!", zap.Error(err))
		response.FailWithMessage("获取设备失败", c)
		return
	}
	response.OkWithDetailed(device, "获取成功", c)
}

// DeleteDevice 删除设备
// @Tags     TR069
// @Summary  删除TR069设备
// @Security ApiKeyAuth
// @accept   application/json
// @Produce  application/json
// @Param    id path int true "设备ID"
// @Success  200  {object} response.Response{msg=string} "成功"
// @Router   /tr069-adapter/device/{id} [delete]
func (api *TR069Api) DeleteDevice(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	err := service.ServiceGroupApp.DeleteDevice(uint(id))
	if err != nil {
		global.GVA_LOG.Error("删除设备失败!", zap.Error(err))
		response.FailWithMessage("删除设备失败", c)
		return
	}
	response.OkWithMessage("删除成功", c)
}

// GetDeviceEvents 获取设备事件
// @Tags     TR069
// @Summary  获取设备事件
// @Security ApiKeyAuth
// @accept   application/json
// @Produce  application/json
// @Param    id path int true "设备ID"
// @Param    limit query int false "限制数量"
// @Success  200  {object} response.Response{data=[]model.TR069Event,msg=string} "成功"
// @Router   /tr069-adapter/device/{id}/events [get]
func (api *TR069Api) GetDeviceEvents(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	limitStr := c.DefaultQuery("limit", "10")
	limit, _ := strconv.Atoi(limitStr)

	events, err := service.ServiceGroupApp.GetDeviceEvents(uint(id), limit)
	if err != nil {
		global.GVA_LOG.Error("获取设备事件失败!", zap.Error(err))
		response.FailWithMessage("获取设备事件失败", c)
		return
	}
	response.OkWithDetailed(events, "获取成功", c)
}

// SetDeviceParameters 设置设备参数
// @Tags     TR069
// @Summary  设置设备参数
// @Security ApiKeyAuth
// @accept   application/json
// @Produce  application/json
// @Param    data body request.TR069DeviceParam true "设备参数设置请求"
// @Success  200  {object} response.Response{msg=string} "成功"
// @Router   /tr069-adapter/device/param [post]
func (api *TR069Api) SetDeviceParameters(c *gin.Context) {
	var req request.TR069DeviceParam
	err := c.ShouldBindJSON(&req)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	err = service.ServiceGroupApp.SetDeviceParameters(req)
	if err != nil {
		global.GVA_LOG.Error("设置设备参数失败!", zap.Error(err))
		response.FailWithMessage("设置设备参数失败: "+err.Error(), c)
		return
	}
	response.OkWithMessage("参数设置请求已提交", c)
}

// GetDeviceStats 获取设备统计信息
// @Tags     TR069
// @Summary  获取设备统计信息
// @Security ApiKeyAuth
// @accept   application/json
// @Produce  application/json
// @Success  200  {object} response.Response{data=response.TR069DeviceStats,msg=string} "成功"
// @Router   /tr069-adapter/device/stats [get]
func (api *TR069Api) GetDeviceStats(c *gin.Context) {
	stats, err := service.ServiceGroupApp.GetDeviceStats()
	if err != nil {
		global.GVA_LOG.Error("获取设备统计信息失败!", zap.Error(err))
		response.FailWithMessage("获取设备统计信息失败", c)
		return
	}
	response.OkWithDetailed(stats, "获取成功", c)
}

// HandleTR069Request 处理TR069请求
// 注意：此接口不需要认证，由TR069服务器直接调用
func (api *TR069Api) HandleTR069Request(c *gin.Context) {
	// 读取请求体
	body, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		global.GVA_LOG.Error("读取请求体失败!", zap.Error(err))
		c.Status(400)
		return
	}

	// 处理TR069请求
	resp, err := service.ServiceGroupApp.ProcessInform(c, body)
	if err != nil {
		global.GVA_LOG.Error("处理TR069请求失败!", zap.Error(err))
		c.Status(500)
		return
	}

	// 设置响应头
	c.Header("Content-Type", "text/xml; charset=utf-8")
	c.Writer.Write(resp)
}
