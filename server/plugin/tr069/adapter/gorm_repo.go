package adapter

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/ddddddddwp/gva-acs/server/global"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/model"
	"github.com/ddddddddwp/tr069-core-only/pkg/core"
	"go.uber.org/zap"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type GormDeviceRepo struct{}

func (r *GormDeviceRepo) UpsertFromInform(ctx context.Context, info *core.InformSummary, ip string) (string, error) {
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
		Status:           "online",
		LastOnline:       time.Now(),
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

	err := global.GVA_DB.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "serial_number"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"oui",
			"product_class",
			"manufacturer",
			"software_ver",
			"hardware_ver",
			"spec_ver",
			"ip",
			"connection_req_url",
			"last_inform",
			"last_online",
			"status",
		}),
	}).Create(&device).Error
	if err != nil {
		return "", err
	}

	// Force restore if soft-deleted
	global.GVA_DB.Unscoped().Model(&model.Device{}).Where("serial_number = ?", serial).Update("deleted_at", nil)

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
		if err := global.GVA_DB.Unscoped().WithContext(ctx).Select("id, deleted_at").Where("serial_number = ?", serial).First(&dbDevice).Error; err == nil {
			// If it was deleted, restore it (clear deleted_at)
			if dbDevice.DeletedAt.Valid {
				global.GVA_DB.Unscoped().Model(&dbDevice).Update("deleted_at", nil)
			}

			now := time.Now()
			var values []model.DataModelValue
			for k, v := range info.Params {
				// Filter out empty names and object paths
				if k == "" || strings.HasSuffix(k, ".") {
					continue
				}

				valType := ""
				if info.ParamTypes != nil {
					valType = normalizeValueType(info.ParamTypes[k])
				}
				val := castValue(valType, v)
				b, _ := json.Marshal(val)
				values = append(values, model.DataModelValue{
					DeviceID:        dbDevice.ID,
					Name:            k,
					ValueJSON:       b,
					LastCollectedAt: now,
					ValueType:       valType,
				})
			}

			if len(values) > 0 {
				// Batch Upsert
				// On conflict (device_id + name), update value_json and last_collected_at
				// Preserve existing ValueType if Inform doesn't carry it
				_ = global.GVA_DB.WithContext(ctx).Clauses(clause.OnConflict{
					Columns: []clause.Column{{Name: "device_id"}, {Name: "name"}},
					DoUpdates: clause.Assignments(map[string]interface{}{
						"value_type":        gorm.Expr("COALESCE(NULLIF(VALUES(value_type),''), value_type)"),
						"value_json":        gorm.Expr("VALUES(value_json)"),
						"last_collected_at": gorm.Expr("VALUES(last_collected_at)"),
						"updated_at":        gorm.Expr("NOW()"),
					}),
				}).CreateInBatches(values, 100).Error
			}
		}
	}

	return serial, nil
}

