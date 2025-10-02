# TR069 架构设计文档

## 概述

本文档详细说明了TR069系统的架构设计，特别是`tr069-core`和`tr069-adapter`两个组件的职责分工和交互关系。此文档旨在帮助AI理解系统架构，确保在开发和维护过程中正确处理两个组件的关系。

## 架构概览

```
┌─────────────────────────────────────────────────────────────┐
│                    GVA Framework                            │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────────┐    ┌─────────────────────────────────┐ │
│  │  TR069-Adapter  │◄──►│         TR069-Core              │ │
│  │   (GVA Plugin)  │    │    (Independent Library)       │ │
│  └─────────────────┘    └─────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
```

## 组件详细说明

### TR069-Core (核心库)

**位置**: `server/plugin/tr069-core/`

**性质**: 独立的Go库，包含TR069协议的核心实现

**职责**:
1. **协议实现**: 实现完整的TR069/CWMP协议栈
2. **设备管理**: 提供设备连接、认证、会话管理
3. **消息处理**: 处理SOAP消息的编解码
4. **参数操作**: 实现GetParameterValues、SetParameterValues等RPC方法
5. **事件处理**: 处理设备事件和通知
6. **任务调度**: 管理设备任务队列和执行

**核心模块**:
- `interfaces/`: 定义核心接口和数据结构
- `factory/`: 工厂模式创建各种组件
- `internal/`: 内部实现，包括协议处理逻辑
- `pool/`: 连接池和资源管理
- `server/`: TR069服务器核心实现
- `pkg/`: 公共工具包

**特点**:
- 可独立运行和测试
- 不依赖GVA框架
- 提供标准的Go接口
- 支持插件化扩展

### TR069-Adapter (适配器插件)

**位置**: `server/plugin/tr069-adapter/`

**性质**: GVA框架插件，将TR069-Core集成到GVA系统中

**职责**:
1. **框架集成**: 将TR069-Core集成到GVA框架
2. **API暴露**: 提供RESTful API供前端调用
3. **数据持久化**: 将设备数据存储到GVA数据库
4. **权限控制**: 集成GVA的RBAC权限系统
5. **用户界面**: 提供设备管理的Web界面
6. **事件转换**: 将TR069事件转换为GVA系统事件

**GVA插件结构**:
```
tr069-adapter/
├── api/           # API控制器层
├── config/        # 配置定义
├── global/        # 全局变量
├── initialize/    # 初始化模块
├── model/         # 数据模型
├── router/        # 路由定义
├── service/       # 业务逻辑层
└── plugin.go      # 插件入口
```

## 依赖关系

### 模块路径
- **TR069-Core**: `github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-core`
- **TR069-Adapter**: `github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-adapter`

### 依赖方向
```
TR069-Adapter ──depends on──► TR069-Core
      │                           │
      │                           │
      ▼                           ▼
  GVA Framework              Independent
```

**重要**: TR069-Adapter依赖TR069-Core，但TR069-Core不依赖任何GVA组件

## 数据流

### 设备连接流程
1. 设备连接到TR069-Core服务器
2. TR069-Core处理协议握手和认证
3. TR069-Adapter监听TR069-Core事件
4. TR069-Adapter将设备信息存储到GVA数据库
5. 前端通过TR069-Adapter API查询设备状态

### 参数操作流程
1. 用户通过GVA前端发起参数操作请求
2. 请求到达TR069-Adapter API层
3. TR069-Adapter调用TR069-Core接口
4. TR069-Core与设备通信执行操作
5. 结果通过相同路径返回给用户

## 接口定义

### TR069-Core 主要接口

```go
// 设备管理接口
type DeviceManager interface {
    GetDevice(deviceID string) (*Device, error)
    ListDevices() ([]*Device, error)
    DeleteDevice(deviceID string) error
}

// 参数操作接口
type ParameterService interface {
    GetParameterValues(deviceID string, params []string) (map[string]interface{}, error)
    SetParameterValues(deviceID string, params map[string]interface{}) error
}

// 事件监听接口
type EventListener interface {
    OnDeviceConnect(device *Device)
    OnDeviceDisconnect(deviceID string)
    OnParameterChange(deviceID string, params map[string]interface{})
}
```

### TR069-Adapter 集成点

```go
// 在service层集成TR069-Core
type TR069Service struct {
    deviceManager    tr069core.DeviceManager
    parameterService tr069core.ParameterService
    // GVA相关服务
    db *gorm.DB
}

// 实现事件监听，将TR069事件转换为GVA事件
func (s *TR069Service) OnDeviceConnect(device *tr069core.Device) {
    // 将设备信息存储到GVA数据库
    gvaDevice := &model.TR069Device{
        DeviceID:   device.ID,
        ModelName:  device.ModelName,
        // ... 其他字段
    }
    s.db.Create(gvaDevice)
}
```

## 配置管理

### TR069-Core 配置
- 独立的配置文件或配置结构
- 包含TR069服务器端口、认证设置等
- 不依赖GVA配置系统

### TR069-Adapter 配置
- 集成到GVA配置系统
- 通过`config.yaml`或环境变量配置
- 包含数据库连接、API设置等

## 开发指南

### 修改TR069-Core时
1. 确保不引入GVA框架依赖
2. 保持接口的向后兼容性
3. 更新相关的接口文档
4. 在TR069-Core目录下进行单元测试

### 修改TR069-Adapter时
1. 遵循GVA插件开发规范
2. 确保正确处理TR069-Core的接口变更
3. 更新API文档和Swagger注释
4. 测试与GVA框架的集成

### 添加新功能时
1. **协议相关功能**: 在TR069-Core中实现
2. **业务逻辑功能**: 在TR069-Adapter中实现
3. **界面功能**: 在前端和TR069-Adapter API中实现

## 部署考虑

### 独立部署
- TR069-Core可以作为独立服务部署
- 适用于只需要TR069协议功能的场景

### 集成部署
- TR069-Adapter作为GVA插件部署
- 提供完整的设备管理界面和API
- 适用于需要完整管理系统的场景

## 故障排查

### 常见问题
1. **模块路径错误**: 确保使用正确的模块路径
2. **循环依赖**: TR069-Core不应依赖TR069-Adapter
3. **接口不匹配**: 检查TR069-Core接口变更是否在Adapter中正确处理

### 调试建议
1. 分别测试TR069-Core和TR069-Adapter
2. 检查事件监听器是否正确注册
3. 验证数据库连接和数据持久化
4. 查看日志文件定位问题

## 版本兼容性

### 版本策略
- TR069-Core使用语义化版本控制
- TR069-Adapter版本与GVA框架版本保持同步
- 主要版本变更时需要更新兼容性文档

### 升级指南
1. 优先升级TR069-Core
2. 测试接口兼容性
3. 更新TR069-Adapter以适配新接口
4. 进行集成测试

---

**注意**: 此文档应随着系统架构的变更及时更新，确保AI和开发人员始终了解最新的设计决策和实现细节。