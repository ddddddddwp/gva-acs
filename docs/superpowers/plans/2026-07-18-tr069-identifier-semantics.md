---
change: unify-tr069-identifier-semantics
design-doc: docs/superpowers/specs/2026-07-18-tr069-identifier-semantics-design.md
base-ref: 632b2566b528fb2beb52a3d205e973cefbedc220
---

# TR-069 标识语义统一实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 一次性统一 GVA 与 tr069-core-only 中 Command ID、CWMP ID、CommandKey、Trace ID 和 Session ID 的语义、存储与展示。

**Architecture:** core 先完成破坏性类型切换和 Trace/Session 解耦，GVA 随后切换模型、仓储、XML 关联和 Trace 命名；数据库在 GVA 停止期间直接重命名/删除旧列。Command ID 保持内部主键，前端只展示协议相关的 CWMP ID 和条件化 CommandKey。

**Tech Stack:** Go、Gin、GORM、MySQL/SQLite、Vue 3、Element Plus、Node test runner、OpenSpec/Comet。

## Global Constraints

- `tr069-core-only` 是独立 Git 仓库，core 任务在其 `dev` 分支独立提交，主仓库不提交该目录内容。
- 不保留 `Request.ID`、`Attributes.RequestID`、`CommandContext.RequestID` 或数据库旧列兼容逻辑。
- `Command ID` 继续作为 GVA 内部主键，但普通 UI 不展示、复制或在成功提示中输出。
- `CWMP ID` 只表示 SOAP `<cwmp:ID>`；Trace ID 由 ACS 为每次 HTTP 请求独立生成且只用于内部日志链路；Session ID 独立于 Trace ID。
- 只有 Reboot、Download、Upload 生成 CommandKey，值为 Command ID 去除连字符后的 32 位小写十六进制字符串。
- 不改写历史 CommandKey，不改变现有 RPC 状态机、FIFO、超时和权限模型。
- 每个任务先验证 RED，再完成 GREEN；每个仓库的提交只包含对应任务文件。

---

### Task 1: core 请求与可观测标识破坏性改名

**OpenSpec tasks:** 1.1、1.2、1.4（类型与 `MarkSending` 部分）、5.1（旧字段移除部分）

**Files:**
- Modify: `server/plugin/tr069/lib/tr069-core-only/pkg/core/types.go`
- Modify: `server/plugin/tr069/lib/tr069-core-only/pkg/core/interfaces.go`
- Modify: `server/plugin/tr069/lib/tr069-core-only/pkg/core/defaults/memory_repos.go`
- Modify: `server/plugin/tr069/lib/tr069-core-only/observability/event.go`
- Modify: `server/plugin/tr069/lib/tr069-core-only/observability/context.go`
- Modify: `server/plugin/tr069/lib/tr069-core-only/adapters/http/engine_handler.go`
- Modify: `server/plugin/tr069/lib/tr069-core-only/internal/builder/builder_observability_test.go`
- Modify: `server/plugin/tr069/lib/tr069-core-only/parser/auto_observability_test.go`
- Modify: `server/plugin/tr069/lib/tr069-core-only/observability/event_test.go`
- Create: `server/plugin/tr069/lib/tr069-core-only/adapters/http/engine_handler_trace_test.go`

**Interfaces:**
- Produces: `core.Request.TraceID string`、`observability.Attributes.TraceID string`、`core.CommandContext.CWMPID string`。
- Produces: `CommandRepo.MarkSending(ctx context.Context, commandID, cwmpID string, sentAt time.Time) error`。

- [x] **Step 1: 写 Trace ID 入口和可观测传播失败测试**

```go
func TestEngineHandlerGeneratesInternalTraceID(t *testing.T) {
	engine := &captureEngine{}
	h := NewEngineHandler(engine)
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(""))
	h.ServeHTTP(httptest.NewRecorder(), req)
	if engine.request == nil || len(engine.request.TraceID) != 36 {
		t.Fatalf("TraceID = %#v, want generated UUID", engine.request)
	}
}
```

同时把 observability 测试结构体字面量改为 `Attributes{TraceID: "trace-1"}`，断言 merge/override 后 `TraceID` 保持正确。

