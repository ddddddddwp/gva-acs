# TR-069 Connection Profile Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 自动发现并安全保存每台 CPE 的 Connection Request Profile，自动配置独立凭据，并使用 HTTP Digest 唤醒设备后继续现有 RPC FIFO。

**Architecture:** 在 GVA TR-069 adapter 层增加 Profile repository、credential cipher、payload protector/hydrator、provisioner 和 Digest client。数据库保存 Profile 与脱敏命令，真实密码只在提交事务中加密、在构造 CWMP 前解密到内存；现有 CommandManager、RedisCommandSource、GormCommandRepo 和 DataModelHook 通过小接口接入。

**Tech Stack:** Go 1.24、GORM、SQLite/MySQL、AES-256-GCM、net/http、TR-069/CWMP、现有 Redis FIFO、Go testing。

## Global Constraints

- `server/plugin/tr069/lib/tr069-core-only` 仍由独立仓库维护，GVA 主仓库不提交该目录。
- 数据库是 Profile 和命令状态的事实来源；Redis 只承担设备队列、锁和唤醒。
- Inform/GPV 复用已解析参数，不进行第二次 XML 解析。
- 实际发送 XML 保持原文；日志、命令参数、API 和持久化 XML 中的 `Device.ManagementServer.ConnectionRequestPassword` 必须显示为 `******` 或内部占位符。
- 自动凭据配置必须复用现有设备 FIFO，并在 RPC 记录中标记来源 `SYSTEM`。
- Connection Request 必须支持 HTTP Digest 和当前 BS 的 HTTP URL；禁止自动重定向并复用 HTTP Transport。
- 密钥不得提交到 Git；仓库模板保持空值，本地密钥仅写入已忽略的 `server/config.local.yaml`。
- 普通命令等待与响应超时继续使用现有独立配置，不把 Connection Request HTTP 超时与其合并。

---

## File Structure

### New files

- `server/plugin/tr069/model/connection_profile.go`: Profile 数据模型、状态和来源常量。
- `server/plugin/tr069/adapter/connection_profile_crypto.go`: AES-GCM keyring 与凭据加解密。
- `server/plugin/tr069/adapter/connection_profile_repository.go`: 单行 Profile 查询、采集、人工/自动凭据写入和状态条件更新。
- `server/plugin/tr069/adapter/connection_profile_payload.go`: SPV 敏感参数保护和队列出队时内存补密。
- `server/plugin/tr069/adapter/connection_profile_provisioner.go`: 自动生成凭据并创建系统 SPV。
- `server/plugin/tr069/adapter/command_xml_sink.go`: 将 core `wire.xml` 事件关联到命令并交给安全存储。
- `server/plugin/tr069/redact/connection_request.go`: XML 与参数 JSON 的纯函数脱敏器。
- 对应的 `_test.go` 文件：每个组件独立测试。

### Modified files

- `server/plugin/tr069/config/config.go`, `runtime.go`, `runtime_test.go`: Connection Request 配置及热加载快照。
- `server/plugin/tr069/initialize/viper.go`, `gorm.go`: 安全日志、迁移和旧 URL 回填。
- `server/config.yaml`: 可提交的非秘密配置模板。
- `server/plugin/tr069/model/command.go`: 增加命令来源字段。
- `server/plugin/tr069/service/command_manager.go`: 支持保护器和系统命令事务钩子。
- `server/plugin/tr069/service/command_store.go`: 保存 XML 前 fail-closed 脱敏。
- `server/plugin/tr069/adapter/gorm_repo.go`: Inform 不清空 URL、采集 Profile、终态同步。
- `server/plugin/tr069/adapter/datamodel_hook.go`: GPV 后采集 Profile。
- `server/plugin/tr069/adapter/redis_command_source.go`: 出队后只在内存中恢复敏感参数。
- `server/plugin/tr069/adapter/connection_request.go`: Profile resolver、URL 策略、共享 Client 与 Digest。
- `server/plugin/tr069/adapter/command_wakeup_consumer.go`: 使用运行时 Connection Request 超时。
- `server/plugin/tr069/api/command.go`: API CommandManager 注入 payload protector。
- `server/plugin/tr069/api/device.go`, `router/device.go`: Profile 查询与人工覆盖 API，复用 GVA 权限链。
- `server/plugin/tr069/model/request/connection_profile.go`, `model/response/connection_profile.go`: 不暴露秘密的类型化请求/响应。
- `server/plugin/tr069/engine/engine.go`: 默认 repo/source/hook 注入 Profile 依赖。
- `server/plugin/tr069/middleware/raw_dump.go`, `raw_response_dump.go`: 记录前脱敏 XML 副本。

---

### Task 1: Profile model, configuration, migration, and backfill

**Files:**
- Create: `server/plugin/tr069/model/connection_profile.go`
- Modify: `server/plugin/tr069/config/config.go`
- Modify: `server/plugin/tr069/config/runtime.go`
- Modify: `server/plugin/tr069/config/runtime_test.go`
- Modify: `server/plugin/tr069/initialize/gorm.go`
- Modify: `server/plugin/tr069/initialize/viper.go`
- Modify: `server/config.yaml`
- Test: `server/plugin/tr069/initialize/connection_profile_migration_test.go`

