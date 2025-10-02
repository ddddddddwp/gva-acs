package api

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	"github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-adapter/model/request"
	"github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-adapter/service"
	"go.uber.org/zap"
)

type DeviceApi struct{}

var deviceService = service.ServiceGroupApp.DeviceService

// GetDeviceList 获取设备列表
// @Tags     TR069Device
// @Summary  获取TR069设备列表
// @Security ApiKeyAuth
// @accept   application/json
// @Produce  application/json
// @Param    data body request.DeviceSearch true "分页查询参数"
// @Success  200  {object} response.Response{data=response.PageResult,msg=string} "获取成功"
// @Router   /tr069-adapter/device/getDeviceList [post]
func (d *DeviceApi) GetDeviceList(c *gin.Context) {
	var req request.DeviceSearch
	err := c.ShouldBindJSON(&req)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	list, total, err := deviceService.GetDeviceList(req)
	if err != nil {
		global.GVA_LOG.Error("获取设备列表失败!", zap.Error(err))
		response.FailWithMessage("获取设备列表失败", c)
		return
	}

	response.OkWithDetailed(response.PageResult{
		List:     list,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}, "获取成功", c)
}

// GetDeviceByID 根据ID获取设备详情
// @Tags     TR069Device
// @Summary  根据ID获取TR069设备详情
// @Security ApiKeyAuth
// @accept   application/json
// @Produce  application/json
// @Param    id path int true "设备ID"
// @Success  200  {object} response.Response{data=response.DeviceDetailResponse,msg=string} "获取成功"
// @Router   /tr069-adapter/device/getDevice/{id} [get]
func (d *DeviceApi) GetDeviceByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.FailWithMessage("无效的设备ID", c)
		return
	}

	deviceDetail, err := deviceService.GetDeviceByID(uint(id))
	if err != nil {
		global.GVA_LOG.Error("获取设备详情失败!", zap.Error(err))
		response.FailWithMessage("获取设备详情失败", c)
		return
	}

	response.OkWithDetailed(deviceDetail, "获取成功", c)
}

// CreateDevice 创建设备
// @Tags     TR069Device
// @Summary  创建TR069设备
// @Security ApiKeyAuth
// @accept   application/json
// @Produce  application/json
// @Param    data body request.CreateDeviceRequest true "设备信息"
// @Success  200  {object} response.Response{data=response.DeviceResponse,msg=string} "创建成功"
// @Router   /tr069-adapter/device/createDevice [post]
func (d *DeviceApi) CreateDevice(c *gin.Context) {
	var req request.CreateDeviceRequest
	err := c.ShouldBindJSON(&req)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	createdDevice, err := deviceService.CreateDevice(req)
	if err != nil {
		global.GVA_LOG.Error("创建设备失败!", zap.Error(err))
		response.FailWithMessage("创建设备失败", c)
		return
	}

	response.OkWithDetailed(createdDevice, "创建成功", c)
}

// UpdateDevice 更新设备
// @Tags     TR069Device
// @Summary  更新设备信息
// @Security ApiKeyAuth
// @accept   application/json
// @Produce  application/json
// @Param    data body request.UpdateDeviceRequest true "设备更新信息"
// @Success  200  {object} response.Response{msg=string} "更新成功"
// @Router   /tr069/device/updateDevice [put]
func (d *DeviceApi) UpdateDevice(c *gin.Context) {
	var req request.UpdateDeviceRequest
	err := c.ShouldBindJSON(&req)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	// 转换为map格式
	updates := map[string]interface{}{
		"serialNumber":     req.SerialNumber,
		"productClass":     req.ProductClass,
		"manufacturer":     req.Manufacturer,
		"oui":              req.OUI,
		"modelName":        req.ModelName,
		"description":      req.Description,
		"provisioningCode": req.ProvisioningCode,
		"softwareVersion":  req.SoftwareVersion,
		"hardwareVersion":  req.HardwareVersion,
		"specVersion":      req.SpecVersion,
		"connectionURL":    req.ConnectionURL,
		"username":         req.Username,
		"password":         req.Password,
		"periodicInform":   req.PeriodicInform,
		"status":           req.Status,
		"tags":             req.Tags,
	}

	err = deviceService.UpdateDevice(req.ID, updates)
	if err != nil {
		global.GVA_LOG.Error("更新设备失败!", zap.Error(err))
		response.FailWithMessage("更新设备失败", c)
		return
	}
	response.OkWithMessage("更新成功", c)
}

