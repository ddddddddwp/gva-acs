package service

import (
	"fmt"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-adapter/model"
	"github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-adapter/model/request"
	tr069core "github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-core/interfaces"
)

type DeviceService struct {
	bridgeService *TR069BridgeService
	tr069Builder  tr069core.Builder
}

// NewDeviceService 创建设备服务实例
func NewDeviceService() *DeviceService {
	return &DeviceService{
		bridgeService: NewTR069BridgeService(global.GVA_DB),
	}
}

// GetDeviceList 获取设备列表
func (d *DeviceService) GetDeviceList(req request.DeviceSearch) ([]model.CpeDevice, int64, error) {
	return d.bridgeService.GetDeviceList(req)
}

// GetDeviceByID 根据ID获取设备
func (d *DeviceService) GetDeviceByID(id uint) (*model.CpeDevice, error) {
	return d.bridgeService.GetDeviceByID(id)
}

// CreateDevice 创建设备
func (d *DeviceService) CreateDevice(req request.CreateDeviceRequest) (*model.CpeDevice, error) {
	return d.bridgeService.CreateDevice(req)
}

// UpdateDevice 更新设备
func (d *DeviceService) UpdateDevice(id uint, updates map[string]interface{}) error {
	return d.bridgeService.UpdateDevice(id, updates)
}

// DeleteDevice 删除设备
func (d *DeviceService) DeleteDevice(id uint) error {
	return d.bridgeService.DeleteDevice(id)
}

// BatchDeleteDevices 批量删除设备
func (d *DeviceService) BatchDeleteDevices(ids []uint) error {
	return d.bridgeService.BatchDeleteDevices(ids)
}

// GetDeviceStats 获取设备统计信息
func (d *DeviceService) GetDeviceStats() (map[string]interface{}, error) {
	return d.bridgeService.GetDeviceStats()
}

// GetDeviceSessionHistory 获取设备会话历史
func (d *DeviceService) GetDeviceSessionHistory(deviceID uint) ([]interface{}, error) {
	return d.bridgeService.GetDeviceSessions(deviceID)
}

// GetDeviceOperationLogs 获取设备操作日志
func (d *DeviceService) GetDeviceOperationLogs(deviceID uint) ([]interface{}, error) {
	// TODO: 实现操作日志获取逻辑
	return []interface{}{}, nil
}

// GetParametersByDevice 获取设备参数
func (d *DeviceService) GetParametersByDevice(deviceID uint, parameterNames []string) ([]interface{}, error) {
	return d.bridgeService.GetParametersByDevice(deviceID, parameterNames)
}

// UpdateDeviceFromInform 从Inform消息更新设备信息
func (d *DeviceService) UpdateDeviceFromInform(informMsg *model.Inform) error {
	// 将Inform消息转换为map格式，以匹配TR069BridgeService的接口
	deviceInfo := map[string]interface{}{
		"SerialNumber": informMsg.DeviceId.SerialNumber,
		"ProductClass": informMsg.DeviceId.ProductClass,
		"Manufacturer": informMsg.DeviceId.Manufacturer,
		"OUI":          informMsg.DeviceId.OUI,
	}

	inform := map[string]interface{}{
		"DeviceId": map[string]interface{}{
			"SerialNumber": informMsg.DeviceId.SerialNumber,
			"ProductClass": informMsg.DeviceId.ProductClass,
			"Manufacturer": informMsg.DeviceId.Manufacturer,
			"OUI":          informMsg.DeviceId.OUI,
		},
		"ParameterList": informMsg.ParameterList,
	}

	// 调用桥接服务创建或更新设备
	_, err := d.bridgeService.CreateOrUpdateDeviceFromInform(deviceInfo, inform)
	return err
}

// updateParametersFromInform 从Inform消息更新参数
func (d *DeviceService) updateParametersFromInform(deviceID uint, parameterList []model.ParameterValueStruct) error {
	// 批量更新参数
	for _, param := range parameterList {
		// 查找参数是否已存在
		var parameter model.CpeParameter
		result := global.GVA_DB.Where("device_id = ? AND name = ?", deviceID, param.Name).First(&parameter)
		
		if result.Error != nil {
			// 参数不存在，创建新参数
			newParam := model.CpeParameter{
				DeviceID:    deviceID,
				Name:        param.Name,
				Value:       param.Value,
				Type:        param.Type,
				Writable:    true, // 默认可写
				LastChanged: time.Now(),
			}
			
			if err := global.GVA_DB.Create(&newParam).Error; err != nil {
				return fmt.Errorf("failed to create parameter %s: %w", param.Name, err)
			}
		} else {
			// 参数存在，更新参数值
			updates := map[string]interface{}{
				"value":        param.Value,
				"type":         param.Type,
				"last_changed": time.Now(),
			}
			
			if err := global.GVA_DB.Model(&parameter).Updates(updates).Error; err != nil {
				return fmt.Errorf("failed to update parameter %s: %w", param.Name, err)
			}
		}
	}
	
	return nil
}

// UpdateParameterValues 更新参数值
func (d *DeviceService) UpdateParameterValues(serialNumber string, parameterList []model.ParameterValueStruct) error {
	// 查找设备
	var device model.CpeDevice
	if err := global.GVA_DB.Where("serial_number = ?", serialNumber).First(&device).Error; err != nil {
		return fmt.Errorf("device not found: %w", err)
	}
	
	// 更新参数
	return d.updateParametersFromInform(device.ID, parameterList)
}
