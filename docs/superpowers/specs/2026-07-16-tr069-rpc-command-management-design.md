# TR-069 主动命令与 RPC 记录设计

**日期：** 2026-07-16  
**状态：** 已完成交互与技术设计确认，待书面规格复核  
**范围：** GVA TR-069 插件、独立维护的 `tr069-core-only`、设备快捷操作、命令编排、RPC 记录与完整 CWMP XML 追踪

## 1. 目标

在 GVA 中完整支持当前 BS 通过 `GetRPCMethods` 声明的 12 种 ACS 主动下发方法，并提供一致、可追踪、可恢复的执行流程。

目标包括：

1. 设备列表“更多”菜单使用具体功能名称，不使用抽象的“RPC 下发”入口。
2. 每种功能使用专用表单、类型化请求结构和独立参数校验。
3. 所有功能接入 GVA API、菜单、按钮和 Casbin 权限体系。
4. 新增独立“RPC 记录”页面，追踪排队、构造、发送、响应、传输完成、失败和超时。
5. 同时保存结构化参数、结构化结果、完整发送 XML 和完整接收 XML。
6. 同一设备严格按提交顺序逐条执行，不同设备允许并行。
7. 数据库作为命令状态的事实来源，Redis 仅用于设备队列、锁和唤醒。
8. `tr069-core-only` 继续只负责 CWMP 协议构造、序列化、解析和关联，不承担 GVA 权限或页面业务。

## 2. 当前设备能力与实现差距

当前 BS 实际返回 12 种方法：

1. `GetRPCMethods`
2. `GetParameterValues`
3. `GetParameterNames`
4. `SetParameterValues`
5. `GetParameterAttributes`
6. `SetParameterAttributes`
7. `Reboot`
8. `AddObject`
9. `DeleteObject`
10. `Download`
11. `Upload`
12. `FactoryReset`

现有实现分为三类：

- GVA 已有接口：`GetRPCMethods`、`GetParameterValues`、`SetParameterValues`。
- core 已有 executor、GVA 尚未完整开放：`GetParameterNames`、`Reboot`、`Download`、`Upload`。
- core 有初步 executor 但 GVA 未注册：`GetParameterAttributes`、`SetParameterAttributes`。
- BS 声明支持但 core 缺少完整 executor：`AddObject`、`DeleteObject`、`FactoryReset`。

现有 `tr069_commands` 已能保存基础命令状态，但缺少统一编排、完整状态时间线、阶段截止时间、重试关联、结构化结果、报文留存和 TransferComplete 生命周期。

## 3. 架构边界

### 3.1 GVA 侧

GVA 负责：

- 接收 12 种类型化 API 请求。
- 执行 GVA 权限、设备能力、设备在线状态和参数校验。
- 应用危险操作确认策略。
- 创建命令主记录和初始事件。
- 维护设备级 FIFO 顺序、阶段状态与截止时间。
- 将可执行命令提交给 core。
- 保存结构化结果、完整 XML、故障和 TransferComplete。
- 提供记录列表、详情和重新下发接口。
- 执行超时扫描、服务重启恢复和 XML 到期清理。

### 3.2 tr069-core-only 侧

core 负责：

- executor 注册和 RPC 参数到 CWMP Message 的转换。
- CWMP XML 序列化和响应解析。
- CWMP ID、Command ID、CommandKey 与请求响应的关联。
- 在准确的发送和接收边界向 GVA sink 输出完整 XML 与协议事件。

core 不直接依赖 GVA 数据库、Redis、Casbin 或 Web 模型。core 代码继续由独立子项目维护，目录 `server/plugin/tr069/lib/tr069-core-only` 保持 Git 忽略。

### 3.3 命令编排器

GVA 增加统一命令编排器和方法注册表。每个方法定义：

- GVA 功能名称。
- core operation 名称。
- 类型化请求与校验器。
- 所需能力名称。
- 权限组。
- 危险等级和确认策略。
- 结果解析策略。
- 是否需要等待 TransferComplete。

前端使用专用表单，后端使用独立类型化路由，但公共状态机、队列、存储和错误处理全部复用编排器。

## 4. 设备入口与 GVA 页面

### 4.1 设备列表“更多”菜单

保持现有设备页结构和 GVA/Element Plus 视觉风格。“更多”菜单按以下分组展示：

