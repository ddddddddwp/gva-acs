package service

import (
	"fmt"
	"time"
	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-adapter/model"
	"github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-adapter/model/request"
	"gorm.io/gorm"
)

// TR069BridgeService TR069桥接服务
type TR069BridgeService struct {
	db *gorm.DB
}

// NewTR069BridgeService 创建TR069桥接服务
func NewTR069BridgeService(db *gorm.DB) *TR069BridgeService {
	return &TR069BridgeService{
		db: db,
	}
}

// CreateOrUpdateDeviceFromInform 从Inform消息创建或更新设备
func (b *TR069BridgeService) CreateOrUpdateDeviceFromInform(deviceInfo map[string]interface{}, inform map[string]interface{}) (*model.CpeDevice, error) {
	// 提取设备基本信息
	serialNumber, _ := inform["DeviceId"].(map[string]interface{})["SerialNumber"].(string)
	if serialNumber == "" {
		return nil, fmt.Errorf("missing serial number in inform message")
	}
	
	// 查找设备是否已存在
	var device model.CpeDevice
	result := global.GVA_DB.Where("serial_number = ?", serialNumber).First(&device)
	
	// 设备不存在，创建新设备
	if result.Error != nil {
		// 从Inform消息中提取设备信息
		deviceId, _ := inform["DeviceId"].(map[string]interface{})
		
		newDevice := model.CpeDevice{
			SerialNumber:     serialNumber,
			ProductClass:     deviceId["ProductClass"].(string),
			Manufacturer:     deviceId["Manufacturer"].(string),
			OUI:              deviceId["OUI"].(string),
			Status:           1, // 在线状态
			LastInform:       time.Now(),
			PeriodicInform:   300, // 默认300秒
		}
		
		// 从参数列表中提取更多设备信息
		if paramList, ok := inform["ParameterList"].([]interface{}); ok {
			for _, p := range paramList {
				param, _ := p.(map[string]interface{})
				name, _ := param["Name"].(string)
				value, _ := param["Value"].(string)
				
				switch name {
				case "Device.DeviceInfo.ModelName", "InternetGatewayDevice.DeviceInfo.ModelName":
					newDevice.ModelName = value
				case "Device.DeviceInfo.HardwareVersion", "InternetGatewayDevice.DeviceInfo.HardwareVersion":
					newDevice.HardwareVersion = value
				case "Device.DeviceInfo.SoftwareVersion", "InternetGatewayDevice.DeviceInfo.SoftwareVersion":
					newDevice.SoftwareVersion = value
				case "Device.DeviceInfo.ProvisioningCode", "InternetGatewayDevice.DeviceInfo.ProvisioningCode":
					newDevice.ProvisioningCode = value
				case "Device.DeviceInfo.Description", "InternetGatewayDevice.DeviceInfo.Description":
					newDevice.Description = value
				case "Device.DeviceInfo.SpecVersion", "InternetGatewayDevice.DeviceInfo.SpecVersion":
					newDevice.SpecVersion = value
				}
			}
		}
		
		// 保存设备信息
		if err := global.GVA_DB.Create(&newDevice).Error; err != nil {
			return nil, fmt.Errorf("failed to create device: %w", err)
		}
		
		// 创建会话记录
		session := model.CpeSession{
			DeviceID:     newDevice.ID,
			SessionID:    fmt.Sprintf("%s-%d", serialNumber, time.Now().Unix()),
			State:        "active",
			StartTime:    time.Now(),
			LastActivity: time.Now(),
			RemoteAddr:   deviceInfo["RemoteAddr"].(string),
			UserAgent:    deviceInfo["UserAgent"].(string),
		}
		
		if err := global.GVA_DB.Create(&session).Error; err != nil {
			return nil, fmt.Errorf("failed to create session: %w", err)
		}
		
		// 记录操作日志
		opLog := model.CpeOperationLog{
			DeviceID:  newDevice.ID,
			SessionID: session.SessionID,
			Operation: "Register",
			Method:    "Inform",
			Parameters: fmt.Sprintf("SerialNumber: %s, ProductClass: %s, Manufacturer: %s", 
				serialNumber, newDevice.ProductClass, newDevice.Manufacturer),
			Status:    1, // 成功
			StartTime: time.Now(),
			EndTime:   &session.LastActivity,
			Duration:  0,
		}
		
		if err := global.GVA_DB.Create(&opLog).Error; err != nil {
			return nil, fmt.Errorf("failed to create operation log: %w", err)
		}
		
		return &newDevice, nil
	}
	
	// 设备存在，更新设备信息
	updates := map[string]interface{}{
		"status":      1, // 在线状态
		"last_inform": time.Now(),
	}
	
	// 更新设备信息
	if err := global.GVA_DB.Model(&device).Updates(updates).Error; err != nil {
		return nil, fmt.Errorf("failed to update device: %w", err)
	}
	
	// 更新会话信息
	var session model.CpeSession
	sessionResult := global.GVA_DB.Where("device_id = ? AND end_time IS NULL", device.ID).
		Order("start_time DESC").First(&session)
	
	if sessionResult.Error != nil {
		// 创建新会话
		session = model.CpeSession{
			DeviceID:     device.ID,
			SessionID:    fmt.Sprintf("%s-%d", serialNumber, time.Now().Unix()),
			State:        "active",
			StartTime:    time.Now(),
			LastActivity: time.Now(),
			RemoteAddr:   deviceInfo["RemoteAddr"].(string),
			UserAgent:    deviceInfo["UserAgent"].(string),
		}
		
		if err := global.GVA_DB.Create(&session).Error; err != nil {
			return nil, fmt.Errorf("failed to create session: %w", err)
		}
	} else {
		// 更新现有会话
		if err := global.GVA_DB.Model(&session).Update("last_activity", time.Now()).Error; err != nil {
			return nil, fmt.Errorf("failed to update session: %w", err)
		}
	}
	
	// 记录操作日志
	opLog := model.CpeOperationLog{
		DeviceID:  device.ID,
		SessionID: session.SessionID,
		Operation: "Inform",
		Method:    "Inform",
		Parameters: fmt.Sprintf("SerialNumber: %s", serialNumber),
		Status:    1, // 成功
		StartTime: time.Now(),
		EndTime:   &session.LastActivity,
		Duration:  0,
	}
	
	if err := global.GVA_DB.Create(&opLog).Error; err != nil {
		return nil, fmt.Errorf("failed to create operation log: %w", err)
	}
	
	return &device, nil
}

