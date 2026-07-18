package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"sync"
	"time"

	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/config"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/model"
	"gorm.io/gorm"
)

const transferWorkerBatchSize = 100

type TransferWorkers struct {
	transfers *TransferStore
	objects   ArtifactStore
	lifecycle *TransferLifecycle
	now       func() time.Time
	interval  time.Duration

	mu     sync.Mutex
	cancel context.CancelFunc
	done   chan struct{}
}

func NewTransferWorkers(transfers *TransferStore, objects ArtifactStore) *TransferWorkers {
	var lifecycle *TransferLifecycle
	if transfers != nil {
		lifecycle = NewTransferLifecycle(transfers.db)
	}
	return &TransferWorkers{transfers: transfers, objects: objects, lifecycle: lifecycle, now: time.Now, interval: 30 * time.Second}
}

func (w *TransferWorkers) Run(ctx context.Context) {
	if w == nil || w.transfers == nil || w.objects == nil {
		return
	}
	w.mu.Lock()
	if w.cancel != nil {
		w.mu.Unlock()
		return
	}
	runCtx, cancel := context.WithCancel(ctx)
	w.cancel = cancel
	w.done = make(chan struct{})
	done := w.done
	w.mu.Unlock()
	defer close(done)

	_ = w.RunOnce(runCtx)
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()
	for {
		select {
		case <-runCtx.Done():
			return
		case <-ticker.C:
			_ = w.RunOnce(runCtx)
		}
	}
}

