# TR-069 Core XML 可观测日志设计

## 目标

为 `tr069-core-only` 与 GVA TR-069 插件建立一条统一、可关联、可热配置的日志链路，解决当前只能看到 HTTP 原始报文、无法观察 core 内部处理阶段的问题。

最终行为：

- `INFO` 完整记录 core 接收到和发送出去的原始 XML；
- `DEBUG` 在 INFO 基础上记录 Parser、Session、Machine、Executor 和 Builder 内部阶段；
- 日志只使用 XML，不生成 Message JSON 或 JSON 索引；
- core 日志等级独立于 GVA 全局 Zap 等级，并支持配置文件热加载；
- XML 不脱敏，保留实际收发内容；
- 日志写入失败不影响 CWMP 协议处理。

## 仓库边界

本变更跨两个独立仓库实施：

1. `tr069-core-only`：定义可观测事件契约并在协议处理流程中产生事件；
2. `gva-acs`：实现事件 Sink、XML 文件写入、配置热加载、轮转和旧日志迁移。

`server/plugin/tr069/lib/tr069-core-only` 继续由父仓库 `.gitignore` 排除。core 的代码、测试和提交保留在独立仓库中，不加入 GVA Git 历史。GVA 通过现有 Go module 依赖与本地 `replace` 进行联调。

## 现状与问题

GVA TR-069 插件目前并行使用四条日志通道：

- GVA Zap：记录插件运行、Redis、数据库和业务错误；
- `infolog`：按日写入原始报文和命令文本块；
- `RawDump` / `RawResponseDump`：捕获 HTTP 请求和响应；
- 内存 `trace.Store`：按 HTTP requestId 保存少量 handler 阶段。

这些通道没有统一事件模型，requestId、deviceKey、commandId 和 cwmpId 不能稳定贯穿全链路。core 虽然定义了 `Telemetry.Logger`，但 GVA 创建 Engine 时没有注入；core 内部还存在另一套不兼容的 `interface.Logger`。Builder 仅返回 XML 字节，没有将构造阶段交给 GVA 的日志系统。

## 架构

### Core：事件生产者

在 `tr069-core-only` 新增顶层 `observability` 包。该包只依赖 Go 标准库，不依赖 GVA、Zap、Gin 或文件系统。

```go
type Level uint8

const (
    LevelOff Level = iota
    LevelError
    LevelWarn
    LevelInfo
    LevelDebug
)

type Direction string

const (
    DirectionNone     Direction = ""
    DirectionInbound Direction = "inbound"
    DirectionOutbound Direction = "outbound"
)

type Event struct {
    Time       time.Time
    Level      Level
    Stage      string
    Direction Direction
    RequestID  string
    SessionID  string
    DeviceKey  string
    CommandID  string
    CWMPID     string
    Method     string
    Payload    []byte
    Fields     map[string]string
}

type EventSink interface {
    Enabled(Level) bool
    Emit(context.Context, Event)
}
```

`EventSink` 不返回错误，确保日志设施不会改变协议控制流。core 在产生 DEBUG 字段或阶段事件前先调用 `Enabled(LevelDebug)`，避免关闭 DEBUG 时产生无用分配。

`observability` 还提供 context 元数据函数，用于在 handler、engine、machine、executor 和 builder 之间传递 requestId 及后续补充的关联字段。元数据只能追加或覆盖当前阶段已知字段，不使用进程级全局变量。

现有两套 legacy Logger 不在本次变更中强制删除，以避免扩大公共 API 破坏范围。新链路只使用 `EventSink`；旧 Logger 标记为待后续收敛的兼容接口。

### GVA：事件消费者

在 `server/plugin/tr069/observability` 新增：

- `GVAEventSink`：实现 core `EventSink`；
- `LevelFilter`：维护原子日志等级；
- `AsyncXMLWriter`：单 goroutine 顺序写入事件；
- `Rotator`：按日期和100 MB 大小分片；
- `RetentionCleaner`：删除30天以前的日志分片；
- `ConfigReloader`：监听 Viper 配置变化并更新运行策略。

GVA 创建 Parser、Builder 和 Engine 时注入同一个 Sink。core 不读取 GVA YAML，也不负责文件路径、轮转和清理。

## 事件流

标准会话按以下顺序产生事件：

