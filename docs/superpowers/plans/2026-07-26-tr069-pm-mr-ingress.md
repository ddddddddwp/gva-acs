---
change: enable-tr069-pm-mr-ingress
design-doc: docs/superpowers/specs/2026-07-26-tr069-pm-mr-ingress-design.md
base-ref: bb89d5f2f98f861d62f9720b5468e5e9380b6104
---

# TR-069 PM/MR 固定文件入口实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在 7458 TR-069 监听器上增加固定 `/acs/pm`、`/acs/mr` 周期文件入口，并让 PM、MR 在认证、设备归属、任务、存储、权限和前端展示上完全隔离。

**Architecture:** 复用已经由 LOG change 建立的流式 FileIngress 管线，只为 PM/MR 增加受控 channel policy、独立凭据映射和周期任务约束。文件认证成功后才按最近成功 Inform 的来源 IP 解析唯一设备；管理端使用固定 channel 路由和服务端白名单，不接受前端任意覆盖 channel。

**Tech Stack:** Go 1.24、Gin、GORM、Redis、MinIO、SQLite 测试、Vue 3、Element Plus、Node test runner。

## Global Constraints

- 必须先完成 `secure-tr069-credential-authentication`，获得 `auth.ChannelPMUpload`、`auth.ChannelMRUpload`、`auth.CredentialProvider`、`auth.Principal` 和正式验证状态记录器。
- 必须先完成修订后的 `add-tr069-log-collection`，获得 `FileAuthAdapter`、`FileAuthSuccessRecorder` 和不携带 Upload 凭据的 LOG 流程。
- 固定入口只能是 `PUT/POST /acs/pm[/*filename]` 和 `PUT/POST /acs/mr[/*filename]`；不得把设备标识加入 URL。
- PM 只能使用 PM Profile，MR 只能使用 MR Profile；用户名和密码都为空时保持无认证业务可用。
- 认证必须先于设备解析和文件正文读取；认证失败返回 401，未知、过期或歧义 IP 返回 403。
- PM/MR 只能创建 `source=PERIODIC` 的任务，不得增加主动 Upload API、命令或 `WAITING_FILE` 任务。
- 文件必须流式写入对象存储，不创建整包内存副本或临时文件，并复用限长、并发、超时、SHA-256、Abort、协调和清理语义。
- 管理 API 必须复核 JWT、Casbin、设备数据范围、固定 channel 和 `AVAILABLE` 状态；响应不得包含对象键、驱动、来源 IP、摘要或凭据字段。
- GVA 不读取后复制、不设置、不下发基站 PM/MR 的 URL、用户名、密码、启用状态或周期参数。
- Basic 只能在生产 HTTPS 下使用；开发 HTTP 只用于两个 BS Docker 的受控验收。

---

## 文件结构与职责

- `server/plugin/tr069/config/config.go`：声明文件 channel 的显式 `activeUpload` 能力，PM/MR 固定为 false。
- `server/plugin/tr069/config/runtime.go`：规范化并复制 PM/MR 的通道资源策略。
- `server/plugin/tr069/config/runtime_test.go`：验证固定路径、唯一路径、PM/MR 周期专用和独立资源配置。
- `server/plugin/tr069/initialize/server.go`：按每个 enabled channel 构造独立认证适配器和路由处理链。
- `server/plugin/tr069/initialize/server_runtime_test.go`：验证固定路由、非法路径、RawDump/CWMP 隔离和 channel 映射。
- `server/plugin/tr069/handler/file_ingress.go`：把通道的 `AllowActive` 策略传入接收服务，不复制 handler。
- `server/plugin/tr069/handler/file_ingress_test.go`：参数化覆盖 PM/MR raw PUT、POST、multipart、限长和认证先行。
- `server/plugin/tr069/service/transfer_receiver.go`：只有 `AllowActive=true` 的 LOG 才匹配 ACTIVE 任务；PM/MR直接创建 PERIODIC。
- `server/plugin/tr069/service/transfer_receiver_test.go`：证明 PM/MR不会消费任何 ACTIVE/WAITING_FILE 任务。
- `server/plugin/tr069/service/transfer_store.go`：让制品列表和下载查询要求固定 channel，并支持设备数据范围。
- `server/plugin/tr069/service/transfer_store_test.go`：验证 AVAILABLE、channel、设备范围和对象前缀隔离。
- `server/plugin/tr069/service/transfer_worker.go`：按制品 channel 使用各自 upload timeout，并保持按制品 `DeleteAt` 清理。
- `server/plugin/tr069/service/transfer_worker_test.go`：验证 PM/MR协调与清理不会串 channel 或误删对象。
- `server/plugin/tr069/service/artifact_access_scope.go`：把当前 GVA authority/data-authority 映射为允许的设备分组范围。
- `server/plugin/tr069/service/artifact_access_scope_test.go`：验证管理员、受限角色和无范围角色。
- `server/plugin/tr069/api/artifact.go`：提供固定 channel 的列表和下载 handler factory。
- `server/plugin/tr069/api/artifact_test.go`：验证 PM/MR列表、下载、越权和秘密字段省略。
- `server/plugin/tr069/router/artifact.go`：注册显式 PM/MR 管理路由，不接受用户任意 channel。
- `server/plugin/tr069/router/artifact_test.go`：验证路由集合且不存在 PM/MR主动上传端点。
- `server/plugin/tr069/initialize/api.go`：注册 PM/MR Casbin API 资源。
- `server/plugin/tr069/initialize/menu.go`：注册 PM、MR 独立菜单。
- `server/plugin/tr069/initialize/menu_pm_mr_file_test.go`：验证菜单幂等、组件、顺序和权限资源。
- `server/config.yaml`、`server/config.docker.yaml`：保留固定地址并显式声明 LOG 可主动、PM/MR仅周期；被 `.gitignore` 排除的 `server/config.local.yaml` 只在黑盒验收时本地调整，不提交秘密。
- `web/src/plugin/tr069/api/artifact-file.js`：按受控常量调用 PM/MR列表和下载 API。
- `web/src/plugin/tr069/view/file-artifact/artifact-file-table.vue`：共享列表、筛选、分页和下载 UI。
- `web/src/plugin/tr069/view/file-artifact/artifact-file-view.js`：共享文件大小、来源和下载帮助函数。
- `web/src/plugin/tr069/view/pm-file/index.vue`、`web/src/plugin/tr069/view/mr-file/index.vue`：只绑定固定 channel/title，不显示立即上传。
- `web/src/plugin/tr069/view/file-artifact/artifact-file-view.test.js`：共享显示与下载单元测试。
- `web/src/plugin/tr069/view/file-artifact/artifact-file.contract.test.js`：PM/MR 固定 channel、无主动上传和主题契约测试。
- `server/plugin/tr069/docs/pm_mr_ingress.md`：运维配置、固定地址、凭据、唯一 IP、HTTPS、回滚和验收说明。

