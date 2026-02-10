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
		// Continue to manual override even if UnmarshalKey fails partially
	}

	// Manual overrides to handle case sensitivity issues with Viper/Mapstructure
	dumpRaw := global.GVA_VP.GetBool("tr069.dumpRaw")
	if !dumpRaw {
		dumpRaw = global.GVA_VP.GetBool("tr069.dumpraw")
	}
	if dumpRaw {
		tr069Global.GlobalConfig.DumpRaw = true
	}

	maxBytes := global.GVA_VP.GetInt("tr069.dumpMaxBytes")
	if maxBytes == 0 {
		maxBytes = global.GVA_VP.GetInt("tr069.dumpmaxbytes")
	}
	if maxBytes > 0 {
		tr069Global.GlobalConfig.DumpMaxBytes = maxBytes
	}

	if global.GVA_VP.IsSet("tr069.dumpRedactAuth") || global.GVA_VP.IsSet("tr069.dumpredactauth") {
		v := global.GVA_VP.GetBool("tr069.dumpRedactAuth")
		if !global.GVA_VP.IsSet("tr069.dumpRedactAuth") {
			v = global.GVA_VP.GetBool("tr069.dumpredactauth")
		}
		tr069Global.GlobalConfig.DumpRedactAuth = v
	}
	if global.GVA_VP.IsSet("tr069.dumpRedactCookie") || global.GVA_VP.IsSet("tr069.dumpredactcookie") {
		v := global.GVA_VP.GetBool("tr069.dumpRedactCookie")
		if !global.GVA_VP.IsSet("tr069.dumpRedactCookie") {
			v = global.GVA_VP.GetBool("tr069.dumpredactcookie")
		}
		tr069Global.GlobalConfig.DumpRedactCookie = v
	}

	zap.L().Info("TR069 Config Loaded",
		zap.String("address", tr069Global.GlobalConfig.Address),
		zap.Bool("dumpRaw", tr069Global.GlobalConfig.DumpRaw),
		zap.Int("dumpMaxBytes", tr069Global.GlobalConfig.DumpMaxBytes),
		zap.Bool("dumpRedactAuth", tr069Global.GlobalConfig.DumpRedactAuth),
		zap.Bool("dumpRedactCookie", tr069Global.GlobalConfig.DumpRedactCookie),
		zap.Any("vp.tr069.dumpRaw", global.GVA_VP.Get("tr069.dumpRaw")),
		zap.Any("vp.tr069.dumpraw", global.GVA_VP.Get("tr069.dumpraw")),
	)
}
