# TR-069 Core XML Observability Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add independently hot-configurable TR-069 core logging that records exact inbound/outbound XML at INFO and internal protocol stages at DEBUG through a GVA-owned rotating XML writer.

**Architecture:** `tr069-core-only` owns a dependency-free observability contract and emits events from Parser, Engine, Machine, Executor selection, correlation, and Builder. GVA injects one Sink into Parser, Builder, and Engine, then applies `coreLevel`, asynchronous XML framing, 100 MB rotation, 30-day retention, and config-file hot reload without coupling core to GVA or Zap.

**Tech Stack:** Go 1.24, `context`, `sync/atomic`, Viper, fsnotify, Zap, standard-library file/XML utilities, Go race detector

## Global Constraints

- Work in two independent Git repositories: parent `gva-acs` on `dev`; nested ignored `tr069-core-only` based on `V2` on a new `feat/xml-observability` branch.
- Never add `server/plugin/tr069/lib/tr069-core-only` to the parent repository index.
- `INFO` records exact inbound/outbound XML bytes without parsing, reformatting, JSON conversion, or redaction.
- `DEBUG` includes INFO plus Parser, Session, Machine, Executor, correlation, and Builder trace events; it must not duplicate full Message parameter trees.
- `tr069.observability.coreLevel` is independent of GVA `zap.level` and hot reloads without rebuilding the Engine.
- Log files use `0600`, rotate at 100 MB, and retain 30 days by default.
- Queue, directory, or disk failures never change CWMP HTTP/SOAP/session/command behavior; they emit rate-limited GVA Zap warnings.
- Preserve the existing `/tr069/debug/trace/:requestId` compatibility endpoint during this change.
- Use TDD for every behavior and commit core and GVA changes separately.

---

### Task 1: Core observability contract and context correlation

**Repository:** `server/plugin/tr069/lib/tr069-core-only`

**Files:**
- Create: `observability/event.go`
- Create: `observability/context.go`
- Create: `observability/sink.go`
- Test: `observability/event_test.go`

**Interfaces:**
- Produces: `Level`, `Direction`, `Attributes`, `Event`, `EventSink`, `NopSink`, `NormalizeSink`, `WithAttributes`, `AttributesFromContext`, and `Enrich`.
- Consumers: Tasks 2–3 core instrumentation and Tasks 4–7 GVA Sink.

- [ ] **Step 1: Create the core feature branch**

```bash
cd server/plugin/tr069/lib/tr069-core-only
git switch -c feat/xml-observability
git status --short --branch
```

Expected: branch is `feat/xml-observability`, based on clean `V2` commit `35a003f` or its current successor.

- [ ] **Step 2: Write failing contract and context tests**

Create tests that assert level ordering, nil normalization, and metadata enrichment without mutating the parent context:

```go
func TestNormalizeSinkNilUsesNoop(t *testing.T) {
    sink := NormalizeSink(nil)
    if sink.Enabled(LevelError) || sink.Enabled(LevelDebug) {
        t.Fatal("nil sink must normalize to a disabled sink")
    }
    sink.Emit(context.Background(), Event{Level: LevelInfo})
}

func TestContextAttributesAreMerged(t *testing.T) {
    ctx := WithAttributes(context.Background(), Attributes{RequestID: "req-1", DeviceKey: "dev-1"})
    ctx = WithAttributes(ctx, Attributes{CommandID: "cmd-1", DeviceKey: "dev-2"})
    got := AttributesFromContext(ctx)
    want := Attributes{RequestID: "req-1", DeviceKey: "dev-2", CommandID: "cmd-1"}
    if got != want { t.Fatalf("got %#v want %#v", got, want) }
}

func TestEnrichUsesContextForMissingFields(t *testing.T) {
    ctx := WithAttributes(context.Background(), Attributes{RequestID: "req-1", CWMPID: "cwmp-1"})
    got := Enrich(ctx, Event{Method: "Inform"})
    if got.RequestID != "req-1" || got.CWMPID != "cwmp-1" || got.Method != "Inform" {
        t.Fatalf("unexpected event: %#v", got)
    }
}
```

