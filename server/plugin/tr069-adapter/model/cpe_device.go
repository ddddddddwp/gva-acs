package model

import (
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
)

// CpeDevice CPE设备信息
type CpeDevice struct {
	global.GVA_MODEL
	SerialNumber     string    `json:"serialNumber" gorm:"uniqueIndex;size:64;not null;comment:设备序列号"`
	ProductClass     string    `json:"productClass" gorm:"size:64;comment:产品类别"`
	Manufacturer     string    `json:"manufacturer" gorm:"size:64;comment:制造商"`
	OUI              string    `json:"oui" gorm:"size:6;comment:组织唯一标识符"`
	ModelName        string    `json:"modelName" gorm:"size:64;comment:型号名称"`
	Description      string    `json:"description" gorm:"size:255;comment:设备描述"`
	ProvisioningCode string    `json:"provisioningCode" gorm:"size:64;comment:配置代码"`
	SoftwareVersion  string    `json:"softwareVersion" gorm:"size:32;comment:软件版本"`
	HardwareVersion  string    `json:"hardwareVersion" gorm:"size:32;comment:硬件版本"`
	SpecVersion      string    `json:"specVersion" gorm:"size:16;comment:规范版本"`
	ConnectionURL    string    `json:"connectionUrl" gorm:"size:255;comment:连接URL"`
	Username         string    `json:"username" gorm:"size:64;comment:用户名"`
	Password         string    `json:"password" gorm:"size:128;comment:密码"`
	PeriodicInform   int       `json:"periodicInform" gorm:"default:300;comment:周期性通知间隔(秒)"`
	Status           int       `json:"status" gorm:"default:1;comment:设备状态(1:在线,2:离线,3:故障)"`
	LastInform       time.Time `json:"lastInform" gorm:"comment:最后通知时间"`
	LastBootstrap    time.Time `json:"lastBootstrap" gorm:"comment:最后启动时间"`
	Tags             string    `json:"tags" gorm:"size:255;comment:设备标签"`
	
	// 关联参数
	Parameters []CpeParameter `json:"parameters,omitempty" gorm:"foreignKey:DeviceID"`
}

// TableName 设置表名
func (CpeDevice) TableName() string {
	return "cpe_devices"
}

// IsOnline 判断设备是否在线
func (c *CpeDevice) IsOnline() bool {
	return c.Status == 1 && time.Since(c.LastInform) < time.Duration(c.PeriodicInform*2)*time.Second
}

// GetDeviceKey 获取设备唯一标识
func (c *CpeDevice) GetDeviceKey() string {
	return c.OUI + "-" + c.SerialNumber
}