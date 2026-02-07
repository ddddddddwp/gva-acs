package model

import (
	"github.com/ddddddddwp/gva-acs/server/global"
	"time"
)

type Device struct {
	global.GVA_MODEL
	SerialNumber string    `json:"serialNumber" gorm:"index;unique;comment:序列号"`
	OUI          string    `json:"oui" gorm:"index;comment:厂商标识"`
	ProductClass string    `json:"productClass" gorm:"comment:产品类别"`
	SoftwareVer  string    `json:"softwareVer" gorm:"comment:软件版本"`
	IP           string    `json:"ip" gorm:"comment:设备IP"`
	LastOnline   time.Time `json:"lastOnline" gorm:"comment:最后上线时间"`
}

func (Device) TableName() string {
	return "tr069_devices"
}