```text
查询
  查询设备能力
  获取参数
  获取参数名称
  获取参数属性

配置
  配置参数
  配置参数属性
  添加对象
  删除对象

文件
  下载文件
  上传文件

设备维护
  重启设备
  恢复出厂设置
```

菜单和按钮不使用“RPC 方式”“通用 RPC”作为用户可见功能名称。离线设备的全部下发功能禁用，后端同步拒绝离线设备提交。

### 4.2 专用表单

具体功能通过 Element Plus Drawer 打开，保留设备列表上下文。每个 Drawer 只展示该方法需要的字段，提交成功后立即提示“命令已提交”，不在表单中阻塞等待设备结果。

危险确认分级：

- 查询类直接提交。
- 配置参数、配置属性、添加对象、下载、上传使用普通提交确认。
- 删除对象、重启设备、恢复出厂设置使用二次确认。
- 恢复出厂设置不要求输入设备序列号。

### 4.3 RPC 记录页面

在 TR-069 菜单下新增独立“RPC 记录”页面，使用现有 GVA 搜索表单、白色卡片、Element 表格、标签、抽屉和分页风格。

列表支持：

- 设备、具体功能、状态、Command ID、时间范围筛选。
- 状态标签、提交时间、阶段更新时间、完成时间展示。
- 查看详情。
- 对失败或超时记录重新下发。

详情支持：

- 设备与命令基本信息。
- 结构化请求参数与结构化结果。
- 状态时间线。
- 完整发送 XML 与完整接收 XML。
- CWMP ID、Command ID、CommandKey。
- FaultCode、FaultString、传输故障和阶段错误。
- 原命令与重试命令的双向关联。

## 5. 12 种功能映射

| 页面功能名称 | CWMP 方法 | 表单输入 | 结构化成功结果 | 权限组 |
|---|---|---|---|---|
| 查询设备能力 | GetRPCMethods | 无 | 方法列表 | 查询 |
| 获取参数 | GetParameterValues | 一个或多个参数路径 | 参数名、类型、值 | 查询 |
| 获取参数名称 | GetParameterNames | 对象路径、NextLevel | 参数名、是否可写 | 查询 |
| 获取参数属性 | GetParameterAttributes | 一个或多个参数名 | 通知级别、访问列表 | 查询 |
| 配置参数 | SetParameterValues | ParameterKey、参数名、类型、值 | Status | 配置 |
| 配置参数属性 | SetParameterAttributes | 属性变更项 | Status | 配置 |
| 添加对象 | AddObject | ObjectName、ParameterKey | InstanceNumber、Status | 配置 |
| 删除对象 | DeleteObject | 完整实例 ObjectName、ParameterKey | Status | 配置 |
| 下载文件 | Download | CommandKey、文件类型、URL、凭据、文件大小、目标文件名、延迟、成功/失败 URL | 接受状态、TransferComplete | 文件传输 |
| 上传文件 | Upload | CommandKey、文件类型、URL、凭据、延迟 | 接受状态、TransferComplete | 文件传输 |
| 重启设备 | Reboot | 可选 CommandKey | RebootResponse | 设备维护 |
| 恢复出厂设置 | FactoryReset | 无 | FactoryResetResponse | 设备维护 |

补充约束：

- 参数名可从现有参数树选择，也可手工输入。
- 下发前必须检查设备最近保存的能力列表；未声明支持的方法直接拒绝。
- `SetParameterAttributes` 的 core 模型补齐标准字段 `NotificationChange` 和 `AccessListChange`。
- Download/Upload 自动生成长度合规且唯一的 CommandKey；用户输入仅作为可选业务标识，实际关联键由系统保证唯一。
- AddObject、DeleteObject 和 FactoryReset 在独立 core 项目补齐 executor、XML 构造、响应匹配和测试。
- 现有 GPV、SPV、GetRPCMethods API 保持兼容，内部迁移到统一编排器。

## 6. GVA 权限

所有新路由登记到 GVA API 管理和 Casbin，不另建插件权限系统。

权限分为：

1. 查询：设备能力、参数值、参数名称、参数属性。
2. 配置：配置参数、配置属性、添加对象、删除对象。
3. 文件传输：下载、上传。
4. 设备维护：重启、恢复出厂。
5. RPC 记录：列表、详情。

