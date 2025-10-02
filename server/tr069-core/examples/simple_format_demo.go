package main

import (
	"fmt"

	"github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-core/interfaces"
	"github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-core/pkg/logger"
)

func main() {
	fmt.Println("=== 用户要求的日志格式演示 ===")
	fmt.Println("格式: 2025-09-25T08:02:35.464Z\tINFO\tinverters/invertersallinone.go:82\t消息内容")
	fmt.Println()

	// 创建紧凑格式的日志记录器
	config := logger.LogStorageConfig{
		FilePath:      "/tmp/tr069_user_format.log",
		MaxSize:       10,
		MaxBackups:    3,
		MaxAge:        7,
		Compress:      false,
		ConsoleOutput: true,
		Level:         interfaces.LogLevelInfo,
		Format:        interfaces.LogFormatCompact, // 使用紧凑格式
	}

	log := logger.NewFileLogger(config)

	// 演示各种日志级别
	log.Info("TR069连接建立成功",
		interfaces.LogField{Key: "device_id", Value: "CPE-001"})

	log.Warn("参数值验证失败",
		interfaces.LogField{Key: "parameter", Value: "Device.WiFi.Radio.1.Channel"})

	log.Error("RPC调用超时",
		interfaces.LogField{Key: "method", Value: "GetParameterValues"})

	fmt.Println()
	fmt.Println("日志已保存到: /tmp/tr069_user_format.log")
	fmt.Println("可以使用以下命令查看:")
	fmt.Println("cat /tmp/tr069_user_format.log")
}