### Task 1: 固化 PM/MR channel policy 与固定路由

**Files:**
- Modify: `server/plugin/tr069/config/config.go`
- Modify: `server/plugin/tr069/config/runtime.go`
- Modify: `server/plugin/tr069/config/runtime_test.go`
- Modify: `server/plugin/tr069/initialize/server.go`
- Modify: `server/plugin/tr069/initialize/server_runtime_test.go`
- Modify: `server/plugin/tr069/handler/file_ingress.go`
- Modify: `server/plugin/tr069/handler/file_ingress_test.go`
- Modify: `server/config.yaml`
- Modify: `server/config.docker.yaml`

**Interfaces:**
- Consumes: `FileRequestAuthenticator.Authenticate(*http.Request) (auth.Principal, string, []string, error)`；`middleware.FileAuthAdapter`；`initialize.CurrentHTTPAuthenticator()`、`CurrentAuthSuccessRecorder()`。
- Produces: `TransferChannelConfig.ActiveUpload bool`；`FileIngressChannel.AllowActive bool`；PM/MR各自绑定到 `auth.ChannelPMUpload`/`auth.ChannelMRUpload` 的路由链。

- [ ] **Step 1: 写入配置与路由失败测试**

在 `config/runtime_test.go` 增加表驱动测试，断言 PM/MR 固定路径、独立前缀、独立限制以及 `ActiveUpload=false`；再在 `initialize/server_runtime_test.go` 构造三个 channel，断言 `/acs/pm`、`/acs/mr` 的 PUT/POST进入文件 handler，而 `/acs/pm/a/b`、DELETE 和 `/acs/PM` 不创建接收调用。

```go
func TestNormalizePMMRChannelsRemainPeriodicOnly(t *testing.T) {
	cfg := validFileIngressConfig()
	cfg.FileIngress.Channels["pm"] = TransferChannelConfig{Enabled: true, Path: "/acs/pm", StoragePrefix: "pm", ActiveUpload: false}
	cfg.FileIngress.Channels["mr"] = TransferChannelConfig{Enabled: true, Path: "/acs/mr", StoragePrefix: "mr", ActiveUpload: false}

	got := NormalizeRuntimeConfig(cfg)
	for name, wantPath := range map[string]string{"pm": "/acs/pm", "mr": "/acs/mr"} {
		channel := got.FileIngress.Channels[name]
		if channel.Path != wantPath || channel.StoragePrefix != name || channel.ActiveUpload {
			t.Fatalf("channel %s = %#v", name, channel)
		}
		if channel.MaxFileSize <= 0 || channel.MaxConcurrent <= 0 || channel.UploadTimeout <= 0 || channel.RetentionDays <= 0 {
			t.Fatalf("channel %s limits were not normalized: %#v", name, channel)
		}
	}
}
```

- [ ] **Step 2: 运行测试确认红灯**

Run: `cd server && env GOWORK=off go test ./plugin/tr069/config ./plugin/tr069/initialize ./plugin/tr069/handler -run 'PMMR|FileIngressRoute|RawDump' -count=1`

Expected: FAIL，至少包含 `unknown field ActiveUpload` 或 `/acs/pm`、`/acs/mr` 未注册的断言。

- [ ] **Step 3: 实现最小 channel policy 与按路由认证 wiring**

在 `TransferChannelConfig` 和 handler policy 中加入显式能力位；路由构造必须把配置 channel 映射成受控认证 channel，未知名称直接返回配置错误。

```go
type TransferChannelConfig struct {
	Enabled                bool   `mapstructure:"enabled" json:"enabled" yaml:"enabled"`
	Path                   string `mapstructure:"path" json:"path" yaml:"path"`
	ActiveUpload           bool   `mapstructure:"activeUpload" json:"activeUpload" yaml:"activeUpload"`
	MaxFileSize            int64  `mapstructure:"maxFileSize" json:"maxFileSize" yaml:"maxFileSize"`
	MaxConcurrent          int    `mapstructure:"maxConcurrent" json:"maxConcurrent" yaml:"maxConcurrent"`
	MaxConcurrentPerDevice int    `mapstructure:"maxConcurrentPerDevice" json:"maxConcurrentPerDevice" yaml:"maxConcurrentPerDevice"`
	UploadTimeout          int    `mapstructure:"uploadTimeout" json:"uploadTimeout" yaml:"uploadTimeout"`
	RetentionDays          int    `mapstructure:"retentionDays" json:"retentionDays" yaml:"retentionDays"`
	StoragePrefix          string `mapstructure:"storagePrefix" json:"storagePrefix" yaml:"storagePrefix"`
}

func fileAuthChannel(name string) (auth.Channel, error) {
	switch strings.ToUpper(strings.TrimSpace(name)) {
	case "LOG":
		return auth.ChannelLogUpload, nil
	case "PM":
		return auth.ChannelPMUpload, nil
	case "MR":
		return auth.ChannelMRUpload, nil
	default:
		return "", fmt.Errorf("unsupported file ingress channel %q", name)
	}
}
```