**Interfaces:**
- Produces: `model.ConnectionProfile`, `model.ConnectionProfileState*`, `config.ConnectionRequestConfig`, `Runtime.ConnectionRequestTimeout`.
- Consumes: existing `config.StoreRuntime`, plugin `initialize.Gorm`, `model.Device.ConnectionReqURL`.

- [ ] **Step 1: Write failing runtime and migration tests**

Add tests that assert defaults, explicit values, key material preservation without logging, unique `device_id`, and URL backfill:

```go
func TestNormalizeRuntimeConfigConnectionRequestDefaults(t *testing.T) {
	got := NormalizeRuntimeConfig(TR069Config{})
	if got.ConnectionRequest.RequestTimeout != 10 {
		t.Fatalf("request timeout = %d, want 10", got.ConnectionRequest.RequestTimeout)
	}
	if got.ConnectionRequest.AuthScheme != "digest" {
		t.Fatalf("auth scheme = %q, want digest", got.ConnectionRequest.AuthScheme)
	}
}

func TestConnectionProfileMigrationBackfillsDeviceURL(t *testing.T) {
	db := newMigrationTestDB(t)
	device := model.Device{SerialNumber: "PROFILE-BACKFILL", ConnectionReqURL: "http://127.0.0.1:8400"}
	if err := db.Create(&device).Error; err != nil { t.Fatal(err) }
	if err := migrateConnectionProfiles(context.Background(), db); err != nil { t.Fatal(err) }
	var profile model.ConnectionProfile
	if err := db.First(&profile, "device_id = ?", device.ID).Error; err != nil { t.Fatal(err) }
	if profile.DiscoveredURL != device.ConnectionReqURL { t.Fatalf("url = %q", profile.DiscoveredURL) }
}
```

- [ ] **Step 2: Run tests and verify RED**

Run:

```bash
cd server && go test ./plugin/tr069/config ./plugin/tr069/initialize -run 'ConnectionRequest|ConnectionProfileMigration' -count=1
```

Expected: build failure because the nested config, Profile model, and migration function do not exist.

- [ ] **Step 3: Implement model and runtime config**

Create the Profile with no JSON exposure for ciphertext:

```go
type ConnectionProfile struct {
	ID                   uint       `json:"id" gorm:"primaryKey"`
	DeviceID             uint       `json:"deviceId" gorm:"uniqueIndex;not null"`
	DiscoveredURL        string     `json:"discoveredUrl" gorm:"size:2048"`
	OverrideURL          string     `json:"overrideUrl" gorm:"size:2048"`
	Username             string     `json:"username" gorm:"size:256"`
	PasswordCiphertext   []byte     `json:"-" gorm:"type:longblob"`
	CredentialKeyVersion string     `json:"-" gorm:"size:32"`
	CredentialSource     string     `json:"credentialSource" gorm:"size:16"`
	AuthScheme           string     `json:"authScheme" gorm:"size:16"`
	ProvisionState       string     `json:"provisionState" gorm:"size:24;index"`
	ProvisionCommandID   string     `json:"provisionCommandId" gorm:"size:64;index"`
	LastError            string     `json:"lastError" gorm:"type:text"`
	LastWakeAt           *time.Time `json:"lastWakeAt"`
	LastWakeStatus       string     `json:"lastWakeStatus" gorm:"size:32"`
	CreatedAt            time.Time  `json:"createdAt"`
	UpdatedAt            time.Time  `json:"updatedAt"`
}

func (ConnectionProfile) TableName() string { return "tr069_connection_request_profiles" }
```

Add `ConnectionRequest ConnectionRequestConfig` to `TR069Config`:

```go
type ConnectionRequestConfig struct {
	AutoProvisionCredentials bool              `mapstructure:"autoProvisionCredentials" yaml:"autoProvisionCredentials"`
	CredentialKeyVersion     string            `mapstructure:"credentialKeyVersion" yaml:"credentialKeyVersion"`
	CredentialEncryptionKey  string            `mapstructure:"credentialEncryptionKey" yaml:"credentialEncryptionKey"`
	CredentialDecryptionKeys map[string]string `mapstructure:"credentialDecryptionKeys" yaml:"credentialDecryptionKeys"`
	RequestTimeout           int               `mapstructure:"requestTimeout" yaml:"requestTimeout"`
	AllowedCIDRs             []string          `mapstructure:"allowedCIDRs" yaml:"allowedCIDRs"`
	AuthScheme               string            `mapstructure:"authScheme" yaml:"authScheme"`
}
```

Normalize `RequestTimeout=10`, `AuthScheme="digest"`, and publish `time.Duration` in `Runtime`. Do not log encryption key fields in `logRuntimeConfig`.

- [ ] **Step 4: Implement migration and committed template**

Add `ConnectionProfile` to plugin AutoMigrate, then backfill without overwriting existing profiles:

```go
func migrateConnectionProfiles(ctx context.Context, db *gorm.DB) error {
	var devices []model.Device
	if err := db.WithContext(ctx).Select("id", "connection_req_url").
		Where("connection_req_url <> ?", "").Find(&devices).Error; err != nil {
		return err
	}
	now := time.Now()
	profiles := make([]model.ConnectionProfile, 0, len(devices))
	for _, device := range devices {
		profiles = append(profiles, model.ConnectionProfile{
			DeviceID: device.ID, DiscoveredURL: device.ConnectionReqURL,
			AuthScheme: "digest", ProvisionState: model.ConnectionProfileStateDiscovered,
			CreatedAt: now, UpdatedAt: now,
		})
	}
	if len(profiles) == 0 { return nil }
	return db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&profiles).Error
}
```

This GORM path works for SQLite and MySQL without dialect-specific SQL. Add this non-secret template:

```yaml
connectionRequest:
    autoProvisionCredentials: true
    credentialKeyVersion: v1
    credentialEncryptionKey: ""
    credentialDecryptionKeys: {}
    requestTimeout: 10
    authScheme: digest
    allowedCIDRs:
        - 127.0.0.0/8
        - 172.16.0.0/12
```

- [ ] **Step 5: Run tests and commit**

Run:

```bash
cd server && go test ./plugin/tr069/config ./plugin/tr069/initialize -run 'ConnectionRequest|ConnectionProfileMigration' -count=1
```

Expected: PASS.

Commit:

```bash
git add server/plugin/tr069/model/connection_profile.go server/plugin/tr069/config server/plugin/tr069/initialize server/config.yaml
git commit -m "feat(tr069): add connection profile model and config"
```

### Task 2: Credential cipher, repository, and Inform/GPV collection

**Files:**
- Create: `server/plugin/tr069/adapter/connection_profile_crypto.go`
- Create: `server/plugin/tr069/adapter/connection_profile_crypto_test.go`
- Create: `server/plugin/tr069/adapter/connection_profile_repository.go`
- Create: `server/plugin/tr069/adapter/connection_profile_repository_test.go`
- Modify: `server/plugin/tr069/adapter/gorm_repo.go`
- Modify: `server/plugin/tr069/adapter/gorm_repo_test.go`
- Modify: `server/plugin/tr069/adapter/datamodel_hook.go`
- Modify: `server/plugin/tr069/adapter/datamodel_hook_test.go`
- Create: `server/plugin/tr069/model/request/connection_profile.go`
- Create: `server/plugin/tr069/model/response/connection_profile.go`
- Modify: `server/plugin/tr069/api/device.go`
- Modify: `server/plugin/tr069/api/device_test.go`
- Modify: `server/plugin/tr069/router/device.go`
- Modify: `server/plugin/tr069/initialize/api.go`

**Interfaces:**
- Produces: `CredentialCipher.Encrypt/Decrypt`, `ConnectionProfileRepository.Collect`, `Resolve`, `StoreCredential`, `MarkProvisioning`, `MarkTerminal`.
- Consumes: Task 1 Profile model and runtime key configuration.

- [ ] **Step 1: Write failing crypto and collector tests**

Cover unique nonces, wrong key, empty password readback, unchanged URL, manual override precedence, Inform without URL, and GPV collection:

```go
func TestCredentialCipherRoundTripUsesUniqueNonce(t *testing.T) {
	cipher := newTestCredentialCipher(t)
	a, err := cipher.Encrypt("secret")
	if err != nil { t.Fatal(err) }
	b, err := cipher.Encrypt("secret")
	if err != nil { t.Fatal(err) }
	if bytes.Equal(a.Ciphertext, b.Ciphertext) { t.Fatal("ciphertexts must differ") }
	plain, err := cipher.Decrypt(a.Version, a.Ciphertext)
	if err != nil || plain != "secret" { t.Fatalf("decrypt = %q, %v", plain, err) }
}

func TestCollectDoesNotClearURLOrPasswordOnAbsentValues(t *testing.T) {
	repo, db := newProfileRepositoryTest(t)
	seedReadyProfile(t, db, 7, "http://127.0.0.1:8400")
	if _, err := repo.Collect(context.Background(), 7, map[string]string{}); err != nil { t.Fatal(err) }
	profile := loadProfile(t, db, 7)
	if profile.DiscoveredURL != "http://127.0.0.1:8400" || len(profile.PasswordCiphertext) == 0 {
		t.Fatalf("profile cleared: %#v", profile)
	}
}
```

- [ ] **Step 2: Run tests and verify RED**

Run:

```bash
cd server && go test ./plugin/tr069/adapter -run 'CredentialCipher|ConnectionProfile|InformDoesNotClear|GPVCollects' -count=1
```

Expected: build failure for missing cipher/repository.

- [ ] **Step 3: Implement AES-GCM keyring**

Expose a narrow interface:

```go
type EncryptedCredential struct {
	Version    string
	Ciphertext []byte
}

type CredentialCipher interface {
	Encrypt(plaintext string) (EncryptedCredential, error)
	Decrypt(version string, ciphertext []byte) (string, error)
}
```

Decode base64 32-byte keys, use a fresh `crypto/rand` nonce for every encryption, prepend the nonce to the ciphertext, and return explicit errors for missing versions or authentication failures. Never include plaintext, key bytes, or ciphertext in errors.

- [ ] **Step 4: Implement repository collection and conditional writes**

Use exact parameter constants and return whether provisioning is needed:

```go
const (
	connectionRequestURLName      = "Device.ManagementServer.ConnectionRequestURL"
	connectionRequestUsernameName = "Device.ManagementServer.ConnectionRequestUsername"
	connectionRequestPasswordName = "Device.ManagementServer.ConnectionRequestPassword"
)

type CollectResult struct {
	Profile          model.ConnectionProfile
	NeedsProvisioning bool
}

func (r *ConnectionProfileRepository) Collect(ctx context.Context, deviceID uint, values map[string]string) (CollectResult, error)
func (r *ConnectionProfileRepository) Resolve(ctx context.Context, deviceID uint) (ResolvedConnectionProfile, error)
func (r *ConnectionProfileRepository) StoreCredential(ctx context.Context, tx *gorm.DB, deviceID uint, username, password, source string) error
func (r *ConnectionProfileRepository) MarkProvisioning(ctx context.Context, tx *gorm.DB, deviceID uint, commandID string) error
func (r *ConnectionProfileRepository) MarkTerminal(ctx context.Context, tx *gorm.DB, commandID string, success bool, message string) error
```

Normalize URL whitespace, only update changed non-empty values, never treat an empty password readback as a new credential, and never overwrite `MANUAL` credentials from automatic collection.

- [ ] **Step 5: Integrate Inform and GPV without a second parse**

In `UpsertFromInform`, build the OnConflict update columns dynamically:

```go
updates := []string{"oui", "product_class", "manufacturer", "software_ver", "hardware_ver", "spec_ver", "ip", "last_inform"}
if device.ConnectionReqURL != "" {
	updates = append(updates, "connection_req_url")
}
```

After resolving the numeric device ID, pass the already available `info.Params` to `Collect`. In `persistGPV`, build a three-entry `map[string]string` while iterating `params`, persist the batch, then call `Collect`; do not reparse `resp` or stored JSON.

- [ ] **Step 6: Add typed query/manual-override APIs through GVA routing**

Define a write-only password field and a response that never contains ciphertext or plaintext:

```go
type ConnectionProfileOverrideRequest struct {
	OverrideURL   string `json:"overrideUrl"`
	Username      string `json:"username"`
	Password      string `json:"password"`
	ClearOverride bool   `json:"clearOverride"`
}

type ConnectionProfileResponse struct {
	DeviceID          uint       `json:"deviceId"`
	EffectiveURL      string     `json:"effectiveUrl"`
	DiscoveredURL     string     `json:"discoveredUrl"`
	OverrideURL       string     `json:"overrideUrl"`
	Username          string     `json:"username"`
	CredentialSource  string     `json:"credentialSource"`
	ProvisionState    string     `json:"provisionState"`
	ProvisionCommandID string    `json:"provisionCommandId"`
	LastError         string     `json:"lastError"`
	LastWakeAt        *time.Time `json:"lastWakeAt"`
	LastWakeStatus    string     `json:"lastWakeStatus"`
}
```

Register `GET /tr069/device/:id/connection-profile` and `PUT /tr069/device/:id/connection-profile` beneath the existing authenticated router group so GVA API/Casbin initialization discovers them. `PUT` validates URL/CIDR, requires username and password together when setting manual credentials, encrypts in one transaction, and supports explicit clearing. Add API tests proving unauthenticated/unprivileged requests are rejected by the existing route chain and responses omit `password`, `ciphertext`, and key version.

- [ ] **Step 7: Run tests and commit**

Run:

```bash
cd server && go test ./plugin/tr069/adapter ./plugin/tr069/api -run 'CredentialCipher|ConnectionProfile|InformDoesNotClear|GPVCollects|ConnectionProfileAPI' -count=1
```

Expected: PASS.

Commit:

```bash
git add server/plugin/tr069/adapter/connection_profile_* server/plugin/tr069/adapter/gorm_repo.go server/plugin/tr069/adapter/gorm_repo_test.go server/plugin/tr069/adapter/datamodel_hook.go server/plugin/tr069/adapter/datamodel_hook_test.go server/plugin/tr069/model/request/connection_profile.go server/plugin/tr069/model/response/connection_profile.go server/plugin/tr069/api/device.go server/plugin/tr069/api/device_test.go server/plugin/tr069/router/device.go server/plugin/tr069/initialize/api.go
git commit -m "feat(tr069): collect encrypted connection profiles"
```

### Task 3: Protect persisted command parameters and hydrate only in memory

**Files:**
- Create: `server/plugin/tr069/adapter/connection_profile_payload.go`
- Create: `server/plugin/tr069/adapter/connection_profile_payload_test.go`
- Modify: `server/plugin/tr069/model/command.go`
- Modify: `server/plugin/tr069/service/command_manager.go`
- Modify: `server/plugin/tr069/service/command_manager_test.go`
- Modify: `server/plugin/tr069/adapter/redis_command_source.go`
- Modify: `server/plugin/tr069/adapter/redis_command_source_test.go`
- Modify: `server/plugin/tr069/api/command.go`

**Interfaces:**
- Produces: `service.CommandPayloadProtector`, `service.WithCommandPayloadProtector`, `CommandManager.SubmitSystem`, `adapter.ConnectionProfilePayloadProtector.Protect/Hydrate`.
- Consumes: Task 2 repository/cipher and existing `service.EncodeRPCRequest`, `DecodeRPCParams`.

- [ ] **Step 1: Write failing persistence and hydration tests**

