package api

import (
	"context"
	"encoding/json"
	"time"

	"github.com/ddddddddwp/gva-acs/server/global"
	"github.com/ddddddddwp/gva-acs/server/model/common/request"
	"github.com/ddddddddwp/gva-acs/server/model/common/response"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/model"
	deviceResponse "github.com/ddddddddwp/gva-acs/server/plugin/tr069/model/response"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// 设备离线阈值（秒），超过此时间未收到 Inform 视为离线
const offlineThreshold = 180 // 3分钟

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
	deviceIDs := make([]uint, 0, len(devices))
	for _, device := range devices {
		deviceIDs = append(deviceIDs, device.ID)
	}
	rpcMethodsByDevice, err := loadDeviceRPCMethods(c.Request.Context(), global.GVA_DB, deviceIDs)
	if err != nil {
		response.FailWithMessage("获取设备能力失败", c)
		return
	}

	// 转换为响应结构体并计算在线状态
	now := time.Now()
	deviceResponses := make([]deviceResponse.DeviceResponse, 0, len(devices))
	for _, d := range devices {
		var isOnline bool
		if !d.LastInform.IsZero() {
			isOnline = now.Sub(d.LastInform).Seconds() < float64(offlineThreshold)
		} else {
			isOnline = false
		}
		deviceResponses = append(deviceResponses, deviceResponse.DeviceResponse{
			ID:               d.ID,
			SerialNumber:     d.SerialNumber,
			OUI:              d.OUI,
			ProductClass:     d.ProductClass,
			Manufacturer:     d.Manufacturer,
			ModelName:        d.ModelName,
			LastInform:       d.LastInform,
			UpTime:           d.UpTime,
			IP:               d.IP,
			MacAddress:       d.MacAddress,
			ConnectionReqURL: d.ConnectionReqURL,
			SoftwareVer:      d.SoftwareVer,
			HardwareVer:      d.HardwareVer,
			SpecVer:          d.SpecVer,
			PhysicalCellID:   d.PhysicalCellID,
			CellID:           d.CellID,
			GroupId:          d.GroupId,
			Remark:           d.Remark,
			IsWhite:          d.IsWhite,
			Online:           isOnline,
			RPCMethods:       rpcMethodsByDevice[d.ID],
		})
	}

	response.OkWithDetailed(response.PageResult{
		List:     deviceResponses,
		Total:    total,
		Page:     pageInfo.Page,
		PageSize: pageInfo.PageSize,
	}, "获取成功", c)
}

func loadDeviceRPCMethods(ctx context.Context, db *gorm.DB, deviceIDs []uint) (map[uint][]string, error) {
	methodsByDevice := make(map[uint][]string, len(deviceIDs))
	for _, deviceID := range deviceIDs {
		methodsByDevice[deviceID] = []string{}
	}
	if len(deviceIDs) == 0 {
		return methodsByDevice, nil
	}

	var rows []model.DeviceRPCMethods
	if err := db.WithContext(ctx).Where("device_id IN ?", deviceIDs).Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		methods := []string{}
		if len(row.MethodsJSON) > 0 {
			if err := json.Unmarshal(row.MethodsJSON, &methods); err != nil {
				return nil, err
			}
		}
		if methods == nil {
			methods = []string{}
		}
		methodsByDevice[row.DeviceID] = methods
	}
	return methodsByDevice, nil
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
		// 1. Delete Alarms (Hard Delete)
		if err := tx.Unscoped().Where("device_id = ?", device.ID).Delete(&model.Tr069Alarm{}).Error; err != nil {
			return err
		}
		// 2. Delete DataModel Values (Hard Delete)
		if err := tx.Unscoped().Where("device_id = ?", device.ID).Delete(&model.DataModelValue{}).Error; err != nil {
			return err
		}
		// 3. Delete Device (Hard Delete)
		if err := tx.Unscoped().Delete(&device).Error; err != nil {
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
