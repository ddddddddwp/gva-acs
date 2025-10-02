package core

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-adapter/interfaces"
	"go.uber.org/zap"
)

// CWMPHandler CWMP协议处理器
type CWMPHandler struct {
	logger           *zap.Logger
	sessionManager   interfaces.TR069SessionManager
	deviceManager    interfaces.TR069DeviceManager
	parameterManager interfaces.TR069ParameterManager
	tr069Logger      interfaces.TR069Logger
	builder          interfaces.TR069Builder
	
	// 内部状态
	sessions map[string]*interfaces.Session
	devices  map[string]*interfaces.DeviceInfo
	mutex    sync.RWMutex
}

// NewCWMPHandler 创建CWMP处理器
func NewCWMPHandler(
	logger *zap.Logger,
	sessionManager interfaces.TR069SessionManager,
	deviceManager interfaces.TR069DeviceManager,
	parameterManager interfaces.TR069ParameterManager,
	tr069Logger interfaces.TR069Logger,
	builder interfaces.TR069Builder,
) *CWMPHandler {
	return &CWMPHandler{
		logger:           logger,
		sessionManager:   sessionManager,
		deviceManager:    deviceManager,
		parameterManager: parameterManager,
		tr069Logger:      tr069Logger,
		builder:          builder,
		sessions:         make(map[string]*interfaces.Session),
		devices:          make(map[string]*interfaces.DeviceInfo),
	}
}

// HandleInform 处理Inform事件
func (h *CWMPHandler) HandleInform(ctx context.Context, session *interfaces.Session, inform *interfaces.InformMessage) (*interfaces.InformResponse, error) {
	h.logger.Info("Handling inform message",
		zap.String("session_id", session.SessionID),
		zap.String("device_id", session.DeviceID),
		zap.Any("events", inform.Event))
	
	// 记录操作日志
	operationLog := &interfaces.OperationLog{
		DeviceID:    session.DeviceID,
		SessionID:   session.SessionID,
		Operation:   "Inform",
		Parameters:  fmt.Sprintf("Events: %v, MaxEnvelopes: %d", inform.Event, inform.MaxEnvelopes),
		Result:      "Processing",
		Timestamp:   time.Now(),
	}
	
	// 更新设备信息
	err := h.updateDeviceFromInform(ctx, session.DeviceID, inform)
	if err != nil {
		h.logger.Error("Failed to update device from inform", zap.Error(err))
		operationLog.Result = "Failed"
		operationLog.ErrorMessage = err.Error()
		h.tr069Logger.LogOperation(ctx, operationLog)
		return nil, fmt.Errorf("failed to update device: %w", err)
	}
	
	// 更新设备参数
	err = h.updateDeviceParameters(ctx, session.DeviceID, inform.ParameterList)
	if err != nil {
		h.logger.Error("Failed to update device parameters", zap.Error(err))
		operationLog.Result = "Failed"
		operationLog.ErrorMessage = err.Error()
		h.tr069Logger.LogOperation(ctx, operationLog)
		return nil, fmt.Errorf("failed to update parameters: %w", err)
	}
	
	// 处理事件
	err = h.processInformEvents(ctx, session, inform.Event)
	if err != nil {
		h.logger.Error("Failed to process inform events", zap.Error(err))
		operationLog.Result = "Failed"
		operationLog.ErrorMessage = err.Error()
		h.tr069Logger.LogOperation(ctx, operationLog)
		return nil, fmt.Errorf("failed to process events: %w", err)
	}
	
	// 更新会话信息
	session.MaxEnvelopes = inform.MaxEnvelopes
	session.CurrentEnvelope = 1
	session.LastActivity = time.Now()
	
	err = h.sessionManager.UpdateSession(ctx, session)
	if err != nil {
		h.logger.Warn("Failed to update session", zap.Error(err))
	}
	
	// 构建响应
	response := &interfaces.InformResponse{
		MaxEnvelopes: inform.MaxEnvelopes,
	}
	
	operationLog.Result = "Success"
	h.tr069Logger.LogOperation(ctx, operationLog)
	
	h.logger.Info("Inform processed successfully",
		zap.String("session_id", session.SessionID),
		zap.String("device_id", session.DeviceID))
	
	return response, nil
}

