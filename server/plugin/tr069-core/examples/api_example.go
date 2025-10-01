package main

import (
"context"
"encoding/json"
"fmt"
"log"
"net/http"
"os"
"os/signal"
"time"

"github.com/root/demo/tr069/interfaces"
"github.com/root/demo/tr069/factory"
"github.com/root/demo/tr069/pkg/api"
)

// DeviceHandler 处理设备相关API
type DeviceHandler struct {
converter *api.TR069Converter
router    interfaces.Router
logger    interfaces.Logger
}

// NewDeviceHandler 创建设备处理器
func NewDeviceHandler(converter *api.TR069Converter, logger interfaces.Logger) *DeviceHandler {
return &DeviceHandler{
converter: converter,
logger:    logger,
}
}

// RegisterRoutes 实现APIHandler接口
func (h *DeviceHandler) RegisterRoutes(router interfaces.Router) {
h.router = router

// 设备信息API
router.Handle("GET", "/devices/{deviceId}", h.getDeviceInfo)

// 设备参数API
router.Handle("GET", "/devices/{deviceId}/parameters", h.getParameters)
router.Handle("PUT", "/devices/{deviceId}/parameters", h.setParameters)

// RPC方法API
router.Handle("POST", "/devices/{deviceId}/rpc", h.executeRPC)
router.Handle("GET", "/devices/{deviceId}/rpc/{commandKey}", h.getRPCStatus)

// 事件API
router.Handle("GET", "/devices/{deviceId}/events", h.getEvents)
}

// SetMiddleware 实现APIHandler接口
func (h *DeviceHandler) SetMiddleware(middleware ...interfaces.Middleware) {
if h.router != nil {
h.router.Use(middleware...)
}
}

// API处理函数
func (h *DeviceHandler) getDeviceInfo(w http.ResponseWriter, r *http.Request) {
// 从URL中提取设备ID
vars := r.Context().Value("vars").(map[string]string)
deviceID := vars["deviceId"]

// 模拟获取设备信息
deviceInfo := map[string]interface{}{
"deviceId":        deviceID,
"manufacturer":    "Example Inc.",
"model":           "Gateway-1000",
"serialNumber":    "SN12345678",
"softwareVersion": "1.2.3",
"status":          "online",
}

// 返回响应
sendJSONResponse(w, true, deviceInfo, "", 0)
}

func (h *DeviceHandler) getParameters(w http.ResponseWriter, r *http.Request) {
// 从URL中提取设备ID
vars := r.Context().Value("vars").(map[string]string)
deviceID := vars["deviceId"]

// 获取参数名称列表
names := r.URL.Query().Get("names")

h.logger.Debug("Getting parameters", "deviceId", deviceID, "names", names)

// 模拟参数数据
parameters := []map[string]interface{}{
{
"name":  "Device.DeviceInfo.Manufacturer",
"value": "Example Inc.",
"type":  "string",
},
{
"name":  "Device.DeviceInfo.ModelName",
"value": "Gateway-1000",
"type":  "string",
},
}

// 返回响应
sendJSONResponse(w, true, map[string]interface{}{"parameters": parameters}, "", 0)
}

func (h *DeviceHandler) setParameters(w http.ResponseWriter, r *http.Request) {
// 从URL中提取设备ID
vars := r.Context().Value("vars").(map[string]string)
deviceID := vars["deviceId"]

// 解析请求体
var requestBody map[string]interface{}
if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
sendJSONResponse(w, false, nil, "Invalid request body", 101)
return
}

// 获取参数列表
parameters, ok := requestBody["parameters"].([]interface{})
if !ok {
sendJSONResponse(w, false, nil, "Invalid parameters format", 101)
return
}

h.logger.Info("Setting parameters", "deviceId", deviceID, "count", len(parameters))

// 返回成功响应
sendJSONResponse(w, true, map[string]interface{}{
"status":           "success",
"updatedParameters": len(parameters),
}, "", 0)
}

func (h *DeviceHandler) executeRPC(w http.ResponseWriter, r *http.Request) {
// 从URL中提取设备ID
vars := r.Context().Value("vars").(map[string]string)
deviceID := vars["deviceId"]

// 解析请求体
var requestBody map[string]interface{}
if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
sendJSONResponse(w, false, nil, "Invalid request body", 101)
return
}

// 获取方法和参数
method, ok := requestBody["method"].(string)
if !ok {
sendJSONResponse(w, false, nil, "Missing method", 101)
return
}

params, _ := requestBody["params"].(map[string]interface{})

