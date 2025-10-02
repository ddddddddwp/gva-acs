package test

import (
	"fmt"
	"github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-core/internal/version"
	"os"
	"testing"
	"time"
)

func TestConfigVersionManager(t *testing.T) {
	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "version_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 创建版本管理器
	factory := version.NewVersionManagerFactory()
	versionManager := factory(
		version.WithVersionStoragePath(tempDir),
	)

	// 测试创建版本
	configData := map[string]interface{}{
		"server": map[string]interface{}{
			"host": "localhost",
			"port": 8080,
		},
	}

	v1, err := versionManager.CreateVersion(configData, "Initial version")
	if err != nil {
		t.Fatalf("Failed to create version: %v", err)
	}

	// 验证版本已创建
	if v1 == "" {
		t.Fatal("Version ID should not be empty")
	}

	// 测试获取版本
	v1Config, err := versionManager.GetVersionConfig(v1)
	if err != nil {
		t.Fatalf("Failed to get version config: %v", err)
	}

	// 验证版本配置
	v1ServerConfig, ok := v1Config.(map[string]interface{})["server"].(map[string]interface{})
	if !ok {
		t.Fatal("Invalid server config format in version")
	}

	v1Port, ok := v1ServerConfig["port"].(float64)
	if !ok || v1Port != 8080 {
		t.Errorf("Expected port 8080 in version, got %v", v1ServerConfig["port"])
	}

	// 创建第二个版本
	updatedConfig := map[string]interface{}{
		"server": map[string]interface{}{
			"host": "localhost",
			"port": 9090,
		},
	}

	time.Sleep(100 * time.Millisecond) // 确保版本时间戳不同
	v2, err := versionManager.CreateVersion(updatedConfig, "Updated port")
	if err != nil {
		t.Fatalf("Failed to create second version: %v", err)
	}

	// 测试列出所有版本
	versions, err := versionManager.ListVersions()
	if err != nil {
		t.Fatalf("Failed to list versions: %v", err)
	}

	if len(versions) != 2 {
		t.Errorf("Expected 2 versions, got %d", len(versions))
	}

	// 测试比较版本
	diff, err := versionManager.CompareVersions(v1, v2)
	if err != nil {
		t.Fatalf("Failed to compare versions: %v", err)
	}

	if len(diff.Changes) == 0 {
		t.Error("Version diff should contain changes")
	}

	// 测试回滚版本
	err = versionManager.RollbackToVersion(v1)
	if err != nil {
		t.Fatalf("Failed to rollback to version: %v", err)
	}

	// 获取最新版本
	latestVersion, err := versionManager.GetLatestVersion()
	if err != nil {
		t.Fatalf("Failed to get latest version: %v", err)
	}

	// 验证最新版本是回滚后的版本
	latestConfig, err := versionManager.GetVersionConfig(latestVersion)
	if err != nil {
		t.Fatalf("Failed to get latest version config: %v", err)
	}

	latestServerConfig, ok := latestConfig.(map[string]interface{})["server"].(map[string]interface{})
	if !ok {
		t.Fatal("Invalid server config format in latest version")
	}

	latestPort, ok := latestServerConfig["port"].(float64)
	if !ok || latestPort != 8080 {
		t.Errorf("Expected port 8080 in latest version after rollback, got %v", latestServerConfig["port"])
	}

	// 测试删除版本
	err = versionManager.DeleteVersion(v2)
	if err != nil {
		t.Fatalf("Failed to delete version: %v", err)
	}

	// 验证版本已删除
	versions, err = versionManager.ListVersions()
	if err != nil {
		t.Fatalf("Failed to list versions after deletion: %v", err)
	}

	if len(versions) != 1 {
		t.Errorf("Expected 1 version after deletion, got %d", len(versions))
	}

	fmt.Println("ConfigVersionManager tests passed")
}
