# TR-069 Reboot Execution Confirmation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make GVA own the Reboot `CommandKey` and report a Reboot as complete only after the device acknowledges the RPC and later reconnects with `M Reboot` or `1 BOOT`.

**Architecture:** Extend the existing RPC metadata so Reboot, Download, and Upload share one server-generated `CommandKey` path. Keep the reboot lifecycle entirely inside the GVA TR-069 plugin: the core callback moves a Reboot to `WAITING_REBOOT`, an Inform confirmation service completes it, and a database scanner times it out. The existing atomic runtime configuration supplies the hot-reloadable timeout, while the Vue UI only renders the persisted state.

**Tech Stack:** Go 1.23, GORM, SQLite/MySQL, Gin, TR-069 core interfaces, Vue 3, Element Plus, Node test runner.

## Global Constraints

- Do not modify or commit `server/plugin/tr069/lib/tr069-core-only`.
- Reboot `CommandKey` is generated only by GVA as `rpc-<UUID>` and is never accepted from a user request.
- A retry creates a new command and a new `CommandKey`.
- `RebootResponse` means acknowledged, not completed.
- Only case-insensitive, whitespace-trimmed exact EventCode values `M Reboot` and `1 BOOT` confirm a reboot.
- `rebootConfirmTimeout` is independently configurable, hot reloadable, expressed in seconds, and defaults to `300`.
- Database command state is authoritative; confirmation and timeout updates must be conditional and idempotent.
- Inform confirmation failures must be logged without failing the CPE Inform response.
- Existing Download/Upload TransferComplete behavior and ordinary RPC completion behavior must not change.

## File Structure

- `server/plugin/tr069/service/rpc_registry.go`: declare which RPCs require a server-generated `CommandKey`; make Reboot an empty request.
- `server/plugin/tr069/service/command_manager.go`: generate a new `CommandKey` for every qualifying command and retry.
- `server/plugin/tr069/adapter/redis_command_source.go`: inject the persisted key into in-memory core parameters immediately before dispatch.
- `server/plugin/tr069/api/command.go`: submit Reboot without binding a client DTO.
- `server/plugin/tr069/model/request/command.go`: remove the obsolete user-controlled Reboot DTO.
- `server/plugin/tr069/model/command_status.go`: add `WAITING_REBOOT` and its legal transitions.
- `server/plugin/tr069/adapter/gorm_repo.go`: convert a successful Reboot response to `WAITING_REBOOT` and call the Inform confirmer after device persistence.
- `server/plugin/tr069/service/reboot_confirmation.go`: match reboot EventCodes and conditionally complete the oldest waiting Reboot for a device.
- `server/plugin/tr069/service/reboot_timeout_scanner.go`: conditionally time out expired reboot confirmations.
- `server/plugin/tr069/config/config.go`, `runtime.go`: expose the independent hot-reloadable timeout.
- `server/plugin/tr069/engine/engine.go`: construct the confirmer and run the scanner under the existing engine lifecycle context.
- `server/config.yaml`, `server/config.local.yaml`: declare the timeout explicitly.
- `web/src/plugin/tr069/view/device/components/rpc-command-dialog.vue`: remove the Reboot CommandKey field and payload.
- `web/src/plugin/tr069/api/command.js`: submit Reboot without a request body.
- `web/src/plugin/tr069/view/command-record/record-view.js`, `index.vue`: render the new lifecycle state and phase.

---

### Task 1: Make Reboot CommandKey Server-Owned

**Files:**
- Modify: `server/plugin/tr069/service/rpc_registry.go`
- Modify: `server/plugin/tr069/service/rpc_registry_test.go`
- Modify: `server/plugin/tr069/service/command_manager.go`
- Modify: `server/plugin/tr069/service/command_manager_test.go`
- Modify: `server/plugin/tr069/adapter/redis_command_source.go`
- Modify: `server/plugin/tr069/adapter/redis_command_source_test.go`
- Modify: `server/plugin/tr069/api/command.go`
- Create: `server/plugin/tr069/api/command_reboot_test.go`
- Modify: `server/plugin/tr069/model/request/command.go`

**Interfaces:**
- Produces: `RPCSpec.ServerCommandKey bool`.
- Produces: Reboot persisted parameters `{}` plus `Command.CommandKey = "rpc-<UUID>"`.
- Produces: dispatch-time `core.Command.Params["commandKey"]` copied from the database command.
- Consumes: existing `CommandManager.Submit`, `CommandManager.Retry`, and `RedisCommandSource.Pull`.

- [ ] **Step 1: Write failing registry and manager tests**

Add these assertions to `rpc_registry_test.go` and `command_manager_test.go`:

```go
func TestRebootUsesEmptyRequestAndServerCommandKeyMetadata(t *testing.T) {
	spec := RPCSpecs["Reboot"]
	if !spec.ServerCommandKey {
		t.Fatal("Reboot must use a server-generated CommandKey")
	}
	persisted, err := EncodeRPCRequest("Reboot", nil)
	if err != nil {
		t.Fatalf("EncodeRPCRequest(Reboot): %v", err)
	}
	if string(persisted) != `{}` {
		t.Fatalf("persisted Reboot request = %s, want {}", persisted)
	}
	params, err := DecodeRPCParams("Reboot", persisted)
	if err != nil {
		t.Fatalf("DecodeRPCParams(Reboot): %v", err)
	}
	if len(params) != 0 {
		t.Fatalf("Reboot user params = %#v, want empty", params)
	}
}

func TestCommandManagerRebootCreatesUniqueServerKeysAndRetryGetsNewKey(t *testing.T) {
	db := newCommandManagerTestDB(t)
	now := time.Date(2026, 7, 18, 10, 0, 0, 0, time.UTC)
	device := createCommandManagerDevice(t, db, "REBOOT-KEY", now)
	setCommandManagerCapabilities(t, db, device.ID, `["Reboot"]`)
	manager := NewCommandManager(db, func(context.Context, string) error { return nil },
		WithCommandManagerNow(func() time.Time { return now }))

	first, err := manager.Submit(context.Background(), device.ID, "Reboot", nil)
	if err != nil {
		t.Fatalf("first Reboot: %v", err)
	}
	second, err := manager.Submit(context.Background(), device.ID, "Reboot", nil)
	if err != nil {
		t.Fatalf("second Reboot: %v", err)
	}
	var commands []model.Command
	if err := db.Where("command_id IN ?", []string{first.CommandID, second.CommandID}).Find(&commands).Error; err != nil {
		t.Fatalf("load Reboot commands: %v", err)
	}
	if len(commands) != 2 || commands[0].CommandKey == nil || commands[1].CommandKey == nil ||
		*commands[0].CommandKey == *commands[1].CommandKey {
		t.Fatalf("Reboot keys are not unique: %#v", commands)
	}
	for _, command := range commands {
		if !strings.HasPrefix(*command.CommandKey, "rpc-") || string(command.ParamsJSON) != `{}` {
			t.Fatalf("Reboot persistence = key:%v params:%s", command.CommandKey, command.ParamsJSON)
		}
	}

	original := commands[0]
	finishedAt := now.Add(time.Second)
	if err := db.Model(&model.Command{}).Where("command_id = ?", original.CommandID).Updates(map[string]any{
		"status": model.CommandStatusTimeout, "finished_at": finishedAt,
	}).Error; err != nil {
		t.Fatalf("mark original retryable: %v", err)
	}
	retried, err := manager.Retry(context.Background(), original.CommandID)
	if err != nil {
		t.Fatalf("retry Reboot: %v", err)
	}
	var retry model.Command
	if err := db.First(&retry, "command_id = ?", retried.CommandID).Error; err != nil {
		t.Fatalf("load retry: %v", err)
	}
	if retry.CommandKey == nil || *retry.CommandKey == *original.CommandKey {
		t.Fatalf("retry key = %v, original = %v", retry.CommandKey, original.CommandKey)
	}
}
```

