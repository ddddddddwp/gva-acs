# TR-069 Log Collection Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add authenticated PUT/POST BS log ingestion on ACS port 7458, persist log artifacts in MinIO, correlate active and periodic transfers with registered TR-069 devices, and expose a GVA “日志文件” page with device-ID filtering and protected downloads.

**Architecture:** Keep `POST /acs` on the existing CWMP middleware chain and mount file channels on a separate raw-stream chain. MySQL is the transfer/artifact/event source of truth, Redis stores expiring Inform/IP identity bindings and Digest replay state, and a storage-neutral `ArtifactStore` streams bytes to MinIO. Management traffic remains on the GVA JWT/Casbin API and never exposes MinIO internals.

**Tech Stack:** Go 1.23, Gin, GORM, MySQL/SQLite tests, go-redis/v9, minio-go/v7, Zap, Vue 3, Element Plus, Axios, Node test runner, Vite.

## Global Constraints

- Device routes remain on port `7458`: `POST /acs`, `PUT/POST /acs/log`, with `/acs/pm` and `/acs/mr` registered only when their channels are enabled.
- PUT and POST use one raw-byte handler; POST MUST NOT require or parse multipart/form-data. Unsupported methods return `405` and `Allow: PUT, POST`.
- Enabled LOG ingress requires a non-empty shared username and password and supports both Basic and Digest (`qop=auth`, MD5, MD5-sess).
- Shared credentials authenticate the LOG channel only; they MUST NOT identify a device.
- A file is accepted only when source IP plus recent Inform/active task resolves to one registered device; zero or multiple candidates return a generic `403`.
- Forwarded IP headers are trusted only from configured proxy CIDRs.
- Default identity binding TTL is `30m`; default file limit is `64 MiB`; default upload timeout is `10m`; default global concurrency is `4`; default per-device concurrency is `1`; default retention is `30d`.
- File bytes MUST NOT pass through RawDump/XML parsing, `io.ReadAll`, Zap, Trace, operation-log response buffering, or database payload columns.
- MySQL owns task/artifact/event state; Redis state is reconstructable; downloads expose only `AVAILABLE` artifacts.
- Production artifact storage uses the neutral interface with MinIO/S3-compatible implementation. Provider object keys and credentials never appear in frontend DTOs.
- GVA does not configure or map `Device.LogMgmt.*`, and this change does not add TR-181 support.
- The new page must use `gva-search-box`, `gva-table-box`, `gva-pagination`, and Element Plus theme variables without fixed light-only colors.
- All new behavior follows TDD: failing focused test, minimal implementation, passing focused test, regression test, then commit.

---

## File Structure

Backend files to create:

- `server/plugin/tr069/model/transfer.go`: transfer task, artifact, event, statuses, and table names.
- `server/plugin/tr069/model/request/artifact.go`: artifact list and log-collection request DTOs.
- `server/plugin/tr069/model/response/artifact.go`: secret-free list DTO.
- `server/plugin/tr069/service/artifact_store.go`: storage-neutral reader/writer contracts.
- `server/plugin/tr069/service/transfer_store.go`: transactional GORM persistence and conditional updates.
- `server/plugin/tr069/service/transfer_lifecycle.go`: ACTIVE/PERIODIC classification and completion state machine.
- `server/plugin/tr069/service/transfer_receiver.go`: admission, limited streaming, SHA-256, storage commit, and idempotency.
- `server/plugin/tr069/service/transfer_worker.go`: stale receive reconciliation and retention cleanup.
- `server/plugin/tr069/adapter/minio_artifact_store.go`: MinIO streaming driver.
- `server/plugin/tr069/adapter/redis_upload_identity.go`: recent Inform/source-IP bindings.
- `server/plugin/tr069/adapter/redis_digest_nonce.go`: Digest nonce and replay state.
- `server/plugin/tr069/adapter/log_upload_payload.go`: hide configured Upload credentials at rest and hydrate them at dispatch.
- `server/plugin/tr069/handler/client_ip.go`: trusted-proxy-aware IP resolution.
- `server/plugin/tr069/handler/file_ingress.go`: PUT/POST method guard and Gin ingress handler.
- `server/plugin/tr069/middleware/file_auth.go`: Basic/Digest authentication.
- `server/plugin/tr069/middleware/download_audit.go`: metadata-only operation audit without response buffering.
- `server/plugin/tr069/api/artifact.go`: list/download management APIs.
- `server/plugin/tr069/router/artifact.go`: JWT/Casbin artifact routes.

Frontend files to create:

- `web/src/plugin/tr069/api/log-file.js`: paginated query and blob download calls.
- `web/src/plugin/tr069/view/log-file/log-file-view.js`: pure format/status/download helpers.
- `web/src/plugin/tr069/view/log-file/log-file-view.test.js`: helper tests.
- `web/src/plugin/tr069/view/log-file/log-file.contract.test.js`: API/menu/theme/source contract tests.
- `web/src/plugin/tr069/view/log-file/index.vue`: GVA-styled log artifact page.

Existing integration files to modify:

- `server/plugin/tr069/config/config.go`, `runtime.go`, and tests.
- `server/plugin/tr069/initialize/server.go`, `gorm.go`, `api.go`, `menu.go`, and tests.
- `server/plugin/tr069/adapter/gorm_repo.go`, `datamodel_hook.go`, and tests.
- `server/plugin/tr069/service/command_manager.go` and tests.
- `server/plugin/tr069/api/command.go`, `router/device.go`, `plugin.go`.
- `server/plugin/tr069/engine/engine.go`.
- `web/src/plugin/tr069/view/device/components/rpc-command-dialog.vue` and its contract test.
- `server/config.yaml`, `server/config.local.yaml`, `server/config.docker.yaml`.
- `deploy/docker-compose/docker-compose.yaml`.
- `web/src/pathInfo.json`, regenerated by Vite.

