# TR-069 Device Cascade Deletion Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use subagent-driven-development (recommended) or executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Deleting a TR-069 device permanently removes its MinIO log objects and every device-owned TR-069 database/Redis record while blocking concurrent work and allowing a later Inform to register the device again.

**Architecture:** Keep `DELETE /tr069/device/:deviceId`, but replace handler-owned SQL with a retry-safe `DeviceDeletionService`. A durable `deleting_at` marker blocks new work, an in-process upload registry aborts active streams, an adapter clears device-scoped Redis keys, and the service deletes MinIO objects before one hard-delete database transaction.

**Tech Stack:** Go 1.x, Gin, GORM, SQLite tests, go-redis/miniredis, MinIO through the existing `ArtifactStore`, Vue 3, Element Plus, Node test runner.

## Global Constraints

- MinIO log objects are permanently deleted immediately; there is no trash or audit copy.
- Deletion forcibly stops GVA-side queued/running RPC work and active uploads, but cannot undo an RPC already executed by the CPE.
- A device in deletion rejects new RPC, ACS, upload, and artifact-download work.
- A fully deleted device may automatically register again through a later Inform.
- GVA operation audit logs remain; all device-owned TR-069 data is hard-deleted.
- Database records are retained when MinIO deletion fails so the operation can be retried safely.
- Object-not-found is an idempotent success.
- Do not add database foreign keys or an asynchronous deletion queue.

---

## File Map

**Create**

- `server/plugin/tr069/service/device_deletion.go` — deletion errors, dependency interfaces, MinIO-first orchestration, and database cascade.
- `server/plugin/tr069/service/device_deletion_test.go` — full two-device cascade, storage failure, retry, rollback, and isolation tests.
- `server/plugin/tr069/service/device_state.go` — shared deleting-device error and runtime cleanup contracts.
- `server/plugin/tr069/service/upload_runtime_registry.go` — per-device upload block/cancel/abort registry.
- `server/plugin/tr069/service/upload_runtime_registry_test.go` — register/block/race/cleanup tests.
- `server/plugin/tr069/adapter/redis_device_runtime_cleaner.go` — purge wake, queue, session, inflight, lock, and upload-identity Redis state.
- `server/plugin/tr069/adapter/redis_device_runtime_cleaner_test.go` — miniredis-backed targeted cleanup tests.

**Modify**

- `server/plugin/tr069/model/device.go` — add nullable indexed `DeletingAt`.
- `server/plugin/tr069/model/response/device.go` — expose a boolean `deleting`, not the timestamp.
- `server/plugin/tr069/service/transfer_receiver.go` — register uploads and attach writer abort callbacks.
- `server/plugin/tr069/service/transfer_receiver_test.go` — prove deletion cancels a streaming upload.
- `server/plugin/tr069/service/command_manager.go` and `command_manager_test.go` — reject command submission for deleting devices.
- `server/plugin/tr069/service/upload_device_resolver.go` and `transfer_receiver_test.go` — exclude deleting devices from all resolution paths.
- `server/plugin/tr069/service/transfer_store.go` and `server/plugin/tr069/api/artifact_test.go` — hide artifacts owned by deleting devices from list/download.
- `server/plugin/tr069/adapter/redis_upload_identity.go` and its test — remove one device binding without affecting peers on the same IP.
- `server/plugin/tr069/adapter/gorm_repo.go` and `gorm_repo_test.go` — reject an Inform while its existing serial number is deleting.
- `server/plugin/tr069/handler/cwmp.go` and `cwmp_test.go` — map the deleting-device rejection without leaking internals.
- `server/plugin/tr069/initialize/server.go` and `server_runtime_test.go` — share the upload registry with ingress and management deletion.
- `server/plugin/tr069/api/device.go` and `device_test.go` — delegate delete behavior and return stage-safe failures.
- `server/plugin/tr069/router/device.go` and `router/device_command_test.go` — inject the deletion service while preserving the route.
- `server/plugin/tr069/plugin.go` — assemble the deletion dependencies after file ingress initialization.
- `web/src/plugin/tr069/view/device/index.vue` — destructive confirmation, deleting state, retry-only actions, and loading state.
- `web/src/plugin/tr069/view/device/device-rpc-actions.contract.test.js` — frontend deletion contract.

