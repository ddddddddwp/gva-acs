package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/model"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/redact"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

var (
	ErrCommandTransitionConflict = errors.New("command transition conflict")
	ErrInvalidCommandTransition  = errors.New("invalid command transition")
	ErrInvalidCommandJSON        = errors.New("invalid command JSON")
	ErrInvalidCommandXML         = errors.New("invalid command XML")
	ErrUncorrelatedCommandXML    = errors.New("uncorrelated command XML")
	ErrInvalidCommandStatus      = errors.New("invalid command status")
)

type CommandTransition struct {
	CommandID       string
	FromStatuses    []string
	ToStatus        string
	ExpectedVersion uint
	EventType       string
	Stage           string
	Message         string
	PayloadJSON     datatypes.JSON
	Updates         map[string]any
}

type CommandListFilter struct {
	DeviceID     uint
	DeviceSerial string
	Operation    string
	Status       string
	CommandID    string
	CreatedFrom  *time.Time
	CreatedTo    *time.Time
	Offset       int
	Limit        int
}

type CommandDetail struct {
	Command model.Command        `json:"command"`
	Events  []model.CommandEvent `json:"events"`
	XML     []model.CommandXML   `json:"xml"`
}

type CommandStore struct {
	db *gorm.DB
}

func NewCommandStore(db *gorm.DB) *CommandStore {
	return &CommandStore{db: db}
}

func (s *CommandStore) Create(ctx context.Context, command *model.Command) error {
	if command == nil {
		return errors.New("command is required")
	}
	if !model.IsCommandStatus(command.Status) {
		return fmt.Errorf("%w: %q", ErrInvalidCommandStatus, command.Status)
	}
	if err := validateJSONDocument("params", command.ParamsJSON); err != nil {
		return err
	}
	if err := validateJSONDocument("result", command.ResultJSON); err != nil {
		return err
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(command).Error; err != nil {
			return err
		}
		return tx.Create(&model.CommandEvent{
			CommandID: command.CommandID,
			EventType: model.CommandEventCreated,
			ToStatus:  command.Status,
			CreatedAt: command.CreatedAt,
		}).Error
	})
}

func validateJSONDocument(field string, value []byte) error {
	if len(value) > 0 && !json.Valid(value) {
		return fmt.Errorf("%w: %s", ErrInvalidCommandJSON, field)
	}
	return nil
}

func validateJSONUpdate(field string, value any) error {
	var raw []byte
	switch typed := value.(type) {
	case nil:
		return nil
	case datatypes.JSON:
		raw = typed
	case model.LongTextJSON:
		raw = typed
	case []byte:
		raw = typed
	case string:
		raw = []byte(typed)
	default:
		return fmt.Errorf("%w: %s must be serialized JSON", ErrInvalidCommandJSON, field)
	}
	if len(raw) > 0 && !json.Valid(raw) {
		return fmt.Errorf("%w: %s", ErrInvalidCommandJSON, field)
	}
	return nil
}

func (s *CommandStore) Transition(ctx context.Context, transition CommandTransition) (model.Command, error) {
	if transition.CommandID == "" || len(transition.FromStatuses) == 0 {
		return model.Command{}, fmt.Errorf("%w: command ID and source statuses are required", ErrInvalidCommandTransition)
	}
	if err := validateJSONDocument("event payload", transition.PayloadJSON); err != nil {
		return model.Command{}, err
	}
	for _, column := range []string{"params_json", "result_json"} {
		if value, ok := transition.Updates[column]; ok {
			if err := validateJSONUpdate(column, value); err != nil {
				return model.Command{}, err
			}
		}
	}

	var transitioned model.Command
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var current model.Command
		if err := tx.Where("command_id = ?", transition.CommandID).First(&current).Error; err != nil {
			return err
		}
		if current.Version != transition.ExpectedVersion || !containsStatus(transition.FromStatuses, current.Status) {
			if current.Status == transition.ToStatus && model.IsTerminalCommandStatus(current.Status) {
				transitioned = current
				return nil
			}
			return ErrCommandTransitionConflict
		}
		if !model.CanTransitionCommand(current.Status, transition.ToStatus) {
			return fmt.Errorf("%w: %s -> %s", ErrInvalidCommandTransition, current.Status, transition.ToStatus)
		}

		updates := make(map[string]any, len(transition.Updates)+3)
		for column, value := range transition.Updates {
			switch column {
			case "command_id", "status", "version", "created_at", "updated_at":
				return fmt.Errorf("%w: transition update cannot set %s", ErrInvalidCommandTransition, column)
			}
			updates[column] = value
		}
		updates["status"] = transition.ToStatus
		updates["version"] = gorm.Expr("version + ?", 1)
		updates["updated_at"] = time.Now()

		result := tx.Model(new(model.Command)).
			Where("command_id = ? AND status IN ? AND version = ?", transition.CommandID, transition.FromStatuses, transition.ExpectedVersion).
			Updates(updates)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return ErrCommandTransitionConflict
		}

		eventType := transition.EventType
		if eventType == "" {
			eventType = "STATUS_TRANSITION"
		}
		if err := tx.Create(&model.CommandEvent{
			CommandID:   transition.CommandID,
			EventType:   eventType,
			FromStatus:  current.Status,
			ToStatus:    transition.ToStatus,
			Stage:       transition.Stage,
			Message:     transition.Message,
			PayloadJSON: model.LongTextJSON(transition.PayloadJSON),
			CreatedAt:   time.Now(),
		}).Error; err != nil {
			return err
		}
		return tx.Where("command_id = ?", transition.CommandID).First(&transitioned).Error
	})
	if errors.Is(err, ErrCommandTransitionConflict) && model.IsTerminalCommandStatus(transition.ToStatus) {
		var current model.Command
		if readErr := s.db.WithContext(ctx).Where("command_id = ?", transition.CommandID).First(&current).Error; readErr == nil && current.Status == transition.ToStatus {
			return current, nil
		}
	}
	return transitioned, err
}

