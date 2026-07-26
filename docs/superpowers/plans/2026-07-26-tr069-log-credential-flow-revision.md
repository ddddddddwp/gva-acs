---
change: add-tr069-log-collection
design-doc: docs/superpowers/specs/2026-07-26-tr069-log-credential-flow-revision-design.md
base-ref: bb89d5f2f98f861d62f9720b5468e5e9380b6104
---

# TR-069 LOG 凭据流程修订实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在不重写既有 LOG 流式接收、对象存储和传输状态机的前提下，让 `/acs/log` 使用全局 LOG Profile 校验，并让主动 Upload RPC 永远携带空用户名和空密码，同时补齐现有 change 尚未完成的管理、前端、集成和双基站验收。

**Architecture:** 本计划依赖 `secure-tr069-credential-authentication` 先提供 `auth.CredentialProvider`、唯一的 Basic/Digest `auth.Authenticator`、`auth.Principal` 和设备认证状态记录器。LOG 文件入口只保留一层薄适配：认证后仍按 Inform/IP 唯一解析设备，随后记录 `LOG_UPLOAD` 当前 revision 并复用现有 `TransferReceiver`。主动 Upload 直接持久化固定 URL 与空凭据，删除 LOG payload hydrate，避免任何文件密码进入命令链路。

**Tech Stack:** Go、Gin、GORM、SQLite、Redis/miniredis、MinIO/S3、Vue 3、Element Plus、Node.js test runner、Docker BS 模拟器。

## Global Constraints

- 本计划从 `bb89d5f2f98f861d62f9720b5468e5e9380b6104` 开始审阅差异，只处理 OpenSpec 未完成的 `8.1`、`8.3`、`9.1`、`9.4`、`10.2`、`10.4`、`10.5` 和 `11.1`—`11.6`。
- 保留已经完成的 `/acs/log` 路由、Inform/IP 唯一解析、64 KiB 缓冲流式接收、MinIO、任务/制品/事件、ACTIVE/PERIODIC 状态机、协调器和保留期；不得重新实现这些组件。
- LOG 上传地址固定为 `/acs/log`；不得加入设备标识，也不得在本 change 修改 `/acs/pm` 或 `/acs/mr`。
- LOG Profile 用户名和密码同时为空时入口无认证可用；同时非空时支持 Basic/Digest；半配置由全局凭据服务原子拒绝并保留旧 revision。
- 主动 LOG Upload 的持久化和线上 RPC 都必须为 `Username=""`、`Password=""`；BS 后续 HTTP 上传使用用户在 `Device.LogMgmt.Username/Password` 中配置的本地凭据。
- YAML `fileIngress.authentication.username/password` 不再是认证事实来源；YAML 仅保留入口开关、路径、realm/nonce 策略、限制和对象存储配置。
- 认证失败发生在设备解析之前，只能记录来源 IP；只有认证成功且设备唯一解析成功后才能记录设备 `LOG_UPLOAD` revision。
- 密码、Authorization、对象键和 MinIO 凭据不得进入 API、命令 JSON/XML、RawDump、Trace、操作日志或测试失败输出。
- 当前仓库没有每设备所有权字段，因此本 change 的设备数据范围以 TR-069 JWT/Casbin 路由授权为边界；不得为了 LOG 下载虚构新的设备所有权表。

## 文件结构

- `server/plugin/tr069/initialize/server.go`：组装共享认证核心的 LOG 薄适配和认证成功状态记录器。
- `server/plugin/tr069/middleware/file_auth_adapter.go`：把 `auth.Authenticator` 的 `LOG_UPLOAD` principal 映射为传输通道 `LOG`；协议算法仍只有 `auth` 包一份。
- `server/plugin/tr069/handler/file_ingress.go`：在设备唯一解析后记录当前 principal revision，再进入既有接收器。
- `server/plugin/tr069/api/command.go`：创建固定 URL、空 Username/Password 的主动 LOG Upload。
- `server/plugin/tr069/adapter/log_upload_payload.go`：删除；命令不再保护或 hydrate LOG 密码。
- `server/plugin/tr069/api/artifact.go`、`router/artifact.go`、`middleware/download_audit.go`：只补齐未完成的权限、失败和流式下载契约。
- `web/src/plugin/tr069/api/log-file.js`、`view/log-file/*`：只补测试和错误/权限展示缺口，不重做页面。
- `server/plugin/tr069/handler/log_ingress_integration_test.go`：可选 MinIO 集成客户端，覆盖 Basic/Digest、绑定、存储、列表和下载。
- `server/plugin/tr069/docs/log_collection.md`：修正凭据来源、主动空凭据与双 BS 验收说明。

