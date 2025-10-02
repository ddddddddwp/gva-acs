package model

import (
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
)

// TR069Device TR069设备管理模型
type TR069Device struct {
	global.GVA_MODEL
	SerialNumber     string     `json:"serialNumber" gorm:"index;comment:设备序列号"`
	Manufacturer     string     `json:"manufacturer" gorm:"comment:制造商"`
	OUI              string     `json:"oui" gorm:"comment:组织唯一标识符"`
	ProductClass     string     `json:"productClass" gorm:"comment:产品类别"`
	ModelName        string     `json:"modelName" gorm:"comment:型号名称"`
	HardwareVersion  string     `json:"hardwareVersion" gorm:"comment:硬件版本"`
	SoftwareVersion  string     `json:"softwareVersion" gorm:"comment:软件版本"`
	ProvisioningCode string     `json:"provisioningCode" gorm:"comment:配置代码"`
	Description      string     `json:"description" gorm:"comment:设备描述"`
	SpecVersion      string     `json:"specVersion" gorm:"comment:规范版本"`
	Status           int        `json:"status" gorm:"comment:设备状态;default:0"` // 0:离线 1:在线 2:故障
	LastInform       time.Time  `json:"lastInform" gorm:"comment:最后一次Inform时间"`
	LastConnect      time.Time  `json:"lastConnect" gorm:"comment:最后一次连接时间"`
	PeriodicInform   int        `json:"periodicInform" gorm:"comment:周期性Inform间隔(秒)"`
	Tags             string     `json:"tags" gorm:"comment:设备标签"`
	Location         string     `json:"location" gorm:"comment:设备位置"`
	IPAddress        string     `json:"ipAddress" gorm:"comment:IP地址"`
	MACAddress       string     `json:"macAddress" gorm:"comment:MAC地址"`
	ConnectionType   string     `json:"connectionType" gorm:"comment:连接类型"`
	GroupID          *uint      `json:"groupId" gorm:"comment:设备组ID"`
	ConfigProfileID  *uint      `json:"configProfileId" gorm:"comment:配置文件ID"`
	FirmwareID       *uint      `json:"firmwareId" gorm:"comment:固件ID"`
	Parameters       []Parameter `json:"parameters" gorm:"foreignKey:DeviceID"`
}