func (s *CommandStore) AppendEvent(ctx context.Context, event *model.CommandEvent) error {
	if event == nil {
		return errors.New("command event is required")
	}
	if err := validateJSONDocument("event payload", event.PayloadJSON); err != nil {
		return err
	}
	return s.db.WithContext(ctx).Create(event).Error
}

func (s *CommandStore) SaveXML(ctx context.Context, record *model.CommandXML) error {
	if record == nil {
		return errors.New("command XML is required")
	}
	copyRecord := *record
	if copyRecord.CommandID == "" {
		if copyRecord.CWMPID == "" {
			return ErrUncorrelatedCommandXML
		}
		var command model.Command
		err := s.db.WithContext(ctx).
			Select("command_id").
			Where("cwmp_id = ?", copyRecord.CWMPID).
			Order("created_at DESC").
			First(&command).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrUncorrelatedCommandXML
		}
		if err != nil {
			return err
		}
		copyRecord.CommandID = command.CommandID
	}
	sanitized, err := redact.CWMPXML(append([]byte(nil), record.Payload...))
	if err != nil {
		return ErrInvalidCommandXML
	}
	copyRecord.Payload = sanitized
	return s.db.WithContext(ctx).Create(&copyRecord).Error
}

func (s *CommandStore) HeadForDevice(ctx context.Context, deviceID uint) (model.Command, error) {
	var command model.Command
	err := s.db.WithContext(ctx).
		Where("device_id = ? AND status IN ?", deviceID, model.NonTerminalCommandStatuses()).
		Order("created_at ASC").
		Order("command_id ASC").
		First(&command).Error
	return command, err
}

func (s *CommandStore) List(ctx context.Context, filter CommandListFilter) ([]model.Command, int64, error) {
	query := s.db.WithContext(ctx).Model(new(model.Command))
	if filter.DeviceID != 0 {
		query = query.Where("device_id = ?", filter.DeviceID)
	}
	if filter.DeviceSerial != "" {
		query = query.Where("device_key LIKE ?", "%"+filter.DeviceSerial+"%")
	}
	if filter.Operation != "" {
		query = query.Where("operation = ?", filter.Operation)
	}
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if filter.CommandID != "" {
		query = query.Where("command_id LIKE ?", "%"+filter.CommandID+"%")
	}
	if filter.CreatedFrom != nil {
		query = query.Where("created_at >= ?", *filter.CreatedFrom)
	}
	if filter.CreatedTo != nil {
		query = query.Where("created_at <= ?", *filter.CreatedTo)
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
	var commands []model.Command
	err := query.Order("created_at DESC").Order("command_id DESC").Offset(offset).Limit(limit).Find(&commands).Error
	return commands, total, err
}

func (s *CommandStore) Detail(ctx context.Context, commandID string) (CommandDetail, error) {
	var detail CommandDetail
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("command_id = ?", commandID).First(&detail.Command).Error; err != nil {
			return err
		}
		if err := tx.Where("command_id = ?", commandID).
			Order("created_at ASC").
			Order("id ASC").
			Find(&detail.Events).Error; err != nil {
			return err
		}
		return tx.Where("command_id = ?", commandID).
			Order("created_at ASC").
			Order("id ASC").
			Find(&detail.XML).Error
	})
	if err == nil && detail.Command.CommandKey != nil {
		params := make(map[string]any)
		if len(detail.Command.ParamsJSON) > 0 && string(detail.Command.ParamsJSON) != "null" {
			if err := json.Unmarshal(detail.Command.ParamsJSON, &params); err != nil {
				return detail, fmt.Errorf("%w: params", ErrInvalidCommandJSON)
			}
		}
		params["commandKey"] = *detail.Command.CommandKey
		merged, marshalErr := json.Marshal(params)
		if marshalErr != nil {
			return detail, fmt.Errorf("encode command detail params: %w", marshalErr)
		}
		detail.Command.ParamsJSON = model.LongTextJSON(merged)
	}
	return detail, err
}

func containsStatus(statuses []string, target string) bool {
	for _, status := range statuses {
		if status == target {
			return true
		}
	}
	return false
}
