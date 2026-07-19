package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/config"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/model"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrTransferTransitionConflict = errors.New("transfer transition conflict")
	ErrArtifactTransitionConflict = errors.New("artifact transition conflict")
	ErrAmbiguousActiveTransfer    = errors.New("multiple active transfer tasks match device and channel")
)

type ReceiveMetadata struct {
	TaskID        string
	StoragePrefix string
	Driver        string
	OriginalName  string
	ContentType   string
	SourceIP      string
	DeleteAt      *time.Time
	CreatedAt     time.Time
}

type ArtifactFinalization struct {
	Size       int64
	SHA256     string
	ReceivedAt time.Time
}

type TransferTransition struct {
	TaskID          string
	FromStatuses    []string
	ToStatus        string
	ExpectedVersion uint
	EventCode       string
	Phase           string
	Message         string
	MetadataJSON    datatypes.JSON
	Updates         map[string]any
}

type ArtifactListFilter struct {
	SerialNumber string
	CreatedFrom  *time.Time
	CreatedTo    *time.Time
	Offset       int
	Limit        int
}

type ArtifactListItem struct {
	FileID       uint64     `json:"fileId"`
	SerialNumber string     `json:"serialNumber"`
	Source       string     `json:"source"`
	OriginalName string     `json:"originalName"`
	Size         int64      `json:"size"`
	ReceivedAt   *time.Time `json:"receivedAt"`
}

type TransferStore struct {
	db *gorm.DB
}

func NewTransferStore(db *gorm.DB) *TransferStore { return &TransferStore{db: db} }

func (s *TransferStore) CreateActive(ctx context.Context, command *model.Command, task *model.TransferTask) error {
	if s == nil || s.db == nil {
		return errors.New("transfer database is required")
	}
	if command == nil || task == nil || command.CommandID == "" || task.TaskID == "" {
		return errors.New("command and active transfer task are required")
	}
	if task.CommandID == nil || *task.CommandID != command.CommandID {
		return errors.New("active transfer task command ID must match command")
	}
	now := time.Now().UTC()
	if command.CreatedAt.IsZero() {
		command.CreatedAt = now
	}
	if command.QueuedAt.IsZero() {
		command.QueuedAt = now
	}
	if task.CreatedAt.IsZero() {
		task.CreatedAt = now
	}
	if task.Source == "" {
		task.Source = model.TransferSourceActive
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(command).Error; err != nil {
			return err
		}
		if err := tx.Create(&model.CommandEvent{CommandID: command.CommandID, EventType: model.CommandEventCreated, ToStatus: command.Status, CreatedAt: command.CreatedAt}).Error; err != nil {
			return err
		}
		if err := tx.Create(task).Error; err != nil {
			return err
		}
		return tx.Create(&model.TransferEvent{TaskID: task.TaskID, Code: "CREATED", ToStatus: task.Status, Phase: "task.create", CreatedAt: task.CreatedAt}).Error
	})
}

func (s *TransferStore) CreateActiveTask(ctx context.Context, command *model.Command, taskID string) error {
	if s == nil || s.db == nil || command == nil || command.CommandID == "" || taskID == "" {
		return errors.New("persisted command and active transfer task ID are required")
	}
	if command.Operation != "Upload" || command.CommandKey == nil || *command.CommandKey == "" {
		return errors.New("active transfer task requires an Upload command and CommandKey")
	}
	now := command.CreatedAt
	if now.IsZero() {
		now = time.Now().UTC()
	}
	waitTimeout := 10 * time.Minute
	if channel, ok := config.CurrentRuntime().FileIngress.Channels["log"]; ok && channel.UploadTimeout > 0 {
		waitTimeout = channel.UploadTimeout
	}
	deadline := now.Add(waitTimeout)
	task := model.TransferTask{
		TaskID: taskID, DeviceID: command.DeviceID, Channel: "LOG", Source: model.TransferSourceActive,
		CommandID: &command.CommandID, CommandKey: command.CommandKey, Status: model.TransferStatusWaitingFile,
		PhaseDeadlineAt: &deadline, CreatedAt: now, UpdatedAt: now,
	}
	if err := s.db.WithContext(ctx).Create(&task).Error; err != nil {
		return err
	}
	return s.db.WithContext(ctx).Create(&model.TransferEvent{
		TaskID: task.TaskID, Code: "CREATED", Phase: "task.create", ToStatus: task.Status, CreatedAt: now,
	}).Error
}