---

### Task 1: Durable deletion state and entry guards

**Files:**

- Modify: `server/plugin/tr069/model/device.go`
- Modify: `server/plugin/tr069/model/response/device.go`
- Create: `server/plugin/tr069/service/device_state.go`
- Modify: `server/plugin/tr069/service/command_manager.go`
- Modify: `server/plugin/tr069/service/command_manager_test.go`
- Modify: `server/plugin/tr069/service/upload_device_resolver.go`
- Modify: `server/plugin/tr069/service/transfer_receiver_test.go`
- Modify: `server/plugin/tr069/service/transfer_store.go`
- Modify: `server/plugin/tr069/api/artifact_test.go`

**Interfaces:**

- Produces: `service.ErrDeviceDeleting` and `model.Device.DeletingAt *time.Time`.
- Consumes later: deletion service marks `DeletingAt`; command/upload/download paths reject it.

- [ ] **Step 1: Write failing command and resolver tests**

Add tests which create a device with `DeletingAt = &now` and assert:

```go
_, err := manager.Submit(context.Background(), device.ID, "Reboot", nil)
if !errors.Is(err, ErrDeviceDeleting) {
	 t.Fatalf("Submit() error = %v, want ErrDeviceDeleting", err)
}

_, err = resolver.Resolve(context.Background(), device.IP, "LOG")
if !errors.Is(err, ErrUploadDeviceNotFound) {
	 t.Fatalf("Resolve() error = %v, want ErrUploadDeviceNotFound", err)
}
```

Extend the artifact API fixture with a deleting device and assert its artifact is absent from list and returns HTTP 404 from download.

- [ ] **Step 2: Run focused tests and verify RED**

Run:

```bash
cd server
go test ./plugin/tr069/service ./plugin/tr069/api -run 'Deleting|ArtifactAPI' -count=1
```

Expected: FAIL because `DeletingAt`/`ErrDeviceDeleting` and deletion-aware artifact queries do not exist.

- [ ] **Step 3: Add the minimal durable marker and guards**

Add to `model.Device`:

```go
DeletingAt *time.Time `json:"-" gorm:"index;comment:设备级联删除开始时间"`
```

Add `service/device_state.go`:

```go
var ErrDeviceDeleting = errors.New("device is being deleted")

type DeviceRuntimeIdentity struct {
	DeviceID  uint
	DeviceKey string
	IP        string
}

type DeviceRuntimeCleaner interface {
	Purge(context.Context, DeviceRuntimeIdentity) error
}
```

In `CommandManager.submitPersisted`, select `deleting_at` under the existing row lock and return `ErrDeviceDeleting` when non-nil. In all three `UploadDeviceResolver` paths add `devices.deleting_at IS NULL` or `deleting_at IS NULL` to the device lookup.

Change artifact list and download selection to join `tr069_devices` and require `devices.deleting_at IS NULL`. Add `Deleting bool` to `DeviceResponse`; map it as `d.DeletingAt != nil` and force `Online=false` while deleting.

- [ ] **Step 4: Run tests and verify GREEN**

```bash
cd server
go test ./plugin/tr069/service ./plugin/tr069/api -run 'Deleting|ArtifactAPI' -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add server/plugin/tr069/model server/plugin/tr069/service/device_state.go server/plugin/tr069/service/command_manager.go server/plugin/tr069/service/command_manager_test.go server/plugin/tr069/service/upload_device_resolver.go server/plugin/tr069/service/transfer_receiver_test.go server/plugin/tr069/service/transfer_store.go server/plugin/tr069/api/artifact_test.go server/plugin/tr069/api/device.go
git commit -m "feat(tr069): block work for deleting devices"
```

---

