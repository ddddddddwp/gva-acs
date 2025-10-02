# TR069-Adapter GVA插件设计文档

## 1. 插件概述

TR069-Adapter作为GVA插件，专门负责与CPE设备的TR069协议通信，不提供Web界面，仅处理设备连接、SOAP消息解析和数据存储。

### 1.1 设计理念

- **职责单一**：专注于TR069协议处理，不涉及用户界面
- **依赖GVA**：充分利用GVA的基础设施（数据库、日志、配置等）
- **独立端口**：使用7547端口处理CPE连接，与GVA主服务（8888端口）分离
- **数据共享**：将设备数据存储到GVA数据库，供其他插件使用

### 1.2 架构优势

1. **统一部署**：与GVA主服务一起部署，简化运维
2. **资源共享**：共享数据库连接池、Redis连接、日志系统
3. **配置统一**：使用GVA的配置管理系统
4. **监控集成**：集成到GVA的监控体系中

## 2. 插件架构设计

### 2.1 整体架构图

```
┌─────────────────────────────────────────────────────────────┐
│                    GVA Framework (8888端口)                  │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────────┐  ┌─────────────────┐  ┌──────────────┐ │
│  │  TR069-Adapter  │  │ TR069-Management│  │  其他插件     │ │
│  │     插件        │  │      插件       │  │              │ │
│  │   (7547端口)    │  │   (Web界面)     │  │              │ │
│  └─────────────────┘  └─────────────────┘  └──────────────┘ │
├─────────────────────────────────────────────────────────────┤
│                    GVA 核心服务                              │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐           │
│  │   数据库     │ │    Redis    │ │   日志系统   │           │
│  └─────────────┘ └─────────────┘ └─────────────┘           │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
                    ┌─────────────────┐
                    │   MySQL数据库    │
                    │                │
                    │ • cpe_devices   │
                    │ • cpe_parameters│
                    │ • cpe_sessions  │
                    │ • cpe_logs      │
                    └─────────────────┘
```

### 2.2 插件目录结构

```
server/plugin/tr069-adapter/
├── plugin.go                 # 插件入口文件
├── initialize/               # 初始化模块
│   ├── gorm.go              # 数据库初始化
│   ├── router.go            # 路由初始化
│   ├── server.go            # TR069服务器初始化
│   └── config.go            # 配置初始化
├── model/                   # 数据模型
│   ├── cpe_device.go        # CPE设备模型
│   ├── cpe_parameter.go     # CPE参数模型
│   ├── cpe_session.go       # CPE会话模型
│   └── cpe_log.go           # CPE日志模型
├── service/                 # 服务层
│   ├── enter.go             # 服务入口
│   ├── device_service.go    # 设备服务
│   ├── parameter_service.go # 参数服务
│   ├── session_service.go   # 会话服务
│   └── soap_service.go      # SOAP服务
├── handler/                 # TR069协议处理器
│   ├── tr069_handler.go     # TR069主处理器
│   ├── soap_handler.go      # SOAP消息处理器
│   ├── cwmp_handler.go      # CWMP协议处理器
│   └── inform_handler.go    # Inform消息处理器
├── server/                  # TR069服务器
│   ├── tr069_server.go      # TR069服务器实现
│   ├── middleware.go        # 中间件
│   └── router.go            # 路由定义
├── pkg/                     # 工具包
│   ├── cwmp/                # CWMP协议包
│   │   ├── types.go         # CWMP类型定义
│   │   ├── inform.go        # Inform消息
│   │   ├── response.go      # 响应消息
│   │   └── parser.go        # 消息解析器
│   ├── soap/                # SOAP协议包
│   │   ├── envelope.go      # SOAP信封
│   │   ├── parser.go        # SOAP解析器
│   │   └── builder.go       # SOAP构建器
│   └── utils/               # 工具函数
│       ├── xml.go           # XML处理
│       ├── session.go       # 会话管理
│       └── validator.go     # 数据验证
├── config/                  # 配置文件
│   └── tr069.yaml          # TR069配置
└── docs/                    # 文档
    └── README.md           # 插件说明
```

## 3. 核心组件设计

### 3.1 插件入口 (plugin.go)

