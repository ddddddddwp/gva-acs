package request

import (
	"github.com/flipped-aurora/gin-vue-admin/server/model/common/request"
)

// DeviceSearch 设备搜索请求
type DeviceSearch struct {
	request.PageInfo
	SerialNumber    string `json:"serialNumber" form:"serialNumber"`
	ProductClass    string `json:"productClass" form:"productClass"`
	Manufacturer    string `json:"manufacturer" form:"manufacturer"`
	OUI             string `json:"oui" form:"oui"`
	ModelName       string `json:"modelName" form:"modelName"`
	Status          *int   `json:"status" form:"status"`
	SoftwareVersion string `json:"softwareVersion" form:"softwareVersion"`
	Tags            string `json:"tags" form:"tags"`
	OnlineOnly      bool   `json:"onlineOnly" form:"onlineOnly"`
}

// CreateDeviceRequest 创建设备请求
type CreateDeviceRequest struct {
	SerialNumber     string `json:"serialNumber" binding:"required" validate:"required"`
	ProductClass     string `json:"productClass"`
	Manufacturer     string `json:"manufacturer"`
	OUI              string `json:"oui" binding:"required" validate:"required"`
	ModelName        string `json:"modelName"`
	Description      string `json:"description"`
	ProvisioningCode string `json:"provisioningCode"`
	SoftwareVersion  string `json:"softwareVersion"`
	HardwareVersion  string `json:"hardwareVersion"`
	SpecVersion      string `json:"specVersion"`
	ConnectionURL    string `json:"connectionUrl"`
	Username         string `json:"username"`
	Password         string `json:"password"`
	PeriodicInform   int    `json:"periodicInform"`
	Tags             string `json:"tags"`
}

// UpdateDeviceRequest 更新设备请求
type UpdateDeviceRequest struct {
	ID               uint   `json:"id" binding:"required"`
	SerialNumber     string `json:"serialNumber" binding:"required"`
	ProductClass     string `json:"productClass"`
	Manufacturer     string `json:"manufacturer"`
	OUI              string `json:"oui" binding:"required"`
	ModelName        string `json:"modelName"`
	Description      string `json:"description"`
	ProvisioningCode string `json:"provisioningCode"`
	SoftwareVersion  string `json:"softwareVersion"`
	HardwareVersion  string `json:"hardwareVersion"`
	SpecVersion      string `json:"specVersion"`
	ConnectionURL    string `json:"connectionUrl"`
	Username         string `json:"username"`
	Password         string `json:"password"`
	PeriodicInform   int    `json:"periodicInform"`
	Status           int    `json:"status"`
	Tags             string `json:"tags"`
}

// DeviceByID 根据ID获取设备请求
type DeviceByID struct {
	ID uint `json:"id" form:"id" binding:"required"`
}

// DeviceBySerialNumber 根据序列号获取设备请求
type DeviceBySerialNumber struct {
	SerialNumber string `json:"serialNumber" form:"serialNumber" binding:"required"`
}

// BatchDeviceOperation 批量设备操作请求
type BatchDeviceOperation struct {
	DeviceIDs []uint `json:"deviceIds" binding:"required,min=1"`
	Operation string `json:"operation" binding:"required"`
	Parameters map[string]interface{} `json:"parameters"`
}