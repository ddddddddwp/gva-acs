package api

import (
	"fmt"
	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	"github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-management/model/request"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// DeviceApi 设备API
type DeviceApi struct{}

// GetDeviceList 获取设备列表
// @Tags     TR069Management
// @Summary  获取设备列表
// @Security ApiKeyAuth
// @accept   application/json
// @Produce  application/json
// @Param    data query request.DeviceSearch true "分页、搜索条件"
// @Success  200  {object} response.Response{data=response.PageResult,msg=string} "设备列表,返回包括列表,总数,页码,每页数量"
// @Router   /tr069-management/device/list [get]
func (a *DeviceApi) GetDeviceList(c *gin.Context) {
	var req request.DeviceSearch
	if err := c.ShouldBindQuery(&req); err != nil {
		response.FailWithMessage("参数错误", c)
		return
	}
	if req.Page == 0 {
		req.Page = 1
	}
	if req.PageSize == 0 {
		req.PageSize = 10
	}

	devices, total, err := deviceService.GetDeviceList(req)
	if err != nil {
		global.GVA_LOG.Error("获取设备列表失败", zap.Error(err))
		response.FailWithMessage("获取设备列表失败: "+err.Error(), c)
		return
	}

	response.OkWithDetailed(response.PageResult{
		List:     devices,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}, "获取成功", c)
}

// GetDeviceByID 根据ID获取设备详情
// @Tags     TR069Management
// @Summary  根据ID获取设备详情
// @Security ApiKeyAuth
// @accept   application/json
// @Produce  application/json
// @Param    id path uint true "设备ID"
// @Success  200  {object} response.Response{data=model.TR069Device,msg=string} "设备详情"
// @Router   /tr069-management/device/{id} [get]
func (a *DeviceApi) GetDeviceByID(c *gin.Context) {
	var id uint
	if err := c.ShouldBindUri(struct{ ID uint `uri:"id"` }{ID: id}); err != nil {
		response.FailWithMessage("参数错误", c)
		return
	}

	device, err := deviceService.GetDeviceByID(id)
	if err != nil {
		global.GVA_LOG.Error("获取设备详情失败", zap.Error(err))
		response.FailWithMessage("获取设备详情失败: "+err.Error(), c)
		return
	}

	response.OkWithDetailed(device, "获取成功", c)
}

// CreateDevice 创建设备
// @Tags     TR069Management
// @Summary  创建设备
// @Security ApiKeyAuth
// @accept   application/json
// @Produce  application/json
// @Param    data body request.CreateDeviceRequest true "设备信息"
// @Success  200  {object} response.Response{data=model.TR069Device,msg=string} "创建成功"
// @Router   /tr069-management/device [post]
func (a *DeviceApi) CreateDevice(c *gin.Context) {
	var req request.CreateDeviceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage("参数错误", c)
		return
	}

	device, err := deviceService.CreateDevice(req)
	if err != nil {
		global.GVA_LOG.Error("创建设备失败", zap.Error(err))
		response.FailWithMessage("创建设备失败: "+err.Error(), c)
		return
	}

	response.OkWithDetailed(device, "创建成功", c)
}

// UpdateDevice 更新设备
// @Tags     TR069Management
// @Summary  更新设备
// @Security ApiKeyAuth
// @accept   application/json
// @Produce  application/json
// @Param    data body request.UpdateDeviceRequest true "设备信息"
// @Success  200  {object} response.Response{msg=string} "更新成功"
// @Router   /tr069-management/device [put]
func (a *DeviceApi) UpdateDevice(c *gin.Context) {
	var req request.UpdateDeviceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage("参数错误", c)
		return
	}

	err := deviceService.UpdateDevice(req)
	if err != nil {
		global.GVA_LOG.Error("更新设备失败", zap.Error(err))
		response.FailWithMessage("更新设备失败: "+err.Error(), c)
		return
	}

	response.OkWithMessage("更新成功", c)
}

// DeleteDevice 删除设备
// @Tags     TR069Management
// @Summary  删除设备
// @Security ApiKeyAuth
// @accept   application/json
// @Produce  application/json
// @Param    id path uint true "设备ID"
// @Success  200  {object} response.Response{msg=string} "删除成功"
// @Router   /tr069-management/device/{id} [delete]
func (a *DeviceApi) DeleteDevice(c *gin.Context) {
	var id uint
	if err := c.ShouldBindUri(struct{ ID uint `uri:"id"` }{ID: id}); err != nil {
		response.FailWithMessage("参数错误", c)
		return
	}

	err := deviceService.DeleteDevice(id)
	if err != nil {
		global.GVA_LOG.Error("删除设备失败", zap.Error(err))
		response.FailWithMessage("删除设备失败: "+err.Error(), c)
		return
	}

	response.OkWithMessage("删除成功", c)
}

// BatchDeleteDevices 批量删除设备
// @Tags     TR069Management
// @Summary  批量删除设备
// @Security ApiKeyAuth
// @accept   application/json
// @Produce  application/json
// @Param    data body request.BatchDeleteRequest true "设备ID列表"
// @Success  200  {object} response.Response{msg=string} "删除成功"
// @Router   /tr069-management/device/batch [delete]
func (a *DeviceApi) BatchDeleteDevices(c *gin.Context) {
	var req request.BatchDeleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage("参数错误", c)
		return
	}

	err := deviceService.BatchDeleteDevices(req.IDs)
	if err != nil {
		global.GVA_LOG.Error("批量删除设备失败", zap.Error(err))
		response.FailWithMessage("批量删除设备失败: "+err.Error(), c)
		return
	}

	response.OkWithMessage("删除成功", c)
}

// GetDeviceStats 获取设备统计信息
// @Tags     TR069Management
// @Summary  获取设备统计信息
// @Security ApiKeyAuth
// @accept   application/json
// @Produce  application/json
// @Success  200  {object} response.Response{data=map[string]interface{},msg=string} "统计信息"
// @Router   /tr069-management/device/stats [get]
func (a *DeviceApi) GetDeviceStats(c *gin.Context) {
	stats, err := deviceService.GetDeviceStats()
	if err != nil {
		global.GVA_LOG.Error("获取设备统计信息失败", zap.Error(err))
		response.FailWithMessage("获取设备统计信息失败: "+err.Error(), c)
		return
	}

	response.OkWithDetailed(stats, "获取成功", c)
}

// UpdateDeviceStatus 更新设备状态
// @Tags     TR069Management
// @Summary  更新设备状态
// @Security ApiKeyAuth
// @accept   application/json
// @Produce  application/json
// @Param    id path uint true "设备ID"
// @Param    status query int true "状态值"
// @Success  200  {object} response.Response{msg=string} "更新成功"
// @Router   /tr069-management/device/{id}/status [put]
func (a *DeviceApi) UpdateDeviceStatus(c *gin.Context) {
	var id uint
	if err := c.ShouldBindUri(struct{ ID uint `uri:"id"` }{ID: id}); err != nil {
		response.FailWithMessage("参数错误", c)
		return
	}

	status := c.Query("status")
	if status == "" {
		response.FailWithMessage("状态参数不能为空", c)
		return
	}

	var statusInt int
	if _, err := fmt.Sscanf(status, "%d", &statusInt); err != nil {
		response.FailWithMessage("状态参数格式错误", c)
		return
	}

	err := deviceService.UpdateDeviceStatus(id, statusInt)
	if err != nil {
		global.GVA_LOG.Error("更新设备状态失败", zap.Error(err))
		response.FailWithMessage("更新设备状态失败: "+err.Error(), c)
		return
	}

	response.OkWithMessage("更新成功", c)
}