---

### Task 1: 将 LOG 入口接到全局 LOG_UPLOAD Profile

**OpenSpec:** 11.1、11.2、11.6 的 revision 状态部分。

**Files:**
- Create: `server/plugin/tr069/middleware/file_auth_adapter.go`
- Create: `server/plugin/tr069/middleware/file_auth_adapter_test.go`
- Modify: `server/plugin/tr069/handler/file_ingress.go`
- Modify: `server/plugin/tr069/handler/file_ingress_test.go`
- Modify: `server/plugin/tr069/initialize/server.go`
- Modify: `server/plugin/tr069/initialize/server_runtime_test.go`
- Delete after migration: YAML-provider portions of `server/plugin/tr069/middleware/file_auth.go`
- Delete after migration: YAML-provider test `TestRuntimeFileCredentialProviderMatchesVendorPathVariants` from `server/plugin/tr069/middleware/file_auth_test.go`

**Interfaces:**
- Consumes: `auth.ChannelLogUpload`, `auth.Principal`, `auth.Authenticator.Authenticate(*http.Request, auth.Channel) (auth.Principal, []string, error)` from the secure change.
- Consumes: `auth.SuccessRecorder.RecordSuccess(context.Context, uint, auth.Principal) error`, constructed by `adapter.NewGormAuthStateRecorder(global.GVA_DB)`.
- Produces: `middleware.FileAuthAdapter.Authenticate(*http.Request) (auth.Principal, string, []string, error)`; the string is exactly `LOG`.

- [ ] **Step 1: 写适配器失败测试**

```go
func TestFileAuthAdapterMapsLogPrincipalWithoutLosingRevision(t *testing.T) {
	principal := auth.Principal{Channel: auth.ChannelLogUpload, ProfileID: 2, Revision: 7, Scheme: "digest"}
	core := fakeCoreAuthenticator{principal: principal}
	adapter := FileAuthAdapter{Core: core, Channel: auth.ChannelLogUpload, TransferChannel: "LOG"}
	req := httptest.NewRequest(http.MethodPut, "/acs/log", nil)
	got, channel, challenges, err := adapter.Authenticate(req)
	if err != nil || got != principal || channel != "LOG" || len(challenges) != 0 {
		t.Fatalf("principal=%#v channel=%q challenges=%#v err=%v", got, channel, challenges, err)
	}
}

func TestFileAuthAdapterAllowsDisabledLogProfile(t *testing.T) {
	principal := auth.Principal{Channel: auth.ChannelLogUpload, ProfileID: 2, Revision: 8, Disabled: true}
	adapter := FileAuthAdapter{Core: fakeCoreAuthenticator{principal: principal}, Channel: auth.ChannelLogUpload, TransferChannel: "LOG"}
	got, channel, challenges, err := adapter.Authenticate(httptest.NewRequest(http.MethodPost, "/acs/log", nil))
	if err != nil || !got.Disabled || channel != "LOG" || len(challenges) != 0 {
		t.Fatalf("principal=%#v channel=%q challenges=%#v err=%v", got, channel, challenges, err)
	}
}
```

- [ ] **Step 2: 运行测试确认旧 YAML Provider/旧签名无法满足契约**

Run: `cd server && go test ./plugin/tr069/middleware ./plugin/tr069/handler -run 'FileAuthAdapter|FileIngressRecordsAuth' -count=1`

Expected: FAIL，提示 `FileAuthAdapter` 或新认证返回签名不存在。

- [ ] **Step 3: 实现薄适配与成功记录端口**

```go
type FileAuthAdapter struct {
	Core            auth.Authenticator
	Channel         auth.Channel
	TransferChannel string
}

func (a FileAuthAdapter) Authenticate(r *http.Request) (auth.Principal, string, []string, error) {
	if a.Core == nil || a.Channel == "" || strings.TrimSpace(a.TransferChannel) == "" {
		return auth.Principal{}, "", nil, ErrFileAuthConfiguration
	}
	principal, challenges, err := a.Core.Authenticate(r, a.Channel)
	return principal, a.TransferChannel, challenges, err
}
```

