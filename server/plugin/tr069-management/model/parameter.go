package model

import (
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
)

// Parameter TR069设备参数模型
type Parameter struct {
	global.GVA_MODEL
	DeviceID     uint       `json:"deviceId" gorm:"index;comment:设备ID"`
	Name         string     `json:"name" gorm:"index;comment:参数名称"`
	Value        string     `json:"value" gorm:"comment:参数值"`
	Type         string     `json:"type" gorm:"comment:参数类型"`
	Writable     bool       `json:"writable" gorm:"comment:是否可写"`
	Notification int        `json:"notification" gorm:"comment:通知类型;default:0"` // 0:关闭 1:被动通知 2:主动通知
	LastChanged  time.Time  `json:"lastChanged" gorm:"comment:最后修改时间"`
	Description  string     `json:"description" gorm:"comment:参数描述"`
	Category     string     `json:"category" gorm:"comment:参数分类"`
	Tags         string     `json:"tags" gorm:"comment:参数标签"`
}