- [ ] **Step 3: Run tests and verify RED**

Run: `go test ./observability`

Expected: FAIL because package/types do not exist.

- [ ] **Step 4: Implement the minimal contract**

Implement exact public shapes:

```go
type Level uint8
const ( LevelOff Level = iota; LevelError; LevelWarn; LevelInfo; LevelDebug )

type Direction string
const ( DirectionNone Direction = ""; DirectionInbound Direction = "inbound"; DirectionOutbound Direction = "outbound" )

type Attributes struct {
    RequestID, SessionID, DeviceKey, CommandID, CWMPID, Method string
}

type Event struct {
    Time time.Time
    Level Level
    Stage string
    Direction Direction
    Attributes
    Payload []byte
    Fields map[string]string
}

type EventSink interface {
    Enabled(Level) bool
    Emit(context.Context, Event)
}
```

Use an unexported typed context key. `WithAttributes` merges non-empty new values over existing values. `Enrich` copies only missing event attributes from context and fills zero `Time` with `time.Now()`.

- [ ] **Step 5: Verify GREEN and race safety**

Run:

```bash
go test ./observability
go test -race ./observability
```

Expected: PASS.

- [ ] **Step 6: Commit core contract**

```bash
git add observability
git commit -m "feat: add core observability event contract"
```

---

### Task 2: Inject Sink into core Config, Parser, and Builder

**Repository:** `server/plugin/tr069/lib/tr069-core-only`

**Files:**
- Modify: `interface/config.go`
- Modify: `config/config.go`
- Modify: `factory/factory.go`
- Modify: `parser/auto.go`
- Modify: `internal/builder/builder.go`
- Test: `parser/auto_observability_test.go`
- Test: `internal/builder/builder_observability_test.go`

**Interfaces:**
- Consumes: `observability.EventSink`, `NormalizeSink`, context attributes from Task 1.
- Produces: `tr069.WithEventSink(sink)`, `Config.GetEventSink()`, and `Config.SetEventSink(sink)`.

- [ ] **Step 1: Write Parser and Builder failing tests**

Use a thread-safe recording Sink and assert:

```go
func TestBuilderEmitsDebugStartAndComplete(t *testing.T) {
    sink := newRecordingSink(observability.LevelDebug)
    b := factory.NewBuilder(tr069.WithEventSink(sink))
    ctx := observability.WithAttributes(context.Background(), observability.Attributes{RequestID: "req-1"})
    out, err := b.BuildMessage(ctx, &tr069.Message{ID: "cwmp-1", Method: tr069.MethodInformResponse, MaxEnvelopes: 1})
    if err != nil || len(out) == 0 { t.Fatalf("build: %v", err) }
    assertStages(t, sink.Events(), "builder.started", "builder.completed")
    assertEventFields(t, sink.Events()[1], "req-1", "cwmp-1", tr069.MethodInformResponse)
}

func TestParserDebugDisabledDoesNotEmit(t *testing.T) {
    sink := newRecordingSink(observability.LevelInfo)
    p := factory.NewParser(tr069.WithEventSink(sink))
    _, err := p.ParseMessage(context.Background(), []byte(sampleInform))
    if err != nil { t.Fatal(err) }
    if len(sink.Events()) != 0 { t.Fatalf("unexpected debug events: %#v", sink.Events()) }
}
```

Add failure-path assertions for `parser.failed` and `builder.failed` at `LevelError`.

- [ ] **Step 2: Run targeted tests and verify RED**

Run:

```bash
go test ./parser -run Observability
go test ./internal/builder -run Observability
```

Expected: FAIL because `WithEventSink` and emitted stages do not exist.

- [ ] **Step 3: Extend configuration without breaking existing callers**

Add to `interface.Config` and its concrete implementation:

```go
GetEventSink() observability.EventSink
SetEventSink(observability.EventSink)

func WithEventSink(sink observability.EventSink) Option {
    return func(c Config) { c.SetEventSink(sink) }
}
```

