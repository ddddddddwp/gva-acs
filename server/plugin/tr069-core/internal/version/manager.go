// Package version provides implementation for configuration version management functionality.
package version

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/root/demo/tr069/interfaces"
)

// versionManager 实现了 interfaces.ConfigVersionManager 接口
type versionManager struct {
storagePath string
}

// NewVersionManager 创建一个新的配置版本管理器
func NewVersionManager(options ...interfaces.VersionManagerOption) interfaces.ConfigVersionManager {
vm := &versionManager{
storagePath: "./versions", // 默认版本存储路径
}

// 应用选项
for _, option := range options {
option(vm)
}

// 确保版本存储目录存在
if err := os.MkdirAll(vm.storagePath, 0755); err != nil {
// 处理错误，可以记录日志
}

return vm
}

// CreateVersion 创建新版本
func (vm *versionManager) CreateVersion(config map[string]interface{}, description string, author string) (string, error) {
// 生成版本号
version := generateVersionID()

// 创建版本信息
versionInfo := &interfaces.ConfigVersion{
Version:     version,
Timestamp:   time.Now(),
Description: description,
Author:      author,
ConfigPath:  filepath.Join(vm.storagePath, version+".config.json"),
}

// 保存版本信息
if err := vm.saveVersionInfo(versionInfo); err != nil {
return "", fmt.Errorf("failed to save version info: %w", err)
}

// 保存配置
if err := vm.saveVersionConfig(version, config); err != nil {
// 如果保存配置失败，尝试删除版本信息
_ = os.Remove(filepath.Join(vm.storagePath, version+".info.json"))
return "", fmt.Errorf("failed to save version config: %w", err)
}

return version, nil
}

// GetVersion 获取指定版本
func (vm *versionManager) GetVersion(version string) (*interfaces.ConfigVersion, error) {
versionInfoPath := filepath.Join(vm.storagePath, version+".info.json")
data, err := os.ReadFile(versionInfoPath)
if err != nil {
return nil, fmt.Errorf("failed to read version info: %w", err)
}

var versionInfo interfaces.ConfigVersion
if err := json.Unmarshal(data, &versionInfo); err != nil {
return nil, fmt.Errorf("failed to parse version info: %w", err)
}

return &versionInfo, nil
}

// GetVersionConfig 获取指定版本的配置
func (vm *versionManager) GetVersionConfig(version string) (map[string]interface{}, error) {
configPath := filepath.Join(vm.storagePath, version+".config.json")
data, err := os.ReadFile(configPath)
if err != nil {
return nil, fmt.Errorf("failed to read version config: %w", err)
}

var config map[string]interface{}
if err := json.Unmarshal(data, &config); err != nil {
return nil, fmt.Errorf("failed to parse version config: %w", err)
}

return config, nil
}

// ListVersions 列出所有版本
func (vm *versionManager) ListVersions() ([]*interfaces.ConfigVersion, error) {
// 读取版本存储目录
entries, err := os.ReadDir(vm.storagePath)
if err != nil {
return nil, fmt.Errorf("failed to read version directory: %w", err)
}

versions := make([]*interfaces.ConfigVersion, 0)
for _, entry := range entries {
// 只处理版本信息文件
if !entry.IsDir() && filepath.Ext(entry.Name()) == ".json" && filepath.Ext(filepath.Base(entry.Name()[:len(entry.Name())-5])) == ".info" {
versionID := filepath.Base(entry.Name()[:len(entry.Name())-10]) // 去掉 .info.json
versionInfo, err := vm.GetVersion(versionID)
if err != nil {
continue
}
versions = append(versions, versionInfo)
}
}

// 按时间戳排序，最新的在前
sort.Slice(versions, func(i, j int) bool {
return versions[i].Timestamp.After(versions[j].Timestamp)
})

return versions, nil
}

