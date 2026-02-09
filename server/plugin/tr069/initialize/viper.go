package initialize

import (
	"fmt"
	"github.com/ddddddddwp/gva-acs/server/global"
	tr069Global "github.com/ddddddddwp/gva-acs/server/plugin/tr069/global"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

func Viper() {
	err := global.GVA_VP.UnmarshalKey("tr069", tr069Global.GlobalConfig)
	if err != nil {
		err = errors.Wrap(err, "初始化TR069配置文件失败!")
		zap.L().Error(fmt.Sprintf("%+v", err))
	} else {
		zap.L().Info("TR069 Config Loaded", zap.String("address", tr069Global.GlobalConfig.Address))
	}
}
