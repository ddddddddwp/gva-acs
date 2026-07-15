# GVA TR-069 RPC Command Management Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add durable, permission-controlled management for all 12 BS-advertised CWMP RPCs, with strict per-device FIFO, three independently configurable deadlines, complete XML/history records, retries, concrete device-list actions, and a GVA-style “RPC 记录” page.

**Architecture:** MySQL is the command and timeline source of truth. Redis stores only device wake-ups and short-lived locks; every dispatch re-reads the head non-terminal database command. The standalone core constructs/parses CWMP and emits protocol events. GVA validates device state/capabilities, owns state transitions and deadlines, persists structured results/XML, exposes typed APIs, and renders the operation/record UI.

**Tech Stack:** Go, Gin, GORM/MySQL, Redis, Viper/fsnotify, GVA JWT/Casbin/operation logging, Vue 3, Element Plus, Node test runner.

## Global Constraints

- Complete the independent core plan first and record its exact commit: `docs/superpowers/plans/2026-07-16-tr069-core-rpc-support.md`.
- Never add `server/plugin/tr069/lib/tr069-core-only` to the parent repository; it remains ignored and separately maintained.
- Follow test-driven development for every task: failing focused test, minimum implementation, focused pass, wider regression pass.
- The accepted command states are exactly `QUEUED`, `WAITING_DEVICE`, `BUILDING`, `SENT`, `WAITING_TRANSFER`, `COMPLETED`, `FAILED`, and `TIMEOUT`.
- Commands for one device are strict FIFO. A deferred `Download`/`Upload` remains the head while in `WAITING_TRANSFER`, so later commands cannot pass it.
- Database writes are authoritative. Redis failure must leave a traceable terminal database record, never an untracked or permanently pending command.
- All submit endpoints are asynchronous and return `{commandId, status}` immediately. Remove the current 15-second synchronous `GetParameterValues` waiter.
- Complete XML is stored exactly as sent/received without redaction and is visible only through the record-detail permission. XML expires; command rows, structured results, and events do not.
- `commandQueueWaitTimeout`, `rpcResponseTimeout`, and `transferCompleteTimeout` are independent. A config reload changes only deadlines created after the reload.
- Query endpoints and command endpoints require GVA JWT/Casbin. Mutations also use GVA `OperationRecord` middleware.
- Keep the existing parameter-tree display unchanged; access remains from the device list’s “参数” action.

---

### Task 1: Add atomic TR-069 runtime configuration and hot reload

**Files:**

- Modify: `server/plugin/tr069/config/config.go`
- Create: `server/plugin/tr069/config/runtime.go`
- Create: `server/plugin/tr069/config/runtime_test.go`
- Modify: `server/plugin/tr069/global/global.go`
- Modify: `server/plugin/tr069/initialize/viper.go`
- Modify: `server/utils/system_events.go`
- Modify: `server/core/viper.go`
- Modify: `server/plugin/tr069/plugin.go`
- Modify: `server/config.yaml`
- Modify: `server/config.local.yaml`

- [ ] Add a failing defaults/snapshot test for the four new values.

```go
func TestNormalizeRuntimeConfigDefaults(t *testing.T) {
	got := NormalizeRuntimeConfig(TR069Config{})
	if got.CommandQueueWaitTimeout != 180 { t.Fatalf("wait=%d", got.CommandQueueWaitTimeout) }
	if got.RPCResponseTimeout != 90 { t.Fatalf("response=%d", got.RPCResponseTimeout) }
	if got.TransferCompleteTimeout != 43200 { t.Fatalf("transfer=%d", got.TransferCompleteTimeout) }
	if got.RPCXMLRetentionDays != 30 { t.Fatalf("retention=%d", got.RPCXMLRetentionDays) }
}
```

- [ ] Add fields to `TR069Config` using seconds/days as the YAML representation.

```go
CommandQueueWaitTimeout int `mapstructure:"commandQueueWaitTimeout" json:"commandQueueWaitTimeout" yaml:"commandQueueWaitTimeout"`
RPCResponseTimeout int `mapstructure:"rpcResponseTimeout" json:"rpcResponseTimeout" yaml:"rpcResponseTimeout"`
TransferCompleteTimeout int `mapstructure:"transferCompleteTimeout" json:"transferCompleteTimeout" yaml:"transferCompleteTimeout"`
RPCXMLRetentionDays int `mapstructure:"rpcXMLRetentionDays" json:"rpcXMLRetentionDays" yaml:"rpcXMLRetentionDays"`
```

- [ ] Implement an immutable atomic snapshot so workers never race with Viper mutations.

```go
type Runtime struct {
	Settings TR069Config
	CommandQueueWaitTimeout time.Duration
	RPCResponseTimeout time.Duration
	TransferCompleteTimeout time.Duration
	RPCXMLRetention time.Duration
}

var current atomic.Pointer[Runtime]

func StoreRuntime(in TR069Config) Runtime {
	normalized := NormalizeRuntimeConfig(in)
	next := Runtime{
		Settings: normalized,
		CommandQueueWaitTimeout: time.Duration(normalized.CommandQueueWaitTimeout) * time.Second,
		RPCResponseTimeout: time.Duration(normalized.RPCResponseTimeout) * time.Second,
		TransferCompleteTimeout: time.Duration(normalized.TransferCompleteTimeout) * time.Second,
		RPCXMLRetention: time.Duration(normalized.RPCXMLRetentionDays) * 24 * time.Hour,
	}
	current.Store(&next)
	return next
}

func CurrentRuntime() Runtime {
	if value := current.Load(); value != nil { return *value }
	return StoreRuntime(TR069Config{})
}
```