func (r *GormDeviceRepo) SyncAlarms(ctx context.Context, deviceID string, alarms []core.Alarm) error {
	if len(alarms) == 0 {
		return nil
	}

	recv := len(alarms)
	created := 0
	updated := 0
	ignored := 0

	// Find the device numeric ID
	var dbDevice model.Device
	if err := global.GVA_DB.WithContext(ctx).Select("id").Where("serial_number = ?", deviceID).First(&dbDevice).Error; err != nil {
		// If device not found (which is weird as Upsert happened), we can't link
		// But Upsert returns serial, so we query by serial
		return err
	}

	for _, a := range alarms {
		// Append event history (no dedup)
		evt := model.Tr069AlarmEvent{
			DeviceID:         dbDevice.ID,
			SerialNumber:     deviceID,
			AlarmIdentifier:  a.AlarmIdentifier,
			NotificationType: a.NotificationType,
			Severity:         a.PerceivedSeverity,
			SpecificProblem:  a.SpecificProblem,
			ProbableCause:    a.ProbableCause,
			EventType:        a.EventType,
			AdditionalText:   a.AdditionalText,
			AddInfo:          a.AdditionalInfo,
			EventTime:        a.EventTime,
		}
		_ = global.GVA_DB.Create(&evt).Error

		// Handle missing NotificationType for CurrentAlarm (treat as Active)
		notifType := a.NotificationType
		if notifType == "" && a.Source == "CurrentAlarm" {
			notifType = "NewAlarm"
		}

		if notifType == "NewAlarm" || notifType == "ChangedAlarm" {
			// Create if not exists (Active)
			var count int64
			global.GVA_DB.Model(&model.Tr069Alarm{}).Where("alarm_identifier = ? AND status = ?", a.AlarmIdentifier, "Active").Count(&count)
			if count == 0 {
				m := model.Tr069Alarm{
					DeviceID:         dbDevice.ID,
					SerialNumber:     deviceID,
					AlarmIdentifier:  a.AlarmIdentifier,
					NotificationType: a.NotificationType,
					Status:           "Active",
					Severity:         a.PerceivedSeverity,
					SpecificProblem:  a.SpecificProblem,
					ProbableCause:    a.ProbableCause,
					EventType:        a.EventType,
					AdditionalText:   a.AdditionalText,
					AddInfo:          a.AdditionalInfo,
					StartTime:        a.EventTime,
				}
				if err := global.GVA_DB.Create(&m).Error; err == nil {
					created++
				} else {
					ignored++
				}
			} else {
				ignored++
			}
		} else if notifType == "ClearedAlarm" {
			// Update Active to Cleared
			endTime := a.EventTime
			res := global.GVA_DB.Model(&model.Tr069Alarm{}).
				Where("alarm_identifier = ? AND status = ?", a.AlarmIdentifier, "Active").
				Updates(map[string]interface{}{
					"status":   "Cleared",
					"end_time": endTime,
				})
			if res.Error == nil && res.RowsAffected > 0 {
				updated++
			} else {
				ignored++
			}
		}
	}
	// Log summary
	global.GVA_LOG.Info("SyncAlarms summary",
		zap.Int("received", recv),
		zap.Int("created", created),
		zap.Int("updated", updated),
		zap.Int("ignored", ignored),
	)
	return nil
}

func (r *GormDeviceRepo) UpdateOnlineStatus(ctx context.Context, deviceID string, status bool, lastSeen time.Time) error {
	statusStr := "offline"
	if status {
		statusStr = "online"
	}

	return global.GVA_DB.WithContext(ctx).Model(&model.Device{}).
		Where("serial_number = ?", deviceID).
		Updates(map[string]interface{}{
			"status":      statusStr,
			"last_online": lastSeen,
		}).Error
}

type GormCommandRepo struct{}

func (r *GormCommandRepo) MarkSending(ctx context.Context, commandID, requestID string, sentAt time.Time) error {
	m := model.Command{
		CommandID: commandID,
		Status:    "SENDING",
		RequestID: requestID,
	}
	if !sentAt.IsZero() {
		t := sentAt
		m.SentAt = &t
	}
	return global.GVA_DB.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "command_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"status",
			"request_id",
			"sent_at",
			"updated_at",
		}),
	}).Create(&m).Error
}

func (r *GormCommandRepo) MarkSuccess(ctx context.Context, commandID string, finishedAt time.Time) error {
	m := model.Command{
		CommandID: commandID,
		Status:    "SUCCESS",
	}
	if !finishedAt.IsZero() {
		t := finishedAt
		m.FinishedAt = &t
	}
	return global.GVA_DB.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "command_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"status",
			"finished_at",
			"updated_at",
		}),
	}).Create(&m).Error
}

func (r *GormCommandRepo) MarkFail(ctx context.Context, commandID string, faultCode int, faultString string, finishedAt time.Time) error {
	m := model.Command{
		CommandID:   commandID,
		Status:      "FAIL",
		FaultCode:   faultCode,
		FaultString: faultString,
	}
	if !finishedAt.IsZero() {
		t := finishedAt
		m.FinishedAt = &t
	}
	return global.GVA_DB.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "command_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"status",
			"fault_code",
			"fault_string",
			"finished_at",
			"updated_at",
		}),
	}).Create(&m).Error
}
