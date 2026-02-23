package model

import (
	"github.com/ddddddddwp/gva-acs/server/global"
	"time"
)

// Device represents a TR-069 CPE device (Base Station / Femtocell)
type Device struct {
	global.GVA_MODEL
	// Identity
	SerialNumber string `json:"serialNumber" gorm:"index;unique;comment:设备序列号(SN)"`
	OUI          string `json:"oui" gorm:"column:oui;index;comment:厂商标识(OUI)"`
	ProductClass string `json:"productClass" gorm:"comment:产品类别"`
	Manufacturer string `json:"manufacturer" gorm:"comment:厂商名称"`
	ModelName    string `json:"modelName" gorm:"comment:型号名称"`

	// Status
	LastInform time.Time `json:"lastInform" gorm:"comment:最后Inform时间"`
	UpTime     uint64    `json:"upTime" gorm:"comment:运行时长(秒)"`

	// Network
	IP               string `json:"ip" gorm:"comment:管理IP地址"`
	MacAddress       string `json:"macAddress" gorm:"comment:MAC地址"`
	ConnectionReqURL string `json:"connectionReqUrl" gorm:"comment:反向连接URL"`

	// Software
	SoftwareVer string `json:"softwareVer" gorm:"comment:当前软件版本"`
	HardwareVer string `json:"hardwareVer" gorm:"comment:硬件版本"`
	SpecVer     string `json:"specVer" gorm:"comment:协议版本(如: 1.0)"`

	// Business (Base Station Specific)
	PhysicalCellID uint   `json:"pci" gorm:"comment:物理小区标识(PCI)"`
	CellID         string `json:"cellId" gorm:"comment:小区ID(ECGI/NCI)"`

	// Management
	GroupId uint   `json:"groupId" gorm:"index;comment:分组ID"`
	Remark  string `json:"remark" gorm:"comment:备注信息"`
	IsWhite bool   `json:"isWhite" gorm:"default:false;comment:是否白名单设备"`
}

func (Device) TableName() string {
	return "tr069_devices"
}
