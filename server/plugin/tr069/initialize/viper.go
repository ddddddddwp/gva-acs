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
	}

	// ==================== 队列配置校验 ====================
	// 如果配置文件没有设置，则报错
	if tr069Global.GlobalConfig.CommandQueueLockTTL <= 0 {
		zap.L().Error("TR069配置缺失: commandQueueLockTTL，请检查 config.yaml 中 tr069.commandQueueLockTTL 配置项")
		tr069Global.GlobalConfig.CommandQueueLockTTL = 30 // 保守默认值
	}
	if tr069Global.GlobalConfig.CommandQueueDedupTTL <= 0 {
		zap.L().Error("TR069配置缺失: commandQueueDedupTTL，请检查 config.yaml 中 tr069.commandQueueDedupTTL 配置项")
		tr069Global.GlobalConfig.CommandQueueDedupTTL = 86400
	}
	if tr069Global.GlobalConfig.CommandQueueMaxScan <= 0 {
		zap.L().Error("TR069配置缺失: commandQueueMaxScan，请检查 config.yaml 中 tr069.commandQueueMaxScan 配置项")
		tr069Global.GlobalConfig.CommandQueueMaxScan = 10
	}
	if tr069Global.GlobalConfig.CommandQueueMaxPendingPerSession <= 0 {
		zap.L().Error("TR069配置缺失: commandQueueMaxPendingPerSession，请检查 config.yaml 中 tr069.commandQueueMaxPendingPerSession 配置项")
		tr069Global.GlobalConfig.CommandQueueMaxPendingPerSession = 5
	}
	if tr069Global.GlobalConfig.CommandQueueImmediateTTL <= 0 {
		zap.L().Error("TR069配置缺失: commandQueueImmediateTTL，请检查 config.yaml 中 tr069.commandQueueImmediateTTL 配置项")
		tr069Global.GlobalConfig.CommandQueueImmediateTTL = 1800
	}

	// ==================== 旧配置项兼容处理 ====================
	// dumpRaw 兼容大小写
	dumpRaw := global.GVA_VP.GetBool("tr069.dumpRaw")
	if !dumpRaw {
		dumpRaw = global.GVA_VP.GetBool("tr069.dumpraw")
	}
	if dumpRaw {
		tr069Global.GlobalConfig.DumpRaw = true
	}

	// dumpMaxBytes 兼容大小写
	maxBytes := global.GVA_VP.GetInt("tr069.dumpMaxBytes")
	if maxBytes == 0 {
		maxBytes = global.GVA_VP.GetInt("tr069.dumpmaxbytes")
	}
	if maxBytes > 0 {
		tr069Global.GlobalConfig.DumpMaxBytes = maxBytes
	}

	// dumpRedactAuth 兼容大小写
	if global.GVA_VP.IsSet("tr069.dumpRedactAuth") || global.GVA_VP.IsSet("tr069.dumpredactauth") {
		v := global.GVA_VP.GetBool("tr069.dumpRedactAuth")
		if !global.GVA_VP.IsSet("tr069.dumpRedactAuth") {
			v = global.GVA_VP.GetBool("tr069.dumpredactauth")
		}
		tr069Global.GlobalConfig.DumpRedactAuth = v
	}

	// dumpRedactCookie 兼容大小写
	if global.GVA_VP.IsSet("tr069.dumpRedactCookie") || global.GVA_VP.IsSet("tr069.dumpredactcookie") {
		v := global.GVA_VP.GetBool("tr069.dumpRedactCookie")
		if !global.GVA_VP.IsSet("tr069.dumpRedactCookie") {
			v = global.GVA_VP.GetBool("tr069.dumpredactcookie")
		}
		tr069Global.GlobalConfig.DumpRedactCookie = v
	}

	// infoLogEnable 兼容大小写
	infoLogEnable := global.GVA_VP.GetBool("tr069.infoLogEnable")
	if !infoLogEnable {
		infoLogEnable = global.GVA_VP.GetBool("tr069.infologenable")
	}
	if infoLogEnable {
		tr069Global.GlobalConfig.InfoLogEnable = true
	}

	// infoLogDir 兼容大小写
	dir := global.GVA_VP.GetString("tr069.infoLogDir")
	if dir == "" {
		dir = global.GVA_VP.GetString("tr069.infologdir")
	}
	if dir != "" {
		tr069Global.GlobalConfig.InfoLogDir = dir
	}
	if tr069Global.GlobalConfig.InfoLogDir == "" {
		tr069Global.GlobalConfig.InfoLogDir = "./log"
	}

	zap.L().Info("TR069 Config Loaded",
		zap.String("address", tr069Global.GlobalConfig.Address),
		zap.Bool("dumpRaw", tr069Global.GlobalConfig.DumpRaw),
		zap.Int("dumpMaxBytes", tr069Global.GlobalConfig.DumpMaxBytes),
		zap.Bool("dumpRedactAuth", tr069Global.GlobalConfig.DumpRedactAuth),
		zap.Bool("dumpRedactCookie", tr069Global.GlobalConfig.DumpRedactCookie),
		zap.Bool("infoLogEnable", tr069Global.GlobalConfig.InfoLogEnable),
		zap.String("infoLogDir", tr069Global.GlobalConfig.InfoLogDir),
		zap.Int("commandQueueLockTTL", tr069Global.GlobalConfig.CommandQueueLockTTL),
		zap.Int("commandQueueDedupTTL", tr069Global.GlobalConfig.CommandQueueDedupTTL),
		zap.Int("commandQueueMaxScan", tr069Global.GlobalConfig.CommandQueueMaxScan),
		zap.Int("commandQueueMaxPendingPerSession", tr069Global.GlobalConfig.CommandQueueMaxPendingPerSession),
		zap.Int("commandQueueImmediateTTL", tr069Global.GlobalConfig.CommandQueueImmediateTTL),
	)
}
