package response

import (
	"time"
)

// DeviceResponse 设备列表响应结构
type DeviceResponse struct {
	ID               uint      `json:"ID"`
	SerialNumber     string    `json:"serialNumber"`
	OUI              string    `json:"oui"`
	ProductClass     string    `json:"productClass"`
	Manufacturer     string    `json:"manufacturer"`
	ModelName        string    `json:"modelName"`
	LastInform       time.Time `json:"lastInform"`
	UpTime           uint64    `json:"upTime"`
	IP               string    `json:"ip"`
	MacAddress       string    `json:"macAddress"`
	ConnectionReqURL string    `json:"connectionReqUrl"`
	SoftwareVer      string    `json:"softwareVer"`
	HardwareVer      string    `json:"hardwareVer"`
	SpecVer          string    `json:"specVer"`
	PhysicalCellID   uint      `json:"pci"`
	CellID           string    `json:"cellId"`
	GroupId          uint      `json:"groupId"`
	Remark           string    `json:"remark"`
	IsWhite          bool      `json:"isWhite"`
	Online           bool      `json:"online"` // 设备在线状态
	RPCMethods       []string  `json:"rpcMethods"`
}