`buildRuntimeFileIngressRoutes` 必须为每个 channel 创建独立的 `FileAuthAdapter` 与 handler，并把固定 auth channel 传给共享 `CurrentHTTPAuthenticator()`；不能继续让一个共享 YAML username/password Provider根据路径猜 channel。`RuntimeFileIngressChannelProvider.Channel` 返回 `AllowActive: channel.ActiveUpload`，并组合全局对象前缀与 channel storage prefix，最终对象键只出现一次 channel 段。

- [ ] **Step 4: 更新三份配置样例**

两个受版本控制的 YAML 中都使用以下受控值；不要恢复已废弃的共享文件用户名/密码：

```yaml
channels:
  log:
    enabled: true
    path: /acs/log
    activeUpload: true
    storagePrefix: log
  pm:
    enabled: true
    path: /acs/pm
    activeUpload: false
    storagePrefix: pm
  mr:
    enabled: true
    path: /acs/mr
    activeUpload: false
    storagePrefix: mr
```

保留各文件现有 `maxFileSize`、`maxConcurrent`、`maxConcurrentPerDevice`、`uploadTimeout`、`retentionDays` 精确值；`fileIngress.enabled` 和 MinIO秘密仍由部署环境决定。

- [ ] **Step 5: 运行聚焦测试确认绿灯**

Run: `cd server && env GOWORK=off go test ./plugin/tr069/config ./plugin/tr069/initialize ./plugin/tr069/handler -run 'PMMR|FileIngressRoute|RawDump|VendorPaths' -count=1`

Expected: PASS；PM/MR文件请求没有进入 `RawDump` 或 CWMP XML handler。

- [ ] **Step 6: 提交本任务**

```bash
git add server/plugin/tr069/config/config.go server/plugin/tr069/config/runtime.go server/plugin/tr069/config/runtime_test.go server/plugin/tr069/initialize/server.go server/plugin/tr069/initialize/server_runtime_test.go server/plugin/tr069/handler/file_ingress.go server/plugin/tr069/handler/file_ingress_test.go server/config.yaml server/config.docker.yaml
git commit -m "feat(tr069): register periodic PM and MR ingress channels"
```

### Task 2: 验证独立凭据、固定路径和认证先行

**Files:**
- Modify: `server/plugin/tr069/middleware/file_auth_test.go`
- Modify: `server/plugin/tr069/handler/file_ingress_test.go`
- Modify: `server/plugin/tr069/initialize/server_runtime_test.go`

**Interfaces:**
- Consumes: `auth.ChannelPMUpload`、`auth.ChannelMRUpload`；`FileAuthAdapter.Authenticate(*http.Request) (auth.Principal, string, []string, error)`。
- Produces: PM/MR Basic、Digest、空 Profile、跨模块拒绝的集成测试夹具。

- [ ] **Step 1: 写独立与交叉凭据失败测试**

为 `/acs/pm` 和 `/acs/mr` 分别提供不同 Profile，使用表驱动测试覆盖正确 Basic、正确 Digest、无 Authorization challenge、空 Profile放行、LOG/MR凭据访问PM以及PM凭据访问MR。正文使用计数 reader，所有401路径必须保持读取计数为0。

```go
tests := []struct {
	name       string
	path       string
	authHeader string
	wantStatus int
	wantReads  int
}{
	{name: "pm accepts pm basic", path: "/acs/pm", authHeader: basic("pm-user", "pm-pass"), wantStatus: http.StatusCreated, wantReads: 1},
	{name: "pm rejects mr basic", path: "/acs/pm", authHeader: basic("mr-user", "mr-pass"), wantStatus: http.StatusUnauthorized, wantReads: 0},
	{name: "mr rejects log basic", path: "/acs/mr", authHeader: basic("log-user", "log-pass"), wantStatus: http.StatusUnauthorized, wantReads: 0},
	{name: "enabled profile challenges missing auth", path: "/acs/mr", wantStatus: http.StatusUnauthorized, wantReads: 0},
}
```

Digest测试必须检查 realm、method、URI、nonce、nc/cnonce，且把 `/acs/pm` 的有效摘要重放到 `/acs/mr` 时返回401。

- [ ] **Step 2: 运行测试确认红灯**

Run: `cd server && env GOWORK=off go test ./plugin/tr069/middleware ./plugin/tr069/handler ./plugin/tr069/initialize -run 'PMMR|CrossChannel|BeforeBody|Digest' -count=1`

Expected: FAIL，交叉凭据被错误接受或 PM/MR尚未映射独立 auth channel。

- [ ] **Step 3: 只调整 channel adapter，不复制认证算法**

保持 Basic/Digest 算法仅存在于共享 `auth.Authenticator`。PM route 构造 `FileAuthAdapter` 时固定 `auth.ChannelPMUpload` 和传输 channel `PM`，MR同理；handler 校验认证返回的 transfer channel 与 route policy name完全一致。

```go
type fileRouteSpec struct {
	TransferChannel string
	AuthChannel     auth.Channel
	Path            string
}

var supportedFileRoutes = map[string]fileRouteSpec{
	"log": {TransferChannel: "LOG", AuthChannel: auth.ChannelLogUpload, Path: "/acs/log"},
	"pm":  {TransferChannel: "PM", AuthChannel: auth.ChannelPMUpload, Path: "/acs/pm"},
	"mr":  {TransferChannel: "MR", AuthChannel: auth.ChannelMRUpload, Path: "/acs/mr"},
}
```

不能从请求 query、header 或文件名读取 channel，也不能为错误凭据解析 RemoteAddr或调用设备 resolver。