### Task 1: Typed File-Ingress Configuration and Validation

**Files:**
- Modify: `server/plugin/tr069/config/config.go`
- Modify: `server/plugin/tr069/config/runtime.go`
- Modify: `server/plugin/tr069/config/runtime_test.go`
- Modify: `server/plugin/tr069/initialize/viper.go`
- Modify: `server/plugin/tr069/initialize/viper_test.go`

**Interfaces:**
- Produces: `config.ValidateFileIngress(TR069Config) error`
- Produces: `config.Runtime.FileIngress FileIngressRuntime`
- Consumes later: middleware, identity binding, receiver, MinIO initializer, workers.

- [ ] **Step 1: Write failing default, clone, and validation tests**

```go
func TestNormalizeFileIngressDefaults(t *testing.T) {
	got := NormalizeRuntimeConfig(TR069Config{FileIngress: FileIngressConfig{Enabled: true, Authentication: FileIngressAuthConfig{Username: "log", Password: "secret"}, Channels: map[string]TransferChannelConfig{"log": {Enabled: true, Path: "/acs/log"}}, ArtifactStore: ArtifactStoreConfig{Driver: "minio", Endpoint: "minio:9000", Bucket: "gva-tr069", AccessKey: "a", SecretKey: "s"}}})
	if got.FileIngress.IdentityBindingTTL != 1800 || got.FileIngress.Channels["log"].MaxFileSize != 64<<20 || got.FileIngress.Channels["log"].UploadTimeout != 600 { t.Fatalf("defaults=%#v", got.FileIngress) }
}

func TestValidateFileIngressRejectsEmptySharedCredentials(t *testing.T) {
	cfg := validFileIngressConfig()
	cfg.FileIngress.Authentication.Password = ""
	if err := ValidateFileIngress(cfg); err == nil || strings.Contains(err.Error(), "secret") { t.Fatalf("error=%v", err) }
}
```

- [ ] **Step 2: Run tests and verify the new types/functions are missing**

Run: `cd server && go test ./plugin/tr069/config ./plugin/tr069/initialize -run 'FileIngress|ReloadConfig'`

Expected: FAIL because `FileIngressConfig`, `ArtifactStoreConfig`, and `ValidateFileIngress` do not exist.

- [ ] **Step 3: Add exact configuration types and runtime defaults**

```go
type FileIngressAuthConfig struct {
	Username string   `mapstructure:"username" json:"-" yaml:"username"`
	Password string   `mapstructure:"password" json:"-" yaml:"password"`
	Schemes  []string `mapstructure:"schemes" json:"schemes" yaml:"schemes"`
	Realm    string   `mapstructure:"realm" json:"realm" yaml:"realm"`
	NonceTTL int      `mapstructure:"nonceTTL" json:"nonceTTL" yaml:"nonceTTL"`
}
type TransferChannelConfig struct {
	Enabled bool `mapstructure:"enabled" json:"enabled" yaml:"enabled"`
	Path string `mapstructure:"path" json:"path" yaml:"path"`
	MaxFileSize int64 `mapstructure:"maxFileSize" json:"maxFileSize" yaml:"maxFileSize"`
	MaxConcurrent int `mapstructure:"maxConcurrent" json:"maxConcurrent" yaml:"maxConcurrent"`
	MaxConcurrentPerDevice int `mapstructure:"maxConcurrentPerDevice" json:"maxConcurrentPerDevice" yaml:"maxConcurrentPerDevice"`
	UploadTimeout int `mapstructure:"uploadTimeout" json:"uploadTimeout" yaml:"uploadTimeout"`
	RetentionDays int `mapstructure:"retentionDays" json:"retentionDays" yaml:"retentionDays"`
	StoragePrefix string `mapstructure:"storagePrefix" json:"storagePrefix" yaml:"storagePrefix"`
}
type ArtifactStoreConfig struct {
	Driver string `mapstructure:"driver" json:"driver" yaml:"driver"`
	Endpoint string `mapstructure:"endpoint" json:"endpoint" yaml:"endpoint"`
	Bucket string `mapstructure:"bucket" json:"bucket" yaml:"bucket"`
	AccessKey string `mapstructure:"accessKey" json:"-" yaml:"accessKey"`
	SecretKey string `mapstructure:"secretKey" json:"-" yaml:"secretKey"`
	UseSSL bool `mapstructure:"useSSL" json:"useSSL" yaml:"useSSL"`
	Prefix string `mapstructure:"prefix" json:"prefix" yaml:"prefix"`
}
```

Add `FileIngressConfig` with `Enabled`, `PublicBaseURL`, `TrustedProxies`, `IdentityBindingTTL`, `Authentication`, `Channels`, and `ArtifactStore`. Normalize exact defaults from Global Constraints and build parsed duration fields in `Runtime`.

- [ ] **Step 4: Validate enabled channels without logging secrets**

`ValidateFileIngress` must reject empty username/password/realm, unsupported schemes, duplicate/invalid paths, non-positive limits, invalid CIDRs, a driver other than `minio`, and incomplete MinIO settings. `ReloadConfig` publishes only a valid snapshot; plugin startup treats enabled invalid configuration as fatal.

- [ ] **Step 5: Run focused and package tests**

Run: `cd server && go test ./plugin/tr069/config ./plugin/tr069/initialize`

Expected: PASS.

- [ ] **Step 6: Commit configuration foundation**

```bash
git add server/plugin/tr069/config server/plugin/tr069/initialize/viper.go server/plugin/tr069/initialize/viper_test.go
git commit -m "feat(tr069): add file ingress configuration"
```