- [x] **Step 2: 运行定向测试并确认 RED**

Run: `cd server/plugin/tr069/lib/tr069-core-only && go test ./adapters/http ./observability ./internal/builder ./parser`

Expected: FAIL，至少包含 `Request.TraceID undefined` 或 `Attributes.TraceID undefined`。

- [x] **Step 3: 完成直接字段切换和入口映射**

```go
type Request struct {
	TraceID       string
	RemoteIP      string
	Headers       map[string]string
	Body          []byte
	ReceivedAt    time.Time
	TransportMeta map[string]interface{}
}

type CommandContext struct {
	CommandID string
	Operation string
	DedupKey  string
	CWMPID    string
	State     string
	Retries   int
}
```

将 `Attributes.RequestID` 改为 `TraceID`，merge/override 只处理 `TraceID`；HTTP helper 改为每次生成内部 Trace ID，不读取外部链路头。将 memory repo 的 `sendingRecord.requestID` 改为 `cwmpID`，并同步所有测试变量名。

- [x] **Step 4: 运行定向测试和旧字段契约检查**

Run: `cd server/plugin/tr069/lib/tr069-core-only && go test ./adapters/http ./observability ./internal/builder ./parser ./pkg/core/defaults`

Expected: PASS。

Run: `cd server/plugin/tr069/lib/tr069-core-only && ! rg -n 'RequestID|Request\{[[:space:]]*ID:|\.ID[[:space:]]*=[[:space:]]*requestID' pkg observability adapters internal parser`

Expected: exit 0；不允许外部链路头字面量、旧公开字段或变量语义。

- [x] **Step 5: 提交 core 类型切换**

```bash
cd server/plugin/tr069/lib/tr069-core-only
git add pkg/core/types.go pkg/core/interfaces.go pkg/core/defaults/memory_repos.go observability adapters/http internal/builder parser
git commit -m "refactor: clarify trace and cwmp identifiers"
```

### Task 2: core Session ID 与 Trace ID 解耦

**OpenSpec tasks:** 1.3、1.4（会话与日志部分）、5.1（core 全量测试）

**Files:**
- Modify: `server/plugin/tr069/lib/tr069-core-only/pkg/core/engine_impl.go`
- Modify: `server/plugin/tr069/lib/tr069-core-only/pkg/core/machine.go`
- Modify: `server/plugin/tr069/lib/tr069-core-only/pkg/core/telemetry.go`
- Modify: `server/plugin/tr069/lib/tr069-core-only/pkg/core/engine_cookie_test.go`
- Modify: `server/plugin/tr069/lib/tr069-core-only/pkg/core/engine_flow_test.go`
- Create: `server/plugin/tr069/lib/tr069-core-only/pkg/core/engine_identifier_test.go`

**Interfaces:**
- Consumes: `Request.TraceID` 和 observability context。
- Produces: 独立 `Session.ID`；日志键 `traceId` 与 `sessionId`。

- [x] **Step 1: 写同一会话多请求失败测试**

```go
func TestSessionIDDoesNotReuseHTTPTraceID(t *testing.T) {
	engine, store := newIdentifierTestEngine(t)
	inform := identifierInformRequest("trace-1")
	if _, err := engine.Handle(context.Background(), inform); err != nil { t.Fatal(err) }
	session, err := store.GetByDeviceKey(context.Background(), "001122-SERIAL")
	if err != nil { t.Fatal(err) }
	if session.ID == "" || session.ID == "trace-1" {
		t.Fatalf("session ID %q must be independent from trace ID", session.ID)
	}
	firstSessionID := session.ID
	if _, err := engine.Handle(context.Background(), identifierFollowupRequest("trace-2")); err != nil { t.Fatal(err) }
	session, _ = store.GetByDeviceKey(context.Background(), "001122-SERIAL")
	if session.ID != firstSessionID { t.Fatalf("session changed: %q", session.ID) }
}
```

测试 helper 使用现有 `SessionHeaderName`/`SessionCookieName` 和内存 SessionStore，避免依赖真实网络。

- [x] **Step 2: 运行定向测试并确认 RED**