将 handler 端口改为：

```go
type FileRequestAuthenticator interface {
	Authenticate(*http.Request) (auth.Principal, string, []string, error)
}

type FileAuthSuccessRecorder interface {
	RecordSuccess(context.Context, uint, auth.Principal) error
}
```

在 `resolver.Resolve` 成功后、解析/写入文件正文之前执行：

```go
if authStates != nil && !principal.Disabled {
	if err := authStates.RecordSuccess(c.Request.Context(), device.DeviceID, principal); err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}
}
```

这保证状态写入失败时尚未创建对象或读取文件正文；无认证 principal 保持“未启用”，不伪造成功状态。

- [ ] **Step 4: 在运行时只装配全局 Profile Provider**

```go
fileAuth := middleware.FileAuthAdapter{
	Core: initialize.CurrentHTTPAuthenticator(), Channel: auth.ChannelLogUpload, TransferChannel: "LOG",
}
authStates := initialize.CurrentAuthSuccessRecorder()
ingressHandler := handler.NewFileIngressHandler(fileAuth, authStates, deviceResolver, receiver, handler.RuntimeFileIngressChannelProvider{})
```

复用 secure change 已装配的共享认证核心和成功状态记录器，不得在 LOG 路由中另建第二套 runtime。删除 `RuntimeFileCredentialProvider`；不要从 `runtime.Settings.FileIngress.Authentication.Username/Password` 读取认证值。realm、schemes 和 nonce TTL 由共享认证核心及通道运行配置提供。

- [ ] **Step 5: 添加 LOG 集成断言**

在 `file_ingress_test.go` 使用 fake principal/recorder，证明：错误认证不调用 resolver/recorder/receiver；成功认证只在唯一设备解析后记录 `DeviceID + ChannelLogUpload + Revision`；disabled principal 正常接收但不调用 recorder；recorder 失败返回 500 且 body read count 为 0。

- [ ] **Step 6: 运行并提交**

Run: `cd server && go test ./plugin/tr069/auth ./plugin/tr069/middleware ./plugin/tr069/handler ./plugin/tr069/initialize -count=1`

Expected: PASS。

```bash
git add server/plugin/tr069/middleware server/plugin/tr069/handler server/plugin/tr069/initialize/server.go server/plugin/tr069/initialize/server_runtime_test.go
git commit -m "refactor(tr069): use global log upload credentials"
```

### Task 2: 主动 Upload 固定为空凭据并删除 hydrate

**OpenSpec:** 11.3、11.4。

**Files:**
- Modify: `server/plugin/tr069/api/command.go`
- Modify: `server/plugin/tr069/api/command_upload_test.go`
- Modify: `server/plugin/tr069/adapter/redis_command_source_test.go`
- Modify: `server/plugin/tr069/adapter/command_xml_sink_test.go`
- Delete: `server/plugin/tr069/adapter/log_upload_payload.go`
- Delete: `server/plugin/tr069/adapter/log_upload_payload_test.go`

**Interfaces:**
- Consumes: YAML `fileIngress.enabled`、`publicBaseURL`、LOG channel `enabled/path` only.
- Produces: persisted and pulled `request.UploadRequest{FileType, URL, Username:"", Password:"", DelaySeconds}`.

- [ ] **Step 1: 把现有 API 测试改为明确的空值失败测试**

```go
decoded, err := service.DecodeRPCRequest("Upload", command.ParamsJSON)
if err != nil {
	t.Fatalf("decode Upload: %v", err)
}
upload := decoded.(req.UploadRequest)
if upload.URL != "http://gva:7458/acs/log" || upload.Username != "" || upload.Password != "" {
	t.Fatalf("persisted Upload = %#v", upload)
}
for _, forbidden := range []string{"configured-log-user", "configured-log-password", "__GVA_TR069_LOG_UPLOAD_"} {
	if strings.Contains(string(command.ParamsJSON), forbidden) {
		t.Fatalf("persisted Upload leaked %q: %s", forbidden, command.ParamsJSON)
	}
}
```

测试配置故意不再填写 YAML authentication credentials，仍应成功创建唯一 ACTIVE `WAITING_FILE` 任务。

