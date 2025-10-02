# GVA-TR069 项目

基于 [gin-vue-admin](https://github.com/flipped-aurora/gin-vue-admin) 框架的 TR069/CWMP 协议实现，提供完整的 CPE 设备管理解决方案。

## 🚀 项目概述

GVA-TR069 是一个现代化的 TR069 设备管理系统，采用前后端分离架构，集成了完整的 TR069/CWMP 协议栈和 Web 管理界面。

### 核心特性

- 🔧 **完整的 TR069 协议支持** - 实现 TR069/CWMP 标准的所有核心功能
- 🌐 **现代化 Web 界面** - 基于 Vue 3 + Element Plus 的响应式管理界面  
- 🔐 **企业级权限管理** - 集成 GVA 的 RBAC 权限控制系统
- 📊 **实时设备监控** - 设备状态实时监控和参数管理
- 🔄 **自动化任务调度** - 支持批量操作和定时任务
- 📈 **数据可视化** - 设备数据图表展示和分析
- 🛡️ **高可用架构** - 支持集群部署和负载均衡

## 🏗️ 架构设计

### 系统架构图

```
┌─────────────────────────────────────────────────────────────┐
│                    GVA Framework                            │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────────┐    ┌─────────────────────────────────┐ │
│  │  TR069-Adapter  │◄──►│         TR069-Core              │ │
│  │   (GVA Plugin)  │    │    (Protocol Library)          │ │
│  └─────────────────┘    └─────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
                    ┌─────────────────┐
                    │   CPE Devices   │
                    │  (路由器/网关)   │
                    └─────────────────┘
```

### 组件说明

#### TR069-Core (协议核心库)
- **位置**: `server/plugin/tr069-core/`
- **职责**: TR069/CWMP 协议的完整实现
- **特性**: 
  - 独立的 Go 库，可单独使用
  - 支持所有标准 RPC 方法
  - 设备会话管理和连接池
  - 事件驱动的消息处理

#### TR069-Adapter (GVA 适配器)
- **位置**: `server/plugin/tr069-adapter/`
- **职责**: 将 TR069-Core 集成到 GVA 框架
- **特性**:
  - RESTful API 接口
  - 数据持久化和缓存
  - 权限控制和用户管理
  - Web 界面和仪表盘

## 📋 功能特性

### TR069 协议支持

- ✅ **设备发现和注册** - 自动发现和注册 CPE 设备
- ✅ **参数管理** - GetParameterValues / SetParameterValues
- ✅ **配置下发** - AddObject / DeleteObject
- ✅ **文件传输** - Download / Upload 文件操作
- ✅ **设备重启** - Reboot 远程重启设备
- ✅ **固件升级** - 远程固件更新和管理
- ✅ **诊断功能** - 网络诊断和故障排查

### 管理功能

- 📱 **设备管理** - 设备列表、详情、分组管理
- 📊 **参数监控** - 实时参数监控和历史数据
- 🔧 **配置模板** - 配置模板管理和批量下发
- 📋 **任务管理** - 任务队列、执行状态、结果查看
- 📈 **统计报表** - 设备统计、性能分析、趋势图表
- 🔔 **告警通知** - 设备异常告警和通知推送

## 🛠️ 技术栈

### 后端技术
- **Go 1.23** - 主要编程语言
- **Gin 1.10.0** - Web 框架
- **GORM 1.25.12** - ORM 框架
- **MySQL/PostgreSQL** - 数据库支持
- **Redis** - 缓存和会话存储
- **Casbin** - 权限管理

### 前端技术
- **Vue 3.5.7** - 前端框架
- **Element Plus 2.10.2** - UI 组件库
- **Vite 6.2.3** - 构建工具
- **Pinia 2.2.2** - 状态管理
- **ECharts 5.5.1** - 数据可视化

## 🚀 快速开始

### 环境要求

- Go 1.23+
- Node.js 18+
- MySQL 8.0+ / PostgreSQL 13+
- Redis 6.0+

### 安装步骤

1. **克隆项目**
```bash
git clone https://github.com/ddddddddwp/gva-tr069.git
cd gva-tr069
```

2. **后端配置**
```bash
cd server
cp config.yaml.example config.yaml
# 编辑 config.yaml 配置数据库连接信息
go mod tidy
go run main.go
```

3. **前端配置**
```bash
cd web
npm install
npm run dev
```

4. **访问系统**
- 前端地址: http://localhost:3000
- 后端API: http://localhost:8888
- Swagger文档: http://localhost:8888/swagger/index.html

### Docker 部署

```bash
# 使用 docker-compose 一键部署
docker-compose up -d
```

## 📖 使用指南

### 设备接入

1. **配置 CPE 设备**
   - ACS URL: `http://your-server:8888/tr069`
   - 用户名/密码: 在系统中配置

2. **设备注册**
   - 设备首次连接时自动注册
   - 在设备管理页面查看和管理

3. **参数配置**
   - 使用配置模板批量配置
   - 支持单设备参数修改

### API 使用

#### 获取设备列表
```bash
curl -X POST http://localhost:8888/api/tr069/getDeviceList \
  -H "Content-Type: application/json" \
  -d '{"page": 1, "pageSize": 10}'
```

#### 获取设备参数
```bash
curl -X POST http://localhost:8888/api/tr069/getParameters \
  -H "Content-Type: application/json" \
  -d '{"deviceId": "device123", "parameters": ["Device.DeviceInfo.ModelName"]}'
```

#### 设置设备参数
```bash
curl -X POST http://localhost:8888/api/tr069/setParameters \
  -H "Content-Type: application/json" \
  -d '{"deviceId": "device123", "parameters": {"Device.WiFi.SSID.1.SSID": "NewSSID"}}'
```

## 🔧 配置说明

### 主要配置项

```yaml
# config.yaml
tr069:
  server:
    port: 7547              # TR069 服务端口
    timeout: 30             # 连接超时时间
    max_connections: 1000   # 最大连接数
  
  database:
    host: localhost
    port: 3306
    username: root
    password: password
    database: gva_tr069
  
  redis:
    host: localhost
    port: 6379
    password: ""
    db: 0
```

### 设备配置模板

```json
{
  "templateName": "基础WiFi配置",
  "parameters": {
    "Device.WiFi.SSID.1.SSID": "MyWiFi",
    "Device.WiFi.SSID.1.Enable": true,
    "Device.WiFi.AccessPoint.1.Security.ModeEnabled": "WPA2-PSK"
  }
}
```

## 📊 监控和运维

### 系统监控

- **设备状态监控** - 在线/离线状态实时监控
- **性能指标** - CPU、内存、网络使用率
- **连接统计** - 并发连接数、请求响应时间
- **错误日志** - 系统错误和异常日志

### 日志管理

```bash
# 查看系统日志
tail -f logs/server.log

# 查看 TR069 协议日志
tail -f logs/tr069.log

# 查看错误日志
tail -f logs/error.log
```

## 🧪 测试

### 单元测试
```bash
cd server
go test ./...
```

### 集成测试
```bash
cd server
go test -tags=integration ./...
```

### 前端测试
```bash
cd web
npm run test
```

## 📚 开发文档

- [架构设计文档](TR069_ARCHITECTURE.md) - 详细的系统架构说明
- [API 文档](http://localhost:8888/swagger/index.html) - 完整的 API 接口文档
- [开发指南](docs/DEVELOPMENT.md) - 开发环境搭建和代码规范
- [部署指南](docs/DEPLOYMENT.md) - 生产环境部署说明

## 🤝 贡献指南

我们欢迎所有形式的贡献！请查看 [贡献指南](CONTRIBUTING.md) 了解详细信息。

### 开发流程

1. Fork 本仓库
2. 创建功能分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 创建 Pull Request

### 代码规范

- 遵循 Go 官方代码规范
- 使用 `gofmt` 格式化代码
- 添加必要的单元测试
- 更新相关文档

## 📄 许可证

本项目采用 [MIT 许可证](LICENSE)。

## 🆘 支持和帮助

### 获取帮助

- 📖 [文档中心](docs/) - 查看详细文档
- 🐛 [问题反馈](https://github.com/ddddddddwp/gva-tr069/issues) - 报告 Bug 或提出建议
- 💬 [讨论区](https://github.com/ddddddddwp/gva-tr069/discussions) - 技术讨论和交流

### 常见问题

**Q: 设备无法连接到 ACS？**
A: 检查网络连接、防火墙设置和 ACS URL 配置。

**Q: 参数设置失败？**
A: 确认参数路径正确，设备支持该参数，且有足够权限。

**Q: 系统性能问题？**
A: 检查数据库连接池配置、Redis 缓存设置和系统资源使用情况。

## 🔗 相关链接

- [gin-vue-admin](https://github.com/flipped-aurora/gin-vue-admin) - 基础框架
- [TR069 标准](https://www.broadband-forum.org/technical/download/TR-069.pdf) - 协议规范
- [CWMP 数据模型](https://cwmp-data-models.broadband-forum.org/) - 数据模型参考

## 📈 项目状态

![GitHub stars](https://img.shields.io/github/stars/ddddddddwp/gva-tr069)
![GitHub forks](https://img.shields.io/github/forks/ddddddddwp/gva-tr069)
![GitHub issues](https://img.shields.io/github/issues/ddddddddwp/gva-tr069)
![GitHub license](https://img.shields.io/github/license/ddddddddwp/gva-tr069)

---

**维护者**: [@ddddddddwp](https://github.com/ddddddddwp)  
**最后更新**: 2024年10月2日