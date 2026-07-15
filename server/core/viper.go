package core

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	serverConfig "github.com/ddddddddwp/gva-acs/server/config"
	"github.com/ddddddddwp/gva-acs/server/core/internal"
	"github.com/ddddddddwp/gva-acs/server/global"
	"github.com/ddddddddwp/gva-acs/server/utils"
	"github.com/fsnotify/fsnotify"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

// Viper 配置
func Viper() *viper.Viper {
	config := getConfigPath()

	v := viper.New()
	v.SetConfigFile(config)
	v.SetConfigType("yaml")
	err := v.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("fatal error config file: %w", err))
	}
	v.WatchConfig()

	v.OnConfigChange(func(e fsnotify.Event) {
		fmt.Println("config file changed:", e.Name)
		if reloadErr := applyConfigChange(v); reloadErr != nil {
			fmt.Println("config reload failed:", reloadErr)
		}
	})
	if err = v.Unmarshal(&global.GVA_CONFIG); err != nil {
		panic(fmt.Errorf("fatal error unmarshal config: %w", err))
	}

	// root 适配性 根据root位置去找到对应迁移位置,保证root路径有效
	global.GVA_CONFIG.AutoCode.Root, _ = filepath.Abs("..")
	return v
}

func applyConfigChange(v *viper.Viper) error {
	if err := v.ReadInConfig(); err != nil {
		return fmt.Errorf("read changed config file: %w", err)
	}

	next, err := deepCopyServerConfig(global.GVA_CONFIG)
	if err != nil {
		return fmt.Errorf("copy current config before reload: %w", err)
	}
	if err := v.Unmarshal(&next); err != nil {
		return fmt.Errorf("unmarshal changed config file: %w", err)
	}
	global.GVA_CONFIG = next
	utils.GlobalSystemEvents.TriggerConfigChange()
	return nil
}

func deepCopyServerConfig(in serverConfig.Server) (serverConfig.Server, error) {
	serialized, err := json.Marshal(in)
	if err != nil {
		return serverConfig.Server{}, err
	}
	var out serverConfig.Server
	if err := json.Unmarshal(serialized, &out); err != nil {
		return serverConfig.Server{}, err
	}
	return out, nil
}

// getConfigPath 获取配置文件路径, 优先级: 命令行 > 环境变量 > 默认值
func getConfigPath() (config string) {
	// `-c` flag parse
	flag.StringVar(&config, "c", "", "choose config file.")
	flag.Parse()
	if config != "" { // 命令行参数不为空 将值赋值于config
		fmt.Printf("您正在使用命令行的 '-c' 参数传递的值, config 的路径为 %s\n", config)
		return
	}
	if env := os.Getenv(internal.ConfigEnv); env != "" { // 判断环境变量 GVA_CONFIG
		config = env
		fmt.Printf("您正在使用 %s 环境变量, config 的路径为 %s\n", internal.ConfigEnv, config)
		return
	}

	switch gin.Mode() { // 根据 gin 模式文件名
	case gin.DebugMode:
		config = internal.ConfigDebugFile
	case gin.ReleaseMode:
		config = internal.ConfigReleaseFile
	case gin.TestMode:
		config = internal.ConfigTestFile
	}
	fmt.Printf("您正在使用 gin 的 %s 模式运行, config 的路径为 %s\n", gin.Mode(), config)

	_, err := os.Stat(config)
	if err != nil || os.IsNotExist(err) {
		config = internal.ConfigDefaultFile
		fmt.Printf("配置文件路径不存在, 使用默认配置文件路径: %s\n", config)
	}

	return
}