Store the normalized Sink behind the concrete config mutex. Existing factory calls without the option must continue using a no-op Sink.

- [ ] **Step 4: Instrument Parser and Builder**

Pass the configured Sink from factory constructors. Emit:

```go
if sink.Enabled(observability.LevelDebug) {
    sink.Emit(ctx, observability.Enrich(ctx, observability.Event{
        Level: observability.LevelDebug,
        Stage: "builder.started",
        Attributes: observability.Attributes{CWMPID: msg.ID, Method: msg.Method},
    }))
}
start := time.Now()
// existing build/parse logic
// completed Fields: elapsed, bytes; failed Fields: elapsed, error
```

Do not put full XML in Parser/Builder DEBUG events. Final XML is emitted once by Engine in Task 3.

- [ ] **Step 5: Verify tests and existing package behavior**

Run:

```bash
go test ./parser ./internal/builder ./factory ./config ./interface
go test -race ./parser ./internal/builder
```

Expected: PASS.

- [ ] **Step 6: Commit core codec instrumentation**

```bash
git add interface config factory parser internal/builder
git commit -m "feat: trace parser and builder stages"
```

---

### Task 3: Emit exact wire XML and internal Engine/Machine trace

**Repository:** `server/plugin/tr069/lib/tr069-core-only`

**Files:**
- Modify: `pkg/core/config.go`
- Modify: `pkg/core/engine_impl.go`
- Modify: `pkg/core/machine.go`
- Test: `pkg/core/engine_observability_test.go`

**Interfaces:**
- Consumes: `observability.EventSink` and codec instrumentation from Tasks 1–2.
- Produces: `core.Config.EventSink observability.EventSink` and stages `wire.received`, `wire.sent`, `session.resolved`, `machine.transition`, `command.pulled`, `executor.request-built`, and `correlation.completed`.

- [ ] **Step 1: Write failing exact-wire tests**

Create an engine with a recording Sink and deterministic IDs. Assert exact byte equality rather than string containment:

```go
func TestEngineEmitsExactInboundAndOutboundXML(t *testing.T) {
    sink := newRecordingSink(observability.LevelInfo)
    engine := newTestEngine(t, sink)
    inbound := []byte(sampleInformXMLBoot)
    resp, err := engine.Handle(context.Background(), &core.Request{ID: "req-1", RemoteIP: "203.0.113.10", Body: inbound})
    if err != nil { t.Fatal(err) }
    got := sink.Events()
    assertPayloadEqual(t, findStage(got, "wire.received").Payload, inbound)
    assertPayloadEqual(t, findStage(got, "wire.sent").Payload, resp.Body)
}

func TestInfoOmitsInternalTrace(t *testing.T) {
    sink := newRecordingSink(observability.LevelInfo)
    runInform(t, newTestEngine(t, sink))
    if hasDebugEvent(sink.Events()) { t.Fatal("INFO emitted DEBUG trace") }
}

func TestDebugCorrelatesCommandAndCWMPID(t *testing.T) {
    sink := newRecordingSink(observability.LevelDebug)
    events := runInformPollAndCommand(t, newTestEngine(t, sink))
    event := findStage(events, "executor.request-built")
    if event.CommandID != "cmd-1" || event.CWMPID == "" || event.DeviceKey == "" { t.Fatalf("bad correlation: %#v", event) }
}
```

Add a parser-error test asserting the built SOAP Fault is emitted as `wire.sent`.

- [ ] **Step 2: Verify RED**

Run: `go test ./pkg/core -run 'Observability|ExactInbound|InfoOmits|DebugCorrelates'`

Expected: FAIL because Engine has no EventSink or wire stages.

- [ ] **Step 3: Add EventSink to core Config and centralize response emission**

Add `EventSink observability.EventSink` and normalize it in `withDefaults`. Refactor repeated response creation through helpers that always emit outbound bytes before returning:

```go
func (e *DefaultEngine) wire(ctx context.Context, level observability.Level, stage string, direction observability.Direction, payload []byte, attrs observability.Attributes) {
    if !e.conf.EventSink.Enabled(level) { return }
    e.conf.EventSink.Emit(ctx, observability.Enrich(ctx, observability.Event{
        Level: level, Stage: stage, Direction: direction, Attributes: attrs, Payload: payload,
    }))
}
```

Emit `wire.received` before parsing non-empty bodies and `wire.sent` for every non-empty normal/Fault response. Empty POST and HTTP 204 produce DEBUG lifecycle events but no fake XML payload.

- [ ] **Step 4: Instrument Machine decisions without copying parameter trees**

Emit DEBUG events around method dispatch, state transitions, command pull, executor selection/build, and CorrelationHook. Fields are limited to scalar summaries such as `previousState`, `nextState`, `operation`, `parameterCount`, `handled`, and `elapsed`.

- [ ] **Step 5: Run complete core verification**

Run:

```bash
go test ./...
go test -race ./observability ./parser ./internal/builder ./pkg/core
```

Expected: all core packages PASS.

- [ ] **Step 6: Commit and push the core feature branch**

```bash
git add pkg/core
git commit -m "feat: emit correlated core wire events"
git push -u origin feat/xml-observability
```

Record the pushed core commit SHA for the GVA integration report.

---

### Task 4: GVA config, level filter, and XML event framing

**Repository:** parent `gva-acs`

**Files:**
- Modify: `server/plugin/tr069/config/config.go`
- Create: `server/plugin/tr069/observability/config.go`
- Create: `server/plugin/tr069/observability/encoder.go`
- Test: `server/plugin/tr069/observability/config_test.go`
- Test: `server/plugin/tr069/observability/encoder_test.go`
- Modify: `server/config.yaml`
- Modify: `server/config.docker.yaml`

**Interfaces:**
- Consumes: core `observability.Event` from Tasks 1–3.
- Produces: `ObservabilityConfig`, `ParseCoreLevel`, `RuntimeConfig`, and `EncodeEvent(event) ([]byte, error)`.

- [ ] **Step 1: Write failing config and encoder tests**

Cover `OFF/ERROR/WARN/INFO/DEBUG`, invalid-level rejection, default values, XML attribute escaping, raw payload byte preservation, and DEBUG trace encoding:

```go
func TestEncodeWireEventPreservesPayload(t *testing.T) {
    payload := []byte(`<?xml version="1.0"?><SOAP-ENV:Envelope><Value>a&amp;b</Value></SOAP-ENV:Envelope>`)
    out, err := EncodeEvent(coreobs.Event{Level: coreobs.LevelInfo, Stage: "wire.sent", Direction: coreobs.DirectionOutbound, Payload: payload})
    if err != nil { t.Fatal(err) }
    if !bytes.Contains(out, payload) { t.Fatalf("payload changed: %q", out) }
    if bytes.Count(out, payload) != 1 { t.Fatalf("payload duplicated: %q", out) }
}
```

- [ ] **Step 2: Verify RED**

Run: `cd server && go test ./plugin/tr069/observability`

Expected: FAIL because package/config types do not exist.

- [ ] **Step 3: Add exact YAML config**

```go
type ObservabilityConfig struct {
    CoreLevel    string `mapstructure:"coreLevel" json:"coreLevel" yaml:"coreLevel"`
    Directory    string `mapstructure:"directory" json:"directory" yaml:"directory"`
    RetentionDays int   `mapstructure:"retentionDays" json:"retentionDays" yaml:"retentionDays"`
    MaxFileSizeMB int   `mapstructure:"maxFileSizeMB" json:"maxFileSizeMB" yaml:"maxFileSizeMB"`
}
```

Add it under `TR069Config.Observability`. Defaults: `INFO`, `./log/tr069`, `30`, `100`.

- [ ] **Step 4: Implement XML framing**