Create `command_reboot_test.go` with a Gin handler test that sends a legacy
body and proves it cannot control the stored key:

```go
func TestRebootIgnoresClientCommandKey(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		new(model.Device), new(model.DeviceRPCMethods),
		new(model.Command), new(model.CommandEvent),
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	now := time.Now()
	device := model.Device{OUI: "001122", SerialNumber: "API-REBOOT", LastInform: now}
	if err := db.Create(&device).Error; err != nil {
		t.Fatalf("create device: %v", err)
	}
	if err := db.Create(&model.DeviceRPCMethods{
		DeviceID: device.ID, MethodsJSON: datatypes.JSON(`["Reboot"]`),
	}).Error; err != nil {
		t.Fatalf("create capabilities: %v", err)
	}
	previousDB := global.GVA_DB
	global.GVA_DB = db
	t.Cleanup(func() { global.GVA_DB = previousDB })

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Params = gin.Params{{Key: "deviceId", Value: strconv.Itoa(int(device.ID))}}
	context.Request = httptest.NewRequest(
		http.MethodPost,
		"/tr069/command/"+strconv.Itoa(int(device.ID))+"/reboot",
		strings.NewReader(`{"commandKey":"client-controlled"}`),
	)
	context.Request.Header.Set("Content-Type", "application/json")

	new(CommandApi).Reboot(context)

	var command model.Command
	if err := db.First(&command, "operation = ?", "Reboot").Error; err != nil {
		t.Fatalf("load Reboot command: %v", err)
	}
	if command.CommandKey == nil || *command.CommandKey == "client-controlled" ||
		!strings.HasPrefix(*command.CommandKey, "rpc-") {
		t.Fatalf("stored CommandKey = %v", command.CommandKey)
	}
	if string(command.ParamsJSON) != `{}` {
		t.Fatalf("stored params = %s, want {}", command.ParamsJSON)
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run:

```bash
cd server
go test ./plugin/tr069/service -run 'Test(RebootUsesEmptyRequestAndServerCommandKeyMetadata|CommandManagerRebootCreatesUniqueServerKeysAndRetryGetsNewKey)' -count=1
```

Expected: FAIL because `ServerCommandKey` does not exist and Reboot still requires `request.RebootRequest`.

- [ ] **Step 3: Implement the server-owned metadata and empty Reboot request**

Change the registry and API to this shape:

```go
type RPCSpec struct {
	Method           string
	DisplayName      string
	Operation        string
	Capability       string
	Permission       string
	Confirmation     RPCConfirmationPolicy
	ResultPolicy     RPCResultPolicy
	DeferredPolicy   RPCDeferredPolicy
	Transfer         bool
	ServerCommandKey bool
	Validate         func(any) error
	newRequest       func() any
	normalize        func(any) (map[string]interface{}, error)
}

"Download": {
	Method: "Download", DisplayName: "下载文件", Operation: "Download",
	Capability: "Download", Permission: "transfer",
	Confirmation: RPCConfirmationNormal, ResultPolicy: RPCResultTransfer,
	DeferredPolicy: RPCDeferredTransferComplete,
	Transfer: true, ServerCommandKey: true,
	Validate: validateDownload,
	newRequest: newRPCRequest[req.DownloadRequest], normalize: normalizeDownload,
},
"Upload": {
	Method: "Upload", DisplayName: "上传文件", Operation: "Upload",
	Capability: "Upload", Permission: "transfer",
	Confirmation: RPCConfirmationNormal, ResultPolicy: RPCResultTransfer,
	DeferredPolicy: RPCDeferredTransferComplete,
	Transfer: true, ServerCommandKey: true,
	Validate: validateUpload,
	newRequest: newRPCRequest[req.UploadRequest], normalize: normalizeUpload,
},
"Reboot": {
	Method: "Reboot", DisplayName: "重启设备", Operation: "Reboot",
	Capability: "Reboot", Permission: "maintenance",
	Confirmation: RPCConfirmationDanger,
	ResultPolicy: RPCResultAcknowledgement,
	DeferredPolicy: RPCDeferredNone,
	ServerCommandKey: true,
	newRequest: newRPCRequest[emptyRPCRequest],
	normalize: normalizeEmptyRequest,
},
```

Delete `normalizeReboot`, delete `request.RebootRequest`, and change the API handler:

```go
func (a *CommandApi) Reboot(c *gin.Context) {
	submitCommand(c, "Reboot", nil)
}
```

Generate and inject keys from the same metadata:

```go
if spec, ok := RPCSpecs[operation]; ok && spec.ServerCommandKey {
	commandKey := "rpc-" + uuid.NewString()
	command.CommandKey = &commandKey
}
```

```go
if spec, ok := service.RPCSpecs[building.Operation]; ok &&
	spec.ServerCommandKey && building.CommandKey != nil {
	params["commandKey"] = *building.CommandKey
}
```

- [ ] **Step 4: Add and run the dispatch injection test**

Add a table test in `redis_command_source_test.go` covering Download and Reboot:

```go
func TestRedisCommandSourceInjectsServerCommandKeyAtDispatch(t *testing.T) {
	tests := []struct {
		operation string
		request   any
	}{
		{"Download", req.DownloadRequest{FileType: "1 Firmware Upgrade Image", URL: "https://example.test/fw.bin"}},
		{"Reboot", nil},
	}
	for _, tt := range tests {
		t.Run(tt.operation, func(t *testing.T) {
			db := newRedisCommandSourceTestDB(t)
			paramsJSON, err := service.EncodeRPCRequest(tt.operation, tt.request)
			if err != nil {
				t.Fatalf("encode %s: %v", tt.operation, err)
			}
			now := time.Now()
			key := "rpc-server-owned-" + strings.ToLower(tt.operation)
			command := model.Command{
				CommandID: "dispatch-" + tt.operation, DeviceID: 1,
				DeviceKey: "001122-" + strings.ToUpper(tt.operation),
				Operation: tt.operation, ParamsJSON: paramsJSON, CommandKey: &key,
				Status: model.CommandStatusWaitingDevice, QueuedAt: now,
				WaitingAt: &now, CreatedAt: now,
			}
			if err := service.NewCommandStore(db).Create(context.Background(), &command); err != nil {
				t.Fatalf("seed command: %v", err)
			}
			source := newRedisCommandSource(db, newCommandSourceLocker(),
				RedisCommandSourceConfig{InstanceID: "server-key-test"})
			pulled, ack, _, err := source.Pull(context.Background(), command.DeviceKey)
			if err != nil {
				t.Fatalf("Pull: %v", err)
			}
			if pulled == nil || pulled.Params["commandKey"] != key {
				t.Fatalf("dispatched params = %#v, want commandKey %q", pulled, key)
			}
			if err := ack(context.Background()); err != nil {
				t.Fatalf("ack: %v", err)
			}
		})
	}
}
```

Run:

```bash
cd server
go test ./plugin/tr069/service ./plugin/tr069/adapter ./plugin/tr069/api -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit the server-owned CommandKey slice**