### Task 2: Transfer Models and Transactional Store

**Files:**
- Create: `server/plugin/tr069/model/transfer.go`
- Create: `server/plugin/tr069/model/request/artifact.go`
- Create: `server/plugin/tr069/model/response/artifact.go`
- Create: `server/plugin/tr069/service/transfer_store.go`
- Create: `server/plugin/tr069/service/transfer_store_test.go`
- Modify: `server/plugin/tr069/initialize/gorm.go`

**Interfaces:**
- Produces: `service.NewTransferStore(*gorm.DB) *TransferStore`
- Produces: `CreateActive`, `CreatePeriodicReceiving`, `FindUniqueWaitingActive`, `MarkArtifactAvailable`, `TransitionTask`, `ListArtifacts`, `GetAvailableArtifact`, `ListStaleReceiving`.
- Consumes: existing `model.Device` and `model.Command`.

- [ ] **Step 1: Write failing model/store tests**

Cover command+ACTIVE task rollback, one active task per command, periodic task+artifact+event in one transaction, exact device ID pagination, conditional version conflicts, and absence of `objectKey`/credentials in JSON.

```go
func TestTransferStoreCreatesPeriodicReceiveAtomically(t *testing.T) {
	store, db, device := newTransferStoreTest(t)
	task, artifact, err := store.CreatePeriodicReceiving(context.Background(), device.ID, "LOG", ReceiveMetadata{ArtifactID: "artifact-1", ObjectKey: "artifacts/log/1/2026/07/19/artifact-1"})
	if err != nil || task.Source != model.TransferSourcePeriodic || artifact.Status != model.ArtifactStatusReceiving { t.Fatalf("task=%#v artifact=%#v err=%v", task, artifact, err) }
	var events int64
	db.Model(new(model.TransferEvent)).Where("task_id = ?", task.TaskID).Count(&events)
	if events != 1 { t.Fatalf("events=%d", events) }
}
```

- [ ] **Step 2: Run and observe missing transfer types**

Run: `cd server && go test ./plugin/tr069/service -run 'TransferStore|TransferModel'`

Expected: FAIL to compile.

- [ ] **Step 3: Implement exact models and statuses**

```go
const (
	TransferSourceActive = "ACTIVE"
	TransferSourcePeriodic = "PERIODIC"
	TransferStatusWaitingFile = "WAITING_FILE"
	TransferStatusReceiving = "RECEIVING"
	TransferStatusWaitingTransfer = "WAITING_TRANSFER"
	TransferStatusCompleted = "COMPLETED"
	TransferStatusFailed = "FAILED"
	TransferStatusTimeout = "TIMEOUT"
	ArtifactStatusReceiving = "RECEIVING"
	ArtifactStatusAvailable = "AVAILABLE"
	ArtifactStatusFailed = "FAILED"
	ArtifactStatusDeleting = "DELETING"
	ArtifactStatusDeleted = "DELETED"
)
```

`TransferTask` uses `TaskID string` as primary key and stores `DeviceID`, `Channel`, `Source`, nullable `CommandID`/`CommandKey`, status, nullable UploadResponse status, file/TransferComplete timestamps, failure fields, `Version`, and timestamps. `Artifact` uses `ArtifactID`, `TaskID`, `DeviceID`, channel/status, driver, `ObjectKey json:"-"`, sanitized original name, content type, size, SHA-256, source IP, receive/delete timestamps. `TransferEvent` stores task/artifact IDs, code, phase, from/to status, message, metadata JSON, and timestamp.

- [ ] **Step 4: Implement GORM store operations with row locks and conditional updates**

All creation methods must append the initial event in the same transaction. `TransitionTask` and artifact finalization use `WHERE status IN ? AND version = ?`. List joins `tr069_devices` for SerialNumber/OUI and applies `device_id = ?` only when the exact filter is non-zero.

- [ ] **Step 5: Register all models in AutoMigrate and run tests**

Run: `cd server && go test ./plugin/tr069/service ./plugin/tr069/initialize`

Expected: PASS.

- [ ] **Step 6: Commit the persistence slice**

```bash
git add server/plugin/tr069/model server/plugin/tr069/service/transfer_store* server/plugin/tr069/initialize/gorm.go
git commit -m "feat(tr069): persist log transfer metadata"
```

### Task 3: Storage-Neutral Streaming and MinIO Driver

**Files:**
- Create: `server/plugin/tr069/service/artifact_store.go`
- Create: `server/plugin/tr069/service/artifact_store_contract_test.go`
- Create: `server/plugin/tr069/adapter/minio_artifact_store.go`
- Create: `server/plugin/tr069/adapter/minio_artifact_store_test.go`

**Interfaces:**
- Produces:

```go
type ArtifactStore interface {
	Begin(context.Context, ObjectSpec) (ArtifactWriter, error)
	Open(context.Context, string) (io.ReadCloser, ObjectStat, error)
	Stat(context.Context, string) (ObjectStat, error)
	Delete(context.Context, string) error
}
type ArtifactWriter interface {
	io.Writer
	Commit(context.Context) (ObjectStat, error)
	Abort(context.Context) error
}
```

- [ ] **Step 1: Write one reusable contract suite**

The suite must verify streaming writes, commit visibility, open bytes, stat metadata, delete, canceled context, idempotent Abort, Commit-after-Abort failure, and no object visibility before Commit. Run it against the in-memory test store and the MinIO client adapter fake.

- [ ] **Step 2: Run and verify interface/driver absence**

Run: `cd server && go test ./plugin/tr069/service ./plugin/tr069/adapter -run 'ArtifactStore|MinioArtifact'`

