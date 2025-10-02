package initialize

import (
	"fmt"
	"path/filepath"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-adapter/config"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

// InitializeConfig 初始化配置
func InitializeConfig() (*config.TR069Config, error) {
	// 创建配置实例
	cfg := config.GetDefaultConfig()

	// 设置配置文件路径
	configPath := filepath.Join("plugin", "tr069-adapter", "config.yaml")
	
	// 创建新的viper实例用于插件配置
	v := viper.New()
	v.SetConfigFile(configPath)
	v.SetConfigType("yaml")

	// 尝试读取配置文件
	if err := v.ReadInConfig(); err != nil {
		global.GVA_LOG.Warn("TR069-Adapter插件配置文件读取失败，使用默认配置", zap.Error(err))
		// 如果配置文件不存在，创建默认配置文件
		if err := createDefaultConfigFile(configPath, cfg); err != nil {
			global.GVA_LOG.Error("创建默认配置文件失败", zap.Error(err))
			return cfg, err
		}
	} else {
		// 解析配置到结构体
		if err := v.Unmarshal(cfg); err != nil {
			global.GVA_LOG.Error("TR069-Adapter插件配置解析失败", zap.Error(err))
			return cfg, err
		}
	}

	global.GVA_LOG.Info("TR069-Adapter插件配置初始化完成")
	return cfg, nil
}

// createDefaultConfigFile 创建默认配置文件
func createDefaultConfigFile(configPath string, cfg *config.TR069Config) error {
	v := viper.New()
	
	// 设置默认值
	v.Set("server.port", cfg.Server.Port)
	v.Set("server.readTimeout", cfg.Server.ReadTimeout)
	v.Set("server.writeTimeout", cfg.Server.WriteTimeout)
	v.Set("server.idleTimeout", cfg.Server.IdleTimeout)
	
	v.Set("database.type", cfg.Database.Type)
	v.Set("database.host", cfg.Database.Host)
	v.Set("database.port", cfg.Database.Port)
	v.Set("database.database", cfg.Database.Database)
	v.Set("database.username", cfg.Database.Username)
	v.Set("database.password", cfg.Database.Password)
	
	v.Set("redis.host", cfg.Redis.Host)
	v.Set("redis.port", cfg.Redis.Port)
	v.Set("redis.password", cfg.Redis.Password)
	v.Set("redis.db", cfg.Redis.DB)
	
	v.Set("log.level", cfg.Log.Level)
	v.Set("log.format", cfg.Log.Format)
	v.Set("log.output", cfg.Log.Output)
	v.Set("log.maxSize", cfg.Log.MaxSize)
	v.Set("log.maxAge", cfg.Log.MaxAge)
	v.Set("log.maxBackups", cfg.Log.MaxBackups)
	v.Set("log.compress", cfg.Log.Compress)
	
	v.Set("cwmp.connectionRequestURL", cfg.CWMP.ConnectionRequestURL)
	v.Set("cwmp.connectionRequestUsername", cfg.CWMP.ConnectionRequestUsername)
	v.Set("cwmp.connectionRequestPassword", cfg.CWMP.ConnectionRequestPassword)
	v.Set("cwmp.periodicInformInterval", cfg.CWMP.PeriodicInformInterval)
	v.Set("cwmp.parameterKey", cfg.CWMP.ParameterKey)

	// 写入配置文件
	v.SetConfigFile(configPath)
	if err := v.WriteConfig(); err != nil {
		return fmt.Errorf("写入配置文件失败: %w", err)
	}

	return nil
}