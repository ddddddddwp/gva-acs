## ADDED Requirements

### Requirement: Command ID 必须保持内部生命周期主键
系统 SHALL 使用 Command ID 作为 GVA 命令、事件、XML、队列状态和重试关系的内部稳定主键，但普通用户界面 SHALL NOT 展示、复制或在成功提示中输出 Command ID。

#### Scenario: 用户查看命令记录
- **WHEN** 用户打开 RPC 命令列表或命令详情
- **THEN** 页面不渲染 Command ID 的列、描述项或复制入口
- **AND** 后端仍能使用 Command ID 加载详情和执行受权限保护的重试操作

#### Scenario: 命令在发送前失败
- **WHEN** 命令在排队、等待设备或 core 构造阶段失败
- **THEN** 系统仍通过 Command ID 保存完整状态时间线和失败原因

### Requirement: CWMP ID 必须只表示 SOAP 请求响应关联
系统 SHALL 使用 `cwmp_id/CWMPID/cwmpId` 表示 SOAP `<cwmp:ID>`，并 SHALL NOT 使用 Request ID 命名该协议标识。

#### Scenario: 出站 RPC 成功构造并发送
- **WHEN** core 为 GVA 命令构造出站 CWMP RPC
- **THEN** core 将实际 SOAP CWMP ID 传给 `MarkSending`
- **AND** GVA 将其保存到命令的 `cwmp_id`
- **AND** 命令详情显示相同的 CWMP ID

#### Scenario: 命令尚未产生 SOAP 报文
- **WHEN** 命令仍在排队或等待设备
- **THEN** 命令的 CWMP ID 保持为空
- **AND** UI 显示“尚未生成”而不是伪造协议 ID

#### Scenario: core 构造命令失败
- **WHEN** core 在生成 SOAP 报文前构造失败
- **THEN** 命令详情显示“未生成（构造失败）”

### Requirement: XML 记录必须保存报文实际 CWMP ID
系统 SHALL 为每条已持久化的入站或出站 XML 保存报文实际 CWMP ID，并 SHALL NOT 在 XML DTO 或 UI 中提供 Request ID。

#### Scenario: 查看请求和响应 XML
- **WHEN** 用户打开命令的完整 XML 标签
- **THEN** 每条记录只展示方向、方法、时间、CWMP ID 和 XML payload
- **AND** 请求与响应展示各自报文中的真实 CWMP ID

#### Scenario: 查看设备主动异步报文
- **WHEN** TransferComplete 或 Inform 使用设备生成的 CWMP ID 到达
- **THEN** XML 记录保存该设备生成的 CWMP ID
- **AND** 系统不使用原命令请求的 CWMP ID 覆盖它

### Requirement: CommandKey 必须符合 32 字符派生规则
系统 SHALL 仅为 Reboot、Download、Upload 生成 CommandKey，并 SHALL 通过删除规范 UUID Command ID 的连字符得到 32 个小写十六进制字符。

#### Scenario: 创建异步关联命令
- **WHEN** GVA 创建 Reboot、Download 或 Upload 命令
- **THEN** CommandKey 等于 Command ID 删除连字符后的值
- **AND** CommandKey 长度为 32
- **AND** CommandKey 只包含小写十六进制字符

#### Scenario: 创建普通 RPC 命令
- **WHEN** GVA 创建不依赖异步完成通知的普通 RPC 命令
- **THEN** 系统不生成 CommandKey
- **AND** UI 不显示 CommandKey 描述项

#### Scenario: 重试异步关联命令
- **WHEN** 用户重试失败或超时的 Reboot、Download 或 Upload 命令
- **THEN** 系统创建新的 Command ID 和新的派生 CommandKey
- **AND** 新命令通过 `retry_of` 关联原命令

#### Scenario: 读取历史命令
- **WHEN** 系统读取使用旧规则生成的历史 CommandKey
- **THEN** 系统保持其原值并继续支持已有异步关联

### Requirement: Trace ID 必须只追踪单次 HTTP 请求
GVA 与 core SHALL 使用 `TraceID/traceId` 表示单次 HTTP 请求链路，Trace ID SHALL 只存在于请求上下文和日志，不得写入命令详情或 XML 记录。