h.logger.Info("Executing RPC", "deviceId", deviceID, "method", method)

// 转换为TR069格式
tr069Data, err := h.converter.JSONToTR069([]byte(fmt.Sprintf(`{"method":"%s","params":%v}`, method, params)))
if err != nil {
sendJSONResponse(w, false, nil, fmt.Sprintf("Failed to convert to TR069: %v", err), 105)
return
}

// 模拟处理TR069请求
h.logger.Debug("TR069 request data", "data", string(tr069Data))

// 返回成功响应
sendJSONResponse(w, true, map[string]interface{}{
"status": "success",
}, "", 0)
}

func (h *DeviceHandler) getRPCStatus(w http.ResponseWriter, r *http.Request) {
// 从URL中提取设备ID和命令ID
vars := r.Context().Value("vars").(map[string]string)
deviceID := vars["deviceId"]
commandKey := vars["commandKey"]

h.logger.Debug("Getting RPC status", "deviceId", deviceID, "commandKey", commandKey)

// 模拟RPC状态
status := map[string]interface{}{
"status":       "completed",
"startTime":    time.Now().Add(-90 * time.Second).Format(time.RFC3339),
"completeTime": time.Now().Format(time.RFC3339),
"result": map[string]interface{}{
"code":    0,
"message": "Success",
},
}

// 返回响应
sendJSONResponse(w, true, status, "", 0)
}

func (h *DeviceHandler) getEvents(w http.ResponseWriter, r *http.Request) {
// 从URL中提取设备ID
vars := r.Context().Value("vars").(map[string]string)
deviceID := vars["deviceId"]

// 获取查询参数
from := r.URL.Query().Get("from")
to := r.URL.Query().Get("to")
eventType := r.URL.Query().Get("type")

h.logger.Debug("Getting events", "deviceId", deviceID, "from", from, "to", to, "type", eventType)

// 模拟事件数据
events := []map[string]interface{}{
{
"id":        "evt123",
"type":      "ValueChange",
"timestamp": time.Now().Add(-30 * time.Minute).Format(time.RFC3339),
"parameters": []map[string]interface{}{
{
"name":  "Device.DeviceInfo.UpTime",
"value": "86400",
"type":  "unsignedInt",
},
},
},
{
"id":        "evt124",
"type":      "Boot",
"timestamp": time.Now().Add(-90 * time.Minute).Format(time.RFC3339),
"cause":     "PowerOn",
},
}

// 返回响应
sendJSONResponse(w, true, map[string]interface{}{"events": events}, "", 0)
}

// 辅助函数：发送JSON响应
func sendJSONResponse(w http.ResponseWriter, success bool, data interface{}, errorMsg string, errorCode int) {
response := map[string]interface{}{
"success": success,
}

if data != nil {
response["data"] = data
}

if errorMsg != "" {
response["error"] = errorMsg
}

if errorCode != 0 {
response["code"] = errorCode
}

w.Header().Set("Content-Type", "application/json")
if !success {
w.WriteHeader(http.StatusBadRequest)
}

json.NewEncoder(w).Encode(response)
}

func main() {
// 创建日志记录器
loggerImpl := logger.NewDefaultLogger(
logger.WithLevel("debug"),
logger.WithOutput(os.Stdout),
)

// 创建解析器和构建器
parserImpl := parser.NewParser()
builderImpl := builder.NewBuilder()

// 创建转换器
converter := api.NewTR069Converter(parserImpl, builderImpl)

// 创建API服务器
server := api.NewServer(
func(config *interfaces.APIConfig) {
config.Port = 8080
config.BasePath = "/api/v1"
config.EnableCORS = true
config.EnableAuth = false
},
)

// 创建并注册设备处理器
deviceHandler := NewDeviceHandler(converter, loggerImpl)
server.RegisterHandler(deviceHandler)

// 添加日志中间件
server.Use(api.LoggingMiddleware(loggerImpl))

// 启动服务器
go func() {
loggerImpl.Info("Starting API server", "port", 8080)
if err := server.Start(); err != nil && err != http.ErrServerClosed {
loggerImpl.Error("Server error", "error", err)
}
}()

// 等待中断信号
stop := make(chan os.Signal, 1)
signal.Notify(stop, os.Interrupt)
<-stop

// 优雅关闭
loggerImpl.Info("Shutting down server...")
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()
if err := server.Stop(ctx); err != nil {
loggerImpl.Error("Server shutdown error", "error", err)
}
loggerImpl.Info("Server stopped")
}