// CompareVersions 比较两个版本的差异
func (vm *versionManager) CompareVersions(oldVersion, newVersion string) ([]interfaces.VersionDiff, error) {
// 获取旧版本配置
oldConfig, err := vm.GetVersionConfig(oldVersion)
if err != nil {
return nil, fmt.Errorf("failed to get old version config: %w", err)
}

// 获取新版本配置
newConfig, err := vm.GetVersionConfig(newVersion)
if err != nil {
return nil, fmt.Errorf("failed to get new version config: %w", err)
}

// 比较差异
diffs := make([]interfaces.VersionDiff, 0)

// 检查新增和修改的配置项
for key, newValue := range newConfig {
if oldValue, exists := oldConfig[key]; !exists {
// 新增配置项
diffs = append(diffs, interfaces.VersionDiff{
Key:        key,
OldValue:   nil,
NewValue:   newValue,
ChangeType: "added",
})
} else if !equals(oldValue, newValue) {
// 修改配置项
diffs = append(diffs, interfaces.VersionDiff{
Key:        key,
OldValue:   oldValue,
NewValue:   newValue,
ChangeType: "modified",
})
}
}

// 检查删除的配置项
for key, oldValue := range oldConfig {
if _, exists := newConfig[key]; !exists {
// 删除配置项
diffs = append(diffs, interfaces.VersionDiff{
Key:        key,
OldValue:   oldValue,
NewValue:   nil,
ChangeType: "deleted",
})
}
}

return diffs, nil
}

// RollbackToVersion 回滚到指定版本
func (vm *versionManager) RollbackToVersion(version string) error {
// 获取指定版本的配置
config, err := vm.GetVersionConfig(version)
if err != nil {
return fmt.Errorf("failed to get version config: %w", err)
}

// 创建新版本，标记为回滚
_, err = vm.CreateVersion(config, fmt.Sprintf("Rollback to version %s", version), "system")
if err != nil {
return fmt.Errorf("failed to create rollback version: %w", err)
}

return nil
}

// DeleteVersion 删除指定版本
func (vm *versionManager) DeleteVersion(version string) error {
// 删除版本信息文件
infoPath := filepath.Join(vm.storagePath, version+".info.json")
if err := os.Remove(infoPath); err != nil {
return fmt.Errorf("failed to delete version info: %w", err)
}

// 删除版本配置文件
configPath := filepath.Join(vm.storagePath, version+".config.json")
if err := os.Remove(configPath); err != nil {
return fmt.Errorf("failed to delete version config: %w", err)
}

return nil
}

// GetLatestVersion 获取最新版本
func (vm *versionManager) GetLatestVersion() (*interfaces.ConfigVersion, error) {
versions, err := vm.ListVersions()
if err != nil {
return nil, fmt.Errorf("failed to list versions: %w", err)
}

if len(versions) == 0 {
return nil, fmt.Errorf("no versions found")
}

// 由于 ListVersions 已经按时间戳排序，最新的在前
return versions[0], nil
}

// SetVersionStoragePath 设置版本存储路径
func (vm *versionManager) SetVersionStoragePath(path string) {
vm.storagePath = path
// 确保版本存储目录存在
if err := os.MkdirAll(vm.storagePath, 0755); err != nil {
// 处理错误，可以记录日志
}
}

// GetVersionStoragePath 获取版本存储路径
func (vm *versionManager) GetVersionStoragePath() string {
return vm.storagePath
}

// 保存版本信息
func (vm *versionManager) saveVersionInfo(versionInfo *interfaces.ConfigVersion) error {
data, err := json.MarshalIndent(versionInfo, "", "  ")
if err != nil {
return fmt.Errorf("failed to marshal version info: %w", err)
}

infoPath := filepath.Join(vm.storagePath, versionInfo.Version+".info.json")
if err := os.WriteFile(infoPath, data, 0644); err != nil {
return fmt.Errorf("failed to write version info: %w", err)
}

return nil
}

// 保存版本配置
func (vm *versionManager) saveVersionConfig(version string, config map[string]interface{}) error {
data, err := json.MarshalIndent(config, "", "  ")
if err != nil {
return fmt.Errorf("failed to marshal version config: %w", err)
}

configPath := filepath.Join(vm.storagePath, version+".config.json")
if err := os.WriteFile(configPath, data, 0644); err != nil {
return fmt.Errorf("failed to write version config: %w", err)
}

return nil
}

// 生成版本ID
func generateVersionID() string {
return time.Now().Format("20060102150405") + "-" + uuid.New().String()[:8]
}

// 比较两个值是否相等
func equals(a, b interface{}) bool {
// 简单比较，可以根据需要扩展
return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
}
