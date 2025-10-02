package request

// DeviceSerialNumberRequest 设备序列号请求
type DeviceSerialNumberRequest struct {
	SerialNumber string `json:"serialNumber" form:"serialNumber" binding:"required"`
}

// DeviceActionRequest 设备操作请求
type DeviceActionRequest struct {
	SerialNumber string `json:"serialNumber" form:"serialNumber" binding:"required"`
	Action       string `json:"action" form:"action" binding:"required"`
}

// DeviceEventsRequest 设备事件请求
type DeviceEventsRequest struct {
	SerialNumber string `json:"serialNumber" form:"serialNumber" binding:"required"`
	Page         int    `json:"page" form:"page" binding:"required,min=1"`
	PageSize     int    `json:"pageSize" form:"pageSize" binding:"required,min=1,max=100"`
}

// DeviceSessionsRequest 设备会话请求
type DeviceSessionsRequest struct {
	SerialNumber string `json:"serialNumber" form:"serialNumber" binding:"required"`
	Page         int    `json:"page" form:"page" binding:"required,min=1"`
	PageSize     int    `json:"pageSize" form:"pageSize" binding:"required,min=1,max=100"`
}

// DeviceLogsRequest 设备日志请求
type DeviceLogsRequest struct {
	SerialNumber string `json:"serialNumber" form:"serialNumber" binding:"required"`
	Page         int    `json:"page" form:"page" binding:"required,min=1"`
	PageSize     int    `json:"pageSize" form:"pageSize" binding:"required,min=1,max=100"`
}