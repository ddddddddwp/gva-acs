// Package interfaces defines the public interfaces for the TR069 library.
package interfaces

// PersistenceFormat 配置持久化格式类型
type PersistenceFormat string

const (
	// PersistenceFormatJSON JSON格式
	PersistenceFormatJSON PersistenceFormat = "json"
	// PersistenceFormatXML XML格式
	PersistenceFormatXML PersistenceFormat = "xml"
	// PersistenceFormatYAML YAML格式
	PersistenceFormatYAML PersistenceFormat = "yaml"
)

// ConfigPersistence 配置持久化接口
type ConfigPersistence interface {
	// Save 保存配置到指定路径
	Save(config map[string]interface{}, path string) error

	// Load 从指定路径加载配置
	Load(path string) (map[string]interface{}, error)

	// GetFormat 获取持久化格式
	GetFormat() PersistenceFormat

	// SetFormat 设置持久化格式
	SetFormat(format PersistenceFormat)

	// Backup 备份配置到指定路径
	Backup(sourcePath, backupPath string) error

	// Restore 从备份恢复配置
	Restore(backupPath, targetPath string) error
}

// PersistenceOption 配置持久化选项函数
type PersistenceOption func(ConfigPersistence)

// WithPersistenceFormat 设置持久化格式
func WithPersistenceFormat(format PersistenceFormat) PersistenceOption {
	return func(cp ConfigPersistence) {
		cp.SetFormat(format)
	}
}
