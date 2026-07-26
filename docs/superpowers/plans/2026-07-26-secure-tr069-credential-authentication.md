---
change: secure-tr069-credential-authentication
design-doc: docs/superpowers/specs/2026-07-26-secure-tr069-credential-authentication-design.md
base-ref: bb89d5f2f98f861d62f9720b5468e5e9380b6104
---

# TR-069 安全凭据认证实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 用四套用户维护的全局凭据和五个独立认证通道替换 GVA 自动生成/下发设备凭据的行为，并仅通过真实 CWMP、Connection Request 和文件交互验证凭据。

**Architecture:** `server/plugin/tr069/auth` 提供不依赖 Gin/GORM 的凭据解析、Basic/Digest 与 Principal 契约；GORM adapter 负责 AES-GCM 密文、revision、状态与审计，CWMP 和文件入口只保留薄中间件。CWMP 认证在读取 SOAP body 前完成，Connection Request 通过同一 CONNECTION Profile 响应设备 challenge，历史 per-device Profile 只读保留。

**Tech Stack:** Go 1.24、Gin、GORM、MySQL/SQLite、Redis、AES-256-GCM、HTTP Basic/Digest MD5/MD5-sess、Vue 3、Element Plus、Node test runner、Docker BS。

## Global Constraints

- 用户可见 Profile 固定为 `CONNECTION`、`LOG`、`PM`、`MR`；内部通道固定为 `CWMP_ACCESS`、`CONNECTION_REQUEST`、`LOG_UPLOAD`、`PM_UPLOAD`、`MR_UPLOAD`。
- `CWMP_ACCESS` 与 `CONNECTION_REQUEST` 默认映射同一个 CONNECTION Profile，但调用方只能按通道解析，不得硬编码共享字段。
- 用户名和密码只能同时为空或同时非空；清除必须显式 `clear=true`，修改用户名必须重新输入密码。
- GVA MUST NOT 生成、调度、下发或重试设置 `ConnectionRequestUsername/Password` 的命令，也不得读取设备密码做比较。
- 密码仅以版本化 AES-256-GCM 密文入库；API、普通日志、Trace、RawDump、命令 JSON/XML 与审计不得出现密码或 `Authorization`。
- Connection 凭据为空时 CWMP 与 Connection Request 保持无认证行为；启用后自动兼容 Basic、Digest MD5、MD5-sess、`qop=auth`，并校验 nonce TTL 与重放。
- 未认证请求不得读取 SOAP body、信任 Device ID 或改变设备正式状态；首次无 Authorization 的 401 challenge 不计为认证失败。
- 已有设备认证失败时保留历史数据且不更新 `LastInform`、IP、版本或参数；设备按现有 180 秒阈值自然离线。
- 不提供手动“测试密码”按钮；验证状态只能由实际 Inform、Connection Request 或文件上传推进。
- 本 change 不改变 `/acs/log`、`/acs/pm`、`/acs/mr` 文件入口、对象存储与传输任务业务；只提供后续 LOG/PM/MR changes 消费的认证核心。
- 每个任务先验证 RED，再完成 GREEN；每个提交只包含该任务列出的文件。

---

### Task 1: 建立凭据、通道、状态和审计数据基础

**OpenSpec tasks:** 1.1、1.2、4.5（历史状态模型部分）

**Files:**
- Create: `server/plugin/tr069/auth/types.go`
- Create: `server/plugin/tr069/model/credential_auth.go`
- Create: `server/plugin/tr069/initialize/credential_auth_migration_test.go`
- Modify: `server/plugin/tr069/model/connection_profile.go`
- Modify: `server/plugin/tr069/initialize/gorm.go`

**Interfaces:**
- Produces: `auth.Channel`、`auth.ProfileKey`、`auth.ResolvedCredential`、`auth.Principal`。
- Produces: `model.CredentialProfile`、`model.AuthChannelBinding`、`model.DeviceAuthState`、`model.AuthAuditEvent`。
- Produces: 四 Profile、五通道幂等种子数据，以及旧 `AUTO` Profile 的 `LEGACY/INACTIVE` 视图。

- [ ] **Step 1: 写模型唯一性、默认映射和历史保留失败测试**

在 `credential_auth_migration_test.go` 建 SQLite DB，迁移旧 `ConnectionProfile` 并插入一个 `READY/AUTO` 行，然后调用尚未存在的 `migrateCredentialAuthentication`：

```go
func TestMigrateCredentialAuthenticationSeedsStableProfilesAndBindings(t *testing.T) {
	db := openTR069MigrationTestDB(t)
	if err := db.AutoMigrate(new(model.Device), new(model.ConnectionProfile)); err != nil {
		t.Fatal(err)
	}
	device := model.Device{SerialNumber: "LEGACY-1", OUI: "001122"}
	if err := db.Create(&device).Error; err != nil { t.Fatal(err) }
	legacy := model.ConnectionProfile{DeviceID: device.ID, CredentialSource: model.ConnectionCredentialSourceAuto, ProvisionState: model.ConnectionProfileStateReady}
	if err := db.Create(&legacy).Error; err != nil { t.Fatal(err) }

	if err := migrateCredentialAuthentication(context.Background(), db); err != nil { t.Fatal(err) }
	if err := migrateCredentialAuthentication(context.Background(), db); err != nil { t.Fatal(err) }

	var profiles []model.CredentialProfile
	if err := db.Order("profile_key").Find(&profiles).Error; err != nil { t.Fatal(err) }
	if got := profileKeys(profiles); !reflect.DeepEqual(got, []string{"CONNECTION", "LOG", "MR", "PM"}) { t.Fatalf("profiles=%v", got) }
	var bindings []model.AuthChannelBinding
	if err := db.Preload("Profile").Order("channel").Find(&bindings).Error; err != nil { t.Fatal(err) }
	assertBinding(t, bindings, "CWMP_ACCESS", "CONNECTION")
	assertBinding(t, bindings, "CONNECTION_REQUEST", "CONNECTION")
	assertBinding(t, bindings, "LOG_UPLOAD", "LOG")
	assertBinding(t, bindings, "PM_UPLOAD", "PM")
	assertBinding(t, bindings, "MR_UPLOAD", "MR")
	if err := db.First(&legacy, legacy.ID).Error; err != nil { t.Fatal(err) }
	if legacy.ProvisionState != model.ConnectionProfileStateLegacy || legacy.CredentialSource != model.ConnectionCredentialSourceInactive { t.Fatalf("legacy=%#v", legacy) }
}
```