- [ ] Extend `SystemEvents` with `RegisterConfigChangeHandler(func())` and `TriggerConfigChange()`. Snapshot the handler slice under lock and invoke it after unlocking.

- [ ] In `server/core/viper.go`, call `utils.GlobalSystemEvents.TriggerConfigChange()` after a successful `v.Unmarshal(&global.GVA_CONFIG)` in the existing fsnotify callback. Do not trigger the heavyweight database reload path.

- [ ] Refactor `initialize.Viper()` into an idempotent `ReloadConfig()` that unmarshals into a local value, normalizes it, and calls `config.StoreRuntime`. Keep `tr069Global.GlobalConfig` as the startup compatibility snapshot; all hot-path readers use `CurrentRuntime().Settings`, avoiding concurrent mutation of the legacy pointer. Register `ReloadConfig` once from `plugin.Register`.

- [ ] Add the accepted values to both active example/local YAML files.

```yaml
tr069:
  commandQueueWaitTimeout: 180
  rpcResponseTimeout: 90
  transferCompleteTimeout: 43200
  rpcXMLRetentionDays: 30
```

`server/config.local.yaml` is intentionally ignored; update it for local execution but do not force-add it to Git.

- [ ] Verify a reload changes `CurrentRuntime()` but a deadline already calculated in a test retains its original timestamp.

Run: `go test ./plugin/tr069/config ./utils -count=1`

- [ ] Commit the configuration slice.

```bash
git add server/plugin/tr069/config server/plugin/tr069/global/global.go server/plugin/tr069/initialize/viper.go server/utils/system_events.go server/core/viper.go server/plugin/tr069/plugin.go server/config.yaml
git commit -m "feat(tr069): add hot-reloadable RPC deadlines"
```

### Task 2: Persist commands, transitions, events, and complete XML

**Files:**

- Modify: `server/plugin/tr069/model/command.go`
- Create: `server/plugin/tr069/model/command_event.go`
- Create: `server/plugin/tr069/model/command_xml.go`
- Create: `server/plugin/tr069/model/command_status.go`
- Modify: `server/plugin/tr069/initialize/gorm.go`
- Create: `server/plugin/tr069/service/command_store.go`
- Create: `server/plugin/tr069/service/command_store_test.go`

- [ ] Add model tests using the existing in-memory SQLite test pattern. Prove command creation and the initial `CREATED` event commit together, and both roll back when event creation fails.

- [ ] Replace the legacy command shape with fields required for recovery, filtering, and optimistic transitions.

```go
type Command struct {
	CommandID string `json:"commandId" gorm:"primaryKey;size:64"`
	DeviceID uint `json:"deviceId" gorm:"index:idx_tr069_command_device_head,priority:1"`
	DeviceKey string `json:"deviceKey" gorm:"size:128;index"`
	Operation string `json:"operation" gorm:"size:64;index"`
	ParamsJSON datatypes.JSON `json:"params" gorm:"type:longtext"`
	ResultJSON datatypes.JSON `json:"result" gorm:"type:longtext"`
	RetryOf string `json:"retryOf" gorm:"size:64;index"`
	DedupKey string `json:"dedupKey" gorm:"size:128;index"`
	CommandKey *string `json:"commandKey" gorm:"size:128;uniqueIndex"`
	Status string `json:"status" gorm:"size:24;index;index:idx_tr069_command_device_head,priority:2"`
	RequestID string `json:"requestId" gorm:"size:64;index"`
	PhaseDeadlineAt *time.Time `json:"phaseDeadlineAt" gorm:"index"`
	QueuedAt time.Time `json:"queuedAt"`
	WaitingAt *time.Time `json:"waitingAt"`
	BuildingAt *time.Time `json:"buildingAt"`
	SentAt *time.Time `json:"sentAt"`
	FinishedAt *time.Time `json:"finishedAt"`
	FailureStage string `json:"failureStage" gorm:"size:64"`
	FaultCode int `json:"faultCode"`
	FaultString string `json:"faultString" gorm:"type:text"`
	Version uint `json:"version"`
	CreatedAt time.Time `json:"createdAt" gorm:"index:idx_tr069_command_device_head,priority:3"`
	UpdatedAt time.Time `json:"updatedAt"`
}
```

- [ ] Add append-only event and XML models.

