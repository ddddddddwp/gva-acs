package model

import "time"

type CommandEvent struct {
	ID          uint64       `json:"id" gorm:"primaryKey"`
	CommandID   string       `json:"commandId" gorm:"size:64;index"`
	EventType   string       `json:"eventType" gorm:"size:48;index"`
	FromStatus  string       `json:"fromStatus" gorm:"size:24"`
	ToStatus    string       `json:"toStatus" gorm:"size:24"`
	Stage       string       `json:"stage" gorm:"size:64"`
	Message     string       `json:"message" gorm:"type:text"`
	PayloadJSON LongTextJSON `json:"payload" gorm:"type:longtext"`
	CreatedAt   time.Time    `json:"createdAt" gorm:"index"`
}

func (CommandEvent) TableName() string {
	return "tr069_command_events"
}
