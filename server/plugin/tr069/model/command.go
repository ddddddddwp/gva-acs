package model

import "time"

type Command struct {
	CommandID string `json:"commandId" gorm:"primaryKey;size:64;comment:命令ID"`

	DeviceKey  string `json:"deviceKey" gorm:"index;size:128;comment:设备Key(OUI-SN)"`
	Operation  string `json:"operation" gorm:"size:64;comment:操作类型(op)"`
	ParamsJSON string `json:"paramsJson" gorm:"type:text;comment:参数JSON"`
	DedupKey   string `json:"dedupKey" gorm:"index;size:128;comment:幂等键"`

	Status    string     `json:"status" gorm:"index;size:16;comment:状态(PENDING/SENDING/SUCCESS/FAIL)"`
	RequestID string     `json:"requestId" gorm:"index;size:64;comment:CWMP请求ID"`
	SentAt    *time.Time `json:"sentAt" gorm:"comment:下发时间"`

	FinishedAt  *time.Time `json:"finishedAt" gorm:"comment:完成时间"`
	FaultCode   int        `json:"faultCode" gorm:"comment:故障码"`
	FaultString string     `json:"faultString" gorm:"type:text;comment:故障描述"`

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (Command) TableName() string {
	return "tr069_commands"
}