// HandleGetParameterValues 处理获取参数值事件
func (h *CWMPHandler) HandleGetParameterValues(ctx context.Context, session *interfaces.Session, request *interfaces.GetParameterValuesRequest) (*interfaces.GetParameterValuesResponse, error) {
	h.logger.Info("Handling get parameter values",
		zap.String("session_id", session.SessionID),
		zap.String("device_id", session.DeviceID),
		zap.Strings("parameter_names", request.ParameterNames))
	
	// 记录操作日志
	operationLog := &interfaces.OperationLog{
		DeviceID:    session.DeviceID,
		SessionID:   session.SessionID,
		Operation:   "GetParameterValues",
		Parameters:  fmt.Sprintf("Parameters: %v", request.ParameterNames),
		Timestamp:   time.Now(),
	}
	
	// 获取参数值
	parameters, err := h.parameterManager.BatchGetParameters(ctx, session.DeviceID, request.ParameterNames)
	if err != nil {
		h.logger.Error("Failed to get parameter values", zap.Error(err))
		operationLog.Result = "Failed"
		operationLog.ErrorMessage = err.Error()
		h.tr069Logger.LogOperation(ctx, operationLog)
		return nil, fmt.Errorf("failed to get parameters: %w", err)
	}
	
	response := &interfaces.GetParameterValuesResponse{
		ParameterList: parameters,
	}
	
	operationLog.Result = "Success"
	operationLog.Parameters = fmt.Sprintf("Retrieved %d parameters", len(parameters))
	h.tr069Logger.LogOperation(ctx, operationLog)
	
	h.logger.Info("Get parameter values processed successfully",
		zap.String("session_id", session.SessionID),
		zap.Int("parameter_count", len(parameters)))
	
	return response, nil
}

// HandleSetParameterValues 处理设置参数值事件
func (h *CWMPHandler) HandleSetParameterValues(ctx context.Context, session *interfaces.Session, request *interfaces.SetParameterValuesRequest) (*interfaces.SetParameterValuesResponse, error) {
	h.logger.Info("Handling set parameter values",
		zap.String("session_id", session.SessionID),
		zap.String("device_id", session.DeviceID),
		zap.Int("parameter_count", len(request.ParameterList)))
	
	// 记录操作日志
	operationLog := &interfaces.OperationLog{
		DeviceID:    session.DeviceID,
		SessionID:   session.SessionID,
		Operation:   "SetParameterValues",
		Parameters:  fmt.Sprintf("Parameters: %d items", len(request.ParameterList)),
		Timestamp:   time.Now(),
	}
	
	// 设置参数值
	err := h.parameterManager.BatchSetParameters(ctx, session.DeviceID, request.ParameterList)
	if err != nil {
		h.logger.Error("Failed to set parameter values", zap.Error(err))
		operationLog.Result = "Failed"
		operationLog.ErrorMessage = err.Error()
		h.tr069Logger.LogOperation(ctx, operationLog)
		return nil, fmt.Errorf("failed to set parameters: %w", err)
	}
	
	response := &interfaces.SetParameterValuesResponse{
		Status: 0, // 0 表示成功
	}
	
	operationLog.Result = "Success"
	h.tr069Logger.LogOperation(ctx, operationLog)
	
	h.logger.Info("Set parameter values processed successfully",
		zap.String("session_id", session.SessionID),
		zap.Int("parameter_count", len(request.ParameterList)))
	
	return response, nil
}

// HandleReboot 处理重启事件
func (h *CWMPHandler) HandleReboot(ctx context.Context, session *interfaces.Session, request *interfaces.RebootRequest) (*interfaces.RebootResponse, error) {
	h.logger.Info("Handling reboot request",
		zap.String("session_id", session.SessionID),
		zap.String("device_id", session.DeviceID),
		zap.String("command_key", request.CommandKey))
	
	// 记录操作日志
	operationLog := &interfaces.OperationLog{
		DeviceID:    session.DeviceID,
		SessionID:   session.SessionID,
		Operation:   "Reboot",
		Parameters:  fmt.Sprintf("CommandKey: %s", request.CommandKey),
		Timestamp:   time.Now(),
	}
	
	// 更新设备状态为维护中
	err := h.deviceManager.UpdateDeviceStatus(ctx, session.DeviceID, interfaces.DeviceStatusMaintenance)
	if err != nil {
		h.logger.Error("Failed to update device status", zap.Error(err))
		operationLog.Result = "Failed"
		operationLog.ErrorMessage = err.Error()
		h.tr069Logger.LogOperation(ctx, operationLog)
		return nil, fmt.Errorf("failed to update device status: %w", err)
	}
	
	response := &interfaces.RebootResponse{
		Status: 0, // 0 表示成功
	}
	
	operationLog.Result = "Success"
	h.tr069Logger.LogOperation(ctx, operationLog)
	
	h.logger.Info("Reboot request processed successfully",
		zap.String("session_id", session.SessionID),
		zap.String("command_key", request.CommandKey))
	
	return response, nil
}

