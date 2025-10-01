package model

import (
	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"time"
)

// TR069Device TR069设备模型
type TR069Device struct {
	global.GVA_MODEL
	Manufacturer  string    `json:"manufacturer" gorm:"comment:设备厂商"`
	ProductClass  string    `json:"productClass" gorm:"comment:产品类别"`
	SerialNumber  string    `json:"serialNumber" gorm:"index;comment:序列号"`
	HardwareVer   string    `json:"hardwareVer" gorm:"comment:硬件版本"`
	SoftwareVer   string    `json:"softwareVer" gorm:"comment:软件版本"`
	Status        string    `json:"status" gorm:"default:online;comment:设备状态"`
	LastInform    time.Time `json:"lastInform" gorm:"comment:最后一次Inform时间"`
	LastConnected time.Time `json:"lastConnected" gorm:"comment:最后一次连接时间"`
	IPAddress     string    `json:"ipAddress" gorm:"comment:IP地址"`
	MACAddress    string    `json:"macAddress" gorm:"comment:MAC地址"`
	Parameters    string    `json:"parameters" gorm:"type:text;comment:设备参数(JSON格式)"`
}

// TableName 设置表名
func (TR069Device) TableName() string {
	return "tr069_devices"
}

// TR069Event TR069事件模型
type TR069Event struct {
	global.GVA_MODEL
	DeviceID    uint      `json:"deviceId" gorm:"index;comment:设备ID"`
	EventType   string    `json:"eventType" gorm:"comment:事件类型"`
	EventCode   string    `json:"eventCode" gorm:"comment:事件代码"`
	Description string    `json:"description" gorm:"comment:事件描述"`
	Timestamp   time.Time `json:"timestamp" gorm:"comment:事件时间"`
	Parameters  string    `json:"parameters" gorm:"type:text;comment:事件参数(JSON格式)"`
}

// TableName 设置表名
func (TR069Event) TableName() string {
	return "tr069_events"
}