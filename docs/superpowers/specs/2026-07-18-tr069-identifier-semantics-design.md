---
comet_change: unify-tr069-identifier-semantics
role: technical-design
canonical_spec: openspec
---

# TR-069 标识语义统一技术设计

## 1. 设计目标与边界

本设计实现 OpenSpec change `unify-tr069-identifier-semantics`，在不改变现有 RPC 状态机、FIFO 队列、超时策略和 GVA 权限模型的前提下，统一五类标识的职责：

| 标识 | 生成时机 | 生命周期 | 持久化 | 普通 UI |
| --- | --- | --- | --- | --- |
| Command ID | GVA 创建命令时 | 整条命令生命周期 | `tr069_commands.command_id` 及关联表 | 不展示，仅作为内部 API 参数 |
| CWMP ID | core 构造 SOAP 时 | 单次 SOAP 请求/响应 | `tr069_commands.cwmp_id` 与每条 XML 的 `cwmp_id` | 展示 |
| CommandKey | GVA 创建异步命令时 | Reboot/Download/Upload 的异步结果关联 | `tr069_commands.command_key` | 仅相关命令展示 |
| Trace ID | 每次 HTTP 请求入口 | 单次 HTTP 请求 | 不进入命令/XML 表 | 不展示，只写日志 |
| Session ID | core 建立 CWMP 会话时 | 整个 CWMP 会话 | SessionStore | 不展示，只写日志 |

Command ID 仍是稳定内部主键。CWMP ID 不适合作为主键，因为排队、等待设备或构造失败的命令可能从未生成 SOAP 报文。隐藏 Command ID 是界面简化，不构成授权控制；详情、重试等接口继续使用 GVA 权限体系。

## 2. 总体数据流

### 2.1 命令与协议关联

```text
API 提交 RPC
  -> GVA 生成 Command ID
  -> 若 RPC 为 Reboot/Download/Upload，派生 CommandKey
  -> 保存命令、初始事件并进入设备 FIFO
  -> core 取出命令并构造 SOAP
  -> core 生成 CWMP ID
  -> MarkSending(commandID, cwmpID, sentAt)
  -> GVA 写入 tr069_commands.cwmp_id
  -> XML sink 按报文保存 command_id + 报文实际 cwmp_id
  -> 同步响应按 CWMP ID 关联
  -> Reboot Inform / TransferComplete 按 CommandKey 关联
```

命令记录中的 `cwmp_id` 表示该命令下发请求的 SOAP ID。XML 记录中的 `cwmp_id` 表示该条报文本身携带的 SOAP ID，因此设备主动发送的 Inform 或 TransferComplete 可以拥有不同值，不能被命令请求 CWMP ID 覆盖。

### 2.2 HTTP Trace 与 CWMP Session

```text
每个 HTTP 请求入口生成 UUID
  -> Request.TraceID
  -> observability.Attributes.TraceID
  -> 当前 HTTP 请求上下文与结构化日志

X-TR069-Session / tr069_session cookie / 新建 token
  -> SessionStore 临时键
  -> Session.ID（独立生成）
  -> 同一 CWMP 会话的多个 HTTP POST 复用
```

Trace ID 完全由 ACS 内部生成，不读取外部链路标识头，也不写入 HTTP 响应。Trace ID 不参与会话查找，不写入 Session ID、命令记录或 XML 记录。

## 3. tr069-core-only 破坏性切换

core SDK 尚未发布，本次直接修改公开结构和调用方，不保留废弃字段、别名或兼容映射。

### 3.1 类型与接口

修改 `server/plugin/tr069/lib/tr069-core-only/pkg/core/types.go`：

- `Request.ID` 直接改为 `Request.TraceID`。
- `CommandContext.RequestID` 直接改为 `CommandContext.CWMPID`。
- `Session.ID` 注释和语义改为独立 CWMP Session ID，不再描述为 Trace ID。