```go
type CommandEvent struct {
	ID uint64 `json:"id" gorm:"primaryKey"`
	CommandID string `json:"commandId" gorm:"size:64;index"`
	EventType string `json:"eventType" gorm:"size:48;index"`
	FromStatus string `json:"fromStatus" gorm:"size:24"`
	ToStatus string `json:"toStatus" gorm:"size:24"`
	Stage string `json:"stage" gorm:"size:64"`
	Message string `json:"message" gorm:"type:text"`
	PayloadJSON datatypes.JSON `json:"payload" gorm:"type:longtext"`
	CreatedAt time.Time `json:"createdAt" gorm:"index"`
}

type CommandXML struct {
	ID uint64 `json:"id" gorm:"primaryKey"`
	CommandID string `json:"commandId" gorm:"size:64;index"`
	Direction string `json:"direction" gorm:"size:12;index"`
	Method string `json:"method" gorm:"size:64;index"`
	CWMPID string `json:"cwmpId" gorm:"size:64;index"`
	RequestID string `json:"requestId" gorm:"size:64;index"`
	Payload []byte `json:"xml" gorm:"type:longblob"`
	ExpiresAt time.Time `json:"expiresAt" gorm:"index"`
	CreatedAt time.Time `json:"createdAt"`
}
```

- [ ] Define status constants, terminal/non-terminal helpers, and an explicit transition map. `WAITING_TRANSFER` may only follow `SENT`; terminal states have no outgoing transitions.

- [ ] Implement `CommandStore.Create`, `Transition`, `AppendEvent`, `SaveXML`, `HeadForDevice`, `List`, and `Detail`. `Transition` must use `WHERE command_id = ? AND status IN ? AND version = ?`, increment `version`, and append the transition event in the same transaction.

- [ ] Add idempotency tests: two concurrent `SENT -> COMPLETED` calls produce one state transition/event; the loser reads the existing terminal state without adding another completion.

- [ ] Migrate all three models in `initialize.Gorm`.

Keep JSON values in `LONGTEXT` rather than converting the legacy `params_json` column to MySQL's strict JSON type; this preserves existing rows whose old value may be empty while the service still validates every new serialized document.

Run: `go test ./plugin/tr069/service -run 'CommandStore|Transition' -count=1`

- [ ] Commit the persistence slice.

```bash
git add server/plugin/tr069/model server/plugin/tr069/initialize/gorm.go server/plugin/tr069/service/command_store.go server/plugin/tr069/service/command_store_test.go
git commit -m "feat(tr069): persist RPC command lifecycle"
```

### Task 3: Define 12 typed requests and validation/capability policy

**Files:**

- Replace: `server/plugin/tr069/model/request/command.go`
- Create: `server/plugin/tr069/service/rpc_registry.go`
- Create: `server/plugin/tr069/service/rpc_registry_test.go`
- Modify: `server/plugin/tr069/model/response/device.go`
- Modify: `server/plugin/tr069/api/device.go`

- [ ] Write table-driven validation tests for required fields, allowed XSD value types, object-name shape, transfer URL/file fields, and notification range `0..2`.

- [ ] Define typed request DTOs; do not expose a generic operation/JSON endpoint.

```go
type GetParameterValuesRequest struct { Paths []string `json:"paths" binding:"required,min=1,dive,required"` }
type GetParameterNamesRequest struct { ParameterPath string `json:"parameterPath" binding:"required"`; NextLevel bool `json:"nextLevel"` }
type GetParameterAttributesRequest struct { ParameterNames []string `json:"parameterNames" binding:"required,min=1,dive,required"` }
type SetParameterValue struct { Name string `json:"name"`; Type string `json:"type"`; Value interface{} `json:"value"` }
type SetParameterValuesRequest struct { ParameterKey string `json:"parameterKey"`; Parameters []SetParameterValue `json:"parameters" binding:"required,min=1"` }
type SetParameterAttribute struct { Name string `json:"name"`; NotificationChange bool `json:"notificationChange"`; Notification int `json:"notification"`; AccessListChange bool `json:"accessListChange"`; AccessList []string `json:"accessList"` }
type SetParameterAttributesRequest struct { ParameterAttributes []SetParameterAttribute `json:"parameterAttributes" binding:"required,min=1"` }
type ObjectRequest struct { ObjectName string `json:"objectName" binding:"required"`; ParameterKey string `json:"parameterKey"` }
type DownloadRequest struct { FileType string `json:"fileType"`; URL string `json:"url"`; Username string `json:"username"`; Password string `json:"password"`; FileSize int `json:"fileSize"`; TargetFileName string `json:"targetFileName"`; DelaySeconds int `json:"delaySeconds"`; SuccessURL string `json:"successURL"`; FailureURL string `json:"failureURL"` }
type UploadRequest struct { FileType string `json:"fileType"`; URL string `json:"url"`; Username string `json:"username"`; Password string `json:"password"`; DelaySeconds int `json:"delaySeconds"` }
type RebootRequest struct { CommandKey string `json:"commandKey"` }
```

- [ ] Implement an operation registry containing method, permission group, transfer flag, and validator. Keep the exact 12-method set in one location.

