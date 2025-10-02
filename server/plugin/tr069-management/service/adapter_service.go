package service

import (
	"errors"
	"fmt"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-management/model"
	"go.uber.org/zap"
)

type AdapterService struct{}

// GetDeviceStatus 获取设备在线状态
func (s *AdapterService) GetDeviceStatus(serialNumber string) (bool, error) {
	var session model.CPESession
	
	// 查询设备最近的会话记录
	result := global.GVA_DB.Where("device_serial_number = ?", serialNumber).
		Order("created_at DESC").
		First(&session)
	
	if result.Error != nil {
		global.GVA_LOG.Error("获取设备会话记录失败", zap.Error(result.Error), zap.String("serialNumber", serialNumber))
		return false, result.Error
	}
	
	// 检查会话是否在有效期内（30分钟）
	if session.Status == "active" && time.Since(session.CreatedAt) < 30*time.Minute {
		return true, nil
	}
	
	return false, nil
}

// TriggerDeviceAction 触发设备操作（如重启、恢复出厂设置等）
func (s *AdapterService) TriggerDeviceAction(serialNumber string, action string) error {
	// 检查设备是否在线
	online, err := s.GetDeviceStatus(serialNumber)
	if err != nil {
		return err
	}
	
	if !online {
		return errors.New("设备离线，无法执行操作")
	}
	
	// 创建设备操作任务
	task := model.DeviceTask{
		DeviceSerialNumber: serialNumber,
		TaskType:           action,
		Status:             "pending",
		Description:        fmt.Sprintf("执行%s操作", action),
	}
	
	if err := global.GVA_DB.Create(&task).Error; err != nil {
		global.GVA_LOG.Error("创建设备操作任务失败", zap.Error(err), zap.String("serialNumber", serialNumber))
		return err
	}
	
	global.GVA_LOG.Info("设备操作任务已创建", 
		zap.String("serialNumber", serialNumber), 
		zap.String("action", action),
		zap.Uint("taskID", task.ID))
	
	return nil
}

// GetDeviceEvents 获取设备事件记录
func (s *AdapterService) GetDeviceEvents(serialNumber string, page, pageSize int) ([]model.CPEEvent, int64, error) {
	var events []model.CPEEvent
	var total int64
	
	// 查询总数
	if err := global.GVA_DB.Model(&model.CPEEvent{}).
		Where("device_serial_number = ?", serialNumber).
		Count(&total).Error; err != nil {
		global.GVA_LOG.Error("获取设备事件总数失败", zap.Error(err), zap.String("serialNumber", serialNumber))
		return nil, 0, err
	}
	
	// 分页查询
	if err := global.GVA_DB.Where("device_serial_number = ?", serialNumber).
		Order("created_at DESC").
		Limit(pageSize).
		Offset((page - 1) * pageSize).
		Find(&events).Error; err != nil {
		global.GVA_LOG.Error("获取设备事件记录失败", zap.Error(err), zap.String("serialNumber", serialNumber))
		return nil, 0, err
	}
	
	return events, total, nil
}

// GetDeviceSessions 获取设备会话记录
func (s *AdapterService) GetDeviceSessions(serialNumber string, page, pageSize int) ([]model.CPESession, int64, error) {
	var sessions []model.CPESession
	var total int64
	
	// 查询总数
	if err := global.GVA_DB.Model(&model.CPESession{}).
		Where("device_serial_number = ?", serialNumber).
		Count(&total).Error; err != nil {
		global.GVA_LOG.Error("获取设备会话总数失败", zap.Error(err), zap.String("serialNumber", serialNumber))
		return nil, 0, err
	}
	
	// 分页查询
	if err := global.GVA_DB.Where("device_serial_number = ?", serialNumber).
		Order("created_at DESC").
		Limit(pageSize).
		Offset((page - 1) * pageSize).
		Find(&sessions).Error; err != nil {
		global.GVA_LOG.Error("获取设备会话记录失败", zap.Error(err), zap.String("serialNumber", serialNumber))
		return nil, 0, err
	}
	
	return sessions, total, nil
}

// GetDeviceOperationLogs 获取设备操作日志
func (s *AdapterService) GetDeviceOperationLogs(serialNumber string, page, pageSize int) ([]model.CPEOperationLog, int64, error) {
	var logs []model.CPEOperationLog
	var total int64
	
	// 查询总数
	if err := global.GVA_DB.Model(&model.CPEOperationLog{}).
		Where("device_serial_number = ?", serialNumber).
		Count(&total).Error; err != nil {
		global.GVA_LOG.Error("获取设备操作日志总数失败", zap.Error(err), zap.String("serialNumber", serialNumber))
		return nil, 0, err
	}
	
	// 分页查询
	if err := global.GVA_DB.Where("device_serial_number = ?", serialNumber).
		Order("created_at DESC").
		Limit(pageSize).
		Offset((page - 1) * pageSize).
		Find(&logs).Error; err != nil {
		global.GVA_LOG.Error("获取设备操作日志失败", zap.Error(err), zap.String("serialNumber", serialNumber))
		return nil, 0, err
	}
	
	return logs, total, nil
}