func (w *TransferWorkers) Stop(ctx context.Context) error {
	if w == nil {
		return nil
	}
	w.mu.Lock()
	cancel := w.cancel
	done := w.done
	w.mu.Unlock()
	if cancel == nil || done == nil {
		return nil
	}
	cancel()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (w *TransferWorkers) RunOnce(ctx context.Context) error {
	if w == nil || w.transfers == nil || w.transfers.db == nil || w.objects == nil || w.lifecycle == nil {
		return errors.New("transfer workers are not initialized")
	}
	now := w.now().UTC()
	return errors.Join(w.reconcileReceiving(ctx, now), w.expireTasks(ctx, now), w.deleteExpiredArtifacts(ctx, now))
}

func (w *TransferWorkers) reconcileReceiving(ctx context.Context, now time.Time) error {
	uploadTimeout := 10 * time.Minute
	if channel, ok := config.CurrentRuntime().FileIngress.Channels["log"]; ok && channel.UploadTimeout > 0 {
		uploadTimeout = channel.UploadTimeout
	}
	artifacts, err := w.transfers.ListStaleReceiving(ctx, now.Add(-uploadTimeout), transferWorkerBatchSize)
	if err != nil {
		return err
	}
	var result error
	buffer := make([]byte, 64*1024)
	for _, artifact := range artifacts {
		stat, statErr := w.objects.Stat(ctx, artifact.ObjectKey)
		if errors.Is(statErr, ErrArtifactNotFound) {
			result = errors.Join(result, w.failReceivingArtifact(ctx, artifact, "RECONCILE_OBJECT_MISSING", now))
			continue
		}
		if statErr != nil {
			result = errors.Join(result, statErr)
			continue
		}
		reader, _, openErr := w.objects.Open(ctx, artifact.ObjectKey)
		if openErr != nil {
			result = errors.Join(result, openErr)
			continue
		}
		hasher := sha256.New()
		size, copyErr := io.CopyBuffer(hasher, reader, buffer)
		closeErr := reader.Close()
		if copyErr != nil || closeErr != nil || size != stat.Size || (artifact.Size > 0 && artifact.Size != size) || (artifact.SHA256 != "" && artifact.SHA256 != hex.EncodeToString(hasher.Sum(nil))) {
			_ = w.objects.Delete(context.Background(), artifact.ObjectKey)
			result = errors.Join(result, copyErr, closeErr, w.failReceivingArtifact(ctx, artifact, "RECONCILE_OBJECT_MISMATCH", now))
			continue
		}
		available, finalizeErr := w.transfers.MarkArtifactAvailable(ctx, artifact.ArtifactID, artifact.Version, ArtifactFinalization{
			Size: size, SHA256: hex.EncodeToString(hasher.Sum(nil)), ReceivedAt: now,
		})
		if errors.Is(finalizeErr, ErrArtifactTransitionConflict) {
			continue
		}
		if finalizeErr != nil {
			result = errors.Join(result, finalizeErr)
			continue
		}
		if lifecycleErr := w.lifecycle.OnArtifactAvailable(ctx, available.TaskID, available.ArtifactID, now); lifecycleErr != nil {
			result = errors.Join(result, lifecycleErr)
		}
	}
	return result
}

func (w *TransferWorkers) failReceivingArtifact(ctx context.Context, artifact model.Artifact, code string, at time.Time) error {
	if err := w.transfers.MarkArtifactFailed(ctx, artifact.ArtifactID, artifact.Version); err != nil && !errors.Is(err, ErrArtifactTransitionConflict) {
		return err
	}
	var task model.TransferTask
	if err := w.transfers.db.WithContext(ctx).First(&task, "task_id = ?", artifact.TaskID).Error; err != nil {
		return err
	}
	if task.Status == model.TransferStatusFailed || task.Status == model.TransferStatusCompleted || task.Status == model.TransferStatusTimeout {
		return nil
	}
	_, err := w.transfers.TransitionTask(ctx, TransferTransition{
		TaskID: task.TaskID, FromStatuses: []string{task.Status}, ToStatus: model.TransferStatusFailed,
		ExpectedVersion: task.Version, EventCode: code, Phase: "worker.reconcile",
		Updates: map[string]any{"failure_stage": "worker.reconcile", "failure_code": code, "completed_at": at, "phase_deadline_at": nil},
	})
	if errors.Is(err, ErrTransferTransitionConflict) {
		return nil
	}
	return err
}

func (w *TransferWorkers) expireTasks(ctx context.Context, now time.Time) error {
	var tasks []model.TransferTask
	if err := w.transfers.db.WithContext(ctx).
		Where("status IN ? AND phase_deadline_at IS NOT NULL AND phase_deadline_at <= ?",
			[]string{model.TransferStatusWaitingFile, model.TransferStatusReceiving, model.TransferStatusWaitingTransfer}, now).
		Order("phase_deadline_at ASC").Limit(transferWorkerBatchSize).Find(&tasks).Error; err != nil {
		return err
	}
	var result error
	for _, task := range tasks {
		_, err := w.transfers.TransitionTask(ctx, TransferTransition{
			TaskID: task.TaskID, FromStatuses: []string{task.Status}, ToStatus: model.TransferStatusTimeout,
			ExpectedVersion: task.Version, EventCode: "TRANSFER_TIMEOUT", Phase: "worker.timeout",
			Updates: map[string]any{"failure_stage": "worker.timeout", "failure_code": "TRANSFER_TIMEOUT", "completed_at": now, "phase_deadline_at": nil},
		})
		if err != nil && !errors.Is(err, ErrTransferTransitionConflict) {
			result = errors.Join(result, err)
		}
	}
	return result
}

func (w *TransferWorkers) deleteExpiredArtifacts(ctx context.Context, now time.Time) error {
	var artifacts []model.Artifact
	if err := w.transfers.db.WithContext(ctx).
		Where("(status = ? AND delete_at IS NOT NULL AND delete_at <= ?) OR status = ?", model.ArtifactStatusAvailable, now, model.ArtifactStatusDeleting).
		Order("delete_at ASC").Limit(transferWorkerBatchSize).Find(&artifacts).Error; err != nil {
		return err
	}
	var result error
	for _, artifact := range artifacts {
		if artifact.Status == model.ArtifactStatusAvailable {
			claim := w.transfers.db.WithContext(ctx).Model(new(model.Artifact)).
				Where("artifact_id = ? AND status = ? AND version = ?", artifact.ArtifactID, model.ArtifactStatusAvailable, artifact.Version).
				Updates(map[string]any{"status": model.ArtifactStatusDeleting, "version": gorm.Expr("version + 1"), "updated_at": now})
			if claim.Error != nil {
				result = errors.Join(result, claim.Error)
				continue
			}
			if claim.RowsAffected != 1 {
				continue
			}
			artifact.Status = model.ArtifactStatusDeleting
			artifact.Version++
		}
		deleteErr := w.objects.Delete(ctx, artifact.ObjectKey)
		if deleteErr != nil && !errors.Is(deleteErr, ErrArtifactNotFound) {
			_ = w.transfers.db.WithContext(ctx).Create(&model.TransferEvent{
				TaskID: artifact.TaskID, ArtifactID: artifact.ArtifactID, Code: "RETENTION_DELETE_FAILED",
				Phase: "worker.retention", Message: deleteErr.Error(), CreatedAt: now,
			}).Error
			result = errors.Join(result, deleteErr)
			continue
		}
		deletedAt := now
		update := w.transfers.db.WithContext(ctx).Model(new(model.Artifact)).
			Where("artifact_id = ? AND status = ? AND version = ?", artifact.ArtifactID, model.ArtifactStatusDeleting, artifact.Version).
			Updates(map[string]any{"status": model.ArtifactStatusDeleted, "deleted_at": deletedAt, "version": gorm.Expr("version + 1"), "updated_at": now})
		if update.Error != nil {
			result = errors.Join(result, update.Error)
			continue
		}
		if update.RowsAffected == 1 {
			_ = w.transfers.db.WithContext(ctx).Create(&model.TransferEvent{
				TaskID: artifact.TaskID, ArtifactID: artifact.ArtifactID, Code: "RETENTION_DELETED",
				Phase: "worker.retention", CreatedAt: now,
			}).Error
		}
	}
	return result
}