Encode one processing-instruction start line with escaped attributes, append raw wire payload unchanged, then append `<?tr069-event-end?>`. For DEBUG events encode one `<tr069:trace xmlns:tr069="urn:gva-acs:tr069:log:1" .../>` element with sorted field names for deterministic output. Never marshal wire payload through `encoding/xml`.

- [ ] **Step 5: Verify and commit config/encoding**

```bash
cd server
go test ./plugin/tr069/observability
cd ..
git add server/plugin/tr069/config server/plugin/tr069/observability server/config.yaml server/config.docker.yaml
git commit -m "feat: define TR-069 XML log configuration"
```

---

### Task 5: Async writer, rotation, retention, and failure isolation

**Repository:** parent `gva-acs`

**Files:**
- Create: `server/plugin/tr069/observability/writer.go`
- Create: `server/plugin/tr069/observability/rotator.go`
- Create: `server/plugin/tr069/observability/sink.go`
- Test: `server/plugin/tr069/observability/writer_test.go`
- Test: `server/plugin/tr069/observability/sink_test.go`

**Interfaces:**
- Consumes: `RuntimeConfig` and `EncodeEvent` from Task 4.
- Produces: `NewSink(cfg, zapLogger) *Sink`, `InitializeDefault(cfg, zapLogger) (*Sink, error)`, `DefaultSink() coreobs.EventSink`, `Sink.Update(RuntimeConfig) error`, `Sink.Close(context.Context) error`, and core `EventSink` methods.

- [ ] **Step 1: Write failing writer behavior tests**

Use `t.TempDir`, injectable clock, and small byte thresholds to prove:

- file mode is `0600`;
- events remain ordered and payload appears once;
- size/date changes create `tr069-wire-0002.xml.log` or a new day directory;
- cleanup removes files older than retention but not active files;
- INFO rejects DEBUG while DEBUG accepts both;
- queue-full and write failures increment drop/error counters and do not panic;
- `Update` changes level atomically.

- [ ] **Step 2: Verify RED**

Run: `cd server && go test ./plugin/tr069/observability -run 'Writer|Sink|Rotate|Retention'`

Expected: FAIL because writer/sink do not exist.

- [ ] **Step 3: Implement bounded asynchronous ownership**

`Emit` must recover from its own panic boundary, check the atomic level, encode one event, reserve queue bytes, and enqueue without blocking. The queued value owns the encoded byte slice, so core payload cannot race with the writer. A single goroutine writes complete encoded frames with one `Write` loop.

Use a byte-budget queue (default internal budget 32 MiB); if a frame exceeds available budget, drop it and rate-limit a Zap warning containing only stage, IDs, byte count, and cumulative drops.

Store the initialized GVA Sink behind a package mutex/atomic pointer. `DefaultSink` returns the initialized Sink or core `NopSink`; this global belongs only to the singleton GVA plugin process and never leaks into `tr069-core-only`.

- [ ] **Step 4: Implement rotator and retention**

Open files with `os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)`. Track current date, segment number, and bytes written. Rotate before a frame would exceed the configured limit. Run retention cleanup on startup and once per day; only delete matching `tr069-wire-*.xml.log` files older than the cutoff.

- [ ] **Step 5: Verify normal and race behavior**

```bash
cd server
go test ./plugin/tr069/observability
go test -race ./plugin/tr069/observability
```

Expected: PASS with no race reports.

- [ ] **Step 6: Commit writer**

```bash
cd ..
git add server/plugin/tr069/observability
git commit -m "feat: add rotating TR-069 XML writer"
```

---

### Task 6: Independent config-file hot reload

**Repository:** parent `gva-acs`

**Files:**
- Create: `server/plugin/tr069/observability/reloader.go`
- Test: `server/plugin/tr069/observability/reloader_test.go`
- Modify: `server/plugin/tr069/initialize/viper.go`

**Interfaces:**
- Consumes: `Sink.Update(RuntimeConfig)` from Task 5.
- Produces: `StartConfigReloader(ctx, configFile, apply, logger) error` and legacy-to-new config mapping.

- [ ] **Step 1: Write failing hot-reload tests**