Run: `cd server/plugin/tr069/lib/tr069-core-only && go test ./pkg/core -run 'TestSessionIDDoesNotReuseHTTPTraceID|Test.*Cookie' -count=1`

Expected: FAIL，旧实现的 `Session.ID` 等于首个请求 Trace ID。

- [x] **Step 3: 独立生成 Session ID 并修正日志字段**

```go
if session == nil {
	session = &Session{ID: e.conf.NewCwmpID()}
}
if session.ID == "" {
	session.ID = e.conf.NewCwmpID()
}
```

在 `telemetry.go` 增加 `LogFieldSessionID = "sessionId"`。`Machine.Process` 从 observability context 读取 Trace ID，并分别记录 `LogFieldTraceID` 与 `LogFieldSessionID`，禁止把 `session.ID` 写入 Trace 字段。

- [x] **Step 4: 运行 core 全量测试**

Run: `cd server/plugin/tr069/lib/tr069-core-only && go test ./...`

Expected: PASS。

- [x] **Step 5: 提交 core 会话修正**

```bash
cd server/plugin/tr069/lib/tr069-core-only
git add pkg/core
git commit -m "refactor: separate cwmp sessions from http traces"
```

### Task 3: GVA 数据库切换、命令模型与 XML 关联

**OpenSpec tasks:** 2.1、2.2、2.3、2.4、5.2（后端部分）

**Files:**
- Modify: `server/plugin/tr069/initialize/gorm.go`
- Create: `server/plugin/tr069/initialize/gorm_identifier_cutover_test.go`
- Modify: `server/plugin/tr069/model/command.go`
- Modify: `server/plugin/tr069/model/command_xml.go`
- Modify: `server/plugin/tr069/model/response/command_record.go`
- Modify: `server/plugin/tr069/adapter/gorm_repo.go`
- Modify: `server/plugin/tr069/adapter/gorm_repo_test.go`
- Modify: `server/plugin/tr069/adapter/command_xml_sink.go`
- Modify: `server/plugin/tr069/adapter/command_xml_sink_test.go`
- Modify: `server/plugin/tr069/service/command_store.go`
- Modify: `server/plugin/tr069/service/command_store_test.go`
- Modify: `server/plugin/tr069/api/command_record_test.go`

**Interfaces:**
- Produces: `model.Command.CWMPID string`，JSON `cwmpId`，列 `cwmp_id`。
- Produces: `cutoverCommandIdentifierSchema(db *gorm.DB) error`，目标结构可重复检查。
- Consumes: core `MarkSending(..., cwmpID, ...)` 与 wire event `CWMPID`。

- [x] **Step 1: 写数据库结构、仓储和 DTO 失败测试**

```go
func TestCutoverCommandIdentifierSchemaRenamesAndPreservesValue(t *testing.T) {
	db := openIdentifierCutoverDB(t)
	mustExec(t, db, `CREATE TABLE tr069_commands (command_id text primary key, request_id text)`)
	mustExec(t, db, `INSERT INTO tr069_commands(command_id, request_id) VALUES ('cmd-1', 'cwmp-1')`)
	mustExec(t, db, `CREATE TABLE tr069_command_xmls (id integer primary key, request_id text, cwmp_id text, payload blob)`)
	if err := cutoverCommandIdentifierSchema(db); err != nil { t.Fatal(err) }
	if db.Migrator().HasColumn("tr069_commands", "request_id") { t.Fatal("request_id still exists") }
	if !db.Migrator().HasColumn("tr069_commands", "cwmp_id") { t.Fatal("cwmp_id missing") }
	var got string
	if err := db.Table("tr069_commands").Select("cwmp_id").Where("command_id = ?", "cmd-1").Scan(&got).Error; err != nil { t.Fatal(err) }
	if got != "cwmp-1" { t.Fatalf("cwmp_id = %q", got) }
	if db.Migrator().HasColumn("tr069_command_xmls", "request_id") { t.Fatal("xml request_id still exists") }
}
```

另加测试覆盖目标结构重复执行、双列并存返回错误、`MarkSending` 写 `cwmp_id`、XML DTO JSON 不含 `requestId`。