```go
type RPCSpec struct {
	Method string
	Permission string
	Transfer bool
	Validate func(any) error
}

var RPCSpecs = map[string]RPCSpec{
	"GetRPCMethods": {Method: "GetRPCMethods", Permission: "query"},
	"GetParameterValues": {Method: "GetParameterValues", Permission: "query", Validate: validateGPV},
	"GetParameterNames": {Method: "GetParameterNames", Permission: "query", Validate: validateGPN},
	"GetParameterAttributes": {Method: "GetParameterAttributes", Permission: "query", Validate: validateGPA},
	"SetParameterValues": {Method: "SetParameterValues", Permission: "config", Validate: validateSPV},
	"SetParameterAttributes": {Method: "SetParameterAttributes", Permission: "config", Validate: validateSPA},
	"AddObject": {Method: "AddObject", Permission: "config", Validate: validateObject},
	"DeleteObject": {Method: "DeleteObject", Permission: "config", Validate: validateObject},
	"Download": {Method: "Download", Permission: "transfer", Transfer: true, Validate: validateDownload},
	"Upload": {Method: "Upload", Permission: "transfer", Transfer: true, Validate: validateUpload},
	"Reboot": {Method: "Reboot", Permission: "maintenance"},
	"FactoryReset": {Method: "FactoryReset", Permission: "maintenance"},
}
```

- [ ] Implement submission policy: device missing/offline or unsupported method returns an error before command creation. Always permit `GetRPCMethods` so capability discovery can recover. Empty capability data rejects the other 11 methods with a clear “请先查询设备能力” message.

- [ ] Add `rpcMethods []string` to `DeviceResponse` and batch-load `tr069_device_rpc_methods` for the page’s device IDs, avoiding one query per row.

Run: `go test ./plugin/tr069/service ./plugin/tr069/api -run 'RPC|Capability|Device' -count=1`

- [ ] Commit the typed contract slice.

```bash
git add server/plugin/tr069/model/request/command.go server/plugin/tr069/service/rpc_registry.go server/plugin/tr069/service/rpc_registry_test.go server/plugin/tr069/model/response/device.go server/plugin/tr069/api/device.go
git commit -m "feat(tr069): define typed RPC command contracts"
```

### Task 4: Build the durable FIFO dispatcher

**Files:**

- Create: `server/plugin/tr069/service/command_manager.go`
- Create: `server/plugin/tr069/service/command_manager_test.go`
- Replace: `server/plugin/tr069/adapter/redis_command_source.go`
- Modify: `server/plugin/tr069/adapter/redis_keys.go`
- Modify: `server/plugin/tr069/adapter/redis_immediate_enqueue.go`
- Modify: `server/plugin/tr069/adapter/gorm_repo.go`
- Delete: `server/plugin/tr069/service/command_execute.go`
- Delete: `server/plugin/tr069/service/command_execute_test.go`

- [ ] Add manager tests proving:

  - command and initial event are one transaction;
  - first command becomes `WAITING_DEVICE`, later commands remain `QUEUED`;
  - Download in `WAITING_TRANSFER` blocks the next command;
  - different device heads can dispatch independently;
  - Redis enqueue failure changes the newly created command to `FAILED` with stage `redis.enqueue`;
  - Download/Upload receive unique system-generated CommandKeys;
  - retry copies parameters and sets `retryOf` while preserving the original row.

- [ ] Implement asynchronous creation with a stable response.

```go
type SubmitResult struct {
	CommandID string `json:"commandId"`
	Status string `json:"status"`
}

func (m *CommandManager) Submit(ctx context.Context, deviceID uint, operation string, request any) (SubmitResult, error)
func (m *CommandManager) Retry(ctx context.Context, commandID string) (SubmitResult, error)
```

Serialize same-device submissions by locking the device row with `clause.Locking{Strength: "UPDATE"}` inside the creation transaction. This prevents two concurrent submissions from both deciding they are the head.

- [ ] Serialize the typed request once into `ParamsJSON`. Generate `uuid.NewString()` for `CommandID`; generate `rpc-<uuid>` for `CommandKey` only on Download/Upload and inject it into the core parameter map at dispatch time.

- [ ] Replace Redis payloads with device wake-up tokens. The key contract is `tr069:command:wakeup`; values are device keys. Duplicate wake-ups are harmless because database head selection is authoritative.

- [ ] Rewrite `RedisCommandSource.Pull` to:

  1. acquire the existing device lock;
  2. query the earliest non-terminal command ordered by `created_at, command_id`;
  3. return nothing when the head is `QUEUED`, `SENT`, or `WAITING_TRANSFER`;
  4. conditionally transition `WAITING_DEVICE -> BUILDING`;
  5. deserialize `ParamsJSON` and return that exact database command to core;
  6. use `ack` only to release the lock and `nack` to restore `BUILDING -> WAITING_DEVICE` if no request was sent.

- [ ] Make `GormCommandRepo.MarkSending` perform `BUILDING -> SENT`, set `request_id`, `sent_at`, and `phase_deadline_at = sentAt + CurrentRuntime().RPCResponseTimeout`, and append `REQUEST_SENT` in the same transaction.

- [ ] Make `MarkSuccess` and `MarkFail` delegate to `CommandStore.Transition` instead of `ON CONFLICT` upserts. No core callback may recreate a missing command.

- [ ] Remove `ExecuteGetParameterValues`, `waitCommandFinal`, and their polling tests. All commands now return immediately.

Run: `go test ./plugin/tr069/service ./plugin/tr069/adapter -run 'FIFO|CommandManager|RedisCommandSource|GormCommandRepo' -count=1`

- [ ] Commit the dispatch slice.

```bash
git add server/plugin/tr069/service server/plugin/tr069/adapter
git commit -m "feat(tr069): dispatch commands from durable FIFO state"
```

### Task 5: Integrate core executors and lifecycle correlation

