package initialize

import (
	"context"

	"github.com/ddddddddwp/gva-acs/server/global"
	tr069Global "github.com/ddddddddwp/gva-acs/server/plugin/tr069/global"
	gormmiddleware "github.com/ddddddddwp/gva-acs/server/plugin/tr069/middleware/gorm_middleware"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/model"
	"go.uber.org/zap"
)

func Gorm(ctx context.Context) {
	migrateDeviceOUIColumn(ctx)
	err := global.GVA_DB.WithContext(ctx).AutoMigrate(
		new(model.Device),
		new(model.Command),
		new(model.DataModelValue),
		new(model.DeviceRPCMethods),
		new(model.FAPService),
		new(model.Tr069Alarm),
		new(model.SupportTr069Alarm),
	)
	if err != nil {
		global.GVA_LOG.Error("TR069 Plugin AutoMigrate Failed", zap.Error(err))
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
