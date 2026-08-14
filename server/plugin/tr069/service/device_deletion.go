package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/ddddddddwp/gva-acs/server/global"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	DeletionStageMark      = "mark_deleting"
	DeletionStageRuntime   = "stop_runtime"
	DeletionStageArtifacts = "delete_artifacts"
	DeletionStageDatabase  = "delete_database"

	deviceArtifactDeleteConcurrency = 4
)

var ErrDeviceNotFound = errors.New("device not found")

type DeviceDeletionResult struct {
	DeviceID       uint             `json:"deviceId"`
	DeletedObjects int              `json:"deletedObjects"`
	DeletedRows    map[string]int64 `json:"deletedRows"`
}

type DeviceDeletionError struct {
	Stage string
	Cause error
}

func (e *DeviceDeletionError) Error() string {
	if e == nil {
		return "device deletion failed"
	}
	return "device deletion failed at " + e.Stage
}

func (e *DeviceDeletionError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

type DeviceDeletionService struct {
	db      *gorm.DB
	objects ArtifactStore
	uploads *UploadRuntimeRegistry
	runtime DeviceRuntimeCleaner
}

func NewDeviceDeletionService(db *gorm.DB, objects ArtifactStore, uploads *UploadRuntimeRegistry, runtime DeviceRuntimeCleaner) *DeviceDeletionService {
	return &DeviceDeletionService{db: db, objects: objects, uploads: uploads, runtime: runtime}
}

func (s *DeviceDeletionService) Delete(ctx context.Context, deviceID uint) (DeviceDeletionResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if s == nil || s.db == nil || deviceID == 0 {
		return DeviceDeletionResult{}, deletionError(DeletionStageMark, ErrDeviceNotFound)
	}
	startedAt := time.Now()
	device, err := s.markDeleting(ctx, deviceID)
	if err != nil {
		return DeviceDeletionResult{}, deletionError(DeletionStageMark, err)
	}
	identity := DeviceRuntimeIdentity{
		DeviceID:  device.ID,
		DeviceKey: strings.Trim(strings.TrimSpace(device.OUI)+"-"+strings.TrimSpace(device.SerialNumber), "-"),
		IP:        strings.TrimSpace(device.IP),
	}
	if s.uploads != nil {
		if err := s.uploads.BlockAndCancel(ctx, device.ID); err != nil {
			return DeviceDeletionResult{}, deletionError(DeletionStageRuntime, err)
		}
	}
	if s.runtime != nil {
		if err := s.runtime.Purge(ctx, identity); err != nil {
			return DeviceDeletionResult{}, deletionError(DeletionStageRuntime, err)
		}
	}

	objectKeys, err := s.artifactObjectKeys(ctx, device.ID)
	if err != nil {
		return DeviceDeletionResult{}, deletionError(DeletionStageArtifacts, err)
	}
	if len(objectKeys) > 0 && s.objects == nil {
		return DeviceDeletionResult{}, deletionError(DeletionStageArtifacts, errors.New("artifact store is unavailable"))
	}
	if err := deleteArtifactObjects(ctx, s.objects, objectKeys); err != nil {
		return DeviceDeletionResult{}, deletionError(DeletionStageArtifacts, err)
	}

	deletedRows, err := s.deleteDatabaseRows(ctx, device.ID)
	if err != nil {
		return DeviceDeletionResult{}, deletionError(DeletionStageDatabase, err)
	}
	if s.uploads != nil {
		s.uploads.Release(device.ID)
	}
	result := DeviceDeletionResult{DeviceID: device.ID, DeletedObjects: len(objectKeys), DeletedRows: deletedRows}
	if global.GVA_LOG != nil {
		global.GVA_LOG.Info("TR069 device cascade deletion completed",
			zap.Uint("deviceId", device.ID),
			zap.String("serialNumber", summarizedSerial(device.SerialNumber)),
			zap.Int("objects", len(objectKeys)),
			zap.Any("rows", deletedRows),
			zap.Duration("elapsed", time.Since(startedAt)),
		)
	}
	return result, nil
}

func deletionError(stage string, cause error) error {
	if cause == nil {
		cause = errors.New("unknown deletion failure")
	}
	return &DeviceDeletionError{Stage: stage, Cause: cause}
}

func (s *DeviceDeletionService) markDeleting(ctx context.Context, deviceID uint) (model.Device, error) {
	var device model.Device
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&device, deviceID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrDeviceNotFound
			}
			return err
		}
		if device.DeletingAt != nil {
			return nil
		}
		now := time.Now().UTC()
		if err := tx.Model(&device).Where("deleting_at IS NULL").Update("deleting_at", now).Error; err != nil {
			return err
		}
		device.DeletingAt = &now
		return nil
	})
	return device, err
}

