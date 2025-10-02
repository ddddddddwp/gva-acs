package service

import (
	"context"
	"errors"
	"github.com/flipped-aurora/gin-vue-admin/server/global"
	adapterGlobal "github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-adapter/global"
	"github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-adapter/model"
	"github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-adapter/model/request"
	"github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-adapter/model/response"
	"github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-core/interfaces"
	"go.uber.org/zap"
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
// ProcessInform 处理TR069 Inform消息
func (s *TR069Service) ProcessInform(ctx context.Context, xmlData []byte) (response []byte, err error) {
	// 使用TR069-core解析器解析消息
	if adapterGlobal.TR069Parser == nil {
		return nil, errors.New("TR069Parser未初始化")
	}

	// 解析TR069消息
	message, err := adapterGlobal.TR069Parser.ParseMessage(ctx, xmlData)
	if err != nil {
		global.GVA_LOG.Error("解析TR069消息失败", zap.Error(err))
		return nil, err
	}

	// 提取设备信息并更新数据库
	if message.DeviceID != nil && message.DeviceID.SerialNumber != "" {
		device := model.TR069Device{
			SerialNumber:  message.DeviceID.SerialNumber,
			Manufacturer:  message.DeviceID.Manufacturer,
			ProductClass:  message.DeviceID.ProductClass,
			Status:        "online",
			LastInform:    time.Now(),
			LastConnected: time.Now(),
		}

		// 检查设备是否存在
		existingDevice, err := s.GetDeviceBySerialNumber(message.DeviceID.SerialNumber)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			global.GVA_LOG.Error("查询设备失败", zap.Error(err))
		} else if errors.Is(err, gorm.ErrRecordNotFound) {
			// 创建新设备
			_, err = s.CreateDevice(device)
			if err != nil {
				global.GVA_LOG.Error("创建设备失败", zap.Error(err))
			}
		} else {
			// 更新现有设备
			existingDevice.Status = "online"
			existingDevice.LastInform = time.Now()
			existingDevice.LastConnected = time.Now()
			err = s.UpdateDevice(existingDevice)
			if err != nil {
				global.GVA_LOG.Error("更新设备失败", zap.Error(err))
			}
		}
	}

	// 使用TR069-core构建器构建响应
	if adapterGlobal.TR069Builder == nil {
		return nil, errors.New("TR069Builder未初始化")
	}

	// 构建响应消息
	responseData := map[string]interface{}{
		"MaxEnvelopes": 1,
	}

	response, err = adapterGlobal.TR069Builder.BuildRPCResponse(ctx, "InformResponse", responseData)
	if err != nil {
		global.GVA_LOG.Error("构建TR069响应失败", zap.Error(err))
		return nil, err
	}

	// 触发事件通知
	if adapterGlobal.TR069EventManager != nil {
		// 将DeviceID转换为字符串格式
		deviceIDStr := ""
		if message.DeviceID != nil {
			deviceIDStr = message.DeviceID.SerialNumber
		}

		event := &interfaces.Event{
			EventType:  interfaces.EventBoot,
			DeviceID:   deviceIDStr,
			Timestamp:  time.Now(),
			Parameters: map[string]interface{}{"message": "Inform processed"},
		}

		err = adapterGlobal.TR069EventManager.Notify(ctx, event)
		if err != nil {
			global.GVA_LOG.Warn("发送事件通知失败", zap.Error(err))
		}
	}

	return response, nil
}

// SetDeviceParameters 设置设备参数
func (s *TR069Service) SetDeviceParameters(req request.TR069DeviceParam) (err error) {
	// 获取设备
	device, err := s.GetDeviceByID(req.DeviceID)
	if err != nil {
		return err
	}

	// 使用TR069-core构建器构建SetParameterValues请求
	if adapterGlobal.TR069Builder == nil {
		return errors.New("TR069Builder未初始化")
	}

	// 构建参数设置请求
	ctx := context.Background()
	params := map[string]interface{}{
		"ParameterList": []map[string]interface{}{
			{
				"Name":  req.ParamPath,
				"Value": req.Value,
			},
		},
		"ParameterKey": req.ParamPath,
	}

	// 构建RPC请求
	_, err = adapterGlobal.TR069Builder.BuildRPCRequest(ctx, "SetParameterValues", params)
	if err != nil {
		global.GVA_LOG.Error("构建SetParameterValues请求失败", zap.Error(err))
		return err
	}

	// 记录参数设置事件
	if adapterGlobal.TR069EventManager != nil {
		event := &interfaces.Event{
			EventType: interfaces.EventValueChange,
			DeviceID:  device.SerialNumber,
			Timestamp: time.Now(),
			Parameters: map[string]interface{}{
				"parameter": req.ParamPath,
				"value":     req.Value,
			},
		}

		err = adapterGlobal.TR069EventManager.Notify(ctx, event)
		if err != nil {
			global.GVA_LOG.Warn("发送参数设置事件失败", zap.Error(err))
		}
	}

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
