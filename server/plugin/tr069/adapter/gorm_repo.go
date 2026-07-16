package adapter

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/ddddddddwp/gva-acs/server/global"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/config"
	tr069Global "github.com/ddddddddwp/gva-acs/server/plugin/tr069/global"
	gormmiddleware "github.com/ddddddddwp/gva-acs/server/plugin/tr069/middleware/gorm_middleware"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/model"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/service"
	"github.com/ddddddddwp/tr069-core-only/pkg/core"
	"go.uber.org/zap"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type GormDeviceRepo struct {
	db       *gorm.DB
	profiles *ConnectionProfileRepository
}

func NewGormDeviceRepo(db *gorm.DB, profiles *ConnectionProfileRepository) *GormDeviceRepo {
	return &GormDeviceRepo{db: db, profiles: profiles}
}

func (r *GormDeviceRepo) database() *gorm.DB {
	if r != nil && r.db != nil {
		return r.db
	}
	return global.GVA_DB
}

func (r *GormDeviceRepo) profileRepository() *ConnectionProfileRepository {
	if r != nil && r.profiles != nil {
		return r.profiles
	}
	return NewConnectionProfileRepository(r.database(), nil)
}

func (r *GormDeviceRepo) UpsertFromInform(ctx context.Context, info *core.InformSummary, ip string) (string, error) {
	db := r.database()
	if db == nil {
		return "", gorm.ErrInvalidDB
	}
	deviceID, ok := deviceIDFromContext(ctx)
	if !ok {
		deviceID = nil
	}
	clientIP, ok := clientIPFromContext(ctx)
	if ok {
		ip = clientIP
	}

	serial := ""
	oui := ""
	productClass := ""
	manufacturer := ""
	if deviceID != nil {
		serial = deviceID.SerialNumber
		oui = deviceID.OUI
		productClass = deviceID.ProductClass
		manufacturer = deviceID.Manufacturer
	}
	if serial == "" && info != nil && info.Params != nil {
		serial = info.Params["Device.DeviceInfo.SerialNumber"]
	}
	if serial == "" {
		return "", nil
	}

	device := model.Device{
		SerialNumber:     serial,
		OUI:              oui,
		ProductClass:     productClass,
		Manufacturer:     manufacturer,
		IP:               ip,
		LastInform:       time.Now(),
		ConnectionReqURL: "",
		SoftwareVer:      "",
		HardwareVer:      "",
		SpecVer:          "",
	}

	if info != nil && info.Params != nil {
		if v := info.Params["Device.DeviceInfo.SoftwareVersion"]; v != "" {
			device.SoftwareVer = v
		}
		if v := info.Params["Device.DeviceInfo.HardwareVersion"]; v != "" {
			device.HardwareVer = v
		}
		if v := info.Params["Device.ManagementServer.ConnectionRequestURL"]; v != "" {
			device.ConnectionReqURL = v
		}
		if v := info.Params["Device.RootDataModelVersion"]; v != "" {
			device.SpecVer = v
		}
	}

	updateColumns := []string{
		"oui",
		"product_class",
		"manufacturer",
		"software_ver",
		"hardware_ver",
		"spec_ver",
		"ip",
		"last_inform",
	}
	if device.ConnectionReqURL != "" {
		updateColumns = append(updateColumns, "connection_req_url")
	}
	err := db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "serial_number"}},
		DoUpdates: clause.AssignmentColumns(updateColumns),
	}).Create(&device).Error
	if err != nil {
		return "", err
	}

	// Force restore if soft-deleted
	if err := db.Unscoped().Model(&model.Device{}).Where("serial_number = ?", serial).Update("deleted_at", nil).Error; err != nil {
		global.GVA_LOG.Warn("failed to restore soft-deleted device", zap.String("serial", serial), zap.Error(err))
	}

	// Re-query the device to get the ID, but handle deleted_at carefully
	// Since we just upserted it, it should exist. However, if it was soft-deleted, we might need to Unscoped() to find it
	// But Upsert Create(&device) should have revived it if we used Save or handle deleted_at in OnConflict?
	// GORM's Create with OnConflict usually respects soft delete unless handled.
	// Let's ensure we are working with a valid record.
	// Actually, if we use Unscoped() we can find even soft-deleted ones, but we probably want it to be "active" now.
	// If the Create call above succeeded, the record should be there.
	// Issue: If it was soft-deleted, Create might have created a NEW record if unique index allows, OR updated the soft-deleted one.
	// Since we have serial_number unique index (likely), OnConflict updates it.
	// But does it clear deleted_at? Our OnConflict columns didn't include deleted_at.
	// Let's add Unscoped to be safe and check if we need to restore it.

	// 2. Sync parameters from Inform to DataModelValue
	if info != nil && len(info.Params) > 0 {
		var dbDevice model.Device
		// Use Unscoped to find the device even if it was soft-deleted
		if err := db.Unscoped().WithContext(ctx).Select("id, deleted_at").Where("serial_number = ?", serial).First(&dbDevice).Error; err != nil {
			global.GVA_LOG.Warn("failed to find device for parameter sync", zap.String("serial", serial), zap.Error(err))
		} else {
			// If it was deleted, restore it (clear deleted_at)
			if dbDevice.DeletedAt.Valid {
				if err := db.Unscoped().Model(&dbDevice).Update("deleted_at", nil).Error; err != nil {
					global.GVA_LOG.Warn("failed to restore device", zap.Uint("deviceID", dbDevice.ID), zap.Error(err))
				}
			}

			now := time.Now()
			var values []model.DataModelValue
			for k, v := range info.Params {
				// Filter out empty names and object paths
				if k == "" || strings.HasSuffix(k, ".") {
					continue
				}
				if gormmiddleware.DenyByPrefixes(k, tr069Global.DataModelValueDenyPrefixes) {
					continue
				}

				valType := ""
				if info.ParamTypes != nil {
					valType = normalizeValueType(info.ParamTypes[k])
				}
				val := castValue(valType, v)
				b, err := json.Marshal(val)
				if err != nil {
					global.GVA_LOG.Warn("failed to marshal parameter valuekey", zap.String("", k), zap.Error(err))
					continue
				}
				values = append(values, model.DataModelValue{
					DeviceID:        dbDevice.ID,
					Name:            k,
					ValueJSON:       b,
					LastCollectedAt: now,
					ValueType:       valType,
				})
			}

			if len(values) > 0 {
				if outAny, replaced := gormmiddleware.FilterForTable(values, "tr069_datamodel_values", gormmiddleware.RulesForPrefixDeny(
					"tr069_datamodel_values",
					"Name",
					tr069Global.DataModelValueDenyPrefixes,
				)); replaced {
					if out, ok := outAny.([]model.DataModelValue); ok {
						values = out
					}
				}
				// Batch Upsert
				// On conflict (device_id + name), update value_json and last_collected_at
				// Preserve existing ValueType if Inform doesn't carry it
				if err := db.WithContext(ctx).Clauses(clause.OnConflict{
					Columns:   []clause.Column{{Name: "device_id"}, {Name: "name"}},
					DoUpdates: clause.AssignmentColumns([]string{"value_type", "value_json", "last_collected_at", "updated_at"}),
				}).CreateInBatches(values, 100).Error; err != nil {
					global.GVA_LOG.Error("failed to batch upsert data model values", zap.Error(err))
					return "", err
				}
			}
			if _, err := r.profileRepository().Collect(ctx, dbDevice.ID, info.Params); err != nil && global.GVA_LOG != nil {
				global.GVA_LOG.Warn("failed to collect connection profile from Inform", zap.Uint("deviceID", dbDevice.ID), zap.Error(err))
			}
		}
	}

	return serial, nil
}

