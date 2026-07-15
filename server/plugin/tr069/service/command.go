package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/ddddddddwp/gva-acs/server/global"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/adapter"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/config"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/infolog"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/model"
	req "github.com/ddddddddwp/gva-acs/server/plugin/tr069/model/request"
	"github.com/ddddddddwp/tr069-core-only/pkg/core"
	"github.com/google/uuid"
	"gorm.io/gorm/clause"
)

type CommandService struct{}

var (
	ErrDeviceOffline           = errors.New("device offline")
	ErrCommandQueueUnavailable = errors.New("command queue unavailable")
)

const commandOnlineThreshold = 180 * time.Second

func deviceParameterSyncParams() map[string]interface{} {
	return map[string]interface{}{"paths": []string{"Device."}}
}

func (s *CommandService) EnqueueGetRPCMethods(deviceID uint) (string, error) {
	deviceKey, err := s.commandTargetByID(deviceID, time.Now())
	if err != nil {
		return "", err
	}
	return s.enqueueImmediate(context.Background(), deviceID, deviceKey, "GetRPCMethods", map[string]interface{}{}, "")
}

func (s *CommandService) EnqueueGetParameterValues(deviceID uint, in req.GetParameterValuesRequest) (string, error) {
	if len(in.Paths) == 0 {
		return "", errors.New("paths is empty")
	}
	deviceKey, err := s.commandTargetByID(deviceID, time.Now())
	if err != nil {
		return "", err
	}
	return s.enqueueImmediate(context.Background(), deviceID, deviceKey, "GetParameterValues", map[string]interface{}{"paths": in.Paths}, "")
}

func (s *CommandService) EnqueueSetParameterValues(deviceID uint, in req.SetParameterValuesRequest) (string, error) {
	if len(in.Parameters) == 0 {
		return "", errors.New("parameters is empty")
	}
	deviceKey, err := s.commandTargetByID(deviceID, time.Now())
	if err != nil {
		return "", err
	}
	items := make([]map[string]interface{}, 0, len(in.Parameters))
	for _, p := range in.Parameters {
		if p.Name == "" {
			continue
		}
		items = append(items, map[string]interface{}{
			"name":  p.Name,
			"value": p.Value,
			"type":  p.Type,
		})
	}
	if len(items) == 0 {
		return "", errors.New("parameters is empty")
	}
	return s.enqueueImmediate(context.Background(), deviceID, deviceKey, "SetParameterValues", map[string]interface{}{
		"parameterKey": in.ParameterKey,
		"parameters":   items,
	}, "")
}

func (s *CommandService) EnqueueDeviceParameterSync(deviceID uint) (string, error) {
	deviceKey, err := s.commandTargetByID(deviceID, time.Now())
	if err != nil {
		return "", err
	}

	cmdID, err := s.enqueueImmediate(
		context.Background(),
		deviceID,
		deviceKey,
		"GetParameterValues",
		deviceParameterSyncParams(),
		fmt.Sprintf("dm:gpv:Device.:%s:%d", deviceKey, time.Now().UnixNano()),
	)
	if err != nil {
		return "", fmt.Errorf("enqueue GetParameterValues failed: %w", err)
	}
	return cmdID, nil
}