```text
GVA Handler 设置 requestId
  → core Engine 收到 Request
  → INFO wire.received（原始 inbound XML）
  → DEBUG parser.started / parser.completed
  → DEBUG session.resolved / machine.transition
  → DEBUG command.pulled / executor.request-built
  → DEBUG builder.started / builder.completed
  → INFO wire.sent（最终 outbound XML）
  → CPE 下一次请求进入同一关联链
```

关联字段规则：

- `requestId`：一次 HTTP 请求的 ID，由 GVA Handler 创建或接收；
- `sessionId`：core Session ID；
- `deviceKey`：解析 Inform 后确定，格式保持现有约定；
- `commandId`：下发命令的业务 ID；
- `cwmpId`：SOAP/CWMP Header ID；
- `method`：Inform、GetParameterValues、SetParameterValues 等方法名。

无法在早期阶段确定的字段留空，后续事件补齐。AI 通过 requestId、sessionId、deviceKey、commandId 和 cwmpId 的组合重建跨 HTTP 请求会话。

### Core 埋点位置

- `DefaultEngine.Handle`：原始 inbound、Parser 结果、Session 解析、最终 outbound、Fault outbound；
- `Machine.Process`：方法分派和状态阶段；
- `Machine.handleEmpty`：命令拉取、Executor 选择和请求构造；
- `CorrelationHook` 前后：commandId 与 cwmpId 关联结果；
- Builder：构造开始、完成、失败和耗时；
- Parser：解析开始、完成、失败和耗时。

`INFO wire.sent` 必须记录真正返回给 GVA Handler 的最终字节，包括 SOAP Fault。`INFO wire.received` 必须记录传入 core `Request.Body` 的原始字节。DEBUG Trace 只记录内部摘要和字段，不复制完整 Message 参数树。

## XML 日志格式

日志采用带处理指令边界的 XML 片段流。SOAP XML 字节不重新解析、不格式化、不转义、不脱敏。

```xml
<?tr069-event level="INFO" stage="wire.received" direction="inbound" requestId="req-123" deviceKey="001A2B-SNB123456789" cwmpId="789" method="Inform" contentLength="2840" timestamp="2026-07-14T16:30:00.123+08:00"?>
<?xml version="1.0" encoding="UTF-8"?>
<SOAP-ENV:Envelope>...</SOAP-ENV:Envelope>
<?tr069-event-end?>
```

DEBUG Trace 使用同一片段流：

```xml
<?tr069-event level="DEBUG" stage="executor.request-built" requestId="req-123" timestamp="2026-07-14T16:30:00.124+08:00"?>
<tr069:trace xmlns:tr069="urn:gva-acs:tr069:log:1" deviceKey="001A2B-SNB123456789" commandId="cmd-456" cwmpId="789" method="SetParameterValues" elapsed="2ms"/>
<?tr069-event-end?>
```

处理指令属性按 XML 属性规则转义；位于边界之间的 wire payload 保持实际字节。每个事件先完整构造边界和元数据，再交给单 Writer 顺序追加，避免不同会话的 XML 交叉写入。

文件路径：

```text
log/tr069/YYYY-MM-DD/tr069-wire-0001.xml.log
```

每个分片最大100 MB，保留30天。达到大小限制或日期变化时关闭当前分片并创建下一分片。

## 等级与配置

core 日志等级独立于 GVA 全局 Zap：

```yaml
zap:
  level: info

tr069:
  observability:
    coreLevel: INFO
    directory: ./log/tr069
    retentionDays: 30
    maxFileSizeMB: 100
```

等级行为：

| coreLevel | 输出内容 |
|---|---|
| `OFF` | 不产生 core 日志 |
| `ERROR` | 解析、构造、Writer 等错误事件 |
| `WARN` | ERROR 加会话失效、关联失败、锁冲突和未知方法 |
| `INFO` | WARN 加完整 inbound/outbound XML |
| `DEBUG` | INFO 加 core 内部阶段 Trace |

修改 GVA `zap.level` 不影响 core；修改 `tr069.observability.coreLevel` 不影响 GVA 其他模块。

### 热加载

GVA 使用现有 Viper 配置监听能力加载新配置。更新先完整校验，再以原子方式替换有效运行配置：

- `coreLevel`：下一条事件立即使用新等级；
- `directory`：关闭当前分片并切换目录；
- `maxFileSizeMB`：下一次写入开始生效；
- `retentionDays`：下一轮清理开始生效。