func (s *DeviceDeletionService) artifactObjectKeys(ctx context.Context, deviceID uint) ([]string, error) {
	var rows []model.Artifact
	if err := s.db.WithContext(ctx).Select("object_key").Where("device_id = ?", deviceID).Find(&rows).Error; err != nil {
		return nil, err
	}
	keys := make([]string, 0, len(rows))
	seen := make(map[string]struct{}, len(rows))
	for _, row := range rows {
		key := strings.TrimSpace(row.ObjectKey)
		if key == "" {
			continue
		}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		keys = append(keys, key)
	}
	return keys, nil
}

func deleteArtifactObjects(ctx context.Context, store ArtifactStore, keys []string) error {
	if len(keys) == 0 {
		return nil
	}
	workerCount := deviceArtifactDeleteConcurrency
	if len(keys) < workerCount {
		workerCount = len(keys)
	}
	jobs := make(chan string)
	errorsByKey := make(chan error, len(keys))
	var workers sync.WaitGroup
	for index := 0; index < workerCount; index++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for key := range jobs {
				err := store.Delete(ctx, key)
				if err != nil && !errors.Is(err, ErrArtifactNotFound) {
					errorsByKey <- fmt.Errorf("delete artifact object: %w", err)
				}
			}
		}()
	}
	for _, key := range keys {
		jobs <- key
	}
	close(jobs)
	workers.Wait()
	close(errorsByKey)
	var result error
	for err := range errorsByKey {
		result = errors.Join(result, err)
	}
	return result
}

func (s *DeviceDeletionService) deleteDatabaseRows(ctx context.Context, deviceID uint) (map[string]int64, error) {
	deleted := make(map[string]int64)
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var commandIDs []string
		if err := tx.Model(new(model.Command)).Where("device_id = ?", deviceID).Pluck("command_id", &commandIDs).Error; err != nil {
			return err
		}
		var taskIDs []string
		if err := tx.Model(new(model.TransferTask)).Where("device_id = ?", deviceID).Pluck("task_id", &taskIDs).Error; err != nil {
			return err
		}
		var fileIDs []uint64
		if err := tx.Model(new(model.Artifact)).Where("device_id = ?", deviceID).Pluck("id", &fileIDs).Error; err != nil {
			return err
		}

		if len(taskIDs) > 0 || len(fileIDs) > 0 {
			query := tx.Unscoped().Model(new(model.TransferEvent))
			switch {
			case len(taskIDs) > 0 && len(fileIDs) > 0:
				query = query.Where("task_id IN ? OR file_id IN ?", taskIDs, fileIDs)
			case len(taskIDs) > 0:
				query = query.Where("task_id IN ?", taskIDs)
			default:
				query = query.Where("file_id IN ?", fileIDs)
			}
			if err := deleteRows(query, new(model.TransferEvent), deleted, "transfer_events"); err != nil {
				return err
			}
		}
		if err := deleteDeviceRows(tx, new(model.Artifact), deviceID, deleted, "artifacts"); err != nil {
			return err
		}
		if err := deleteDeviceRows(tx, new(model.TransferTask), deviceID, deleted, "transfer_tasks"); err != nil {
			return err
		}
		if len(commandIDs) > 0 {
			if err := deleteRows(tx.Unscoped().Where("command_id IN ?", commandIDs), new(model.CommandXML), deleted, "command_xmls"); err != nil {
				return err
			}
			if err := deleteRows(tx.Unscoped().Where("command_id IN ?", commandIDs), new(model.CommandEvent), deleted, "command_events"); err != nil {
				return err
			}
		}
		for _, item := range []struct {
			value any
			name  string
		}{
			{new(model.Command), "commands"},
			{new(model.Tr069Alarm), "alarms"},
			{new(model.SupportTr069Alarm), "support_alarms"},
			{new(model.DataModelValue), "datamodel_values"},
			{new(model.DeviceRPCMethods), "device_rpc_methods"},
			{new(model.FAPService), "fap_services"},
			{new(model.ConnectionProfile), "connection_profiles"},
		} {
			if err := deleteDeviceRows(tx, item.value, deviceID, deleted, item.name); err != nil {
				return err
			}
		}
		result := tx.Unscoped().Where("id = ?", deviceID).Delete(new(model.Device))
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return ErrDeviceNotFound
		}
		deleted["devices"] = result.RowsAffected
		return nil
	})
	return deleted, err
}

func deleteDeviceRows(tx *gorm.DB, value any, deviceID uint, deleted map[string]int64, name string) error {
	return deleteRows(tx.Unscoped().Where("device_id = ?", deviceID), value, deleted, name)
}

func deleteRows(query *gorm.DB, value any, deleted map[string]int64, name string) error {
	result := query.Delete(value)
	if result.Error != nil {
		return result.Error
	}
	deleted[name] += result.RowsAffected
	return nil
}

func summarizedSerial(serial string) string {
	serial = strings.TrimSpace(serial)
	if len(serial) <= 4 {
		return serial
	}
	return "***" + serial[len(serial)-4:]
}
