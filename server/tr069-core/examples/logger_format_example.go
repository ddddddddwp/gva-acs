package main

import (
	"fmt"
	"os"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-core/interfaces"
	"github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-core/pkg/logger"
)

func main() {
	fmt.Println("=== TR069 Core 日志格式测试 ===")
	
	// 测试Tab分隔格式
	fmt.Println("\n1. 测试Tab分隔格式 (LogFormatTabSeparated)")
	testTabSeparatedFormat()
	
	// 测试紧凑格式
	fmt.Println("\n2. 测试紧凑格式 (LogFormatCompact)")
	testCompactFormat()
	
	// 测试传统格式对比
	fmt.Println("\n3. 传统格式对比 (LogFormatText)")
	testTraditionalFormat()
	
	fmt.Println("\n=== 测试完成 ===")
	fmt.Println("日志文件位置:")
	fmt.Println("- Tab分隔格式: /tmp/tr069_tab.log")
	fmt.Println("- 紧凑格式: /tmp/tr069_compact.log")
	fmt.Println("- 传统格式: /tmp/tr069_traditional.log")
}

func testTabSeparatedFormat() {
	config := logger.LogStorageConfig{
		FilePath:      "/tmp/tr069_tab.log",
		MaxSize:       10,
		MaxBackups:    3,
		MaxAge:        7,
		Compress:      false,
		ConsoleOutput: true,
		Level:         interfaces.LogLevelInfo,
		Format:        interfaces.LogFormatTabSeparated,
	}
	
	log := logger.NewFileLogger(config)
	
	fmt.Println("输出格式: 时间\\t级别\\t调用者\\t消息")
	log.Info("这是Tab分隔格式的信息日志", interfaces.LogField{Key: "component", Value: "formatter"})
	log.Warn("这是Tab分隔格式的警告日志", interfaces.LogField{Key: "module", Value: "test"})
	log.Error("这是Tab分隔格式的错误日志", interfaces.LogField{Key: "error_code", Value: 500})
	
	time.Sleep(100 * time.Millisecond) // 确保日志写入
}

func testCompactFormat() {
	config := logger.LogStorageConfig{
		FilePath:      "/tmp/tr069_compact.log",
		MaxSize:       10,
		MaxBackups:    3,
		MaxAge:        7,
		Compress:      false,
		ConsoleOutput: true,
		Level:         interfaces.LogLevelInfo,
		Format:        interfaces.LogFormatCompact,
	}
	
	log := logger.NewFileLogger(config)
	
	fmt.Println("输出格式: 时间\\t级别\\t调用者\\t消息")
	log.Info("这是紧凑格式的信息日志", interfaces.LogField{Key: "service", Value: "tr069"})
	log.Warn("这是紧凑格式的警告日志", interfaces.LogField{Key: "request_id", Value: "req-123"})
	log.Error("这是紧凑格式的错误日志", interfaces.LogField{Key: "duration", Value: 250})
	
	time.Sleep(100 * time.Millisecond) // 确保日志写入
}

func testTraditionalFormat() {
	config := logger.LogStorageConfig{
		FilePath:      "/tmp/tr069_traditional.log",
		MaxSize:       10,
		MaxBackups:    3,
		MaxAge:        7,
		Compress:      false,
		ConsoleOutput: true,
		Level:         interfaces.LogLevelInfo,
		Format:        interfaces.LogFormatText,
	}
	
	log := logger.NewFileLogger(config)
	
	fmt.Println("输出格式: 传统控制台格式")
	log.Info("这是传统格式的信息日志", interfaces.LogField{Key: "component", Value: "formatter"})
	log.Warn("这是传统格式的警告日志", interfaces.LogField{Key: "module", Value: "test"})
	log.Error("这是传统格式的错误日志", interfaces.LogField{Key: "error_code", Value: 500})
	
	time.Sleep(100 * time.Millisecond) // 确保日志写入
}

// 演示如何在实际应用中使用新格式
func demonstrateUsage() {
	fmt.Println("\n=== 实际应用示例 ===")
	
	// 创建类似用户要求的格式
	config := logger.LogStorageConfig{
		FilePath:      "/tmp/tr069_demo.log",
		MaxSize:       10,
		MaxBackups:    3,
		MaxAge:        7,
		Compress:      false,
		ConsoleOutput: true,
		Level:         interfaces.LogLevelInfo,
		Format:        interfaces.LogFormatCompact, // 使用紧凑格式
	}
	
	log := logger.NewFileLogger(config)
	
	// 模拟实际应用场景
	log.Info("TR069 连接建立成功", 
		interfaces.LogField{Key: "device_id", Value: "CPE-001"},
		interfaces.LogField{Key: "session_id", Value: "sess-abc123"})
		
	log.Warn("参数值验证失败", 
		interfaces.LogField{Key: "parameter", Value: "Device.WiFi.Radio.1.Channel"},
		interfaces.LogField{Key: "value", Value: "invalid"})
		
	log.Error("RPC调用超时", 
		interfaces.LogField{Key: "method", Value: "GetParameterValues"},
		interfaces.LogField{Key: "timeout", Value: "30s"})
}

func init() {
	// 确保日志目录存在
	os.MkdirAll("/tmp", 0755)
}