- [x] **Step 2: 运行定向测试并确认 RED**

Run: `cd server && go test ./plugin/tr069/initialize ./plugin/tr069/adapter ./plugin/tr069/service ./plugin/tr069/api -run 'Identifier|MarkSending|CommandXML|CommandRecord' -count=1`

Expected: FAIL，原因是 cutover 函数或 `CWMPID` 字段不存在，或 DTO 仍含 `requestId`。

- [x] **Step 3: 实现目标结构切换与模型改名**

```go
func cutoverCommandIdentifierSchema(db *gorm.DB) error {
	m := db.Migrator()
	hasOld := m.HasColumn("tr069_commands", "request_id")
	hasNew := m.HasColumn("tr069_commands", "cwmp_id")
	if hasOld && hasNew { return errors.New("tr069_commands contains both request_id and cwmp_id") }
	if hasOld {
		if err := m.RenameColumn("tr069_commands", "request_id", "cwmp_id"); err != nil { return err }
	}
	if m.HasColumn("tr069_command_xmls", "request_id") {
		if err := m.DropColumn("tr069_command_xmls", "request_id"); err != nil { return err }
	}
	return nil
}
```

在 AutoMigrate 前调用该函数；新库不存在目标表时允许直接 AutoMigrate。将命令模型字段改为 `CWMPID string 'json:"cwmpId" gorm:"column:cwmp_id;size:64;index"'`，XML 模型和响应 DTO 删除 `RequestID`。

- [x] **Step 4: 切换仓储与双向 XML 关联**

将 `MarkSending` updates key 改为 `cwmp_id`；`SaveXML` 在 wire event 没有 Command ID 时使用：

```go
if copyRecord.CommandID == "" && copyRecord.CWMPID != "" {
	var command model.Command
	if err := s.db.WithContext(ctx).Select("command_id").
		Where("cwmp_id = ?", copyRecord.CWMPID).First(&command).Error; err == nil {
		copyRecord.CommandID = command.CommandID
	}
}
```

XML sink 只复制 `CommandID`、`CWMPID`、方向、方法、payload 和时间，不保存 Trace ID。

- [x] **Step 5: 运行 GVA 插件测试并提交**

Run: `cd server && go test ./plugin/tr069/...`

Expected: PASS。

```bash
git add server/plugin/tr069/initialize server/plugin/tr069/model server/plugin/tr069/adapter server/plugin/tr069/service server/plugin/tr069/api
git commit -m "refactor: persist cwmp identifiers explicitly"
```

### Task 4: CommandKey 确定性派生与异步关联

**OpenSpec tasks:** 3.1、3.2、3.3、3.4

**Files:**
- Modify: `server/plugin/tr069/service/command_manager.go`
- Modify: `server/plugin/tr069/service/command_manager_test.go`
- Modify: `server/plugin/tr069/adapter/gorm_repo_test.go`
- Modify: `server/plugin/tr069/service/command_store_test.go`

**Interfaces:**
- Produces: `commandKeyFromCommandID(commandID string) (string, error)`。
- Consumes: RPC registry 的 `ServerCommandKey bool`。

- [x] **Step 1: 写 32 字符派生、方法范围和重试失败测试**

```go
func TestCommandKeyFromCommandID(t *testing.T) {
	got, err := commandKeyFromCommandID("62a53a00-786f-4a45-b31f-b23f7d23bb6e")
	if err != nil { t.Fatal(err) }
	if got != "62a53a00786f4a45b31fb23f7d23bb6e" { t.Fatalf("got %q", got) }
}
```

表驱动提交测试断言 Reboot/Download/Upload 的 `commandKey == strings.ReplaceAll(commandID, "-", "")`，GetParameterValues 的 CommandKey 为 nil；重试断言新旧 Command ID/CommandKey 不同且 `RetryOf` 指向原命令。

- [x] **Step 2: 运行定向测试并确认 RED**

Run: `cd server && go test ./plugin/tr069/service -run 'CommandKey|Retry' -count=1`

Expected: FAIL，旧值以 `rpc-` 开头且长度为 40。

- [x] **Step 3: 从唯一 Command ID 派生 CommandKey**

