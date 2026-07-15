package model

import "time"

type CommandXML struct {
	ID        uint64    `json:"id" gorm:"primaryKey"`
	CommandID string    `json:"commandId" gorm:"size:64;index"`
	Direction string    `json:"direction" gorm:"size:12;index"`
	Method    string    `json:"method" gorm:"size:64;index"`
	CWMPID    string    `json:"cwmpId" gorm:"size:64;index"`
	RequestID string    `json:"requestId" gorm:"size:64;index"`
	Payload   []byte    `json:"xml" gorm:"type:longblob"`
	ExpiresAt time.Time `json:"expiresAt" gorm:"index"`
	CreatedAt time.Time `json:"createdAt"`
}

func (CommandXML) TableName() string {
	return "tr069_command_xmls"
}