Assert that a submitted SPV never stores plaintext but a pulled `core.Command` contains the real value:

```go
func TestConnectionRequestPasswordIsProtectedAtRestAndHydratedForBuild(t *testing.T) {
	request := req.SetParameterValuesRequest{Parameters: []req.SetParameterValue{{
		Name: connectionRequestPasswordName, Type: "xsd:string", Value: "device-secret",
	}}}
	result := submitProtectedCommand(t, request)
	stored := loadCommand(t, result.CommandID)
	if strings.Contains(string(stored.ParamsJSON), "device-secret") { t.Fatal("plaintext persisted") }
	if !strings.Contains(string(stored.ParamsJSON), connectionRequestPasswordPlaceholder) { t.Fatal("placeholder missing") }
	pulled := pullCommand(t, stored.DeviceKey)
	if got := pulled.Params["parameters"].([]any)[0].(map[string]any)["value"]; got != "device-secret" {
		t.Fatalf("hydrated value = %#v", got)
	}
}
```

- [ ] **Step 2: Run tests and verify RED**

Run:

```bash
cd server && go test ./plugin/tr069/service ./plugin/tr069/adapter -run 'ProtectedAtRest|HydratedForBuild|SubmitSystemOrigin' -count=1
```

Expected: failing assertions because plaintext is currently stored and no origin exists.

- [ ] **Step 3: Add command origin and protector transaction interface**

Add `Origin string` to `model.Command`, default user submissions to `USER`, and define:

```go
type CommandPayloadProtector interface {
	Protect(ctx context.Context, tx *gorm.DB, deviceID uint, operation, origin string, encoded []byte) ([]byte, error)
}

type CommandCreatedHook func(ctx context.Context, tx *gorm.DB, command *model.Command) error

func WithCommandPayloadProtector(protector CommandPayloadProtector) CommandManagerOption
func (m *CommandManager) SubmitSystem(ctx context.Context, deviceID uint, operation string, request any, dedupKey string, hook CommandCreatedHook) (SubmitResult, error)
```

`Submit` and `SubmitSystem` must encode and protect inside the existing device-locking DB transaction before `CommandStore.Create`. `SubmitSystem` bypasses only capability discovery for the bootstrap SPV; it still validates the typed request, requires the device row, and requires a recent Inform.

- [ ] **Step 4: Implement exact password placeholder protection**

Use a sentinel that is never a valid generated password:

```go
const connectionRequestPasswordPlaceholder = "__GVA_TR069_CONNECTION_REQUEST_PASSWORD__"
```

For `SetParameterValues`, decode the typed request, find only the exact password parameter, call `StoreCredential` in the same transaction, replace only its `Value`, and re-encode. `Hydrate` performs the inverse after `DecodeRPCParams` and before creating `core.Command`; it decrypts from Profile and mutates only the in-memory map. If decryption fails, NACK the BUILDING command without sending a placeholder.

- [ ] **Step 5: Wire API and Redis source, run tests, and commit**

Construct the API manager with `WithCommandPayloadProtector(adapter.NewConnectionProfilePayloadProtector(nil))`. Give `RedisCommandSource` a hydrator option and install the same adapter implementation by default.

Run:

```bash
cd server && go test ./plugin/tr069/service ./plugin/tr069/adapter -run 'ProtectedAtRest|HydratedForBuild|SubmitSystemOrigin' -count=1
```

Expected: PASS.

Commit:

```bash
git add server/plugin/tr069/model/command.go server/plugin/tr069/service/command_manager.go server/plugin/tr069/service/command_manager_test.go server/plugin/tr069/adapter/connection_profile_payload.go server/plugin/tr069/adapter/connection_profile_payload_test.go server/plugin/tr069/adapter/redis_command_source.go server/plugin/tr069/adapter/redis_command_source_test.go server/plugin/tr069/api/command.go
git commit -m "feat(tr069): protect connection request command secrets"
```

### Task 4: Automatic per-device credential provisioning and terminal correlation

**Files:**
- Create: `server/plugin/tr069/adapter/connection_profile_provisioner.go`
- Create: `server/plugin/tr069/adapter/connection_profile_provisioner_test.go`
- Modify: `server/plugin/tr069/adapter/gorm_repo.go`
- Modify: `server/plugin/tr069/adapter/gorm_repo_test.go`
- Modify: `server/plugin/tr069/engine/engine.go`

**Interfaces:**
- Produces: `ConnectionCredentialProvisioner.Schedule/Run/Ensure`, system SPV dedup key, terminal Profile state updates.
- Consumes: Tasks 2-3 repository, cipher, `CommandManager.SubmitSystem`, existing `adapter.EnqueueImmediate`.

- [ ] **Step 1: Write failing idempotency and terminal tests**

Cover one in-flight provisioning command per device, successful READY, failed FAILED, disabled auto provisioning, missing key, non-blocking scheduling, and startup recovery:

```go
func TestProvisionerCreatesOneSystemSPVAndMarksReady(t *testing.T) {
	provisioner, db := newProvisionerTest(t)
	first, err := provisioner.Ensure(context.Background(), 9)
	if err != nil { t.Fatal(err) }
	second, err := provisioner.Ensure(context.Background(), 9)
	if err != nil { t.Fatal(err) }
	if first.CommandID == "" || second.CommandID != first.CommandID { t.Fatalf("results = %#v %#v", first, second) }
	command := loadCommand(t, first.CommandID)
	if command.Origin != "SYSTEM" || command.Operation != "SetParameterValues" { t.Fatalf("command = %#v", command) }
	repo := newGormCommandRepo(db)
	if err := repo.MarkSuccess(context.Background(), command.CommandID, time.Now()); err != nil { t.Fatal(err) }
	if got := loadProfile(t, db, 9).ProvisionState; got != model.ConnectionProfileStateReady { t.Fatalf("state = %s", got) }
}
```

- [ ] **Step 2: Run tests and verify RED**

Run:

```bash
cd server && go test ./plugin/tr069/adapter -run 'Provisioner|ProfileTerminal' -count=1
```

Expected: build failure for missing provisioner.

- [ ] **Step 3: Implement provisioning with atomic command correlation**

Generate username and password with `crypto/rand`, construct:

```go
req.SetParameterValuesRequest{
	ParameterKey: "gva-connection-request",
	Parameters: []req.SetParameterValue{
		{Name: connectionRequestUsernameName, Type: "xsd:string", Value: username},
		{Name: connectionRequestPasswordName, Type: "xsd:string", Value: password},
	},
}
```

Use `SubmitSystem` with dedup key `connection-profile:<deviceID>:<credential-key-version>`. Its transaction hook conditionally changes `DISCOVERED/FAILED` to `PROVISIONING` and writes `provision_command_id`. If an existing `PROVISIONING` command remains non-terminal, return its ID instead of creating another.

- [ ] **Step 4: Keep provisioning off the Inform request path**

Expose a bounded worker API:

```go
type ConnectionCredentialProvisioner struct {
	queue chan uint
	manager *service.CommandManager
	repository *ConnectionProfileRepository
}

func (p *ConnectionCredentialProvisioner) Schedule(deviceID uint) bool
func (p *ConnectionCredentialProvisioner) Run(ctx context.Context)
func (p *ConnectionCredentialProvisioner) Ensure(ctx context.Context, deviceID uint) (service.SubmitResult, error)
```

`Schedule` performs a non-blocking send and returns false when full; duplicate safety remains in the database. `Run` first scans `DISCOVERED` profiles, then consumes the channel with a fixed worker count of one. Inform/GPV collection only calls `Schedule`, so HTTP processing never waits on random generation, command creation, Redis, or device wakeup. `FAILED` profiles require explicit retry or new manual credentials and are not looped automatically.

- [ ] **Step 5: Trigger provisioning and correlate terminal callbacks atomically**

After Inform/GPV Profile collection, call `Schedule` only when `NeedsProvisioning` is true. Refactor `GormCommandRepo.MarkSuccess` and `markFailAtStage` to open one outer GORM transaction, perform `CommandStore.Transition` using the transaction, then `MarkTerminal` using the same transaction and `provision_command_id`. Any Profile update error rolls back both records; ordinary SPV commands match no Profile and cannot alter it.

- [ ] **Step 6: Run tests and commit**

Run:

```bash
cd server && go test ./plugin/tr069/adapter -run 'Provisioner|ProfileTerminal' -count=1
```

Expected: PASS.

Commit:

```bash
git add server/plugin/tr069/adapter/connection_profile_provisioner.go server/plugin/tr069/adapter/connection_profile_provisioner_test.go server/plugin/tr069/adapter/gorm_repo.go server/plugin/tr069/adapter/gorm_repo_test.go server/plugin/tr069/engine/engine.go
git commit -m "feat(tr069): auto provision connection request credentials"
```

### Task 5: HTTP Digest Connection Request, URL validation, and shared transport

**Files:**
- Modify: `server/plugin/tr069/adapter/connection_request.go`
- Modify: `server/plugin/tr069/adapter/connection_request_test.go`
- Modify: `server/plugin/tr069/adapter/command_wakeup_consumer.go`
- Modify: `server/plugin/tr069/adapter/command_wakeup_consumer_test.go`

**Interfaces:**
- Produces: Profile-backed `TriggerConnectionRequest`, `digestAuthorization`, URL/CIDR validator, shared `http.Client`.
- Consumes: Task 1 runtime config and Task 2 `Resolve`.

- [ ] **Step 1: Write failing HTTP behavior tests**

Use `httptest.Server` to require a 401 Digest challenge and verify the second request:

```go
func TestDoConnectionRequestUsesDigestAndReusesClient(t *testing.T) {
	var calls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			w.Header().Set("WWW-Authenticate", `Digest realm="cpe", nonce="abc", qop="auth", algorithm=MD5`)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if !strings.HasPrefix(r.Header.Get("Authorization"), "Digest ") { t.Fatalf("authorization = %q", r.Header.Get("Authorization")) }
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	status, err := doConnectionRequest(context.Background(), server.URL, "acs", "secret", time.Second)
	if err != nil || status != http.StatusNoContent || calls != 2 { t.Fatalf("status=%d calls=%d err=%v", status, calls, err) }
}
```

Also test HTTP accepted, redirect rejected, URL userinfo rejected, response body capped, disallowed CIDR rejected, timeout, wrong credentials, and log URL userinfo removal.

