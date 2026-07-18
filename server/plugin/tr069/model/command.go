package model

import (
	"database/sql/driver"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

const (
	CommandOriginUser   = "USER"
	CommandOriginSystem = "SYSTEM"
)

// LongTextJSON preserves datatypes.JSON's scan and JSON encoding behavior while
// keeping legacy command JSON columns as text instead of strict MySQL JSON.
type LongTextJSON datatypes.JSON

func (j LongTextJSON) Value() (driver.Value, error) {
	return datatypes.JSON(j).Value()
}

func (j *LongTextJSON) Scan(value any) error {
	var decoded datatypes.JSON
	if err := decoded.Scan(value); err != nil {
		return err
	}
	*j = LongTextJSON(decoded)
	return nil
}

func (j LongTextJSON) MarshalJSON() ([]byte, error) {
	return datatypes.JSON(j).MarshalJSON()
}

func (j *LongTextJSON) UnmarshalJSON(value []byte) error {
	var decoded datatypes.JSON
	if err := decoded.UnmarshalJSON(value); err != nil {
		return err
	}
	*j = LongTextJSON(decoded)
	return nil
}

func (j LongTextJSON) String() string {
	return string(j)
}

func (LongTextJSON) GormDataType() string {
	return "string"
}

func (LongTextJSON) GormDBDataType(db *gorm.DB, _ *schema.Field) string {
	switch db.Dialector.Name() {
	case "mysql", "sqlite":
		return "LONGTEXT"
	case "sqlserver":
		return "NVARCHAR(MAX)"
	default:
		return "TEXT"
	}
}

type Command struct {
	CommandID       string       `json:"commandId" gorm:"primaryKey;size:64"`
	DeviceID        uint         `json:"deviceId" gorm:"index:idx_tr069_command_device_head,priority:1"`
	DeviceKey       string       `json:"deviceKey" gorm:"size:128;index"`
	Operation       string       `json:"operation" gorm:"size:64;index"`
	Origin          string       `json:"origin" gorm:"size:16;index"`
	ParamsJSON      LongTextJSON `json:"params" gorm:"type:longtext"`
	ResultJSON      LongTextJSON `json:"result" gorm:"type:longtext"`
	RetryOf         string       `json:"retryOf" gorm:"size:64;index"`
	DedupKey        string       `json:"dedupKey" gorm:"size:128;index"`
	CommandKey      *string      `json:"commandKey" gorm:"size:128;uniqueIndex"`
	Status          string       `json:"status" gorm:"size:24;index;index:idx_tr069_command_device_head,priority:2"`
	CWMPID          string       `json:"cwmpId" gorm:"column:cwmp_id;size:64;index"`
	PhaseDeadlineAt *time.Time   `json:"phaseDeadlineAt" gorm:"index"`
	QueuedAt        time.Time    `json:"queuedAt"`
	WaitingAt       *time.Time   `json:"waitingAt"`
	BuildingAt      *time.Time   `json:"buildingAt"`
	SentAt          *time.Time   `json:"sentAt"`
	FinishedAt      *time.Time   `json:"finishedAt"`
	FailureStage    string       `json:"failureStage" gorm:"size:64"`
	FaultCode       int          `json:"faultCode"`
	FaultString     string       `json:"faultString" gorm:"type:text"`
	Version         uint         `json:"version"`
	CreatedAt       time.Time    `json:"createdAt" gorm:"index:idx_tr069_command_device_head,priority:3"`
	UpdatedAt       time.Time    `json:"updatedAt"`
}

func (Command) TableName() string {
	return "tr069_commands"
}