- [ ] **Step 2: 运行测试确认当前 API 仍要求并持久化凭据占位符**

Run: `cd server && go test ./plugin/tr069/api ./plugin/tr069/adapter -run 'Upload|LogUpload' -count=1`

Expected: FAIL，当前实现返回“配置不完整”或出现 LOG placeholder。

- [ ] **Step 3: 最小化 Upload API**

```go
if !runtime.Enabled || !ok || !logChannel.Enabled || strings.TrimSpace(runtime.PublicBaseURL) == "" {
	response.FailWithMessage("LOG 文件入口未启用或配置不完整", c)
	return
}
submitCommand(c, "Upload", req.UploadRequest{
	FileType: in.FileType,
	URL: strings.TrimRight(runtime.PublicBaseURL, "/") + logChannel.Path,
	Username: "",
	Password: "",
	DelaySeconds: in.DelaySeconds,
})
```

从 `commandPayloadProtector` 删除 `adapter.LogUploadPayloadCodec{}`，删除 codec 源文件及测试；Connection 密码 protector 保持不变。

- [ ] **Step 4: 验证 Pull/重试不会重新 hydrate 文件凭据**

在 `redis_command_source_test.go` 种子一个空凭据 Upload 命令，调用 `Pull` 后断言：

```go
if pulled.Params["url"] != "http://gva:7458/acs/log" || pulled.Params["username"] != "" || pulled.Params["password"] != "" {
	t.Fatalf("pulled Upload params = %#v", pulled.Params)
}
```

在 `command_xml_sink_test.go` 保存包含空 `<Username></Username><Password></Password>` 的 Upload XML，断言数据库 XML 不包含本次测试 Profile 密码 `log-profile-secret-never-on-wire`。

- [ ] **Step 5: 运行秘密扫描测试并提交**

Run: `cd server && go test ./plugin/tr069/api ./plugin/tr069/adapter ./plugin/tr069/middleware ./plugin/tr069/redact ./plugin/tr069/trace -run 'Upload|CommandXML|Raw|Trace|Redact' -count=1`

Expected: PASS；命令 JSON、Pull 参数和 XML 均只有空文件凭据。

```bash
git add server/plugin/tr069/api/command.go server/plugin/tr069/api/command_upload_test.go server/plugin/tr069/adapter server/plugin/tr069/middleware server/plugin/tr069/redact server/plugin/tr069/trace
git commit -m "fix(tr069): omit credentials from active log upload"
```

### Task 3: 补齐制品列表和受保护下载契约

**OpenSpec:** 8.1、8.3。

**Files:**
- Modify: `server/plugin/tr069/api/artifact_test.go`
- Modify only if a test exposes a gap: `server/plugin/tr069/api/artifact.go`
- Create: `server/plugin/tr069/router/artifact_test.go`
- Modify: `server/plugin/tr069/middleware/download_audit_test.go`

**Interfaces:**
- Produces: `GET /tr069/artifact/list` only returns AVAILABLE LOG DTOs and exact `serialNumber` matches.
- Produces: `GET /tr069/artifact/:fileId/download` re-reads AVAILABLE state, streams with a 64 KiB pooled buffer and records metadata-only audit.

- [ ] **Step 1: 扩展表驱动 API 测试**

为列表增加 `page=0` 默认页、`pageSize=1000` 截断为 100、起止时间非法、精确 serialNumber、删除中设备、RECEIVING/FAILED/DELETED 制品不可见、DTO 不含 `objectKey/driver/sourceIp/deviceId/oui/status/sha256`。为下载增加零/非法/不存在 ID、不可用状态、对象缺失、store 不可用、安全 `Content-Disposition` 和 reader 取消。

核心断言使用：

```go
for _, forbidden := range []string{"objectKey", "driver", "sourceIp", "deviceId", "oui", "status", "sha256", "password"} {
	if strings.Contains(recorder.Body.String(), forbidden) {
		t.Fatalf("response leaked %q: %s", forbidden, recorder.Body.String())
	}
}
```

- [ ] **Step 2: 添加路由授权边界测试**

