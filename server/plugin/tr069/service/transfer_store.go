package service

import (
	"context"
	"errors"
	"fmt"
	"time"

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
	TaskID       string
	ArtifactID   string
	ObjectKey    string
	Driver       string
	OriginalName string
	ContentType  string
	SourceIP     string
	DeleteAt     *time.Time
	CreatedAt    time.Time
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
	DeviceID    uint
	Channel     string
	Status      string
	CreatedFrom *time.Time
	CreatedTo   *time.Time
	Offset      int
	Limit       int
}

type ArtifactListItem struct {
	ArtifactID   string     `json:"artifactId"`
	TaskID       string     `json:"taskId"`
	DeviceID     uint       `json:"deviceId"`
	SerialNumber string     `json:"serialNumber"`
	OUI          string     `json:"oui"`
	Channel      string     `json:"channel"`
	Source       string     `json:"source"`
	Status       string     `json:"status"`
	OriginalName string     `json:"originalName"`
	ContentType  string     `json:"contentType"`
	Size         int64      `json:"size"`
	SHA256       string     `json:"sha256"`
	ReceivedAt   *time.Time `json:"receivedAt"`
	CreatedAt    time.Time  `json:"createdAt"`
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
	if metadata.ArtifactID == "" {
		metadata.ArtifactID = uuid.NewString()
	}
	task := model.TransferTask{
		TaskID: metadata.TaskID, DeviceID: deviceID, Channel: channel,
		Source: model.TransferSourcePeriodic, Status: model.TransferStatusReceiving, CreatedAt: now, UpdatedAt: now,
	}
	artifact := model.Artifact{
		ArtifactID: metadata.ArtifactID, TaskID: task.TaskID, DeviceID: deviceID, Channel: channel,
		Status: model.ArtifactStatusReceiving, Driver: metadata.Driver, ObjectKey: metadata.ObjectKey,
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
		return tx.Create(&model.TransferEvent{
			TaskID: task.TaskID, ArtifactID: artifact.ArtifactID, Code: "RECEIVING_STARTED",
			Phase: "ingress.receive", ToStatus: task.Status, CreatedAt: now,
		}).Error
	})
	return task, artifact, err
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

func (s *TransferStore) MarkArtifactAvailable(ctx context.Context, artifactID string, expectedVersion uint, final ArtifactFinalization) (model.Artifact, error) {
	receivedAt := final.ReceivedAt.UTC()
	if receivedAt.IsZero() {
		receivedAt = time.Now().UTC()
	}
	var artifact model.Artifact
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(new(model.Artifact)).
			Where("artifact_id = ? AND status = ? AND version = ?", artifactID, model.ArtifactStatusReceiving, expectedVersion).
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
		return tx.Where("artifact_id = ?", artifactID).First(&artifact).Error
	})
	return artifact, err
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
		Joins("JOIN tr069_devices AS devices ON devices.id = artifacts.device_id")
	if filter.DeviceID != 0 {
		query = query.Where("artifacts.device_id = ?", filter.DeviceID)
	}
	if filter.Channel != "" {
		query = query.Where("artifacts.channel = ?", filter.Channel)
	}
	if filter.Status != "" {
		query = query.Where("artifacts.status = ?", filter.Status)
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
	err := query.Select(`artifacts.artifact_id, artifacts.task_id, artifacts.device_id,
		devices.serial_number, devices.oui, artifacts.channel, tasks.source, artifacts.status,
		artifacts.original_name, artifacts.content_type, artifacts.size, artifacts.sha256,
		artifacts.received_at, artifacts.created_at`).
		Order("artifacts.created_at DESC").Order("artifacts.artifact_id DESC").
		Offset(offset).Limit(limit).Scan(&items).Error
	return items, total, err
}

func (s *TransferStore) GetAvailableArtifact(ctx context.Context, artifactID string) (model.Artifact, error) {
	var artifact model.Artifact
	err := s.db.WithContext(ctx).Where("artifact_id = ? AND status = ?", artifactID, model.ArtifactStatusAvailable).First(&artifact).Error
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