**Files:**

- Modify: `server/plugin/tr069/engine/engine.go`
- Replace: `server/plugin/tr069/adapter/datamodel_hook.go`
- Create: `server/plugin/tr069/adapter/command_hook_test.go`
- Modify: `server/plugin/tr069/adapter/redis_inflight_repo.go`
- Modify: `server/plugin/tr069/adapter/memory_inflight_repo.go`

- [ ] Add an engine registry test asserting all 12 method names resolve to an executor. Register the new core types.

```go
r.Register(tr069.MethodGetRPCMethods, &defaults.GetRPCMethodsExecutor{})
r.Register(tr069.MethodGetParameterValues, &defaults.GetParameterValuesExecutor{})
r.Register(tr069.MethodGetParameterNames, &defaults.GetParameterNamesExecutor{DefaultPath: "Device.", DefaultNextLvl: true})
r.Register(tr069.MethodGetParameterAttributes, &defaults.GetParameterAttributesExecutor{})
r.Register(tr069.MethodSetParameterValues, &defaults.SetParameterValuesExecutor{})
r.Register(tr069.MethodSetParameterAttributes, &defaults.SetParameterAttributesExecutor{})
r.Register(tr069.MethodAddObject, &defaults.AddObjectExecutor{})
r.Register(tr069.MethodDeleteObject, &defaults.DeleteObjectExecutor{})
r.Register(tr069.MethodDownload, &defaults.DownloadExecutor{})
r.Register(tr069.MethodUpload, &defaults.UploadExecutor{})
r.Register(tr069.MethodReboot, &defaults.RebootExecutor{})
r.Register(tr069.MethodFactoryReset, &defaults.FactoryResetExecutor{})
```

- [ ] Replace the layered default/data-model hook with a GVA `CommandHook` that implements both `core.CorrelationHook` and `core.TransferCompleteHook`. It may call small data-model persistence helpers, but owns each durable lifecycle transition exactly once.

- [ ] On normal response, lookup inflight by device/CWMP ID, validate core correlation, persist a method-specific `ResultJSON`, then:

  - complete all non-transfer methods;
  - complete Download/Upload when response `Status == 0`;
  - transition to `WAITING_TRANSFER` with `phase_deadline_at = now + CurrentRuntime().TransferCompleteTimeout` when response `Status == 1`.

- [ ] On CWMP Fault, persist fault code/string/result and transition `SENT -> FAILED` with stage `cwmp.fault`.

- [ ] On `TransferComplete`, lookup the unique `command_key`; transition `WAITING_TRANSFER -> COMPLETED` when transfer fault code is zero, otherwise `WAITING_TRANSFER -> FAILED`. Repeated TransferComplete messages must only append an `IGNORED_DUPLICATE` event, not a second terminal transition.

- [ ] Keep the existing GPV/GPN/GetRPCMethods side effects: update parameter values/names/capability cache before completing the command.

- [ ] Extend inflight records with `CommandKey` only if useful for diagnostics; TransferComplete lookup remains database-backed so Redis loss/restart cannot lose completion correlation.

Run: `go test ./plugin/tr069/engine ./plugin/tr069/adapter -run 'Executor|Response|Fault|TransferComplete|Duplicate' -count=1`

- [ ] Commit the integration slice and mention the nested core commit in the message body.

```bash
git add server/plugin/tr069/engine server/plugin/tr069/adapter
git commit -m "feat(tr069): correlate complete RPC lifecycle" -m "Requires tr069-core-only revision recorded during core plan execution."
```

### Task 6: Persist core wire XML and DEBUG protocol events

**Files:**

- Create: `server/plugin/tr069/adapter/core_event_sink.go`
- Create: `server/plugin/tr069/adapter/core_event_sink_test.go`
- Modify: `server/plugin/tr069/engine/engine.go`
- Modify: `server/plugin/tr069/handler/cwmp.go`
- Modify: `server/plugin/tr069/initialize/server.go`
- Modify: `server/plugin/tr069/middleware/raw_dump.go`
- Modify: `server/plugin/tr069/middleware/raw_response_dump.go`
- Modify: `server/plugin/tr069/infolog/infolog.go`

- [ ] Add sink tests with exact password-bearing XML proving payloads are stored byte-for-byte, never converted to JSON and never redacted.

```go
raw := []byte(`<cwmp:Download><Username>admin</Username><Password>secret</Password></cwmp:Download>`)
sink.Emit(ctx, observability.Event{
	Level: observability.LevelInfo, Stage: "wire.xml", Direction: observability.DirectionOutbound,
	Attributes: observability.Attributes{CWMPID: "cwmp-1", Method: tr069.MethodDownload}, Payload: raw,
})
```

- [ ] Implement `CoreEventSink.Enabled` as follows: INFO and ERROR are always enabled because XML persistence and failure tracking are product data; DEBUG follows `config.CurrentRuntime().Settings.Debug`; OFF/WARN follow normal level ordering.

- [ ] For a `wire.xml` event, resolve command in this order:

  1. `event.CommandID` when present;
  2. command row with `request_id = event.CWMPID`;
  3. for `TransferComplete`, extract `CommandKey` from the event payload and query `command_key`.

If no command matches, keep the existing raw/info log behavior but do not create an orphan `CommandXML` row.