- [ ] **Step 4: 运行认证与秘密扫描测试**

Run: `cd server && env GOWORK=off go test ./plugin/tr069/auth ./plugin/tr069/middleware ./plugin/tr069/handler ./plugin/tr069/initialize -run 'PMMR|CrossChannel|BeforeBody|Digest|Secret|Authorization' -count=1`

Expected: PASS；正常首次 challenge不产生失败状态，错误认证审计不含 Authorization、用户名关联密码或正文。

- [ ] **Step 5: 提交本任务**

```bash
git add server/plugin/tr069/middleware/file_auth_test.go server/plugin/tr069/handler/file_ingress_test.go server/plugin/tr069/initialize/server_runtime_test.go
git commit -m "test(tr069): enforce isolated PM and MR file authentication"
```

### Task 3: 强制唯一设备归属与 PERIODIC-only 任务

**Files:**
- Modify: `server/plugin/tr069/service/transfer_receiver.go`
- Modify: `server/plugin/tr069/service/transfer_receiver_test.go`
- Modify: `server/plugin/tr069/service/upload_device_resolver.go`
- Modify: `server/plugin/tr069/handler/file_ingress.go`
- Modify: `server/plugin/tr069/handler/file_ingress_test.go`

**Interfaces:**
- Consumes: `UploadDeviceResolver.Resolve(ctx, sourceIP, channel)`；`FileAuthSuccessRecorder.RecordSuccess(ctx, deviceID, principal)`。
- Produces: `ReceiveRequest.AllowActive bool`；PM/MR只产生 `PERIODIC`；认证成功且唯一设备解析后记录对应 principal revision。

- [ ] **Step 1: 写唯一IP和周期专用失败测试**

在 `transfer_receiver_test.go` 用 SQLite + fake identity resolver 覆盖唯一 Redis绑定、无绑定、过期绑定、两个候选的 NAT 歧义、MySQL近期唯一回退。为同一设备预置 PM/MR `ACTIVE + WAITING_FILE` 任务，再接收文件并断言旧任务未变化、新任务为 PERIODIC。

```go
func TestPMMRReceiveNeverConsumesActiveTask(t *testing.T) {
	for _, channel := range []string{"PM", "MR"} {
		t.Run(channel, func(t *testing.T) {
			store, db, device := newTransferStoreTest(t)
			seedWaitingActiveTask(t, db, device.ID, channel)
			receiver := NewTransferReceiver(store, newMemoryArtifactStore())
			artifact, err := receiver.Receive(context.Background(), ReceiveRequest{
				Device: UploadDeviceIdentity{DeviceID: device.ID}, Channel: channel,
				Body: strings.NewReader("periodic"), ContentLength: 8, Driver: "memory",
				StoragePrefix: "artifacts", MaxFileSize: 1024, UploadTimeout: time.Minute,
				RetentionDays: 30, MaxConcurrent: 2, MaxConcurrentPerDevice: 1, AllowActive: false,
			})
			if err != nil { t.Fatal(err) }
			var task model.TransferTask
			if err := db.First(&task, "task_id = ?", artifact.TaskID).Error; err != nil { t.Fatal(err) }
			if task.Source != model.TransferSourcePeriodic || task.Channel != channel { t.Fatalf("task = %#v", task) }
			assertWaitingActiveTaskUnchanged(t, db, device.ID, channel)
		})
	}
}
```

- [ ] **Step 2: 运行测试确认红灯**

Run: `cd server && env GOWORK=off go test ./plugin/tr069/service ./plugin/tr069/handler -run 'PMMR|UploadDevice|Periodic|Ambiguous' -count=1`

Expected: FAIL，PM/MR错误消费 ACTIVE 任务，或 `ReceiveRequest` 尚无 `AllowActive`。

- [ ] **Step 3: 实现周期专用分支**

`ReceiveRequest` 增加 `AllowActive`。`createReceivingMetadata` 只有该值为 true 时才调用 `FindUniqueWaitingActive`；false 时直接生成 UUID 并调用 `CreatePeriodicReceiving`。

```go
func (r *TransferReceiver) createReceivingMetadata(ctx context.Context, request ReceiveRequest, metadata ReceiveMetadata) (model.TransferTask, model.Artifact, error) {
	if request.AllowActive {
		active, err := r.transfers.FindUniqueWaitingActive(ctx, request.Device.DeviceID, request.Channel)
		if err == nil {
			return r.transfers.CreateActiveReceiving(ctx, active, metadata)
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return model.TransferTask{}, model.Artifact{}, err
		}
	}
	metadata.TaskID = uuid.NewString()
	return r.transfers.CreatePeriodicReceiving(ctx, request.Device.DeviceID, request.Channel, metadata)
}
```

handler从固定 channel policy传 `AllowActive`。与已锁定的 LOG 端口保持一致：认证成功、唯一设备解析完成后，读取正文前调用 `FileAuthSuccessRecorder.RecordSuccess(ctx, device.DeviceID, principal)`；disabled principal不记录成功。状态写入失败返回500，且正文读取数、任务数和对象数都保持0。

- [ ] **Step 4: 验证403与可信状态边界**

Run: `cd server && env GOWORK=off go test ./plugin/tr069/service ./plugin/tr069/handler -run 'PMMR|UploadDevice|Periodic|Ambiguous|AuthSuccess' -count=1`

Expected: PASS；未知/过期/歧义均403且无任务、无对象、无正式设备失败状态；唯一设备认证成功后记录 PM_UPLOAD/MR_UPLOAD 当前 revision，recorder失败不读取正文。

- [ ] **Step 5: 提交本任务**

```bash
git add server/plugin/tr069/service/transfer_receiver.go server/plugin/tr069/service/transfer_receiver_test.go server/plugin/tr069/service/upload_device_resolver.go server/plugin/tr069/handler/file_ingress.go server/plugin/tr069/handler/file_ingress_test.go
git commit -m "feat(tr069): persist PM and MR uploads as periodic transfers"
```

