package model

import (
	"time"

	"github.com/ddddddddwp/gva-acs/server/global"
	"gorm.io/datatypes"
)

type DataModelValue struct {
	global.GVA_MODEL

	DeviceID uint   `json:"deviceId" gorm:"index:idx_tr069_dm_device_param,unique;comment:关联的设备ID"`
	Name     string `json:"name" gorm:"size:512;index:idx_tr069_dm_device_param,unique;comment:参数/对象名称"`

	Writable  bool           `json:"writable" gorm:"comment:是否可写"`
	ValueType string         `json:"valueType" gorm:"size:64;comment:值类型(xsd:*)"`
	ValueJSON datatypes.JSON `json:"valueJson" gorm:"type:json;comment:值(JSON)"`

	LastCollectedAt time.Time `json:"-" gorm:"index;comment:最后采集时间"`
}

func (DataModelValue) TableName() string {
	return "tr069_datamodel_values"
}
