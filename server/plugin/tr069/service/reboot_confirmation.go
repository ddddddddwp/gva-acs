package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type RebootConfirmationService struct {
	db *gorm.DB
}

func NewRebootConfirmationService(db *gorm.DB) *RebootConfirmationService {
	return &RebootConfirmationService{db: db}
}

func hasRebootConfirmationEvent(events []string) bool {
	for _, event := range events {
		switch strings.ToUpper(strings.TrimSpace(event)) {
		case "M REBOOT", "1 BOOT":
			return true
		}
	}
	return false
}

func (s *RebootConfirmationService) ConfirmFromInform(ctx context.Context, deviceID uint, events []string, confirmedAt time.Time) error {
	if s == nil || s.db == nil || deviceID == 0 || !hasRebootConfirmationEvent(events) {
		return nil
	}
	if confirmedAt.IsZero() {
		confirmedAt = time.Now()
	}
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var command model.Command
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("device_id = ? AND operation = ? AND status = ?", deviceID, "Reboot", model.CommandStatusWaitingReboot).
			Order("created_at ASC").Order("command_id ASC").First(&command).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		_, err = NewCommandStore(tx).Transition(ctx, CommandTransition{
			CommandID: command.CommandID, FromStatuses: []string{model.CommandStatusWaitingReboot},
			ToStatus: model.CommandStatusCompleted, ExpectedVersion: command.Version,
			EventType: "REBOOT_CONFIRMED", Stage: "reboot.inform",
			Updates: map[string]any{"finished_at": confirmedAt, "phase_deadline_at": nil},
		})
		return err
	})
	if errors.Is(err, ErrCommandTransitionConflict) {
		return nil
	}
	return err
}