### Task 4: 参数化对象存储、协调和清理

**Files:**
- Modify: `server/plugin/tr069/service/artifact_store.go`
- Modify: `server/plugin/tr069/service/artifact_store_contract_test.go`
- Modify: `server/plugin/tr069/service/transfer_receiver_test.go`
- Modify: `server/plugin/tr069/service/transfer_store.go`
- Modify: `server/plugin/tr069/service/transfer_store_test.go`
- Modify: `server/plugin/tr069/service/transfer_worker.go`
- Modify: `server/plugin/tr069/service/transfer_worker_test.go`

**Interfaces:**
- Consumes: `ArtifactStore.Begin/Open/Stat/Delete`；`TransferChannelRuntime.UploadTimeout`、`RetentionDays`、`StoragePrefix`。
- Produces: 确定性 `artifacts/{pm|mr}/{deviceID}/{yyyy/mm/dd}/{fileID}` 对象键；按 channel 的 stale receiving协调。

- [ ] **Step 1: 写存储和可靠性失败测试**

对 PM/MR参数化现有 memory store 合约：超过声明长度和流式上限返回413语义、并发限制互不串通道、客户端中断调用一次 Abort、SHA-256准确、重复周期文件产生独立任务、stale PM按PM timeout协调而MR按MR timeout保留、PM到期只删除PM对象。

```go
func TestPMMRObjectKeysUseIsolatedPrefixes(t *testing.T) {
	at := time.Date(2026, 7, 26, 12, 0, 0, 0, time.UTC)
	for _, channel := range []string{"PM", "MR"} {
		key, err := ArtifactObjectKey("artifacts", channel, 42, at, 99)
		if err != nil { t.Fatal(err) }
		want := "artifacts/" + strings.ToLower(channel) + "/42/2026/07/26/99"
		if key != want { t.Fatalf("key = %q, want %q", key, want) }
	}
}
```

- [ ] **Step 2: 运行测试确认红灯**

Run: `cd server && env GOWORK=off go test ./plugin/tr069/service -run 'PMMR|ObjectKey|Admission|Abort|Reconcile|Retention' -count=1`

Expected: FAIL，worker仍只使用 LOG timeout 或 channel测试尚不满足。

- [ ] **Step 3: 让协调器按 channel 扫描**

把 `ListStaleReceiving` 改为显式 channel 查询，worker遍历 runtime 中 enabled channel并使用各自 timeout；对未知历史 channel使用10分钟安全默认值，但不把其当成LOG。

```go
func (s *TransferStore) ListStaleReceivingByChannel(ctx context.Context, channel string, before time.Time, limit int) ([]model.Artifact, error) {
	if limit <= 0 { limit = 100 }
	var artifacts []model.Artifact
	err := s.db.WithContext(ctx).
		Where("channel = ? AND status = ? AND created_at < ?", strings.ToUpper(channel), model.ArtifactStatusReceiving, before).
		Order("created_at ASC").Limit(limit).Find(&artifacts).Error
	return artifacts, err
}
```

`deleteExpiredArtifacts` 继续以每个制品的 `DeleteAt` 为准；事件必须沿用 artifact自己的 TaskID/FileID，禁止用当前循环 channel 重写。

- [ ] **Step 4: 运行 service 全量与竞态测试**

Run: `cd server && env GOWORK=off go test ./plugin/tr069/service -count=1`

Run: `cd server && env GOWORK=off go test -race ./plugin/tr069/service -run 'PMMR|Admission|Receiver|Worker' -count=1`

Expected: 两条命令均 PASS；race detector无数据竞争。

- [ ] **Step 5: 提交本任务**

```bash
git add server/plugin/tr069/service/artifact_store.go server/plugin/tr069/service/artifact_store_contract_test.go server/plugin/tr069/service/transfer_receiver_test.go server/plugin/tr069/service/transfer_store.go server/plugin/tr069/service/transfer_store_test.go server/plugin/tr069/service/transfer_worker.go server/plugin/tr069/service/transfer_worker_test.go
git commit -m "feat(tr069): isolate PM and MR artifact lifecycle policies"
```

### Task 5: 增加固定 channel 管理 API 与设备数据范围

**Files:**
- Create: `server/plugin/tr069/service/artifact_access_scope.go`
- Create: `server/plugin/tr069/service/artifact_access_scope_test.go`
- Modify: `server/plugin/tr069/model/request/artifact.go`
- Modify: `server/plugin/tr069/api/artifact.go`
- Modify: `server/plugin/tr069/api/artifact_test.go`
- Modify: `server/plugin/tr069/router/artifact.go`
- Create: `server/plugin/tr069/router/artifact_test.go`
- Modify: `server/plugin/tr069/plugin.go`
- Modify: `server/plugin/tr069/initialize/api.go`

**Interfaces:**
- Produces: `ArtifactAccessScope{Unrestricted bool, GroupIDs []uint}`；`ArtifactAccessScopeResolver.Resolve(ctx, authorityID)`；`ArtifactListFilter.Channel`/`GroupIDs`；`ArtifactApi.ListChannel(channel)`、`DownloadChannel(channel)`。
- Consumes: GVA JWT `utils.GetUserAuthorityId(c)`；`system.SysAuthority.DataAuthorityId`；`model.Device.GroupId`。

- [ ] **Step 1: 写 scope、channel和AVAILABLE失败测试**

测试权限888为 unrestricted；普通角色只允许自身 authority ID及 `DataAuthorityId` 对应的 `Device.GroupId`；无范围角色看不到 group 0。API测试创建 LOG/PM/MR、AVAILABLE/RECEIVING和不同group制品，断言PM列表只返回当前范围内 AVAILABLE PM，PM下载MR fileId为404，越权设备也为404且不泄露对象键。