修改 `pkg/core/interfaces.go` 及实现：

- `CommandRepo.MarkSending(ctx, commandID, cwmpID, sentAt)` 的第三个业务参数统一命名为 `cwmpID`。
- `pkg/core/defaults/memory_repos.go` 中 `sendingRecord.requestID` 改为 `cwmpID`。
- GVA 的 `GormCommandRepo` 同步实现新语义。

Go 接口的方法类型不因形参名变化而改变，但实现、测试变量名和断言都必须使用 CWMP ID，避免语义再次退化。

### 3.2 可观测上下文

修改 `observability/event.go`、`observability/context.go` 及相关测试：

- `Attributes.RequestID` 改为 `Attributes.TraceID`。
- merge 与 override 只合并 `TraceID`，不保留旧字段分支。
- parser、builder 和 HTTP 入口产生的事件继续携带相同 Trace ID。
- Wire event 的 `CWMPID` 和 `CommandID` 保持独立字段，不允许用 Trace ID 补值。

修改 `pkg/core/telemetry.go` 与 `pkg/core/machine.go`：

- 保留 `traceId` 日志字段并从请求上下文的可观测属性读取。
- 新增明确的 `sessionId` 日志字段。
- 禁止继续把 `session.ID` 写到 `traceId`。
- 协议构造和响应日志可同时出现 `traceId`、`sessionId`、`cwmpId`、`commandId`，每个字段都使用自身来源。

### 3.3 HTTP 入口与 Session 建立

修改 `adapters/http/engine_handler.go`：

- `requestID` helper 改名为 `traceID`。
- 每次请求生成新的 UUID 风格 Trace ID，不接受外部请求头覆盖。
- 构造 `core.Request{TraceID: ...}`，并把 Trace ID 注入 observability context。

修改 `pkg/core/engine_impl.go`：

- 新建 Session 时用 `e.conf.NewCwmpID()` 生成独立 `Session.ID`，不使用 `req.TraceID`。
- 既有 Session 的 ID 为空时也独立补生成，不从当前 HTTP 请求回填。
- `X-TR069-Session`、`tr069_session` cookie 和 IP 临时键继续用于 SessionStore 查找与迁移；session token 与 `Session.ID` 可以使用同一新生成 token，但都不得来自 Trace ID。
- 每次 `Handle` 使用本次请求的 Trace ID 丰富 context；从 SessionStore 恢复的 Session ID 在会话期间保持不变。

关键测试场景是：同一 cookie 的连续两个 POST 具有两个不同 Trace ID，但读取到同一个 Session ID，且日志字段不串位。

## 4. GVA 后端改造

### 4.1 数据模型与仓储

修改以下模型和 DTO：

- `server/plugin/tr069/model/command.go`：`RequestID` 改为 `CWMPID`，JSON 为 `cwmpId`，GORM 列为 `cwmp_id` 并保留索引。
- `server/plugin/tr069/model/command_xml.go`：删除 `RequestID`，保留 `CommandID`、`CWMPID` 和 XML payload。
- `server/plugin/tr069/model/response/command_record.go`：命令摘要与详情使用 `cwmpId`；XML DTO 删除 `requestId`。

修改仓储和关联逻辑：

- `adapter/gorm_repo.go` 的 `MarkSending` 更新 `cwmp_id`。
- `service/command_store.go` 在 wire event 缺少 Command ID 时，通过 `tr069_commands.cwmp_id = event.CWMPID` 找到命令。
- `adapter/command_xml_sink.go` 只保存 wire event 的实际 `CWMPID`，不再读取或落库 `RequestID`。
- 命令内部的 `CommandID` 仍用于 XML、事件、重试和队列关联，DTO 可以继续返回它供前端内部调用，但普通 UI 不渲染。

构造前失败时 `cwmp_id` 保持空；状态为排队/等待设备时前端显示“尚未生成”，构造阶段失败且没有 CWMP ID 时显示“未生成（构造失败）”。