Expected: FAIL to compile.

- [ ] **Step 3: Implement object naming and in-memory test store**

```go
func ArtifactObjectKey(prefix, channel string, deviceID uint, receivedAt time.Time, artifactID string) string {
	return path.Join(strings.Trim(prefix, "/"), strings.ToLower(channel), strconv.FormatUint(uint64(deviceID), 10), receivedAt.UTC().Format("2006/01/02"), artifactID)
}
```

Reject empty IDs and path traversal. Original filenames are cleaned with `filepath.Base`, control characters removed, and length capped; they never become an object key.

- [ ] **Step 4: Implement MinIO writer with `io.Pipe` and `PutObject(..., -1, ...)`**

`Begin` starts one goroutine that calls MinIO `PutObject` on the pipe reader. `Write` feeds the pipe. `Commit` closes the writer and waits for the upload result. `Abort` cancels context and closes the pipe with a sentinel error. No `bytes.Buffer`, multipart form object, or whole-file byte slice is permitted.

- [ ] **Step 5: Run focused tests and static source guard**

Run:

```bash
cd server && go test ./plugin/tr069/service ./plugin/tr069/adapter -run 'ArtifactStore|MinioArtifact'
! rg -n 'io\.ReadAll|bytes\.Buffer|multipart\.FileHeader' plugin/tr069/adapter/minio_artifact_store.go
```

Expected: tests PASS and source guard exits 0.

- [ ] **Step 6: Commit storage abstraction**

```bash
git add server/plugin/tr069/service/artifact_store* server/plugin/tr069/adapter/minio_artifact_store*
git commit -m "feat(tr069): stream artifacts to minio"
```

### Task 4: Trusted Client IP and Inform Identity Binding

**Files:**
- Create: `server/plugin/tr069/handler/client_ip.go`
- Create: `server/plugin/tr069/handler/client_ip_test.go`
- Create: `server/plugin/tr069/adapter/redis_upload_identity.go`
- Create: `server/plugin/tr069/adapter/redis_upload_identity_test.go`
- Modify: `server/plugin/tr069/handler/cwmp.go`
- Modify: `server/plugin/tr069/handler/cwmp_test.go`
- Modify: `server/plugin/tr069/adapter/gorm_repo.go`
- Modify: `server/plugin/tr069/adapter/gorm_repo_test.go`

**Interfaces:**
- Produces: `handler.NewClientIPResolver([]string) (*ClientIPResolver, error)` and `Resolve(*http.Request) string`.
- Produces: `adapter.UploadIdentityStore.Bind` and `Resolve`.
- Produces: `GormDeviceRepo.SetUploadIdentityBinder(UploadIdentityBinder)`.

- [ ] **Step 1: Write failing IP and binding tests**

Cover direct IPv4/IPv6, trusted first-hop forwarding, untrusted spoofed headers, invalid forwarded values, two devices sharing one IP, expired members, Inform persistence failure, and MySQL unique fallback.

- [ ] **Step 2: Run focused tests**

Run: `cd server && go test ./plugin/tr069/handler ./plugin/tr069/adapter -run 'ClientIP|UploadIdentity|InformBinding'`

Expected: FAIL to compile.

- [ ] **Step 3: Implement trusted-proxy IP resolution**

`Resolve` checks the socket peer against parsed trusted CIDRs before reading forwarding headers, normalizes with `netip.ParseAddr(...).Unmap().String()`, and otherwise returns the normalized `RemoteAddr` host. Replace the current blindly trusting `remoteIPFromRequest` call in `CWMPHandler`.

- [ ] **Step 4: Implement Redis multi-candidate bindings**

Use a sorted set `tr069:file-ingress:ip:<ip>` where member is decimal device ID and score is expiry Unix milliseconds. Store identity JSON separately at `tr069:file-ingress:identity:<ip>:<deviceID>` with the same TTL. Resolve removes expired scores and returns all live candidates, so a second device cannot overwrite the first and hide ambiguity.

- [ ] **Step 5: Refresh binding only after successful Inform upsert**

After `UpsertFromInform` re-queries the persisted device ID, call:

```go
type UploadIdentityBinder interface {
	Bind(context.Context, UploadIdentityBinding, time.Duration) error
}
```

Binding failures log device ID and stage without failing the accepted Inform. No binding is written when device persistence fails.

- [ ] **Step 6: Run tests and commit identity support**

Run: `cd server && go test ./plugin/tr069/handler ./plugin/tr069/adapter`

```bash
git add server/plugin/tr069/handler/client_ip* server/plugin/tr069/handler/cwmp* server/plugin/tr069/adapter/redis_upload_identity* server/plugin/tr069/adapter/gorm_repo*
git commit -m "feat(tr069): bind inform identity to upload source"
```

### Task 5: Shared Basic and Digest Authentication

**Files:**
- Create: `server/plugin/tr069/middleware/file_auth.go`
- Create: `server/plugin/tr069/middleware/file_auth_test.go`
- Create: `server/plugin/tr069/adapter/redis_digest_nonce.go`
- Create: `server/plugin/tr069/adapter/redis_digest_nonce_test.go`

**Interfaces:**
- Produces: `middleware.NewFileAuthenticator(CredentialProvider, DigestNonceStore) *FileAuthenticator`.
- Produces: `Authenticate(*http.Request) (channel string, challenge []string, err error)`.

- [ ] **Step 1: Add table tests using RFC Digest vectors**

Test Basic success/failure, PUT and POST Digest method sensitivity, qop=auth, MD5, MD5-sess, bad realm/URI/response, expired nonce, repeated nonce-count, missing auth, and empty configured credentials. Assert failure messages and logs never include password or full Authorization.