```go
type ArtifactAccessScope struct {
	Unrestricted bool
	GroupIDs     []uint
}

type ArtifactAccessScopeResolver interface {
	Resolve(context.Context, uint) (ArtifactAccessScope, error)
}

type ArtifactListFilter struct {
	Channel      string
	GroupIDs     []uint
	Unrestricted bool
	SerialNumber string
	CreatedFrom  *time.Time
	CreatedTo    *time.Time
	Offset       int
	Limit        int
}
```

- [ ] **Step 2: 运行测试确认红灯**

Run: `cd server && env GOWORK=off go test ./plugin/tr069/service ./plugin/tr069/api ./plugin/tr069/router -run 'ArtifactAccess|PMMR|Channel|Available|Scope' -count=1`

Expected: FAIL，当前store硬编码 `LOG` 且下载不要求 channel/scope。

- [ ] **Step 3: 实现 GVA authority 到设备分组范围的解析**

`GormArtifactAccessScopeResolver` 加载当前角色及 `DataAuthorityId`，去重排序 group IDs；authority 888为管理员 unrestricted。查询层必须把 scope 条件和 `channel/status/deleting_at` 放在同一条 SQL 中，不能先查对象键再做内存权限判断。

```go
func applyArtifactScope(query *gorm.DB, scope ArtifactAccessScope) *gorm.DB {
	if scope.Unrestricted {
		return query
	}
	if len(scope.GroupIDs) == 0 {
		return query.Where("1 = 0")
	}
	return query.Where("devices.group_id IN ?", scope.GroupIDs)
}
```

`ListArtifacts` 拒绝 LOG/PM/MR之外的 channel；`GetAvailableArtifact` 改成 `(ctx, fileID, channel, scope)`，所有API固定传入服务端 route channel。

- [ ] **Step 4: 注册显式 PM/MR API 与 Casbin资源**

保留原 LOG 路由，新增以下四个资源：

```go
{Path: "/tr069/artifact/pm/list", Description: "获取基站PM文件列表", ApiGroup: "TR069", Method: "GET"},
{Path: "/tr069/artifact/pm/:fileId/download", Description: "下载基站PM文件", ApiGroup: "TR069", Method: "GET"},
{Path: "/tr069/artifact/mr/list", Description: "获取基站MR文件列表", ApiGroup: "TR069", Method: "GET"},
{Path: "/tr069/artifact/mr/:fileId/download", Description: "下载基站MR文件", ApiGroup: "TR069", Method: "GET"},
```

router使用 `ListChannel("PM")`、`DownloadChannel("PM")` 和对应MR factory；不要注册 `/tr069/command/:deviceId/pmUpload`、`mrUpload` 或泛化 `:channel` 路由。

- [ ] **Step 5: 验证 JWT/Casbin、范围和响应最小化**

Run: `cd server && env GOWORK=off go test ./plugin/tr069/service ./plugin/tr069/api ./plugin/tr069/router ./plugin/tr069/initialize -run 'Artifact|PMMR|Scope|Casbin|Menu' -count=1`

Expected: PASS；列表DTO只有 `fileId/serialNumber/source/originalName/size/receivedAt`，下载审计只有用户、设备ID、文件ID、结果和耗时。

- [ ] **Step 6: 提交本任务**

```bash
git add server/plugin/tr069/service/artifact_access_scope.go server/plugin/tr069/service/artifact_access_scope_test.go server/plugin/tr069/model/request/artifact.go server/plugin/tr069/api/artifact.go server/plugin/tr069/api/artifact_test.go server/plugin/tr069/router/artifact.go server/plugin/tr069/router/artifact_test.go server/plugin/tr069/plugin.go server/plugin/tr069/initialize/api.go
git commit -m "feat(tr069): expose scoped PM and MR artifact APIs"
```

### Task 6: 复用文件页面并增加 PM/MR 菜单

**Files:**
- Create: `web/src/plugin/tr069/api/artifact-file.js`
- Create: `web/src/plugin/tr069/view/file-artifact/artifact-file-view.js`
- Create: `web/src/plugin/tr069/view/file-artifact/artifact-file-table.vue`
- Create: `web/src/plugin/tr069/view/file-artifact/artifact-file-view.test.js`
- Create: `web/src/plugin/tr069/view/file-artifact/artifact-file.contract.test.js`
- Create: `web/src/plugin/tr069/view/pm-file/index.vue`
- Create: `web/src/plugin/tr069/view/mr-file/index.vue`
- Modify: `web/src/plugin/tr069/view/log-file/index.vue`
- Modify: `web/src/plugin/tr069/view/log-file/log-file-view.js`
- Modify: `web/src/plugin/tr069/view/log-file/log-file-view.test.js`
- Modify: `server/plugin/tr069/initialize/menu.go`
- Create: `server/plugin/tr069/initialize/menu_pm_mr_file_test.go`

**Interfaces:**
- Produces: `listArtifacts(channel, params)`、`downloadArtifact(channel, fileId)`；`ArtifactFileTable` props `channel/title/periodicOnly`。
- Consumes: Task 5 固定 PM/MR API；现有 `formatBytes`、`sourceView`、`filenameFromDisposition`、`triggerBlobDownload` 行为。

- [ ] **Step 1: 写前端和菜单失败测试**

Node contract测试读取两个 wrapper和共享组件，断言 PM/MR 常量、中文标题、分页、下载、主题变量以及不存在 `立即上传`、`upload`、`ACTIVE` 操作。Go菜单测试连续调用 `Menu` 两次并断言菜单各一条、组件路径稳定、排序为 LOG=3、PM=4、MR=5、告警=6。