```bash
git add server/plugin/tr069/service/rpc_registry.go \
  server/plugin/tr069/service/rpc_registry_test.go \
  server/plugin/tr069/service/command_manager.go \
  server/plugin/tr069/service/command_manager_test.go \
  server/plugin/tr069/adapter/redis_command_source.go \
  server/plugin/tr069/adapter/redis_command_source_test.go \
  server/plugin/tr069/api/command.go \
  server/plugin/tr069/api/command_reboot_test.go \
  server/plugin/tr069/model/request/command.go
git commit -m "fix(tr069): make reboot command key server managed"
```

### Task 2: Add the WAITING_REBOOT Lifecycle

**Files:**
- Modify: `server/plugin/tr069/model/command_status.go`
- Modify: `server/plugin/tr069/service/command_store_test.go`
- Modify: `server/plugin/tr069/adapter/gorm_repo.go`
- Modify: `server/plugin/tr069/adapter/gorm_repo_test.go`
- Modify: `server/plugin/tr069/config/config.go`
- Modify: `server/plugin/tr069/config/runtime.go`
- Modify: `server/plugin/tr069/config/runtime_test.go`
- Modify: `server/plugin/tr069/initialize/viper.go`
- Modify: `server/plugin/tr069/initialize/viper_test.go`
- Modify: `server/config.yaml`
- Modify: `server/config.local.yaml`

**Interfaces:**
- Produces: `model.CommandStatusWaitingReboot = "WAITING_REBOOT"`.
- Produces: Reboot `MarkSuccess` transition `SENT -> WAITING_REBOOT`.
- Produces: `TR069Config.RebootConfirmTimeout int` and `Runtime.RebootConfirmTimeout time.Duration`.

- [ ] **Step 1: Write failing configuration, status, and repository tests**

Add to `runtime_test.go`:

```go
func TestRebootConfirmTimeoutDefaultsAndPublishesDuration(t *testing.T) {
	normalized := NormalizeRuntimeConfig(TR069Config{})
	if normalized.RebootConfirmTimeout != 300 {
		t.Fatalf("default reboot confirmation timeout = %d", normalized.RebootConfirmTimeout)
	}
	previous := CurrentRuntime()
	t.Cleanup(func() { StoreRuntime(previous.Settings) })
	stored := StoreRuntime(TR069Config{RebootConfirmTimeout: 17})
	if stored.RebootConfirmTimeout != 17*time.Second {
		t.Fatalf("runtime reboot confirmation timeout = %s", stored.RebootConfirmTimeout)
	}
}
```

Extend `TestReloadConfigPublishesRuntimeWithoutMutatingLegacyConfig` to set
`tr069.rebootConfirmTimeout = 67` and assert `67*time.Second`.

Extend `TestCommandTransitionMapAndTerminalHelpers` so `WAITING_REBOOT` is accepted, nonterminal, legal only after `SENT`, and may transition to `COMPLETED`, `FAILED`, or `TIMEOUT`.

Add:

```go
func TestGormCommandRepoRebootResponseWaitsForBootInform(t *testing.T) {
	db := newGormCommandRepoTestDB(t)
	previous := config.CurrentRuntime()
	settings := previous.Settings
	settings.RebootConfirmTimeout = 41
	config.StoreRuntime(settings)
	t.Cleanup(func() { config.StoreRuntime(previous.Settings) })

	acknowledgedAt := time.Date(2026, 7, 18, 11, 0, 0, 0, time.UTC)
	command := model.Command{
		CommandID: "reboot-ack", DeviceID: 70, DeviceKey: "001122-REBOOT-ACK",
		Operation: "Reboot", ParamsJSON: model.LongTextJSON(`{}`),
		Status: model.CommandStatusSent, QueuedAt: acknowledgedAt.Add(-time.Minute),
		CreatedAt: acknowledgedAt.Add(-time.Minute),
	}
	if err := service.NewCommandStore(db).Create(context.Background(), &command); err != nil {
		t.Fatalf("seed Reboot: %v", err)
	}
	if err := newGormCommandRepo(db).MarkSuccess(context.Background(), command.CommandID, acknowledgedAt); err != nil {
		t.Fatalf("MarkSuccess(Reboot): %v", err)
	}
	var got model.Command
	if err := db.First(&got, "command_id = ?", command.CommandID).Error; err != nil {
		t.Fatalf("load Reboot: %v", err)
	}
	if got.Status != model.CommandStatusWaitingReboot || got.FinishedAt != nil {
		t.Fatalf("Reboot response finalized command: %#v", got)
	}
	wantDeadline := acknowledgedAt.Add(41 * time.Second)
	if got.PhaseDeadlineAt == nil || !got.PhaseDeadlineAt.Equal(wantDeadline) {
		t.Fatalf("deadline = %v, want %v", got.PhaseDeadlineAt, wantDeadline)
	}
	var event model.CommandEvent
	if err := db.Where("command_id = ? AND event_type = ?", command.CommandID, "REBOOT_ACKNOWLEDGED").First(&event).Error; err != nil {
		t.Fatalf("load acknowledgement event: %v", err)
	}
	if event.Stage != "reboot.acknowledged" ||
		event.FromStatus != model.CommandStatusSent ||
		event.ToStatus != model.CommandStatusWaitingReboot {
		t.Fatalf("acknowledgement event = %#v", event)
	}
}
```

- [ ] **Step 2: Run tests and observe failure**

Run:

```bash
cd server
go test ./plugin/tr069/config ./plugin/tr069/initialize ./plugin/tr069/service ./plugin/tr069/adapter \
  -run 'Test(RebootConfirmTimeoutDefaultsAndPublishesDuration|ReloadConfigPublishesRuntimeWithoutMutatingLegacyConfig|CommandTransitionMapAndTerminalHelpers|GormCommandRepoRebootResponseWaitsForBootInform)' \
  -count=1
```

Expected: FAIL because the runtime option, state, and Reboot-specific transition do not exist.

- [ ] **Step 3: Implement the runtime option, state, and Reboot response transition**

Add `RebootConfirmTimeout` beside the existing RPC/transfer timeout fields:

```go
const defaultRebootConfirmTimeout = 300

// TR069Config
RebootConfirmTimeout int `mapstructure:"rebootConfirmTimeout" json:"rebootConfirmTimeout" yaml:"rebootConfirmTimeout"`

// Runtime
RebootConfirmTimeout time.Duration
```

Normalize and publish it:

```go
if in.RebootConfirmTimeout <= 0 {
	in.RebootConfirmTimeout = defaultRebootConfirmTimeout
}
```

```go
RebootConfirmTimeout: time.Duration(normalized.RebootConfirmTimeout) * time.Second,
```

Add `zap.Int("rebootConfirmTimeout", next.RebootConfirmTimeout)` to
`logRuntimeConfig` and declare this in both YAML files:

```yaml
    rebootConfirmTimeout: 300
```

Add the state and transitions:

```go
const CommandStatusWaitingReboot = "WAITING_REBOOT"

CommandStatusSent: {
	CommandStatusWaitingTransfer: {},
	CommandStatusWaitingReboot:   {},
	CommandStatusCompleted:       {},
	CommandStatusFailed:          {},
	CommandStatusTimeout:         {},
},
CommandStatusWaitingReboot: {
	CommandStatusCompleted: {},
	CommandStatusFailed:    {},
	CommandStatusTimeout:   {},
},
```

Include `CommandStatusWaitingReboot` in `nonTerminalCommandStatuses` and in the source status list used by `markFailAtStage`.

At the start of the `MarkSuccess` transaction, branch on Reboot:

```go
if current.Operation == "Reboot" && current.Status == model.CommandStatusSent {
	deadline := finishedAt.Add(config.CurrentRuntime().RebootConfirmTimeout)
	_, err := service.NewCommandStore(tx).Transition(ctx, service.CommandTransition{
		CommandID: current.CommandID,
		FromStatuses: []string{model.CommandStatusSent},
		ToStatus: model.CommandStatusWaitingReboot,
		ExpectedVersion: current.Version,
		EventType: "REBOOT_ACKNOWLEDGED",
		Stage: "reboot.acknowledged",
		Updates: map[string]any{
			"phase_deadline_at": deadline,
			"finished_at": nil,
		},
	})
	return err
}
```

Leave the existing ordinary RPC and `WAITING_TRANSFER -> COMPLETED` branch unchanged.

- [ ] **Step 4: Run focused state tests**

Run:

```bash
cd server
go test ./plugin/tr069/config ./plugin/tr069/initialize ./plugin/tr069/service ./plugin/tr069/adapter -count=1
```

Expected: PASS, including existing ordinary success and profile terminal tests.

- [ ] **Step 5: Commit the lifecycle slice**

```bash
git add server/plugin/tr069/model/command_status.go \
  server/plugin/tr069/service/command_store_test.go \
  server/plugin/tr069/adapter/gorm_repo.go \
  server/plugin/tr069/adapter/gorm_repo_test.go \
  server/plugin/tr069/config/config.go \
  server/plugin/tr069/config/runtime.go \
  server/plugin/tr069/config/runtime_test.go \
  server/plugin/tr069/initialize/viper.go \
  server/plugin/tr069/initialize/viper_test.go \
  server/config.yaml server/config.local.yaml
git commit -m "feat(tr069): wait for reboot inform after rpc acknowledgement"
```

### Task 3: Complete Reboot from Inform Events

**Files:**
- Create: `server/plugin/tr069/service/reboot_confirmation.go`
- Create: `server/plugin/tr069/service/reboot_confirmation_test.go`
- Modify: `server/plugin/tr069/adapter/gorm_repo.go`
- Modify: `server/plugin/tr069/adapter/gorm_repo_test.go`
- Modify: `server/plugin/tr069/engine/engine.go`

**Interfaces:**
- Produces: `service.NewRebootConfirmationService(db *gorm.DB) *RebootConfirmationService`.
- Produces: `(*RebootConfirmationService).ConfirmFromInform(ctx context.Context, deviceID uint, events []string, confirmedAt time.Time) error`.
- Produces: `adapter.RebootInformConfirmer` with the same method signature.
- Consumes: `WAITING_REBOOT` from Task 2.

- [ ] **Step 1: Write failing service tests for exact matching and idempotency**

Create `reboot_confirmation_test.go` with:

```go
func TestRebootConfirmationMatchesOnlyBootEventsAndCompletesOldestWaitingCommand(t *testing.T) {
	for _, eventCode := range []string{"M Reboot", "1 BOOT", "  m reboot  "} {
		t.Run(eventCode, func(t *testing.T) {
			db := newRebootConfirmationTestDB(t)
			now := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
			command := seedWaitingReboot(t, db, "reboot-"+strings.ReplaceAll(eventCode, " ", "-"), 81, now.Add(-time.Minute))
			service := NewRebootConfirmationService(db)
			if err := service.ConfirmFromInform(context.Background(), command.DeviceID, []string{eventCode}, now); err != nil {
				t.Fatalf("ConfirmFromInform: %v", err)
			}
			var got model.Command
			if err := db.First(&got, "command_id = ?", command.CommandID).Error; err != nil {
				t.Fatalf("load command: %v", err)
			}
			if got.Status != model.CommandStatusCompleted || got.FinishedAt == nil || got.PhaseDeadlineAt != nil {
				t.Fatalf("confirmed command = %#v", got)
			}
			var count int64
			db.Model(&model.CommandEvent{}).
				Where("command_id = ? AND event_type = ?", command.CommandID, "REBOOT_CONFIRMED").
				Count(&count)
			if count != 1 {
				t.Fatalf("confirmation events = %d, want 1", count)
			}
			if err := service.ConfirmFromInform(context.Background(), command.DeviceID, []string{"1 BOOT"}, now.Add(time.Second)); err != nil {
				t.Fatalf("duplicate ConfirmFromInform: %v", err)
			}
			db.Model(&model.CommandEvent{}).
				Where("command_id = ? AND event_type = ?", command.CommandID, "REBOOT_CONFIRMED").
				Count(&count)
			if count != 1 {
				t.Fatalf("duplicate confirmation events = %d, want 1", count)
			}
		})
	}
}

func TestRebootConfirmationIgnoresUnrelatedEvents(t *testing.T) {
	for _, eventCode := range []string{"2 PERIODIC", "4 VALUE CHANGE", "0 BOOTSTRAP", "1 BOOTSTRAP", "M Rebooted"} {
		t.Run(eventCode, func(t *testing.T) {
			db := newRebootConfirmationTestDB(t)
			now := time.Now()
			command := seedWaitingReboot(t, db, "ignored-"+strings.ReplaceAll(eventCode, " ", "-"), 82, now)
			if err := NewRebootConfirmationService(db).ConfirmFromInform(
				context.Background(), command.DeviceID, []string{eventCode}, now.Add(time.Second),
			); err != nil {
				t.Fatalf("ConfirmFromInform: %v", err)
			}
			var got model.Command
			db.First(&got, "command_id = ?", command.CommandID)
			if got.Status != model.CommandStatusWaitingReboot {
				t.Fatalf("event %q changed status to %s", eventCode, got.Status)
			}
		})
	}
}
```