func (s *TransferStore) CreatePeriodicReceiving(ctx context.Context, deviceID uint, channel string, metadata ReceiveMetadata) (model.TransferTask, model.Artifact, error) {
	if s == nil || s.db == nil {
		return model.TransferTask{}, model.Artifact{}, errors.New("transfer database is required")
	}
	now := metadata.CreatedAt.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}
	if metadata.TaskID == "" {
		metadata.TaskID = uuid.NewString()
	}
	task := model.TransferTask{
		TaskID: metadata.TaskID, DeviceID: deviceID, Channel: channel,
		Source: model.TransferSourcePeriodic, Status: model.TransferStatusReceiving, CreatedAt: now, UpdatedAt: now,
	}
	artifact := model.Artifact{
		TaskID: task.TaskID, DeviceID: deviceID, Channel: channel,
		Status: model.ArtifactStatusReceiving, Driver: metadata.Driver, ObjectKey: "pending/" + task.TaskID,
		OriginalName: metadata.OriginalName, ContentType: metadata.ContentType, SourceIP: metadata.SourceIP,
		DeleteAt: metadata.DeleteAt, CreatedAt: now, UpdatedAt: now,
	}
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&task).Error; err != nil {
			return err
		}
		if err := tx.Create(&artifact).Error; err != nil {
			return err
		}
		objectKey, err := ArtifactObjectKey(metadata.StoragePrefix, artifact.Channel, artifact.DeviceID, now, artifact.ID)
		if err != nil {
			return err
		}
		if err := tx.Model(&artifact).Update("object_key", objectKey).Error; err != nil {
			return err
		}
		artifact.ObjectKey = objectKey
		return tx.Create(&model.TransferEvent{
			TaskID: task.TaskID, FileID: artifact.ID, Code: "RECEIVING_STARTED",
			Phase: "ingress.receive", ToStatus: task.Status, CreatedAt: now,
		}).Error
	})
	return task, artifact, err
}

func (s *TransferStore) CreateActiveReceiving(ctx context.Context, task model.TransferTask, metadata ReceiveMetadata) (model.TransferTask, model.Artifact, error) {
	if task.TaskID == "" || task.Source != model.TransferSourceActive {
		return model.TransferTask{}, model.Artifact{}, errors.New("active transfer task is required")
	}
	now := metadata.CreatedAt.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}
	artifact := model.Artifact{
		TaskID: task.TaskID, DeviceID: task.DeviceID, Channel: task.Channel,
		Status: model.ArtifactStatusReceiving, Driver: metadata.Driver, ObjectKey: "pending/" + task.TaskID,
		OriginalName: metadata.OriginalName, ContentType: metadata.ContentType, SourceIP: metadata.SourceIP,
		DeleteAt: metadata.DeleteAt, CreatedAt: now, UpdatedAt: now,
	}
	var transitioned model.TransferTask
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(new(model.TransferTask)).
			Where("task_id = ? AND source = ? AND status = ? AND version = ?", task.TaskID, model.TransferSourceActive, model.TransferStatusWaitingFile, task.Version).
			Updates(map[string]any{"status": model.TransferStatusReceiving, "version": gorm.Expr("version + 1"), "updated_at": now})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return ErrTransferTransitionConflict
		}
		if err := tx.Create(&artifact).Error; err != nil {
			return err
		}
		objectKey, err := ArtifactObjectKey(metadata.StoragePrefix, artifact.Channel, artifact.DeviceID, now, artifact.ID)
		if err != nil {
			return err
		}
		if err := tx.Model(&artifact).Update("object_key", objectKey).Error; err != nil {
			return err
		}
		artifact.ObjectKey = objectKey
		if err := tx.Create(&model.TransferEvent{
			TaskID: task.TaskID, FileID: artifact.ID, Code: "RECEIVING_STARTED", Phase: "ingress.receive",
			FromStatus: task.Status, ToStatus: model.TransferStatusReceiving, CreatedAt: now,
		}).Error; err != nil {
			return err
		}
		return tx.Where("task_id = ?", task.TaskID).First(&transitioned).Error
	})
	return transitioned, artifact, err
}

