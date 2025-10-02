package model

import (
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
)

// CpeOperationLog CPE操作日志
type CpeOperationLog struct {
	global.GVA_MODEL
	DeviceID    uint      `json:"deviceId" gorm:"not null;index;comment:设备ID"`
	SessionID   string    `json:"sessionId" gorm:"size:64;index;comment:会话ID"`
	Operation   string    `json:"operation" gorm:"size:64;not null;comment:操作类型"`
	Method      string    `json:"method" gorm:"size:32;comment:CWMP方法"`
	Parameters  string    `json:"parameters" gorm:"type:text;comment:操作参数"`
	Result      string    `json:"result" gorm:"type:text;comment:操作结果"`
	Status      int       `json:"status" gorm:"default:0;comment:操作状态(0:进行中,1:成功,2:失败)"`
	ErrorCode   string    `json:"errorCode" gorm:"size:16;comment:错误代码"`
	ErrorMsg    string    `json:"errorMsg" gorm:"size:255;comment:错误信息"`
	StartTime   time.Time `json:"startTime" gorm:"comment:开始时间"`
	EndTime     *time.Time `json:"endTime" gorm:"comment:结束时间"`
	Duration    int       `json:"duration" gorm:"comment:执行时长(毫秒)"`
	
	// 关联设备
	Device CpeDevice `json:"device,omitempty" gorm:"foreignKey:DeviceID"`
	
	// 关联会话
	Session CpeSession `json:"session,omitempty" gorm:"foreignKey:SessionID;references:SessionID"`
}

// TableName 设置表名
func (CpeOperationLog) TableName() string {
	return "cpe_operation_logs"
}

// IsCompleted 判断操作是否完成
func (c *CpeOperationLog) IsCompleted() bool {
	return c.Status == 1 || c.Status == 2
}

// IsSuccess 判断操作是否成功
func (c *CpeOperationLog) IsSuccess() bool {
	return c.Status == 1
}

// GetDuration 获取操作持续时间
func (c *CpeOperationLog) GetDuration() time.Duration {
	if c.EndTime != nil {
		return c.EndTime.Sub(c.StartTime)
	}
	return time.Since(c.StartTime)
}