### 4.2 CommandKey 派生

在 `service/command_manager.go` 附近增加单一 helper，例如：

```go
func commandKeyFromCommandID(commandID string) (string, error)
```

实现规则：

1. 使用 UUID 库解析 `commandID`，拒绝非规范 UUID。
2. 使用 UUID 的 32 位小写十六进制表示，等价于删除规范 UUID 字符串中的四个连字符。
3. 仅当 RPC registry 的 `ServerCommandKey` 为 true 时调用。
4. 创建命令时先生成一次 Command ID，再从同一个值派生 CommandKey；禁止再次生成 UUID。

当前 registry 只有 Reboot、Download、Upload 设置 `ServerCommandKey`，因此普通 RPC 的 CommandKey 保持 `nil`。重试沿用创建新命令的路径，自动获得新的 Command ID、CommandKey，并保留 `retry_of`。读取历史记录时不重新计算，避免改写旧的 40 字符值或破坏尚未完成的异步关联。

### 4.3 GVA Trace 命名

修改 `server/plugin/tr069/trace`、`middleware/raw_dump.go`、`middleware/raw_response_dump.go`、`handler/cwmp.go`、调试 API 与路由：

- `WithRequestID/RequestID/EnsureRequestID` 改为 `WithTraceID/TraceID/EnsureTraceID`。
- Gin context 内部键和结构化日志字段使用 `traceId`。
- Trace ID 仅保留在内部上下文和日志中，不读取请求头，也不写入响应头。
- 调试路由参数由 `:requestId` 改为 `:traceId`。
- GVA 构造 core 请求时只设置 `core.Request.TraceID`。

Trace ID 只进入请求日志和 debug trace 存储，不加入 `model.Command`、`model.CommandXML`、命令 DTO 或前端详情。

## 5. 数据库一次性停机切换

### 5.1 切换实现

在 `server/plugin/tr069/initialize/gorm.go` 附近实现一个幂等的目标结构检查/切换函数，使用 GORM Migrator 完成：

1. 若 `tr069_commands` 存在 `request_id` 且不存在 `cwmp_id`，执行列重命名。
2. 若两列同时存在，视为异常结构并返回错误，不做猜测或合并。
3. 若已只有 `cwmp_id`，跳过重命名，便于失败后的安全重试和新数据库启动。
4. 若 `tr069_command_xmls.request_id` 存在，删除该列；不存在则跳过。
5. 随后运行新模型的 AutoMigrate，确认 `cwmp_id` 索引和目标表结构。

该幂等性只服务于同一次停机维护的失败重试，不是旧程序兼容层。代码不双写、不回填第二列，也不允许切换完成后再运行旧版本。

### 5.2 运维顺序与失败处理

```text
停止 GVA 后端（ACS 7458 同时停止接入）
  -> 备份/确认 MySQL 可连接
  -> 运行新版本结构切换
  -> 校验 commands 只有 cwmp_id、XML 表没有 request_id
  -> 运行后端测试/启动检查
  -> 启动 GVA 后端与前端
```

DDL 或目标结构校验失败时，启动函数返回错误，后端不得继续监听端口。此时修复数据库结构后重试新版本；不通过应用双字段兼容回退旧二进制。列重命名保留原 `request_id` 中已有的 CWMP ID 值，且不修改 Command ID、CommandKey、状态、事件或 XML payload。

### 5.3 数据库测试

SQLite/GORM 测试至少覆盖：

- 旧命令表的 `request_id` 被重命名，原值保留。
- XML 表的 `request_id` 被删除，其他 XML 字段和 payload 保留。
- 目标结构上重复执行安全。
- 两列同时存在时明确失败。
- 新数据库直接得到目标结构。
- 新仓储只更新和查询 `cwmp_id`，API JSON 不出现 `requestId`。

