package initialize

import (
	"fmt"

	"github.com/ddddddddwp/gva-acs/server/global"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/config"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

// ReloadConfig loads TR-069 settings into a local value and atomically publishes
// a normalized runtime snapshot. It never mutates the legacy startup config.
func ReloadConfig() {
	if global.GVA_VP == nil {
		zap.L().Error("TR069配置加载失败: Viper 未初始化")
		return
	}

	var next config.TR069Config
	if err := global.GVA_VP.UnmarshalKey("tr069", &next); err != nil {
		err = errors.Wrap(err, "初始化TR069配置文件失败!")
		zap.L().Error(fmt.Sprintf("%+v", err))
		return
	}

	logMissingQueueConfig(next)
	applyLegacyConfigAliases(&next)
	next = config.NormalizeRuntimeConfig(next)
	runtime := config.StoreRuntime(next)
	logRuntimeConfig(runtime.Settings)
}

func logMissingQueueConfig(next config.TR069Config) {
	if next.CommandQueueLockTTL <= 0 {
		zap.L().Error("TR069配置缺失: commandQueueLockTTL，请检查 config.yaml 中 tr069.commandQueueLockTTL 配置项")
	}
	if next.CommandQueueDedupTTL <= 0 {
		zap.L().Error("TR069配置缺失: commandQueueDedupTTL，请检查 config.yaml 中 tr069.commandQueueDedupTTL 配置项")
	}
	if next.CommandQueueMaxScan <= 0 {
		zap.L().Error("TR069配置缺失: commandQueueMaxScan，请检查 config.yaml 中 tr069.commandQueueMaxScan 配置项")
	}
	if next.CommandQueueMaxPendingPerSession <= 0 {
		zap.L().Error("TR069配置缺失: commandQueueMaxPendingPerSession，请检查 config.yaml 中 tr069.commandQueueMaxPendingPerSession 配置项")
	}
	if next.CommandQueueImmediateTTL <= 0 {
		zap.L().Error("TR069配置缺失: commandQueueImmediateTTL，请检查 config.yaml 中 tr069.commandQueueImmediateTTL 配置项")
	}
}

func applyLegacyConfigAliases(next *config.TR069Config) {
	// 旧配置项兼容：Viper 的显式 Get 路径保留原有大小写行为。
	dumpRaw := global.GVA_VP.GetBool("tr069.dumpRaw")
	if !dumpRaw {
		dumpRaw = global.GVA_VP.GetBool("tr069.dumpraw")
	}
	if dumpRaw {
		next.DumpRaw = true
	}

	maxBytes := global.GVA_VP.GetInt("tr069.dumpMaxBytes")
	if maxBytes == 0 {
		maxBytes = global.GVA_VP.GetInt("tr069.dumpmaxbytes")
	}
	if maxBytes > 0 {
		next.DumpMaxBytes = maxBytes
	}

	if global.GVA_VP.IsSet("tr069.dumpRedactAuth") || global.GVA_VP.IsSet("tr069.dumpredactauth") {
		value := global.GVA_VP.GetBool("tr069.dumpRedactAuth")
		if !global.GVA_VP.IsSet("tr069.dumpRedactAuth") {
			value = global.GVA_VP.GetBool("tr069.dumpredactauth")
		}
		next.DumpRedactAuth = value
	}

	if global.GVA_VP.IsSet("tr069.dumpRedactCookie") || global.GVA_VP.IsSet("tr069.dumpredactcookie") {
		value := global.GVA_VP.GetBool("tr069.dumpRedactCookie")
		if !global.GVA_VP.IsSet("tr069.dumpRedactCookie") {
			value = global.GVA_VP.GetBool("tr069.dumpredactcookie")
		}
		next.DumpRedactCookie = value
	}

	infoLogEnable := global.GVA_VP.GetBool("tr069.infoLogEnable")
	if !infoLogEnable {
		infoLogEnable = global.GVA_VP.GetBool("tr069.infologenable")
	}
	if infoLogEnable {
		next.InfoLogEnable = true
	}

	dir := global.GVA_VP.GetString("tr069.infoLogDir")
	if dir == "" {
		dir = global.GVA_VP.GetString("tr069.infologdir")
	}
	if dir != "" {
		next.InfoLogDir = dir
	}
}

func logRuntimeConfig(next config.TR069Config) {
	zap.L().Info("TR069 Config Loaded",
		zap.String("address", next.Address),
		zap.Bool("dumpRaw", next.DumpRaw),
		zap.Int("dumpMaxBytes", next.DumpMaxBytes),
		zap.Bool("dumpRedactAuth", next.DumpRedactAuth),
		zap.Bool("dumpRedactCookie", next.DumpRedactCookie),
		zap.Bool("infoLogEnable", next.InfoLogEnable),
		zap.String("infoLogDir", next.InfoLogDir),
		zap.Int("commandQueueLockTTL", next.CommandQueueLockTTL),
		zap.Int("commandQueueDedupTTL", next.CommandQueueDedupTTL),
		zap.Int("commandQueueMaxScan", next.CommandQueueMaxScan),
		zap.Int("commandQueueMaxPendingPerSession", next.CommandQueueMaxPendingPerSession),
		zap.Int("commandQueueImmediateTTL", next.CommandQueueImmediateTTL),
		zap.Int("commandQueueWaitTimeout", next.CommandQueueWaitTimeout),
		zap.Int("rpcResponseTimeout", next.RPCResponseTimeout),
		zap.Int("rebootConfirmTimeout", next.RebootConfirmTimeout),
		zap.Int("transferCompleteTimeout", next.TransferCompleteTimeout),
		zap.Int("rpcXMLRetentionDays", next.RPCXMLRetentionDays),
		zap.Bool("connectionRequestAutoProvisionCredentials", next.ConnectionRequest.AutoProvisionCredentials),
		zap.String("connectionRequestCredentialKeyVersion", next.ConnectionRequest.CredentialKeyVersion),
		zap.Int("connectionRequestTimeout", next.ConnectionRequest.RequestTimeout),
		zap.String("connectionRequestAuthScheme", next.ConnectionRequest.AuthScheme),
		zap.Int("connectionRequestAllowedCIDRCount", len(next.ConnectionRequest.AllowedCIDRs)),
	)
}