- [ ] **Step 2: 运行迁移测试并确认 RED**

Run: `cd server && go test ./plugin/tr069/initialize -run TestMigrateCredentialAuthentication -count=1`

Expected: FAIL，包含 `undefined: model.CredentialProfile` 或 `undefined: migrateCredentialAuthentication`。

- [ ] **Step 3: 定义稳定领域类型和数据库模型**

`auth/types.go` 固定跨 change 接口，不允许 LOG/PM/MR 各自复制常量：

```go
package auth

import (
	"context"
	"net/http"
)

type ProfileKey string
const (
	ProfileConnection ProfileKey = "CONNECTION"
	ProfileLog ProfileKey = "LOG"
	ProfilePM ProfileKey = "PM"
	ProfileMR ProfileKey = "MR"
)

type Channel string
const (
	ChannelCWMPAccess Channel = "CWMP_ACCESS"
	ChannelConnectionRequest Channel = "CONNECTION_REQUEST"
	ChannelLogUpload Channel = "LOG_UPLOAD"
	ChannelPMUpload Channel = "PM_UPLOAD"
	ChannelMRUpload Channel = "MR_UPLOAD"
)

type ResolvedCredential struct {
	Channel Channel
	ProfileID uint
	ProfileKey ProfileKey
	Username string
	Password string
	Revision uint64
	Realm string
	Enabled bool
}

type Principal struct {
	Channel Channel
	ProfileID uint
	Revision uint64
	Scheme string
	Disabled bool
}

type CredentialProvider interface {
	Resolve(context.Context, Channel) (ResolvedCredential, error)
}

type Authenticator interface {
	Authenticate(*http.Request, Channel) (Principal, []string, error)
}

type SuccessRecorder interface {
	RecordSuccess(context.Context, uint, Principal) error
}
```

`credential_auth.go` 使用独立表和复合唯一索引：Profile `profile_key` 唯一；Binding `channel` 唯一；State `(device_id,channel)` 唯一；Audit 只保存 channel、profile/revision、scheme、result、source IP、error code 和可选不可信声明，不保存 header/body。

- [ ] **Step 4: 注册迁移、幂等种子和旧 Profile 标记**

在 `initialize/gorm.go` 的 `AutoMigrate` 列表追加四个模型，再实现事务种子：

```go
var defaultCredentialBindings = map[auth.Channel]auth.ProfileKey{
	auth.ChannelCWMPAccess: auth.ProfileConnection,
	auth.ChannelConnectionRequest: auth.ProfileConnection,
	auth.ChannelLogUpload: auth.ProfileLog,
	auth.ChannelPMUpload: auth.ProfilePM,
	auth.ChannelMRUpload: auth.ProfileMR,
}

func migrateCredentialAuthentication(ctx context.Context, db *gorm.DB) error {
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		profiles := map[auth.ProfileKey]model.CredentialProfile{}
		for _, key := range []auth.ProfileKey{auth.ProfileConnection, auth.ProfileLog, auth.ProfilePM, auth.ProfileMR} {
			row := model.CredentialProfile{ProfileKey: string(key), Revision: 0}
			if err := tx.Where("profile_key = ?", key).FirstOrCreate(&row).Error; err != nil { return err }
			profiles[key] = row
		}
		for channel, key := range defaultCredentialBindings {
			binding := model.AuthChannelBinding{Channel: string(channel), ProfileID: profiles[key].ID}
			if err := tx.Where("channel = ?", channel).FirstOrCreate(&binding).Error; err != nil { return err }
		}
		return tx.Model(new(model.ConnectionProfile)).Where("credential_source = ?", model.ConnectionCredentialSourceAuto).
			Updates(map[string]any{"credential_source": model.ConnectionCredentialSourceInactive, "provision_state": model.ConnectionProfileStateLegacy}).Error
	})
}
```

- [ ] **Step 5: 运行模型/迁移测试并提交**

Run: `cd server && go test ./plugin/tr069/initialize ./plugin/tr069/model/... -count=1`

Expected: PASS，重复 migration 后仍只有四 Profile、五 Binding，旧行仍存在。

```bash
git add server/plugin/tr069/auth/types.go server/plugin/tr069/model/credential_auth.go server/plugin/tr069/model/connection_profile.go server/plugin/tr069/initialize/gorm.go server/plugin/tr069/initialize/credential_auth_migration_test.go
git commit -m "feat(tr069): add global credential authentication schema"
```

---

### Task 2: 实现加密仓储、原子 set/clear 和 revision 状态失效

**OpenSpec tasks:** 1.3、1.4、5.2（服务层规则部分）

**Files:**
- Create: `server/plugin/tr069/adapter/credential_profile_repository.go`
- Create: `server/plugin/tr069/adapter/credential_profile_repository_test.go`
- Create: `server/plugin/tr069/service/credential.go`
- Create: `server/plugin/tr069/service/credential_test.go`
- Modify: `server/plugin/tr069/adapter/connection_profile_crypto.go`
- Modify: `server/plugin/tr069/adapter/connection_profile_crypto_test.go`

**Interfaces:**
- Consumes: Task 1 的 `auth.ProfileKey`、`auth.Channel` 和四张表。
- Produces: `adapter.NewCredentialProfileRepository(db, cipher)`；`adapter.NewGormCredentialProvider(db, cipher)`。
- Produces: `service.NewCredentialService(store)` 及 `List`、`Set`、`Clear`；读 DTO 不含 password 字段。

- [ ] **Step 1: 写原子配置、缺密钥、并发 revision 和通道失效失败测试**

测试必须覆盖：半配置不改变旧行；`Set(CONNECTION)` revision 从 0 到 1 并把两个 Connection 通道置 `PENDING`；并发两次 set 最终 revision 为 3；`Clear` revision 再加一并置 `DISABLED`；cipher 缺失只阻止非空 set，不阻止 list/clear。

```go
func TestCredentialRepositoryRotatesConnectionAtomically(t *testing.T) {
	db, repo := newCredentialRepositoryTest(t)
	profile, err := repo.Set(context.Background(), auth.ProfileConnection, "acs-user", "secret-1", 7)
	if err != nil { t.Fatal(err) }
	if profile.Revision != 1 || !profile.PasswordConfigured { t.Fatalf("profile=%#v", profile) }
	assertChannelStatus(t, db, auth.ChannelCWMPAccess, "PENDING", 1)
	assertChannelStatus(t, db, auth.ChannelConnectionRequest, "PENDING", 1)
	assertChannelUntouched(t, db, auth.ChannelLogUpload)
	resolved, err := repo.Resolve(context.Background(), auth.ChannelCWMPAccess)
	if err != nil { t.Fatal(err) }
	if !resolved.Enabled || resolved.Username != "acs-user" || resolved.Password != "secret-1" || resolved.Revision != 1 { t.Fatalf("resolved=%#v", resolved) }
}
```