MySQL 实际切换前额外执行结构查询确认索引状态；SQLite 测试负责代码路径，MySQL 验证负责生产方言差异。

## 6. 前端展示调整

修改 `web/src/plugin/tr069/view/command-record/index.vue`：

- 删除 Command ID 搜索条件和列表列。
- 保留 `commandId` 作为 `row-key`、打开详情和重试 API 的内部不透明参数。

修改 `components/record-detail.vue`：

- 删除 Command ID 和 Request ID 描述项。
- 新增/改用 CWMP ID 描述项，并按命令状态显示空值文案。
- 仅当操作为 Reboot、Download、Upload 时渲染 CommandKey。
- XML 元数据删除 Request ID，只保留方向、方法、时间和 CWMP ID。
- 重试成功提示改为“命令已重新提交”，不拼接返回的 Command ID。

修改设备 RPC 对话框和参数同步入口：

- `rpc-command-dialog.vue`、`data-model-viewer.vue` 的成功提示不再输出 `commandId=...`。
- API 返回结构和内部跳转仍可使用 Command ID，不通过页面文本、Tooltip、复制入口或通知暴露。

Element Plus 主题变量继续用于抽屉、描述表、XML 和 JSON 区域，不引入固定亮色，保持现有明暗主题兼容。

## 7. TDD 与验证矩阵

实现顺序遵循先失败测试、再最小实现、最后重构：

| 层级 | RED 证据 | GREEN 验证 |
| --- | --- | --- |
| core 类型/观测 | 新测试引用 `TraceID`，旧实现无法编译或断言失败 | `go test ./...`，并用源码契约确认旧公开字段不存在 |
| core Session | 连续 POST 测试发现 Session ID 等于首个 Trace ID | 两次 Trace 不同、Session 相同、日志字段独立 |
| GVA schema | 旧表结构无法满足新模型/目标列断言 | 重命名、删列、值保留、重复执行测试通过 |
| GVA 仓储/XML | 仍写 `request_id` 或 XML DTO 含 `requestId` | `cwmp_id` 关联和双向 XML 保存测试通过 |
| CommandKey | 当前 `rpc-<UUID>` 长度/值断言失败 | 仅三种 RPC 生成精确 32 位小写十六进制值 |
| 前端契约 | 现有模板仍含 Command ID、Request ID 和提示文本 | Node contract tests 与前端构建通过 |

定向验证完成后运行：

- core：在 `tr069-core-only` 仓库执行 `go test ./...`。
- GVA：在 `server` 执行 `go test ./plugin/tr069/...`，再执行必要的后端构建。
- 前端：执行现有 `node --test` contract 测试集合和 `npm run build`。
- 字面契约：检查用户可见 Vue 模板、DTO JSON tag 和日志字段，确认没有含义错误的 `requestId`。
- 运行检查：按停机切换顺序重启后端和前端，确认 18888、18080、7458 监听，并提交一条安全查询 RPC 验证 CWMP ID 与双向 XML 展示。

## 8. 提交边界与仓库协调

`tr069-core-only` 是独立 Git 仓库且被 GVA 主仓库忽略。实施时分别提交：

1. core 仓库 `dev`：类型、观测、Session、接口及测试的破坏性切换。
2. GVA 主仓库 `dev`：模型、数据库切换、仓储、Trace、前端、文档及测试。

两个提交必须在同一开发窗口验证；任何一侧单独部署都会导致编译或数据库契约不匹配。数据库结构切换只在新 GVA 后端已经构建成功后执行。

## 9. 明确不做的事项

- 不以 CWMP ID 取代 Command ID 主键。
- 不新增 Public ID、签名令牌或第五套用户标识。
- 不保留 core 旧字段兼容别名。
- 不保留数据库旧列、双写逻辑或旧程序回滚桥。
- 不批量改写历史 CommandKey。
- 不修改现有 RPC 状态机、队列、超时或权限模型。
