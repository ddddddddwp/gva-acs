package service

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/config"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type TransferLifecycle struct {
	db       *gorm.DB
	advancer *CommandQueueAdvancer
}

func NewTransferLifecycle(db *gorm.DB, advancers ...*CommandQueueAdvancer) *TransferLifecycle {
	lifecycle := &TransferLifecycle{db: db}
	if len(advancers) > 0 {
		lifecycle.advancer = advancers[0]
	}
	return lifecycle
}

func (l *TransferLifecycle) OnUploadResponse(ctx context.Context, commandID string, status int, at time.Time) error {
	return l.mutate(ctx, "command_id = ?", commandID, "UPLOAD_RESPONSE", "rpc.response", 0, at,
		func(task *model.TransferTask, updates map[string]any) bool {
			if task.UploadResponseStatus != nil && *task.UploadResponseStatus == status {
				return false
			}
			task.UploadResponseStatus = &status
			updates["upload_response_status"] = status
			if status != 0 && status != 1 {
				task.FailureStage = "rpc.response"
				task.FailureCode = "INVALID_UPLOAD_STATUS"
				task.FailureMessage = "UploadResponse returned an unsupported status"
				updates["failure_stage"] = task.FailureStage
				updates["failure_code"] = task.FailureCode
				updates["failure_message"] = task.FailureMessage
			}
			return true
		})
}

func (l *TransferLifecycle) OnTransferComplete(ctx context.Context, commandKey string, faultCode int, faultString string, at time.Time) error {
	return l.mutate(ctx, "command_key = ?", commandKey, "TRANSFER_COMPLETE", "transfer.complete", 0, at,
		func(task *model.TransferTask, updates map[string]any) bool {
			failureCode := ""
			if faultCode != 0 {
				failureCode = strconv.Itoa(faultCode)
			}
			if task.TransferCompletedAt != nil && task.FailureCode == failureCode && task.FailureMessage == faultString {
				return false
			}
			task.TransferCompletedAt = &at
			updates["transfer_completed_at"] = at
			if faultCode != 0 {
				task.FailureStage = "transfer.complete"
				task.FailureCode = failureCode
				task.FailureMessage = faultString
				updates["failure_stage"] = task.FailureStage
				updates["failure_code"] = task.FailureCode
				updates["failure_message"] = task.FailureMessage
			}
			return true
		})
}

func (l *TransferLifecycle) OnArtifactAvailable(ctx context.Context, taskID string, fileID uint64, at time.Time) error {
	if l == nil || l.db == nil {
		return errors.New("transfer lifecycle database is required")
	}
	var artifact model.Artifact
	if err := l.db.WithContext(ctx).Where("id = ? AND task_id = ? AND status = ?", fileID, taskID, model.ArtifactStatusAvailable).First(&artifact).Error; err != nil {
		return err
	}
	return l.mutate(ctx, "task_id = ?", taskID, "ARTIFACT_AVAILABLE", "storage.commit", fileID, at,
		func(task *model.TransferTask, updates map[string]any) bool {
			if task.FileReceivedAt != nil {
				return false
			}
			task.FileReceivedAt = &at
			updates["file_received_at"] = at
			return true
		})
}

type transferFactMutation func(*model.TransferTask, map[string]any) bool

func (l *TransferLifecycle) mutate(ctx context.Context, where string, value any, eventCode, phase string, fileID uint64, at time.Time, apply transferFactMutation) error {
	if l == nil || l.db == nil {
		return errors.New("transfer lifecycle database is required")
	}
	if at.IsZero() {
		at = time.Now().UTC()
	} else {
		at = at.UTC()
	}
	var terminalDeviceID uint
	err := l.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var task model.TransferTask
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where(where, value).First(&task).Error; err != nil {
			return err
		}
		updates := make(map[string]any)
		if !apply(&task, updates) {
			return nil
		}
		fromStatus := task.Status
		artifactAvailable, err := hasAvailableArtifact(tx, task.TaskID)
		if err != nil {
			return err
		}
		target := recomputeTransferStatus(task, artifactAvailable)
		task.Status = target
		updates["status"] = target
		updates["version"] = gorm.Expr("version + 1")
		updates["updated_at"] = at
		if target == model.TransferStatusCompleted || target == model.TransferStatusFailed || target == model.TransferStatusTimeout {
			task.CompletedAt = &at
			updates["completed_at"] = at
			updates["phase_deadline_at"] = nil
		} else if task.Source == model.TransferSourceActive && task.UploadResponseStatus != nil {
			deadline := at.Add(config.CurrentRuntime().TransferCompleteTimeout)
			task.PhaseDeadlineAt = &deadline
			updates["phase_deadline_at"] = deadline
		}
		result := tx.Model(new(model.TransferTask)).Where("task_id = ? AND version = ?", task.TaskID, task.Version).Updates(updates)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return ErrTransferTransitionConflict
		}
		if err := tx.Create(&model.TransferEvent{
			TaskID: task.TaskID, FileID: fileID, Code: eventCode, Phase: phase,
			FromStatus: fromStatus, ToStatus: target, CreatedAt: at,
		}).Error; err != nil {
			return err
		}
		deviceID, terminal, err := updateLifecycleCommand(tx, task, target, at)
		if err == nil && terminal {
			terminalDeviceID = deviceID
		}
		return err
	})
	if err == nil && terminalDeviceID != 0 && l.advancer != nil {
		l.advancer.AdvanceAfterTerminal(ctx, terminalDeviceID)
	}
	return err
}

