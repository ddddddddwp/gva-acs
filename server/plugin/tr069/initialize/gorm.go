package initialize

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ddddddddwp/gva-acs/server/global"
	tr069Global "github.com/ddddddddwp/gva-acs/server/plugin/tr069/global"
	gormmiddleware "github.com/ddddddddwp/gva-acs/server/plugin/tr069/middleware/gorm_middleware"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/model"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/redact"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func Gorm(ctx context.Context) error {
	migrateDeviceOUIColumn(ctx)
	if global.GVA_DB == nil {
		return errors.New("TR069 database is not initialized")
	}
	if err := cutoverCommandIdentifierSchema(global.GVA_DB.WithContext(ctx)); err != nil {
		return fmt.Errorf("TR069 command identifier schema cutover: %w", err)
	}
	err := global.GVA_DB.WithContext(ctx).AutoMigrate(
		new(model.Device),
		new(model.Command),
		new(model.CommandEvent),
		new(model.CommandXML),
		new(model.DataModelValue),
		new(model.DeviceRPCMethods),
		new(model.FAPService),
		new(model.Tr069Alarm),
		new(model.SupportTr069Alarm),
		new(model.ConnectionProfile),
		new(tr069MigrationMarker),
	)
	if err != nil {
		return fmt.Errorf("TR069 plugin auto migrate: %w", err)
	}
	if err := migrateConnectionProfiles(ctx, global.GVA_DB); err != nil {
		global.GVA_LOG.Error("TR069 ConnectionProfile Migration Failed", zap.Error(err))
	}
	if err := sanitizeStoredCommandXML(ctx, global.GVA_DB); err != nil {
		global.GVA_LOG.Error("TR069 Command XML Sanitization Failed", zap.Error(err))
	}

	if global.GVA_DB != nil {
		if err := global.GVA_DB.Use(gormmiddleware.New(gormmiddleware.RulesForPrefixDeny(
			"tr069_datamodel_values",
			"Name",
			tr069Global.DataModelValueDenyPrefixes,
		))); err != nil {
			global.GVA_LOG.Error("TR069 DataModelValue IngestFilter Init Failed", zap.Error(err))
		}
	}
	return nil
}

func cutoverCommandIdentifierSchema(db *gorm.DB) error {
	if db == nil {
		return errors.New("TR069 command identifier schema cutover requires a database")
	}
	migrator := db.Migrator()
	hasRequestID := migrator.HasColumn("tr069_commands", "request_id")
	hasCWMPID := migrator.HasColumn("tr069_commands", "cwmp_id")
	if hasRequestID && hasCWMPID {
		return errors.New("tr069_commands contains both request_id and cwmp_id")
	}
	if hasRequestID {
		if err := migrator.RenameColumn("tr069_commands", "request_id", "cwmp_id"); err != nil {
			return fmt.Errorf("rename tr069_commands.request_id to cwmp_id: %w", err)
		}
	}
	if migrator.HasColumn("tr069_command_xmls", "request_id") {
		var err error
		if db.Dialector.Name() == "sqlite" {
			err = db.Exec("ALTER TABLE tr069_command_xmls DROP COLUMN request_id").Error
		} else {
			err = migrator.DropColumn("tr069_command_xmls", "request_id")
		}
		if err != nil {
			return fmt.Errorf("drop tr069_command_xmls.request_id: %w", err)
		}
	}
	return nil
}

const (
	commandXMLRedactionMigration = "connection_request_password_xml_redaction_v1"
	commandXMLSanitizeBatchSize  = 100
)

type tr069MigrationMarker struct {
	Name        string    `gorm:"primaryKey;size:128"`
	CompletedAt time.Time `gorm:"not null"`
}

func (tr069MigrationMarker) TableName() string { return "tr069_migration_markers" }

