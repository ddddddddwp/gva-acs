package initialize

import (
	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-adapter/config"
	tr069global "github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-adapter/global"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"os"
)

// InitializeViper 初始化配置
func InitializeViper() {
	// 设置默认配置
	tr069global.TR069Config = &config.TR069AdapterConfig{
		Enabled:     true,
		ServerPort:  7547,
		LogLevel:    "info",
		MaxDevices:  1000,
		EnableCache: true,
		CacheTTL:    3600,
	}

	// 检查配置文件是否存在
	configFile := "plugin/tr069-adapter/config.yaml"
	_, err := os.Stat(configFile)
	if os.IsNotExist(err) {
		global.GVA_LOG.Info("TR069-Adapter插件配置文件不存在，使用默认配置")
		return
	}

	// 读取配置文件
	v := viper.New()
	v.SetConfigFile(configFile)
	err = v.ReadInConfig()
	if err != nil {
		global.GVA_LOG.Error("TR069-Adapter插件配置文件读取失败", zap.Error(err))
		return
	}

	// 解析配置
	if err := v.Unmarshal(tr069global.TR069Config); err != nil {
		global.GVA_LOG.Error("TR069-Adapter插件配置解析失败", zap.Error(err))
		return
	}

	global.GVA_LOG.Info("TR069-Adapter插件配置加载成功")
}