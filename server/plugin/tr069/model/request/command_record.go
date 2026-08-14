package request

type CommandRecordListRequest struct {
	Page         int    `form:"page"`
	PageSize     int    `form:"pageSize"`
	DeviceID     uint   `form:"deviceId"`
	DeviceSerial string `form:"deviceSerial"`
	Operation    string `form:"operation"`
	Status       string `form:"status"`
	CommandID    string `form:"commandId"`
	CreatedFrom  string `form:"createdFrom"`
	CreatedTo    string `form:"createdTo"`
}
