package service

import (
	"errors"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-management/model"
	"github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-management/model/request"
	"gorm.io/gorm"
)

// DeviceService 设备服务
type DeviceService struct{}

// GetDeviceList 获取设备列表
func (s *DeviceService) GetDeviceList(req request.DeviceSearch) ([]model.TR069Device, int64, error) {
	var devices []model.TR069Device
	var total int64

	db := global.GVA_DB.Model(&model.TR069Device{})

	// 构建查询条件
	if req.SerialNumber != "" {
		db = db.Where("serial_number LIKE ?", "%"+req.SerialNumber+"%")
	}
	if req.Manufacturer != "" {
		db = db.Where("manufacturer LIKE ?", "%"+req.Manufacturer+"%")
	}
	if req.ProductClass != "" {
		db = db.Where("product_class LIKE ?", "%"+req.ProductClass+"%")
	}
	if req.ModelName != "" {
		db = db.Where("model_name LIKE ?", "%"+req.ModelName+"%")
	}
	if req.Status != nil {
		db = db.Where("status = ?", *req.Status)
	}
	if req.GroupID != nil {
		db = db.Where("group_id = ?", *req.GroupID)
	}
	if req.Tags != "" {
		db = db.Where("tags LIKE ?", "%"+req.Tags+"%")
	}

	// 计算总数
	err := db.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	// 分页查询
	err = db.Limit(req.PageSize).Offset((req.Page - 1) * req.PageSize).Order("created_at desc").Find(&devices).Error
	if err != nil {
		return nil, 0, err
	}

	return devices, total, nil
}

// GetDeviceByID 根据ID获取设备详情
func (s *DeviceService) GetDeviceByID(id uint) (*model.TR069Device, error) {
	var device model.TR069Device
	err := global.GVA_DB.First(&device, id).Error
	if err != nil {
		return nil, err
	}
	return &device, nil
}

// GetDeviceBySerialNumber 根据序列号获取设备
func (s *DeviceService) GetDeviceBySerialNumber(serialNumber string) (*model.TR069Device, error) {
	var device model.TR069Device
	err := global.GVA_DB.Where("serial_number = ?", serialNumber).First(&device).Error
	if err != nil {
		return nil, err
	}
	return &device, nil
}

// CreateDevice 创建设备
func (s *DeviceService) CreateDevice(req request.CreateDeviceRequest) (*model.TR069Device, error) {
	// 检查设备是否已存在
	var count int64
	global.GVA_DB.Model(&model.TR069Device{}).Where("serial_number = ?", req.SerialNumber).Count(&count)
	if count > 0 {
		return nil, errors.New("设备已存在")
	}

	// 创建设备
	device := model.TR069Device{
		SerialNumber:     req.SerialNumber,
		Manufacturer:     req.Manufacturer,
		OUI:              req.OUI,
		ProductClass:     req.ProductClass,
		ModelName:        req.ModelName,
		HardwareVersion:  req.HardwareVersion,
		SoftwareVersion:  req.SoftwareVersion,
		ProvisioningCode: req.ProvisioningCode,
		Description:      req.Description,
		Status:           0, // 默认离线
		LastInform:       time.Now(),
		LastConnect:      time.Now(),
		PeriodicInform:   300, // 默认300秒
		Tags:             req.Tags,
		Location:         req.Location,
		GroupID:          req.GroupID,
		ConfigProfileID:  req.ConfigProfileID,
	}

	err := global.GVA_DB.Create(&device).Error
	if err != nil {
		return nil, err
	}

	return &device, nil
}

// UpdateDevice 更新设备
func (s *DeviceService) UpdateDevice(req request.UpdateDeviceRequest) error {
	// 检查设备是否存在
	var device model.TR069Device
	err := global.GVA_DB.First(&device, req.ID).Error
	if err != nil {
		return errors.New("设备不存在")
	}

	// 构建更新数据
	updates := map[string]interface{}{}

	if req.ModelName != "" {
		updates["model_name"] = req.ModelName
	}
	if req.HardwareVersion != "" {
		updates["hardware_version"] = req.HardwareVersion
	}
	if req.SoftwareVersion != "" {
		updates["software_version"] = req.SoftwareVersion
	}
	if req.ProvisioningCode != "" {
		updates["provisioning_code"] = req.ProvisioningCode
	}
	if req.Description != "" {
		updates["description"] = req.Description
	}
	if req.Tags != "" {
		updates["tags"] = req.Tags
	}
	if req.Location != "" {
		updates["location"] = req.Location
	}
	if req.GroupID != nil {
		updates["group_id"] = *req.GroupID
	}
	if req.ConfigProfileID != nil {
		updates["config_profile_id"] = *req.ConfigProfileID
	}
	if req.PeriodicInform != nil {
		updates["periodic_inform"] = *req.PeriodicInform
	}

	// 更新设备
	return global.GVA_DB.Model(&device).Updates(updates).Error
}

// DeleteDevice 删除设备
func (s *DeviceService) DeleteDevice(id uint) error {
	// 检查设备是否存在
	var device model.TR069Device
	err := global.GVA_DB.First(&device, id).Error
	if err != nil {
		return errors.New("设备不存在")
	}

	// 删除设备参数
	err = global.GVA_DB.Where("device_id = ?", id).Delete(&model.Parameter{}).Error
	if err != nil {
		return err
	}

	// 删除设备
	return global.GVA_DB.Delete(&device).Error
}

// BatchDeleteDevices 批量删除设备
func (s *DeviceService) BatchDeleteDevices(ids []uint) error {
	// 开启事务
	return global.GVA_DB.Transaction(func(tx *gorm.DB) error {
		// 删除设备参数
		if err := tx.Where("device_id IN ?", ids).Delete(&model.Parameter{}).Error; err != nil {
			return err
		}

		// 删除设备
		if err := tx.Where("id IN ?", ids).Delete(&model.TR069Device{}).Error; err != nil {
			return err
		}

		return nil
	})
}

// GetDeviceStats 获取设备统计信息
func (s *DeviceService) GetDeviceStats() (map[string]interface{}, error) {
	var total, online, offline, fault int64

	// 获取总数
	global.GVA_DB.Model(&model.TR069Device{}).Count(&total)

	// 获取在线数
	global.GVA_DB.Model(&model.TR069Device{}).Where("status = ?", 1).Count(&online)

	// 获取离线数
	global.GVA_DB.Model(&model.TR069Device{}).Where("status = ?", 0).Count(&offline)

	// 获取故障数
	global.GVA_DB.Model(&model.TR069Device{}).Where("status = ?", 2).Count(&fault)

	// 返回统计信息
	return map[string]interface{}{
		"total":   total,
		"online":  online,
		"offline": offline,
		"fault":   fault,
	}, nil
}

// UpdateDeviceStatus 更新设备状态
func (s *DeviceService) UpdateDeviceStatus(id uint, status int) error {
	return global.GVA_DB.Model(&model.TR069Device{}).Where("id = ?", id).Update("status", status).Error
}

// SyncDeviceFromAdapter 从TR069-Adapter同步设备信息
func (s *DeviceService) SyncDeviceFromAdapter(serialNumber string) (*model.TR069Device, error) {
	// 这里将实现与TR069-Adapter的集成，从Adapter获取设备信息并同步到Management
	// 暂时返回空实现
	return nil, errors.New("功能尚未实现")
}