```go
func commandKeyFromCommandID(commandID string) (string, error) {
	id, err := uuid.Parse(commandID)
	if err != nil { return "", fmt.Errorf("invalid command id: %w", err) }
	return strings.ReplaceAll(id.String(), "-", ""), nil
}
```

创建命令时先保存 `commandID := uuid.NewString()`；仅 `definition.ServerCommandKey` 为 true 时调用 helper，禁止生成第二个 UUID。历史记录读取和异步匹配逻辑不重新计算 CommandKey。

- [x] **Step 4: 验证同步/异步关联并提交**

Run: `cd server && go test ./plugin/tr069/service ./plugin/tr069/adapter -run 'CommandKey|Retry|Reboot|Transfer' -count=1`

Expected: PASS。

```bash
git add server/plugin/tr069/service/command_manager.go server/plugin/tr069/service/command_manager_test.go server/plugin/tr069/adapter/gorm_repo_test.go server/plugin/tr069/service/command_store_test.go
git commit -m "fix: derive protocol-safe command keys"
```

### Task 5: GVA HTTP Trace 命名统一

**OpenSpec tasks:** 4.1、4.2

**Files:**
- Modify: `server/plugin/tr069/trace/trace.go`
- Create: `server/plugin/tr069/trace/trace_test.go`
- Modify: `server/plugin/tr069/middleware/raw_dump.go`
- Modify: `server/plugin/tr069/middleware/raw_response_dump.go`
- Modify: `server/plugin/tr069/middleware/raw_response_dump_test.go`
- Modify: `server/plugin/tr069/handler/cwmp.go`
- Modify: `server/plugin/tr069/handler/cwmp_test.go`
- Modify: `server/plugin/tr069/initialize/server.go`
- Modify: `server/plugin/tr069/api/debug.go`
- Modify: `server/plugin/tr069/initialize/api.go`
- Modify: `server/plugin/tr069/router/device.go`

**Interfaces:**
- Produces: `trace.WithTraceID(ctx, traceID)`、`trace.TraceID(ctx)`、`middleware.EnsureTraceID()`。
- Consumes: core `Request.TraceID`；不依赖外部链路标识头。

- [x] **Step 1: 写头映射和持久化隔离失败测试**

```go
func TestEnsureTraceIDGeneratesInternalUUID(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/acs", nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = r
	EnsureTraceID()(c)
	if got := c.GetString("traceId"); len(got) != 36 { t.Fatalf("got %q", got) }
}
```

handler 测试捕获 core Request 并断言 `TraceID`；DTO/模型测试断言 Trace ID 不进入命令或 XML JSON。

- [x] **Step 2: 运行定向测试并确认 RED**

Run: `cd server && go test ./plugin/tr069/trace ./plugin/tr069/middleware ./plugin/tr069/handler -count=1`

Expected: FAIL，新 API/键尚不存在。

- [x] **Step 3: 完成 GVA 内部 Trace 改名**

将 context key、map 名、日志字段和调试路由统一为 `traceId`；中间件只生成内部 UUID，不读取或回写外部链路头。`handler/cwmp.go` 构造：

```go
request := &core.Request{
	TraceID:    traceID,
	RemoteIP:   c.ClientIP(),
	Headers:    requestHeaders(c.Request),
	Body:       body,
	ReceivedAt: time.Now(),
}
```

- [x] **Step 4: 运行插件测试并提交**

Run: `cd server && go test ./plugin/tr069/...`

Expected: PASS。

```bash
git add server/plugin/tr069/trace server/plugin/tr069/middleware server/plugin/tr069/handler server/plugin/tr069/initialize server/plugin/tr069/api server/plugin/tr069/router
git commit -m "refactor: name http correlation as trace id"
```

### Task 6: RPC 命令记录 UI 收敛

**OpenSpec tasks:** 4.3、4.4、5.2（前端 contract 部分）、5.3（用户可见 JSON/文案部分）