The test helpers must migrate `model.Command` and `model.CommandEvent`, create a `WAITING_REBOOT` command with a future deadline, and use one SQLite connection.

- [ ] **Step 2: Run tests and observe failure**

Run:

```bash
cd server
go test ./plugin/tr069/service -run '^TestRebootConfirmation' -count=1
```

Expected: FAIL because `RebootConfirmationService` does not exist.

- [ ] **Step 3: Implement the confirmation service**

Create:

```go
package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type RebootConfirmationService struct {
	db *gorm.DB
}

func NewRebootConfirmationService(db *gorm.DB) *RebootConfirmationService {
	return &RebootConfirmationService{db: db}
}

func isRebootConfirmationEvent(events []string) bool {
	for _, event := range events {
		switch strings.ToUpper(strings.TrimSpace(event)) {
		case "M REBOOT", "1 BOOT":
			return true
		}
	}
	return false
}

func (s *RebootConfirmationService) ConfirmFromInform(
	ctx context.Context,
	deviceID uint,
	events []string,
	confirmedAt time.Time,
) error {
	if s == nil || s.db == nil || deviceID == 0 || !isRebootConfirmationEvent(events) {
		return nil
	}
	if confirmedAt.IsZero() {
		confirmedAt = time.Now()
	}
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var command model.Command
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("device_id = ? AND operation = ? AND status = ?",
				deviceID, "Reboot", model.CommandStatusWaitingReboot).
			Order("created_at ASC").Order("command_id ASC").
			First(&command).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		_, err = NewCommandStore(tx).Transition(ctx, CommandTransition{
			CommandID: command.CommandID,
			FromStatuses: []string{model.CommandStatusWaitingReboot},
			ToStatus: model.CommandStatusCompleted,
			ExpectedVersion: command.Version,
			EventType: "REBOOT_CONFIRMED",
			Stage: "reboot.inform",
			Updates: map[string]any{
				"finished_at": confirmedAt,
				"phase_deadline_at": nil,
			},
		})
		return err
	})
	if errors.Is(err, ErrCommandTransitionConflict) {
		return nil
	}
	return err
}
```

- [ ] **Step 4: Wire confirmation after device persistence and make failures nonfatal**

Add to `adapter/gorm_repo.go`:

```go
type RebootInformConfirmer interface {
	ConfirmFromInform(context.Context, uint, []string, time.Time) error
}

func (r *GormDeviceRepo) SetRebootInformConfirmer(confirmer RebootInformConfirmer) {
	r.rebootConfirmer = confirmer
}
```

Refactor `UpsertFromInform` so it loads the persisted device ID once after the device upsert. After parameter/profile collection, call the confirmer:

```go
if info != nil && r.rebootConfirmer != nil {
	if err := r.rebootConfirmer.ConfirmFromInform(ctx, dbDevice.ID, info.Events, time.Now()); err != nil {
		if global.GVA_LOG != nil {
			global.GVA_LOG.Warn("failed to confirm Reboot from Inform",
				zap.Uint("deviceID", dbDevice.ID),
				zap.Strings("eventCodes", info.Events),
				zap.Error(err))
		}
	}
}
```

In `engine.newEngine`, create and inject the service only when the database is available:

```go
gormDeviceRepo := adapter.NewGormDeviceRepo(nil, profileRepository, provisioner)
if adapter.DBAvailable() {
	gormDeviceRepo.SetRebootInformConfirmer(service.NewRebootConfirmationService(global.GVA_DB))
}
devRepo = gormDeviceRepo
```

Add adapter tests with this fake:

```go
type recordingRebootInformConfirmer struct {
	deviceID uint
	events   []string
	err      error
}

func (r *recordingRebootInformConfirmer) ConfirmFromInform(
	_ context.Context,
	deviceID uint,
	events []string,
	_ time.Time,
) error {
	r.deviceID = deviceID
	r.events = append([]string(nil), events...)
	return r.err
}

func TestGormDeviceRepoForwardsPersistedDeviceAndEventsToRebootConfirmer(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		new(model.Device), new(model.DataModelValue), new(model.ConnectionProfile),
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	confirmer := new(recordingRebootInformConfirmer)
	repo := NewGormDeviceRepo(db, NewConnectionProfileRepository(db, nil))
	repo.SetRebootInformConfirmer(confirmer)
	info := &tr069core.InformSummary{
		Events: []string{"M Reboot"},
		Params: map[string]string{"Device.DeviceInfo.SerialNumber": "INFORM-REBOOT"},
	}
	if _, err := repo.UpsertFromInform(context.Background(), info, "192.0.2.40"); err != nil {
		t.Fatalf("UpsertFromInform: %v", err)
	}
	var device model.Device
	if err := db.First(&device, "serial_number = ?", "INFORM-REBOOT").Error; err != nil {
		t.Fatalf("load device: %v", err)
	}
	if confirmer.deviceID != device.ID ||
		!reflect.DeepEqual(confirmer.events, []string{"M Reboot"}) {
		t.Fatalf("confirmation input = device:%d events:%#v", confirmer.deviceID, confirmer.events)
	}
}

func TestGormDeviceRepoDoesNotFailInformWhenRebootConfirmationFails(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		new(model.Device), new(model.DataModelValue), new(model.ConnectionProfile),
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	confirmer := &recordingRebootInformConfirmer{err: errors.New("confirmation database unavailable")}
	repo := NewGormDeviceRepo(db, NewConnectionProfileRepository(db, nil))
	repo.SetRebootInformConfirmer(confirmer)
	info := &tr069core.InformSummary{
		Events: []string{"1 BOOT"},
		Params: map[string]string{"Device.DeviceInfo.SerialNumber": "INFORM-NONFATAL"},
	}
	if _, err := repo.UpsertFromInform(context.Background(), info, "192.0.2.41"); err != nil {
		t.Fatalf("confirmation failure escaped Inform handling: %v", err)
	}
}
```

- [ ] **Step 5: Run service, adapter, and engine tests**

Run:

```bash
cd server
go test ./plugin/tr069/service ./plugin/tr069/adapter ./plugin/tr069/engine -count=1
```

Expected: PASS.

- [ ] **Step 6: Commit Inform confirmation**

```bash
git add server/plugin/tr069/service/reboot_confirmation.go \
  server/plugin/tr069/service/reboot_confirmation_test.go \
  server/plugin/tr069/adapter/gorm_repo.go \
  server/plugin/tr069/adapter/gorm_repo_test.go \
  server/plugin/tr069/engine/engine.go
git commit -m "feat(tr069): confirm reboot completion from boot inform"
```

### Task 4: Add the Database Timeout Scanner

**Files:**
- Create: `server/plugin/tr069/service/reboot_timeout_scanner.go`
- Create: `server/plugin/tr069/service/reboot_timeout_scanner_test.go`
- Modify: `server/plugin/tr069/engine/engine.go`
- Modify: `server/plugin/tr069/engine/engine_test.go`