在 `router/artifact_test.go` 将 `ArtifactRouter` 挂载到与生产相同的 `/tr069` group，并用认证/授权 sentinel 证明未通过 JWT 或 Casbin 时 handler 不执行、响应不含制品元数据；同时断言 `initialize.Api` 注册的两个 method/path 与生产路由完全一致。不要新增 per-device ownership 表。

- [ ] **Step 3: 运行测试并只修复暴露的缺口**

Run: `cd server && go test ./plugin/tr069/api ./plugin/tr069/router ./plugin/tr069/middleware ./plugin/tr069/initialize -run 'Artifact|DownloadAudit' -count=1`

Expected: 初次运行至少覆盖新增分支；若已有代码全部满足则直接 PASS，不重写 `ArtifactApi`。

- [ ] **Step 4: 验证 20 MiB 下载审计不缓存正文**

保留现有 `TestDownloadAuditDoesNotBufferStreamedResponse`，再断言审计 `Body` 只含 `fileId/deviceId`、`Resp==""`、用户 ID 正确、失败下载记录非 2xx 状态且错误字符串不含对象键。

- [ ] **Step 5: 提交管理 API 收尾**

Run: `cd server && go test ./plugin/tr069/api ./plugin/tr069/router ./plugin/tr069/middleware ./plugin/tr069/initialize -count=1`

```bash
git add server/plugin/tr069/api/artifact* server/plugin/tr069/router/artifact* server/plugin/tr069/middleware/download_audit* server/plugin/tr069/initialize
git commit -m "test(tr069): complete protected log artifact contracts"
```

### Task 4: 补齐日志文件页面 API 与组件测试

**OpenSpec:** 9.1、9.4。

**Files:**
- Create: `web/src/plugin/tr069/api/log-file.contract.test.js`
- Modify: `web/src/plugin/tr069/view/log-file/log-file-view.test.js`
- Modify: `web/src/plugin/tr069/view/log-file/log-file.contract.test.js`
- Modify only if tests expose a gap: `web/src/plugin/tr069/api/log-file.js`
- Modify only if tests expose a gap: `web/src/plugin/tr069/view/log-file/index.vue`

**Interfaces:**
- `getLogArtifactList(params)` sends GET query parameters unchanged.
- `downloadLogArtifact(fileId)` encodes numeric ID, uses `responseType:'blob'`, and relies on the shared request client for JWT/error handling.

- [ ] **Step 1: 添加前端 API 源契约**

```js
test('log artifact API uses protected backend endpoints', async () => {
  const source = await readFile(new URL('./log-file.js', import.meta.url), 'utf8')
  assert.match(source, /url:\s*['"]\/tr069\/artifact\/list['"]/)
  assert.match(source, /method:\s*['"]get['"]/)
  assert.match(source, /encodeURIComponent\(fileId\)/)
  assert.match(source, /responseType:\s*['"]blob['"]/)
  assert.doesNotMatch(source, /minio|objectKey|accessKey|secretKey/i)
})
```

- [ ] **Step 2: 扩展页面契约**

断言完整字母数字 `serialNumber`、分页/重置、Blob 下载、后端非零 code 和异常均调用 `ElMessage.error`、不可下载行禁用、object URL 始终 revoke、没有 MinIO URL、没有固定亮色背景，并使用 `var(--el-text-color-secondary)`。

- [ ] **Step 3: 运行测试并最小修复**

Run: `cd web && node --test src/plugin/tr069/api/log-file.contract.test.js src/plugin/tr069/view/log-file/*.test.js`

Expected: PASS；若失败，只修改对应 wrapper/helper/template，不改菜单、路由或已完成列表布局。

- [ ] **Step 4: 生产构建并提交**

Run: `cd web && npm run build`

Expected: production build PASS，`pathInfo.json` 仍解析日志文件页面。

```bash
git add web/src/plugin/tr069/api/log-file* web/src/plugin/tr069/view/log-file
git commit -m "test(tr069): complete log file page contracts"
```

### Task 5: 增加 HTTP/MinIO 端到端客户端

**OpenSpec:** 10.2。

**Files:**
- Create: `server/plugin/tr069/handler/log_ingress_integration_test.go`
- Modify: `server/plugin/tr069/adapter/minio_integration_test.go` only to share environment helpers if needed.

**Interfaces:**
- Test opt-in: `TR069_MINIO_INTEGRATION=1`.
- MinIO defaults: endpoint `127.0.0.1:19000`, bucket `gva-tr069-artifacts-test`.