// HandleFactoryReset 处理恢复出厂设置事件
func (h *CWMPHandler) HandleFactoryReset(ctx context.Context, session *interfaces.Session, request *interfaces.FactoryResetRequest) (*interfaces.FactoryResetResponse, error) {
	h.logger.Info("Handling factory reset request",
		zap.String("session_id", session.SessionID),
		zap.String("device_id", session.DeviceID),
		zap.String("command_key", request.CommandKey))
	
	// 记录操作日志
	operationLog := &interfaces.OperationLog{
		DeviceID:    session.DeviceID,
		SessionID:   session.SessionID,
		Operation:   "FactoryReset",
		Parameters:  fmt.Sprintf("CommandKey: %s", request.CommandKey),
		Timestamp:   time.Now(),
	}
	
	// 更新设备状态为维护中
	err := h.deviceManager.UpdateDeviceStatus(ctx, session.DeviceID, interfaces.DeviceStatusMaintenance)
	if err != nil {
		h.logger.Error("Failed to update device status", zap.Error(err))
		operationLog.Result = "Failed"
		operationLog.ErrorMessage = err.Error()
		h.tr069Logger.LogOperation(ctx, operationLog)
		return nil, fmt.Errorf("failed to update device status: %w", err)
	}
	
	response := &interfaces.FactoryResetResponse{
		Status: 0, // 0 表示成功
	}
	
	operationLog.Result = "Success"
	h.tr069Logger.LogOperation(ctx, operationLog)
	
	h.logger.Info("Factory reset request processed successfully",
		zap.String("session_id", session.SessionID),
		zap.String("command_key", request.CommandKey))
	
	return response, nil
}

// HandleDownload 处理下载事件
func (h *CWMPHandler) HandleDownload(ctx context.Context, session *interfaces.Session, request *interfaces.DownloadRequest) (*interfaces.DownloadResponse, error) {
	h.logger.Info("Handling download request",
		zap.String("session_id", session.SessionID),
		zap.String("device_id", session.DeviceID),
		zap.String("command_key", request.CommandKey),
		zap.String("file_type", request.FileType),
		zap.String("url", request.URL))
	
	// 记录操作日志
	operationLog := &interfaces.OperationLog{
		DeviceID:    session.DeviceID,
		SessionID:   session.SessionID,
		Operation:   "Download",
		Parameters:  fmt.Sprintf("CommandKey: %s, FileType: %s, URL: %s", request.CommandKey, request.FileType, request.URL),
		Timestamp:   time.Now(),
	}
	
	// 验证下载请求
	if request.URL == "" {
		err := fmt.Errorf("download URL cannot be empty")
		operationLog.Result = "Failed"
		operationLog.ErrorMessage = err.Error()
		h.tr069Logger.LogOperation(ctx, operationLog)
		return nil, err
	}
	
	response := &interfaces.DownloadResponse{
		Status:    0, // 0 表示成功
		StartTime: time.Now(),
	}
	
	operationLog.Result = "Success"
	h.tr069Logger.LogOperation(ctx, operationLog)
	
	h.logger.Info("Download request processed successfully",
		zap.String("session_id", session.SessionID),
		zap.String("command_key", request.CommandKey))
	
	return response, nil
}

// HandleTransferComplete 处理传输完成事件
func (h *CWMPHandler) HandleTransferComplete(ctx context.Context, session *interfaces.Session, request *interfaces.TransferCompleteRequest) (*interfaces.TransferCompleteResponse, error) {
	h.logger.Info("Handling transfer complete",
		zap.String("session_id", session.SessionID),
		zap.String("device_id", session.DeviceID),
		zap.String("command_key", request.CommandKey),
		zap.Int("fault_code", request.FaultCode))
	
	// 记录操作日志
	operationLog := &interfaces.OperationLog{
		DeviceID:    session.DeviceID,
		SessionID:   session.SessionID,
		Operation:   "TransferComplete",
		Parameters:  fmt.Sprintf("CommandKey: %s, FaultCode: %d", request.CommandKey, request.FaultCode),
		Timestamp:   time.Now(),
	}
	
	// 检查传输结果
	if request.FaultCode != 0 {
		h.logger.Warn("Transfer completed with fault",
			zap.String("command_key", request.CommandKey),
			zap.Int("fault_code", request.FaultCode),
			zap.String("fault_string", request.FaultString))
		
		operationLog.ErrorCode = request.FaultCode
		operationLog.ErrorMessage = request.FaultString
		operationLog.Result = "Failed"
	} else {
		operationLog.Result = "Success"
	}
	
	response := &interfaces.TransferCompleteResponse{
		Status: 0, // 0 表示成功
	}
	
	h.tr069Logger.LogOperation(ctx, operationLog)
	
	h.logger.Info("Transfer complete processed successfully",
		zap.String("session_id", session.SessionID),
		zap.String("command_key", request.CommandKey))
	
	return response, nil
}