**Interfaces:**
- Produces: `NewRebootTimeoutScanner(db *gorm.DB) *RebootTimeoutScanner`, `Run(ctx)`, and testable `ScanOnce(ctx, now)`.
- Consumes: `WAITING_REBOOT` and `RebootConfirmTimeout` from Task 2.

- [ ] **Step 1: Write failing timeout scanner tests**

Create scanner tests:

```go
func TestRebootTimeoutScannerExpiresOnlyDueWaitingReboots(t *testing.T) {
	db := newRebootTimeoutScannerTestDB(t)
	now := time.Date(2026, 7, 18, 13, 0, 0, 0, time.UTC)
	expired := seedScannerCommand(t, db, "expired-reboot", model.CommandStatusWaitingReboot, now.Add(-time.Second))
	future := seedScannerCommand(t, db, "future-reboot", model.CommandStatusWaitingReboot, now.Add(time.Minute))
	sent := seedScannerCommand(t, db, "sent-command", model.CommandStatusSent, now.Add(-time.Second))

	scanner := NewRebootTimeoutScanner(db)
	if err := scanner.ScanOnce(context.Background(), now); err != nil {
		t.Fatalf("ScanOnce: %v", err)
	}
	assertScannerStatus(t, db, expired.CommandID, model.CommandStatusTimeout)
	assertScannerStatus(t, db, future.CommandID, model.CommandStatusWaitingReboot)
	assertScannerStatus(t, db, sent.CommandID, model.CommandStatusSent)

	var event model.CommandEvent
	if err := db.Where("command_id = ? AND event_type = ?", expired.CommandID, "REBOOT_CONFIRM_TIMEOUT").First(&event).Error; err != nil {
		t.Fatalf("load timeout event: %v", err)
	}
	if event.Stage != "reboot.confirm" {
		t.Fatalf("timeout stage = %q", event.Stage)
	}
}

func TestRebootTimeoutScannerIsIdempotent(t *testing.T) {
	db := newRebootTimeoutScannerTestDB(t)
	now := time.Now()
	command := seedScannerCommand(t, db, "idempotent-reboot", model.CommandStatusWaitingReboot, now.Add(-time.Second))
	scanner := NewRebootTimeoutScanner(db)
	if err := scanner.ScanOnce(context.Background(), now); err != nil {
		t.Fatalf("first scan: %v", err)
	}
	if err := scanner.ScanOnce(context.Background(), now.Add(time.Second)); err != nil {
		t.Fatalf("second scan: %v", err)
	}
	var count int64
	db.Model(&model.CommandEvent{}).
		Where("command_id = ? AND event_type = ?", command.CommandID, "REBOOT_CONFIRM_TIMEOUT").
		Count(&count)
	if count != 1 {
		t.Fatalf("timeout events = %d, want 1", count)
	}
}

func TestRebootConfirmationAndTimeoutRaceHasOneTerminalOutcome(t *testing.T) {
	db := newRebootTimeoutScannerTestDB(t)
	now := time.Now()
	command := seedScannerCommand(t, db, "reboot-race", model.CommandStatusWaitingReboot, now)
	confirmer := NewRebootConfirmationService(db)
	scanner := NewRebootTimeoutScanner(db)

	start := make(chan struct{})
	errorsByWorker := make(chan error, 2)
	go func() {
		<-start
		errorsByWorker <- confirmer.ConfirmFromInform(
			context.Background(), command.DeviceID, []string{"1 BOOT"}, now,
		)
	}()
	go func() {
		<-start
		errorsByWorker <- scanner.ScanOnce(context.Background(), now)
	}()
	close(start)
	for range 2 {
		if err := <-errorsByWorker; err != nil {
			t.Fatalf("race worker: %v", err)
		}
	}

	var got model.Command
	if err := db.First(&got, "command_id = ?", command.CommandID).Error; err != nil {
		t.Fatalf("load raced command: %v", err)
	}
	if got.Status != model.CommandStatusCompleted && got.Status != model.CommandStatusTimeout {
		t.Fatalf("race status = %s", got.Status)
	}
	var terminalEvents int64
	if err := db.Model(&model.CommandEvent{}).
		Where("command_id = ? AND event_type IN ?",
			command.CommandID, []string{"REBOOT_CONFIRMED", "REBOOT_CONFIRM_TIMEOUT"}).
		Count(&terminalEvents).Error; err != nil {
		t.Fatalf("count terminal events: %v", err)
	}
	if terminalEvents != 1 {
		t.Fatalf("terminal events = %d, want 1", terminalEvents)
	}
}
```

- [ ] **Step 2: Run scanner tests and observe failure**

Run:

```bash
cd server
go test ./plugin/tr069/service -run '^TestRebootTimeoutScanner' -count=1
```

Expected: FAIL because `RebootTimeoutScanner` does not exist.

- [ ] **Step 3: Implement the scanner**

Create a scanner with a five-second internal interval and a bounded batch:

```go
const (
	rebootTimeoutScanInterval = 5 * time.Second
	rebootTimeoutScanLimit    = 100
)

type RebootTimeoutScanner struct {
	db *gorm.DB
}

func NewRebootTimeoutScanner(db *gorm.DB) *RebootTimeoutScanner {
	return &RebootTimeoutScanner{db: db}
}

func (s *RebootTimeoutScanner) Run(ctx context.Context) {
	ticker := time.NewTicker(rebootTimeoutScanInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			if err := s.ScanOnce(ctx, now); err != nil && global.GVA_LOG != nil {
				global.GVA_LOG.Error("failed to scan expired Reboot confirmations", zap.Error(err))
			}
		}
	}
}

func (s *RebootTimeoutScanner) ScanOnce(ctx context.Context, now time.Time) error {
	var commands []model.Command
	if err := s.db.WithContext(ctx).
		Where("status = ? AND phase_deadline_at IS NOT NULL AND phase_deadline_at <= ?",
			model.CommandStatusWaitingReboot, now).
		Order("phase_deadline_at ASC").Order("command_id ASC").
		Limit(rebootTimeoutScanLimit).Find(&commands).Error; err != nil {
		return err
	}
	for _, command := range commands {
		_, err := NewCommandStore(s.db).Transition(ctx, CommandTransition{
			CommandID: command.CommandID,
			FromStatuses: []string{model.CommandStatusWaitingReboot},
			ToStatus: model.CommandStatusTimeout,
			ExpectedVersion: command.Version,
			EventType: "REBOOT_CONFIRM_TIMEOUT",
			Stage: "reboot.confirm",
			Message: "device did not report a reboot Inform before the deadline",
			Updates: map[string]any{
				"finished_at": now,
				"phase_deadline_at": nil,
				"failure_stage": "reboot.confirm",
				"fault_string": "等待设备重启确认超时",
			},
		})
		if err != nil && !errors.Is(err, ErrCommandTransitionConflict) {
			return err
		}
	}
	return nil
}
```

