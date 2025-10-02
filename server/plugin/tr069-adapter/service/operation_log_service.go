package service

import (
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-adapter/model"
	"gorm.io/gorm"
)

type OperationLogService struct{}

// CreateOperationLog 创建操作日志
func (s *OperationLogService) CreateOperationLog(log model.CpeOperationLog) error {
	if log.StartTime.IsZero() {
		log.StartTime = time.Now()
	}
	
	return global.GVA_DB.Create(&log).Error
}

// GetOperationLogByID 根据ID获取操作日志
func (s *OperationLogService) GetOperationLogByID(id uint) (log model.CpeOperationLog, err error) {
	err = global.GVA_DB.Preload("Device").Preload("Session").Where("id = ?", id).First(&log).Error
	return log, err
}

// GetOperationLogList 获取操作日志列表
func (s *OperationLogService) GetOperationLogList(page, pageSize int, deviceID uint, operation, status string) (list []model.CpeOperationLog, total int64, err error) {
	limit := pageSize
	offset := pageSize * (page - 1)
	
	db := global.GVA_DB.Model(&model.CpeOperationLog{}).Preload("Device").Preload("Session")
	
	// 构建查询条件
	if deviceID != 0 {
		db = db.Where("device_id = ?", deviceID)
	}
	if operation != "" {
		db = db.Where("operation = ?", operation)
	}
	if status != "" {
		db = db.Where("status = ?", status)
	}
	
	// 获取总数
	err = db.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}
	
	// 获取列表
	err = db.Limit(limit).Offset(offset).Order("start_time DESC").Find(&list).Error
	return list, total, err
}

// UpdateOperationLog 更新操作日志
func (s *OperationLogService) UpdateOperationLog(id uint, updates map[string]interface{}) error {
	return global.GVA_DB.Model(&model.CpeOperationLog{}).Where("id = ?", id).Updates(updates).Error
}

// CompleteOperationLog 完成操作日志
func (s *OperationLogService) CompleteOperationLog(id uint, status string, result, errorMessage string) error {
	now := time.Now()
	updates := map[string]interface{}{
		"status":    status,
		"result":    result,
		"end_time":  &now,
	}
	
	if errorMessage != "" {
		updates["error_message"] = errorMessage
	}
	
	return global.GVA_DB.Model(&model.CpeOperationLog{}).Where("id = ?", id).Updates(updates).Error
}

// DeleteOperationLog 删除操作日志
func (s *OperationLogService) DeleteOperationLog(id uint) error {
	return global.GVA_DB.Where("id = ?", id).Delete(&model.CpeOperationLog{}).Error
}

// GetOperationLogsByDevice 获取设备的操作日志
func (s *OperationLogService) GetOperationLogsByDevice(deviceID uint, limit int) (logs []model.CpeOperationLog, err error) {
	db := global.GVA_DB.Where("device_id = ?", deviceID)
	
	if limit > 0 {
		db = db.Limit(limit)
	}
	
	err = db.Order("start_time DESC").Find(&logs).Error
	return logs, err
}

// GetOperationLogsBySession 获取会话的操作日志
func (s *OperationLogService) GetOperationLogsBySession(sessionID uint, limit int) (logs []model.CpeOperationLog, err error) {
	db := global.GVA_DB.Where("session_id = ?", sessionID)
	
	if limit > 0 {
		db = db.Limit(limit)
	}
	
	err = db.Order("start_time DESC").Find(&logs).Error
	return logs, err
}

