package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ddddddddwp/gva-acs/server/global"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/model"
	req "github.com/ddddddddwp/gva-acs/server/plugin/tr069/model/request"
)

var (
	ErrDeviceOffline           = errors.New("device offline")
	ErrCommandQueueUnavailable = errors.New("command queue unavailable")
)

const commandOnlineThreshold = 180 * time.Second

// CommandService keeps the original API-facing methods while delegating all
// durable creation and FIFO decisions to CommandManager.
type CommandService struct {
	manager *CommandManager
}

func NewCommandService(manager *CommandManager) *CommandService {
	return &CommandService{manager: manager}
}

func (s *CommandService) commandManager() *CommandManager {
	if s != nil && s.manager != nil {
		return s.manager
	}
	return NewCommandManager(global.GVA_DB, nil)
}

func (s *CommandService) Submit(ctx context.Context, deviceID uint, operation string, request any) (SubmitResult, error) {
	return s.commandManager().Submit(ctx, deviceID, operation, request)
}

func (s *CommandService) Retry(ctx context.Context, commandID string) (SubmitResult, error) {
	return s.commandManager().Retry(ctx, commandID)
}

func (s *CommandService) EnqueueGetRPCMethods(deviceID uint) (string, error) {
	result, err := s.Submit(context.Background(), deviceID, "GetRPCMethods", nil)
	return result.CommandID, err
}

func (s *CommandService) EnqueueGetParameterValues(deviceID uint, in req.GetParameterValuesRequest) (string, error) {
	result, err := s.Submit(context.Background(), deviceID, "GetParameterValues", in)
	return result.CommandID, err
}

func (s *CommandService) EnqueueSetParameterValues(deviceID uint, in req.SetParameterValuesRequest) (string, error) {
	result, err := s.Submit(context.Background(), deviceID, "SetParameterValues", in)
	return result.CommandID, err
}

func (s *CommandService) EnqueueDeviceParameterSync(deviceID uint) (string, error) {
	result, err := s.Submit(context.Background(), deviceID, "GetParameterValues", req.GetParameterValuesRequest{Paths: []string{"Device."}})
	if err != nil {
		return result.CommandID, fmt.Errorf("enqueue GetParameterValues failed: %w", err)
	}
	return result.CommandID, nil
}

func deviceParameterSyncParams() map[string]interface{} {
	return map[string]interface{}{"paths": []string{"Device."}}
}

func validateCurrentRPCSubmission(deviceID uint, operation string, request any) error {
	if err := ValidateRPCRequest(operation, request); err != nil {
		return err
	}
	return ValidateRPCSubmission(context.Background(), global.GVA_DB, deviceID, operation, time.Now())
}

func (s *CommandService) commandTargetByID(deviceID uint, now time.Time) (string, error) {
	if global.GVA_DB == nil {
		return "", errors.New("db not initialized")
	}
	var device model.Device
	if err := global.GVA_DB.Select("id", "oui", "serial_number", "last_inform").First(&device, deviceID).Error; err != nil {
		return "", err
	}
	if device.OUI == "" || device.SerialNumber == "" {
		return "", fmt.Errorf("device missing oui/serialNumber: %d", deviceID)
	}
	if device.LastInform.IsZero() || now.Sub(device.LastInform) >= commandOnlineThreshold {
		return "", ErrDeviceOffline
	}
	return fmt.Sprintf("%s-%s", device.OUI, device.SerialNumber), nil
}
