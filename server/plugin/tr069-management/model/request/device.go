package request

import (
	"github.com/flipped-aurora/gin-vue-admin/server/model/common/request"
)

// DeviceSearch 设备搜索请求
type DeviceSearch struct {
	request.PageInfo
	SerialNumber string `json:"serialNumber" form:"serialNumber"`
	Manufacturer string `json:"manufacturer" form:"manufacturer"`
	ProductClass string `json:"productClass" form:"productClass"`
	ModelName    string `json:"modelName" form:"modelName"`
	Status       *int   `json:"status" form:"status"`
	GroupID      *uint  `json:"groupId" form:"groupId"`
	Tags         string `json:"tags" form:"tags"`
}

// CreateDeviceRequest 创建设备请求
type CreateDeviceRequest struct {
	SerialNumber     string `json:"serialNumber" binding:"required"`
	Manufacturer     string `json:"manufacturer" binding:"required"`
	OUI              string `json:"oui"`
	ProductClass     string `json:"productClass"`
	ModelName        string `json:"modelName"`
	HardwareVersion  string `json:"hardwareVersion"`
	SoftwareVersion  string `json:"softwareVersion"`
	ProvisioningCode string `json:"provisioningCode"`
	Description      string `json:"description"`
	Tags             string `json:"tags"`
	Location         string `json:"location"`
	GroupID          *uint  `json:"groupId"`
	ConfigProfileID  *uint  `json:"configProfileId"`
}

// UpdateDeviceRequest 更新设备请求
type UpdateDeviceRequest struct {
	ID               uint   `json:"id" binding:"required"`
	ModelName        string `json:"modelName"`
	HardwareVersion  string `json:"hardwareVersion"`
	SoftwareVersion  string `json:"softwareVersion"`
	ProvisioningCode string `json:"provisioningCode"`
	Description      string `json:"description"`
	Tags             string `json:"tags"`
	Location         string `json:"location"`
	GroupID          *uint  `json:"groupId"`
	ConfigProfileID  *uint  `json:"configProfileId"`
	PeriodicInform   *int   `json:"periodicInform"`
}

// BatchDeleteRequest 批量删除请求
type BatchDeleteRequest struct {
	IDs []uint `json:"ids" binding:"required"`
}