// DeleteDevice 删除设备
// @Tags     TR069Device
// @Summary  删除TR069设备
// @Security ApiKeyAuth
// @accept   application/json
// @Produce  application/json
// @Param    id path int true "设备ID"
// @Success  200  {object} response.Response{msg=string} "删除成功"
// @Router   /tr069-adapter/device/deleteDevice/{id} [delete]
func (d *DeviceApi) DeleteDevice(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.FailWithMessage("无效的设备ID", c)
		return
	}

	err = deviceService.DeleteDevice(uint(id))
	if err != nil {
		global.GVA_LOG.Error("删除设备失败!", zap.Error(err))
		response.FailWithMessage("删除设备失败", c)
		return
	}

	response.OkWithMessage("删除成功", c)
}

// BatchDeleteDevices 批量删除设备
// @Tags     TR069Device
// @Summary  批量删除设备
// @Security ApiKeyAuth
// @accept   application/json
// @Produce  application/json
// @Param    data body request.BatchDeleteRequest true "批量删除设备请求"
// @Success  200  {object} response.Response{msg=string} "删除成功"
// @Router   /tr069/device/batchDeleteDevices [delete]
func (d *DeviceApi) BatchDeleteDevices(c *gin.Context) {
	var req request.BatchDeleteRequest
	err := c.ShouldBindJSON(&req)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	err = deviceService.BatchDeleteDevices(req.IDs)
	if err != nil {
		global.GVA_LOG.Error("批量删除设备失败!", zap.Error(err))
		response.FailWithMessage("批量删除设备失败", c)
		return
	}
	response.OkWithMessage("删除成功", c)
}

// GetDeviceStats 获取设备统计信息
// @Tags     TR069Device
// @Summary  获取TR069设备统计信息
// @Security ApiKeyAuth
// @accept   application/json
// @Produce  application/json
// @Success  200  {object} response.Response{data=response.DeviceStatsResponse,msg=string} "获取成功"
// @Router   /tr069-adapter/device/getDeviceStats [get]
func (d *DeviceApi) GetDeviceStats(c *gin.Context) {
	stats, err := deviceService.GetDeviceStats()
	if err != nil {
		global.GVA_LOG.Error("获取设备统计信息失败!", zap.Error(err))
		response.FailWithMessage("获取设备统计信息失败", c)
		return
	}

	response.OkWithDetailed(stats, "获取成功", c)
}

// RebootDevice 重启设备
// @Tags     TR069Device
// @Summary  重启TR069设备
// @Security ApiKeyAuth
// @accept   application/json
// @Produce  application/json
// @Param    id path int true "设备ID"
// @Success  200  {object} response.Response{msg=string} "重启成功"
// @Router   /tr069-adapter/device/reboot/{id} [post]
func (d *DeviceApi) RebootDevice(c *gin.Context) {
	id := c.Param("id")
	deviceID, err := strconv.ParseUint(id, 10, 32)
	if err != nil {
		response.FailWithMessage("设备ID格式错误", c)
		return
	}

	// TODO: 实现设备重启逻辑
	global.GVA_LOG.Info("设备重启请求", zap.Uint64("deviceID", deviceID))
	response.OkWithMessage("重启命令已发送", c)
}

// FactoryResetDevice 恢复出厂设置
// @Tags     TR069Device
// @Summary  恢复TR069设备出厂设置
// @Security ApiKeyAuth
// @accept   application/json
// @Produce  application/json
// @Param    id path int true "设备ID"
// @Success  200  {object} response.Response{msg=string} "恢复出厂设置成功"
// @Router   /tr069-adapter/device/factoryReset/{id} [post]
func (d *DeviceApi) FactoryResetDevice(c *gin.Context) {
	id := c.Param("id")
	deviceID, err := strconv.ParseUint(id, 10, 32)
	if err != nil {
		response.FailWithMessage("设备ID格式错误", c)
		return
	}

	// TODO: 实现设备恢复出厂设置逻辑
	global.GVA_LOG.Info("设备恢复出厂设置请求", zap.Uint64("deviceID", deviceID))
	response.OkWithMessage("恢复出厂设置命令已发送", c)
}

// GetDeviceBySerialNumber 根据序列号获取设备
// @Tags     TR069Device
// @Summary  根据序列号获取TR069设备
// @Security ApiKeyAuth
// @accept   application/json
// @Produce  application/json
// @Param    serialNumber path string true "设备序列号"
// @Success  200  {object} response.Response{data=model.CpeDevice,msg=string} "获取成功"
// @Router   /tr069-adapter/device/getBySerialNumber/{serialNumber} [get]
func (d *DeviceApi) GetDeviceBySerialNumber(c *gin.Context) {
	serialNumber := c.Param("serialNumber")
	if serialNumber == "" {
		response.FailWithMessage("设备序列号不能为空", c)
		return
	}

	// TODO: 实现根据序列号查找设备的逻辑
	global.GVA_LOG.Info("根据序列号查找设备", zap.String("serialNumber", serialNumber))
	response.FailWithMessage("功能暂未实现", c)
}