### Task 2: Active upload cancellation registry

**Files:**

- Create: `server/plugin/tr069/service/upload_runtime_registry.go`
- Create: `server/plugin/tr069/service/upload_runtime_registry_test.go`
- Modify: `server/plugin/tr069/service/transfer_receiver.go`
- Modify: `server/plugin/tr069/service/transfer_receiver_test.go`
- Modify: `server/plugin/tr069/initialize/server.go`
- Modify: `server/plugin/tr069/initialize/server_runtime_test.go`

**Interfaces:**

- Produces: `NewUploadRuntimeRegistry`, `Register`, `AttachWriter`, `BlockAndCancel`, and `Release`.
- Consumes: `ArtifactWriter.Abort(context.Context)`.
- Consumed later by: `DeviceDeletionService` and runtime assembly.

- [ ] **Step 1: Write failing registry and receiver tests**

Cover these behaviors using a blocking fake writer:

```go
registry := NewUploadRuntimeRegistry()
ctx, cancel := context.WithCancel(context.Background())
handle, err := registry.Register(41, cancel)
if err != nil { t.Fatal(err) }
handle.AttachWriter(writer)
registry.BlockAndCancel(41)

select {
case <-ctx.Done():
case <-time.After(time.Second):
	t.Fatal("upload context was not cancelled")
}
if !writer.aborted.Load() { t.Fatal("writer was not aborted") }
if _, err := registry.Register(41, func() {}); !errors.Is(err, ErrDeviceDeleting) {
	t.Fatalf("register error = %v", err)
}
```

Add a receiver test that starts `Receive` with a body blocked mid-stream, invokes `BlockAndCancel(device.ID)`, and asserts `Receive` exits and the store contains no committed object.

- [ ] **Step 2: Run tests and verify RED**

```bash
cd server
go test ./plugin/tr069/service -run 'UploadRuntime|CancelsActiveUpload' -count=1
```

Expected: FAIL because the registry is not implemented.

- [ ] **Step 3: Implement the registry and receiver integration**

Implement a mutex-protected registry with a blocked-device set and per-device upload handles. `BlockAndCancel` must atomically mark the device blocked, copy active handles, then cancel and abort outside the mutex. `Release` removes the block only after database deletion succeeds.

Extend the receiver constructor without breaking existing callers:

```go
func NewTransferReceiver(transfers *TransferStore, objects ArtifactStore, registries ...*UploadRuntimeRegistry) *TransferReceiver
```

Inside `Receive`, create the timeout context, register its cancel function before metadata creation, attach the writer immediately after `Begin`, and always unregister with `defer`. Treat `ErrDeviceDeleting` as a conflict-level rejection.

In `initialize.SetupEngine`, create one registry, pass it to the receiver, and store it beside the current artifact store:

```go
func CurrentUploadRuntimeRegistry() *service.UploadRuntimeRegistry
```

- [ ] **Step 4: Run tests and verify GREEN**

```bash
cd server
go test ./plugin/tr069/service ./plugin/tr069/initialize -run 'UploadRuntime|CancelsActiveUpload|ServerRuntime' -count=1
```

Expected: PASS with no leaked goroutines.

- [ ] **Step 5: Commit**

```bash
git add server/plugin/tr069/service/upload_runtime_registry* server/plugin/tr069/service/transfer_receiver* server/plugin/tr069/initialize/server.go server/plugin/tr069/initialize/server_runtime_test.go
git commit -m "feat(tr069): cancel uploads during device deletion"
```

---

### Task 3: Device-scoped Redis cleanup

**Files:**

- Create: `server/plugin/tr069/adapter/redis_device_runtime_cleaner.go`
- Create: `server/plugin/tr069/adapter/redis_device_runtime_cleaner_test.go`
- Modify: `server/plugin/tr069/adapter/redis_upload_identity.go`
- Modify: `server/plugin/tr069/adapter/redis_upload_identity_test.go`

**Interfaces:**

