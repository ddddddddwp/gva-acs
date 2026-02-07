package initialize

import (
	"context"
	"github.com/ddddddddwp/gva-acs/server/global"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/model"
	"go.uber.org/zap"
)

func Gorm(ctx context.Context) {
	err := global.GVA_DB.WithContext(ctx).AutoMigrate(
		new(model.Device),
	)
	if err != nil {
		global.GVA_LOG.Error("TR069 Plugin AutoMigrate Failed", zap.Error(err))
	}
}
