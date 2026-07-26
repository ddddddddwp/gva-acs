## Why

当前 GVA 会为设备自动生成并通过 TR-069 下发 Connection Request 用户名和密码，文件上传也会把共享凭据写入 Upload RPC。这覆盖了基站侧由用户维护凭据的职责，并扩大了密码进入命令、XML 和日志链路的风险。系统需要改为“用户分别配置、GVA 只在真实协议交互中校验”的认证模型。

## What Changes

- 新增 Connection、LOG、PM、MR 四套全局凭据配置，用户名和密码必须同时为空或同时非空。
- 新增可复用的 Basic/Digest 认证核心、加密凭据存储、版本化验证状态和不含秘密的审计记录。
- Connection 凭据默认同时供设备到 ACS 的 CWMP 接入认证和 ACS 到设备的 Connection Request 使用，但两个通道独立记录验证结果。
- 在 CWMP Handler 前增加设备接入认证；认证失败的 Inform 不解析、不注册、不更新设备。
- 移除自动生成、自动调度和自动下发 Connection Request 凭据的行为；历史自动凭据保留为只读审计数据。
- Connection Request 只在实际请求时使用用户配置的共享凭据，并自动兼容 Basic/Digest challenge。
- 新增全局凭据管理页面和设备认证状态展示；密码不回显，也不提供脱离真实协议动作的手动校验。
- **BREAKING**：`autoProvisionCredentials` 不再启用凭据下发；旧 YAML 文件上传凭据不再作为认证事实来源。

## Capabilities

### New Capabilities

- `tr069-credential-management`: 定义四套全局凭据、通道映射、原子保存、加密存储、版本轮换、状态和审计契约。
- `tr069-cwmp-access-authentication`: 定义 CWMP 设备接入认证、失败阻断、Connection Request 共享凭据和旧自动配置停用行为。

### Modified Capabilities

- 无。

## Impact

- 后端：TR-069 配置、模型、迁移、仓储、凭据服务、Basic/Digest 校验、CWMP 路由中间件、Connection Request 客户端、Inform 注册链路和安全日志。
- 前端：全局凭据管理页、设备详情认证状态、密码更新与清除交互。
- 数据库：新增凭据配置、认证通道映射、设备通道状态和认证审计表；旧连接配置表保留但停止参与运行时解析。
- 部署：需要独立主加密密钥；Basic 在生产环境必须由 HTTPS 保护。
- 测试：认证协议向量、未认证注册阻断、凭据轮换、旧数据迁移、脱敏和两台 BS Docker 的真实交互。
