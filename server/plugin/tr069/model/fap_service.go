package model

import (
	"github.com/ddddddddwp/gva-acs/server/global"
)

// FAPService represents TR-196 Femto Access Point Service parameters
// Path: Device.Services.FAPService.{i}.
type FAPService struct {
	global.GVA_MODEL
	DeviceID uint `json:"deviceId" gorm:"index;comment:关联的设备ID"`

	// Core Status
	// Path: .FAPControl.LTE.AdminState
	AdminState bool `json:"adminState" gorm:"comment:管理状态(true=Up/false=Down)"`
	// Path: .FAPControl.LTE.OpState
	OpState bool `json:"opState" gorm:"comment:运行状态(true=Enable/false=Disable)"`

	// RF Configuration (LTE)
	// Path: .CellConfig.LTE.RAN.RF.
	PhysicalCellID uint   `json:"pci" gorm:"comment:物理小区标识(PCI)"`
	DLBandwidth    string `json:"dlBandwidth" gorm:"comment:下行带宽(e.g. n100=20MHz)"`
	ULBandwidth    string `json:"ulBandwidth" gorm:"comment:上行带宽"`
	EARFCNDL       uint   `json:"earfcnDl" gorm:"comment:下行频点"`
	EARFCNUL       uint   `json:"earfcnUl" gorm:"comment:上行频点"`
	TxPower        string `json:"txPower" gorm:"comment:发射功率(dBm)"` // string to support "20" or "20dBm"

	// Identification
	// Path: .CellConfig.LTE.RAN.Common.CellIdentity
	CellID string `json:"cellId" gorm:"comment:小区全局标识(ECGI/NCI)"`
	// Path: .CellConfig.LTE.EPC.TAC
	TAC uint `json:"tac" gorm:"comment:跟踪区代码"`
}

func (FAPService) TableName() string {
	return "tr069_fap_services"
}
