package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ddddddddwp/gva-acs/server/global"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/config"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const queueAdvanceWakeTimeout = 5 * time.Second

var activeCommandStatuses = []string{
	model.CommandStatusWaitingDevice,
	model.CommandStatusBuilding,
	model.CommandStatusSent,
	model.CommandStatusWaitingTransfer,
	model.CommandStatusWaitingReboot,
}

type CommandQueueAdvanceResult struct {
	CommandID string
	DeviceKey string
	Promoted  bool
	Woken     bool
}

type CommandQueueAdvancer struct {
	db     *gorm.DB
	wakeup CommandWakeupFunc
	now    func() time.Time
}

type CommandQueueAdvancerOption func(*CommandQueueAdvancer)

func WithCommandQueueAdvancerNow(now func() time.Time) CommandQueueAdvancerOption {
	return func(advancer *CommandQueueAdvancer) {
		if now != nil {
			advancer.now = now
		}
	}
}

func NewCommandQueueAdvancer(db *gorm.DB, wakeup CommandWakeupFunc, options ...CommandQueueAdvancerOption) *CommandQueueAdvancer {
	advancer := &CommandQueueAdvancer{db: db, wakeup: wakeup, now: time.Now}
	for _, option := range options {
		option(advancer)
	}
	return advancer
}

// AdvanceAfterTerminal is intentionally best-effort for the caller: the
// terminal state has already committed, so a follow-up dispatch failure must
// not make the completed CWMP response look failed. Durable recovery retries it.
func (a *CommandQueueAdvancer) AdvanceAfterTerminal(ctx context.Context, deviceID uint) {
	if a == nil || deviceID == 0 {
		return
	}
	if _, err := a.AdvanceDevice(ctx, deviceID); err != nil && global.GVA_LOG != nil {
		global.GVA_LOG.Error("failed to advance TR-069 command FIFO",
			zap.Uint("deviceId", deviceID), zap.Error(err))
	}
}

// AdvanceDevice promotes exactly one queued command when the device has no
// active command. The durable state is committed before the Redis wake token.
func (a *CommandQueueAdvancer) AdvanceDevice(ctx context.Context, deviceID uint) (CommandQueueAdvanceResult, error) {
	if a == nil || a.db == nil {
		return CommandQueueAdvanceResult{}, errors.New("command queue advancer database is required")
	}
	if deviceID == 0 {
		return CommandQueueAdvanceResult{}, errors.New("command queue advancer device ID is required")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	var result CommandQueueAdvanceResult
	err := a.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockCommandQueueDevice(tx, deviceID); err != nil {
			return err
		}
		var active model.Command
		err := tx.Where("device_id = ? AND status IN ?", deviceID, activeCommandStatuses).
			Order("created_at ASC").Order("command_id ASC").First(&active).Error
		if err == nil {
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		var queued model.Command
		err = tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("device_id = ? AND status = ?", deviceID, model.CommandStatusQueued).
			Order("created_at ASC").Order("command_id ASC").First(&queued).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		if err != nil {
			return err
		}

		now := a.now()
		deadline := now.Add(config.CurrentRuntime().CommandQueueWaitTimeout)
		promoted, err := NewCommandStore(tx).Transition(ctx, CommandTransition{
			CommandID:       queued.CommandID,
			FromStatuses:    []string{model.CommandStatusQueued},
			ToStatus:        model.CommandStatusWaitingDevice,
			ExpectedVersion: queued.Version,
			EventType:       "QUEUE_ADVANCED",
			Stage:           "queue.advance",
			Updates: map[string]any{
				"waiting_at":        now,
				"phase_deadline_at": deadline,
			},
		})
		if err != nil {
			return err
		}
		result = CommandQueueAdvanceResult{
			CommandID: promoted.CommandID,
			DeviceKey: promoted.DeviceKey,
			Promoted:  true,
		}
		return nil
	})
	if err != nil || !result.Promoted {
		return result, err
	}
	return a.wake(ctx, result)
}

// Recover repairs devices whose durable queue has no active head and restores
// wake tokens for heads that were already waiting when the process stopped.
func (a *CommandQueueAdvancer) Recover(ctx context.Context) error {
	if a == nil || a.db == nil {
		return errors.New("command queue advancer database is required")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	var deviceIDs []uint
	if err := a.db.WithContext(ctx).Model(new(model.Command)).Distinct("device_id").
		Where("status IN ?", []string{model.CommandStatusQueued, model.CommandStatusWaitingDevice}).
		Order("device_id ASC").Pluck("device_id", &deviceIDs).Error; err != nil {
		return err
	}
	var recoveryErr error
	for _, deviceID := range deviceIDs {
		if err := a.recoverDevice(ctx, deviceID); err != nil {
			recoveryErr = errors.Join(recoveryErr, fmt.Errorf("recover device %d command queue: %w", deviceID, err))
		}
	}
	return recoveryErr
}

func (a *CommandQueueAdvancer) recoverDevice(ctx context.Context, deviceID uint) error {
	var waiting CommandQueueAdvanceResult
	err := a.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockCommandQueueDevice(tx, deviceID); err != nil {
			return err
		}
		var head model.Command
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("device_id = ? AND status IN ?", deviceID, model.NonTerminalCommandStatuses()).
			Order("created_at ASC").Order("command_id ASC").First(&head).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		if head.Status == model.CommandStatusWaitingDevice {
			waiting = CommandQueueAdvanceResult{CommandID: head.CommandID, DeviceKey: head.DeviceKey}
		}
		return nil
	})
	if err != nil {
		return err
	}
	if waiting.CommandID != "" {
		_, err = a.wake(ctx, waiting)
		return err
	}
	_, err = a.AdvanceDevice(ctx, deviceID)
	return err
}

func lockCommandQueueDevice(tx *gorm.DB, deviceID uint) error {
	var device model.Device
	return tx.Clauses(clause.Locking{Strength: "UPDATE"}).Select("id").First(&device, deviceID).Error
}

func (a *CommandQueueAdvancer) wake(ctx context.Context, result CommandQueueAdvanceResult) (CommandQueueAdvanceResult, error) {
	var wakeErr error
	if a.wakeup == nil {
		wakeErr = ErrCommandQueueUnavailable
	} else {
		wakeErr = a.wakeup(ctx, result.DeviceKey)
	}
	if wakeErr == nil {
		result.Woken = true
		return result, nil
	}

	eventCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), queueAdvanceWakeTimeout)
	defer cancel()
	eventErr := NewCommandStore(a.db).AppendEvent(eventCtx, &model.CommandEvent{
		CommandID:  result.CommandID,
		EventType:  "WAKE_ENQUEUE_FAILED",
		FromStatus: model.CommandStatusWaitingDevice,
		ToStatus:   model.CommandStatusWaitingDevice,
		Stage:      "redis.enqueue",
		Message:    wakeErr.Error(),
		CreatedAt:  a.now(),
	})
	if eventErr != nil {
		return result, errors.Join(wakeErr, fmt.Errorf("record wake enqueue failure: %w", eventErr))
	}
	return result, wakeErr
}