```go
package tr069adapter

import (
    "context"
    "fmt"
    "net/http"
    "time"

    "github.com/flipped-aurora/gin-vue-admin/server/global"
    "github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-adapter/initialize"
    "github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-adapter/server"
    "github.com/gin-gonic/gin"
    "go.uber.org/zap"
)

type TR069AdapterPlugin struct {
    tr069Server *http.Server
    ctx         context.Context
    cancel      context.CancelFunc
}

func CreateTR069AdapterPlug() *TR069AdapterPlugin {
    ctx, cancel := context.WithCancel(context.Background())
    return &TR069AdapterPlugin{
        ctx:    ctx,
        cancel: cancel,
    }
}

func (p *TR069AdapterPlugin) Register(group *gin.RouterGroup) {
    // 初始化数据库表
    initialize.InitializeDB()
    
    // 初始化配置
    config := initialize.InitializeConfig()
    
    // 启动TR069服务器（7547端口）
    go p.startTR069Server(config)
    
    global.GVA_LOG.Info("TR069-Adapter plugin registered successfully")
}

func (p *TR069AdapterPlugin) RouterPath() string {
    return "tr069-adapter"
}

func (p *TR069AdapterPlugin) startTR069Server(config *initialize.TR069Config) {
    tr069Router := server.NewTR069Router()
    
    p.tr069Server = &http.Server{
        Addr:           fmt.Sprintf(":%d", config.Port),
        Handler:        tr069Router,
        ReadTimeout:    30 * time.Second,
        WriteTimeout:   30 * time.Second,
        MaxHeaderBytes: 1 << 20, // 1MB
    }
    
    global.GVA_LOG.Info("Starting TR069 server", 
        zap.Int("port", config.Port))
    
    if err := p.tr069Server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
        global.GVA_LOG.Error("TR069 server failed to start", zap.Error(err))
    }
}

func (p *TR069AdapterPlugin) Stop() error {
    if p.tr069Server != nil {
        ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
        defer cancel()
        
        if err := p.tr069Server.Shutdown(ctx); err != nil {
            global.GVA_LOG.Error("Failed to shutdown TR069 server", zap.Error(err))
            return err
        }
    }
    
    p.cancel()
    global.GVA_LOG.Info("TR069-Adapter plugin stopped")
    return nil
}
```

### 3.2 数据模型设计

#### 3.2.1 CPE设备模型

```go
package model

import (
    "time"
    "github.com/flipped-aurora/gin-vue-admin/server/global"
)

// CPEDevice CPE设备信息
type CPEDevice struct {
    global.GVA_MODEL
    DeviceID             string     `json:"device_id" gorm:"uniqueIndex;size:255;not null;comment:设备ID"`
    Manufacturer         string     `json:"manufacturer" gorm:"size:100;comment:制造商"`
    OUI                  string     `json:"oui" gorm:"size:6;comment:组织唯一标识符"`
    ProductClass         string     `json:"product_class" gorm:"size:100;comment:产品类别"`
    SerialNumber         string     `json:"serial_number" gorm:"size:100;comment:序列号"`
    HardwareVersion      string     `json:"hardware_version" gorm:"size:50;comment:硬件版本"`
    SoftwareVersion      string     `json:"software_version" gorm:"size:50;comment:软件版本"`
    ConnectionRequestURL string     `json:"connection_request_url" gorm:"size:500;comment:连接请求URL"`
    LastInformTime       *time.Time `json:"last_inform_time" gorm:"comment:最后通信时间"`
    OnlineStatus         int        `json:"online_status" gorm:"default:0;comment:在线状态 0:离线 1:在线"`
    IPAddress            string     `json:"ip_address" gorm:"size:45;comment:设备IP地址"`
    Port                 int        `json:"port" gorm:"comment:设备端口"`
}

func (CPEDevice) TableName() string {
    return "cpe_devices"
}
```

#### 3.2.2 CPE参数模型

```go
// CPEParameter CPE参数信息
type CPEParameter struct {
    global.GVA_MODEL
    DeviceID       string     `json:"device_id" gorm:"index;size:255;not null;comment:设备ID"`
    ParameterName  string     `json:"parameter_name" gorm:"index;size:500;not null;comment:参数名称"`
    ParameterValue string     `json:"parameter_value" gorm:"type:text;comment:参数值"`
    ParameterType  string     `json:"parameter_type" gorm:"size:50;comment:参数类型"`
    Writable       bool       `json:"writable" gorm:"default:false;comment:是否可写"`
    LastUpdated    *time.Time `json:"last_updated" gorm:"comment:最后更新时间"`
}

func (CPEParameter) TableName() string {
    return "cpe_parameters"
}
```