非法等级、不可用目录或无效数值不替换当前配置，并通过 GVA Zap 记录一次错误。

## 性能与背压

`AsyncXMLWriter` 使用单 Writer goroutine 保证事件顺序，协议 goroutine 只进行等级判断、事件封装和有界入队。队列按累计字节数限制，避免大量大 XML 占满内存。

采用协议可用性优先策略：

- 队列已满时丢弃新日志事件，不阻塞 CWMP 会话；
- 单事件超过队列容量时丢弃并告警；
- 磁盘写满、目录不可写或 Writer 失败时暂停文件写入并周期性尝试恢复；
- 丢弃数量和 Writer 错误通过 GVA Zap 限频告警；
- 日志错误不再次写入故障 EventSink，避免递归。

core 传递给 Sink 的 `Payload` 在 `Emit` 返回后不得由 core 修改。GVA Writer 在异步持有前负责取得稳定数据所有权；实现和测试必须证明不会发生数据竞争或内容变化。

## 安全约束

用户明确选择完整原文、不脱敏。日志可能包含设备密码、Cookie 对应会话数据、下载/上传凭证及业务参数，因此：

- 日志目录权限限制为服务用户可读写；
- 新文件使用 `0600`；
- 不向普通 GVA Zap 复制 XML payload；
- GVA Zap 只记录事件标识、路径、分片和错误摘要；
- 文档和配置注释明确标识敏感数据风险。

## 旧日志迁移

新链路启用后：

- 从独立 CWMP Gin Engine 移除 `RawDump` 和 `RawResponseDump`，避免重复记录 XML；
- `infolog` 不再记录 core 收发报文和命令文本块；
- handler 的内存 trace 不再承担 core 阶段日志，调试接口后续改为读取新事件或在兼容期保留有限功能；
- 插件启动、Redis、数据库、Connection Request 等运行日志继续使用 GVA Zap。

兼容规则：

- 新 `tr069.observability` 配置存在时，以新配置为准；
- 新配置缺失且旧 `dumpRaw=true` 或 `infoLogEnable=true` 时，映射为 `coreLevel=INFO`；
- 旧 `infoLogDir` 映射为 `observability.directory`；
- 使用旧配置时启动阶段输出一次弃用警告。

## 故障处理

- EventSink 为 nil 时 core 使用 no-op Sink，保持原有行为；
- EventSink panic 必须由 GVA adapter 隔离，不能传播到 core；
- Parser/Builder 失败先产生 ERROR 事件，再按原有协议逻辑返回 SOAP Fault 或 HTTP 错误；
- Writer 故障不会改变 HTTP 状态码、SOAP 内容、Session 状态或命令状态；
- 热加载失败保留上一份有效配置。

## 测试策略

### tr069-core-only

- FakeSink 验证 inbound/outbound payload 与实际传入/返回字节完全一致；
- 验证 Inform、空 POST、命令请求、响应和 SOAP Fault 事件；
- 验证 INFO 不产生内部 Trace，DEBUG 产生阶段 Trace；
- 验证 requestId、sessionId、deviceKey、commandId、cwmpId 和 method 关联；
- 验证 nil/no-op Sink 保持现有测试行为；
- 验证关闭 DEBUG 时不构造 DEBUG payload 和大字段。

### gva-acs

- XML Writer 验证边界、属性转义、原始 payload 和顺序；
- 验证 INFO/DEBUG/OFF 等级过滤；
- 验证配置热加载不重启 Engine；
- 验证100 MB 分片、跨日期分片和30天清理；
- 模拟目录不可写、磁盘错误、队列满和 Sink panic，确认 CWMP 请求仍完成；
- 运行 race test 验证异步 payload 所有权；
- 集成验证 `Inform → InformResponse → Empty POST → 命令 XML → CPE Response` 的双向日志链。

## 完成标准

- GVA `coreLevel=INFO` 时可看到 core 双向完整 XML；
- `coreLevel=DEBUG` 时可看到内部阶段并关联到对应 wire 事件；
- 调整 `coreLevel` 后无需重启，下一条事件立即采用新等级；
- GVA 全局 Zap 等级不会改变 core 日志等级；
- 新日志启用后不再重复写入旧 RawDump/infolog；
- 日志 Writer 故障不影响 CWMP 会话；
- core 与 GVA 两个仓库各自测试通过并分别提交。