- [ ] **Step 1: 建立真实组合测试**

测试使用 SQLite 迁移 Device/Transfer/Artifact/Auth 状态表、miniredis 保存 Inform/IP 绑定和 Digest nonce、真实 MinIO `ArtifactStore`、真实 `UploadDeviceResolver`/`TransferReceiver`/`ArtifactApi`，由 `httptest.Server` 暴露 `/acs/log` 与管理列表/下载路由。

固定字节：

```go
basicPayload := bytes.Repeat([]byte("basic-log-block\n"), 4096)
digestPayload := bytes.Repeat([]byte("digest-log-block\n"), 4096)
```

- [ ] **Step 2: 覆盖完整协议矩阵**

依次验证：无 Authorization 得到 401 challenge 且 body 未读取；Basic PUT 201；Digest POST 首次 401、携带 nonce 后 201；MinIO metadata `file-id` 与数据库自增 ID一致；精确 serialNumber 列出两个 AVAILABLE 制品；下载 SHA-256 与输入一致；重复相同 ACTIVE 内容不新增制品；错误密码 401；未知/歧义 IP 403；DELETE 405 且 `Allow: PUT, POST`。

- [ ] **Step 3: 运行 opt-in 集成测试**

Run:

```bash
cd server
TR069_MINIO_INTEGRATION=1 \
TR069_MINIO_ENDPOINT=127.0.0.1:19000 \
TR069_MINIO_BUCKET=gva-tr069-artifacts-test \
go test ./plugin/tr069/handler ./plugin/tr069/adapter -run 'LogIngressIntegration|MinioArtifactStoreIntegration' -count=1
```

Expected: PASS；测试用自己的对象前缀并在 `t.Cleanup` 删除对象，不触碰开发 LOG 数据。

- [ ] **Step 4: 提交**

```bash
git add server/plugin/tr069/handler/log_ingress_integration_test.go server/plugin/tr069/adapter/minio_integration_test.go
git commit -m "test(tr069): cover log ingress end to end"
```

### Task 6: 固定缓冲、race 和 static 验证

**OpenSpec:** 10.5。

**Files:**
- Modify: `server/plugin/tr069/service/transfer_receiver_test.go`

**Interfaces:**
- 64 MiB 输入不允许整文件 slice；单次写入不得超过 receiver 的 `64 * 1024` buffer。

- [ ] **Step 1: 添加不保存正文的 64 MiB 流式测试**

实现测试 writer 只累计 `total` 和 `maxWrite`，输入使用 `io.LimitReader(zeroReader{}, 64<<20)`；不要使用 `bytes.Repeat(64<<20)`。断言：

```go
if writer.total != 64<<20 || writer.maxWrite > 64*1024 {
	t.Fatalf("stream total=%d maxWrite=%d", writer.total, writer.maxWrite)
}
```

同时确认 artifact 可用、SHA-256 长度 64、测试日志中没有正文片段。

- [ ] **Step 2: 运行 race/static 回归**

Run:

```bash
cd server
go test ./plugin/tr069/...
go test -race ./plugin/tr069/service ./plugin/tr069/handler ./plugin/tr069/middleware ./plugin/tr069/adapter
go vet ./plugin/tr069/...
```

Expected: all PASS，无 race、vet 或大块分配测试失败。

- [ ] **Step 3: 前端总回归并提交**

Run:

```bash
cd web
rg --files src/plugin/tr069 -g '*.test.js' | sort | xargs node --test
npm run build
```

Expected: all PASS。

```bash
git add server/plugin/tr069/service/transfer_receiver_test.go
git commit -m "test(tr069): prove bounded log upload streaming"
```

### Task 7: 两台 Docker BS 主动与周期上传验收

**OpenSpec:** 10.4、11.5、11.6。

**Files:**
- Modify: `server/plugin/tr069/docs/log_collection.md`
- Modify after evidence: `openspec/changes/add-tr069-log-collection/tasks.md`

**Interfaces:**
- BS 1: `gva-acs-bs`，上传来源 IP `172.17.0.2`。
- BS 2: `gva-acs-bs-02`，上传来源 IP `172.17.0.3`。
- 固定 URL: `http://host.docker.internal:7458/acs/log`。