```js
test('PM and MR pages bind fixed periodic channels', async () => {
  const pm = await readFile(new URL('../pm-file/index.vue', import.meta.url), 'utf8')
  const mr = await readFile(new URL('../mr-file/index.vue', import.meta.url), 'utf8')
  assert.match(pm, /channel="PM"/)
  assert.match(mr, /channel="MR"/)
  for (const source of [pm, mr]) {
    assert.match(source, /periodic-only/)
    assert.doesNotMatch(source, /立即上传|requestUpload|activeUpload/)
  }
})
```

- [ ] **Step 2: 运行测试确认红灯**

Run: `cd web && node --test src/plugin/tr069/view/file-artifact/*.test.js`

Run: `cd server && env GOWORK=off go test ./plugin/tr069/initialize -run 'PMMR|Menu' -count=1`

Expected: 两条命令 FAIL，文件和菜单尚不存在。

- [ ] **Step 3: 实现受控 API helper 和共享表格**

API helper只接受常量白名单并生成固定端点；非法 channel在发出网络请求前抛错。

```js
import service from '@/utils/request'

const artifactPaths = Object.freeze({
  LOG: '/tr069/artifact',
  PM: '/tr069/artifact/pm',
  MR: '/tr069/artifact/mr'
})

const channelPath = (channel) => {
  const value = artifactPaths[channel]
  if (!value) throw new TypeError(`Unsupported artifact channel: ${channel}`)
  return value
}

export const listArtifacts = (channel, params) => service({
  url: `${channelPath(channel)}/list`, method: 'get', params
})

export const downloadArtifact = (channel, fileId) => service({
  url: `${channelPath(channel)}/${encodeURIComponent(fileId)}/download`,
  method: 'get', responseType: 'blob', donNotShowLoading: true
})
```

共享表格保留完整序列号精确搜索、日期范围、分页和流式blob下载。`periodicOnly=true` 时来源固定显示“周期上传”，组件不渲染任何主动操作插槽。

- [ ] **Step 4: 增加薄 wrapper 和幂等菜单**

```vue
<template>
  <ArtifactFileTable channel="PM" title="PM 文件" periodic-only />
</template>

<script setup>
import ArtifactFileTable from '@/plugin/tr069/view/file-artifact/artifact-file-table.vue'
defineOptions({ name: 'Tr069PMFiles' })
</script>
```

MR wrapper同结构，固定 `channel="MR"`、`title="MR 文件"`、name `Tr069MRFiles`。LOG wrapper改用共享组件但保留 LOG现有行为和测试。菜单使用 `FirstOrCreate` 后显式 `Save`更新字段，不能依赖首次种子。

- [ ] **Step 5: 运行前端测试与构建**

Run: `cd web && node --test src/plugin/tr069/view/log-file/*.test.js src/plugin/tr069/view/file-artifact/*.test.js`

Run: `cd web && npm run build`

Run: `cd server && env GOWORK=off go test ./plugin/tr069/initialize -run 'Menu|Api' -count=1`

Expected: 全部 PASS；构建无未解析组件或 API import。

- [ ] **Step 6: 提交本任务**

```bash
git add web/src/plugin/tr069/api/artifact-file.js web/src/plugin/tr069/view/file-artifact web/src/plugin/tr069/view/pm-file web/src/plugin/tr069/view/mr-file web/src/plugin/tr069/view/log-file server/plugin/tr069/initialize/menu.go server/plugin/tr069/initialize/menu_pm_mr_file_test.go
git commit -m "feat(tr069): add PM and MR artifact management pages"
```

### Task 7: 文档、禁止下发证明与自动化回归

**Files:**
- Create: `server/plugin/tr069/docs/pm_mr_ingress.md`
- Modify: `server/plugin/tr069/api/command_upload_test.go`
- Modify: `server/plugin/tr069/router/device_command_test.go`
- Modify: `openspec/changes/enable-tr069-pm-mr-ingress/tasks.md`

**Interfaces:**
- Consumes: `/acs/pm`、`/acs/mr`；PM/MR固定 Profile和周期上传行为。
- Produces: 可执行运维手册、无主动Upload回归证明、OpenSpec任务勾选证据。

- [ ] **Step 1: 写禁止主动 PM/MR 的回归测试**

router测试枚举所有已注册路由，断言只有现有 LOG `POST /tr069/command/:deviceId/upload`，不存在PM/MR命令端点。command测试扫描提交的 `UploadRequest`，断言 API只接受 LOG file type，PM/MR输入被拒绝且不创建 Command或TransferTask。

```go
for _, forbidden := range []string{
	"/tr069/command/:deviceId/pmUpload",
	"/tr069/command/:deviceId/mrUpload",
	"/tr069/command/:deviceId/upload/pm",
	"/tr069/command/:deviceId/upload/mr",
} {
	if routes[http.MethodPost+" "+forbidden] {
		t.Fatalf("unexpected active upload route %s", forbidden)
	}
}
```

- [ ] **Step 2: 运行测试确认禁止项成立**

Run: `cd server && env GOWORK=off go test ./plugin/tr069/api ./plugin/tr069/router -run 'Upload|PMMR|PeriodicOnly' -count=1`

Expected: PASS；如果实现误加主动入口则 FAIL。

- [ ] **Step 3: 编写运维文档**

文档必须给出以下精确配置和排障顺序：

```text
PM URL: http(s)://<GVA-7458>/acs/pm
MR URL: http(s)://<GVA-7458>/acs/mr
设备侧: 人工配置各自 URL、周期、用户名和密码
GVA侧: 分别设置 PM、MR Profile；两者同时为空表示无认证
归属前提: 最近成功 Inform 的来源 IP 在 TTL 内唯一映射到一个已注册设备
401: 凭据/challenge失败
403: IP未知、过期或NAT歧义
413: 超过对应channel上限
503: 达到通道或单设备并发上限
```

