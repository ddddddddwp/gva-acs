// TR069 Core日志系统使用示例
package main

import (
	"fmt"
	"os"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-core/interfaces"
	"github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-core/pkg/logger"
)

func main() {
	fmt.Println("=== TR069 Core日志系统示例 ===")

	// 1. 初始化Core日志系统
	fmt.Println("1. 初始化Core日志系统...")
	if err := logger.InitCoreLogger(); err != nil {
		fmt.Printf("初始化日志系统失败: %v\n", err)
		// 如果无法创建/var/log目录（权限问题），使用临时目录
		fmt.Println("尝试使用临时目录...")
		tempConfig := logger.LogStorageConfig{
			FilePath:      "/tmp/tr069.log",
			MaxSize:       50,
			MaxBackups:    5,
			MaxAge:        7,
			Compress:      true,
			ConsoleOutput: true, // 示例中同时输出到控制台
			Level:         interfaces.LogLevelInfo,
			Format:        interfaces.LogFormatJSON,
		}
		tempLogger := logger.NewFileLogger(tempConfig)
		logger.SetGlobalLogger(tempLogger)
		fmt.Println("使用临时日志文件: /tmp/tr069.log")
	} else {
		fmt.Println("Core日志系统初始化成功，日志文件: /var/log/tr069.log")
	}

	// 2. 使用Core专用日志函数
	fmt.Println("\n2. 使用Core专用日志函数...")
	logger.LogCoreInfo("TR069 Core服务启动",
		interfaces.LogField{Key: "version", Value: "1.0.0"},
		interfaces.LogField{Key: "pid", Value: os.Getpid()},
	)

	logger.LogCoreDebug("解析器初始化",
		interfaces.LogField{Key: "parser_type", Value: "xml"},
		interfaces.LogField{Key: "strict_mode", Value: true},
	)

	logger.LogCoreWarn("配置文件未找到，使用默认配置",
		interfaces.LogField{Key: "config_path", Value: "/etc/tr069/config.yaml"},
	)

	// 3. 使用全局日志函数
	fmt.Println("\n3. 使用全局日志函数...")
	logger.Info("处理TR069消息",
		interfaces.LogField{Key: "message_type", Value: "GetParameterValues"},
		interfaces.LogField{Key: "session_id", Value: "sess_12345"},
	)

	logger.Error("消息解析失败",
		interfaces.LogField{Key: "error", Value: "invalid XML format"},
		interfaces.LogField{Key: "raw_data", Value: "<invalid>"},
	)

	// 4. 获取专用Core日志记录器
	fmt.Println("\n4. 使用专用Core日志记录器...")
	coreLogger := logger.GetCoreLogger()
	coreLogger.Info("使用专用Core日志记录器",
		interfaces.LogField{Key: "component", Value: "parser"},
		interfaces.LogField{Key: "operation", Value: "parse_message"},
	)

	// 5. 演示不同日志级别
	fmt.Println("\n5. 演示不同日志级别...")
	logger.Debug("调试信息：内存使用情况",
		interfaces.LogField{Key: "memory_mb", Value: 128},
	)

	logger.Info("信息：连接建立成功",
		interfaces.LogField{Key: "remote_addr", Value: "192.168.1.100"},
	)

	logger.Warn("警告：连接超时，正在重试",
		interfaces.LogField{Key: "retry_count", Value: 2},
	)

	logger.Error("错误：数据库连接失败",
		interfaces.LogField{Key: "db_host", Value: "localhost"},
		interfaces.LogField{Key: "error_code", Value: 1045},
	)

	// 6. 演示日志轮转
	fmt.Println("\n6. 演示日志轮转...")
	for i := 0; i < 100; i++ {
		logger.LogCoreInfo(fmt.Sprintf("批量日志测试 %d", i),
			interfaces.LogField{Key: "batch_id", Value: i},
			interfaces.LogField{Key: "timestamp", Value: time.Now().Unix()},
		)
	}

	// 7. 手动触发日志轮转
	fmt.Println("\n7. 手动触发日志轮转...")
	if err := logger.RotateLogFiles(); err != nil {
		fmt.Printf("日志轮转失败: %v\n", err)
	} else {
		fmt.Println("日志轮转成功")
	}

	fmt.Println("\n=== 示例完成 ===")
	fmt.Println("请检查日志文件:")
	fmt.Println("- /var/log/tr069.log (如果有权限)")
	fmt.Println("- /tmp/tr069.log (临时文件)")
}
