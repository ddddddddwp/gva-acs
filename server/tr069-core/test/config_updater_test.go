package test

import (
	"fmt"
	"github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-core/internal/config"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestConfigUpdater(t *testing.T) {
	// 创建临时配置文件
	tempDir, err := os.MkdirTemp("", "config_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	configFile := filepath.Join(tempDir, "config.json")
	err = os.WriteFile(configFile, []byte(`{"server": {"port": 8080}}`), 0644)
	if err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// 创建配置更新器
	factory := config.NewConfigUpdaterFactory()
	configUpdater := factory(
		config.WithConfigSource(configFile),
		config.WithConfigFormat("json"),
		config.WithConfigWatchInterval(100*time.Millisecond),
	)

	// 测试配置加载
	err = configUpdater.LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// 测试获取配置
	var port int
	err = configUpdater.GetConfigValue("server.port", &port)
	if err != nil {
		t.Fatalf("Failed to get config value: %v", err)
	}
	if port != 8080 {
		t.Errorf("Expected port 8080, got %d", port)
	}

	// 测试配置监听器
	configChanged := false
	listener := config.NewDefaultListener(func(oldConfig, newConfig interface{}) {
		configChanged = true
	})

	configUpdater.AddListener(listener)

	// 更新配置文件
	err = os.WriteFile(configFile, []byte(`{"server": {"port": 9090}}`), 0644)
	if err != nil {
		t.Fatalf("Failed to update config file: %v", err)
	}

	// 等待配置更新
	time.Sleep(300 * time.Millisecond)

	// 验证配置已更新
	err = configUpdater.GetConfigValue("server.port", &port)
	if err != nil {
		t.Fatalf("Failed to get updated config value: %v", err)
	}
	if port != 9090 {
		t.Errorf("Expected updated port 9090, got %d", port)
	}

	// 验证监听器被调用
	if !configChanged {
		t.Error("Config listener was not called")
	}

	// 测试移除监听器
	configChanged = false
	configUpdater.RemoveListener(listener)

	// 再次更新配置
	err = os.WriteFile(configFile, []byte(`{"server": {"port": 7070}}`), 0644)
	if err != nil {
		t.Fatalf("Failed to update config file again: %v", err)
	}

	// 等待配置更新
	time.Sleep(300 * time.Millisecond)

	// 验证监听器不再被调用
	if configChanged {
		t.Error("Config listener was called after removal")
	}

	// 停止配置更新器
	configUpdater.Stop()

	fmt.Println("ConfigUpdater tests passed")
}