- [ ] Save XML with `expires_at = event.Time + CurrentRuntime().RPCXMLRetention`. Store `RequestID`, direction, method, and CWMP ID as searchable metadata.

- [ ] For DEBUG stages (`builder.started`, `builder.completed`, `parser.started`, `parser.completed`), append compact `CommandEvent` metadata only when a command resolves; never duplicate the XML payload into event JSON.

- [ ] Construct both core parser and builder with `tr069.WithEventSink(sink)` in `engine.New`.

- [ ] Remove the extra pre-parse in `CWMPHandler`; the core parser is the single parse path. Preserve request ID and device metadata through core observability/context instead of parsing twice.

- [ ] Always install raw middleware, but have it and `infolog` consult `config.CurrentRuntime().Settings` per request/write. This makes existing debug/info switches, maximum bytes, redaction flags, and info-log directory hot-loadable without rebuilding the Gin engine. Ensure raw middleware does not persist a second command XML copy.

Run: `go test ./plugin/tr069/adapter ./plugin/tr069/handler ./plugin/tr069/middleware -run 'EventSink|XML|Raw|CWMP' -count=1`

- [ ] Commit the observability slice.

```bash
git add server/plugin/tr069/adapter/core_event_sink.go server/plugin/tr069/adapter/core_event_sink_test.go server/plugin/tr069/engine/engine.go server/plugin/tr069/handler/cwmp.go server/plugin/tr069/initialize/server.go server/plugin/tr069/middleware server/plugin/tr069/infolog/infolog.go
git commit -m "feat(tr069): persist exact command XML and core trace events"
```

### Task 7: Add recovery, timeout scanning, and XML retention workers

**Files:**

- Create: `server/plugin/tr069/service/command_workers.go`
- Create: `server/plugin/tr069/service/command_workers_test.go`
- Modify: `server/plugin/tr069/plugin.go`

- [ ] Use a fake clock in tests to cover all independent deadlines:

  - `WAITING_DEVICE` times out at 180 seconds;
  - `SENT` times out at 90 seconds;
  - `WAITING_TRANSFER` times out at 43200 seconds;
  - changing runtime config does not alter a stored deadline;
  - expired XML is deleted without deleting command/event rows.

- [ ] Implement `Recover` at startup:

  - transition stranded `BUILDING` rows back to `WAITING_DEVICE`, creating a new wait deadline from the current config and a `RECOVERED` event;
  - promote an earliest `QUEUED` head when the device has no earlier non-terminal command, creating its wait deadline;
  - enqueue wake-ups for every device whose head is `WAITING_DEVICE`;
  - leave `SENT` and `WAITING_TRANSFER` untouched so their persisted deadlines remain authoritative;
  - immediately finalize already-expired rows.

- [ ] Implement a ticker-driven scanner using conditional transitions:

```go
func (w *CommandWorkers) ScanTimeouts(ctx context.Context, now time.Time) error
func (w *CommandWorkers) Recover(ctx context.Context, now time.Time) error
func (w *CommandWorkers) CleanupXML(ctx context.Context, now time.Time) (int64, error)
```

- [ ] On terminal transition, append a device wake-up so the next FIFO command can be promoted. Promotion sets the next command to `WAITING_DEVICE` and stores a new deadline using the config snapshot at promotion time.

- [ ] Start one cancellable worker set from `plugin.Register` after migrations and before accepting command submissions. Use process context/shutdown hooks if available; do not spawn one worker per request.

Run: `go test ./plugin/tr069/service -run 'Recover|Timeout|Retention|Promotion' -count=1`

- [ ] Commit the reliability slice.

```bash
git add server/plugin/tr069/service/command_workers.go server/plugin/tr069/service/command_workers_test.go server/plugin/tr069/plugin.go
git commit -m "feat(tr069): recover and expire durable RPC commands"
```

### Task 8: Expose typed command and record APIs with GVA permissions

**Files:**

- Replace: `server/plugin/tr069/api/command.go`
- Create: `server/plugin/tr069/api/command_record.go`
- Create: `server/plugin/tr069/api/command_test.go`
- Modify: `server/plugin/tr069/router/device.go`
- Modify: `server/plugin/tr069/plugin.go`
- Modify: `server/plugin/tr069/initialize/api.go`
- Modify: `server/plugin/tr069/initialize/menu.go`

- [ ] Add API tests for all typed routes, invalid body rejection, offline rejection, unsupported method rejection, retry permission behavior, and immediate `{commandId,status}` responses.

- [ ] Implement the exact command routes under `/tr069/command/:deviceId/`:

```text
POST getRPCMethods
POST getParameterValues
POST getParameterNames
POST getParameterAttributes
POST setParameterValues
POST setParameterAttributes
POST addObject
POST deleteObject
POST download
POST upload
POST reboot
POST factoryReset
```

- [ ] Each handler binds its own request type and calls `CommandManager.Submit`; use the message “命令已提交”. Do not wait for the device response.

- [ ] Add record routes:

```text
GET  /tr069/command-record/list
GET  /tr069/command-record/:commandId
POST /tr069/command-record/:commandId/retry
```

List filters: device ID/serial, operation, status, command ID, creation time range. Detail returns the command, parsed structured params/result, ordered events, and ordered XML records.

