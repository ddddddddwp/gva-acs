package adapter

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/ddddddddwp/gva-acs/server/global"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/model"
	"github.com/ddddddddwp/tr069-core-only/pkg/core"
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

	// 2. Sync parameters from Inform to DataModelValue
	if info != nil && len(info.Params) > 0 {
		var dbDevice model.Device
		if err := global.GVA_DB.WithContext(ctx).Select("id").Where("serial_number = ?", serial).First(&dbDevice).Error; err == nil {
			now := time.Now()
			var values []model.DataModelValue
			for k, v := range info.Params {
				// Filter out empty names and object paths
				if k == "" || strings.HasSuffix(k, ".") {
					continue
				}

				b, _ := json.Marshal(v)
				values = append(values, model.DataModelValue{
					DeviceID:        dbDevice.ID,
					Name:            k,
					ValueJSON:       b,
					LastCollectedAt: now,
					// ValueType is unknown here, leave it empty (or preserve existing on update)
				})
			}

			if len(values) > 0 {
				// Batch Upsert
				// On conflict (device_id + name), update value_json and last_collected_at
				// Preserve existing ValueType if present
				_ = global.GVA_DB.WithContext(ctx).Clauses(clause.OnConflict{
					Columns:   []clause.Column{{Name: "device_id"}, {Name: "name"}},
					DoUpdates: clause.AssignmentColumns([]string{"value_json", "last_collected_at", "updated_at"}),
				}).CreateInBatches(values, 100).Error
			}
		}
	}

	return serial, nil
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