重新下发同时要求记录详情权限和原功能执行权限。完整 XML 仅对具备记录详情权限的用户展示。

### 6.1 API 形态

命令接口保持 `/tr069/command/:deviceId/` 前缀，每种方法使用独立路由：

- `getRPCMethods`
- `getParameterValues`
- `getParameterNames`
- `getParameterAttributes`
- `setParameterValues`
- `setParameterAttributes`
- `addObject`
- `deleteObject`
- `download`
- `upload`
- `reboot`
- `factoryReset`

所有命令接口统一异步返回 `commandId` 和初始状态，不在 HTTP 请求内等待设备结果。现有 GPV 的同步等待逻辑迁移为异步提交，结果统一从 RPC 记录详情读取。

记录接口使用：

- `GET /tr069/command-record/list`：分页和筛选。
- `GET /tr069/command-record/:commandId`：主记录、结果、事件和 XML 详情。
- `POST /tr069/command-record/:commandId/retry`：按原参数创建新记录。

不提供接收任意 operation 和任意 JSON 的通用执行接口。

## 7. 状态机与 FIFO

### 7.1 状态

| 状态 | 中文 | 含义 |
|---|---|---|
| QUEUED | 排队中 | 等待同设备前面的命令进入终态 |
| WAITING_DEVICE | 等待设备连接 | 已到队首，等待可用 CWMP 会话 |
| BUILDING | 构造报文 | core 正在构造请求 |
| SENT | 已发送 | XML 已发送，等待匹配响应 |
| WAITING_TRANSFER | 等待传输完成 | Download/Upload 已接受，等待 TransferComplete |
| COMPLETED | 已完成 | 收到匹配的成功响应或成功 TransferComplete |
| FAILED | 执行失败 | 构造、队列、发送、CWMP Fault 或传输失败 |
| TIMEOUT | 已超时 | 任一执行阶段超过对应截止时间 |

最终状态为 `COMPLETED`、`FAILED`、`TIMEOUT`。已创建的命令必须进入其中一个最终状态。

### 7.2 提交与顺序

- 离线设备禁止创建命令。
- 同一设备严格 FIFO，前一条进入最终状态后下一条才可执行。
- 不同设备的命令可以并行。
- 排队时间不计入等待设备连接超时。
- 命令到达队首并进入 `WAITING_DEVICE` 时才生成等待截止时间。
- Download/Upload 在 TransferComplete 前不进入最终状态，因此会继续阻塞同设备后续命令；这是已确认的严格顺序语义。

### 7.3 重试

失败和超时记录提供“重新下发”。重试操作复制原始结构化参数，创建新的 Command ID 和新记录，并使用 `retryOf` 关联原记录。原记录的状态、XML 和时间线不得被覆盖。

## 8. 配置与超时

配置统一放在 GVA 当前生效 YAML 的 `tr069:` 节点：

- 仓库模板：`server/config.yaml`。
- 本地开发：`server/config.local.yaml`，由 `go run . -c config.local.yaml` 加载且保持 Git 忽略。

新增配置以秒和天为单位：

```yaml
tr069:
  commandQueueWaitTimeout: 180
  rpcResponseTimeout: 90
  transferCompleteTimeout: 43200
  rpcXMLRetentionDays: 30
```

含义：

- `commandQueueWaitTimeout`：命令到达队首后等待设备 CWMP 会话，默认 3 分钟。
- `rpcResponseTimeout`：请求 XML 发出后等待普通 RPC Response，默认 90 秒。
- `transferCompleteTimeout`：Download/Upload 被接受后等待 TransferComplete，默认 12 小时。
- `rpcXMLRetentionDays`：完整收发 XML 保存时间，默认 30 天。

三类超时独立配置。配置热加载后影响新进入相应阶段的命令；已写入数据库的阶段截止时间保持不变，避免运行中命令的期限因配置变化而漂移。

现有 `commandQueueImmediateTTL` 仅作为旧版本兼容项。在新配置存在时，命令业务超时完全由 `commandQueueWaitTimeout` 和数据库截止时间控制，不能再依赖 Redis key 过期推断命令状态。

## 9. 数据模型

### 9.1 命令主记录

扩展 `tr069_commands`，至少保存：