- [ ] Register all 15 APIs in GVA `SysApi`, grouped by permission intent in descriptions: RPC查询、RPC配置、RPC传输、RPC维护、RPC记录列表、RPC记录详情.

- [ ] Protect the plugin group explicitly:

```go
private := group.Group("tr069")
private.Use(gvaMiddleware.JWTAuth(), gvaMiddleware.CasbinHandler())
```

Apply `OperationRecord()` to all POST/DELETE/PUT command and retry routes. Read-only list/detail routes use JWT/Casbin without operation recording.

- [ ] Add “RPC 记录” under the existing TR069 parent menu with component `plugin/tr069/view/command-record/index.vue`, path `commandRecord`, name `tr069CommandRecord`, and sort order after device list.

- [ ] Verify a role without the route’s Casbin rule gets HTTP 403 and no command row; verify validation failure creates only a GVA operation log and no command row.

Run: `go test ./plugin/tr069/api ./plugin/tr069/router ./plugin/tr069/initialize -count=1`

- [ ] Commit the API/permission slice.

```bash
git add server/plugin/tr069/api server/plugin/tr069/router/device.go server/plugin/tr069/plugin.go server/plugin/tr069/initialize/api.go server/plugin/tr069/initialize/menu.go
git commit -m "feat(tr069): expose permissioned typed RPC APIs"
```

### Task 9: Build concrete device-list actions and dedicated RPC forms

**Files:**

- Replace: `web/src/plugin/tr069/api/command.js`
- Create: `web/src/plugin/tr069/api/command-record.js`
- Replace: `web/src/plugin/tr069/utils/device-actions.js`
- Replace: `web/src/plugin/tr069/utils/device-actions.test.js`
- Create: `web/src/plugin/tr069/view/device/components/rpc-command-drawer.vue`
- Create: `web/src/plugin/tr069/view/device/components/rpc-forms/query-form.vue`
- Create: `web/src/plugin/tr069/view/device/components/rpc-forms/parameter-values-form.vue`
- Create: `web/src/plugin/tr069/view/device/components/rpc-forms/parameter-names-form.vue`
- Create: `web/src/plugin/tr069/view/device/components/rpc-forms/parameter-attributes-form.vue`
- Create: `web/src/plugin/tr069/view/device/components/rpc-forms/set-parameter-values-form.vue`
- Create: `web/src/plugin/tr069/view/device/components/rpc-forms/set-parameter-attributes-form.vue`
- Create: `web/src/plugin/tr069/view/device/components/rpc-forms/object-form.vue`
- Create: `web/src/plugin/tr069/view/device/components/rpc-forms/transfer-form.vue`
- Create: `web/src/plugin/tr069/view/device/components/rpc-forms/reboot-form.vue`
- Modify: `web/src/plugin/tr069/view/device/index.vue`

- [ ] Add Node tests for the complete concrete action registry and confirmation levels.

```js
export const RPC_ACTION_GROUPS = [
  { label: '查询', actions: [
    { key: 'getRPCMethods', label: '查询设备能力', method: 'GetRPCMethods', confirm: 'none' },
    { key: 'getParameterValues', label: '获取参数', method: 'GetParameterValues', confirm: 'none' },
    { key: 'getParameterNames', label: '获取参数名称', method: 'GetParameterNames', confirm: 'none' },
    { key: 'getParameterAttributes', label: '获取参数属性', method: 'GetParameterAttributes', confirm: 'none' }
  ]},
  { label: '配置', actions: [
    { key: 'setParameterValues', label: '配置参数', method: 'SetParameterValues', confirm: 'normal' },
    { key: 'setParameterAttributes', label: '配置参数属性', method: 'SetParameterAttributes', confirm: 'normal' },
    { key: 'addObject', label: '添加对象', method: 'AddObject', confirm: 'normal' },
    { key: 'deleteObject', label: '删除对象', method: 'DeleteObject', confirm: 'danger' }
  ]},
  { label: '文件', actions: [
    { key: 'download', label: '下载文件', method: 'Download', confirm: 'normal' },
    { key: 'upload', label: '上传文件', method: 'Upload', confirm: 'normal' }
  ]},
  { label: '设备维护', actions: [
    { key: 'reboot', label: '重启设备', method: 'Reboot', confirm: 'danger' },
    { key: 'factoryReset', label: '恢复出厂设置', method: 'FactoryReset', confirm: 'danger' }
  ]}
]
```

- [ ] Add one API function for every typed backend route. All functions return the asynchronous command result and share no generic arbitrary-operation function.

- [ ] Implement `rpc-command-drawer.vue` with GVA/Element Plus table-box styling. It selects a dedicated form component by action key, emits a typed payload, shows the configured confirmation, calls the matching API, displays “命令已提交”, and closes after successful submission.

- [ ] Dangerous confirmation uses `ElMessageBox.confirm` for Reboot, DeleteObject, and FactoryReset. Factory reset does not require typing the serial number.

- [ ] In the device list:

  - retain “同步参数” and “参数”;
  - replace the current two-item RPC dropdown with grouped concrete actions;
  - keep device deletion separate from CWMP DeleteObject;
  - disable every RPC when `row.online !== true`;
  - disable methods absent from `row.rpcMethods`, except “查询设备能力”.

