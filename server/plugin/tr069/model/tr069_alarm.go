package model

import (
	"github.com/ddddddddwp/gva-acs/server/global"
	"time"
)

type Tr069Alarm struct {
	global.GVA_MODEL
	DeviceID              uint       `json:"deviceId" gorm:"index;comment:关联设备ID"`
	SerialNumber          string     `json:"serialNumber" gorm:"index;size:64;comment:设备序列号"`
	OUI                   string     `json:"oui" gorm:"column:oui;size:64;comment:组织唯一标识"`
	AlarmIdentifier       string     `json:"alarmIdentifier" gorm:"uniqueIndex:idx_alarm_ident;size:128;comment:告警唯一标识"`
	Source                string     `json:"source" gorm:"index;size:32;comment:来源(CurrentAlarm/ExpeditedEvent/HistoryEvent)"`
	NotificationType      string     `json:"notificationType" gorm:"index;size:32;comment:通知类型(NewAlarm/ClearedAlarm)"`
	Status                string     `json:"status" gorm:"index;size:16;default:Active;comment:状态(Active/Cleared)"`
	EventType             string     `json:"eventType" gorm:"size:64;comment:事件类型"`
	PerceivedSeverity     string     `json:"perceivedSeverity" gorm:"index;size:32;comment:严重程度(Critical/Major/Minor/Warning/Indeterminate)"`
	ProbableCause         string     `json:"probableCause" gorm:"size:128;comment:可能原因"`
	SpecificProblem       string     `json:"specificProblem" gorm:"size:256;comment:具体问题"`
	AdditionalText        string     `json:"additionalText" gorm:"size:256;comment:附加文本"`
	AdditionalInformation string     `json:"additionalInformation" gorm:"type:text;comment:附加信息"`
	ManagedObjectInstance string     `json:"managedObjectInstance" gorm:"size:256;comment:管理对象实例"`
	EventTime             time.Time  `json:"eventTime" gorm:"index;comment:事件时间"`
	StartTime             time.Time  `json:"startTime" gorm:"index;comment:告警开始时间"`
	EndTime               *time.Time `json:"endTime" gorm:"index;comment:告警结束时间"`
	LastChanged           time.Time  `json:"lastChanged" gorm:"comment:最后变更时间"`
}

func (Tr069Alarm) TableName() string {
	return "tr069_alarms"
}

func (a *Tr069Alarm) IsCleared() bool {
	return a.Status == "Cleared" || a.NotificationType == "ClearedAlarm"
}

func (a *Tr069Alarm) IsActive() bool {
	return a.Status == "Active" && a.EndTime == nil
}
