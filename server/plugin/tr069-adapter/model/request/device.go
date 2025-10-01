package request

import (
	"github.com/flipped-aurora/gin-vue-admin/server/model/common/request"
)

// TR069DeviceSearch 设备查询请求
type TR069DeviceSearch struct {
	request.PageInfo
	Manufacturer string `json:"manufacturer" form:"manufacturer"`
	ProductClass string `json:"productClass" form:"productClass"`
	SerialNumber string `json:"serialNumber" form:"serialNumber"`
	Status       string `json:"status" form:"status"`
	IPAddress    string `json:"ipAddress" form:"ipAddress"`
	MACAddress   string `json:"macAddress" form:"macAddress"`
}

// TR069DeviceParam 设备参数设置请求
type TR069DeviceParam struct {
	DeviceID  uint                   `json:"deviceId" form:"deviceId"`
	ParamPath string                 `json:"paramPath" form:"paramPath"`
	ParamType string                 `json:"paramType" form:"paramType"`
	Value     interface{}            `json:"value" form:"value"`
	Params    map[string]interface{} `json:"params" form:"params"`
}

// TR069CommandRequest 设备命令请求
type TR069CommandRequest struct {
	DeviceID uint                   `json:"deviceId" form:"deviceId"`
	Command  string                 `json:"command" form:"command"`
	Params   map[string]interface{} `json:"params" form:"params"`
}