func (s *TransferStore) FindUniqueWaitingActive(ctx context.Context, deviceID uint, channel string) (model.TransferTask, error) {
	var tasks []model.TransferTask
	err := s.db.WithContext(ctx).
		Where("device_id = ? AND channel = ? AND source = ? AND status IN ?", deviceID, channel, model.TransferSourceActive,
			[]string{model.TransferStatusWaitingFile, model.TransferStatusReceiving, model.TransferStatusWaitingTransfer}).
		Order("created_at ASC").Limit(2).Find(&tasks).Error
	if err != nil {
		return model.TransferTask{}, err
	}
	if len(tasks) == 0 {
		return model.TransferTask{}, gorm.ErrRecordNotFound
	}
	if len(tasks) > 1 {
		return model.TransferTask{}, ErrAmbiguousActiveTransfer
	}
	return tasks[0], nil
}

func (s *TransferStore) GetTaskArtifact(ctx context.Context, taskID string) (model.Artifact, error) {
	var artifact model.Artifact
	err := s.db.WithContext(ctx).Where("task_id = ?", taskID).First(&artifact).Error
	return artifact, err
}

func (s *TransferStore) AppendTransferEvent(ctx context.Context, event model.TransferEvent) error {
	if s == nil || s.db == nil || event.TaskID == "" || event.Code == "" {
		return errors.New("transfer event task ID and code are required")
	}
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now().UTC()
	}
	return s.db.WithContext(ctx).Create(&event).Error
}

func (s *TransferStore) MarkArtifactAvailable(ctx context.Context, fileID uint64, expectedVersion uint, final ArtifactFinalization) (model.Artifact, error) {
	receivedAt := final.ReceivedAt.UTC()
	if receivedAt.IsZero() {
		receivedAt = time.Now().UTC()
	}
	var artifact model.Artifact
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(new(model.Artifact)).
			Where("id = ? AND status = ? AND version = ?", fileID, model.ArtifactStatusReceiving, expectedVersion).
			Updates(map[string]any{
				"status": model.ArtifactStatusAvailable, "size": final.Size, "sha256": final.SHA256,
				"received_at": receivedAt, "version": gorm.Expr("version + 1"), "updated_at": receivedAt,
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return ErrArtifactTransitionConflict
		}
		return tx.Where("id = ?", fileID).First(&artifact).Error
	})
	return artifact, err
}

func (s *TransferStore) MarkArtifactFailed(ctx context.Context, fileID uint64, expectedVersion uint) error {
	result := s.db.WithContext(ctx).Model(new(model.Artifact)).
		Where("id = ? AND status = ? AND version = ?", fileID, model.ArtifactStatusReceiving, expectedVersion).
		Updates(map[string]any{"status": model.ArtifactStatusFailed, "version": gorm.Expr("version + 1"), "updated_at": time.Now().UTC()})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return ErrArtifactTransitionConflict
	}
	return nil
}