func (r *GormDeviceRepo) SyncAlarms(ctx context.Context, deviceID string, alarms []core.Alarm) error {
	if len(alarms) == 0 {
		return nil
	}

	var dbDevice model.Device
	if err := global.GVA_DB.WithContext(ctx).Select("id").Where("serial_number = ?", deviceID).First(&dbDevice).Error; err != nil {
		return err
	}

	var toInsert []model.Tr069Alarm
	var toClear []string
	stats := struct {
		Inserted int
		Cleared  int
		Skipped  int
		Errors   int
	}{}

	for _, a := range alarms {
		if a.Source == "QueuedEvent" {
			stats.Skipped++
			continue
		}

		serialNum := a.SerialNumber
		if serialNum == "" {
			serialNum = deviceID
		}

		var status string
		var endTime *time.Time
		now := time.Now()

		switch a.Source {
		case "CurrentAlarm":
			status = "Active"
		case "ExpeditedEvent":
			if a.NotificationType == "ClearedAlarm" {
				status = "Cleared"
				endTime = &now
				toClear = append(toClear, a.AlarmIdentifier)
			} else {
				status = "Active"
			}
		case "HistoryEvent":
			status = "Cleared"
			endTime = &now
		default:
			status = "Active"
		}

		alarm := model.Tr069Alarm{
			DeviceID:              dbDevice.ID,
			SerialNumber:          serialNum,
			OUI:                   a.OUI,
			AlarmIdentifier:       a.AlarmIdentifier,
			Source:                a.Source,
			NotificationType:      a.NotificationType,
			Status:                status,
			EventType:             a.EventType,
			PerceivedSeverity:     a.PerceivedSeverity,
			ProbableCause:         a.ProbableCause,
			SpecificProblem:       a.SpecificProblem,
			AdditionalText:        a.AdditionalText,
			AdditionalInformation: a.AdditionalInfo,
			ManagedObjectInstance: a.ManagedObjectInstance,
			EventTime:             a.EventTime,
			StartTime:             a.EventTime,
			EndTime:               endTime,
			LastChanged:           now,
		}
		toInsert = append(toInsert, alarm)
	}

	if len(toInsert) > 0 {
		if err := r.batchUpsertAlarms(ctx, toInsert); err != nil {
			global.GVA_LOG.Error("batchUpsertAlarms failed", zap.Error(err))
			stats.Errors++
		} else {
			stats.Inserted = len(toInsert)
		}
	}

	if len(toClear) > 0 {
		now := time.Now()
		res := global.GVA_DB.WithContext(ctx).Model(&model.Tr069Alarm{}).
			Where("alarm_identifier IN ?", toClear).
			Where("status = ?", "Active").
			Updates(map[string]interface{}{
				"status":       "Cleared",
				"end_time":     now,
				"last_changed": now,
			})
		if res.Error != nil {
			global.GVA_LOG.Error("clearAlarms failed", zap.Error(res.Error))
			stats.Errors++
		} else {
			stats.Cleared = int(res.RowsAffected)
		}
	}

	global.GVA_LOG.Info("SyncAlarms summary",
		zap.Int("received", len(alarms)),
		zap.Int("inserted", stats.Inserted),
		zap.Int("cleared", stats.Cleared),
		zap.Int("skipped(QueuedEvent)", stats.Skipped),
		zap.Int("errors", stats.Errors),
	)
	return nil
}

