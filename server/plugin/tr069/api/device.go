package api

import (
	"github.com/ddddddddwp/gva-acs/server/global"
	"github.com/ddddddddwp/gva-acs/server/model/common/request"
	"github.com/ddddddddwp/gva-acs/server/model/common/response"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/model"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type DeviceApi struct{}

// GetDeviceList
// @Tags TR069
// @Summary 分页获取设备列表
// @Security ApiKeyAuth
// @accept application/json
// @Produce application/json
// @Param page query int true "页码"
// @Param pageSize query int true "每页数量"
// @Param serialNumber query string false "序列号"
// @Success 200 {object} response.Response{data=response.PageResult,msg=string} "获取成功"
// @Router /tr069/device/list [get]
func (a *DeviceApi) GetDeviceList(c *gin.Context) {
	var pageInfo request.PageInfo
	_ = c.ShouldBindQuery(&pageInfo)

	// Search criteria
	serialNumber := c.Query("serialNumber")

	db := global.GVA_DB.Model(&model.Device{})

	if serialNumber != "" {
		db = db.Where("serial_number LIKE ?", "%"+serialNumber+"%")
	}

	var total int64
	db.Count(&total)

	var devices []model.Device
	limit := pageInfo.PageSize
	offset := pageInfo.PageSize * (pageInfo.Page - 1)

	err := db.Limit(limit).Offset(offset).Find(&devices).Error
	if err != nil {
		response.FailWithMessage("获取设备列表失败", c)
		return
	}

	response.OkWithDetailed(response.PageResult{
		List:     devices,
		Total:    total,
		Page:     pageInfo.Page,
		PageSize: pageInfo.PageSize,
	}, "获取成功", c)
}

// CreateDevice (Whitelist)
// @Tags TR069
// @Summary 录入设备(白名单)
// @Security ApiKeyAuth
// @accept application/json
// @Produce application/json
// @Param data body model.Device true "设备信息"
// @Success 200 {object} response.Response{msg=string} "录入成功"
// @Router /tr069/device [post]
func (a *DeviceApi) CreateDevice(c *gin.Context) {
	var device model.Device
	_ = c.ShouldBindJSON(&device)

	// Force whitelist flag
	device.IsWhite = true

	if err := global.GVA_DB.Create(&device).Error; err != nil {
		global.GVA_LOG.Error("录入设备失败", zap.Error(err))
		response.FailWithMessage("录入设备失败，可能序列号已存在", c)
		return
	}
	response.OkWithMessage("录入成功", c)
}

// DeleteDevice
// @Tags TR069
// @Summary 删除设备
// @Security ApiKeyAuth
// @accept application/json
// @Produce application/json
// @Param deviceId path int true "设备ID"
// @Success 200 {object} response.Response{msg=string} "删除成功"
// @Router /tr069/device/{deviceId} [delete]
func (a *DeviceApi) DeleteDevice(c *gin.Context) {
	deviceId := c.Param("deviceId")

	var device model.Device
	if err := global.GVA_DB.First(&device, deviceId).Error; err != nil {
		response.FailWithMessage("设备不存在", c)
		return
	}

	// Transaction to delete device and related data (alarms, values)
	err := global.GVA_DB.Transaction(func(tx *gorm.DB) error {
		// 1. Delete Alarms
		if err := tx.Where("device_id = ?", device.ID).Delete(&model.Tr069Alarm{}).Error; err != nil {
			return err
		}
		// 2. Delete DataModel Values
		if err := tx.Where("device_id = ?", device.ID).Delete(&model.DataModelValue{}).Error; err != nil {
			return err
		}
		// 3. Delete Device
		if err := tx.Delete(&device).Error; err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		global.GVA_LOG.Error("删除设备失败", zap.Error(err))
		response.FailWithMessage("删除失败", c)
		return
	}
	response.OkWithMessage("删除成功", c)
}
