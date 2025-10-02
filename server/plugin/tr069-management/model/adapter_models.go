package model

import (
	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"time"
)

// CPESession TR069会话模型
type CPESession struct {
	global.GVA_MODEL
	DeviceSerialNumber string    `json:"deviceSerialNumber" gorm:"index:idx_device_serial;comment:设备序列号"`
	SessionID          string    `json:"sessionID" gorm:"comment:会话ID"`
	SessionType        string    `json:"sessionType" gorm:"comment:会话类型"`
	Status             string    `json:"status" gorm:"comment:会话状态;default:active"`
	StartTime          time.Time `json:"startTime" gorm:"comment:开始时间"`
	EndTime            *time.Time `json:"endTime" gorm:"comment:结束时间"`
	ClientIP           string    `json:"clientIP" gorm:"comment:客户端IP"`
}

// CPEEvent TR069事件模型
type CPEEvent struct {
	global.GVA_MODEL
	DeviceSerialNumber string `json:"deviceSerialNumber" gorm:"index:idx_device_serial;comment:设备序列号"`
	EventCode          string `json:"eventCode" gorm:"comment:事件代码"`
	EventType          string `json:"eventType" gorm:"comment:事件类型"`
	Description        string `json:"description" gorm:"comment:事件描述"`
}

// CPEOperationLog TR069操作日志模型
type CPEOperationLog struct {
	global.GVA_MODEL
	DeviceSerialNumber string `json:"deviceSerialNumber" gorm:"index:idx_device_serial;comment:设备序列号"`
	OperationType      string `json:"operationType" gorm:"comment:操作类型"`
	OperationTarget    string `json:"operationTarget" gorm:"comment:操作目标"`
	OperationContent   string `json:"operationContent" gorm:"comment:操作内容"`
	OperationResult    string `json:"operationResult" gorm:"comment:操作结果"`
	OperatorID         uint   `json:"operatorID" gorm:"comment:操作人ID"`
	OperatorName       string `json:"operatorName" gorm:"comment:操作人名称"`
}

// DeviceTask 设备任务模型
type DeviceTask struct {
	global.GVA_MODEL
	DeviceSerialNumber string `json:"deviceSerialNumber" gorm:"index:idx_device_serial;comment:设备序列号"`
	TaskType           string `json:"taskType" gorm:"comment:任务类型"`
	Status             string `json:"status" gorm:"comment:任务状态;default:pending"`
	Description        string `json:"description" gorm:"comment:任务描述"`
	Result             string `json:"result" gorm:"comment:任务结果"`
	CompletedAt        *time.Time `json:"completedAt" gorm:"comment:完成时间"`
}