- [ ] **Step 2: Run tests and verify RED**

Run:

```bash
cd server && go test ./plugin/tr069/adapter -run 'ConnectionRequest|Digest|CIDR|Redirect' -count=1
```

Expected: Digest test fails because current implementation only sends Basic and rejects HTTP.

- [ ] **Step 3: Implement validator and Digest challenge response**

Support MD5 and SHA-256 with `qop=auth`; reject unsupported algorithms/qop explicitly. Compute:

```text
HA1 = H(username + ":" + realm + ":" + password)
HA2 = H(method + ":" + request-uri)
response = H(HA1 + ":" + nonce + ":" + nc + ":" + cnonce + ":auth:" + HA2)
```

The first unauthenticated 401 response must be closed before sending the authenticated request. Disable redirects with `CheckRedirect`, cap body draining, and share one package-level Transport/Client. The Transport `DialContext` resolves the hostname once, validates the chosen IP against the latest runtime CIDRs, and dials that exact IP while preserving the original Host/TLS server name; this prevents validation followed by DNS rebinding. Never log Authorization or password.

- [ ] **Step 4: Resolve one Profile row and record wake summary**

Replace the device-table plus datamodel-table queries with `ConnectionProfileRepository.Resolve`. Accept `http` and `https`, reject URL userinfo, validate the resolved host against `AllowedCIDRs`, and update only `last_wake_at/status/error` after the attempt.

- [ ] **Step 5: Run tests and commit**

Run:

```bash
cd server && go test ./plugin/tr069/adapter -run 'ConnectionRequest|Digest|CIDR|Redirect|WakeConsumer' -count=1
```

Expected: PASS.

Commit:

```bash
git add server/plugin/tr069/adapter/connection_request.go server/plugin/tr069/adapter/connection_request_test.go server/plugin/tr069/adapter/command_wakeup_consumer.go server/plugin/tr069/adapter/command_wakeup_consumer_test.go
git commit -m "feat(tr069): wake devices with digest connection requests"
```

### Task 6: Selective XML/log redaction and fail-closed persistence

**Files:**
- Create: `server/plugin/tr069/redact/connection_request.go`
- Create: `server/plugin/tr069/redact/connection_request_test.go`
- Create: `server/plugin/tr069/adapter/command_xml_sink.go`
- Create: `server/plugin/tr069/adapter/command_xml_sink_test.go`
- Modify: `server/plugin/tr069/service/command_store.go`
- Modify: `server/plugin/tr069/service/command_store_test.go`
- Modify: `server/plugin/tr069/middleware/raw_dump.go`
- Modify: `server/plugin/tr069/middleware/raw_response_dump.go`
- Modify: `server/plugin/tr069/middleware/raw_runtime_test.go`
- Modify: `server/plugin/tr069/middleware/raw_response_dump_test.go`
- Modify: `server/plugin/tr069/model/response/command_record.go`
- Modify: `server/plugin/tr069/api/command_record_test.go`
- Modify: `server/plugin/tr069/engine/engine.go`
- Modify: `server/plugin/tr069/initialize/gorm.go`
- Modify: `server/plugin/tr069/initialize/connection_profile_migration_test.go`

**Interfaces:**
- Produces: `redact.CWMPXML([]byte) ([]byte,error)`, `redact.CommandJSON([]byte) ([]byte,error)`, `adapter.CommandXMLSink`.
- Consumes: persisted placeholder from Task 3 and existing raw request/response logging.

- [ ] **Step 1: Write failing exact-match and fail-closed tests**

Test namespace variants and ensure unrelated Password fields remain unchanged:

```go
func TestCWMPXMLRedactsOnlyConnectionRequestPassword(t *testing.T) {
	in := []byte(`<Envelope><ParameterValueStruct><Name>Device.ManagementServer.ConnectionRequestPassword</Name><Value xsi:type="xsd:string">secret</Value></ParameterValueStruct><Password>download-secret</Password></Envelope>`)
	out, err := CWMPXML(in)
	if err != nil { t.Fatal(err) }
	if bytes.Contains(out, []byte(">secret<")) { t.Fatal("connection password leaked") }
	if !bytes.Contains(out, []byte("<Password>download-secret</Password>")) { t.Fatal("unrelated password changed") }
}
```

- [ ] **Step 2: Run tests and verify RED**

Run:

```bash
cd server && go test ./plugin/tr069/redact ./plugin/tr069/service ./plugin/tr069/middleware -run 'Redact|Sanitize|SaveXML' -count=1
```

Expected: missing package and plaintext persistence failures.

- [ ] **Step 3: Implement streaming XML copy sanitizer**

Use `encoding/xml.Decoder` and `Encoder`, track the current `ParameterValueStruct` Name, and replace only the paired Value character data with `******`. On malformed XML return an error and no payload. Do not use regex or global string replacement.

- [ ] **Step 4: Apply sanitizer only to copies**

- `CommandStore.SaveXML`: sanitize a cloned byte slice; on error do not create a row.
- Raw request/response middleware: sanitize the body copy before formatting and writing to stdout/infolog; the handler still receives/sends the original bytes.
- Command record response: ensure sentinel and ciphertext never escape even if legacy rows exist.
- Run a one-time targeted cleanup during migration for already stored XML rows containing the exact parameter name; malformed matching rows are deleted rather than returned.