- [ ] **Step 2: 运行仓储测试并确认 RED**

Run: `cd server && go test ./plugin/tr069/adapter ./plugin/tr069/service -run 'Credential(Profile|Service|Repository)' -count=1`

Expected: FAIL，包含 `undefined: NewCredentialProfileRepository`。

- [ ] **Step 3: 泛化现有 AES-GCM envelope 并实现事务仓储**

在 `auth/types.go` 增加 `SealedSecret` 和 `SecretCipher`，将 adapter 旧类型改为别名，保持历史 Connection Profile 解密能力：

```go
type SealedSecret struct { Version string; Ciphertext []byte }
type SecretCipher interface {
	Encrypt(string) (SealedSecret, error)
	Decrypt(string, []byte) (string, error)
}
```

```go
type EncryptedCredential = auth.SealedSecret
type CredentialCipher = auth.SecretCipher
```

Repository 的 `Set`/`Clear` 必须在单个 GORM 事务内 `clause.Locking{Strength:"UPDATE"}` 读取 Profile、写密文、`revision = revision + 1`，并按 Binding 找到引用通道批量 upsert `DeviceAuthState`。`Resolve` 先通过 Binding 加载 Profile；用户名密码都空返回 `Enabled:false`，非空则当次解密并返回，不缓存明文。

- [ ] **Step 4: 加入不含秘密的 service DTO 与显式命令**

```go
type CredentialProfileView struct {
	Key auth.ProfileKey `json:"key"`
	Username string `json:"username"`
	PasswordConfigured bool `json:"passwordConfigured"`
	Revision uint64 `json:"revision"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type SetCredentialInput struct { Username string; Password string; ModifiedBy uint }

func (s *CredentialService) Set(ctx context.Context, key auth.ProfileKey, in SetCredentialInput) (CredentialProfileView, error) {
	if strings.TrimSpace(in.Username) == "" || in.Password == "" { return CredentialProfileView{}, ErrCredentialPairRequired }
	return s.store.Set(ctx, key, strings.TrimSpace(in.Username), in.Password, in.ModifiedBy)
}
```

`Clear` 不接受 username/password；`List` 固定按 CONNECTION、LOG、PM、MR 返回，即使数据库顺序不同。任何 error 只含 profile key 和稳定错误码，不含 username、password、密文或长度。

- [ ] **Step 5: 运行 race 定向测试并提交**

Run: `cd server && go test -race ./plugin/tr069/adapter ./plugin/tr069/service -run 'Credential(Profile|Service|Repository)' -count=1`

Expected: PASS；并发 revision 无重复，API DTO JSON 不含 `password`、`ciphertext`、`secret-1`。

```bash
git add server/plugin/tr069/auth/types.go server/plugin/tr069/adapter/credential_profile_repository.go server/plugin/tr069/adapter/credential_profile_repository_test.go server/plugin/tr069/adapter/connection_profile_crypto.go server/plugin/tr069/adapter/connection_profile_crypto_test.go server/plugin/tr069/service/credential.go server/plugin/tr069/service/credential_test.go
git commit -m "feat(tr069): store global credentials atomically"
```

---

### Task 3: 抽取唯一的 Basic/Digest 认证核心与安全审计

**OpenSpec tasks:** 2.1、2.2、2.3、2.4

**Files:**
- Create: `server/plugin/tr069/auth/http.go`
- Create: `server/plugin/tr069/auth/http_test.go`
- Create: `server/plugin/tr069/auth/context.go`
- Create: `server/plugin/tr069/adapter/auth_recorders.go`
- Create: `server/plugin/tr069/adapter/auth_recorders_test.go`

**Interfaces:**
- Consumes: `auth.CredentialProvider`、现有 `adapter.RedisDigestNonceStore`。
- Produces: `auth.NewHTTPAuthenticator(provider, nonces, audit)`；`auth.WithPrincipal`/`auth.PrincipalFromContext`。
- Produces: `adapter.NewGormAuthAuditRecorder(db)`、`adapter.NewGormAuthStateRecorder(db)`。
- Leaves to LOG change: 文件入口薄适配、旧 YAML 文件凭据 Provider 迁移和 `handler.FileRequestAuthenticator` 签名调整；本 task 只建立唯一共享认证核心。

- [ ] **Step 1: 把现有协议向量改写为通道无关失败测试**

从 `middleware/file_auth_test.go` 搬迁 Basic、MD5、MD5-sess、method/URI、nonce replay 用例，并新增关闭认证与 challenge 审计区分：

```go
func TestHTTPAuthenticatorDisabledProfilePassesWithoutAuthorization(t *testing.T) {
	provider := providerFunc(func(context.Context, Channel) (ResolvedCredential, error) {
		return ResolvedCredential{Channel: ChannelCWMPAccess, ProfileKey: ProfileConnection, Revision: 4, Enabled: false}, nil
	})
	a := NewHTTPAuthenticator(provider, nil, &captureAudit{})
	req := httptest.NewRequest(http.MethodPost, "/acs", nil)
	principal, challenges, err := a.Authenticate(req, ChannelCWMPAccess)
	if err != nil || !principal.Disabled || len(challenges) != 0 { t.Fatalf("principal=%#v challenges=%v err=%v", principal, challenges, err) }
}