- [ ] **Step 1: 记录验收前状态且不打印密码**

Run:

```bash
docker inspect -f '{{.Name}} {{.State.Status}} {{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' gva-acs-bs gva-acs-bs-02
docker exec gva-acs-bs sh -lc 'pgrep -f "oamProcess|odsNameServer|upapp|m2m.x86.bs"'
docker exec gva-acs-bs-02 sh -lc 'pgrep -f "oamProcess|odsNameServer|upapp|m2m.x86.bs"'
```

Expected: 两个容器 running，各有四个必需进程，IP 分别唯一。

- [ ] **Step 2: 在 GVA 和两台 BS 人工配置同一组临时 LOG Profile**

在 GVA “TR-069 凭据”页面设置 LOG 用户名 `gva-log-e2e` 和一次性强密码；在两台 BS 的 `Device.LogMgmt.Username/Password` 填相同值，并保持 URL `/acs/log`。密码只在 UI 输入，不复制到终端、文档或截图；保存后记录 GVA 返回的 LOG revision。

- [ ] **Step 3: 验证两台 BS 的主动空凭据 Upload**

在 GVA 分别对两台设备点击“立即上传 LOG”。每台都验证：命令 `ParamsJSON` 解码后 URL 为固定 `/acs/log`、username/password 为空；BS 使用本地凭据上传；HTTP 201；唯一 ACTIVE 任务完成；artifact 分别归属正确 SerialNumber；`LOG_UPLOAD` 状态使用当前 revision 成功。

- [ ] **Step 4: 验证两台 BS 的周期上传与轮换**

等待两台 BS 各自周期上传，确认创建 PERIODIC 任务且按 `172.17.0.2`/`172.17.0.3` 唯一归属。随后只轮换 GVA LOG Profile：两个设备状态都变待验证；再人工更新 BS 本地密码并等待上传，状态恢复当前 revision 成功。错误密码阶段应 401、不创建 artifact、不把未认证请求关联为正式设备失败。

- [ ] **Step 5: 扫描数据库和运行日志**

使用一次性密码的 SHA-256 指纹或应用内秘密扫描测试核对命令 JSON/XML、Trace、操作日志和后端日志均无明文；不要把密码本身作为 `rg` 命令参数，以免进入 shell history。确认旧 YAML 文件凭据和 `__GVA_TR069_LOG_UPLOAD_*` placeholder 不再出现在新命令。

- [ ] **Step 6: 更新运维文档和 OpenSpec 证据**

将文档改为：凭据来自数据库 LOG Profile；空 Profile 为无认证；主动 RPC 凭据为空；周期/主动 HTTP 均由 BS 本地 `Device.LogMgmt.*` 提供；固定 `/acs/log` 和 NAT/唯一 IP 限制保持不变。只有上述证据齐全后勾选 `10.2`、`10.4`、`10.5`、`11.1`—`11.6` 以及已完成的 `8.1/8.3/9.1/9.4`。

- [ ] **Step 7: 最终验证与提交**

Run:

```bash
git diff --check
git status --short
git diff bb89d5f2f98f861d62f9720b5468e5e9380b6104 -- openspec/changes/add-tr069-log-collection server/plugin/tr069 web/src/plugin/tr069
```

Expected: 无空白错误；差异只包含本计划未完成任务和凭据流程修订，没有重写已完成 LOG 实现。

```bash
git add server/plugin/tr069/docs/log_collection.md openspec/changes/add-tr069-log-collection/tasks.md
git commit -m "docs(tr069): verify local credential log uploads"
```

## 完成定义

- `openspec/changes/add-tr069-log-collection/tasks.md` 的原未完成项与第 11 组均有自动化或双 BS 证据。
- 两台 BS 的主动 Upload 命令 JSON/线上 XML都为空 Username/Password，随后上传仍通过全局 LOG Profile 校验并完成 ACTIVE 任务。
- 两台 BS 的周期上传形成 PERIODIC 任务，来源 IP 唯一归属正确，LOG revision 状态更新正确。
- 关闭 LOG Profile 后 `/acs/log` 无认证可用；半配置不能破坏旧 Profile；错误认证不读取/保存正文。
- 后端、race、vet、前端 contract/build、MinIO opt-in 集成全部通过，且秘密扫描无明文。
