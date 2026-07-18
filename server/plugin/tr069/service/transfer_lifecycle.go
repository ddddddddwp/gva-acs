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
	db *gorm.DB
}

func NewTransferLifecycle(db *gorm.DB) *TransferLifecycle {
	return &TransferLifecycle{db: db}
}

func (l *TransferLifecycle) OnUploadResponse(ctx context.Context, commandID string, status int, at time.Time) error {
	return l.mutate(ctx, "command_id = ?", commandID, "UPLOAD_RESPONSE", "rpc.response", "", at,
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
	return l.mutate(ctx, "command_key = ?", commandKey, "TRANSFER_COMPLETE", "transfer.complete", "", at,
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

func (l *TransferLifecycle) OnArtifactAvailable(ctx context.Context, taskID, artifactID string, at time.Time) error {
	if l == nil || l.db == nil {
		return errors.New("transfer lifecycle database is required")
	}
	var artifact model.Artifact
	if err := l.db.WithContext(ctx).Where("artifact_id = ? AND task_id = ? AND status = ?", artifactID, taskID, model.ArtifactStatusAvailable).First(&artifact).Error; err != nil {
		return err
	}
	return l.mutate(ctx, "task_id = ?", taskID, "ARTIFACT_AVAILABLE", "storage.commit", artifactID, at,
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

func (l *TransferLifecycle) mutate(ctx context.Context, where string, value any, eventCode, phase, artifactID string, at time.Time, apply transferFactMutation) error {
	if l == nil || l.db == nil {
		return errors.New("transfer lifecycle database is required")
	}
	if at.IsZero() {
		at = time.Now().UTC()
	} else {
		at = at.UTC()
	}
	return l.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
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
			TaskID: task.TaskID, ArtifactID: artifactID, Code: eventCode, Phase: phase,
			FromStatus: fromStatus, ToStatus: target, CreatedAt: at,
		}).Error; err != nil {
			return err
		}
		return updateLifecycleCommand(tx, task, target, at)
	})
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

func updateLifecycleCommand(tx *gorm.DB, task model.TransferTask, target string, at time.Time) error {
	if task.CommandID == nil || (target != model.TransferStatusCompleted && target != model.TransferStatusFailed && target != model.TransferStatusTimeout) {
		return nil
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
		return result.Error
	}
	return tx.Create(&model.CommandEvent{
		CommandID: *task.CommandID, EventType: "TRANSFER_" + target, ToStatus: target,
		Stage: "transfer.lifecycle", Message: task.FailureMessage, CreatedAt: at,
	}).Error
}