func hasAvailableArtifact(tx *gorm.DB, taskID string) (bool, error) {
	var count int64
	err := tx.Model(new(model.Artifact)).Where("task_id = ? AND status = ?", taskID, model.ArtifactStatusAvailable).Limit(1).Count(&count).Error
	return count > 0, err
}

func recomputeTransferStatus(task model.TransferTask, artifactAvailable bool) string {
	if task.FailureCode != "" {
		return model.TransferStatusFailed
	}
	if task.Source == model.TransferSourcePeriodic {
		if artifactAvailable {
			return model.TransferStatusCompleted
		}
		return model.TransferStatusReceiving
	}
	if task.UploadResponseStatus == nil {
		if artifactAvailable {
			return model.TransferStatusWaitingTransfer
		}
		return model.TransferStatusWaitingFile
	}
	switch *task.UploadResponseStatus {
	case 0:
		if artifactAvailable {
			return model.TransferStatusCompleted
		}
		return model.TransferStatusWaitingFile
	case 1:
		if artifactAvailable && task.TransferCompletedAt != nil {
			return model.TransferStatusCompleted
		}
		if artifactAvailable {
			return model.TransferStatusWaitingTransfer
		}
		return model.TransferStatusWaitingFile
	default:
		return model.TransferStatusFailed
	}
}

func updateLifecycleCommand(tx *gorm.DB, task model.TransferTask, target string, at time.Time) (uint, bool, error) {
	if task.CommandID == nil {
		return 0, false, nil
	}
	if task.UploadResponseStatus != nil && *task.UploadResponseStatus == 1 &&
		target != model.TransferStatusCompleted && target != model.TransferStatusFailed && target != model.TransferStatusTimeout {
		result := tx.Model(new(model.Command)).
			Where("command_id = ? AND status = ?", *task.CommandID, model.CommandStatusSent).
			Updates(map[string]any{
				"status": model.CommandStatusWaitingTransfer, "phase_deadline_at": task.PhaseDeadlineAt,
				"version": gorm.Expr("version + 1"), "updated_at": at,
			})
		if result.Error != nil {
			return 0, false, result.Error
		}
		if result.RowsAffected == 1 {
			return 0, false, tx.Create(&model.CommandEvent{
				CommandID: *task.CommandID, EventType: "WAITING_TRANSFER", FromStatus: model.CommandStatusSent,
				ToStatus: model.CommandStatusWaitingTransfer, Stage: "transfer.lifecycle", CreatedAt: at,
			}).Error
		}
		return 0, false, nil
	}
	if target != model.TransferStatusCompleted && target != model.TransferStatusFailed && target != model.TransferStatusTimeout {
		return 0, false, nil
	}
	updates := map[string]any{
		"status": target, "finished_at": at, "phase_deadline_at": nil,
		"version": gorm.Expr("version + 1"), "updated_at": at,
	}
	if target == model.TransferStatusFailed {
		faultCode, _ := strconv.Atoi(task.FailureCode)
		updates["failure_stage"] = task.FailureStage
		updates["fault_code"] = faultCode
		updates["fault_string"] = task.FailureMessage
	}
	result := tx.Model(new(model.Command)).Where("command_id = ? AND status IN ?", *task.CommandID, model.NonTerminalCommandStatuses()).Updates(updates)
	if result.Error != nil || result.RowsAffected == 0 {
		return 0, false, result.Error
	}
	if err := tx.Create(&model.CommandEvent{
		CommandID: *task.CommandID, EventType: "TRANSFER_" + target, ToStatus: target,
		Stage: "transfer.lifecycle", Message: task.FailureMessage, CreatedAt: at,
	}).Error; err != nil {
		return 0, false, err
	}
	var command model.Command
	if err := tx.Select("device_id").First(&command, "command_id = ?", *task.CommandID).Error; err != nil {
		return 0, false, err
	}
	return command.DeviceID, true, nil
}
