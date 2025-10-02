package test

import (
"fmt"
"os"
"path/filepath"
"github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-core/internal/persistence"
"testing"
)

func TestConfigPersistence(t *testing.T) {
// 创建临时目录
tempDir, err := os.MkdirTemp("", "persistence_test")
if err != nil {
t.Fatalf("Failed to create temp dir: %v", err)
}
defer os.RemoveAll(tempDir)

// 创建配置持久化对象
factory := persistence.NewPersistenceFactory()
configPersistence := factory(
persistence.WithPersistenceFormat("json"),
)

// 测试配置保存
configData := map[string]interface{}{
"server": map[string]interface{}{
"host": "localhost",
"port": 8080,
},
"logging": map[string]interface{}{
"level": "info",
},
}

configFile := filepath.Join(tempDir, "config.json")
err = configPersistence.Save(configFile, configData)
if err != nil {
t.Fatalf("Failed to save config: %v", err)
}

// 验证文件已创建
if _, err := os.Stat(configFile); os.IsNotExist(err) {
t.Fatalf("Config file was not created")
}

// 测试配置加载
var loadedConfig map[string]interface{}
err = configPersistence.Load(configFile, &loadedConfig)
if err != nil {
t.Fatalf("Failed to load config: %v", err)
}

// 验证加载的配置
serverConfig, ok := loadedConfig["server"].(map[string]interface{})
if !ok {
t.Fatalf("Invalid server config format")
}

port, ok := serverConfig["port"].(float64)
if !ok || port != 8080 {
t.Errorf("Expected port 8080, got %v", serverConfig["port"])
}

// 测试备份功能
backupFile := filepath.Join(tempDir, "config.json.bak")
err = configPersistence.Backup(configFile, backupFile)
if err != nil {
t.Fatalf("Failed to backup config: %v", err)
}

// 验证备份文件已创建
if _, err := os.Stat(backupFile); os.IsNotExist(err) {
t.Fatalf("Backup file was not created")
}

// 修改原始配置
err = os.WriteFile(configFile, []byte(`{"server":{"host":"localhost","port":9090}}`), 0644)
if err != nil {
t.Fatalf("Failed to modify config file: %v", err)
}

// 测试恢复功能
err = configPersistence.Restore(backupFile, configFile)
if err != nil {
t.Fatalf("Failed to restore config: %v", err)
}

// 验证恢复后的配置
var restoredConfig map[string]interface{}
err = configPersistence.Load(configFile, &restoredConfig)
if err != nil {
t.Fatalf("Failed to load restored config: %v", err)
}

restoredServerConfig, ok := restoredConfig["server"].(map[string]interface{})
if !ok {
t.Fatalf("Invalid restored server config format")
}

restoredPort, ok := restoredServerConfig["port"].(float64)
if !ok || restoredPort != 8080 {
t.Errorf("Expected restored port 8080, got %v", restoredServerConfig["port"])
}

fmt.Println("ConfigPersistence tests passed")
}