明确 HTTPS、可信代理、MinIO、保留期、只周期任务、GVA不下发PM/MR参数、回滚只关闭 PM/MR channel且不删除历史制品。

- [ ] **Step 4: 运行全量自动化验证**

Run: `cd server && env GOWORK=off go test ./plugin/tr069/... -count=1`

Run: `cd server && env GOWORK=off go test -race ./plugin/tr069/service ./plugin/tr069/handler ./plugin/tr069/middleware -count=1`

Run: `cd web && rg --files src/plugin/tr069 -g '*.test.js' | xargs node --test`

Run: `cd web && npm run build`

Run: `git diff --check`

Expected: 全部退出0；日志、测试失败和构建输出中不出现配置密码或 Authorization。

- [ ] **Step 5: 更新 OpenSpec 任务证据并提交**

只勾选已经有测试或黑盒证据的 1.x—5.2；5.3、5.4保留到 Task 8 验收完成后。

```bash
git add server/plugin/tr069/docs/pm_mr_ingress.md server/plugin/tr069/api/command_upload_test.go server/plugin/tr069/router/device_command_test.go openspec/changes/enable-tr069-pm-mr-ingress/tasks.md
git commit -m "docs(tr069): document PM and MR periodic file ingress"
```

### Task 8: 两台 BS Docker 黑盒验收与可回滚启用

**Files:**
- Modify: `server/plugin/tr069/docs/pm_mr_ingress.md`
- Modify: `openspec/changes/enable-tr069-pm-mr-ingress/tasks.md`

**Interfaces:**
- Consumes: `gva-acs-bs`、`gva-acs-bs-02`；7458 `/acs/pm`、`/acs/mr`；管理端PM/MR列表和下载。
- Produces: 两台设备的成功、错误、交叉、无绑定和歧义验收记录。

- [ ] **Step 1: 启用前只读检查**

Run: `docker ps --format '{{.Names}} {{.Status}} {{.Ports}}' | rg 'gva-acs-bs|gva-acs-bs-02|1Panel-minio|1Panel-redis|1Panel-mysql'`

Run: `curl -fsS http://127.0.0.1:18888/health`

Run: `curl -sS -o /dev/null -w '%{http_code}\n' http://127.0.0.1:7458/acs/pm`

Expected: 两台BS和依赖容器运行；后端 health成功；无认证GET PM入口返回405而不是进入CWMP。

- [ ] **Step 2: 第一台 BS 正确凭据验收**

在第一台 BS OAM 界面人工设置 PM URL `/acs/pm`、MR URL `/acs/mr`、各自周期和本地凭据；在GVA分别设置相同PM/MR Profile。等待一次周期，确认数据库中该设备新增 `channel=PM/MR, source=PERIODIC, status=COMPLETED`，制品为 `AVAILABLE`，对象键分别含 `/pm/`、`/mr/`。

Run: `curl -fsS 'http://127.0.0.1:18888/tr069/artifact/pm/list?page=1&pageSize=10' -H "x-token: ${GVA_TEST_TOKEN}"`

Run: `curl -fsS 'http://127.0.0.1:18888/tr069/artifact/mr/list?page=1&pageSize=10' -H "x-token: ${GVA_TEST_TOKEN}"`

Expected: 对应列表只出现第一台设备自身 channel 的 AVAILABLE文件；响应没有 `objectKey`、`sourceIp`、`driver`、`sha256`、`password`。

- [ ] **Step 3: 第二台 BS 正确凭据和归属验收**

在第二台 BS 重复人工配置并触发周期上传。确认两台Docker不同来源IP各自唯一归属，下载两台PM/MR文件并比对长度与服务端记录的SHA-256；GVA页面没有PM/MR“立即上传”按钮。

- [ ] **Step 4: 错误、交叉和无绑定验收**

依次把第二台PM密码改错、把MR配置成PM凭据、清除测试IP绑定、制造同一IP两个有效候选；每个案例上传前记录对象和任务数，上传后分别确认401、401、403、403，且计数不增加。恢复正确配置后下一周期成功，正式验证状态更新到当前 revision。

- [ ] **Step 5: 检查秘密与完成任务清单**

Run: `rg -n 'Authorization:|Basic [A-Za-z0-9+/=]{12,}|response="[0-9a-f]+"|<Password>[^<]+|"password"\s*:\s*"[^\"]+"' /tmp/gva-backend.log server/log -g '*.log'`

Expected: 无匹配；如有固定脱敏占位符，人工确认其不是明文、密文或可重放token。

将 OpenSpec 5.3、5.4 勾选并在文档记录日期、设备序列号、来源IP、channel、HTTP状态、任务ID和fileId；不得记录用户名、密码、Authorization或Digest response。

- [ ] **Step 6: 提交验收证据**

```bash
git add server/plugin/tr069/docs/pm_mr_ingress.md openspec/changes/enable-tr069-pm-mr-ingress/tasks.md
git commit -m "test(tr069): verify PM and MR ingress on both base stations"
```

## 最终完成标准

- `/acs/pm`、`/acs/mr` 接受 raw PUT、普通POST和兼容 multipart POST，非法方法/多段后缀拒绝且不创建任务。
- PM/MR分别只认自己的 Profile；空 Profile为无认证；错误或跨模块凭据在读正文前401。
- 唯一近期Inform IP可以归属；未知、过期、NAT歧义403且不猜测设备。
- PM/MR永远创建 PERIODIC任务和AVAILABLE制品，不存在主动命令或WAITING_FILE。
- channel限长、并发、超时、SHA-256、Abort、协调、保留期和对象前缀互不串扰。
- 管理API与页面固定 channel、复核权限/范围/AVAILABLE，且PM/MR页面没有立即上传。
- GVA没有创建任何设置PM/MR URL、凭据、周期或启用参数的SPV命令。
- 两台BS黑盒验收与全量Go、race、Node测试、Vue构建、秘密扫描均通过。