### 3.3 TR069服务器实现

```go
package server

import (
    "net/http"
    "tr069-adapter/handler"
    "tr069-adapter/middleware"
    
    "github.com/gin-gonic/gin"
    "github.com/flipped-aurora/gin-vue-admin/server/global"
)

func NewTR069Router() *gin.Engine {
    router := gin.New()
    
    // 添加中间件
    router.Use(middleware.TR069Logger())
    router.Use(middleware.TR069Recovery())
    router.Use(middleware.TR069CORS())
    
    // 健康检查
    router.GET("/health", func(c *gin.Context) {
        c.JSON(http.StatusOK, gin.H{
            "status": "ok",
            "service": "tr069-adapter",
        })
    })
    
    // TR069协议处理
    tr069Handler := handler.NewTR069Handler()
    
    // CPE连接入口 - 标准TR069路径
    router.POST("/", tr069Handler.HandleTR069Request)
    router.POST("/tr069", tr069Handler.HandleTR069Request)
    
    // 连接请求处理
    router.POST("/connection-request", tr069Handler.HandleConnectionRequest)
    
    return router
}
```

### 3.4 SOAP消息处理器

```go
package handler

import (
    "encoding/xml"
    "fmt"
    "net/http"
    "strings"
    
    "tr069-adapter/pkg/cwmp"
    "tr069-adapter/pkg/soap"
    "tr069-adapter/service"
    
    "github.com/gin-gonic/gin"
    "github.com/flipped-aurora/gin-vue-admin/server/global"
    "go.uber.org/zap"
)

type TR069Handler struct {
    deviceService    *service.DeviceService
    parameterService *service.ParameterService
    sessionService   *service.SessionService
    soapService      *service.SOAPService
}

func NewTR069Handler() *TR069Handler {
    return &TR069Handler{
        deviceService:    service.ServiceGroupApp.DeviceService,
        parameterService: service.ServiceGroupApp.ParameterService,
        sessionService:   service.ServiceGroupApp.SessionService,
        soapService:      service.ServiceGroupApp.SOAPService,
    }
}

func (h *TR069Handler) HandleTR069Request(c *gin.Context) {
    // 读取请求体
    body, err := c.GetRawData()
    if err != nil {
        global.GVA_LOG.Error("Failed to read request body", zap.Error(err))
        c.Status(http.StatusBadRequest)
        return
    }
    
    // 解析SOAP消息
    soapEnvelope, err := h.soapService.ParseSOAPMessage(body)
    if err != nil {
        global.GVA_LOG.Error("Failed to parse SOAP message", zap.Error(err))
        c.Status(http.StatusBadRequest)
        return
    }
    
    // 处理CWMP消息
    response, err := h.processCWMPMessage(c, soapEnvelope)
    if err != nil {
        global.GVA_LOG.Error("Failed to process CWMP message", zap.Error(err))
        c.Status(http.StatusInternalServerError)
        return
    }
    
    // 返回SOAP响应
    c.Header("Content-Type", "text/xml; charset=utf-8")
    c.Header("SOAPAction", "")
    c.Data(http.StatusOK, "text/xml; charset=utf-8", response)
}

func (h *TR069Handler) processCWMPMessage(c *gin.Context, envelope *soap.Envelope) ([]byte, error) {
    // 根据消息类型处理
    switch {
    case strings.Contains(string(envelope.Body), "Inform"):
        return h.handleInform(c, envelope)
    case strings.Contains(string(envelope.Body), "GetParameterValuesResponse"):
        return h.handleGetParameterValuesResponse(c, envelope)
    case strings.Contains(string(envelope.Body), "SetParameterValuesResponse"):
        return h.handleSetParameterValuesResponse(c, envelope)
    default:
        return h.handleUnknownMessage(c, envelope)
    }
}

func (h *TR069Handler) handleInform(c *gin.Context, envelope *soap.Envelope) ([]byte, error) {
    // 解析Inform消息
    var inform cwmp.Inform
    if err := xml.Unmarshal(envelope.Body, &inform); err != nil {
        return nil, fmt.Errorf("failed to unmarshal Inform message: %w", err)
    }
    
    // 获取设备信息
    deviceID := h.buildDeviceID(inform.DeviceId)
    
    // 更新设备信息
    if err := h.deviceService.UpdateDeviceFromInform(deviceID, &inform); err != nil {
        global.GVA_LOG.Error("Failed to update device", 
            zap.String("device_id", deviceID), 
            zap.Error(err))
    }
    
    // 更新参数信息
    if len(inform.ParameterList) > 0 {
        if err := h.parameterService.BatchUpdateParameters(deviceID, inform.ParameterList); err != nil {
            global.GVA_LOG.Error("Failed to update parameters", 
                zap.String("device_id", deviceID), 
                zap.Error(err))
        }
    }
    
    // 创建会话
    sessionID, err := h.sessionService.CreateSession(deviceID, "inform")
    if err != nil {
        global.GVA_LOG.Error("Failed to create session", 
            zap.String("device_id", deviceID), 
            zap.Error(err))
    }
    
    // 构建InformResponse
    response := &cwmp.InformResponse{
        MaxEnvelopes: 1,
    }
    
    // 构建SOAP响应
    return h.soapService.BuildSOAPResponse(response)
}

func (h *TR069Handler) buildDeviceID(deviceId cwmp.DeviceIdStruct) string {
    return fmt.Sprintf("%s-%s-%s", 
        deviceId.Manufacturer, 
        deviceId.OUI, 
        deviceId.SerialNumber)
}
```

