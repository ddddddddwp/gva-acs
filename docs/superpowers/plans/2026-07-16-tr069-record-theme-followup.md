# TR-069 Theme and RPC Record Follow-up Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the TR-069 device UI follow GVA light/dark themes and add an independently permissioned RPC record page that exposes each command's lifecycle, result, fault, and exact bidirectional XML.

**Architecture:** Keep `tr069_commands`, `tr069_command_events`, `tr069_command_xmls`, and `CommandStore` as the database-backed source of truth. Add thin typed record APIs and response DTOs, then build a GVA-style list plus detail drawer that refreshes from those APIs every five seconds while visible. Do not add a second state machine or synchronously wait for RPC completion in submission dialogs.

**Tech Stack:** Go, Gin, GORM, MySQL/SQLite tests, Vue 3, Element Plus, Node test runner, GVA theme utility classes.

## Global Constraints

- Work directly on `dev`; do not create a feature branch or subagent.
- Preserve the existing parameter tree behavior.
- Use GVA JWT/Casbin for all record routes and `OperationRecord` only for retry.
- The list response must not expose structured request credentials or XML; those are detail-permission data.
- XML detail must return the exact UTF-8 payload, not Go `[]byte` base64 JSON.
- Never automatically run Reboot, DeleteObject, or FactoryReset against the real BS.
- The record page auto-refresh interval is 5 seconds and only refreshes while the browser page is visible.

---

### Task 1: Fix device page theme compatibility

**Files:**
- Modify: `web/src/plugin/tr069/view/device/index.vue`
- Create: `web/src/plugin/tr069/view/device/device-theme.contract.test.js`

**Interfaces:**
- Consumes: global `.gva-search-box` and `.gva-table-box` classes from `web/src/style/main.scss`.
- Produces: a device page with no hard-coded light background.

- [ ] **Step 1: Write the failing theme contract**

```js
test('device page delegates light and dark surfaces to GVA theme classes', async () => {
  const source = await readFile(new URL('./index.vue', import.meta.url), 'utf8')
  assert.match(source, /class="gva-search-box"/)
  assert.match(source, /class="gva-table-box"/)
  assert.doesNotMatch(source, /background-color:\s*#fff/i)
})
```

- [ ] **Step 2: Run the test and verify RED**

Run: `node --test src/plugin/tr069/view/device/device-theme.contract.test.js`

Expected: FAIL because the page uses `search-box`, `table-box`, and `#fff`.

- [ ] **Step 3: Use GVA theme surfaces**

Replace the page wrappers with `gva-search-box` and `gva-table-box`, keep only layout-specific scoped CSS, and remove both `background-color: #fff` declarations.

- [ ] **Step 4: Run the theme test and existing device tests**

Run: `node --test src/plugin/tr069/view/device/device-theme.contract.test.js src/plugin/tr069/view/device/device-rpc-actions.contract.test.js src/plugin/tr069/view/device/components/data-model-viewer.contract.test.js`

Expected: PASS.

---

### Task 2: Expose safe command-record list, detail, and retry APIs

**Files:**
- Create: `server/plugin/tr069/model/request/command_record.go`
- Create: `server/plugin/tr069/model/response/command_record.go`
- Create: `server/plugin/tr069/api/command_record.go`
- Create: `server/plugin/tr069/api/command_record_test.go`
- Modify: `server/plugin/tr069/router/device.go`
- Modify: `server/plugin/tr069/router/device_command_test.go`
- Modify: `server/plugin/tr069/initialize/api.go`
- Modify: `server/plugin/tr069/initialize/menu.go`

**Interfaces:**
- Consumes: `CommandStore.List(context.Context, CommandListFilter)`, `CommandStore.Detail(context.Context, string)`, and `CommandService.Retry(context.Context, string)`.
- Produces:
  - `GET /tr069/command-record/list`
  - `GET /tr069/command-record/:commandId`
  - `POST /tr069/command-record/:commandId/retry`

- [ ] **Step 1: Write failing API and route tests**

The tests seed one failed command, events, and outbound/inbound XML in SQLite and assert:

```go
if strings.Contains(listBody, "password") || strings.Contains(listBody, "<cwmp:") {
    t.Fatal("list leaked detail-only content")
}
if !strings.Contains(detailBody, `"xml":"<cwmp:GetParameterValues`) {
    t.Fatal("detail did not return exact XML text")
}
```

Extend the route contract with the exact three record paths and methods.

- [ ] **Step 2: Run tests and verify RED**

Run: `go test ./plugin/tr069/api ./plugin/tr069/router -run 'CommandRecord|TypedCommandRoutes' -count=1`

Expected: FAIL because the record API types and routes do not exist.

- [ ] **Step 3: Add typed request and response models**

```go
type CommandRecordListRequest struct {
    Page         int    `form:"page"`
    PageSize     int    `form:"pageSize"`
    DeviceID     uint   `form:"deviceId"`
    DeviceSerial string `form:"deviceSerial"`
    Operation    string `form:"operation"`
    Status       string `form:"status"`
    CommandID    string `form:"commandId"`
    CreatedFrom  string `form:"createdFrom"`
    CreatedTo    string `form:"createdTo"`
}