- [ ] **Step 2: Run focused test and see missing authenticator**

Run: `cd server && go test ./plugin/tr069/middleware ./plugin/tr069/adapter -run 'FileAuth|DigestNonce'`

Expected: FAIL to compile.

- [ ] **Step 3: Implement strict Basic and Digest parser/verifier**

Use `subtle.ConstantTimeCompare` for Basic and final Digest response. Digest computes HA2 with `r.Method + ":" + digestURI`, validates the digest URI against `r.URL.RequestURI()`, permits only configured schemes/algorithms, and emits both configured challenges using `Header().Add("WWW-Authenticate", value)`.

- [ ] **Step 4: Implement expiring nonce and replay protection**

`Issue` generates 32 random bytes and stores the nonce in Redis for `nonceTTL`. `Consume` atomically rejects missing/expired nonce and any `nc` not greater than the stored count for `(nonce, username, cnonce)`.

- [ ] **Step 5: Run tests and commit authentication**

Run: `cd server && go test ./plugin/tr069/middleware ./plugin/tr069/adapter -run 'FileAuth|DigestNonce'`

```bash
git add server/plugin/tr069/middleware/file_auth* server/plugin/tr069/adapter/redis_digest_nonce*
git commit -m "feat(tr069): authenticate log file ingress"
```

### Task 6: PUT/POST Streaming Ingress and Device Resolution

**Files:**
- Create: `server/plugin/tr069/service/transfer_receiver.go`
- Create: `server/plugin/tr069/service/transfer_receiver_test.go`
- Create: `server/plugin/tr069/handler/file_ingress.go`
- Create: `server/plugin/tr069/handler/file_ingress_test.go`
- Modify: `server/plugin/tr069/initialize/server.go`
- Modify: `server/plugin/tr069/initialize/server_runtime_test.go`

**Interfaces:**
- Produces: `TransferReceiver.Receive(context.Context, ReceiveRequest) (model.Artifact, error)`.
- Produces: `handler.NewFileIngressHandler(auth, resolver, receiver, channelProvider) gin.HandlerFunc`.

- [ ] **Step 1: Write failing handler tests for PUT and POST parity**

For both methods assert raw bytes reach the fake store, Basic/Digest happens before body reads, a registered unique device returns 204, unknown/ambiguous device returns generic 403, oversized declared/chunked body returns 413, device/global saturation returns 503 plus Retry-After, and DELETE returns 405 plus `Allow: PUT, POST`. Assert `/acs/log` bytes never appear in RawDump output.

- [ ] **Step 2: Run focused tests**

Run: `cd server && go test ./plugin/tr069/handler ./plugin/tr069/service ./plugin/tr069/initialize -run 'FileIngress|TransferReceiver|RouteSpecific'`

Expected: FAIL to compile.

- [ ] **Step 3: Implement unique device resolver and admission controller**

Resolution order is unique waiting ACTIVE task by source IP, live Redis Inform candidates, then MySQL `ip = ? AND last_inform >= cutoff LIMIT 2`. Reject zero/multiple/deleted/incomplete-identity results. Admission acquires global, channel, then `deviceID` semaphore and releases in reverse order on every return path.

- [ ] **Step 4: Implement bounded receive pipeline**

Use `http.MaxBytesReader` plus `io.LimitedReader{N: max+1}`, `io.MultiWriter(writer, sha256.New())`, and a pooled 64 KiB buffer. Create RECEIVING metadata before `Begin`, abort on context/timeout/copy/size failure, commit object, verify returned size, conditionally mark AVAILABLE, and return an idempotent success for same task+size+SHA-256.

- [ ] **Step 5: Mount route-specific middleware on 7458**

Move RawDump and RawResponseDump from global engine middleware to the CWMP route group. Mount each enabled file path with `Any`, run the PUT/POST method guard first, then authentication and the shared ingress handler. Disabled PM/MR paths remain unmounted and cannot create LOG artifacts.

- [ ] **Step 6: Run full ingress regression and commit**

Run: `cd server && go test ./plugin/tr069/handler ./plugin/tr069/service ./plugin/tr069/initialize ./plugin/tr069/middleware`

```bash
git add server/plugin/tr069/service/transfer_receiver* server/plugin/tr069/handler/file_ingress* server/plugin/tr069/initialize/server*
git commit -m "feat(tr069): receive put and post log streams"
```

### Task 7: Configured Active Upload Command and Secret Hydration

**Files:**
- Create: `server/plugin/tr069/adapter/log_upload_payload.go`
- Create: `server/plugin/tr069/adapter/log_upload_payload_test.go`
- Modify: `server/plugin/tr069/model/request/artifact.go`
- Modify: `server/plugin/tr069/service/command_manager.go`
- Modify: `server/plugin/tr069/service/command_manager_test.go`
- Modify: `server/plugin/tr069/api/command.go`
- Create: `server/plugin/tr069/api/command_upload_test.go`
- Modify: `server/plugin/tr069/engine/engine.go`
- Modify: `web/src/plugin/tr069/view/device/components/rpc-command-dialog.vue`
- Modify: `web/src/plugin/tr069/view/device/device-rpc-actions.contract.test.js`

**Interfaces:**
- Produces: `service.WithCommandCreatedHook(CommandCreatedHook)` for all submissions.
- Produces: `adapter.LogUploadPayloadCodec` implementing payload Protect/Hydrate.
- Produces: `request.LogCollectionRequest{FileType string, DelaySeconds int}`.

- [ ] **Step 1: Write failing atomic task and secret-at-rest tests**

Submit Upload with configured URL/user/password and assert command plus ACTIVE task are one transaction, CommandKey equals command ID without hyphens, persisted params contain placeholders rather than credentials, dispatch hydration contains current runtime credentials, and retries create new task/CommandKey.