// updateDeviceFromInform 从Inform消息更新设备信息
func (h *CWMPHandler) updateDeviceFromInform(ctx context.Context, deviceID string, inform *interfaces.InformMessage) error {
	// 获取现有设备信息
	device, err := h.deviceManager.GetDevice(ctx, deviceID)
	if err != nil {
		// 如果设备不存在，创建新设备
		device = &interfaces.DeviceInfo{
			DeviceID:     deviceID,
			SerialNumber: inform.DeviceID.SerialNumber,
			Manufacturer: inform.DeviceID.Manufacturer,
			OUI:          inform.DeviceID.OUI,
			ProductClass: inform.DeviceID.ProductClass,
			Status:       interfaces.DeviceStatusOnline,
			LastInform:   time.Now(),
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}
		
		return h.deviceManager.RegisterDevice(ctx, device)
	}
	
	// 更新设备状态和最后Inform时间
	device.Status = interfaces.DeviceStatusOnline
	device.LastInform = time.Now()
	device.UpdatedAt = time.Now()
	
	return h.deviceManager.UpdateDeviceStatus(ctx, deviceID, interfaces.DeviceStatusOnline)
}

// updateDeviceParameters 更新设备参数
func (h *CWMPHandler) updateDeviceParameters(ctx context.Context, deviceID string, parameters []interfaces.Parameter) error {
	if len(parameters) == 0 {
		return nil
	}
	
	// 批量设置参数
	return h.parameterManager.BatchSetParameters(ctx, deviceID, parameters)
}

// processInformEvents 处理Inform事件
func (h *CWMPHandler) processInformEvents(ctx context.Context, session *interfaces.Session, events []interfaces.EventStruct) error {
	for _, event := range events {
		h.logger.Debug("Processing inform event",
			zap.String("event_code", event.EventCode),
			zap.String("command_key", event.CommandKey))
		
		switch event.EventCode {
		case "0 BOOTSTRAP":
			h.logger.Info("Device bootstrap event", zap.String("device_id", session.DeviceID))
		case "1 BOOT":
			h.logger.Info("Device boot event", zap.String("device_id", session.DeviceID))
		case "2 PERIODIC":
			h.logger.Debug("Device periodic inform", zap.String("device_id", session.DeviceID))
		case "4 VALUE CHANGE":
			h.logger.Info("Device value change event", zap.String("device_id", session.DeviceID))
		case "6 CONNECTION REQUEST":
			h.logger.Info("Device connection request event", zap.String("device_id", session.DeviceID))
		case "7 TRANSFER COMPLETE":
			h.logger.Info("Device transfer complete event", zap.String("device_id", session.DeviceID))
		case "8 DIAGNOSTICS COMPLETE":
			h.logger.Info("Device diagnostics complete event", zap.String("device_id", session.DeviceID))
		default:
			h.logger.Warn("Unknown event code", 
				zap.String("event_code", event.EventCode),
				zap.String("device_id", session.DeviceID))
		}
	}
	
	return nil
}

// GetSupportedMethods 获取支持的RPC方法
func (h *CWMPHandler) GetSupportedMethods() []string {
	return []string{
		"GetParameterValues",
		"SetParameterValues",
		"GetParameterNames",
		"SetParameterAttributes",
		"GetParameterAttributes",
		"AddObject",
		"DeleteObject",
		"Reboot",
		"FactoryReset",
		"Download",
		"Upload",
		"GetRPCMethods",
	}
}

// ValidateSession 验证会话
func (h *CWMPHandler) ValidateSession(ctx context.Context, session *interfaces.Session) error {
	if session == nil {
		return fmt.Errorf("session is nil")
	}
	
	if session.SessionID == "" {
		return fmt.Errorf("session ID is empty")
	}
	
	if session.DeviceID == "" {
		return fmt.Errorf("device ID is empty")
	}
	
	if session.Status != interfaces.SessionStatusActive {
		return fmt.Errorf("session is not active")
	}
	
	// 检查会话是否过期
	if time.Since(session.LastActivity) > 30*time.Minute {
		return fmt.Errorf("session has expired")
	}
	
	return nil
}