# TR069-Adapter 插件

## 概述

TR069-Adapter 是一个基于 gin-vue-admin 框架开发的 TR069 协议适配器插件，提供完整的 CPE（Customer Premises Equipment）设备管理功能。

## 功能特性

### 核心功能
- **设备管理**: CPE设备的注册、配置、监控和管理
- **参数管理**: 设备参数的读取、设置和批量操作
- **会话管理**: TR069会话的创建、维护和监控
- **操作日志**: 完整的操作记录和审计功能

### 技术特性
- 完全遵循 TR069/CWMP 协议标准
- 支持多种 CPE 设备类型
- 高并发会话处理
- 实时设备状态监控
- 灵活的参数配置管理
- 完整的操作审计日志

## 目录结构

```
tr069-adapter/
├── api/                    # API控制器层
│   ├── enter.go           # API组入口
│   ├── device_api.go      # 设备管理API
│   ├── parameter_api.go   # 参数管理API
│   ├── session_api.go     # 会话管理API
│   └── operation_log_api.go # 操作日志API
├── model/                 # 数据模型层
│   ├── device.go         # 设备模型
│   ├── parameter.go      # 参数模型
│   ├── session.go        # 会话模型
│   ├── operation_log.go  # 操作日志模型
│   ├── request/          # 请求模型
│   └── response/         # 响应模型
├── service/              # 服务层
│   ├── enter.go         # 服务组入口
│   ├── device_service.go # 设备服务
│   ├── parameter_service.go # 参数服务
│   ├── session_service.go # 会话服务
│   └── operation_log_service.go # 操作日志服务
├── router/               # 路由层
│   ├── enter.go         # 路由组入口
│   ├── device_router.go # 设备路由
│   ├── parameter_router.go # 参数路由
│   ├── session_router.go # 会话路由
│   └── operation_log_router.go # 操作日志路由
├── initialize/           # 初始化层
│   ├── register.go      # 注册器
│   ├── gorm.go         # 数据库初始化
│   ├── router.go       # 路由初始化
│   ├── viper.go        # 配置初始化
│   ├── api.go          # API初始化
│   └── menu.go         # 菜单初始化
├── config.yaml          # 配置文件
├── plugin.go            # 插件入口
└── README.md           # 说明文档
```

## 安装配置

### 1. 插件安装

将插件目录放置到 GVA 项目的 `server/plugin/` 目录下：

```bash
cp -r tr069-adapter /path/to/gin-vue-admin/server/plugin/
```

### 2. 配置文件

编辑 `config.yaml` 文件，配置数据库、Redis、CWMP等相关参数：

```yaml
# 数据库配置
database:
  type: "mysql"
  host: "127.0.0.1"
  port: 3306
  database: "gva_tr069"
  username: "root"
  password: "your_password"

# CWMP配置
cwmp:
  connection_request_url: "http://localhost:7547/cwmp"
  auth:
    username: "admin"
    password: "admin123"
```

### 3. 数据库初始化

插件会自动创建所需的数据库表：
- `tr069_devices` - 设备信息表
- `tr069_parameters` - 参数信息表
- `tr069_sessions` - 会话信息表
- `tr069_operation_logs` - 操作日志表

### 4. 菜单初始化

插件会自动创建管理菜单：
- TR069适配器
  - 设备管理
  - 参数管理
  - 会话管理
  - 操作日志
  - 监控面板

## API 接口

### 设备管理 API

- `GET /tr069-adapter/device/list` - 获取设备列表
- `GET /tr069-adapter/device/:id` - 获取设备详情
- `POST /tr069-adapter/device/create` - 创建设备
- `PUT /tr069-adapter/device/update` - 更新设备
- `DELETE /tr069-adapter/device/delete` - 删除设备
- `POST /tr069-adapter/device/reboot` - 重启设备
- `POST /tr069-adapter/device/factoryReset` - 恢复出厂设置

### 参数管理 API

- `GET /tr069-adapter/parameter/list` - 获取参数列表
- `GET /tr069-adapter/parameter/tree` - 获取参数树
- `POST /tr069-adapter/parameter/setValue` - 设置参数值
- `POST /tr069-adapter/parameter/getValue` - 获取参数值
- `POST /tr069-adapter/parameter/addObject` - 添加对象
- `POST /tr069-adapter/parameter/deleteObject` - 删除对象

### 会话管理 API

- `GET /tr069-adapter/session/list` - 获取会话列表
- `GET /tr069-adapter/session/:id` - 获取会话详情
- `POST /tr069-adapter/session/end` - 结束会话
- `GET /tr069-adapter/session/active` - 获取活跃会话

### 操作日志 API

- `GET /tr069-adapter/operationLog/list` - 获取操作日志列表
- `GET /tr069-adapter/operationLog/:id` - 获取日志详情
- `GET /tr069-adapter/operationLog/statistics` - 获取统计信息

## 使用说明

### 设备管理

1. **设备注册**: CPE设备首次连接时会自动注册到系统
2. **设备监控**: 实时监控设备在线状态和基本信息
3. **设备操作**: 支持远程重启、恢复出厂设置等操作

### 参数管理

1. **参数读取**: 从CPE设备读取参数值
2. **参数设置**: 向CPE设备设置参数值
3. **批量操作**: 支持批量参数读取和设置
4. **参数树**: 以树形结构展示设备参数

### 会话管理

1. **会话监控**: 实时监控TR069会话状态
2. **会话控制**: 手动结束异常会话
3. **会话历史**: 查看设备的会话历史记录

### 操作日志

1. **操作记录**: 记录所有设备操作和参数变更
2. **审计功能**: 提供完整的操作审计轨迹
3. **日志分析**: 支持按设备、时间等维度分析

## 开发说明

### 扩展功能

如需扩展插件功能，请遵循以下步骤：

1. 在对应的 `model` 目录下添加数据模型
2. 在 `service` 层实现业务逻辑
3. 在 `api` 层添加接口处理
4. 在 `router` 层注册路由
5. 更新初始化文件

### 代码规范

- 严格遵循 GVA 框架的分层架构
- 使用统一的错误处理和响应格式
- 添加完整的 Swagger 注释
- 遵循 Go 语言编码规范

## 注意事项

1. **安全性**: 确保 CWMP 认证配置的安全性
2. **性能**: 合理配置会话并发数和超时时间
3. **日志**: 定期清理操作日志以避免数据库膨胀
4. **备份**: 定期备份设备配置和参数数据

## 版本历史

- v1.0.0: 初始版本，提供基础的TR069适配器功能

## 技术支持

如有问题或建议，请联系开发团队或提交 Issue。