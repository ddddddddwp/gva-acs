package response

import (
	"time"
)

// TR069DeviceResponse 设备响应
type TR069DeviceResponse struct {
	ID           uint      `json:"id"`
	Manufacturer string    `json:"manufacturer"`
	ProductClass string    `json:"productClass"`
	SerialNumber string    `json:"serialNumber"`
	HardwareVer  string    `json:"hardwareVer"`
	SoftwareVer  string    `json:"softwareVer"`
	Status       string    `json:"status"`
	LastInform   time.Time `json:"lastInform"`
	IPAddress    string    `json:"ipAddress"`
	MACAddress   string    `json:"macAddress"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

// TR069EventResponse 事件响应
type TR069EventResponse struct {
	ID          uint      `json:"id"`
	DeviceID    uint      `json:"deviceId"`
	EventType   string    `json:"eventType"`
	EventCode   string    `json:"eventCode"`
	Description string    `json:"description"`
	Timestamp   time.Time `json:"timestamp"`
	CreatedAt   time.Time `json:"createdAt"`
}

// TR069DeviceStats 设备统计信息
type TR069DeviceStats struct {
	TotalDevices    int `json:"totalDevices"`
	OnlineDevices   int `json:"onlineDevices"`
	OfflineDevices  int `json:"offlineDevices"`
	EventsLast24h   int `json:"eventsLast24h"`
	NewDevicesLast7d int `json:"newDevicesLast7d"`
}