func TestHTTPAuthenticatorMissingHeaderChallengesWithoutFailureAudit(t *testing.T) {
	a, audit := newEnabledAuthenticator(t)
	_, challenges, err := a.Authenticate(httptest.NewRequest(http.MethodPost, "/acs", nil), ChannelCWMPAccess)
	if !errors.Is(err, ErrChallengeRequired) || len(challenges) != 2 { t.Fatalf("challenges=%v err=%v", challenges, err) }
	if audit.failureCount != 0 { t.Fatalf("failure audits=%d", audit.failureCount) }
}
```

- [ ] **Step 2: 运行认证测试并确认 RED**

Run: `cd server && go test ./plugin/tr069/auth ./plugin/tr069/adapter -run '(HTTPAuth|DigestNonce|AuthRecorder)' -count=1`

Expected: FAIL，因为 `plugin/tr069/auth/http.go` 和新构造器尚不存在。

- [ ] **Step 3: 迁移唯一协议实现并保留 Redis 防重放**

`HTTPAuthenticator.Authenticate` 必须按以下确定顺序执行：Resolve channel；disabled 立即返回；无 header 返回 `ErrChallengeRequired` 和 Basic/Digest 两个 challenge；解析 scheme；Basic 对 username/password 先 SHA-256 成固定长度后 `subtle.ConstantTimeCompare`；Digest 校验 username、realm、method、URI、algorithm、qop、cnonce、nc 和 response，再消费 Redis nonce。成功 Principal 只含 channel/profile/revision/scheme。

```go
func (a *HTTPAuthenticator) Authenticate(r *http.Request, channel Channel) (Principal, []string, error) {
	credential, err := a.provider.Resolve(r.Context(), channel)
	if err != nil { return Principal{}, nil, ErrAuthConfiguration }
	principal := Principal{Channel: channel, ProfileID: credential.ProfileID, Revision: credential.Revision, Disabled: !credential.Enabled}
	if !credential.Enabled { return principal, nil, nil }
	authorization := strings.TrimSpace(r.Header.Get("Authorization"))
	if authorization == "" { return Principal{}, a.challenges(r.Context(), credential), ErrChallengeRequired }
	scheme, payload, ok := strings.Cut(authorization, " ")
	if !ok || !a.verify(r, scheme, payload, credential) {
		a.auditFailure(r.Context(), credential, scheme, sourceIP(r), "INVALID_CREDENTIAL")
		return Principal{}, a.challenges(r.Context(), credential), ErrAuthentication
	}
	principal.Scheme = strings.ToUpper(scheme)
	return principal, nil, nil
}
```

- [ ] **Step 4: 实现审计/成功 Recorder**

`GormAuthAuditRecorder` 仅写稳定枚举和 IP；`GormAuthStateRecorder.RecordSuccess(ctx, deviceID, principal)` 按 `(device_id,channel)` upsert 当前 revision、`SUCCESS` 与时间。不得在本 task 修改文件入口或创建文件认证适配器，避免与后续 LOG change 重复实现。

- [ ] **Step 5: 运行协议/审计/race 测试并提交**

Run: `cd server && go test -race ./plugin/tr069/auth ./plugin/tr069/adapter -run '(HTTPAuth|DigestNonce|AuthRecorder)' -count=1`

Expected: PASS；错误、challenge、审计 JSON 和 `fmt.Sprint(principal)` 均不含测试密码或 Authorization token。

```bash
git add server/plugin/tr069/auth server/plugin/tr069/adapter/auth_recorders.go server/plugin/tr069/adapter/auth_recorders_test.go
git commit -m "feat(tr069): centralize HTTP credential authentication"
```

---

### Task 4: 在读取 Inform 前阻断未认证 CWMP 并记录可信成功

**OpenSpec tasks:** 3.1、3.2、3.3、3.4

**Files:**
- Create: `server/plugin/tr069/middleware/cwmp_auth.go`
- Create: `server/plugin/tr069/middleware/cwmp_auth_test.go`
- Create: `server/plugin/tr069/initialize/credential_runtime.go`
- Create: `server/plugin/tr069/initialize/credential_runtime_test.go`
- Modify: `server/plugin/tr069/initialize/server.go`
- Modify: `server/plugin/tr069/initialize/server_runtime_test.go`
- Modify: `server/plugin/tr069/handler/cwmp.go`
- Modify: `server/plugin/tr069/handler/cwmp_test.go`
- Modify: `server/plugin/tr069/adapter/gorm_repo.go`
- Modify: `server/plugin/tr069/adapter/gorm_repo_test.go`
- Modify: `server/plugin/tr069/plugin.go`

**Interfaces:**
- Consumes: Task 3 `auth.Authenticator` 与 `auth.SuccessRecorder`。
- Produces: `initialize.ConfigureCredentialRuntime`、`CurrentCredentialProvider`、`CurrentHTTPAuthenticator`、`CurrentAuthSuccessRecorder`。
- Produces: `middleware.NewCWMPAccessAuth(core)`；认证 Principal 通过 request context 到 `GormDeviceRepo.UpsertFromInform`。

- [ ] **Step 1: 写“认证失败绝不读取 body/调用 next/upsert”失败测试**

```go
func TestCWMPAccessAuthRejectsBeforeBodyAndHandler(t *testing.T) {
	read := false
	body := &observedBody{Reader: strings.NewReader(validInformXML), onRead: func(){ read = true }}
	called := false
	engine := gin.New()
	engine.POST("/acs", NewCWMPAccessAuth(rejectingAuthenticator{}), func(c *gin.Context) { called = true; c.Status(204) })
	req := httptest.NewRequest(http.MethodPost, "/acs", nil)
	req.Body = body
	res := httptest.NewRecorder()
	engine.ServeHTTP(res, req)
	if res.Code != http.StatusUnauthorized || read || called { t.Fatalf("status=%d read=%v called=%v", res.Code, read, called) }
}
```

在 engine 集成测试记录 DeviceRepo 调用次数，错误凭据后断言 `UpsertFromInform` 为 0，数据库设备/参数/绑定/命令计数全为 0。

- [ ] **Step 2: 运行 CWMP 定向测试并确认 RED**

Run: `cd server && go test ./plugin/tr069/middleware ./plugin/tr069/handler ./plugin/tr069/engine ./plugin/tr069/adapter -run '(CWMPAccess|UnauthenticatedInform|AuthPrincipal)' -count=1`

Expected: FAIL，当前 `setupEngine` 在认证前挂载 RawDump 并直接进入 `CWMPHandler`。

- [ ] **Step 3: 建立共享运行时并调整 7458 中间件顺序**

```go
func ConfigureCredentialRuntime(provider auth.CredentialProvider, authenticator auth.Authenticator, success auth.SuccessRecorder) {
	credentialRuntime.Lock()
	defer credentialRuntime.Unlock()
	credentialRuntime.provider = provider
	credentialRuntime.authenticator = authenticator
	credentialRuntime.success = success
}
```

`plugin.Register` 在 Gorm migration 后构造 `adapter.NewRuntimeCredentialCipher()`、`adapter.NewGormCredentialProvider`、`adapter.NewGormAuthAuditRecorder`、`adapter.NewGormAuthStateRecorder` 和 `auth.NewHTTPAuthenticator`，再调用 Configure。`setupEngine` 顺序固定为 request limit → trace ID → `NewCWMPAccessAuth` → RawDump → RawResponseDump → CWMPHandler；根 `/` 与 `/acs` 使用同一 group。

- [ ] **Step 4: 认证成功后才记录设备通道状态**

`NewCWMPAccessAuth` 只在成功时把 Principal 写到 request context；`CWMPHandler` 不复制 Authorization 给 core engine。`GormDeviceRepo.UpsertFromInform` 成功查得 numeric device ID 后调用注入的 `auth.SuccessRecorder`，仅 Principal 为 enabled 且 revision 当前时写 `CWMP_ACCESS/SUCCESS`；disabled Profile 保持 `DISABLED`。

首次无 Authorization 返回 401 challenge 且无失败审计；错误凭据返回 401 只写 source IP 审计；已有设备的 `LastInform` 保持旧值。

- [ ] **Step 5: 运行 CWMP 全包测试并提交**

Run: `cd server && go test -race ./plugin/tr069/middleware ./plugin/tr069/handler ./plugin/tr069/engine ./plugin/tr069/adapter ./plugin/tr069/initialize -count=1`

Expected: PASS；认证关闭时既有 CWMP 测试保持 200/204，错误认证测试不产生业务写入。

```bash
git add server/plugin/tr069/middleware/cwmp_auth.go server/plugin/tr069/middleware/cwmp_auth_test.go server/plugin/tr069/initialize/credential_runtime.go server/plugin/tr069/initialize/credential_runtime_test.go server/plugin/tr069/initialize/server.go server/plugin/tr069/initialize/server_runtime_test.go server/plugin/tr069/handler/cwmp.go server/plugin/tr069/handler/cwmp_test.go server/plugin/tr069/adapter/gorm_repo.go server/plugin/tr069/adapter/gorm_repo_test.go server/plugin/tr069/plugin.go
git commit -m "feat(tr069): authenticate devices before Inform parsing"
```

---

### Task 5: 停止自动 Provisioning 并让 Connection Request 使用全局凭据

**OpenSpec tasks:** 4.1、4.2、4.3、4.4、4.5

**Files:**
- Create: `server/plugin/tr069/auth/client.go`
- Create: `server/plugin/tr069/auth/client_test.go`
- Modify: `server/plugin/tr069/adapter/connection_request.go`
- Modify: `server/plugin/tr069/adapter/connection_request_test.go`
- Modify: `server/plugin/tr069/adapter/connection_profile_repository.go`
- Modify: `server/plugin/tr069/adapter/connection_profile_repository_test.go`
- Modify: `server/plugin/tr069/adapter/gorm_repo.go`
- Modify: `server/plugin/tr069/adapter/gorm_repo_test.go`
- Modify: `server/plugin/tr069/adapter/datamodel_hook.go`
- Modify: `server/plugin/tr069/adapter/datamodel_hook_test.go`
- Modify: `server/plugin/tr069/engine/engine.go`
- Modify: `server/plugin/tr069/engine/engine_test.go`
- Modify: `server/plugin/tr069/config/config.go`
- Modify: `server/plugin/tr069/config/runtime_test.go`
- Delete: `server/plugin/tr069/adapter/connection_profile_provisioner.go`
- Delete: `server/plugin/tr069/adapter/connection_profile_provisioner_test.go`

**Interfaces:**
- Consumes: `auth.ChannelConnectionRequest`、`auth.CredentialProvider`、`auth.SuccessRecorder`。
- Produces: `auth.AuthorizationForChallenge(challenge, method, requestURI, credential)`，支持 Basic、Digest MD5/MD5-sess。
- Preserves: per-device `ConnectionProfile` 的 discovered/override URL 与 wake audit；不再读取其 username/password。

- [ ] **Step 1: 写无 SPV 与共享凭据 Connection Request 失败测试**

```go
func TestInformMissingConnectionCredentialsNeverSchedulesProvisioning(t *testing.T) {
	db, repo := newGormRepoTest(t)
	_, err := repo.UpsertFromInform(authenticatedContext(t), informWith(map[string]string{
		"Device.ManagementServer.ConnectionRequestURL": "http://192.0.2.10:7547/",
	}), "192.0.2.10")
	if err != nil { t.Fatal(err) }
	var count int64
	if err := db.Model(new(model.Command)).Where("dedup_key LIKE ?", "connection-profile:%").Count(&count).Error; err != nil { t.Fatal(err) }
	if count != 0 { t.Fatalf("credential SPV commands=%d", count) }
}
```

Fake server 分别返回 Basic/Digest challenge，断言第二次请求成功；provider 调用 channel 必须为 CONNECTION_REQUEST；旧 AUTO row 密码不同也不得被使用；全局 Profile disabled 时只发送一次无 Authorization 请求。

- [ ] **Step 2: 运行 Connection Request 测试并确认 RED**

Run: `cd server && go test ./plugin/tr069/auth ./plugin/tr069/adapter ./plugin/tr069/engine -run '(ConnectionRequest|Provision|GlobalCredential)' -count=1`

Expected: FAIL；当前会调度 Provisioner，且 `resolveConnectionRequestTarget` 解密旧 per-device 密码。

- [ ] **Step 3: 移除所有自动生成、调度、worker 和 payload hydrate 路径**

删除 Provisioner 文件及 engine 构造/worker；`GormDeviceRepo` 不再接受 scheduler；`DataModelHook` 删除 provisioner option；`Collect` 只接受 ConnectionRequestURL，不收集 username/password，不返回 `NeedsProvisioning`。保留 `autoProvisionCredentials` 字段用于 YAML 解析，但 runtime 若为 true 只执行一次：

```go
global.GVA_LOG.Warn("TR069 autoProvisionCredentials is deprecated and ignored")
```

警告不得打印配置对象、username、key 或 ciphertext。

- [ ] **Step 4: 实现共享凭据 challenge responder 和独立成功状态**

`TriggerConnectionRequest` 仍从旧 Repository 取 discovered/override URL，但从 `CurrentCredentialProvider().Resolve(ctx, auth.ChannelConnectionRequest)` 取凭据。首请求永不带 Authorization；401 后优先匹配设备返回的 Digest 或 Basic challenge，再构造第二请求。HTTP 2xx 时调用 `RecordSuccess(deviceID, principal)`；后续 `6 CONNECTION REQUEST` Inform 通过 CWMP event/状态字段单独记录，不把网络 2xx 等同于回呼成功。

```go
credential, err := initialize.CurrentCredentialProvider().Resolve(ctx, auth.ChannelConnectionRequest)
if err != nil { return failedResult(err) }
status, scheme, err := doConnectionRequest(ctx, profile.URL, credential, timeout)
if err == nil && credential.Enabled {
	_ = initialize.CurrentAuthSuccessRecorder().RecordSuccess(ctx, deviceID, auth.Principal{
		Channel: auth.ChannelConnectionRequest, ProfileID: credential.ProfileID,
		Revision: credential.Revision, Scheme: scheme,
	})
}
```

- [ ] **Step 5: 运行定向与无命令回归并提交**

Run: `cd server && go test -race ./plugin/tr069/auth ./plugin/tr069/adapter ./plugin/tr069/engine ./plugin/tr069/config -run '(ConnectionRequest|ConnectionProfile|Provision|GlobalCredential)' -count=1`

Expected: PASS；测试数据库没有新的 `gva-connection-request` ParameterKey/dedup 命令，旧 Profile 行仍存在且运行时未调用旧密文解密。

```bash
git add -A server/plugin/tr069/auth/client.go server/plugin/tr069/auth/client_test.go server/plugin/tr069/adapter server/plugin/tr069/engine server/plugin/tr069/config
git commit -m "feat(tr069): replace credential provisioning with validation"
```

---

### Task 6: 提供凭据管理、设备状态 API 与独立权限

**OpenSpec tasks:** 5.1、5.2

**Files:**
- Create: `server/plugin/tr069/model/request/credential.go`
- Create: `server/plugin/tr069/model/response/credential.go`
- Create: `server/plugin/tr069/api/credential.go`
- Create: `server/plugin/tr069/api/credential_test.go`
- Create: `server/plugin/tr069/router/credential.go`
- Create: `server/plugin/tr069/router/credential_test.go`
- Modify: `server/plugin/tr069/plugin.go`
- Modify: `server/plugin/tr069/initialize/api.go`
- Modify: `server/plugin/tr069/initialize/menu.go`
- Create: `server/plugin/tr069/initialize/menu_credential_test.go`
- Modify: `server/plugin/tr069/redact/connection_request.go`
- Modify: `server/plugin/tr069/redact/connection_request_test.go`

**Interfaces:**
- Produces: `GET /tr069/credentials`、`PUT /tr069/credentials/:profileKey`、`DELETE /tr069/credentials/:profileKey`。
- Produces: `GET /tr069/device/:deviceId/auth-state`，返回五通道正式状态和最近不可信事件。
- Consumes: `service.CredentialService`；所有路由沿用插件 JWT + Casbin group，并注册独立 SysApi 资源。

- [ ] **Step 1: 写 API 契约、半配置拒绝、权限注册与脱敏失败测试**

```go
func TestCredentialAPISetAndListNeverReturnsPassword(t *testing.T) {
	router, db := newCredentialAPITestRouter(t)
	putJSON(t, router, "/tr069/credentials/CONNECTION", `{"username":"acs-user","password":"api-secret"}`, http.StatusOK)
	body := getJSON(t, router, "/tr069/credentials", http.StatusOK)
	if bytes.Contains(body, []byte("api-secret")) || bytes.Contains(bytes.ToLower(body), []byte("cipher")) { t.Fatalf("secret response=%s", body) }
	assertRevision(t, db, "CONNECTION", 1)
}
```

另测 `{username:"x"}`、`{password:"x"}`、未知 key、隐式空对象均 400 且 revision 不变；DELETE 才能清除；OperationRecord 存固定 `***`；初始化 API 表包含四个新路径/方法。

- [ ] **Step 2: 运行 API/路由/菜单测试并确认 RED**

Run: `cd server && go test ./plugin/tr069/api ./plugin/tr069/router ./plugin/tr069/initialize ./plugin/tr069/redact -run '(Credential|AuthState)' -count=1`

Expected: FAIL，路由、request/response DTO 与菜单尚不存在。

- [ ] **Step 3: 实现原子命令 API 和统一错误映射**

PUT 请求只接受：

```go
type SetCredentialRequest struct {
	Username *string `json:"username" binding:"required"`
	Password *string `json:"password" binding:"required"`
}
```

两个非空才调用 Set；DELETE 调 Clear；任何响应只使用 `CredentialProfileView`。设备状态接口按固定五通道补齐缺失行：Profile disabled 显示 `DISABLED`，enabled 且无当前 revision 成功记录显示 `PENDING`，匹配当前 revision 才显示 `SUCCESS`；失败审计标记 `trusted:false`，不得覆盖正式状态。

- [ ] **Step 4: 注册路由、Casbin 资源和“凭据管理”菜单**

新建 `CredentialRouter` 并在 `plugin.Register` 注入 service。`initialize.Api` 注册上述 GET/PUT/DELETE/GET；菜单 `tr069Credentials` 使用 path `credentials`、component `plugin/tr069/view/credential/index.vue`、title `凭据管理`，位于设备列表之后。PUT 路由挂 `OperationRecordWithBodySanitizer(redact.CredentialJSON)`，sanitizer 对任意大小写 password/authorization 固定替换。

- [ ] **Step 5: 运行 API/权限/菜单回归并提交**

Run: `cd server && go test -race ./plugin/tr069/api ./plugin/tr069/router ./plugin/tr069/initialize ./plugin/tr069/redact -run '(Credential|AuthState)' -count=1`

Expected: PASS；四 Profile 全部可读，密码字段不存在，未授权请求被插件 JWT/Casbin 拦截。

```bash
git add server/plugin/tr069/model/request/credential.go server/plugin/tr069/model/response/credential.go server/plugin/tr069/api/credential.go server/plugin/tr069/api/credential_test.go server/plugin/tr069/router/credential.go server/plugin/tr069/router/credential_test.go server/plugin/tr069/plugin.go server/plugin/tr069/initialize/api.go server/plugin/tr069/initialize/menu.go server/plugin/tr069/initialize/menu_credential_test.go server/plugin/tr069/redact/connection_request.go server/plugin/tr069/redact/connection_request_test.go
git commit -m "feat(tr069): expose secure credential management APIs"
```

---

### Task 7: 实现四卡凭据页面与显式清除交互

**OpenSpec tasks:** 5.3、5.5（凭据页面部分）

**Files:**
- Create: `web/src/plugin/tr069/api/credential.js`
- Create: `web/src/plugin/tr069/view/credential/credential-view.js`
- Create: `web/src/plugin/tr069/view/credential/credential-view.test.js`
- Create: `web/src/plugin/tr069/view/credential/components/credential-card.vue`
- Create: `web/src/plugin/tr069/view/credential/index.vue`
- Create: `web/src/plugin/tr069/view/credential/credential.contract.test.js`

**Interfaces:**
- Consumes: Task 6 list/set/clear API 与 `passwordConfigured`。
- Produces: 四卡固定顺序；Connection 卡说明 CWMP_ACCESS 与 CONNECTION_REQUEST 共享。
- Preserves: 密码输入框每次打开和请求完成后为空；不存在“测试”操作。

- [ ] **Step 1: 写 API payload、表单规则和页面秘密契约失败测试**

```js
import test from 'node:test'
import assert from 'node:assert/strict'
import { buildSetPayload, profileCards, validateCredentialPair } from './credential-view.js'

