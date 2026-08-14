## Why

现有 TR-069 文件通道只启用了固定地址 `/acs/log`，PM 和 MR 虽然已有配置骨架，但没有独立认证、接收、任务和管理能力。基站已经提供本地 PM/MR URL、用户名、密码和周期参数，GVA 应在不修改这些设备参数的前提下接收并校验定时上传。

## What Changes

- 在现有 7458 文件入口启用固定路径 `PUT/POST /acs/pm[/*filename]` 和 `PUT/POST /acs/mr[/*filename]`。
- PM、MR分别使用全局 PM、MR凭据，支持用户名和密码同时为空的无认证模式以及 Basic/Digest 自动兼容。
- 复用最近成功 Inform 的来源 IP绑定解析设备；没有唯一已注册设备时返回 `403`。
- PM、MR上传只产生 `PERIODIC` 传输任务，不提供主动 Upload RPC或“立即上传”操作。
- 复用 LOG 的流式限长、并发控制、对象存储、SHA-256、一致性恢复、保留期和安全审计能力。
- 增加 PM/MR 文件查询和权限控制，不解析文件内部业务内容。
- GVA 不读取、设置或下发 `Device.FAP.PerfMgmt.*` 与 `Device.FAP.MRMgmt.*` 的 URL、用户名、密码或周期参数。

## Capabilities

### New Capabilities

- `tr069-pm-mr-file-ingress`: 定义 PM/MR 固定文件入口、独立凭据、设备解析、周期任务、制品管理和权限行为。

### Modified Capabilities

- 无。

## Impact

- 后端：文件通道注册、认证 Provider、传输任务分类、制品查询 API、权限初始化和配置校验。
- 前端：PM、MR 文件管理入口、列表、筛选和下载操作。
- 数据库与对象存储：复用现有传输表，以 channel 区分 LOG/PM/MR，并使用独立存储前缀和保留策略。
- 基站：用户继续人工配置固定地址 `/acs/pm`、`/acs/mr` 及本地凭据；无需 URL 迁移。
- 测试：固定路径隔离、跨模块凭据拒绝、IP 歧义拒绝、流式上传和两台 BS Docker 周期上传验收。
