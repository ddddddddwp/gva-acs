package model

import (
	"time"
)

// DeviceInfo 设备信息
type DeviceInfo struct {
	SerialNumber  string    `json:"serial_number"`
	ProductClass  string    `json:"product_class"`
	Manufacturer  string    `json:"manufacturer"`
	OUI           string    `json:"oui"`
	RemoteAddr    string    `json:"remote_addr"`
	UserAgent     string    `json:"user_agent"`
	Timestamp     time.Time `json:"timestamp"`
}

// Inform TR069 Inform消息结构体
type Inform struct {
	ID            string                 `json:"id"`
	DeviceId      DeviceID               `json:"device_id"`
	Event         []EventStruct          `json:"event"`
	MaxEnvelopes  int                    `json:"max_envelopes"`
	CurrentTime   string                 `json:"current_time"`
	RetryCount    int                    `json:"retry_count"`
	ParameterList []ParameterValueStruct `json:"parameter_list"`
}

// ParameterValueStruct 参数值结构体
type ParameterValueStruct struct {
	Name  string `json:"name"`
	Value string `json:"value"`
	Type  string `json:"type"`
}

// ParameterInfoStruct 参数信息结构体
type ParameterInfoStruct struct {
	Name     string `json:"name"`
	Writable bool   `json:"writable"`
}

// ParameterAttributeStruct 参数属性结构体
type ParameterAttributeStruct struct {
	Name         string `json:"name"`
	Notification int    `json:"notification"`
	AccessList   string `json:"access_list"`
}

// DeviceID 设备ID结构体
type DeviceID struct {
	Manufacturer string `json:"manufacturer"`
	OUI          string `json:"oui"`
	ProductClass string `json:"product_class"`
	SerialNumber string `json:"serial_number"`
}

// EventStruct 事件结构体
type EventStruct struct {
	EventCode  string `json:"event_code"`
	CommandKey string `json:"command_key"`
}