#### Scenario: 请求进入 ACS
- **WHEN** 任意 HTTP 请求进入 GVA 或 core ACS 入口
- **THEN** 入口生成新的 UUID Trace ID，不从外部请求头读取链路标识
- **AND** 该值只覆盖当前 HTTP 请求链路
- **AND** core 可观测事件和 GVA 结构化日志使用 `traceId` 字段

#### Scenario: ACS 返回协议响应
- **WHEN** ACS 向 CPE 返回 HTTP 响应
- **THEN** 响应不携带内部 Trace ID 或额外链路标识头

#### Scenario: 用户查看 RPC 详情
- **WHEN** 用户查看命令概览或 XML 详情
- **THEN** 页面不显示 Trace ID 或 Request ID

### Requirement: Session ID 必须独立于 Trace ID
core SHALL 为 CWMP 会话维护独立 Session ID，并 SHALL NOT 使用 HTTP Trace ID 初始化 Session ID 或将 Session ID 写入 `trace_id` 日志字段。

#### Scenario: 同一会话包含多个 HTTP POST
- **WHEN** 同一 CWMP Session 连续处理多个 HTTP 请求
- **THEN** 每个请求拥有各自的 Trace ID
- **AND** 所有请求共享同一个 Session ID
- **AND** 日志分别记录 `traceId` 和 `sessionId`

### Requirement: 数据库必须停机一次性切换标识字段
GVA SHALL 在服务停止期间将 `tr069_commands.request_id` 直接重命名为 `cwmp_id`，并删除 `tr069_command_xmls.request_id`；系统 SHALL NOT 双写、保留旧列或允许旧版本程序在切换后连接数据库。

#### Scenario: 切换已有数据库结构
- **WHEN** GVA 已停止且数据库含有 `tr069_commands.request_id`
- **THEN** 结构切换将该列直接重命名为 `cwmp_id` 并保留其中的 CWMP ID 值
- **AND** 删除 XML 表中的 `request_id`
- **AND** 不修改 Command ID、CommandKey、状态、事件或 XML payload

#### Scenario: 新版本写入命令发送状态
- **WHEN** 新版本 GVA 在结构切换后标记命令已发送
- **THEN** 应用只通过 `cwmp_id` 模型字段读写协议 ID
- **AND** API 不再返回 `requestId`

#### Scenario: 结构切换失败
- **WHEN** 数据库 DDL 未达到目标结构
- **THEN** GVA 保持停止且不得启动新旧版本服务
- **AND** 操作者修复结构后重新验证再启动新版本

### Requirement: core SDK 必须直接使用新标识语义
tr069-core SHALL 使用明确的 Trace ID 和 CWMP ID 字段，并 SHALL 删除旧的公开 Request ID 字段和兼容逻辑；GVA SHALL 在同一变更中只使用新字段。

#### Scenario: core 编译新接口
- **WHEN** tr069-core 和 GVA 完成本次修改
- **THEN** 公开请求和可观测属性只提供 Trace ID 新字段
- **AND** 命令上下文和发送仓储接口只使用 CWMP ID 语义
- **AND** 旧 Request ID 字段和入口兼容桥不存在

#### Scenario: GVA 调用新 core API
- **WHEN** GVA 构造 core 请求或处理可观测事件
- **THEN** GVA 只使用 `TraceID`、`CWMPID` 和 `cwmpID` 命名
- **AND** GVA 代码不新增对废弃 Request ID 字段的依赖

### Requirement: 所有命令操作必须继续受 GVA 权限保护
系统 SHALL 将 Command ID、CWMP ID 和 CommandKey 视为不透明关联值而不是授权凭证，详情、重试和其他命令操作 SHALL 继续执行现有 GVA 权限验证。

#### Scenario: 未授权用户获得任一标识
- **WHEN** 未授权用户知道 Command ID、CWMP ID 或 CommandKey
- **THEN** 后端仍拒绝其访问命令详情或执行重试
