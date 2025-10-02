# TR069 系统架构设计文档

## 1. 系统概述

TR069 系统采用三层架构设计，将TR069协议处理、设备通信和用户管理功能分为三个独立的组件：

- **TR069-Core**: 核心库，提供TR069协议解析、构建和处理的基础功能
- **TR069-Adapter**: GVA插件，负责与CPE设备通信，调用TR069-Core处理协议
- **TR069-Management**: GVA插件，负责数据查询和用户交互界面

## 2. 架构设计图

```
┌─────────────────┐    7547端口     ┌──────────────────┐      调用接口     ┌──────────────────┐
│   CPE 设备      │ ◄──────────────► │  TR069-Adapter   │◄──────────────────►│   TR069-Core     │
│                 │    SOAP/HTTP     │  GVA插件         │                   │   核心库         │
└─────────────────┘                  └──────────────────┘                   └──────────────────┘
                                              │
                                              │ 数据存储
                                              ▼
                                     ┌──────────────────┐
                                     │   数据库          │
                                     │   (MySQL/PG)     │
                                     └──────────────────┘
                                              ▲
                                              │ 数据查询
                                              │
┌─────────────────┐    HTTP API     ┌──────────────────┐
│   前端用户      │ ◄──────────────► │ TR069-Management │
│   (Web界面)     │    8888端口      │  GVA插件         │
└─────────────────┘                  └──────────────────┘
```

## 3. 组件详细设计

### 3.1 TR069-Core (核心库)

#### 职责范围
- **提供TR069协议的核心实现**
- 解析和构建TR069/CWMP消息
- 处理TR069会话和事件
- 提供接口供适配器调用
- 不直接与设备或数据库交互

#### 技术架构
```
TR069-Core 库
├── 接口层 (interfaces/)
│   ├── API接口
│   ├── 构建器接口
│   ├── 解析器接口
│   ├── 会话接口
│   └── 事件接口
├── 实现层 (internal/)
│   ├── 消息解析器
│   ├── 消息构建器
│   ├── 会话管理
│   ├── 事件处理
│   └── RPC方法实现
└── 工具层
    ├── 日志系统
    ├── 缓存系统
    ├── 错误处理
    └── 配置管理
```

#### 核心功能模块
1. **消息解析与构建**
   - TR069 SOAP消息解析
   - TR069 SOAP消息构建
   - XML流处理

2. **会话管理**
   - 会话状态跟踪
   - 会话超时处理
   - 会话恢复机制

3. **事件系统**
   - 事件发布/订阅
   - 事件处理回调
   - 事件优先级管理

### 3.2 TR069-Adapter (GVA插件)

#### 职责范围
- **负责与CPE设备通信**
- 监听7547端口，处理CPE的SOAP请求
- 调用TR069-Core进行协议处理
- 将CPE数据存储到数据库
- 提供设备管理API

#### 技术架构
```
TR069-Adapter 插件
├── API层 (api/)
│   ├── 设备API
│   ├── 参数API
│   ├── 会话API
│   └── 操作日志API
├── 服务层 (service/)
│   ├── 设备服务
│   ├── 参数服务
│   ├── 会话服务
│   ├── 操作日志服务
│   └── TR069桥接服务
├── 数据模型层 (model/)
│   ├── 设备模型
│   ├── 参数模型
│   ├── 会话模型
│   └── 操作日志模型
├── 路由层 (router/)
│   ├── 设备路由
│   ├── 参数路由
│   ├── 会话路由
│   └── 操作日志路由
└── TR069服务器 (tr069server/)
    ├── SOAP处理器
    ├── 会话管理
    └── CWMP类型定义
```

#### 核心功能模块
1. **SOAP服务器**
   - 监听7547端口
   - 处理CPE的HTTP/SOAP请求
   - 支持基本认证和摘要认证

2. **设备管理**
   - 设备注册和发现
   - 设备状态监控
   - 设备操作(重启、恢复出厂设置等)

3. **参数管理**
   - 参数值获取
   - 参数值设置
   - 参数历史记录

4. **会话管理**
   - 会话创建和跟踪
   - 会话状态维护
   - 会话超时处理

5. **操作日志**
   - 设备操作记录
   - 参数变更记录
   - 系统事件记录

#### 配置示例
```yaml
# tr069-adapter.yaml
server:
  port: 7547
  host: "0.0.0.0"
  
database:
  host: "localhost"
  port: 3306
  dbname: "tr069_data"
  username: "tr069_user"
  password: "password"

tr069:
  session_timeout: 300
  max_envelopes: 1
  connection_request_url: "http://acs.example.com:7547"
```

### 3.3 TR069-Management (GVA插件)

#### 职责范围
- **专门负责数据查询和用户交互**
- 提供Web管理界面
- 查询和展示CPE设备信息
- 配置管理和监控功能
- 集成到GVA权限系统

