// Package persistence provides implementation for configuration persistence functionality.
package persistence

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-core/interfaces"
	"gopkg.in/yaml.v3"
)

// configPersistence 实现了 interfaces.ConfigPersistence 接口
type configPersistence struct {
	format interfaces.PersistenceFormat
}

// NewConfigPersistence 创建一个新的配置持久化实例
func NewConfigPersistence(options ...interfaces.PersistenceOption) interfaces.ConfigPersistence {
	cp := &configPersistence{
		format: interfaces.PersistenceFormatJSON, // 默认使用JSON格式
	}

	// 应用选项
	for _, option := range options {
		option(cp)
	}

	return cp
}

// Save 保存配置到指定路径
func (cp *configPersistence) Save(config map[string]interface{}, path string) error {
	// 确保目录存在
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// 创建文件
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	// 根据格式序列化配置
	switch cp.format {
	case interfaces.PersistenceFormatJSON:
		encoder := json.NewEncoder(file)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(config); err != nil {
			return fmt.Errorf("failed to encode JSON: %w", err)
		}
	case interfaces.PersistenceFormatXML:
		encoder := xml.NewEncoder(file)
		encoder.Indent("", "  ")
		if err := encoder.Encode(config); err != nil {
			return fmt.Errorf("failed to encode XML: %w", err)
		}
	case interfaces.PersistenceFormatYAML:
		encoder := yaml.NewEncoder(file)
		if err := encoder.Encode(config); err != nil {
			return fmt.Errorf("failed to encode YAML: %w", err)
		}
	default:
		return fmt.Errorf("unsupported format: %s", cp.format)
	}

	return nil
}

// Load 从指定路径加载配置
func (cp *configPersistence) Load(path string) (map[string]interface{}, error) {
	// 打开文件
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// 读取文件内容
	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// 根据格式反序列化配置
	var config map[string]interface{}
	switch cp.format {
	case interfaces.PersistenceFormatJSON:
		if err := json.Unmarshal(data, &config); err != nil {
			return nil, fmt.Errorf("failed to decode JSON: %w", err)
		}
	case interfaces.PersistenceFormatXML:
		if err := xml.Unmarshal(data, &config); err != nil {
			return nil, fmt.Errorf("failed to decode XML: %w", err)
		}
	case interfaces.PersistenceFormatYAML:
		if err := yaml.Unmarshal(data, &config); err != nil {
			return nil, fmt.Errorf("failed to decode YAML: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported format: %s", cp.format)
	}

	return config, nil
}

// GetFormat 获取持久化格式
func (cp *configPersistence) GetFormat() interfaces.PersistenceFormat {
	return cp.format
}

// SetFormat 设置持久化格式
func (cp *configPersistence) SetFormat(format interfaces.PersistenceFormat) {
	cp.format = format
}

// Backup 备份配置到指定路径
func (cp *configPersistence) Backup(sourcePath, backupPath string) error {
	// 如果备份路径为空，则使用源路径加上时间戳
	if backupPath == "" {
		ext := filepath.Ext(sourcePath)
		baseFilename := sourcePath[:len(sourcePath)-len(ext)]
		timestamp := time.Now().Format("20060102_150405")
		backupPath = fmt.Sprintf("%s_backup_%s%s", baseFilename, timestamp, ext)
	}

	// 确保备份目录存在
	backupDir := filepath.Dir(backupPath)
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	// 读取源文件
	sourceData, err := os.ReadFile(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to read source file: %w", err)
	}

	// 写入备份文件
	if err := os.WriteFile(backupPath, sourceData, 0644); err != nil {
		return fmt.Errorf("failed to write backup file: %w", err)
	}

	return nil
}

// Restore 从备份恢复配置
func (cp *configPersistence) Restore(backupPath, targetPath string) error {
	// 读取备份文件
	backupData, err := os.ReadFile(backupPath)
	if err != nil {
		return fmt.Errorf("failed to read backup file: %w", err)
	}

	// 确保目标目录存在
	targetDir := filepath.Dir(targetPath)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}

	// 写入目标文件
	if err := os.WriteFile(targetPath, backupData, 0644); err != nil {
		return fmt.Errorf("failed to write target file: %w", err)
	}

	return nil
}
