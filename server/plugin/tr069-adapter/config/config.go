package config

// TR069Config TR069适配器配置
type TR069Config struct {
	Server   ServerConfig   `mapstructure:"server" json:"server" yaml:"server"`
	Database DatabaseConfig `mapstructure:"database" json:"database" yaml:"database"`
	Redis    RedisConfig    `mapstructure:"redis" json:"redis" yaml:"redis"`
	Log      LogConfig      `mapstructure:"log" json:"log" yaml:"log"`
	CWMP     CWMPConfig     `mapstructure:"cwmp" json:"cwmp" yaml:"cwmp"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Port         int    `mapstructure:"port" json:"port" yaml:"port"`                   // TR069服务端口，默认7547
	ReadTimeout  int    `mapstructure:"read_timeout" json:"readTimeout" yaml:"read_timeout"`
	WriteTimeout int    `mapstructure:"write_timeout" json:"writeTimeout" yaml:"write_timeout"`
	IdleTimeout  int    `mapstructure:"idle_timeout" json:"idleTimeout" yaml:"idle_timeout"`
	MaxHeaderBytes int  `mapstructure:"max_header_bytes" json:"maxHeaderBytes" yaml:"max_header_bytes"`
	TLS          TLSConfig `mapstructure:"tls" json:"tls" yaml:"tls"`
}

// TLSConfig TLS配置
type TLSConfig struct {
	Enabled  bool   `mapstructure:"enabled" json:"enabled" yaml:"enabled"`
	CertFile string `mapstructure:"cert_file" json:"certFile" yaml:"cert_file"`
	KeyFile  string `mapstructure:"key_file" json:"keyFile" yaml:"key_file"`
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Type         string `mapstructure:"type" json:"type" yaml:"type"`
	Host         string `mapstructure:"host" json:"host" yaml:"host"`
	Port         int    `mapstructure:"port" json:"port" yaml:"port"`
	Database     string `mapstructure:"database" json:"database" yaml:"database"`
	Username     string `mapstructure:"username" json:"username" yaml:"username"`
	Password     string `mapstructure:"password" json:"password" yaml:"password"`
	MaxIdleConns int    `mapstructure:"max_idle_conns" json:"maxIdleConns" yaml:"max_idle_conns"`
	MaxOpenConns int    `mapstructure:"max_open_conns" json:"maxOpenConns" yaml:"max_open_conns"`
	MaxLifetime  int    `mapstructure:"max_lifetime" json:"maxLifetime" yaml:"max_lifetime"`
}

// RedisConfig Redis配置
type RedisConfig struct {
	Host     string `mapstructure:"host" json:"host" yaml:"host"`
	Port     int    `mapstructure:"port" json:"port" yaml:"port"`
	Password string `mapstructure:"password" json:"password" yaml:"password"`
	DB       int    `mapstructure:"db" json:"db" yaml:"db"`
	PoolSize int    `mapstructure:"pool_size" json:"poolSize" yaml:"pool_size"`
}

// LogConfig 日志配置
type LogConfig struct {
	Level      string `mapstructure:"level" json:"level" yaml:"level"`
	Format     string `mapstructure:"format" json:"format" yaml:"format"`
	Output     string `mapstructure:"output" json:"output" yaml:"output"`
	MaxSize    int    `mapstructure:"max_size" json:"maxSize" yaml:"max_size"`
	MaxBackups int    `mapstructure:"max_backups" json:"maxBackups" yaml:"max_backups"`
	MaxAge     int    `mapstructure:"max_age" json:"maxAge" yaml:"max_age"`
	Compress   bool   `mapstructure:"compress" json:"compress" yaml:"compress"`
}

// CWMPConfig CWMP协议配置
type CWMPConfig struct {
	ConnectionRequestURL string `mapstructure:"connection_request_url" json:"connectionRequestUrl" yaml:"connection_request_url"`
	ConnectionRequestUsername string `mapstructure:"connection_request_username" json:"connectionRequestUsername" yaml:"connection_request_username"`
	ConnectionRequestPassword string `mapstructure:"connection_request_password" json:"connectionRequestPassword" yaml:"connection_request_password"`
	PeriodicInformInterval    int    `mapstructure:"periodic_inform_interval" json:"periodicInformInterval" yaml:"periodic_inform_interval"`
	ParameterKey              string `mapstructure:"parameter_key" json:"parameterKey" yaml:"parameter_key"`
	RetryCount                int    `mapstructure:"retry_count" json:"retryCount" yaml:"retry_count"`
	RetryInterval             int    `mapstructure:"retry_interval" json:"retryInterval" yaml:"retry_interval"`
	SessionTimeout            int    `mapstructure:"session_timeout" json:"sessionTimeout" yaml:"session_timeout"`
	MaxEnvelopes              int    `mapstructure:"max_envelopes" json:"maxEnvelopes" yaml:"max_envelopes"`
}

// GetDefaultConfig 获取默认配置
func GetDefaultConfig() *TR069Config {
	return &TR069Config{
		Server: ServerConfig{
			Port:         7547,
			ReadTimeout:  60,
			WriteTimeout: 60,
			IdleTimeout:  120,
			MaxHeaderBytes: 1 << 20, // 1MB
			TLS: TLSConfig{
				Enabled: false,
			},
		},
		Database: DatabaseConfig{
			Type:         "mysql",
			Host:         "localhost",
			Port:         3306,
			Database:     "gva",
			Username:     "root",
			Password:     "",
			MaxIdleConns: 10,
			MaxOpenConns: 100,
			MaxLifetime:  3600,
		},
		Redis: RedisConfig{
			Host:     "localhost",
			Port:     6379,
			Password: "",
			DB:       0,
			PoolSize: 10,
		},
		Log: LogConfig{
			Level:      "info",
			Format:     "json",
			Output:     "stdout",
			MaxSize:    100,
			MaxBackups: 5,
			MaxAge:     30,
			Compress:   true,
		},
		CWMP: CWMPConfig{
			ConnectionRequestURL:      "",
			ConnectionRequestUsername: "",
			ConnectionRequestPassword: "",
			PeriodicInformInterval:    300,
			ParameterKey:              "",
			RetryCount:                3,
			RetryInterval:             30,
			SessionTimeout:            300,
			MaxEnvelopes:              1,
		},
	}
}