// GetDeviceList 获取设备列表
func (b *TR069BridgeService) GetDeviceList(req request.DeviceSearch) ([]model.CpeDevice, int64, error) {
	var devices []model.CpeDevice
	var total int64
	
	db := b.db.Model(&model.CpeDevice{})
	
	// 添加搜索条件
	if req.SerialNumber != "" {
		db = db.Where("serial_number LIKE ?", "%"+req.SerialNumber+"%")
	}
	if req.Status != nil {
		db = db.Where("status = ?", *req.Status)
	}
	
	// 获取总数
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	
	// 分页查询
	offset := (req.Page - 1) * req.PageSize
	if err := db.Offset(offset).Limit(req.PageSize).Find(&devices).Error; err != nil {
		return nil, 0, err
	}
	
	return devices, total, nil
}

// GetDeviceByID 根据ID获取设备详情
func (b *TR069BridgeService) GetDeviceByID(id uint) (*model.CpeDevice, error) {
	var device model.CpeDevice
	if err := b.db.First(&device, id).Error; err != nil {
		return nil, err
	}
	return &device, nil
}

// CreateDevice 创建设备
func (b *TR069BridgeService) CreateDevice(req request.CreateDeviceRequest) (*model.CpeDevice, error) {
	device := model.CpeDevice{
		SerialNumber:     req.SerialNumber,
		ProductClass:     req.ProductClass,
		Manufacturer:     req.Manufacturer,
		OUI:              req.OUI,
		ModelName:        req.ModelName,
		HardwareVersion:  req.HardwareVersion,
		SoftwareVersion:  req.SoftwareVersion,
		ProvisioningCode: req.ProvisioningCode,
		Description:      req.Description,
		SpecVersion:      req.SpecVersion,
		Status:           0, // 离线状态
		PeriodicInform:   300,
	}
	
	if err := b.db.Create(&device).Error; err != nil {
		return nil, err
	}
	
	return &device, nil
}