**Files:**
- Modify: `web/src/plugin/tr069/view/command-record/index.vue`
- Modify: `web/src/plugin/tr069/view/command-record/components/record-detail.vue`
- Modify: `web/src/plugin/tr069/view/command-record/record-view.js`
- Modify: `web/src/plugin/tr069/view/command-record/record-view.test.js`
- Create: `web/src/plugin/tr069/view/command-record/command-record.contract.test.js`
- Modify: `web/src/plugin/tr069/view/device/components/rpc-command-dialog.vue`
- Modify: `web/src/plugin/tr069/view/device/components/data-model-viewer.vue`
- Modify: `web/src/plugin/tr069/view/device/components/data-model-viewer.contract.test.js`

**Interfaces:**
- Consumes: API 内部 `commandId`、用户可见 `cwmpId`、可选 `commandKey`。
- Produces: `cwmpIDDisplay(command)` 和 `showsCommandKey(operation)` 等纯展示 helper。

- [x] **Step 1: 写模板契约和展示 helper 失败测试**

```js
test('RPC command UI hides internal identifiers', () => {
  const detail = readFileSync(new URL('./components/record-detail.vue', import.meta.url), 'utf8')
  const list = readFileSync(new URL('./index.vue', import.meta.url), 'utf8')
  assert.doesNotMatch(detail, /label="Command ID"|label="Request ID"|commandId=/)
  assert.doesNotMatch(list, /label="Command ID"|placeholder="Command ID"/)
  assert.match(detail, /label="CWMP ID"/)
})
```

纯函数测试覆盖：排队/等待设备返回“尚未生成”；构造失败且无 CWMP ID 返回“未生成（构造失败）”；仅三种异步命令展示 CommandKey。

- [x] **Step 2: 运行 contract 测试并确认 RED**

Run: `cd web && node --test src/plugin/tr069/view/command-record/*.test.js src/plugin/tr069/view/device/components/*.contract.test.js`

Expected: FAIL，当前模板仍展示 Command ID/Request ID 或成功提示含 `commandId=`。

- [x] **Step 3: 更新列表、详情、XML 和成功提示**

列表删除 Command ID 搜索框与列，但保留 `row-key="commandId"` 和详情 API 参数。详情使用：

```vue
<el-descriptions-item label="CWMP ID">{{ cwmpIDDisplay(command) }}</el-descriptions-item>
<el-descriptions-item v-if="showsCommandKey(command.operation)" label="CommandKey">
  {{ command.commandKey || '-' }}
</el-descriptions-item>
```

XML 元数据删除 Request ID；重试、RPC 提交和参数同步成功文案不拼接 Command ID。样式只使用 Element Plus CSS 变量，保持明暗主题。

- [x] **Step 4: 运行前端测试与构建并提交**

Run: `cd web && node --test src/plugin/tr069/view/command-record/*.test.js src/plugin/tr069/view/device/components/*.contract.test.js`

Expected: PASS。

Run: `cd web && npm run build`

Expected: exit 0。

```bash
git add web/src/plugin/tr069/view/command-record web/src/plugin/tr069/view/device/components
git commit -m "refactor: simplify rpc identifier display"
```

### Task 7: 联合验证、停机切换与运行检查

**OpenSpec tasks:** 5.1、5.2、5.3、5.4；收口 1.1-4.4 的验收证据

**Files:**
- Modify: `openspec/changes/unify-tr069-identifier-semantics/tasks.md`
- Verify only: core、`server/plugin/tr069`、`web/src/plugin/tr069`

**Interfaces:**
- Consumes: Task 1-6 的 core 与 GVA 提交。
- Produces: 目标 MySQL schema、运行中的 GVA 前后端/ACS 和全部已勾选 OpenSpec tasks。

- [x] **Step 1: 运行两个仓库和前端完整验证**

Run: `cd server/plugin/tr069/lib/tr069-core-only && go test ./...`

Expected: PASS。

Run: `cd server && go test ./plugin/tr069/...`

Expected: PASS。

Run: `cd web && node --test src/plugin/tr069/view/command-record/*.test.js src/plugin/tr069/view/device/components/*.contract.test.js && npm run build`

Expected: PASS，构建 exit 0。

- [x] **Step 2: 执行旧字段和 UI 字面契约检查**

Run: `! rg -n 'json:"requestId"|gorm:"[^\"]*request_id|label="Request ID"|label="Command ID"|commandId=' server/plugin/tr069 web/src/plugin/tr069/view`