#### 技术架构
```
TR069-Management 插件
├── API层 (GVA集成)
│   ├── 设备查询API
│   ├── 参数查询API
│   ├── 配置管理API
│   └── 统计报表API
├── 服务层
│   ├── 设备管理服务
│   ├── 参数管理服务
│   ├── 配置管理服务
│   └── 报表统计服务
├── 数据模型层
│   ├── 设备信息模型
│   ├── 参数值模型
│   ├── 配置模板模型
│   └── 操作日志模型
└── 前端界面
    ├── 设备列表页面
    ├── 设备详情页面
    ├── 参数配置页面
    └── 监控仪表盘
```

#### 核心功能模块
1. **设备管理**
   - 设备列表查询和筛选
   - 设备详细信息展示
   - 设备状态监控

2. **参数管理**
   - 参数树结构展示
   - 参数值查询和历史记录
   - 批量参数配置

3. **配置管理**
   - 配置模板管理
   - 批量配置下发
   - 配置任务跟踪

4. **监控报表**
   - 设备在线状态统计
   - 参数变化趋势图
   - 操作日志查询

## 4. 数据库设计

### 4.1 核心数据表

```sql
-- CPE设备信息表
CREATE TABLE cpe_devices (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    device_id VARCHAR(255) UNIQUE NOT NULL,
    manufacturer VARCHAR(100),
    oui VARCHAR(6),
    product_class VARCHAR(100),
    serial_number VARCHAR(100),
    hardware_version VARCHAR(50),
    software_version VARCHAR(50),
    connection_request_url VARCHAR(500),
    last_inform_time TIMESTAMP,
    online_status TINYINT DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

-- 参数信息表
CREATE TABLE cpe_parameters (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    device_id VARCHAR(255) NOT NULL,
    parameter_name VARCHAR(500) NOT NULL,
    parameter_value TEXT,
    parameter_type VARCHAR(50),
    writable TINYINT DEFAULT 0,
    last_updated TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_device_param (device_id, parameter_name)
);

-- 操作日志表
CREATE TABLE cpe_operation_logs (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    device_id VARCHAR(255) NOT NULL,
    operation_type VARCHAR(50) NOT NULL,
    operation_data JSON,
    result_status VARCHAR(20),
    error_message TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_device_time (device_id, created_at)
);

-- 会话记录表
CREATE TABLE cpe_sessions (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    session_id VARCHAR(100) UNIQUE NOT NULL,
    device_id VARCHAR(255) NOT NULL,
    session_type VARCHAR(50),
    start_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    end_time TIMESTAMP NULL,
    status VARCHAR(20) DEFAULT 'active',
    INDEX idx_device_session (device_id, session_id)
);
```

## 5. 接口规范

### 5.1 TR069-Core 接口

TR069-Core 提供以下核心接口供 TR069-Adapter 调用：

```go
// 构建器接口
type Builder interface {
    BuildInform(deviceInfo DeviceInfo, events []Event) ([]byte, error)
    BuildGetParameterValuesResponse(parameters []ParameterValueStruct) ([]byte, error)
    BuildSetParameterValuesResponse(status int) ([]byte, error)
    // 其他消息构建方法...
}

// 解析器接口
type Parser interface {
    ParseInform(data []byte) (*InformRequest, error)
    ParseGetParameterValues(data []byte) (*GetParameterValuesRequest, error)
    ParseSetParameterValues(data []byte) (*SetParameterValuesRequest, error)
    // 其他消息解析方法...
}

// 会话接口
type Session interface {
    NewSession(deviceID string) (string, error)
    GetSession(sessionID string) (SessionData, error)
    UpdateSession(sessionID string, data SessionData) error
    CloseSession(sessionID string) error
}

// API处理接口
type APIHandler interface {
    HandleInform(inform *InformRequest) (*InformResponse, error)
    HandleGetParameterValues(req *GetParameterValuesRequest) (*GetParameterValuesResponse, error)
    HandleSetParameterValues(req *SetParameterValuesRequest) (*SetParameterValuesResponse, error)
    // 其他API处理方法...
}
```

### 5.2 TR069-Adapter API接口

```go
// 设备管理接口
GET    /api/v1/tr069/devices              // 获取设备列表
GET    /api/v1/tr069/devices/{id}         // 获取设备详情
POST   /api/v1/tr069/devices/reboot       // 重启设备
POST   /api/v1/tr069/devices/factoryReset // 恢复出厂设置

// 参数管理接口
GET    /api/v1/tr069/parameters/{deviceId}           // 获取设备参数
POST   /api/v1/tr069/parameters/{deviceId}/get       // 获取指定参数
POST   /api/v1/tr069/parameters/{deviceId}/set       // 设置指定参数

// 会话管理接口
GET    /api/v1/tr069/sessions/{deviceId}             // 获取设备会话
DELETE /api/v1/tr069/sessions/{sessionId}            // 关闭会话

// 操作日志接口
GET    /api/v1/tr069/logs/{deviceId}                 // 获取设备操作日志
```

### 5.3 TR069-Management API接口