func sanitizeStoredCommandXML(ctx context.Context, db *gorm.DB) error {
	if db == nil {
		return nil
	}
	if err := db.WithContext(ctx).AutoMigrate(new(tr069MigrationMarker)); err != nil {
		return err
	}
	const parameterName = "Device.ManagementServer.ConnectionRequestPassword"
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var completed int64
		if err := tx.Model(new(tr069MigrationMarker)).Where("name = ?", commandXMLRedactionMigration).Count(&completed).Error; err != nil {
			return err
		}
		if completed > 0 {
			return nil
		}

		var records []model.CommandXML
		candidates, err := commandXMLRedactionCandidates(tx, parameterName)
		if err != nil {
			return err
		}
		if err := candidates.Order("id ASC").FindInBatches(&records, commandXMLSanitizeBatchSize, func(_ *gorm.DB, _ int) error {
			for _, record := range records {
				if !bytes.Contains(record.Payload, []byte(parameterName)) {
					continue
				}
				sanitized, err := redact.CWMPXML(record.Payload)
				if err != nil {
					if err := tx.Delete(new(model.CommandXML), record.ID).Error; err != nil {
						return err
					}
					continue
				}
				if bytes.Equal(sanitized, record.Payload) {
					continue
				}
				if err := tx.Model(new(model.CommandXML)).Where("id = ?", record.ID).Update("payload", sanitized).Error; err != nil {
					return err
				}
			}
			return nil
		}).Error; err != nil {
			return err
		}
		return tx.Create(&tr069MigrationMarker{Name: commandXMLRedactionMigration, CompletedAt: time.Now()}).Error
	})
}

func commandXMLRedactionCandidates(tx *gorm.DB, parameterName string) (*gorm.DB, error) {
	query := tx.Model(new(model.CommandXML)).Select("id", "payload")
	switch tx.Dialector.Name() {
	case "sqlite":
		return query.Where("instr(CAST(payload AS TEXT), ?) > 0", parameterName), nil
	case "mysql":
		return query.Where("LOCATE(?, CONVERT(payload USING utf8mb4)) > 0", parameterName), nil
	case "postgres":
		return query.Where("POSITION(? IN convert_from(payload, 'UTF8')) > 0", parameterName), nil
	case "sqlserver":
		return query.Where("CHARINDEX(?, CONVERT(varchar(max), payload)) > 0", parameterName), nil
	case "oracle":
		return query.Where("DBMS_LOB.INSTR(payload, UTL_RAW.CAST_TO_RAW(?)) > 0", parameterName), nil
	default:
		return nil, fmt.Errorf("unsupported command XML migration dialect %q", tx.Dialector.Name())
	}
}

func migrateConnectionProfiles(ctx context.Context, db *gorm.DB) error {
	if db == nil {
		return nil
	}
	var devices []model.Device
	if err := db.WithContext(ctx).
		Select("id", "connection_req_url").
		Where("connection_req_url <> ?", "").
		Find(&devices).Error; err != nil {
		return err
	}
	if len(devices) == 0 {
		return nil
	}
	now := time.Now()
	profiles := make([]model.ConnectionProfile, 0, len(devices))
	for _, device := range devices {
		profiles = append(profiles, model.ConnectionProfile{
			DeviceID:       device.ID,
			DiscoveredURL:  device.ConnectionReqURL,
			AuthScheme:     "digest",
			ProvisionState: model.ConnectionProfileStateDiscovered,
			CreatedAt:      now,
			UpdatedAt:      now,
		})
	}
	return db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&profiles).Error
}

func migrateDeviceOUIColumn(ctx context.Context) {
	if global.GVA_DB == nil {
		return
	}
	if global.GVA_DB.Dialector == nil || global.GVA_DB.Dialector.Name() != "mysql" {
		return
	}
	m := global.GVA_DB.Migrator()
	if m == nil {
		return
	}
	if m.HasColumn(&model.Device{}, "oui") {
		return
	}
	if !m.HasColumn(&model.Device{}, "o_ui") {
		return
	}
	if err := global.GVA_DB.WithContext(ctx).Exec("ALTER TABLE tr069_devices RENAME COLUMN o_ui TO oui").Error; err == nil {
		return
	}
	_ = global.GVA_DB.WithContext(ctx).Exec("ALTER TABLE tr069_devices CHANGE COLUMN o_ui oui varchar(64)").Error
}