Expected: exit 0；数据库切换代码中的 SQL/列名检查不属于该匹配范围。

- [x] **Step 3: 停止 GVA 并启动新版本执行 schema cutover**

先停止当前 GVA 后端与前端进程，保持 MySQL/Redis 运行。启动新后端，让初始化逻辑完成 `request_id -> cwmp_id` 列重命名和 XML 旧列删除；若启动失败，不恢复旧二进制，先修复目标结构再重试新版本。

Run: `mysql -N -e "SELECT COLUMN_NAME FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'tr069_commands' AND COLUMN_NAME IN ('request_id','cwmp_id') ORDER BY COLUMN_NAME; SELECT COLUMN_NAME FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'tr069_command_xmls' AND COLUMN_NAME = 'request_id';"`

Expected: 第一条查询只返回 `cwmp_id`，第二条无输出。连接参数使用项目当前 MySQL 配置注入，不在命令或日志中打印密码。

- [x] **Step 4: 重启前后端并验证运行链路**

确认监听：后端 `18888`、前端 `18080`、ACS `7458`。打开 RPC 命令页面并提交一条安全查询 RPC，确认命令详情显示 CWMP ID、双向 XML 分别显示报文实际 CWMP ID，页面不显示 Command ID/Request ID。

- [x] **Step 5: 勾选 OpenSpec 任务并提交主仓库收口**

逐项核对测试证据后把 `openspec/changes/unify-tr069-identifier-semantics/tasks.md` 的 20 项改为 `[x]`。

```bash
git add openspec/changes/unify-tr069-identifier-semantics docs/superpowers/specs/2026-07-18-tr069-identifier-semantics-design.md docs/superpowers/plans/2026-07-18-tr069-identifier-semantics.md
git commit -m "docs: complete tr069 identifier semantics change"
```

## OpenSpec 任务映射检查

| OpenSpec 范围 | 计划任务 |
| --- | --- |
| 1.1-1.2 | Task 1 |
| 1.3-1.4 | Task 2（`MarkSending` 命名起点在 Task 1） |
| 2.1-2.4 | Task 3 |
| 3.1-3.4 | Task 4 |
| 4.1-4.2 | Task 5 |
| 4.3-4.4 | Task 6 |
| 5.1-5.4 | Task 7，分别复用 Task 1-6 的定向证据 |

### Task 8: 删除 ACS 外部链路头并验证 CPE 兼容性

**OpenSpec tasks:** 4.1、6.2、6.4（厂商协议栈兼容性修正）

**Files:**
- Modify: `server/plugin/tr069/middleware/trace_id_test.go`
- Modify: `server/plugin/tr069/middleware/raw_dump.go`
- Modify: `server/plugin/tr069/handler/cwmp.go`
- Modify: `server/plugin/tr069/lib/tr069-core-only/adapters/http/engine_handler_trace_test.go`
- Modify: `server/plugin/tr069/lib/tr069-core-only/adapters/http/engine_handler.go`

- [x] **Step 1: 先写边界失败测试**

GVA 与 core 测试都向请求注入任意外部链路头，断言内部 Trace ID 是新生成 UUID、不会复用外部值，并断言响应不返回内部 Trace ID。

- [x] **Step 2: 运行定向测试并确认 RED**

Run: `cd server && go test ./plugin/tr069/middleware -run TestEnsureTraceID -count=1`

Run: `cd server/plugin/tr069/lib/tr069-core-only && go test ./adapters/http -run TraceID -count=1`

Expected: FAIL，旧实现仍读取或回写外部链路头。

- [x] **Step 3: 最小实现内部 Trace ID**

两个 HTTP 入口都直接生成 UUID。GVA handler 的安全回退同样只生成 UUID；任何路径都不从请求头取值，也不把 Trace ID 写到响应头。

- [x] **Step 4: 全量测试、构建与 BS 实际验证**

Run core 和 GVA TR-069 全量测试及后端构建，重启 GVA 前后端后触发 BS Inform。新交互必须正常收到 ACS 响应，且 BS 新日志中不再出现未知链路头错误。