## 4. 配置管理

### 4.1 插件配置文件

```yaml
# server/plugin/tr069-adapter/config/tr069.yaml
tr069:
  # TR069服务器配置
  server:
    port: 7547                    # TR069服务端口
    read_timeout: 30              # 读取超时(秒)
    write_timeout: 30             # 写入超时(秒)
    max_header_bytes: 1048576     # 最大请求头大小(1MB)
    
  # 设备管理配置
  device:
    session_timeout: 300          # 会话超时时间(秒)
    max_sessions: 1000            # 最大并发会话数
    cleanup_interval: 60          # 清理间隔(秒)
    
  # 参数管理配置
  parameter:
    batch_size: 100               # 批量更新大小
    cache_ttl: 300                # 缓存TTL(秒)
    
  # 日志配置
  log:
    level: "info"                 # 日志级别
    enable_soap_log: true         # 是否记录SOAP消息
    max_message_size: 10240       # 最大消息记录大小(字节)
```

### 4.2 配置初始化

```go
package initialize

import (
    "github.com/flipped-aurora/gin-vue-admin/server/global"
    "github.com/spf13/viper"
)

type TR069Config struct {
    Server struct {
        Port           int `mapstructure:"port"`
        ReadTimeout    int `mapstructure:"read_timeout"`
        WriteTimeout   int `mapstructure:"write_timeout"`
        MaxHeaderBytes int `mapstructure:"max_header_bytes"`
    } `mapstructure:"server"`
    
    Device struct {
        SessionTimeout  int `mapstructure:"session_timeout"`
        MaxSessions     int `mapstructure:"max_sessions"`
        CleanupInterval int `mapstructure:"cleanup_interval"`
    } `mapstructure:"device"`
    
    Parameter struct {
        BatchSize int `mapstructure:"batch_size"`
        CacheTTL  int `mapstructure:"cache_ttl"`
    } `mapstructure:"parameter"`
    
    Log struct {
        Level           string `mapstructure:"level"`
        EnableSOAPLog   bool   `mapstructure:"enable_soap_log"`
        MaxMessageSize  int    `mapstructure:"max_message_size"`
    } `mapstructure:"log"`
}

func InitializeConfig() *TR069Config {
    config := &TR069Config{}
    
    // 设置默认值
    viper.SetDefault("tr069.server.port", 7547)
    viper.SetDefault("tr069.server.read_timeout", 30)
    viper.SetDefault("tr069.server.write_timeout", 30)
    viper.SetDefault("tr069.server.max_header_bytes", 1048576)
    
    viper.SetDefault("tr069.device.session_timeout", 300)
    viper.SetDefault("tr069.device.max_sessions", 1000)
    viper.SetDefault("tr069.device.cleanup_interval", 60)
    
    viper.SetDefault("tr069.parameter.batch_size", 100)
    viper.SetDefault("tr069.parameter.cache_ttl", 300)
    
    viper.SetDefault("tr069.log.level", "info")
    viper.SetDefault("tr069.log.enable_soap_log", true)
    viper.SetDefault("tr069.log.max_message_size", 10240)
    
    // 从配置文件读取
    if err := viper.UnmarshalKey("tr069", config); err != nil {
        global.GVA_LOG.Error("Failed to unmarshal TR069 config, using defaults")
    }
    
    return config
}
```

