package api

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/ddddddddwp/gva-acs/server/global"
	"github.com/ddddddddwp/gva-acs/server/model/common/request"
	"github.com/ddddddddwp/gva-acs/server/model/common/response"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/adapter"
	tr069Config "github.com/ddddddddwp/gva-acs/server/plugin/tr069/config"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/model"
	tr069Request "github.com/ddddddddwp/gva-acs/server/plugin/tr069/model/request"
	deviceResponse "github.com/ddddddddwp/gva-acs/server/plugin/tr069/model/response"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// 设备离线阈值（秒），超过此时间未收到 Inform 视为离线
const offlineThreshold = 180 // 3分钟

type DeviceApi struct{}

func connectionProfileResponse(profile model.ConnectionProfile) deviceResponse.ConnectionProfileResponse {
	effectiveURL := strings.TrimSpace(profile.OverrideURL)
	if effectiveURL == "" {
		effectiveURL = strings.TrimSpace(profile.DiscoveredURL)
	}
	return deviceResponse.ConnectionProfileResponse{
		DeviceID: profile.DeviceID, EffectiveURL: effectiveURL,
		DiscoveredURL: profile.DiscoveredURL, OverrideURL: profile.OverrideURL,
		Username: profile.Username, CredentialSource: profile.CredentialSource,
		AuthScheme: profile.AuthScheme, ProvisionState: profile.ProvisionState,
		ProvisionCommandID: profile.ProvisionCommandID, LastError: profile.LastError,
		LastWakeAt: profile.LastWakeAt, LastWakeStatus: profile.LastWakeStatus,
	}
}

func connectionProfileDeviceID(c *gin.Context) (uint, bool) {
	value, err := strconv.ParseUint(c.Param("deviceId"), 10, 64)
	if err != nil || value == 0 {
		response.FailWithMessage("设备ID错误", c)
		return 0, false
	}
	return uint(value), true
}

func connectionProfileRepository(requireCipher bool) (*adapter.ConnectionProfileRepository, error) {
	var credentialCipher adapter.CredentialCipher
	if requireCipher {
		var err error
		credentialCipher, err = adapter.NewCredentialCipher(tr069Config.CurrentRuntime().Settings.ConnectionRequest)
		if err != nil {
			return nil, err
		}
	}
	return adapter.NewConnectionProfileRepository(global.GVA_DB, credentialCipher), nil
}

func (a *DeviceApi) GetConnectionProfile(c *gin.Context) {
	deviceID, ok := connectionProfileDeviceID(c)
	if !ok {
		return
	}
	repository, _ := connectionProfileRepository(false)
	profile, err := repository.Get(c.Request.Context(), deviceID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.FailWithMessage("设备尚无 Connection Profile", c)
			return
		}
		response.FailWithMessage("获取 Connection Profile 失败", c)
		return
	}
	response.OkWithDetailed(connectionProfileResponse(profile), "获取成功", c)
}

func (a *DeviceApi) UpdateConnectionProfile(c *gin.Context) {
	deviceID, ok := connectionProfileDeviceID(c)
	if !ok {
		return
	}
	var input tr069Request.ConnectionProfileOverrideRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		response.FailWithMessage("参数错误", c)
		return
	}
	repository, err := connectionProfileRepository(input.Username != "" || input.Password != "")
	if err != nil {
		response.FailWithMessage("Connection Profile 凭据加密配置不可用", c)
		return
	}
	profile, err := repository.UpdateOverride(c.Request.Context(), deviceID, adapter.ConnectionProfileOverride{
		OverrideURL: input.OverrideURL, Username: input.Username, Password: input.Password,
		ClearOverride: input.ClearOverride, ClearCredentials: input.ClearCredentials,
	})
	if err != nil {
		response.FailWithMessage("更新 Connection Profile 失败: "+err.Error(), c)
		return
	}
	response.OkWithDetailed(connectionProfileResponse(profile), "更新成功", c)
}

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
		if d.DeletingAt == nil && !d.LastInform.IsZero() {
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
			Deleting:         d.DeletingAt != nil,
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
		if err := tx.Unscoped().Where("device_id = ?", device.ID).Delete(&model.ConnectionProfile{}).Error; err != nil {
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