func (r *GormDeviceRepo) batchUpsertAlarms(ctx context.Context, alarms []model.Tr069Alarm) error {
	if len(alarms) == 0 {
		return nil
	}

	return global.GVA_DB.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "alarm_identifier"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"notification_type", "status", "event_type", "perceived_severity",
			"probable_cause", "specific_problem", "additional_text",
			"additional_information", "managed_object_instance",
			"event_time", "last_changed",
		}),
	}).CreateInBatches(alarms, 100).Error
}

func extractOUI(deviceID string) string {
	if len(deviceID) >= 6 {
		return deviceID[:6]
	}
	return ""
}

// UpdateOnlineStatus 实现 core.DeviceRepo 接口（仅更新 lastInform）
func (r *GormDeviceRepo) UpdateOnlineStatus(ctx context.Context, deviceID string, online bool, lastSeen time.Time) error {
	return global.GVA_DB.WithContext(ctx).Model(&model.Device{}).
		Where("serial_number = ?", deviceID).
		Update("last_inform", lastSeen).Error
}

type GormCommandRepo struct {
	db    *gorm.DB
	store *service.CommandStore
}

func newGormCommandRepo(db *gorm.DB) *GormCommandRepo {
	return &GormCommandRepo{db: db, store: service.NewCommandStore(db)}
}