```go
// 设备管理接口
GET    /api/v1/tr069-mgmt/devices              // 获取设备列表
GET    /api/v1/tr069-mgmt/devices/{id}         // 获取设备详情
POST   /api/v1/tr069-mgmt/devices/search       // 设备搜索
PUT    /api/v1/tr069-mgmt/devices/{id}/config  // 更新设备配置

// 参数管理接口
GET    /api/v1/tr069-mgmt/parameters/{deviceId}           // 获取设备参数
POST   /api/v1/tr069-mgmt/parameters/{deviceId}/batch     // 批量设置参数
GET    /api/v1/tr069-mgmt/parameters/{deviceId}/history   // 参数历史记录

// 监控接口
GET    /api/v1/tr069-mgmt/statistics/devices    // 设备统计信息
GET    /api/v1/tr069-mgmt/statistics/online     // 在线状态统计
GET    /api/v1/tr069-mgmt/logs/{deviceId}       // 操作日志查询
```

## 6. 部署架构

### 6.1 单服务器部署模式

```
┌─────────────────────────────────────────────────────────────────┐
│                           服务器                                │
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │                    GVA 主应用                           │    │
│  │                                                         │    │
│  │  ┌─────────────────┐  ┌───────────────┐  ┌───────────┐  │    │
│  │  │ TR069-Adapter   │  │ TR069-Mgmt    │  │ 其他插件   │  │    │
│  │  │ (7547端口)      │  │ (Web界面)     │  │           │  │    │
│  │  └─────────────────┘  └───────────────┘  └───────────┘  │    │
│  │            │                   │               │        │    │
│  │            └─────────┬─────────┴───────────────┘        │    │
│  │                      │                                  │    │
│  │              ┌───────────────┐                          │    │
│  │              │   TR069-Core  │                          │    │
│  │              │   核心库      │                          │    │
│  │              └───────────────┘                          │    │
│  │                      │                                  │    │
│  └──────────────────────┼──────────────────────────────────┘    │
│                         │                                       │
│                 ┌───────────────┐                               │
│                 │   数据库      │                               │
│                 │  (MySQL)      │                               │
│                 └───────────────┘                               │
└─────────────────────────────────────────────────────────────────┘
```

### 6.2 容器化部署

```yaml
# docker-compose.yml
version: '3.8'
services:
  gva-server:
    build: ./gin-vue-admin/server
    ports:
      - "8888:8888"
      - "7547:7547"  # TR069-Adapter端口
    environment:
      - DB_HOST=mysql
      - DB_PORT=3306
    volumes:
      - ./plugins:/app/plugin
    depends_on:
      - mysql

  mysql:
    image: mysql:8.0
    environment:
      - MYSQL_ROOT_PASSWORD=password
      - MYSQL_DATABASE=gva_db
    volumes:
      - mysql_data:/var/lib/mysql

volumes:
  mysql_data:
```

## 7. 开发计划

### 7.1 第一阶段：TR069-Core库开发
1. 实现核心接口定义
2. 实现TR069消息解析和构建
3. 实现会话管理和事件系统
4. 完成单元测试和性能测试
5. 编写集成指南和API文档

### 7.2 第二阶段：TR069-Adapter插件开发
1. 创建GVA插件结构
2. 实现7547端口SOAP服务器
3. 集成TR069-Core功能
4. 实现数据库存储逻辑
5. 完成CPE设备通信测试

### 7.3 第三阶段：TR069-Management插件开发
1. 创建GVA插件结构
2. 实现数据查询API
3. 开发前端管理界面
4. 集成权限控制系统
5. 完成功能测试

### 7.4 第四阶段：系统集成和优化
1. 完善数据同步机制
2. 性能优化和监控
3. 安全加固
4. 文档完善
5. 生产环境部署

## 8. 优势分析

### 8.1 职责分离清晰
- **TR069-Core**: 专注协议处理，可独立升级
- **TR069-Adapter**: 专注设备通信，性能优化
- **TR069-Management**: 专注用户交互，功能丰富

### 8.2 技术架构优势
- 模块化设计，组件可独立升级
- 接口驱动，实现与接口分离
- 高性能协议处理
- 可扩展性强

### 8.3 开发维护优势
- 团队分工明确
- 代码耦合度低
- 测试独立进行
- 版本发布灵活

## 9. 注意事项

### 9.1 数据一致性
- 确保数据库事务完整性
- 实现数据同步机制
- 处理并发访问问题

### 9.2 性能考虑
- TR069-Adapter需要高并发处理能力
- 数据库查询优化
- 缓存策略设计

### 9.3 安全要求
- CPE认证机制
- API访问控制
- 数据传输加密
- 日志审计功能

---

**总结**: 这种三层架构设计能够很好地解决职责混淆问题，提高系统的可维护性和扩展性。TR069-Core专注于协议处理，TR069-Adapter专注于设备通信，TR069-Management专注于数据管理，各司其职，协同工作。