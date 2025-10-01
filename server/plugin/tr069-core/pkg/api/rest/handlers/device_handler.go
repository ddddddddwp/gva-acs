package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/root/demo/tr069/interfaces"
	"github.com/root/demo/tr069/pkg/api/rest"
)

// DeviceHandler 处理设备相关的API请求
type DeviceHandler struct {
	parser   interfaces.Parser
	builder  interfaces.Builder
}

// NewDeviceHandler 创建设备处理器
func NewDeviceHandler(parser interfaces.Parser, builder interfaces.Builder) *DeviceHandler {
	return &DeviceHandler{
		parser:  parser,
		builder: builder,
	}
}

// RegisterRoutes 注册API路由
func (h *DeviceHandler) RegisterRoutes(router interfaces.Router) {
	router.Handle(http.MethodGet, "/devices", h.ListDevices)
	router.Handle(http.MethodGet, "/devices/{id}", h.GetDevice)
	router.Handle(http.MethodPost, "/devices/{id}/inform", h.ProcessInform)
	router.Handle(http.MethodPost, "/devices/{id}/configure", h.ConfigureDevice)
	router.Handle(http.MethodGet, "/devices/{id}/parameters", h.GetParameters)
	router.Handle(http.MethodPut, "/devices/{id}/parameters", h.SetParameters)
}

// SetMiddleware 设置中间件
func (h *DeviceHandler) SetMiddleware(middleware ...interfaces.Middleware) {
	// 这里可以存储特定于处理器的中间件
}

// ListDevices 列出所有设备
func (h *DeviceHandler) ListDevices(w http.ResponseWriter, r *http.Request) {
	// 模拟设备列表
	devices := []map[string]interface{}{
		{"id": "device1", "serialNumber": "SN001", "manufacturer": "Vendor1", "model": "Model1", "status": "online"},
		{"id": "device2", "serialNumber": "SN002", "manufacturer": "Vendor2", "model": "Model2", "status": "offline"},
	}
	
	rest.SendSuccessResponse(w, devices)
}

// GetDevice 获取单个设备信息
func (h *DeviceHandler) GetDevice(w http.ResponseWriter, r *http.Request) {
	// 从URL中提取设备ID
	id := extractIDFromPath(r.URL.Path)
	if id == "" {
		rest.SendErrorResponse(w, http.StatusBadRequest, "Invalid device ID", 400)
		return
	}
	
	// 模拟设备信息
	device := map[string]interface{}{
		"id":           id,
		"serialNumber": "SN" + id,
		"manufacturer": "Vendor",
		"model":        "Model",
		"status":       "online",
		"lastContact":  "2023-05-01T12:00:00Z",
		"ipAddress":    "192.168.1.100",
		"macAddress":   "00:11:22:33:44:55",
	}
	
	rest.SendSuccessResponse(w, device)
}

// ProcessInform 处理设备Inform请求
func (h *DeviceHandler) ProcessInform(w http.ResponseWriter, r *http.Request) {
	// 从URL中提取设备ID
	id := extractIDFromPath(r.URL.Path)
	if id == "" {
		rest.SendErrorResponse(w, http.StatusBadRequest, "Invalid device ID", 400)
		return
	}
	
	// 解析请求体
	var informRequest map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&informRequest); err != nil {
		rest.SendErrorResponse(w, http.StatusBadRequest, "Invalid request body: "+err.Error(), 400)
		return
	}
	
	// 处理Inform消息
	// 在实际应用中，这里会使用parser解析XML消息，然后处理
	
	// 构建响应
	response := map[string]interface{}{
		"deviceId": id,
		"status":   "success",
		"message":  "Inform processed successfully",
	}
	
	rest.SendSuccessResponse(w, response)
}

// ConfigureDevice 配置设备
func (h *DeviceHandler) ConfigureDevice(w http.ResponseWriter, r *http.Request) {
	// 从URL中提取设备ID
	id := extractIDFromPath(r.URL.Path)
	if id == "" {
		rest.SendErrorResponse(w, http.StatusBadRequest, "Invalid device ID", 400)
		return
	}
	
	// 解析请求体
	var configRequest map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&configRequest); err != nil {
		rest.SendErrorResponse(w, http.StatusBadRequest, "Invalid request body: "+err.Error(), 400)
		return
	}
	
	// 处理配置请求
	// 在实际应用中，这里会使用builder构建TR069消息
	
	// 构建响应
	response := map[string]interface{}{
		"deviceId": id,
		"status":   "success",
		"message":  "Device configured successfully",
	}
	
	rest.SendSuccessResponse(w, response)
}

// GetParameters 获取设备参数
func (h *DeviceHandler) GetParameters(w http.ResponseWriter, r *http.Request) {
	// 从URL中提取设备ID
	id := extractIDFromPath(r.URL.Path)
	if id == "" {
		rest.SendErrorResponse(w, http.StatusBadRequest, "Invalid device ID", 400)
		return
	}
	
	// 解析查询参数
	names := r.URL.Query()["name"]
	
	// 模拟参数列表
	parameters := []map[string]interface{}{
		{"name": "Device.DeviceInfo.Manufacturer", "value": "Vendor", "type": "string", "writable": false},
		{"name": "Device.DeviceInfo.ModelName", "value": "Model", "type": "string", "writable": false},
		{"name": "Device.DeviceInfo.SerialNumber", "value": "SN" + id, "type": "string", "writable": false},
		{"name": "Device.ManagementServer.URL", "value": "http://acs.example.com", "type": "string", "writable": true},
	}
	
	// 如果指定了参数名，过滤结果
	if len(names) > 0 {
		filteredParams := make([]map[string]interface{}, 0)
		for _, param := range parameters {
			for _, name := range names {
				if param["name"] == name {
					filteredParams = append(filteredParams, param)
					break
				}
			}
		}
		parameters = filteredParams
	}
	
	rest.SendSuccessResponse(w, parameters)
}

// SetParameters 设置设备参数
func (h *DeviceHandler) SetParameters(w http.ResponseWriter, r *http.Request) {
	// 从URL中提取设备ID
	id := extractIDFromPath(r.URL.Path)
	if id == "" {
		rest.SendErrorResponse(w, http.StatusBadRequest, "Invalid device ID", 400)
		return
	}
	
	// 解析请求体
	var paramRequest struct {
		Parameters []struct {
			Name  string      `json:"name"`
			Value interface{} `json:"value"`
		} `json:"parameters"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&paramRequest); err != nil {
		rest.SendErrorResponse(w, http.StatusBadRequest, "Invalid request body: "+err.Error(), 400)
		return
	}
	
	// 处理参数设置请求
	// 在实际应用中，这里会使用builder构建TR069消息
	
	// 构建响应
	response := map[string]interface{}{
		"deviceId": id,
		"status":   "success",
		"message":  "Parameters set successfully",
		"updated":  len(paramRequest.Parameters),
	}
	
	rest.SendSuccessResponse(w, response)
}

// 从路径中提取ID
func extractIDFromPath(path string) string {
	parts := strings.Split(path, "/")
	for i, part := range parts {
		if part == "devices" && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return ""
}