type CommandXMLResponse struct {
    ID        uint64    `json:"id"`
    Direction string    `json:"direction"`
    Method    string    `json:"method"`
    CWMPID    string    `json:"cwmpId"`
    RequestID string    `json:"requestId"`
    XML       string    `json:"xml"`
    CreatedAt time.Time `json:"createdAt"`
}
```

Add a summary DTO containing identifiers, operation, status, deadlines, timestamps, failure fields, and retry lineage, but no params/result/XML.

- [ ] **Step 4: Implement list, detail, and retry handlers**

List binds filters, parses `YYYY-MM-DD HH:mm:ss`, converts page/pageSize to offset/limit, and returns `response.PageResult`. Detail converts every `CommandXML.Payload` using `string(payload)`. Retry calls the existing `commandService.Retry` and returns `{commandId,status}`.

- [ ] **Step 5: Register routes, GVA APIs, and menu**

```go
recordRouter := Router.Group("command-record")
recordRouter.GET("list", recordApi.List)
recordRouter.GET(":commandId", recordApi.Detail)
recordRouter.POST(":commandId/retry", middleware.OperationRecord(), recordApi.Retry)
```

Register three `SysApi` rows. Add `tr069CommandRecord` under the TR069 parent with component `plugin/tr069/view/command-record/index.vue`, and move the alarm group to the next sort value.

- [ ] **Step 6: Run backend tests**

Run: `go test ./plugin/tr069/api ./plugin/tr069/router ./plugin/tr069/initialize ./plugin/tr069/service -count=1`

Expected: PASS.

---

### Task 3: Build the theme-aware RPC record page and detail drawer

**Files:**
- Create: `web/src/plugin/tr069/api/command-record.js`
- Create: `web/src/plugin/tr069/view/command-record/record-view.js`
- Create: `web/src/plugin/tr069/view/command-record/record-view.test.js`
- Create: `web/src/plugin/tr069/view/command-record/components/record-detail.vue`
- Create: `web/src/plugin/tr069/view/command-record/index.vue`
- Modify: `web/src/pathInfo.json` through the normal Vite path scan.

**Interfaces:**
- Consumes: the three fixed record endpoints from Task 2.
- Produces: `tr069CommandRecord` GVA route component and `statusView`, `operationLabel`, `formatDuration` helpers.

- [ ] **Step 1: Write failing pure helper and page contract tests**

```js
assert.deepEqual(statusView('COMPLETED'), { label: '完成', type: 'success' })
assert.equal(operationLabel('DeleteObject'), '删除对象')
assert.equal(formatDuration('2026-07-16T10:00:00Z', '2026-07-16T10:00:03Z'), '3秒')
```

The page contract asserts GVA theme classes, `AUTO_REFRESH_MS = 5000`, and a `document.visibilityState === 'visible'` guard.

- [ ] **Step 2: Run tests and verify RED**

Run: `node --test src/plugin/tr069/view/command-record/record-view.test.js`

Expected: FAIL because the files do not exist.

- [ ] **Step 3: Implement fixed record API functions and view helpers**

```js
export const getCommandRecordList = params => service({ url: '/tr069/command-record/list', method: 'get', params })
export const getCommandRecordDetail = commandId => service({ url: `/tr069/command-record/${commandId}`, method: 'get' })
export const retryCommandRecord = commandId => service({ url: `/tr069/command-record/${commandId}/retry`, method: 'post' })
```

Map all eight statuses and twelve operations to concrete Chinese text.

- [ ] **Step 4: Implement the list page**

Use `gva-search-box`, `gva-table-box`, inline filters, standard pagination, status tags, duration, detail action, and a manual refresh button. Start one 5-second interval on mount, refresh only while visible, and clear it on unmount.

- [ ] **Step 5: Implement the detail drawer**

Show overview/fault fields, formatted request/result JSON, chronological `el-timeline`, and outbound/inbound XML tabs in scrollable `<pre>` blocks using Element/GVA CSS variables. Show retry only for `FAILED` and `TIMEOUT`, then report the new command ID without modifying original history.

- [ ] **Step 6: Run frontend tests and production build**

Run: `node --test src/plugin/tr069/utils/device-actions.test.js src/plugin/tr069/view/device/device-theme.contract.test.js src/plugin/tr069/view/device/device-rpc-actions.contract.test.js src/plugin/tr069/view/device/components/data-model-viewer.contract.test.js src/plugin/tr069/view/command-record/record-view.test.js`

Run: `npm run build`

Expected: all tests and build PASS.

---

### Task 4: Runtime and safety verification

**Files:** No production files beyond Tasks 1-3.

- [ ] **Step 1: Restart the backend and preserve the frontend dev server**

Wait for ports `18888`, `7458`, and `18080` and verify `/health` plus the frontend root return HTTP 200.

- [ ] **Step 2: Verify registered APIs and routes**

Confirm all three APIs exist in `sys_apis`. Without a JWT, all three runtime paths must enter the auth chain rather than return 404.

- [ ] **Step 3: Verify existing BS remains safe**

Confirm `gva-acs-bs` remains up and periodic Inform still reaches ACS. Do not submit any dangerous command.

- [ ] **Step 4: Review and commit**

Run `git diff --check`, all focused tests, `npm run build`, and `git status --short`. Commit one coherent follow-up on `dev`; do not push without a separate instruction.
