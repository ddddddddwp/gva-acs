package model

import (
	"github.com/ddddddddwp/gva-acs/server/global"
	"time"
)

type Tr069AlarmEvent struct {
	global.GVA_MODEL
	DeviceID         uint      `json:"deviceId" gorm:"index;comment:关联设备ID"`
	SerialNumber     string    `json:"serialNumber" gorm:"index;comment:冗余设备序列号"`
	AlarmIdentifier  string    `json:"alarmIdentifier" gorm:"index;comment:告警唯一标识"`
	NotificationType string    `json:"notificationType" gorm:"comment:告警变更类型(NewAlarm/ClearedAlarm/ChangedAlarm)"`
	Severity         string    `json:"severity" gorm:"comment:严重程度"`
	SpecificProblem  string    `json:"specificProblem" gorm:"comment:具体问题"`
	ProbableCause    string    `json:"probableCause" gorm:"comment:可能原因"`
	EventType        string    `json:"eventType" gorm:"comment:事件类型"`
	AdditionalText   string    `json:"additionalText" gorm:"comment:附加文本"`
	AddInfo          string    `json:"addInfo" gorm:"comment:附加信息"`
	EventTime        time.Time `json:"eventTime" gorm:"index;comment:事件时间"`
}

func (Tr069AlarmEvent) TableName() string {
	return "tr069_alarm_events"
}