## 5. 服务层实现

### 5.1 设备服务

```go
package service

import (
    "time"
    "tr069-adapter/model"
    "tr069-adapter/pkg/cwmp"
    
    "github.com/flipped-aurora/gin-vue-admin/server/global"
    "gorm.io/gorm/clause"
)

type DeviceService struct{}

func (s *DeviceService) UpdateDeviceFromInform(deviceID string, inform *cwmp.Inform) error {
    device := &model.CPEDevice{
        DeviceID:         deviceID,
        Manufacturer:     inform.DeviceId.Manufacturer,
        OUI:              inform.DeviceId.OUI,
        ProductClass:     inform.DeviceId.ProductClass,
        SerialNumber:     inform.DeviceId.SerialNumber,
        LastInformTime:   &time.Time{},
        OnlineStatus:     1,
    }
    
    now := time.Now()
    device.LastInformTime = &now
    
    // 使用UPSERT操作
    return global.GVA_DB.Clauses(clause.OnConflict{
        Columns: []clause.Column{{Name: "device_id"}},
        DoUpdates: clause.AssignmentColumns([]string{
            "last_inform_time", 
            "online_status", 
            "updated_at",
        }),
    }).Create(device).Error
}

func (s *DeviceService) GetDeviceByID(deviceID string) (*model.CPEDevice, error) {
    var device model.CPEDevice
    err := global.GVA_DB.Where("device_id = ?", deviceID).First(&device).Error
    return &device, err
}

func (s *DeviceService) UpdateOnlineStatus(deviceID string, status int) error {
    return global.GVA_DB.Model(&model.CPEDevice{}).
        Where("device_id = ?", deviceID).
        Update("online_status", status).Error
}
```

## 6. 部署和集成

### 6.1 插件注册

在 `server/initialize/plugin.go` 中注册插件：

```go
func InstallPlugin(Router *gin.RouterGroup) {
    // ... 其他插件注册
    
    // 注册TR069-Adapter插件
    tr069AdapterPlug := tr069adapter.CreateTR069AdapterPlug()
    bizPluginV1.Use(tr069AdapterPlug.Register)
    
    // 注册插件停止函数
    global.GVA_PLUGINS = append(global.GVA_PLUGINS, tr069AdapterPlug)
}
```

### 6.2 配置文件更新

在 `server/config.yaml` 中添加TR069配置：

```yaml
# ... 其他配置

# TR069配置
tr069:
  server:
    port: 7547
    read_timeout: 30
    write_timeout: 30
  device:
    session_timeout: 300
    max_sessions: 1000
  parameter:
    batch_size: 100
    cache_ttl: 300
  log:
    level: "info"
    enable_soap_log: true
```

## 7. 监控和日志

### 7.1 性能监控

```go
// 添加Prometheus指标
var (
    tr069RequestsTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "tr069_requests_total",
            Help: "Total number of TR069 requests",
        },
        []string{"method", "status"},
    )
    
    tr069RequestDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "tr069_request_duration_seconds",
            Help: "TR069 request duration in seconds",
        },
        []string{"method"},
    )
)
```

### 7.2 结构化日志

```go
func LogTR069Operation(deviceID, operation string, duration time.Duration, err error) {
    fields := []zap.Field{
        zap.String("plugin", "tr069-adapter"),
        zap.String("device_id", deviceID),
        zap.String("operation", operation),
        zap.Duration("duration", duration),
    }
    
    if err != nil {
        fields = append(fields, zap.Error(err))
        global.GVA_LOG.Error("TR069 operation failed", fields...)
    } else {
        global.GVA_LOG.Info("TR069 operation completed", fields...)
    }
}
```

## 8. 总结

将TR069-Adapter设计为GVA插件具有以下优势：

1. **统一管理**：与GVA主服务一起部署和管理
2. **资源共享**：共享数据库、Redis、日志等基础设施
3. **配置统一**：使用GVA的配置管理系统
4. **监控集成**：集成到GVA的监控体系
5. **开发效率**：利用GVA的开发框架和工具

这种设计既保持了TR069-Adapter的独立性（使用独立的7547端口），又充分利用了GVA框架的优势，是一个理想的架构选择。