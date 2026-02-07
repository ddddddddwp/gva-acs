package initialize

import (
	"fmt"
	"github.com/ddddddddwp/gva-acs/server/global"
	"github.com/ddddddddwp/gva-acs/server/plugin/announcement/plugin"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

func Viper() {
	err := global.GVA_VP.UnmarshalKey("announcement", &plugin.Config)
	if err != nil {
		err = errors.Wrap(err, "初始化配置文件失败!")
		zap.L().Error(fmt.Sprintf("%+v", err))
	}
}