- [ ] **Step 5: Connect core wire events to command XML storage**

Implement the existing core extension point without changing the core repository:

```go
type CommandXMLSink struct {
	store *service.CommandStore
	retention func() time.Duration
}

func (s *CommandXMLSink) Enabled(level observability.Level) bool {
	return level == observability.LevelInfo
}

func (s *CommandXMLSink) Emit(ctx context.Context, event observability.Event) {
	if event.Stage != "wire.xml" || event.CommandID == "" || len(event.Payload) == 0 { return }
	_ = s.store.SaveXML(ctx, &model.CommandXML{
		CommandID: event.CommandID, Direction: string(event.Direction), Method: event.Method,
		CWMPID: event.CWMPID, RequestID: event.RequestID, Payload: append([]byte(nil), event.Payload...),
		ExpiresAt: time.Now().Add(s.retention()), CreatedAt: time.Now(),
	})
}
```

Create parser and builder with `tr069.WithEventSink(commandXMLSink)` in `engine.New`. The sink persists through `CommandStore.SaveXML`, so sanitization happens before the database and the original event payload remains untouched. Sink errors are logged without payload or secrets.

- [ ] **Step 6: Run tests and commit**

Run:

```bash
cd server && go test ./plugin/tr069/redact ./plugin/tr069/service ./plugin/tr069/middleware ./plugin/tr069/adapter ./plugin/tr069/api ./plugin/tr069/engine -run 'Redact|Sanitize|SaveXML|CommandXMLSink|CommandRecord' -count=1
```

Expected: PASS.

Commit:

```bash
git add server/plugin/tr069/redact server/plugin/tr069/adapter/command_xml_sink.go server/plugin/tr069/adapter/command_xml_sink_test.go server/plugin/tr069/service/command_store.go server/plugin/tr069/service/command_store_test.go server/plugin/tr069/middleware server/plugin/tr069/model/response/command_record.go server/plugin/tr069/api/command_record_test.go server/plugin/tr069/engine/engine.go server/plugin/tr069/initialize/gorm.go server/plugin/tr069/initialize/connection_profile_migration_test.go
git commit -m "fix(tr069): redact connection request secrets from records"
```

### Task 7: End-to-end wiring, local secret, regression suite, and BS smoke test

**Files:**
- Modify: `server/plugin/tr069/engine/engine.go`
- Modify: `server/plugin/tr069/initialize/server_runtime_test.go`
- Local-only modify: `server/config.local.yaml`
- Test: relevant packages and live BS/GVA logs.

**Interfaces:**
- Consumes: all previous task interfaces.
- Produces: running plugin with collector, provisioning, hydration, Digest wakeups, and redaction enabled.

- [ ] **Step 1: Add an engine integration test**

Build an engine with SQLite repositories and fake wake endpoint, submit a system credential SPV, pull it into a CWMP session, mark its response successful, and assert Profile `READY` plus no plaintext in any persisted table.

- [ ] **Step 2: Run the focused integration test and verify RED if wiring is incomplete**

Run:

```bash
cd server && go test ./plugin/tr069/engine ./plugin/tr069/initialize -run 'ConnectionProfile|Runtime' -count=1
```

Expected before final wiring: failure identifying the missing default dependency; after wiring: PASS.

- [ ] **Step 3: Complete default dependency wiring**

Install one repository/cipher instance per engine, pass the payload hydrator into `RedisCommandSource`, use the Profile-aware `GormDeviceRepo` and `GormCommandRepo`, and ensure config reload affects new Connection Requests without mutating in-flight command deadlines.

- [ ] **Step 4: Generate and install a local-only key**

Generate 32 random bytes:

```bash
openssl rand -base64 32
```

Insert the result into ignored `server/config.local.yaml` under `tr069.connectionRequest.credentialEncryptionKey`; set `autoProvisionCredentials: true`. Verify `git status --short` does not list the local file and never print the key again.

- [ ] **Step 5: Run full regression suites**

Run:

```bash
cd server && go test ./plugin/tr069/... -count=1
cd web && node --test src/plugin/tr069/**/*.test.js
cd web && npm run build
git diff --check
```

Expected: all Go tests, Node contract tests, and production build PASS; `git diff --check` has no output.

- [ ] **Step 6: Run BS smoke test**

Start GVA and BS using the project `oamstart` workflow. Verify:

```bash
docker logs --since 10m gva-acs-bs
tail -n 200 /root/code/gva-acs/bs-runtime/logs/oamProcess.log
```

Acceptance evidence:

- Inform stores the reported Connection Request URL.
- one `SYSTEM` SetParameterValues configures Username and Password;
- persisted command/XML/log output contains no plaintext password;
- after the CWMP session ends, a safe query RPC triggers a Digest Connection Request and later completes;
- RPC timeline distinguishes `WAKE_TRIGGERED`, `REQUEST_SENT`, and `RESPONSE_COMPLETED`.

- [ ] **Step 7: Commit final wiring**

```bash
git add server/plugin/tr069/engine server/plugin/tr069/initialize
git commit -m "test(tr069): verify connection profile onboarding"
```