- [ ] Keep the existing `data-model-viewer.vue` unchanged.

Run: `node --test web/src/plugin/tr069/utils/device-actions.test.js web/src/plugin/tr069/view/device/components/data-model-viewer.contract.test.js`

Run: `npm run build`

Workdir for the build: `web`

- [ ] Commit the device-action slice.

```bash
git add web/src/plugin/tr069/api web/src/plugin/tr069/utils web/src/plugin/tr069/view/device
git commit -m "feat(tr069): add concrete device RPC actions"
```

### Task 10: Build the GVA-style RPC record page

**Files:**

- Create: `web/src/plugin/tr069/view/command-record/index.vue`
- Create: `web/src/plugin/tr069/view/command-record/components/record-detail.vue`
- Create: `web/src/plugin/tr069/view/command-record/record-view.js`
- Create: `web/src/plugin/tr069/view/command-record/record-view.test.js`

- [ ] Add pure helper tests for status tags, terminal-state detection, retry visibility, duration formatting, and XML direction labels.

- [ ] Implement the page using the existing GVA layout conventions: `search-box`, `table-box`, inline `el-form`, `el-table`, and standard pagination.

Columns: command ID, device, concrete function name, status, current phase/deadline, creation time, sent time, completion time, duration, and actions.

- [ ] Add filters for command ID, device/serial, function, status, and time range. Refresh reads only from the record API; do not poll a single command synchronously from the submit drawer.

- [ ] Implement `record-detail.vue` as a Drawer containing:

  - overview and failure/fault fields;
  - formatted structured request and structured result;
  - chronological state/event timeline;
  - inbound/outbound XML tabs showing the complete exact payload in a scrollable monospace block;
  - retry action only for `FAILED` and `TIMEOUT`.

- [ ] Retry calls the dedicated endpoint, closes no history, and shows the new command ID. Link the new row to the original with `retryOf` in both list/detail.

- [ ] Apply status colors consistently: queued/waiting info, building/sent warning, completed success, failed danger, timeout danger/info distinction with explicit text.

Run: `node --test web/src/plugin/tr069/view/command-record/record-view.test.js`

Run: `npm run build`

Workdir for the build: `web`

- [ ] Commit the record UI slice.

```bash
git add web/src/plugin/tr069/view/command-record
git commit -m "feat(tr069): add RPC lifecycle record UI"
```

### Task 11: Verify state, permissions, restart recovery, and BS interoperability

**Files:**

- Create: `server/plugin/tr069/service/command_integration_test.go`
- Modify: `server/plugin/tr069/docs/application_layer_features.md`
- Modify: `server/plugin/tr069/docs/protocol_interface_requirements.md`

- [ ] Add an integration test that submits two same-device commands and one other-device command, drives fake responses, and asserts strict FIFO plus cross-device parallelism.

- [ ] Add integration coverage for Redis outage, process recovery, duplicate response, duplicate TransferComplete, all three timeouts, XML cleanup, retry lineage, and CWMP Fault persistence.

- [ ] Run all parent Go tests and race-sensitive plugin tests.

Run: `go test ./plugin/tr069/... -count=1`

Run: `go test -race ./plugin/tr069/service ./plugin/tr069/adapter -count=1`

Run: `go test ./... -count=1`

Workdir: `server`

- [ ] Run frontend contract tests and production build.

Run: `node --test src/plugin/tr069/utils/device-actions.test.js src/plugin/tr069/view/device/components/data-model-viewer.contract.test.js src/plugin/tr069/view/command-record/record-view.test.js`

Run: `npm run build`

Workdir: `web`

- [ ] Start GVA/TR-069 with the active local YAML and use the project `oamstart` skill to verify the `gva-acs-bs` OAM stack before live protocol tests.

- [ ] Automatically exercise safe RPCs against the real BS:

  - 查询设备能力;
  - 获取参数 with shallow `Device.`;
  - 获取参数名称;
  - 获取/配置 parameter attributes only with a reversible safe value;
  - 配置参数 only with a reversible safe value.

Verify each record has ordered events plus exact outbound and inbound XML.

- [ ] Do not automatically execute real-BS Reboot, DeleteObject, or FactoryReset. Cover them with a mock CPE; only run them against BS after an explicit human confirmation at execution time.

- [ ] Verify Download/Upload with a mock CPE through response `Status=1` and later TransferComplete, including the 12-hour stored deadline. Use a controlled test file endpoint before any real BS transfer.

- [ ] Update the plugin docs with the 12 operations, state machine, API list, permission groups, YAML keys, retention behavior, and safe BS verification procedure.

- [ ] Inspect final parent and nested repository state. The parent diff must not include the nested core directory.

```bash
git status --short
git diff --check
git -C server/plugin/tr069/lib/tr069-core-only status --short
git -C server/plugin/tr069/lib/tr069-core-only rev-parse HEAD
```

- [ ] Commit the integration tests/docs, then run the final verification commands again before claiming completion.

```bash
git add server/plugin/tr069/service/command_integration_test.go server/plugin/tr069/docs/application_layer_features.md server/plugin/tr069/docs/protocol_interface_requirements.md
git commit -m "test(tr069): verify durable RPC command management"
```