// GetOperationLogStatistics 获取操作日志统计信息
func (s *OperationLogService) GetOperationLogStatistics() (stats map[string]interface{}, err error) {
	stats = make(map[string]interface{})
	
	// 总操作数
	var totalOperations int64
	err = global.GVA_DB.Model(&model.CpeOperationLog{}).Count(&totalOperations).Error
	if err != nil {
		return nil, err
	}
	stats["total_operations"] = totalOperations
	
	// 成功操作数
	var successOperations int64
	err = global.GVA_DB.Model(&model.CpeOperationLog{}).Where("status = ?", "completed").Count(&successOperations).Error
	if err != nil {
		return nil, err
	}
	stats["success_operations"] = successOperations
	
	// 失败操作数
	var failedOperations int64
	err = global.GVA_DB.Model(&model.CpeOperationLog{}).Where("status = ?", "failed").Count(&failedOperations).Error
	if err != nil {
		return nil, err
	}
	stats["failed_operations"] = failedOperations
	
	// 进行中操作数
	var pendingOperations int64
	err = global.GVA_DB.Model(&model.CpeOperationLog{}).Where("status = ?", "pending").Count(&pendingOperations).Error
	if err != nil {
		return nil, err
	}
	stats["pending_operations"] = pendingOperations
	
	// 今日操作数
	today := time.Now().Truncate(24 * time.Hour)
	var todayOperations int64
	err = global.GVA_DB.Model(&model.CpeOperationLog{}).Where("start_time >= ?", today).Count(&todayOperations).Error
	if err != nil {
		return nil, err
	}
	stats["today_operations"] = todayOperations
	
	// 按操作类型统计
	var operationStats []struct {
		Operation string `json:"operation"`
		Count     int64  `json:"count"`
	}
	err = global.GVA_DB.Model(&model.CpeOperationLog{}).
		Select("operation, COUNT(*) as count").
		Group("operation").
		Find(&operationStats).Error
	if err != nil {
		return nil, err
	}
	stats["operation_statistics"] = operationStats
	
	// 按CWMP方法统计
	var methodStats []struct {
		CWMPMethod string `json:"cwmp_method"`
		Count      int64  `json:"count"`
	}
	err = global.GVA_DB.Model(&model.CpeOperationLog{}).
		Select("cwmp_method, COUNT(*) as count").
		Where("cwmp_method != ''").
		Group("cwmp_method").
		Find(&methodStats).Error
	if err != nil {
		return nil, err
	}
	stats["method_statistics"] = methodStats
	
	// 平均执行时间（已完成的操作）
	var avgDuration float64
	err = global.GVA_DB.Model(&model.CpeOperationLog{}).
		Select("AVG(EXTRACT(EPOCH FROM (end_time - start_time)))").
		Where("status = ? AND end_time IS NOT NULL", "completed").
		Scan(&avgDuration).Error
	if err != nil {
		return nil, err
	}
	stats["average_duration_seconds"] = avgDuration
	
	return stats, nil
}

// GetOperationLogsByTimeRange 根据时间范围获取操作日志
func (s *OperationLogService) GetOperationLogsByTimeRange(startTime, endTime time.Time, page, pageSize int) (logs []model.CpeOperationLog, total int64, err error) {
	limit := pageSize
	offset := pageSize * (page - 1)
	
	db := global.GVA_DB.Model(&model.CpeOperationLog{}).Preload("Device").Preload("Session").
		Where("start_time BETWEEN ? AND ?", startTime, endTime)
	
	// 获取总数
	err = db.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}
	
	// 获取列表
	err = db.Limit(limit).Offset(offset).Order("start_time DESC").Find(&logs).Error
	return logs, total, err
}

// CleanupOldOperationLogs 清理旧的操作日志
func (s *OperationLogService) CleanupOldOperationLogs(retentionDays int) error {
	if retentionDays <= 0 {
		return nil
	}
	
	cutoffTime := time.Now().AddDate(0, 0, -retentionDays)
	
	return global.GVA_DB.Where("start_time < ?", cutoffTime).Delete(&model.CpeOperationLog{}).Error
}

// GetRecentOperationLogs 获取最近的操作日志
func (s *OperationLogService) GetRecentOperationLogs(limit int) (logs []model.CpeOperationLog, err error) {
	db := global.GVA_DB.Preload("Device").Preload("Session")
	
	if limit > 0 {
		db = db.Limit(limit)
	}
	
	err = db.Order("start_time DESC").Find(&logs).Error
	return logs, err
}

// GetFailedOperationLogs 获取失败的操作日志
func (s *OperationLogService) GetFailedOperationLogs(page, pageSize int) (logs []model.CpeOperationLog, total int64, err error) {
	limit := pageSize
	offset := pageSize * (page - 1)
	
	db := global.GVA_DB.Model(&model.CpeOperationLog{}).Preload("Device").Preload("Session").
		Where("status = ?", "failed")
	
	// 获取总数
	err = db.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}
	
	// 获取列表
	err = db.Limit(limit).Offset(offset).Order("start_time DESC").Find(&logs).Error
	return logs, total, err
}

// RetryFailedOperation 重试失败的操作
func (s *OperationLogService) RetryFailedOperation(id uint) error {
	updates := map[string]interface{}{
		"status":        "pending",
		"error_message": "",
		"error_code":    "",
		"end_time":      nil,
		"retry_count":   gorm.Expr("retry_count + 1"),
	}
	
	return global.GVA_DB.Model(&model.CpeOperationLog{}).Where("id = ?", id).Updates(updates).Error
}