- Command ID、Device ID、Device Key、operation、用户可见功能名称。
- 结构化请求 JSON、结构化结果 JSON。
- 当前状态、当前阶段、阶段错误。
- CWMP ID、CommandKey、DedupKey。
- `retryOf`。
- 排队、到达队首、发送、响应、传输完成和最终完成时间。
- 当前阶段截止时间。
- FaultCode、FaultString、传输故障码和故障文本。

### 9.2 命令事件

新增命令事件表，按时间追加状态变化和关键动作。事件不可覆盖，用于恢复时间线和审计重复响应。

### 9.3 XML 报文

新增命令报文表，保存方向、CWMP ID、HTTP/CWMP 元数据、完整原始 XML、创建时间和到期时间。报文到期清理不影响命令主记录、结构化结果和事件时间线。

### 9.4 能力表

继续使用 `tr069_device_rpc_methods`。成功完成“查询设备能力”后更新该设备能力列表和采集时间。

## 10. 可靠性与恢复

- 数据库是命令状态的事实来源，Redis 只承担设备队列、锁和唤醒。
- 创建命令与初始事件在同一数据库事务中完成。
- Redis 入队失败必须记录失败原因并使命令进入明确失败状态，不留下无法追踪的 PENDING。
- 后台扫描器处理等待设备、普通响应和传输完成三类超时。
- 状态更新使用当前状态条件和幂等键；重复 Response 或 TransferComplete 不得重复完成命令。
- 服务重启后扫描数据库中的非最终记录，恢复队首、截止时间和必要的 Redis 唤醒数据。
- Request/Response 使用 Command ID 与 CWMP ID 关联；TransferComplete 使用系统生成的唯一 CommandKey 关联。
- XML 清理任务默认删除 30 天前的报文，保留命令、结果和事件。

## 11. 错误处理

- core 构造失败：记录 BUILDING 阶段错误并置为 FAILED。
- Redis 或网络发送失败：记录发送阶段错误并置为 FAILED。
- CWMP Fault：保存 FaultCode、FaultString、完整 XML 并置为 FAILED。
- TransferComplete 故障：保存传输故障码、原因、完成时间并置为 FAILED。
- 设备未声明支持：提交时拒绝，不创建命令，由 GVA 操作日志记录。
- 设备离线：提交时拒绝，不创建命令，由 GVA 操作日志记录。
- 权限拒绝或表单校验失败：不创建命令，由 GVA 操作日志记录。
- 已创建的命令无论成功、失败还是超时，都必须进入明确终态并保留事件。

## 12. 测试策略

### 12.1 tr069-core-only

- 12 种方法的 executor 构造测试。
- XML 序列化快照与必需字段测试。
- Response、CWMP Fault、TransferComplete 解析与关联测试。
- SetParameterAttributes 标准 change 字段测试。
- 重复响应幂等事件测试。

### 12.2 GVA 后端

- 12 种请求的参数校验和能力校验。
- 在线准入、设备 FIFO、到达队首后开始计时。
- 180 秒、90 秒、43200 秒三类可配置超时。
- 完整状态转换、非法状态转换拒绝和最终状态幂等。
- 重试新建记录及 `retryOf` 关联。
- MySQL 命令、事件、XML 持久化和 XML 到期清理。
- Redis 入队失败、服务重启恢复和并发设备隔离。
- API、GVA Casbin 权限和记录详情权限。

### 12.3 GVA 前端

- “更多”菜单分组、具体功能名称和离线禁用。
- 12 种专用表单与字段校验。
- 普通确认与二次确认。
- RPC 记录筛选、状态展示、详情、XML 和重新下发。
- GVA/Element Plus 生产构建。

### 12.4 联调

- 模拟 CPE 自动覆盖 12 种方法、Fault、超时、重复响应和 TransferComplete。
- 真实 BS 自动验证查询类和安全配置类。
- 重启、删除对象、恢复出厂等危险操作不纳入真实 BS 自动测试，只在人工确认后执行。

## 13. 非目标

- 不提供接收任意方法名和任意 JSON 的通用 RPC 控制台。
- 不允许离线设备预先排队等待上线。
- 不修改现有参数树展示方式。
- 不在 GVA 插件中复制一套权限系统。
- 不在父仓库提交 `tr069-core-only` 源码。
- 不在第一期增加批量设备下发、定时任务或命令取消。
