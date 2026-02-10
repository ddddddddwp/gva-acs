package model

import (
	"github.com/ddddddddwp/gva-acs/server/global"
	"gorm.io/datatypes"
)

type DeviceRPCMethods struct {
	global.GVA_MODEL

	DeviceID    uint           `json:"deviceId" gorm:"uniqueIndex;comment:关联的设备ID"`
	MethodsJSON datatypes.JSON `json:"methodsJson" gorm:"type:json;comment:RPC方法列表(JSON)"`
}

func (DeviceRPCMethods) TableName() string {
	return "tr069_device_rpc_methods"
}

