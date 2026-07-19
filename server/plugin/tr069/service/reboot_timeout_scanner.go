package service

import (
	"context"
	"errors"
	"time"

	"github.com/ddddddddwp/gva-acs/server/global"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

const (
	rebootTimeoutScanInterval = 5 * time.Second
	rebootTimeoutScanLimit    = 100
)

type RebootTimeoutScanner struct {
	db       *gorm.DB
	advancer *CommandQueueAdvancer
}

func NewRebootTimeoutScanner(db *gorm.DB, advancers ...*CommandQueueAdvancer) *RebootTimeoutScanner {
	scanner := &RebootTimeoutScanner{db: db}
	if len(advancers) > 0 {
		scanner.advancer = advancers[0]
	}
	return scanner
}

func (s *RebootTimeoutScanner) Run(ctx context.Context) {
	if s == nil || s.db == nil {
		return
	}
	ticker := time.NewTicker(rebootTimeoutScanInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			if err := s.ScanOnce(ctx, now); err != nil && global.GVA_LOG != nil {
				global.GVA_LOG.Error("failed to scan expired Reboot confirmations", zap.Error(err))
			}
		}
	}
}

func (s *RebootTimeoutScanner) ScanOnce(ctx context.Context, now time.Time) error {
	if s == nil || s.db == nil {
		return gorm.ErrInvalidDB
	}
	var commands []model.Command
	if err := s.db.WithContext(ctx).
		Where("status = ? AND phase_deadline_at IS NOT NULL AND phase_deadline_at <= ?", model.CommandStatusWaitingReboot, now).
		Order("phase_deadline_at ASC").Order("command_id ASC").Limit(rebootTimeoutScanLimit).
		Find(&commands).Error; err != nil {
		return err
	}
	for _, command := range commands {
		_, err := NewCommandStore(s.db).Transition(ctx, CommandTransition{
			CommandID: command.CommandID, FromStatuses: []string{model.CommandStatusWaitingReboot},
			ToStatus: model.CommandStatusTimeout, ExpectedVersion: command.Version,
			EventType: "REBOOT_CONFIRM_TIMEOUT", Stage: "reboot.confirm",
			Message: "device did not report a reboot Inform before the deadline",
			Updates: map[string]any{
				"finished_at": now, "phase_deadline_at": nil,
				"failure_stage": "reboot.confirm", "fault_string": "等待设备重启确认超时",
			},
		})
		if err != nil && !errors.Is(err, ErrCommandTransitionConflict) {
			return err
		}
		if err == nil && s.advancer != nil {
			s.advancer.AdvanceAfterTerminal(ctx, command.DeviceID)
		}
	}
	return nil
}
