package model

import (
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
)

// CpeParameter CPE设备参数
type CpeParameter struct {
	global.GVA_MODEL
	DeviceID    uint      `json:"deviceId" gorm:"not null;index;comment:设备ID"`
	Name        string    `json:"name" gorm:"size:255;not null;comment:参数名称"`
	Value       string    `json:"value" gorm:"type:text;comment:参数值"`
	Type        string    `json:"type" gorm:"size:32;comment:参数类型"`
	Writable    bool      `json:"writable" gorm:"default:false;comment:是否可写"`
	Notification int      `json:"notification" gorm:"default:0;comment:通知级别(0:关闭,1:被动,2:主动)"`
	LastChanged time.Time `json:"lastChanged" gorm:"comment:最后修改时间"`
	
	// 关联设备
	Device CpeDevice `json:"device,omitempty" gorm:"foreignKey:DeviceID"`
}

// TableName 设置表名
func (CpeParameter) TableName() string {
	return "cpe_parameters"
}

// ParameterInfo 参数信息结构体
type ParameterInfo struct {
	Name         string `json:"name"`
	Writable     bool   `json:"writable"`
	Notification int    `json:"notification"`
	Type         string `json:"type"`
}

// ParameterValue 参数值结构体
type ParameterValue struct {
	Name  string `json:"name"`
	Value string `json:"value"`
	Type  string `json:"type"`
}

// ParameterAttribute 参数属性结构体
type ParameterAttribute struct {
	Name         string `json:"name"`
	Notification int    `json:"notification"`
	AccessList   string `json:"accessList"`
}