- [ ] **Step 2: Run focused tests**

Run: `cd server && go test ./plugin/tr069/service ./plugin/tr069/adapter ./plugin/tr069/api -run 'Upload|CommandCreatedHook|LogPayload'`

Expected: FAIL before the hook/codec exists.

- [ ] **Step 3: Add manager-wide creation hook without breaking SubmitSystem**

Add `createdHook CommandCreatedHook` to `CommandManager`, a `WithCommandCreatedHook` option, and invoke it inside the existing transaction after `NewCommandStore(tx).Create`. If both manager-wide and submission-specific hooks exist, invoke manager-wide first and rollback on either error.

- [ ] **Step 4: Build active Upload payload only from runtime configuration**

`CommandApi.Upload` binds only `LogCollectionRequest`, validates enabled LOG ingress, and creates `req.UploadRequest{FileType, DelaySeconds, URL: publicBaseURL+logPath, Username: configuredUsername, Password: configuredPassword}`. The payload codec replaces username/password with fixed placeholders before persistence and hydrates from `config.CurrentRuntime()` immediately before core dispatch.

- [ ] **Step 5: Compose existing connection-profile and log codecs**

Introduce a small ordered composite implementing both `service.CommandPayloadProtector` and `adapter.CommandPayloadHydrator`; wire it into the API manager and `RedisCommandSource` in `engine.go`. Do not modify the external core SDK for this task.

- [ ] **Step 6: Simplify the existing Upload dialog**

For action `upload`, show only FileType and DelaySeconds; keep URL/username/password fields for Download only. The submitted JSON contains no configured credential fields.

- [ ] **Step 7: Run backend/frontend focused tests and commit**

Run:

```bash
cd server && go test ./plugin/tr069/service ./plugin/tr069/adapter ./plugin/tr069/api ./plugin/tr069/engine
cd ../web && node --test src/plugin/tr069/view/device/device-rpc-actions.contract.test.js
```

```bash
git add server/plugin/tr069/adapter/log_upload_payload* server/plugin/tr069/model/request/artifact.go server/plugin/tr069/service/command_manager* server/plugin/tr069/api/command.go server/plugin/tr069/engine/engine.go web/src/plugin/tr069/view/device
git commit -m "feat(tr069): create configured log upload tasks"
```

### Task 8: UploadResponse, File, and TransferComplete State Machine

**Files:**
- Create: `server/plugin/tr069/service/transfer_lifecycle.go`
- Create: `server/plugin/tr069/service/transfer_lifecycle_test.go`
- Modify: `server/plugin/tr069/adapter/datamodel_hook.go`
- Modify: `server/plugin/tr069/adapter/datamodel_hook_test.go`
- Modify: `server/plugin/tr069/adapter/gorm_repo.go`
- Modify: `server/plugin/tr069/adapter/gorm_repo_test.go`
- Modify: `server/plugin/tr069/engine/engine.go`

**Interfaces:**
- Produces: `OnUploadResponse(ctx, commandID string, status int, at time.Time) error`.
- Produces: `OnTransferComplete(ctx, commandKey string, faultCode int, faultString string, at time.Time) error`.
- Produces: `OnArtifactAvailable(ctx, taskID, artifactID string, at time.Time) error`.

- [ ] **Step 1: Write state-table tests before implementation**

Cover Status 0 before/after file, Status 1 across all file/TransferComplete orders, TransferComplete fault with retained AVAILABLE artifact, duplicate events, unknown CommandKey, periodic completion on file only, and multiple waiting active task rejection.

- [ ] **Step 2: Run focused lifecycle tests**

Run: `cd server && go test ./plugin/tr069/service ./plugin/tr069/adapter -run 'TransferLifecycle|UploadResponse|TransferComplete'`

Expected: FAIL.

- [ ] **Step 3: Implement one locked recompute function**

Every event persists its fact, then `recompute(tx, task)` derives the terminal status:

```text
PERIODIC + artifact AVAILABLE                         -> COMPLETED
ACTIVE status=0 + artifact AVAILABLE                 -> COMPLETED
ACTIVE status=1 + artifact AVAILABLE + TC success    -> COMPLETED
ACTIVE + TC fault                                    -> FAILED (artifact retained)
expired non-terminal                                  -> TIMEOUT
```

All task and command updates are conditional and duplicate terminal events are no-ops.

- [ ] **Step 4: Extend DataModelHook for response and transfer callbacks**

Before delegating an UploadResponse to the base correlation hook, resolve its inflight command and call `OnUploadResponse`. Implement `core.TransferCompleteHook` on `DataModelHook`, pass `Message.CommandKey`, `TransferFaultCode`, and `TransferFaultString`, and return a normal TransferCompleteResponse even for duplicate completion.

- [ ] **Step 5: Keep RPC command lifecycle synchronized**

Status 1 transitions the command from SENT to WAITING_TRANSFER with the existing 12-hour deadline. Successful matching TransferComplete completes the command; transfer fault fails it. Status 0 may complete the RPC command immediately while the transfer task continues waiting for the actual file.

- [ ] **Step 6: Run core/GVA integration tests and commit**

Run: `cd server && go test ./plugin/tr069/service ./plugin/tr069/adapter ./plugin/tr069/engine`

```bash
git add server/plugin/tr069/service/transfer_lifecycle* server/plugin/tr069/adapter/datamodel_hook* server/plugin/tr069/adapter/gorm_repo* server/plugin/tr069/engine/engine.go
git commit -m "feat(tr069): correlate log transfer lifecycle"
```

### Task 9: Reconciliation, Timeouts, and Retention Workers