Log scan errors from `Run` through the GVA logger rather than silently discarding them in the final implementation.

- [ ] **Step 4: Run scanner under the existing engine lifecycle**

In `engine.newEngine`, collect background workers under `deps.RuntimeContext`:

```go
var workers []func(context.Context)
if provisioner != nil {
	workers = append(workers, provisioner.Run)
}
if adapter.DBAvailable() {
	scanner := service.NewRebootTimeoutScanner(global.GVA_DB)
	workers = append(workers, scanner.Run)
}
if len(workers) > 0 && deps.RuntimeContext != nil {
	runtimeDone = make(chan struct{})
	go func() {
		defer close(runtimeDone)
		var wait sync.WaitGroup
		for _, worker := range workers {
			wait.Add(1)
			go func(run func(context.Context)) {
				defer wait.Done()
				run(deps.RuntimeContext)
			}(worker)
		}
		wait.Wait()
	}()
}
```

Extract the worker runner so lifecycle behavior is independently testable:

```go
func runRuntimeWorkers(ctx context.Context, workers ...func(context.Context)) <-chan struct{} {
	if ctx == nil || len(workers) == 0 {
		return nil
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		var wait sync.WaitGroup
		for _, worker := range workers {
			wait.Add(1)
			go func(run func(context.Context)) {
				defer wait.Done()
				run(ctx)
			}(worker)
		}
		wait.Wait()
	}()
	return done
}
```

Use `runtimeDone = runRuntimeWorkers(deps.RuntimeContext, workers...)` in
`newEngine`, and add:

```go
func TestRunRuntimeWorkersStopsAllWorkersOnContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	worker := func(ctx context.Context) { <-ctx.Done() }
	done := runRuntimeWorkers(ctx, worker, worker)
	if done == nil {
		t.Fatal("runtime worker group did not start")
	}
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("runtime workers did not stop after cancellation")
	}
}
```

- [ ] **Step 5: Run scanner and engine tests**

Run:

```bash
cd server
go test ./plugin/tr069/service ./plugin/tr069/engine -count=1
```

Expected: PASS.

- [ ] **Step 6: Commit timeout behavior**

```bash
git add server/plugin/tr069/service/reboot_timeout_scanner.go \
  server/plugin/tr069/service/reboot_timeout_scanner_test.go \
  server/plugin/tr069/engine/engine.go \
  server/plugin/tr069/engine/engine_test.go
git commit -m "feat(tr069): time out unconfirmed reboot commands"
```

### Task 5: Update the GVA UI for Server-Owned Reboot and WAITING_REBOOT

**Files:**
- Modify: `web/src/plugin/tr069/view/device/components/rpc-command-dialog.vue`
- Modify: `web/src/plugin/tr069/api/command.js`
- Modify: `web/src/plugin/tr069/view/device/device-rpc-actions.contract.test.js`
- Modify: `web/src/plugin/tr069/view/command-record/record-view.js`
- Modify: `web/src/plugin/tr069/view/command-record/record-view.test.js`
- Modify: `web/src/plugin/tr069/view/command-record/index.vue`

**Interfaces:**
- Produces: `rebootDevice(deviceId)` with no request body.
- Produces: `statusView("WAITING_REBOOT") = { label: "设备已受理，等待重启", type: "warning" }`.
- Consumes: persisted `commandKey`, events, status, and deadline from existing command-record APIs.

- [ ] **Step 1: Write failing UI contract and status tests**

Add:

```js
test('reboot form does not expose or submit CommandKey', async () => {
  const dialog = await readFile(
    new URL('./components/rpc-command-dialog.vue', import.meta.url),
    'utf8'
  )
  const api = await readFile(
    new URL('../../api/command.js', import.meta.url),
    'utf8'
  )

  assert.doesNotMatch(dialog, /v-model="form\.commandKey"/)
  assert.doesNotMatch(dialog, /return\s*\{\s*commandKey:/)
  assert.match(dialog, /case 'reboot':\s*return undefined/)
  assert.match(api, /rebootDevice\s*=\s*\(deviceId\)\s*=>\s*postCommand\(deviceId,\s*'reboot'\)/)
})
```

Extend `record-view.test.js`:

```js
assert.deepEqual(
  statusView('WAITING_REBOOT'),
  { label: '设备已受理，等待重启', type: 'warning' }
)
assert.equal(canRetryRecord({ status: 'WAITING_REBOOT' }), false)
```

- [ ] **Step 2: Run tests and observe failure**

Run:

```bash
cd web
node --test \
  src/plugin/tr069/view/device/device-rpc-actions.contract.test.js \
  src/plugin/tr069/view/command-record/record-view.test.js
```

Expected: FAIL because the form and request still contain `commandKey`, and the status is unknown.

- [ ] **Step 3: Implement the Reboot form and status rendering**

Replace the Reboot input with a read-only explanation:

```vue
<template v-else-if="action.key === 'reboot'">
  <el-alert
    title="GVA 将自动生成本次重启的唯一标识；设备受理后，还需等待启动 Inform 才会显示完成。"
    type="warning"
    :closable="false"
    show-icon
  />
</template>
```

Remove `commandKey` from `defaultForm`, return no payload, and update the API:

```js
case 'reboot':
  return undefined
```

```js
export const rebootDevice = (deviceId) => postCommand(deviceId, 'reboot')
```

Add the status:

```js
WAITING_REBOOT: { label: '设备已受理，等待重启', type: 'warning' },
```

Add the list phase:

```js
WAITING_REBOOT: '等待设备重新上线',
```

Keep `CommandKey` visible only as the existing read-only value in command details.

- [ ] **Step 4: Run UI tests and production build**

Run:

```bash
cd web
node --test \
  src/plugin/tr069/view/device/device-rpc-actions.contract.test.js \
  src/plugin/tr069/view/command-record/record-view.test.js
npm run build
```

Expected: tests PASS and Vite production build exits with code 0.

- [ ] **Step 5: Commit the UI slice**

```bash
git add web/src/plugin/tr069/view/device/components/rpc-command-dialog.vue \
  web/src/plugin/tr069/api/command.js \
  web/src/plugin/tr069/view/device/device-rpc-actions.contract.test.js \
  web/src/plugin/tr069/view/command-record/record-view.js \
  web/src/plugin/tr069/view/command-record/record-view.test.js \
  web/src/plugin/tr069/view/command-record/index.vue
git commit -m "feat(tr069): show reboot confirmation lifecycle"
```

### Task 6: Verify the End-to-End GVA Reboot Contract

**Files:**
- Modify: `server/plugin/tr069/adapter/redis_command_source_test.go`
- Modify: `server/plugin/tr069/adapter/gorm_repo_test.go`

**Interfaces:**
- Consumes: every interface produced in Tasks 1–5.
- Produces: test evidence that the database, dispatched XML inputs, Inform confirmation, timeout, API output, and UI agree.

- [ ] **Step 1: Add database-to-XML and ACK-to-Inform integration tests**

