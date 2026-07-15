package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ddddddddwp/gva-acs/server/global"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/config"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type SubmitResult struct {
	CommandID string `json:"commandId"`
	Status    string `json:"status"`
}

type CommandWakeupFunc func(ctx context.Context, deviceKey string) error

var ErrCommandNotRetryable = errors.New("command is not retryable")

type CommandManager struct {
	db     *gorm.DB
	wakeup CommandWakeupFunc
	now    func() time.Time
}

type CommandManagerOption func(*CommandManager)

func WithCommandManagerNow(now func() time.Time) CommandManagerOption {
	return func(manager *CommandManager) {
		if now != nil {
			manager.now = now
		}
	}
}

func NewCommandManager(db *gorm.DB, wakeup CommandWakeupFunc, options ...CommandManagerOption) *CommandManager {
	manager := &CommandManager{db: db, wakeup: wakeup, now: time.Now}
	for _, option := range options {
		option(manager)
	}
	return manager
}

func (m *CommandManager) database() *gorm.DB {
	if m != nil && m.db != nil {
		return m.db
	}
	return global.GVA_DB
}

func (m *CommandManager) Submit(ctx context.Context, deviceID uint, operation string, request any) (SubmitResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if m == nil {
		return SubmitResult{}, errors.New("command manager is required")
	}
	db := m.database()
	if db == nil {
		return SubmitResult{}, errors.New("db not initialized")
	}
	paramsJSON, err := EncodeRPCRequest(operation, request)
	if err != nil {
		return SubmitResult{}, err
	}
	return m.submitPersisted(ctx, deviceID, operation, paramsJSON, "")
}

func (m *CommandManager) Retry(ctx context.Context, commandID string) (SubmitResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if m == nil {
		return SubmitResult{}, errors.New("command manager is required")
	}
	db := m.database()
	if db == nil {
		return SubmitResult{}, errors.New("db not initialized")
	}
	var original model.Command
	if err := db.WithContext(ctx).First(&original, "command_id = ?", commandID).Error; err != nil {
		return SubmitResult{}, err
	}
	if original.Status != model.CommandStatusFailed && original.Status != model.CommandStatusTimeout {
		return SubmitResult{}, fmt.Errorf("%w: status %s", ErrCommandNotRetryable, original.Status)
	}
	if _, err := DecodeRPCRequest(original.Operation, original.ParamsJSON); err != nil {
		return SubmitResult{}, err
	}
	paramsJSON := append([]byte(nil), original.ParamsJSON...)
	return m.submitPersisted(ctx, original.DeviceID, original.Operation, paramsJSON, original.CommandID)
}

func (m *CommandManager) submitPersisted(ctx context.Context, deviceID uint, operation string, paramsJSON []byte, retryOf string) (SubmitResult, error) {
	db := m.database()
	if db == nil {
		return SubmitResult{}, errors.New("db not initialized")
	}
	now := m.now()
	command := model.Command{
		CommandID:  uuid.NewString(),
		DeviceID:   deviceID,
		Operation:  operation,
		ParamsJSON: model.LongTextJSON(paramsJSON),
		RetryOf:    retryOf,
		QueuedAt:   now,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if spec, ok := RPCSpecs[operation]; ok && spec.Transfer {
		commandKey := "rpc-" + uuid.NewString()
		command.CommandKey = &commandKey
	}

	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var device model.Device
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Select("id", "oui", "serial_number", "last_inform").
			First(&device, deviceID).Error; err != nil {
			return err
		}
		if err := ValidateRPCSubmission(ctx, tx, deviceID, operation, now); err != nil {
			return err
		}
		if device.OUI == "" || device.SerialNumber == "" {
			return fmt.Errorf("device missing oui/serialNumber: %d", deviceID)
		}
		command.DeviceKey = fmt.Sprintf("%s-%s", device.OUI, device.SerialNumber)

		var head model.Command
		headErr := tx.Where("device_id = ? AND status IN ?", deviceID, model.NonTerminalCommandStatuses()).
			Order("created_at ASC").
			Order("command_id ASC").
			First(&head).Error
		switch {
		case errors.Is(headErr, gorm.ErrRecordNotFound):
			command.Status = model.CommandStatusWaitingDevice
			command.WaitingAt = &now
			deadline := now.Add(config.CurrentRuntime().CommandQueueWaitTimeout)
			command.PhaseDeadlineAt = &deadline
		case headErr != nil:
			return headErr
		default:
			command.Status = model.CommandStatusQueued
		}
		return NewCommandStore(tx).Create(ctx, &command)
	})
	if err != nil {
		return SubmitResult{}, err
	}

	result := SubmitResult{CommandID: command.CommandID, Status: command.Status}
	if command.Status != model.CommandStatusWaitingDevice {
		return result, nil
	}
	if m.wakeup == nil {
		return m.failWakeup(ctx, command, result, ErrCommandQueueUnavailable)
	}
	if err := m.wakeup(ctx, command.DeviceKey); err != nil {
		return m.failWakeup(ctx, command, result, err)
	}
	return result, nil
}

func (m *CommandManager) failWakeup(ctx context.Context, command model.Command, result SubmitResult, wakeupErr error) (SubmitResult, error) {
	finishedAt := m.now()
	failed, transitionErr := NewCommandStore(m.database()).Transition(ctx, CommandTransition{
		CommandID:       command.CommandID,
		FromStatuses:    []string{model.CommandStatusWaitingDevice},
		ToStatus:        model.CommandStatusFailed,
		ExpectedVersion: command.Version,
		EventType:       "DISPATCH_FAILED",
		Stage:           "redis.enqueue",
		Message:         wakeupErr.Error(),
		Updates: map[string]any{
			"failure_stage":     "redis.enqueue",
			"fault_string":      wakeupErr.Error(),
			"finished_at":       finishedAt,
			"phase_deadline_at": nil,
		},
	})
	if transitionErr == nil {
		result.Status = failed.Status
	}
	return result, errors.Join(ErrCommandQueueUnavailable, fmt.Errorf("redis.enqueue: %w", wakeupErr), transitionErr)
}
