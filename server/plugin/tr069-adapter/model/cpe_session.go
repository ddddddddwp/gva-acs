package model

import (
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
)

// CpeSession CPE会话信息
type CpeSession struct {
	global.GVA_MODEL
	DeviceID     uint      `json:"deviceId" gorm:"not null;index;comment:设备ID"`
	SessionID    string    `json:"sessionId" gorm:"uniqueIndex;size:64;not null;comment:会话ID"`
	State        string    `json:"state" gorm:"size:32;comment:会话状态"`
	StartTime    time.Time `json:"startTime" gorm:"comment:会话开始时间"`
	EndTime      *time.Time `json:"endTime" gorm:"comment:会话结束时间"`
	LastActivity time.Time `json:"lastActivity" gorm:"comment:最后活动时间"`
	RemoteAddr   string    `json:"remoteAddr" gorm:"size:45;comment:远程地址"`
	UserAgent    string    `json:"userAgent" gorm:"size:255;comment:用户代理"`
	
	// 关联设备
	Device CpeDevice `json:"device,omitempty" gorm:"foreignKey:DeviceID"`
	
	// 关联操作日志
	OperationLogs []CpeOperationLog `json:"operationLogs,omitempty" gorm:"foreignKey:SessionID"`
}

// TableName 设置表名
func (CpeSession) TableName() string {
	return "cpe_sessions"
}

// IsActive 判断会话是否活跃
func (c *CpeSession) IsActive() bool {
	return c.EndTime == nil && time.Since(c.LastActivity) < 5*time.Minute
}

// Duration 获取会话持续时间
func (c *CpeSession) Duration() time.Duration {
	if c.EndTime != nil {
		return c.EndTime.Sub(c.StartTime)
	}
	return time.Since(c.StartTime)
}