In `redis_command_source_test.go`, extend the Reboot dispatch test through the
core executor and builder:

```go
request, err := new(defaults.RebootExecutor).BuildRequest(
	context.Background(), new(core.Session), pulled,
)
if err != nil {
	t.Fatalf("build Reboot request: %v", err)
}
message, ok := request.(*tr069.Message)
if !ok {
	t.Fatalf("Reboot request type = %T", request)
}
xml, err := factory.NewBuilder().BuildMessage(context.Background(), message)
if err != nil {
	t.Fatalf("serialize Reboot XML: %v", err)
}
if !bytes.Contains(xml, []byte("<CommandKey>"+commandKey+"</CommandKey>")) {
	t.Fatalf("Reboot XML does not contain persisted CommandKey %q:\n%s", commandKey, xml)
}
if bytes.Contains(xml, []byte("client-controlled")) {
	t.Fatalf("Reboot XML contains client-controlled CommandKey:\n%s", xml)
}
```

In `gorm_repo_test.go`, add:

```go
func TestRebootLifecycleAcknowledgementThenBootInformCompletesCommand(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		new(model.Device), new(model.DataModelValue), new(model.ConnectionProfile),
		new(model.Command), new(model.CommandEvent),
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	now := time.Date(2026, 7, 18, 14, 0, 0, 0, time.UTC)
	device := model.Device{OUI: "001122", SerialNumber: "REBOOT-LIFECYCLE", LastInform: now.Add(-time.Minute)}
	if err := db.Create(&device).Error; err != nil {
		t.Fatalf("create device: %v", err)
	}
	key := "rpc-reboot-lifecycle"
	command := model.Command{
		CommandID: "reboot-lifecycle", DeviceID: device.ID,
		DeviceKey: "001122-REBOOT-LIFECYCLE", Operation: "Reboot",
		ParamsJSON: model.LongTextJSON(`{}`), CommandKey: &key,
		Status: model.CommandStatusSent, QueuedAt: now.Add(-time.Minute),
		CreatedAt: now.Add(-time.Minute),
	}
	if err := service.NewCommandStore(db).Create(context.Background(), &command); err != nil {
		t.Fatalf("seed Reboot: %v", err)
	}
	if err := newGormCommandRepo(db).MarkSuccess(context.Background(), command.CommandID, now); err != nil {
		t.Fatalf("acknowledge Reboot: %v", err)
	}
	var acknowledged model.Command
	if err := db.First(&acknowledged, "command_id = ?", command.CommandID).Error; err != nil {
		t.Fatalf("load acknowledged Reboot: %v", err)
	}
	if acknowledged.Status != model.CommandStatusWaitingReboot {
		t.Fatalf("status after RebootResponse = %s", acknowledged.Status)
	}

	deviceRepo := NewGormDeviceRepo(db, NewConnectionProfileRepository(db, nil))
	deviceRepo.SetRebootInformConfirmer(service.NewRebootConfirmationService(db))
	info := &tr069core.InformSummary{
		Events: []string{"1 BOOT"},
		Params: map[string]string{"Device.DeviceInfo.SerialNumber": device.SerialNumber},
	}
	if _, err := deviceRepo.UpsertFromInform(context.Background(), info, "192.0.2.42"); err != nil {
		t.Fatalf("process boot Inform: %v", err)
	}
	var completed model.Command
	if err := db.First(&completed, "command_id = ?", command.CommandID).Error; err != nil {
		t.Fatalf("load completed Reboot: %v", err)
	}
	if completed.Status != model.CommandStatusCompleted ||
		completed.FinishedAt == nil || completed.PhaseDeadlineAt != nil {
		t.Fatalf("status after boot Inform = %#v", completed)
	}
}
```

- [ ] **Step 2: Run the complete TR-069 backend suite**

Run:

```bash
cd server
go test ./plugin/tr069/... -count=1
```

Expected: PASS.

- [ ] **Step 3: Run the complete TR-069 frontend test set and build**

Run:

```bash
cd web
node --test \
  src/plugin/tr069/utils/device-actions.test.js \
  src/plugin/tr069/view/device/*.test.js \
  src/plugin/tr069/view/device/components/*.test.js \
  src/plugin/tr069/view/command-record/*.test.js
npm run build
```

Expected: all tests PASS and production build exits with code 0.

- [ ] **Step 4: Check formatting, generated artifacts, and repository scope**

Run:

```bash
gofmt -w \
  server/plugin/tr069/service/rpc_registry.go \
  server/plugin/tr069/service/rpc_registry_test.go \
  server/plugin/tr069/service/command_manager.go \
  server/plugin/tr069/service/command_manager_test.go \
  server/plugin/tr069/service/command_store_test.go \
  server/plugin/tr069/service/reboot_confirmation.go \
  server/plugin/tr069/service/reboot_confirmation_test.go \
  server/plugin/tr069/service/reboot_timeout_scanner.go \
  server/plugin/tr069/service/reboot_timeout_scanner_test.go \
  server/plugin/tr069/adapter/redis_command_source.go \
  server/plugin/tr069/adapter/redis_command_source_test.go \
  server/plugin/tr069/adapter/gorm_repo.go \
  server/plugin/tr069/adapter/gorm_repo_test.go \
  server/plugin/tr069/config/config.go \
  server/plugin/tr069/config/runtime.go \
  server/plugin/tr069/config/runtime_test.go \
  server/plugin/tr069/initialize/viper.go \
  server/plugin/tr069/initialize/viper_test.go \
  server/plugin/tr069/engine/engine.go \
  server/plugin/tr069/engine/engine_test.go \
  server/plugin/tr069/api/command.go \
  server/plugin/tr069/api/command_reboot_test.go \
  server/plugin/tr069/model/request/command.go \
  server/plugin/tr069/model/command_status.go
git diff --check
git status --short
git diff -- server/plugin/tr069/lib/tr069-core-only
```

Expected:

- `git diff --check` exits with code 0.
- No generated `web/dist`, logs, runtime state, or local secrets are staged.
- The final core diff is empty.

- [ ] **Step 5: Commit any integration-only changes**

```bash
git add server/plugin/tr069/adapter/redis_command_source_test.go \
  server/plugin/tr069/adapter/gorm_repo_test.go
git diff --cached --quiet || git commit -m "test(tr069): verify reboot execution confirmation"
```

- [ ] **Step 6: Record the manual BS acceptance procedure**

After automated tests pass:

1. Start GVA and the BS/OAM stack.
2. Submit “文件与维护 -> 重启设备” without any CommandKey field.
3. Confirm the command detail shows a generated `rpc-<UUID>`.
4. Confirm the outbound Reboot XML uses the same value.
5. Confirm `RebootResponse` changes the state to “设备已受理，等待重启”.
6. On a BS mode that actually reboots, confirm the following `M Reboot` or `1 BOOT` Inform completes the command.
7. On the current non-restarting BS mode, wait for `rebootConfirmTimeout` and confirm the command becomes `TIMEOUT`, never `COMPLETED`.