func (s *TransferStore) TransitionTask(ctx context.Context, transition TransferTransition) (model.TransferTask, error) {
	if transition.TaskID == "" || len(transition.FromStatuses) == 0 || transition.ToStatus == "" {
		return model.TransferTask{}, errors.New("task ID, source statuses, and target status are required")
	}
	var transitioned model.TransferTask
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var current model.TransferTask
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("task_id = ?", transition.TaskID).First(&current).Error; err != nil {
			return err
		}
		if current.Version != transition.ExpectedVersion || !containsStatus(transition.FromStatuses, current.Status) {
			return ErrTransferTransitionConflict
		}
		updates := make(map[string]any, len(transition.Updates)+3)
		for column, value := range transition.Updates {
			switch column {
			case "task_id", "device_id", "command_id", "status", "version", "created_at":
				return fmt.Errorf("transfer transition cannot update %s", column)
			default:
				updates[column] = value
			}
		}
		now := time.Now().UTC()
		updates["status"] = transition.ToStatus
		updates["version"] = gorm.Expr("version + 1")
		updates["updated_at"] = now
		result := tx.Model(new(model.TransferTask)).
			Where("task_id = ? AND status IN ? AND version = ?", transition.TaskID, transition.FromStatuses, transition.ExpectedVersion).
			Updates(updates)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return ErrTransferTransitionConflict
		}
		if err := tx.Create(&model.TransferEvent{
			TaskID: transition.TaskID, Code: transition.EventCode, Phase: transition.Phase,
			FromStatus: current.Status, ToStatus: transition.ToStatus, Message: transition.Message,
			MetadataJSON: model.LongTextJSON(transition.MetadataJSON), CreatedAt: now,
		}).Error; err != nil {
			return err
		}
		return tx.Where("task_id = ?", transition.TaskID).First(&transitioned).Error
	})
	return transitioned, err
}

func (s *TransferStore) ListArtifacts(ctx context.Context, filter ArtifactListFilter) ([]ArtifactListItem, int64, error) {
	query := s.db.WithContext(ctx).Table("tr069_artifacts AS artifacts").
		Joins("JOIN tr069_transfer_tasks AS tasks ON tasks.task_id = artifacts.task_id").
		Joins("JOIN tr069_devices AS devices ON devices.id = artifacts.device_id").
		Where("artifacts.channel = ? AND artifacts.status = ?", "LOG", model.ArtifactStatusAvailable)
	if filter.SerialNumber != "" {
		query = query.Where("devices.serial_number = ?", filter.SerialNumber)
	}
	if filter.CreatedFrom != nil {
		query = query.Where("artifacts.created_at >= ?", *filter.CreatedFrom)
	}
	if filter.CreatedTo != nil {
		query = query.Where("artifacts.created_at <= ?", *filter.CreatedTo)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	limit := filter.Limit
	if limit <= 0 {
		limit = 20
	} else if limit > 100 {
		limit = 100
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}
	var items []ArtifactListItem
	err := query.Select(`artifacts.id AS file_id, devices.serial_number, tasks.source,
		artifacts.original_name, artifacts.size, artifacts.received_at`).
		Order("artifacts.created_at DESC").Order("artifacts.id DESC").
		Offset(offset).Limit(limit).Scan(&items).Error
	return items, total, err
}

func (s *TransferStore) GetAvailableArtifact(ctx context.Context, fileID uint64) (model.Artifact, error) {
	var artifact model.Artifact
	err := s.db.WithContext(ctx).Where("id = ? AND status = ?", fileID, model.ArtifactStatusAvailable).First(&artifact).Error
	return artifact, err
}

func (s *TransferStore) ListStaleReceiving(ctx context.Context, before time.Time, limit int) ([]model.Artifact, error) {
	if limit <= 0 {
		limit = 100
	}
	var artifacts []model.Artifact
	err := s.db.WithContext(ctx).Where("status = ? AND created_at < ?", model.ArtifactStatusReceiving, before).
		Order("created_at ASC").Limit(limit).Find(&artifacts).Error
	return artifacts, err
}
