package model

import (
	"github.com/ddddddddwp/gva-acs/server/global"
)

type SupportTr069Alarm struct {
	global.GVA_MODEL
	DeviceID          uint   `json:"deviceId" gorm:"index;comment:关联设备ID"`
	SerialNumber      string `json:"serialNumber" gorm:"index;size:64;comment:设备序列号"`
	EventType         string `json:"eventType" gorm:"size:64;comment:事件类型"`
	PerceivedSeverity string `json:"perceivedSeverity" gorm:"size:32;comment:严重程度"`
	ProbableCause     string `json:"probableCause" gorm:"size:128;comment:可能原因"`
	SpecificProblem   string `json:"specificProblem" gorm:"size:256;comment:具体问题"`
}

func (SupportTr069Alarm) TableName() string {
	return "support_tr069_alarms"
}
