package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
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

type CommandPayloadProtector interface {
	Protect(ctx context.Context, tx *gorm.DB, deviceID uint, operation, origin string, encoded []byte) ([]byte, error)
}

type CommandCreatedHook func(ctx context.Context, tx *gorm.DB, command *model.Command) error

var ErrCommandNotRetryable = errors.New("command is not retryable")

const (
	commandWakeupCompensationTimeout  = 5 * time.Second
	commandWakeupCompensationAttempts = 3
)

type CommandManager struct {
	db          *gorm.DB
	wakeup      CommandWakeupFunc
	now         func() time.Time
	protector   CommandPayloadProtector
	createdHook CommandCreatedHook
}

func WithCommandPayloadProtector(protector CommandPayloadProtector) CommandManagerOption {
	return func(manager *CommandManager) {
		manager.protector = protector
	}
}

func WithCommandCreatedHook(hook CommandCreatedHook) CommandManagerOption {
	return func(manager *CommandManager) {
		manager.createdHook = hook
	}
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
	return m.submitPersisted(ctx, deviceID, operation, paramsJSON, commandSubmission{
		origin: model.CommandOriginUser,
	})
}

func (m *CommandManager) SubmitSystem(ctx context.Context, deviceID uint, operation string, request any, dedupKey string, hook CommandCreatedHook) (SubmitResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if m == nil {
		return SubmitResult{}, errors.New("command manager is required")
	}
	if m.database() == nil {
		return SubmitResult{}, errors.New("db not initialized")
	}
	paramsJSON, err := EncodeRPCRequest(operation, request)
	if err != nil {
		return SubmitResult{}, err
	}
	return m.submitPersisted(ctx, deviceID, operation, paramsJSON, commandSubmission{
		origin: model.CommandOriginSystem, dedupKey: dedupKey, system: true, hook: hook,
	})
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
	return m.submitPersisted(ctx, original.DeviceID, original.Operation, paramsJSON, commandSubmission{
		origin: model.CommandOriginUser, retryOf: original.CommandID,
	})
}

type commandSubmission struct {
	origin   string
	dedupKey string
	retryOf  string
	system   bool
	hook     CommandCreatedHook
}

func (m *CommandManager) submitPersisted(ctx context.Context, deviceID uint, operation string, paramsJSON []byte, submission commandSubmission) (SubmitResult, error) {
	db := m.database()
	if db == nil {
		return SubmitResult{}, errors.New("db not initialized")
	}
	now := m.now()
	commandID := uuid.NewString()
	command := model.Command{
		CommandID: commandID,
		DeviceID:  deviceID,
		Operation: operation,
		Origin:    submission.origin,
		DedupKey:  submission.dedupKey,
		RetryOf:   submission.retryOf,
		QueuedAt:  now,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if spec, ok := RPCSpecs[operation]; ok && spec.ServerCommandKey {
		commandKey, err := commandKeyFromCommandID(commandID)
		if err != nil {
			return SubmitResult{}, err
		}
		command.CommandKey = &commandKey
	}

	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var device model.Device
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Select("id", "oui", "serial_number", "last_inform").
			First(&device, deviceID).Error; err != nil {
			return err
		}
		if submission.system {
			if device.LastInform.IsZero() || now.Sub(device.LastInform) >= commandOnlineThreshold {
				return ErrDeviceOffline
			}
		} else {
			if err := ValidateRPCSubmission(ctx, tx, deviceID, operation, now); err != nil {
				return err
			}
		}
		if device.OUI == "" || device.SerialNumber == "" {
			return fmt.Errorf("device missing oui/serialNumber: %d", deviceID)
		}
		command.DeviceKey = fmt.Sprintf("%s-%s", device.OUI, device.SerialNumber)
		protected := append([]byte(nil), paramsJSON...)
		if m.protector != nil {
			var err error
			protected, err = m.protector.Protect(ctx, tx, deviceID, operation, command.Origin, protected)
			if err != nil {
				return err
			}
		}
		command.ParamsJSON = model.LongTextJSON(protected)

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
		if err := NewCommandStore(tx).Create(ctx, &command); err != nil {
			return err
		}
		if m.createdHook != nil {
			if err := m.createdHook(ctx, tx, &command); err != nil {
				return err
			}
		}
		if submission.hook != nil {
			return submission.hook(ctx, tx, &command)
		}
		return nil
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

func commandKeyFromCommandID(commandID string) (string, error) {
	parsed, err := uuid.Parse(commandID)
	if err != nil {
		return "", fmt.Errorf("invalid command ID for CommandKey: %w", err)
	}
	return strings.ReplaceAll(parsed.String(), "-", ""), nil
}

func (m *CommandManager) failWakeup(ctx context.Context, command model.Command, result SubmitResult, wakeupErr error) (SubmitResult, error) {
	if wakeupErr == nil {
		wakeupErr = ErrCommandQueueUnavailable
	}
	cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), commandWakeupCompensationTimeout)
	defer cancel()

	var terminal model.Command
	var compensationErr error
	for attempt := 0; attempt < commandWakeupCompensationAttempts; attempt++ {
		compensationErr = m.database().WithContext(cleanupCtx).Transaction(func(tx *gorm.DB) error {
			var current model.Command
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
				First(&current, "command_id = ?", command.CommandID).Error; err != nil {
				return err
			}
			if current.Status != model.CommandStatusWaitingDevice && current.Status != model.CommandStatusBuilding {
				terminal = current
				return nil
			}

			finishedAt := m.now()
			failed, err := NewCommandStore(tx).Transition(cleanupCtx, CommandTransition{
				CommandID:       current.CommandID,
				FromStatuses:    []string{current.Status},
				ToStatus:        model.CommandStatusFailed,
				ExpectedVersion: current.Version,
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
			if err == nil {
				terminal = failed
			}
			return err
		})
		if compensationErr == nil {
			result.Status = terminal.Status
			return result, nil
		}
		if !errors.Is(compensationErr, ErrCommandTransitionConflict) {
			break
		}
	}

	return result, errors.Join(
		ErrCommandQueueUnavailable,
		fmt.Errorf("redis.enqueue: %w", wakeupErr),
		fmt.Errorf("persist wakeup compensation: %w", compensationErr),
	)
}