Create a temporary YAML file, start the reloader, atomically replace the file, and wait on a condition channel rather than sleeping a fixed duration. Assert `INFO → DEBUG → OFF` is applied, invalid values preserve the last valid config, and directory/size/retention updates reach `Sink.Update`.

- [ ] **Step 2: Verify RED**

Run: `cd server && go test ./plugin/tr069/observability -run Reload`

Expected: FAIL because the reloader does not exist.

- [ ] **Step 3: Implement an isolated fsnotify watcher**

Watch the containing directory from `global.GVA_VP.ConfigFileUsed()` so atomic file replacement is detected. Debounce write/create/rename events, read a fresh Viper instance, unmarshal only `tr069`, validate all observability fields, then call `apply`. Do not call `global.GVA_VP.OnConfigChange`, because GVA core already owns that single callback.

Invalid config logs one Zap error and leaves the active Sink unchanged.

- [ ] **Step 4: Implement legacy mapping**

When `tr069.observability` is absent:

```go
if cfg.DumpRaw || cfg.InfoLogEnable { cfg.Observability.CoreLevel = "INFO" }
if cfg.InfoLogDir != "" { cfg.Observability.Directory = cfg.InfoLogDir }
```

Emit one deprecation warning. New config always wins.

- [ ] **Step 5: Verify and commit reloader**

```bash
cd server
go test ./plugin/tr069/observability ./plugin/tr069/initialize
cd ..
git add server/plugin/tr069/observability/reloader.go server/plugin/tr069/observability/reloader_test.go server/plugin/tr069/initialize/viper.go
git commit -m "feat: hot reload TR-069 core log level"
```

---

### Task 7: Wire GVA Engine and retire duplicate raw logging

**Repository:** parent `gva-acs`

**Files:**
- Modify: `server/plugin/tr069/engine/engine.go`
- Modify: `server/plugin/tr069/handler/cwmp.go`
- Modify: `server/plugin/tr069/initialize/server.go`
- Create: `server/plugin/tr069/middleware/request_id.go`
- Modify: `server/plugin/tr069/service/command.go`
- Modify: `server/plugin/tr069/adapter/connection_request.go`
- Delete: `server/plugin/tr069/middleware/raw_dump.go`
- Delete: `server/plugin/tr069/middleware/raw_response_dump.go`
- Delete: `server/plugin/tr069/middleware/raw_response_dump_test.go`
- Delete: `server/plugin/tr069/infolog/infolog.go`
- Modify: `server/plugin/tr069/engine/engine_test.go`
- Test: `server/plugin/tr069/initialize/server_observability_test.go`

**Interfaces:**
- Consumes: core `WithEventSink`, `core.Config.EventSink`, and GVA `Sink`/reloader.
- Produces: one shared Sink used by Parser, Builder, and Engine for the lifetime of the TR-069 server.

- [ ] **Step 1: Write failing GVA integration tests**

Build a test Engine with the GVA Sink and verify an Inform produces exactly one inbound and one outbound wire frame. Set `coreLevel=DEBUG`, execute the existing onboarding/command flow, and assert `parser.completed`, `executor.request-built`, and `builder.completed` share correlation fields.

Add a server middleware test proving the response is not captured by obsolete RawDump middleware.

- [ ] **Step 2: Verify RED**

Run:

```bash
cd server
go test ./plugin/tr069/engine ./plugin/tr069/initialize -run Observability
```

Expected: FAIL because the shared Sink is not injected.

- [ ] **Step 3: Inject one Sink into all core components**

Extend `engine.Deps` with `EventSink coreobs.EventSink`. Resolve one Sink, then construct:

```go
parser := factory.NewParser(tr069.WithStrictMode(false), tr069.WithEventSink(sink))
builder := factory.NewBuilder(tr069.WithEventSink(sink))
return core.NewEngine(core.Config{
    Parser: parser, Builder: builder, EventSink: sink,
    // existing repositories, queue, registry and hook unchanged
})
```

