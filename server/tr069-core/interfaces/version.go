package interfaces

import "time"

// VersionManager 定义版本管理接口
type VersionManager interface {
	// GetVersion 获取当前版本号
	GetVersion() string

	// GetBuildInfo 获取构建信息
	GetBuildInfo() BuildInfo

	// IsCompatible 检查版本兼容性
	IsCompatible(version string) bool
}

// BuildInfo 构建信息
type BuildInfo struct {
	Version   string
	BuildTime string
	GitCommit string
	GoVersion string
}

// ConfigVersionManager 配置版本管理接口
type ConfigVersionManager interface {
	// CreateVersion 创建新版本
	CreateVersion(config map[string]interface{}, description string, author string) (string, error)

	// GetVersion 获取指定版本
	GetVersion(version string) (*ConfigVersion, error)

	// ListVersions 列出所有版本
	ListVersions() ([]*ConfigVersion, error)

	// CompareVersions 比较两个版本
	CompareVersions(version1, version2 string) ([]VersionDiff, error)

	// RollbackToVersion 回滚到指定版本
	RollbackToVersion(version string) error

	// SetVersionStoragePath 设置版本存储路径
	SetVersionStoragePath(path string)
}

// ConfigVersion 配置版本信息
type ConfigVersion struct {
	Version     string
	Timestamp   time.Time
	Description string
	Author      string
	Config      map[string]interface{}
	ConfigPath  string
}

// VersionDiff 版本差异
type VersionDiff struct {
	Key        string
	OldValue   interface{}
	NewValue   interface{}
	ChangeType string
	Added      map[string]interface{}
	Modified   map[string]interface{}
	Removed    map[string]interface{}
}

// VersionManagerOption 版本管理器选项函数类型
type VersionManagerOption func(ConfigVersionManager)