func (s *CommandService) enqueue(ctx context.Context, deviceKey string, op string, params map[string]interface{}, dedupKey string) (string, error) {
	if !adapter.RedisAvailable() {
		return "", ErrCommandQueueUnavailable
	}
	cmdID := uuid.NewString()
	now := time.Now()

	paramsJSON := ""
	if params != nil {
		b, err := json.Marshal(params)
		if err != nil {
			return "", err
		}
		paramsJSON = string(b)
	}

	if adapter.DBAvailable() {
		if err := global.GVA_DB.WithContext(ctx).Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "command_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"device_key", "operation", "params_json", "dedup_key", "status", "updated_at"}),
		}).Create(&model.Command{
			CommandID:  cmdID,
			DeviceKey:  deviceKey,
			Operation:  op,
			ParamsJSON: []byte(paramsJSON),
			DedupKey:   dedupKey,
			Status:     "PENDING",
			CreatedAt:  now,
			UpdatedAt:  now,
		}).Error; err != nil {
			return "", err
		}
	}

	ingest, err := adapter.NewRedisCommandIngest("")
	if err != nil {
		return "", err
	}
	if err := ingest.Enqueue(ctx, &core.Command{
		ID:        cmdID,
		DeviceKey: deviceKey,
		Operation: op,
		Params:    params,
		DedupKey:  dedupKey,
		CreatedAt: now,
	}); err != nil {
		return "", err
	}

	dump := fmt.Sprintf("----- TR069 COMMAND ENQUEUE BEGIN -----\ncommandId: %s\ndeviceKey: %s\noperation: %s\ndedupKey: %s\nparamsJson: %s\n----- TR069 COMMAND ENQUEUE END -----", cmdID, deviceKey, op, dedupKey, paramsJSON)
	if config.CurrentRuntime().Settings.DumpRaw {
		_, _ = fmt.Fprintln(os.Stdout, dump)
	}
	infolog.Write(dump)
	return cmdID, nil
}

func (s *CommandService) enqueueImmediate(ctx context.Context, deviceID uint, deviceKey string, op string, params map[string]interface{}, dedupKey string) (string, error) {
	if !adapter.RedisAvailable() {
		return "", ErrCommandQueueUnavailable
	}
	cmdID := uuid.NewString()
	now := time.Now()

	paramsJSON := ""
	if params != nil {
		b, err := json.Marshal(params)
		if err != nil {
			return "", err
		}
		paramsJSON = string(b)
	}

	if adapter.DBAvailable() {
		if err := global.GVA_DB.WithContext(ctx).Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "command_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"device_key", "operation", "params_json", "dedup_key", "status", "updated_at"}),
		}).Create(&model.Command{
			CommandID:  cmdID,
			DeviceKey:  deviceKey,
			Operation:  op,
			ParamsJSON: []byte(paramsJSON),
			DedupKey:   dedupKey,
			Status:     "PENDING",
			CreatedAt:  now,
			UpdatedAt:  now,
		}).Error; err != nil {
			return "", err
		}
	}

	if err := adapter.EnqueueImmediate(ctx, adapter.DispatcherPayload{
		DeviceKey: deviceKey,
		CommandID: cmdID,
		DedupKey:  dedupKey,
		Op:        op,
		Params:    paramsJSON,
		CreatedAt: now.Unix(),
	}, adapter.ImmediateEnqueueConfig{TTL: 30 * time.Minute}); err != nil {
		return "", err
	}

	dump := fmt.Sprintf("----- TR069 COMMAND IMMEDIATE BEGIN -----\ncommandId: %s\ndeviceId: %d\ndeviceKey: %s\noperation: %s\ndedupKey: %s\nparamsJson: %s\n----- TR069 COMMAND IMMEDIATE END -----", cmdID, deviceID, deviceKey, op, dedupKey, paramsJSON)
	_, _ = fmt.Fprintln(os.Stdout, dump)
	infolog.Write(dump)

	_ = adapter.TriggerConnectionRequest(ctx, deviceID, adapter.ConnectionRequestConfig{Timeout: 5 * time.Second, Retries: 1})

	return cmdID, nil
}

func (s *CommandService) deviceKeyByID(deviceID uint) (string, error) {
	if !adapter.DBAvailable() {
		return "", errors.New("db not initialized")
	}
	var d model.Device
	if err := global.GVA_DB.Select("id", "oui", "serial_number").First(&d, deviceID).Error; err != nil {
		return "", err
	}
	if d.OUI == "" || d.SerialNumber == "" {
		return "", fmt.Errorf("device missing oui/serialNumber: %d", deviceID)
	}
	return fmt.Sprintf("%s-%s", d.OUI, d.SerialNumber), nil
}

func (s *CommandService) commandTargetByID(deviceID uint, now time.Time) (string, error) {
	if !adapter.DBAvailable() {
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