test('profile cards are stable and connection explains both directions', () => {
  assert.deepEqual(profileCards.map(item => item.key), ['CONNECTION', 'LOG', 'PM', 'MR'])
  assert.match(profileCards[0].description, /设备接入/)
  assert.match(profileCards[0].description, /Connection Request/)
})

test('set requires username and a fresh password', () => {
  assert.equal(validateCredentialPair({ username: 'u', password: '' }), '用户名和密码必须同时配置')
  assert.deepEqual(buildSetPayload({ username: 'u', password: 'p' }), { username: 'u', password: 'p' })
})
```

Contract test 读取 Vue 源码，断言存在 `type="password"`、显式清除确认和 `passwordConfigured`，且不存在 `手动校验`、`测试密码`、把后端 password 赋回 form 的代码。

- [ ] **Step 2: 运行前端测试并确认 RED**

Run: `cd web && node --test src/plugin/tr069/view/credential/*.test.js`

Expected: FAIL，模块和组件尚不存在。

- [ ] **Step 3: 实现 API helper 和纯函数状态模型**

```js
export const listCredentialProfiles = () => service({ url: '/tr069/credentials', method: 'get' })
export const setCredentialProfile = (key, data) => service({ url: `/tr069/credentials/${key}`, method: 'put', data })
export const clearCredentialProfile = key => service({ url: `/tr069/credentials/${key}`, method: 'delete' })

export const validateCredentialPair = ({ username, password }) => {
  if (!username.trim() || !password) return '用户名和密码必须同时配置'
  return ''
}
```

页面 load 时只保存 username/passwordConfigured/revision/time/summary；`password` 永远初始化为 `''`，finally 中再次清空。

- [ ] **Step 4: 实现四卡页面和显式清除确认**

`CredentialCard` emits `save`/`clear`；保存按钮在 username/password 任一缺失时 disabled；修改 username 后必须输入新密码；清除使用 `ElMessageBox.confirm('清除后该模块将进入无认证模式，确定继续吗？', '清除凭据')`。Connection 卡显示两个独立状态汇总，不显示拆分开关；页面不存在人工校验按钮。

- [ ] **Step 5: 运行 Node 测试与生产构建并提交**

Run: `cd web && node --test src/plugin/tr069/view/credential/*.test.js && npm run build`

Expected: PASS；Vite production build 成功，生成页面不含服务端密码值。

```bash
git add web/src/plugin/tr069/api/credential.js web/src/plugin/tr069/view/credential
git commit -m "feat(tr069): add global credential management page"
```

---

### Task 8: 在设备详情展示五通道正式状态与不可信事件

**OpenSpec tasks:** 5.4、5.5（设备状态部分）

**Files:**
- Modify: `web/src/plugin/tr069/api/device.js`
- Create: `web/src/plugin/tr069/view/device/components/auth-state-drawer.vue`
- Create: `web/src/plugin/tr069/view/device/auth-state-view.js`
- Create: `web/src/plugin/tr069/view/device/auth-state-view.test.js`
- Modify: `web/src/plugin/tr069/view/device/index.vue`
- Create: `web/src/plugin/tr069/view/device/device-auth-state.contract.test.js`

**Interfaces:**
- Consumes: `GET /tr069/device/:deviceId/auth-state`。
- Produces: 五通道固定标签、`未启用/待验证/验证成功`，最近成功、来源 IP、revision；不可信事件明确标记 `trusted:false`。

- [ ] **Step 1: 写状态映射和“不把审计失败当正式失败”失败测试**

```js
test('untrusted audit does not replace formal pending status', () => {
  const view = buildAuthStateView({
    states: [{ channel: 'LOG_UPLOAD', status: 'PENDING', revision: 2 }],
    events: [{ channel: 'LOG_UPLOAD', result: 'REJECTED', trusted: false, sourceIp: '192.0.2.8' }]
  })
  assert.equal(view.channels.find(item => item.channel === 'LOG_UPLOAD').label, '待验证')
  assert.equal(view.events[0].trustedLabel, '未可信的认证尝试')
})
```

- [ ] **Step 2: 运行设备状态前端测试并确认 RED**

Run: `cd web && node --test src/plugin/tr069/view/device/auth-state-view.test.js src/plugin/tr069/view/device/device-auth-state.contract.test.js`

Expected: FAIL，helper 与 drawer 尚不存在。

- [ ] **Step 3: 实现 API、固定映射和 Drawer**

```js
export const getDeviceAuthState = deviceId => service({ url: `/tr069/device/${deviceId}/auth-state`, method: 'get' })

export const channelLabels = {
  CWMP_ACCESS: '设备接入', CONNECTION_REQUEST: 'Connection Request',
  LOG_UPLOAD: 'LOG', PM_UPLOAD: 'PM', MR_UPLOAD: 'MR'
}
export const statusLabels = { DISABLED: '未启用', PENDING: '待验证', SUCCESS: '验证成功' }
```

Drawer 的事件区域显示模块、来源 IP、时间和 error code，不显示 Authorization、username、password 或请求 body。

- [ ] **Step 4: 接入设备行操作且不增加测试按钮**

在设备表“操作”区增加“认证状态”，打开 Drawer 时按设备 ID 加载；不因审计失败改变现有在线状态 tag；源码 contract 断言没有 `validateCredential`、`testCredential` 或“手动校验”。

- [ ] **Step 5: 运行设备 UI 测试与构建并提交**

Run: `cd web && node --test src/plugin/tr069/view/device/*.test.js src/plugin/tr069/view/device/components/*.test.js && npm run build`

Expected: PASS；五通道按固定顺序展示，空事件时显示空状态。

```bash
git add web/src/plugin/tr069/api/device.js web/src/plugin/tr069/view/device
git commit -m "feat(tr069): show per-device authentication state"
```

---

### Task 9: 完成全链路脱敏、配置迁移文档和两台 BS 验收

**OpenSpec tasks:** 6.1、6.2、6.3、6.4、6.5

**Files:**
- Modify: `server/plugin/tr069/redact/connection_request.go`
- Modify: `server/plugin/tr069/redact/connection_request_test.go`
- Modify: `server/plugin/tr069/middleware/raw_dump.go`
- Modify: `server/plugin/tr069/middleware/raw_response_dump.go`
- Modify: `server/plugin/tr069/middleware/raw_response_dump_test.go`
- Modify: `server/plugin/tr069/trace/trace.go`
- Modify: `server/plugin/tr069/trace/trace_test.go`
- Modify: `server/plugin/tr069/adapter/command_xml_sink_test.go`
- Modify: `server/plugin/tr069/api/command_record_test.go`
- Modify: `server/config.yaml`
- Modify: `server/config.docker.yaml`
- Modify: `server/plugin/tr069/docs/protocol_interface_requirements.md`
- Create: `server/plugin/tr069/docs/credential_authentication.md`
- Modify: `openspec/changes/secure-tr069-credential-authentication/tasks.md`

**Interfaces:**
- Consumes: 所有前述任务。
- Produces: 任意 XML `Username`/`Password`、JSON password、HTTP Authorization 的统一固定占位符；部署文档与可重复 BS 验收证据。

- [ ] **Step 1: 写秘密 canary 的存储/日志/Trace 失败测试**

使用 `CANARY-CWMP-PASSWORD-7f2c`、`Basic Q0FOQVJZ`、Digest `response="deadbeef"` 贯穿 RawDump、RawResponse、Trace、CommandXML、CommandRecord detail 和 operation log；把所有序列化结果合并后断言：

```go
for _, forbidden := range []string{"CANARY-CWMP-PASSWORD-7f2c", "Basic Q0FOQVJZ", `response="deadbeef"`} {
	if strings.Contains(combined, forbidden) { t.Fatalf("secret leaked: %q", forbidden) }
}
```

同时断言 SOAP 中普通 `Username`、`Password` element 及 ParameterValueStruct 名称匹配的值都变为固定 `***`，而非只处理 ConnectionRequestPassword。

- [ ] **Step 2: 运行安全测试并确认 RED**

Run: `cd server && go test ./plugin/tr069/redact ./plugin/tr069/middleware ./plugin/tr069/trace ./plugin/tr069/adapter ./plugin/tr069/api -run '(Redact|Secret|Authorization|RawDump|CommandXML)' -count=1`

Expected: FAIL；当前 `CWMPXML` 只覆盖特定 ConnectionRequestPassword，通用 Upload/认证字段仍可能泄漏。

- [ ] **Step 3: 统一脱敏边界并删除 Authorization 向下传播**

所有 HTTP header dump 在格式化前删除 Authorization；Trace 只记录 `hasAuth` 和 scheme；XML sanitizer 同时处理 element 与 ParameterValueStruct；JSON sanitizer 对大小写不敏感的 username/password/authorization 使用固定占位符。认证 error 只允许稳定码：`CHALLENGE_REQUIRED`、`INVALID_CREDENTIAL`、`NONCE_EXPIRED`、`NONCE_REPLAY`、`AUTH_CONFIG_UNAVAILABLE`。

- [ ] **Step 4: 更新配置和部署/回滚文档**

配置保留 `autoProvisionCredentials` 但设为 `false` 并标注 deprecated；新增独立凭据主密钥版本/密钥说明，YAML 不再包含运行时 LOG/PM/MR username/password。文档明确：用户必须先在每台 BS 两个 Connection 方向设置同一凭据，再在 GVA 保存；Basic 生产必须 HTTPS；清空 Profile 回到无认证；旧 AUTO Profile 与历史命令只读保留；全局凭据泄漏影响所有设备。

- [ ] **Step 5: 运行完整静态、race、前端回归**

Run: `cd server && go test ./plugin/tr069/... -count=1 && go test -race ./plugin/tr069/... -count=1 && go vet ./plugin/tr069/...`

Expected: 全部 PASS，无 race/vet 错误。

Run: `cd web && node --test src/plugin/tr069/**/*.test.js && npm run build`

Expected: 全部 PASS，production build 成功。

- [ ] **Step 6: 使用两台 Docker BS 完成黑盒矩阵**

依次验证并保存命令输出到实施会话记录，不把密码写入 shell history：

1. Connection Profile 为空：两台 BS Inform 成功，状态 `DISABLED`。
2. 两台 BS 与 GVA 配置相同 Connection 凭据：首次 401 challenge 后 Inform 成功，两个设备 `CWMP_ACCESS/SUCCESS`。
3. 仅把 BS-02 改为错误凭据：BS-02 返回 401、其 `LastInform` 不更新且历史行保留；BS-01 不受影响；失败审计只含 IP。
4. 恢复 BS-02：下一次 Inform 自动恢复 SUCCESS。
5. 对两台设备执行真实 Connection Request：共享凭据完成 Basic/Digest，分别更新 `CONNECTION_REQUEST/SUCCESS` 并收到 `6 CONNECTION REQUEST` Inform。
6. 查询数据库：新命令中 `ParameterKey <> 'gva-connection-request'`，旧 AUTO Profile 仍在且为 LEGACY/INACTIVE。
7. 对后端日志、TR-069 info log、Trace API、命令详情和数据库文本列搜索 canary，结果为 0。

- [ ] **Step 7: 更新 OpenSpec 清单并提交**

把 `openspec/changes/secure-tr069-credential-authentication/tasks.md` 中经测试和黑盒确认的 1.1–6.5 全部标记 `[x]`；任何未执行的 BS 场景保持 `[ ]`，不得以单元测试代替。

```bash
git add server/plugin/tr069 server/config.yaml server/config.docker.yaml web/src/plugin/tr069 openspec/changes/secure-tr069-credential-authentication/tasks.md
git commit -m "test(tr069): verify secure credential authentication"
```

Run: `git diff --check HEAD^ && git status --short`

Expected: `git diff --check` 无输出；工作树只剩用户原有、与本 change 无关的修改。