// UpdateDevice 更新设备
func (b *TR069BridgeService) UpdateDevice(id uint, updates map[string]interface{}) error {
	return b.db.Model(&model.CpeDevice{}).Where("id = ?", id).Updates(updates).Error
}

// DeleteDevice 删除设备
func (b *TR069BridgeService) DeleteDevice(id uint) error {
	return b.db.Delete(&model.CpeDevice{}, id).Error
}

// BatchDeleteDevices 批量删除设备
func (b *TR069BridgeService) BatchDeleteDevices(ids []uint) error {
	return b.db.Delete(&model.CpeDevice{}, ids).Error
}

// GetDeviceStats 获取设备统计信息
func (b *TR069BridgeService) GetDeviceStats() (map[string]interface{}, error) {
	var stats map[string]interface{} = make(map[string]interface{})
	
	// 总设备数
	var totalCount int64
	if err := b.db.Model(&model.CpeDevice{}).Count(&totalCount).Error; err != nil {
		return nil, err
	}
	stats["total"] = totalCount
	
	// 在线设备数
	var onlineCount int64
	if err := b.db.Model(&model.CpeDevice{}).Where("status = ?", 1).Count(&onlineCount).Error; err != nil {
		return nil, err
	}
	stats["online"] = onlineCount
	
	// 离线设备数
	stats["offline"] = totalCount - onlineCount
	
	return stats, nil
}

// GetDeviceSessions 获取设备会话历史
func (b *TR069BridgeService) GetDeviceSessions(deviceID uint) ([]interface{}, error) {
	var sessions []model.CpeSession
	if err := b.db.Where("device_id = ?", deviceID).Order("start_time DESC").Find(&sessions).Error; err != nil {
		return nil, err
	}
	
	result := make([]interface{}, len(sessions))
	for i, session := range sessions {
		result[i] = session
	}
	
	return result, nil
}

// GetParametersByDevice 获取设备参数
func (b *TR069BridgeService) GetParametersByDevice(deviceID uint, parameterNames []string) ([]interface{}, error) {
	var parameters []model.CpeParameter
	db := b.db.Where("device_id = ?", deviceID)
	
	if len(parameterNames) > 0 {
		db = db.Where("name IN ?", parameterNames)
	}
	
	if err := db.Find(&parameters).Error; err != nil {
		return nil, err
	}
	
	result := make([]interface{}, len(parameters))
	for i, param := range parameters {
		result[i] = param
	}
	
	return result, nil
}

// SetDeviceParameters 设置设备参数
func (b *TR069BridgeService) SetDeviceParameters(deviceID uint, req request.SetParametersRequest) error {
	for _, param := range req.Parameters {
		// 查找参数是否存在
		var existingParam model.CpeParameter
		result := b.db.Where("device_id = ? AND name = ?", deviceID, param.Name).First(&existingParam)
		
		if result.Error != nil {
			// 参数不存在，创建新参数
			newParam := model.CpeParameter{
				DeviceID:    deviceID,
				Name:        param.Name,
				Value:       param.Value,
				Type:        "string", // 默认类型
				Writable:    true,
				LastChanged: time.Now(),
			}
			
			if err := b.db.Create(&newParam).Error; err != nil {
				return fmt.Errorf("failed to create parameter %s: %w", param.Name, err)
			}
		} else {
			// 参数存在，更新参数值
			updates := map[string]interface{}{
				"value":        param.Value,
				"last_changed": time.Now(),
			}
			
			if err := b.db.Model(&existingParam).Updates(updates).Error; err != nil {
				return fmt.Errorf("failed to update parameter %s: %w", param.Name, err)
			}
		}
	}
	
	return nil
}