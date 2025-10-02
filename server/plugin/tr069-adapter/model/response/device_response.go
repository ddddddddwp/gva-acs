package response

import (
	"time"
	"github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-adapter/model"
)

// DeviceResponse 设备响应
type DeviceResponse struct {
	ID               uint      `json:"id"`
	SerialNumber     string    `json:"serialNumber"`
	ProductClass     string    `json:"productClass"`
	Manufacturer     string    `json:"manufacturer"`
	OUI              string    `json:"oui"`
	ModelName        string    `json:"modelName"`
	Description      string    `json:"description"`
	ProvisioningCode string    `json:"provisioningCode"`
	SoftwareVersion  string    `json:"softwareVersion"`
	HardwareVersion  string    `json:"hardwareVersion"`
	SpecVersion      string    `json:"specVersion"`
	ConnectionURL    string    `json:"connectionUrl"`
	Username         string    `json:"username"`
	PeriodicInform   int       `json:"periodicInform"`
	Status           int       `json:"status"`
	StatusText       string    `json:"statusText"`
	LastInform       time.Time `json:"lastInform"`
	LastBootstrap    time.Time `json:"lastBootstrap"`
	Tags             string    `json:"tags"`
	IsOnline         bool      `json:"isOnline"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
}

// DeviceListResponse 设备列表响应
type DeviceListResponse struct {
	List     []DeviceResponse `json:"list"`
	Total    int64            `json:"total"`
	Page     int              `json:"page"`
	PageSize int              `json:"pageSize"`
}

// DeviceDetailResponse 设备详情响应
type DeviceDetailResponse struct {
	DeviceResponse
	Parameters    []model.CpeParameter    `json:"parameters,omitempty"`
	Sessions      []model.CpeSession      `json:"sessions,omitempty"`
	OperationLogs []model.CpeOperationLog `json:"operationLogs,omitempty"`
}

// DeviceStatsResponse 设备统计响应
type DeviceStatsResponse struct {
	Total        int64 `json:"total"`
	Online       int64 `json:"online"`
	Offline      int64 `json:"offline"`
	Fault        int64 `json:"fault"`
	LastHour     int64 `json:"lastHour"`
	LastDay      int64 `json:"lastDay"`
	LastWeek     int64 `json:"lastWeek"`
	LastMonth    int64 `json:"lastMonth"`
}

// DeviceOperationResponse 设备操作响应
type DeviceOperationResponse struct {
	DeviceID    uint   `json:"deviceId"`
	Operation   string `json:"operation"`
	Status      string `json:"status"`
	Message     string `json:"message"`
	RequestID   string `json:"requestId,omitempty"`
}

// BatchOperationResponse 批量操作响应
type BatchOperationResponse struct {
	Total     int                       `json:"total"`
	Success   int                       `json:"success"`
	Failed    int                       `json:"failed"`
	Results   []DeviceOperationResponse `json:"results"`
}