- Produces: `service.DeviceRuntimeCleaner.Purge(ctx, identity)` implementation.
- Consumes: device database ID, `OUI-SerialNumber` device key, and management IP.

- [ ] **Step 1: Write failing miniredis tests**

Seed Redis with target and peer values for:

```text
tr069:command:wakeup
tr069:cmd:pending:<deviceKey>
tr069:cmd:lock:<deviceKey>
tr069:cmd:inflight:<deviceKey>
tr069:sess:<deviceKey>
tr069:sess:lock:<deviceKey>
tr069:sess:tmp:<ip>
tr069:file-ingress:ip:<ip>
tr069:file-ingress:identity:<ip>:<deviceId>
```

Call `Purge` and assert only target list values/keys/binding members disappear. A peer sharing the same IP must remain in the upload identity sorted set.

- [ ] **Step 2: Run tests and verify RED**

```bash
cd server
go test ./plugin/tr069/adapter -run 'RedisDeviceRuntimeCleaner|UploadIdentityUnbind' -count=1
```

Expected: FAIL because `Unbind` and the cleaner do not exist.

- [ ] **Step 3: Implement targeted cleanup**

`UploadIdentityStore.Unbind` must normalize the IP, `ZREM` only the device ID member, and delete only its member payload key. The cleaner uses one Redis pipeline to `LREM` all target wake tokens and delete target pending/lock/inflight/session keys. It then calls `Unbind`; missing keys are successful.

- [ ] **Step 4: Run tests and verify GREEN**

```bash
cd server
go test ./plugin/tr069/adapter -run 'RedisDeviceRuntimeCleaner|UploadIdentityUnbind' -count=1
```

Expected: PASS and peer keys remain.

- [ ] **Step 5: Commit**

```bash
git add server/plugin/tr069/adapter/redis_device_runtime_cleaner* server/plugin/tr069/adapter/redis_upload_identity*
git commit -m "feat(tr069): purge deleted device Redis state"
```

---

### Task 4: MinIO-first cascade deletion service

**Files:**

- Create: `server/plugin/tr069/service/device_deletion.go`
- Create: `server/plugin/tr069/service/device_deletion_test.go`

**Interfaces:**

- Consumes: `*gorm.DB`, `ArtifactStore`, `*UploadRuntimeRegistry`, and `DeviceRuntimeCleaner`.
- Produces:

```go
func NewDeviceDeletionService(db *gorm.DB, objects ArtifactStore, uploads *UploadRuntimeRegistry, runtime DeviceRuntimeCleaner) *DeviceDeletionService
func (s *DeviceDeletionService) Delete(context.Context, uint) (DeviceDeletionResult, error)
```

- [ ] **Step 1: Write the full failing cascade test**

Create an in-memory SQLite schema for all device-owned models and seed two complete devices. For the target include alarms, values, RPC methods, FAP service, profile, commands/events/XML, transfer task/events, artifact metadata, and fake-store objects.

After deleting the target, assert zero target rows in every table, all target objects removed, and every peer row/object preserved. Assert the runtime cleaner received exactly the target identity.

Also add focused tests:

```go
func TestDeviceDeletionStorageFailureKeepsDatabaseAndDeletingMarker(t *testing.T)
func TestDeviceDeletionRetryTreatsMissingObjectsAsSuccess(t *testing.T)
func TestDeviceDeletionDatabaseFailureKeepsAllDatabaseRows(t *testing.T)
func TestDeviceDeletionMissingDeviceReturnsNotFound(t *testing.T)
```

- [ ] **Step 2: Run tests and verify RED**

```bash
cd server
go test ./plugin/tr069/service -run '^TestDeviceDeletion' -count=1
```

Expected: FAIL because `DeviceDeletionService` does not exist.

- [ ] **Step 3: Implement mark, quiesce, object deletion, and transaction**

Implement these stages in order:

```go
const (
	DeletionStageMark      = "mark_deleting"
	DeletionStageRuntime   = "stop_runtime"
	DeletionStageArtifacts = "delete_artifacts"
	DeletionStageDatabase  = "delete_database"
)

type DeviceDeletionResult struct {
	DeviceID       uint
	DeletedObjects int
	DeletedRows    map[string]int64
}

type DeviceDeletionError struct {
	Stage string
	Cause error
}

func (e *DeviceDeletionError) Error() string {
	return "device deletion failed at " + e.Stage
}

func (e *DeviceDeletionError) Unwrap() error { return e.Cause }

var ErrDeviceNotFound = errors.New("device not found")
```

1. Load and conditionally mark the device under a short transaction.
2. Call `uploads.BlockAndCancel(device.ID)` and the Redis runtime cleaner.
3. Load all artifact object keys while database metadata still exists.
4. Delete keys with at most four workers; ignore `ErrArtifactNotFound`; stop on other errors.
5. In one transaction query task/file/command IDs, then hard-delete transfer events, artifacts, tasks, command XML, command events, commands, alarms, values, RPC methods, FAP services, profiles, and finally the device.
6. Call `uploads.Release(device.ID)` only after the database transaction succeeds.

Wrap errors in a typed `DeviceDeletionError{Stage, Cause}` whose public message exposes only the stage. Log counts and durations, never XML, credentials, or object contents.

- [ ] **Step 4: Run tests and verify GREEN**

```bash
cd server
go test ./plugin/tr069/service -run '^TestDeviceDeletion' -count=1
```

Expected: PASS, including storage retry and peer isolation.

- [ ] **Step 5: Run service regression tests**

```bash
cd server
go test ./plugin/tr069/service -count=1
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add server/plugin/tr069/service/device_deletion*
git commit -m "feat(tr069): cascade device-owned data deletion"
```

---

### Task 5: Inform guard and management API wiring

**Files:**

- Modify: `server/plugin/tr069/adapter/gorm_repo.go`
- Modify: `server/plugin/tr069/adapter/gorm_repo_test.go`
- Modify: `server/plugin/tr069/handler/cwmp.go`
- Modify: `server/plugin/tr069/handler/cwmp_test.go`
- Modify: `server/plugin/tr069/api/device.go`
- Modify: `server/plugin/tr069/api/device_test.go`
- Modify: `server/plugin/tr069/router/device.go`
- Modify: `server/plugin/tr069/router/device_command_test.go`
- Modify: `server/plugin/tr069/plugin.go`

**Interfaces:**

- Consumes: `DeviceDeletionService.Delete` and `ErrDeviceDeleting`.
- Preserves: existing `DELETE /tr069/device/:deviceId` route and Casbin permission.

- [ ] **Step 1: Write failing Inform/API tests**

Add an adapter test with an existing `DeletingAt` device and assert `UpsertFromInform` returns `ErrDeviceDeleting` without changing `last_inform` or inserting parameters.

Add API tests using a fake deletion service:

```go
type fakeDeviceDeletion struct { result service.DeviceDeletionResult; err error }
func (f fakeDeviceDeletion) Delete(context.Context, uint) (service.DeviceDeletionResult, error) {
	return f.result, f.err
}
```

Verify invalid ID, missing device, stage-safe failure, and success. Add a router contract assertion that the DELETE route still exists and calls the injected API.

- [ ] **Step 2: Run tests and verify RED**

```bash
cd server
go test ./plugin/tr069/adapter ./plugin/tr069/handler ./plugin/tr069/api ./plugin/tr069/router -run 'DeletingInform|DeleteDevice' -count=1
```

Expected: FAIL because Inform is not guarded and the API is not injectable.

- [ ] **Step 3: Add Inform protection and API injection**

Before `UpsertFromInform` performs its upsert, load any existing row by serial number with `Unscoped`, selecting `id,deleting_at`. Return `service.ErrDeviceDeleting` when marked; do not revive or update it. Map this error in the CWMP handler to a controlled conflict response and do not expose the internal error text.

Replace `type DeviceApi struct{}` with an injected interface:

```go
type DeviceDeletion interface {
	Delete(context.Context, uint) (service.DeviceDeletionResult, error)
}

type DeviceApi struct { deletion DeviceDeletion }
func NewDeviceApi(deletion DeviceDeletion) *DeviceApi
```

`DeleteDevice` parses a positive integer, delegates once, maps not-found and typed stage errors, and contains no table-specific SQL. Update `DeviceRouter` to accept the API/deletion dependency.

In `plugin.Register`, after `StartTR069Server` has initialized the object store and upload registry, construct:

```go
runtimeCleaner := adapter.NewRedisDeviceRuntimeCleaner(global.GVA_REDIS)
deletion := service.NewDeviceDeletionService(
	global.GVA_DB,
	initialize.CurrentArtifactStore(),
	initialize.CurrentUploadRuntimeRegistry(),
	runtimeCleaner,
)
```

Pass it to the device router. If file ingress is disabled, deletion remains valid when the device has no artifact rows; artifact rows with no store return a storage-stage failure.

- [ ] **Step 4: Run tests and verify GREEN**

```bash
cd server
go test ./plugin/tr069/adapter ./plugin/tr069/handler ./plugin/tr069/api ./plugin/tr069/router -run 'DeletingInform|DeleteDevice' -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add server/plugin/tr069/adapter/gorm_repo* server/plugin/tr069/handler/cwmp* server/plugin/tr069/api/device* server/plugin/tr069/router/device* server/plugin/tr069/plugin.go
git commit -m "feat(tr069): wire safe device cascade deletion"
```

---

### Task 6: Device-list destructive UX

**Files:**

- Modify: `web/src/plugin/tr069/view/device/index.vue`
- Modify: `web/src/plugin/tr069/view/device/device-rpc-actions.contract.test.js`

**Interfaces:**

- Consumes: device response field `deleting` and existing `deleteDevice(deviceId)` API.
- Produces: one destructive confirmation and per-row deletion loading state.

- [ ] **Step 1: Write the failing frontend contract test**

Assert the page contains the exact permanent-deletion warning, tracks the deleting row ID, hides normal actions for `row.deleting`, labels deletion retry correctly, and disables duplicate deletion requests:

```js
assert.match(source, /永久删除该设备的参数树、RPC 记录、告警和全部日志文件/)
assert.match(source, /deletingDeviceId/)
assert.match(source, /scope\.row\.deleting/)
assert.match(source, /重试删除/)
assert.match(source, /:loading="deletingDeviceId === scope\.row\.ID"/)
```

- [ ] **Step 2: Run the test and verify RED**

```bash
cd web
node --test src/plugin/tr069/view/device/device-rpc-actions.contract.test.js
```

Expected: FAIL because the current confirmation is generic and has no per-row state.

- [ ] **Step 3: Implement the minimal UI behavior**

Add `const deletingDeviceId = ref(0)`. Replace the confirmation with:

```text
删除设备将永久删除该设备的参数树、RPC 记录、告警和全部日志文件，且无法恢复。确定继续吗？
```

Set/reset the row ID in `try/finally`. For a `deleting` row, hide parameter/RPC menus and show only `重试删除`; otherwise preserve the current GVA button style and action alignment. Refresh the list only after the API reports success.

- [ ] **Step 4: Run frontend tests and verify GREEN**

```bash
cd web
node --test src/plugin/tr069/view/device/device-rpc-actions.contract.test.js
node --test src/plugin/tr069/utils/device-actions.test.js
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add web/src/plugin/tr069/view/device/index.vue web/src/plugin/tr069/view/device/device-rpc-actions.contract.test.js
git commit -m "feat(tr069): clarify permanent device deletion"
```

---

### Task 7: Full verification and local runtime restart

**Files:**

- Verify only; update implementation files only if a test exposes a defect.

**Interfaces:**

- Verifies the complete feature against existing TR-069 behavior and the local two-BS environment.

- [ ] **Step 1: Format and statically check Go changes**