**Files:**
- Create: `server/plugin/tr069/service/transfer_worker.go`
- Create: `server/plugin/tr069/service/transfer_worker_test.go`
- Modify: `server/plugin/tr069/plugin.go`

**Interfaces:**
- Produces: `TransferWorkers.Run(context.Context)` and `Stop(context.Context) error`.
- Consumes: `TransferStore`, `ArtifactStore`, current runtime timeouts/retention.

- [ ] **Step 1: Write fake-clock worker tests**

Cover committed object with stale RECEIVING metadata, stale row without object, stat mismatch, repeated reconciliation, waiting-file timeout, TransferComplete timeout, successful deletion, failed deletion retry, and two workers competing for one versioned row.

- [ ] **Step 2: Run focused worker tests**

Run: `cd server && go test ./plugin/tr069/service -run 'TransferWorker|Reconcile|Retention'`

Expected: FAIL.

- [ ] **Step 3: Implement bounded scans and idempotent recovery**

Each loop selects at most 100 rows, uses conditional status/version claims, and sleeps with context-aware timers. Reconcile matching `Stat` size/SHA metadata to AVAILABLE; otherwise delete/abort residual object and mark FAILED. Retention claims AVAILABLE as DELETING, deletes object, then marks DELETED; failure appends an event and leaves DELETING retryable.

- [ ] **Step 4: Register worker lifecycle**

Start workers after database and MinIO initialization. Register cancel/wait in the existing plugin shutdown handler alongside engine/Redis shutdown. Disabled ingress starts no file workers.

- [ ] **Step 5: Run tests and commit workers**

Run: `cd server && go test ./plugin/tr069/service ./plugin/tr069`

```bash
git add server/plugin/tr069/service/transfer_worker* server/plugin/tr069/plugin.go
git commit -m "feat(tr069): reconcile and retain log artifacts"
```

### Task 10: Protected Artifact List and Download APIs

**Files:**
- Create: `server/plugin/tr069/api/artifact.go`
- Create: `server/plugin/tr069/api/artifact_test.go`
- Create: `server/plugin/tr069/router/artifact.go`
- Create: `server/plugin/tr069/middleware/download_audit.go`
- Create: `server/plugin/tr069/middleware/download_audit_test.go`
- Modify: `server/plugin/tr069/plugin.go`
- Modify: `server/plugin/tr069/initialize/api.go`

**Interfaces:**
- Produces: `GET /tr069/artifact/list?page&pageSize&deviceId`.
- Produces: `GET /tr069/artifact/:artifactId/download`.

- [ ] **Step 1: Write API and audit tests**

Test pagination defaults/max 100, exact device ID filter, no provider fields/secrets, unavailable/deleted/missing object rejection, Casbin route registration, download filename sanitization, cancellation, and a 20 MiB streamed response that does not accumulate in the operation audit writer.

- [ ] **Step 2: Run focused API tests**

Run: `cd server && go test ./plugin/tr069/api ./plugin/tr069/middleware ./plugin/tr069/router -run 'Artifact|DownloadAudit'`

Expected: FAIL.

- [ ] **Step 3: Implement list DTO and service query**

Return only artifact ID, task ID, device ID, SerialNumber, OUI, original filename, content type, size, SHA-256, source, status, and receive time. `ObjectKey`, driver configuration, source IP, and credentials remain server-only.

- [ ] **Step 4: Implement backend-streamed download**

Re-read the artifact as AVAILABLE after authorization, open Store, set safe `Content-Disposition`, `Content-Type`, optional Content-Length, and call `io.CopyBuffer(c.Writer, reader, pooledBuffer)`. Never use existing `OperationRecord`, because it buffers response bytes; use the metadata-only audit middleware to persist artifact/device/user/status/latency.

- [ ] **Step 5: Register JWT/Casbin routes and plugin API metadata**

```go
artifact := r.Group("artifact")
artifact.GET("list", artifactAPI.List)
artifact.GET(":artifactId/download", tr069Middleware.DownloadAudit(), artifactAPI.Download)
```

Add both exact paths/methods to `initialize/api.go`, and initialize `ArtifactRouter` inside `tr069Plugin.Register` after the shared JWT/Casbin group.

- [ ] **Step 6: Run tests and commit APIs**

Run: `cd server && go test ./plugin/tr069/api ./plugin/tr069/middleware ./plugin/tr069/router ./plugin/tr069`

```bash
git add server/plugin/tr069/api/artifact* server/plugin/tr069/router/artifact* server/plugin/tr069/middleware/download_audit* server/plugin/tr069/plugin.go server/plugin/tr069/initialize/api.go
git commit -m "feat(tr069): expose protected log artifact APIs"
```

### Task 11: GVA “日志文件” Menu and Page

**Files:**
- Create: `web/src/plugin/tr069/api/log-file.js`
- Create: `web/src/plugin/tr069/view/log-file/log-file-view.js`
- Create: `web/src/plugin/tr069/view/log-file/log-file-view.test.js`
- Create: `web/src/plugin/tr069/view/log-file/log-file.contract.test.js`
- Create: `web/src/plugin/tr069/view/log-file/index.vue`
- Modify: `server/plugin/tr069/initialize/menu.go`
- Create: `server/plugin/tr069/initialize/menu_log_file_test.go`
- Modify generated: `web/src/pathInfo.json`

**Interfaces:**
- Consumes: artifact list/download APIs from Task 10.
- Produces: route name `tr069LogFiles`, path `logFiles`, component `plugin/tr069/view/log-file/index.vue`.

- [ ] **Step 1: Write frontend helper and contract tests first**