Initialize the GVA Sink before starting the TR-069 Gin Engine and start the reloader with a process-lifetime context. `Sink.Close` is mandatory in tests and explicit test servers; the current production Gin server has no plugin shutdown hook, so this change must not invent an unrelated lifecycle framework.

Because `handler.CWMPHandler` obtains a lazy singleton through `engine.Get`, make `engine.New` resolve `tr069observability.DefaultSink()` when `Deps.EventSink` is nil. `initialize.StartTR069Server` must call `InitializeDefault` before `SetupEngine`, so the first request cannot construct an Engine with a no-op Sink. Tests that pass `Deps.EventSink` remain isolated from the global default.

- [ ] **Step 4: Preserve request correlation and remove duplicate writers**

Move `EnsureRequestID` unchanged from `raw_dump.go` to `middleware/request_id.go`, and keep `trace.WithRequestID` for compatibility. Add core observability attributes to the request context before `eng.Handle`. Remove RawDump/RawResponseDump registration and files. Replace command and Connection Request `fmt`/`infolog` blocks with structured GVA Zap summaries; never include XML or secrets in Zap.

- [ ] **Step 5: Run plugin tests**

```bash
cd server
go test ./plugin/tr069/...
go test -race ./plugin/tr069/observability ./plugin/tr069/engine ./plugin/tr069/handler
```

Expected: PASS.

- [ ] **Step 6: Commit GVA integration**

```bash
cd ..
git add server/plugin/tr069 server/config.yaml server/config.docker.yaml
git commit -m "feat: integrate core XML observability"
```

---

### Task 8: Cross-repository verification and branch delivery

**Repositories:** both

**Files:**
- Verify: all files from Tasks 1–7
- Update if needed: `server/plugin/tr069/docs/application_layer_features.md`

**Interfaces:**
- Consumes: completed core and GVA implementations.
- Produces: verified core feature branch and updated GVA `dev` branch.

- [ ] **Step 1: Verify core independently**

```bash
cd server/plugin/tr069/lib/tr069-core-only
go test ./...
go test -race ./observability ./parser ./internal/builder ./pkg/core
git status --short --branch
```

Expected: all tests PASS; only intentional documentation changes, if any, remain.

- [ ] **Step 2: Verify GVA plugin and server build**

```bash
cd /root/code/gva-acs/gva-acs/server
go test ./plugin/tr069/...
go test -race ./plugin/tr069/observability ./plugin/tr069/engine ./plugin/tr069/handler
go build ./...
```

Expected: tests and build PASS.

- [ ] **Step 3: Run a live BS/GVA smoke test**

Start GVA/TR-069 on port 7458, use the existing `oamstart` Skill to confirm BS connectivity, then verify:

- INFO creates one inbound Inform XML and one outbound InformResponse XML;
- DEBUG adds internal trace events without duplicating XML;
- changing `coreLevel` in the active config takes effect on the next event without restart;
- the log segment uses `0600` and follows the date/segment path;
- restoring the config leaves no test-only setting behind.

- [ ] **Step 4: Audit parent repository boundaries**

```bash
cd /root/code/gva-acs/gva-acs
git diff --check
git check-ignore -v server/plugin/tr069/lib/tr069-core-only
if git ls-files | rg 'server/plugin/tr069/lib/tr069-core-only'; then exit 1; fi
git status --short --branch
```

Expected: nested core remains ignored and untracked by parent Git.

- [ ] **Step 5: Update documentation and commit only if verification changed it**

Document the coreLevel matrix, sensitive XML warning, log path, hot reload, and legacy mappings. If modified:

```bash
git add server/plugin/tr069/docs/application_layer_features.md
git commit -m "docs: document TR-069 XML observability"
```

- [ ] **Step 6: Push both branches without merging core into V2**

```bash
git -C server/plugin/tr069/lib/tr069-core-only push origin feat/xml-observability
git push origin dev
```

Expected: core feature branch and GVA `dev` are available remotely; GVA `main` and core `V2` remain unchanged pending later review/merge.