```bash
cd server
gofmt -w plugin/tr069/model/device.go plugin/tr069/model/response/device.go plugin/tr069/service/device_state.go plugin/tr069/service/device_deletion.go plugin/tr069/service/device_deletion_test.go plugin/tr069/service/upload_runtime_registry.go plugin/tr069/service/upload_runtime_registry_test.go plugin/tr069/service/transfer_receiver.go plugin/tr069/service/transfer_receiver_test.go plugin/tr069/service/command_manager.go plugin/tr069/service/command_manager_test.go plugin/tr069/service/upload_device_resolver.go plugin/tr069/service/transfer_store.go plugin/tr069/adapter/redis_device_runtime_cleaner.go plugin/tr069/adapter/redis_device_runtime_cleaner_test.go plugin/tr069/adapter/redis_upload_identity.go plugin/tr069/adapter/redis_upload_identity_test.go plugin/tr069/adapter/gorm_repo.go plugin/tr069/adapter/gorm_repo_test.go plugin/tr069/handler/cwmp.go plugin/tr069/handler/cwmp_test.go plugin/tr069/initialize/server.go plugin/tr069/initialize/server_runtime_test.go plugin/tr069/api/device.go plugin/tr069/api/device_test.go plugin/tr069/api/artifact_test.go plugin/tr069/router/device.go plugin/tr069/router/device_command_test.go plugin/tr069/plugin.go
go vet ./plugin/tr069/...
```

Expected: no formatting diff after the first run and no vet errors. Limit `gofmt` arguments to files actually changed if the broad directory command touches unrelated user work.

- [ ] **Step 2: Run all TR-069 backend tests**

```bash
cd server
go test ./plugin/tr069/... -count=1
```

Expected: PASS.

- [ ] **Step 3: Run all TR-069 frontend contract/unit tests and build**

```bash
cd web
rg --files src/plugin/tr069 -g '*.test.js' | xargs node --test
npm run build
```

Expected: all tests pass and the production build succeeds.

- [ ] **Step 4: Restart GVA components and BS stacks**

Use the project `oamstart` workflow. Verify:

```text
Frontend  http://127.0.0.1:18080
Backend   http://127.0.0.1:18888
ACS       :7458
BS-01     Web :8400 / Connection Request :7547
BS-02     Web :8402 / Connection Request :7540
```

Expected: both devices send Inform successfully before the destructive test.

- [ ] **Step 5: Perform the two-device integration test**

Create/confirm parameters, RPC records, alarms, and at least one MinIO log object for each device. Delete one device from the UI and verify:

```sql
-- Each target-scoped query must return 0.
SELECT COUNT(*) FROM tr069_devices WHERE id = ?;
SELECT COUNT(*) FROM tr069_datamodel_values WHERE device_id = ?;
SELECT COUNT(*) FROM tr069_commands WHERE device_id = ?;
SELECT COUNT(*) FROM tr069_alarms WHERE device_id = ?;
SELECT COUNT(*) FROM tr069_transfer_tasks WHERE device_id = ?;
SELECT COUNT(*) FROM tr069_artifacts WHERE device_id = ?;
```

Confirm target MinIO keys are absent and peer counts/objects are unchanged. Then allow the deleted BS to send Inform and verify it registers with a new database ID and no old parameters/RPC/log history is restored.

- [ ] **Step 6: Inspect the final diff**

```bash
git diff --check
git status --short
git log --oneline --decorate -10
```

Expected: no whitespace errors, only planned files changed, and all task commits present. Do not merge or push without a separate user request.

---

## Plan Self-Review

- Every confirmed requirement is mapped to an implementation task.
- MinIO failure retains database metadata and deletion state; retry is idempotent.
- The plan explicitly protects the second device and same-IP upload identity bindings.
- The plan does not rely on database foreign keys or frontend multi-request cleanup.
- The upload cancellation path attaches `ArtifactWriter.Abort`, not only a context cancel.
- Deletion completion releases the in-memory block, while the deleted database row allows later Inform registration.
- No placeholder implementation or undefined cross-task dependency remains.