```js
test('log page keeps GVA theme and device ID filter', async () => {
  const source = await readFile(new URL('./index.vue', import.meta.url), 'utf8')
  assert.match(source, /class="gva-search-box"/)
  assert.match(source, /class="gva-table-box"/)
  assert.match(source, /deviceId/)
  assert.doesNotMatch(source, /background(?:-color)?:\s*#fff/i)
})
```

Also test `formatBytes`, `shortSHA256`, source/status labels, unavailable download disabling, and URL revocation after blob download.

- [ ] **Step 2: Run tests and verify files/menu are absent**

Run: `cd web && node --test src/plugin/tr069/view/log-file/*.test.js`

Expected: FAIL because files do not exist.

- [ ] **Step 3: Add frontend API wrappers**

```js
export const getLogArtifactList = params => service({ url: '/tr069/artifact/list', method: 'get', params })
export const downloadLogArtifact = artifactId => service({ url: `/tr069/artifact/${artifactId}/download`, method: 'get', responseType: 'blob', donNotShowLoading: true })
```

- [ ] **Step 4: Build the GVA-styled page**

Use a numeric Device ID input, 查询/重置 buttons, paginated `el-table`, receive time, device identity, filename, size, shortened SHA with tooltip, ACTIVE/PERIODIC tag, status tag, and a download link enabled only for AVAILABLE. On download, derive filename from Content-Disposition when available, create/revoke an object URL, and show an Element Plus error on failure.

- [ ] **Step 5: Register and stabilize the menu**

Create `tr069LogFiles` with sort 3 and move the alarm parent to sort 4. Follow the existing FirstOrCreate plus explicit Save pattern so upgrades repair stale menu fields.

- [ ] **Step 6: Run Vite to regenerate pathInfo and verify tests/build**

Run:

```bash
cd web
node --test src/plugin/tr069/view/log-file/*.test.js src/plugin/tr069/view/device/*.contract.test.js
npm run build
```

Expected: tests PASS, production build PASS, and `src/pathInfo.json` contains `/src/plugin/tr069/view/log-file/index.vue`.

- [ ] **Step 7: Commit the management UI**

```bash
git add web/src/plugin/tr069/api/log-file.js web/src/plugin/tr069/view/log-file web/src/pathInfo.json server/plugin/tr069/initialize/menu.go server/plugin/tr069/initialize/menu_log_file_test.go
git commit -m "feat(tr069): add log artifact management page"
```

### Task 12: Runtime Configuration, MinIO Compose, and Final Verification

**Files:**
- Modify: `server/config.yaml`
- Modify: `server/config.local.yaml`
- Modify: `server/config.docker.yaml`
- Modify: `deploy/docker-compose/docker-compose.yaml`
- Modify: `openspec/changes/add-tr069-log-collection/tasks.md`
- Create: `server/plugin/tr069/docs/log_collection.md`

**Interfaces:**
- Produces: reproducible local MinIO service and documented BS/GVA configuration.

- [ ] **Step 1: Add safe disabled defaults to all configuration variants**

Add the complete `tr069.fileIngress` structure with `enabled: false`; do not commit real shared credentials. Local examples use `http://host.docker.internal:7458`, 64 MiB, 30m identity TTL, 10m upload timeout, 30-day retention, and MinIO driver fields.

- [ ] **Step 2: Add MinIO to project Compose without automatic restart**

Use service name `gva-minio`, internal endpoint `minio:9000`, host ports `19000:9000` and `19001:9001`, a named volume, health check, and `restart: "no"` to respect the development rule that project containers do not auto-start after Windows/Docker restart. Add the server dependency only when Compose syntax permits healthy optional startup; do not change 18080/18888/7458.

- [ ] **Step 3: Document device and server setup**

Document shared Basic/Digest credentials, user-managed `Device.LogMgmt.URL/Username/Password`, PUT/POST raw-body compatibility, Inform-before-upload requirement, 403 ambiguity behavior, HTTPS production requirement, MinIO bucket, retention/reconciliation, and the NAT limitation.

- [ ] **Step 4: Run all backend unit and race-focused tests**

Run:

```bash
cd server
go test ./plugin/tr069/...
go test -race ./plugin/tr069/service ./plugin/tr069/handler ./plugin/tr069/middleware ./plugin/tr069/adapter
go vet ./plugin/tr069/...
```

Expected: all PASS with no race or vet findings.

- [ ] **Step 5: Run all frontend contract tests and production build**

Run:

```bash
cd web
node --test src/plugin/tr069/**/*.test.js
npm run build
```

Expected: PASS.

- [ ] **Step 6: Run an HTTP/MinIO integration smoke test**

Start MySQL, Redis, MinIO, backend, then submit a valid Inform fixture followed by one Basic PUT and one Digest POST containing deterministic test bytes. Verify both return 204, only one device owns both artifacts, list filtering by that numeric device ID returns them, downloads reproduce identical SHA-256, invalid auth returns 401, unknown IP returns 403, and DELETE returns 405 with `Allow: PUT, POST`.

- [ ] **Step 7: Mark OpenSpec tasks only after evidence exists and commit operations/docs**

Update checkboxes that have passing evidence; leave real-BS-only checks open until exercised. Then:

```bash
git add server/config.yaml server/config.local.yaml server/config.docker.yaml deploy/docker-compose/docker-compose.yaml server/plugin/tr069/docs/log_collection.md openspec/changes/add-tr069-log-collection/tasks.md
git commit -m "docs(tr069): document log collection operations"
```

- [ ] **Step 8: Final working-tree and diff review**

Run:

```bash
git diff --check
git status --short
git log --oneline --decorate -12
```

Expected: no whitespace errors, only intentional uncommitted runtime artifacts if any, and small reviewable commits matching Tasks 1-12.