func (r *GormCommandRepo) database() *gorm.DB {
	if r != nil && r.db != nil {
		return r.db
	}
	return global.GVA_DB
}

func (r *GormCommandRepo) commandStore() *service.CommandStore {
	if r != nil && r.store != nil {
		return r.store
	}
	return service.NewCommandStore(r.database())
}

func (r *GormCommandRepo) current(ctx context.Context, commandID string) (model.Command, error) {
	var command model.Command
	err := r.database().WithContext(ctx).First(&command, "command_id = ?", commandID).Error
	return command, err
}

func (r *GormCommandRepo) MarkSending(ctx context.Context, commandID, requestID string, sentAt time.Time) error {
	if sentAt.IsZero() {
		sentAt = time.Now()
	}
	current, err := r.current(ctx, commandID)
	if err != nil {
		return err
	}
	deadline := sentAt.Add(config.CurrentRuntime().RPCResponseTimeout)
	_, err = r.commandStore().Transition(ctx, service.CommandTransition{
		CommandID:       commandID,
		FromStatuses:    []string{model.CommandStatusBuilding},
		ToStatus:        model.CommandStatusSent,
		ExpectedVersion: current.Version,
		EventType:       "REQUEST_SENT",
		Stage:           "request",
		Updates: map[string]any{
			"request_id":        requestID,
			"sent_at":           sentAt,
			"phase_deadline_at": deadline,
		},
	})
	return err
}

func (r *GormCommandRepo) MarkSuccess(ctx context.Context, commandID string, finishedAt time.Time) error {
	if finishedAt.IsZero() {
		finishedAt = time.Now()
	}
	current, err := r.current(ctx, commandID)
	if err != nil {
		return err
	}
	_, err = r.commandStore().Transition(ctx, service.CommandTransition{
		CommandID:       commandID,
		FromStatuses:    []string{model.CommandStatusSent, model.CommandStatusWaitingTransfer},
		ToStatus:        model.CommandStatusCompleted,
		ExpectedVersion: current.Version,
		EventType:       "RESPONSE_COMPLETED",
		Stage:           "response",
		Updates: map[string]any{
			"finished_at":       finishedAt,
			"phase_deadline_at": nil,
		},
	})
	return err
}

func (r *GormCommandRepo) MarkFail(ctx context.Context, commandID string, faultCode int, faultString string, finishedAt time.Time) error {
	return r.markFailAtStage(ctx, commandID, faultCode, faultString, finishedAt, "")
}

func (r *GormCommandRepo) MarkFailAtStage(ctx context.Context, commandID string, faultCode int, faultString string, finishedAt time.Time, failureStage core.CommandFailureStage) error {
	return r.markFailAtStage(ctx, commandID, faultCode, faultString, finishedAt, string(failureStage))
}

func (r *GormCommandRepo) markFailAtStage(ctx context.Context, commandID string, faultCode int, faultString string, finishedAt time.Time, explicitStage string) error {
	if finishedAt.IsZero() {
		finishedAt = time.Now()
	}
	current, err := r.current(ctx, commandID)
	if err != nil {
		return err
	}
	stage := explicitStage
	if stage == "" {
		stage = "cwmp.fault"
		if current.Status == model.CommandStatusBuilding {
			stage = "core.build"
		}
	}
	_, err = r.commandStore().Transition(ctx, service.CommandTransition{
		CommandID: commandID,
		FromStatuses: []string{
			model.CommandStatusWaitingDevice,
			model.CommandStatusBuilding,
			model.CommandStatusSent,
			model.CommandStatusWaitingTransfer,
		},
		ToStatus:        model.CommandStatusFailed,
		ExpectedVersion: current.Version,
		EventType:       "COMMAND_FAILED",
		Stage:           stage,
		Message:         faultString,
		Updates: map[string]any{
			"failure_stage":     stage,
			"fault_code":        faultCode,
			"fault_string":      faultString,
			"finished_at":       finishedAt,
			"phase_deadline_at": nil,
		},
	})
	return err
}
