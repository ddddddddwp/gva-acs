package service

import (
	"context"
	"errors"
	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-adapter/model"
	"github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-adapter/model/request"
	"github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-adapter/model/response"
	"gorm.io/gorm"
	"time"
)

type TR069Service struct{}

// GetDeviceList 获取设备列表
func (s *TR069Service) GetDeviceList(info request.TR069DeviceSearch) (list []model.TR069Device, total int64, err error) {
	limit := info.PageSize
	offset := info.PageSize * (info.Page - 1)
	db := global.GVA_DB.Model(&model.TR069Device{})

	// 构建查询条件
	if info.Manufacturer != "" {
		db = db.Where("manufacturer LIKE ?", "%"+info.Manufacturer+"%")
	}
	if info.ProductClass != "" {
		db = db.Where("product_class LIKE ?", "%"+info.ProductClass+"%")
	}
	if info.SerialNumber != "" {
		db = db.Where("serial_number LIKE ?", "%"+info.SerialNumber+"%")
	}
	if info.Status != "" {
		db = db.Where("status = ?", info.Status)
	}
	if info.IPAddress != "" {
		db = db.Where("ip_address LIKE ?", "%"+info.IPAddress+"%")
	}
	if info.MACAddress != "" {
		db = db.Where("mac_address LIKE ?", "%"+info.MACAddress+"%")
	}

	// 执行查询
	err = db.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}
	err = db.Limit(limit).Offset(offset).Order("created_at desc").Find(&list).Error
	return list, total, err
}

// GetDeviceByID 根据ID获取设备
func (s *TR069Service) GetDeviceByID(id uint) (device model.TR069Device, err error) {
	err = global.GVA_DB.Where("id = ?", id).First(&device).Error
	return device, err
}

// GetDeviceBySerialNumber 根据序列号获取设备
func (s *TR069Service) GetDeviceBySerialNumber(serialNumber string) (device model.TR069Device, err error) {
	err = global.GVA_DB.Where("serial_number = ?", serialNumber).First(&device).Error
	return device, err
}

// CreateDevice 创建设备
func (s *TR069Service) CreateDevice(device model.TR069Device) (id uint, err error) {
	// 检查设备是否已存在
	var count int64
	global.GVA_DB.Model(&model.TR069Device{}).Where("serial_number = ?", device.SerialNumber).Count(&count)
	if count > 0 {
		return 0, errors.New("设备已存在")
	}

	// 创建设备
	result := global.GVA_DB.Create(&device)
	if result.Error != nil {
		return 0, result.Error
	}
	return device.ID, nil
}

// UpdateDevice 更新设备
func (s *TR069Service) UpdateDevice(device model.TR069Device) error {
	return global.GVA_DB.Updates(&device).Error
}

// DeleteDevice 删除设备
func (s *TR069Service) DeleteDevice(id uint) error {
	return global.GVA_DB.Transaction(func(tx *gorm.DB) error {
		// 删除设备相关的事件
		if err := tx.Where("device_id = ?", id).Delete(&model.TR069Event{}).Error; err != nil {
			return err
		}
		// 删除设备
		if err := tx.Delete(&model.TR069Device{}, id).Error; err != nil {
			return err
		}
		return nil
	})
}

// GetDeviceEvents 获取设备事件
func (s *TR069Service) GetDeviceEvents(deviceID uint, limit int) (events []model.TR069Event, err error) {
	err = global.GVA_DB.Where("device_id = ?", deviceID).
		Order("created_at desc").
		Limit(limit).
		Find(&events).Error
	return events, err
}

// ProcessInform 处理设备Inform请求
func (s *TR069Service) ProcessInform(ctx context.Context, xmlData []byte) (response []byte, err error) {
	// 简化实现，直接返回空响应
	// TODO: 实现完整的TR069协议解析
	return []byte(""), nil
}

// SetDeviceParameters 设置设备参数
func (s *TR069Service) SetDeviceParameters(req request.TR069DeviceParam) (err error) {
	// 获取设备
	_, err = s.GetDeviceByID(req.DeviceID)
	if err != nil {
		return err
	}

	// 简化实现，直接返回成功
	// TODO: 实现完整的参数设置功能
	return nil
}

// GetDeviceStats 获取设备统计信息
func (s *TR069Service) GetDeviceStats() (stats response.TR069DeviceStats, err error) {
	var totalDevices int64
	var onlineDevices int64
	var eventsLast24h int64
	var newDevicesLast7d int64

	// 总设备数
	global.GVA_DB.Model(&model.TR069Device{}).Count(&totalDevices)

	// 在线设备数
	global.GVA_DB.Model(&model.TR069Device{}).Where("status = ?", "online").Count(&onlineDevices)

	// 过去24小时的事件数
	global.GVA_DB.Model(&model.TR069Event{}).
		Where("created_at > ?", time.Now().Add(-24*time.Hour)).
		Count(&eventsLast24h)

	// 过去7天新增设备数
	global.GVA_DB.Model(&model.TR069Device{}).
		Where("created_at > ?", time.Now().AddDate(0, 0, -7)).
		Count(&newDevicesLast7d)

	stats.TotalDevices = int(totalDevices)
	stats.OnlineDevices = int(onlineDevices)
	stats.OfflineDevices = stats.TotalDevices - stats.OnlineDevices
	stats.EventsLast24h = int(eventsLast24h)
	stats.NewDevicesLast7d = int(newDevicesLast7d)

	return stats, nil
}
