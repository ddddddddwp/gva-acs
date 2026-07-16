# TR-069 Device Page Simplification Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the overloaded TR-069 device detail workflow with a compact `同步参数 | 参数 | 更多` action column, enforce online-only command admission, and make the existing parameter tree a cache-only viewer.

**Architecture:** The active plugin device page keeps ownership of dialogs and the parameter-tree drawer, while a small pure JavaScript policy module owns command eligibility and the fixed `Device.` sync payload. The Go `CommandService` becomes the authoritative online-admission boundary before Redis/database mutation and exposes one fixed single-command device-parameter sync operation. Redis remains the durable command queue; the parameter-tree component only reads persisted data-model values.

**Tech Stack:** Go 1.x, Gin, GORM with `glebarez/sqlite` tests, Redis, Vue 3, Element Plus, Node.js built-in test runner, Vite.

## Global Constraints

- The active route component is `web/src/plugin/tr069/view/device/index.vue`; do not modify the legacy `web/src/view/tr069/device/index.vue`.
- The table actions are exactly `同步参数 | 参数 | 更多`, where `更多` contains `获取参数`, `配置参数`, and `删除设备`.
- Remove the detail drawer, RPC refresh, TR-196/FAP area, and every `FapInfo` dependency from the active page.
- A quick sync creates exactly one `GetParameterValues` command whose only path is `Device.`; it never emits `GetParameterNames` and never loops over paths.
- `同步参数`, custom GPV, and SPV are unavailable when `row.online` is false; cached parameter viewing and device deletion remain available.
- Backend online admission is authoritative and uses the existing 180-second last-Inform threshold.
- An offline request must not create a command record or write Redis.
- The parameter tree keeps its existing drawer/tree/table layout but becomes query-only; opening or refreshing it never emits a CWMP command.
- Success feedback remains lightweight: only report that the task was queued; do not add polling, progress, task history, or automatic parameter-tree refresh.
- Use the existing Redis immediate queue with local `system.use-redis: true`; do not add a MySQL queue or in-memory fallback.
- Connection Request failure remains non-fatal after successful Redis enqueue; HTTP/HTTPS Connection Request adaptation is outside this change.
- Preserve the ignored `server/config.local.yaml`; never commit its database or Redis runtime values.
- Preserve the independent ignored `server/plugin/tr069/lib/tr069-core-only` repository and do not include it in parent commits.

---

### Task 1: Enforce online command admission and fixed single GPV sync

**Files:**
- Modify: `server/plugin/tr069/service/command.go`
- Modify: `server/plugin/tr069/api/datamodel.go`
- Modify: `server/plugin/tr069/api/command.go`
- Create: `server/plugin/tr069/service/command_sync_test.go`

**Interfaces:**
- Consumes: `model.Device.LastInform`, `adapter.RedisAvailable`, and the existing `CommandService.enqueueImmediate`.
- Produces: `service.ErrDeviceOffline`, `service.ErrCommandQueueUnavailable`, `CommandService.EnqueueDeviceParameterSync(deviceID uint) (string, error)`, `commandTargetByID(deviceID uint, now time.Time) (string, error)`, and a fixed `deviceParameterSyncParams() map[string]interface{}`.

- [ ] **Step 1: Write failing online-admission and fixed-payload tests**

Create `server/plugin/tr069/service/command_sync_test.go` with an isolated SQLite database:

```go
package service

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/ddddddddwp/gva-acs/server/global"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/model"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func useCommandTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil { t.Fatalf("open db: %v", err) }
	if err := db.AutoMigrate(&model.Device{}, &model.Command{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	previous := global.GVA_DB
	global.GVA_DB = db
	t.Cleanup(func() { global.GVA_DB = previous })
	return db
}

func TestCommandTargetRejectsOfflineDeviceBeforeQueueAccess(t *testing.T) {
	db := useCommandTestDB(t)
	now := time.Date(2026, 7, 14, 12, 0, 0, 0, time.UTC)
	device := model.Device{SerialNumber: "SN-1", OUI: "001122", LastInform: now.Add(-181 * time.Second)}
	if err := db.Create(&device).Error; err != nil { t.Fatal(err) }

	_, err := new(CommandService).commandTargetByID(device.ID, now)
	if !errors.Is(err, ErrDeviceOffline) { t.Fatalf("got %v", err) }
	var count int64
	if err := db.Model(&model.Command{}).Count(&count).Error; err != nil { t.Fatal(err) }
	if count != 0 { t.Fatalf("offline request created %d commands", count) }
}

func TestCommandTargetAcceptsRecentInform(t *testing.T) {
	db := useCommandTestDB(t)
	now := time.Date(2026, 7, 14, 12, 0, 0, 0, time.UTC)
	device := model.Device{SerialNumber: "SN-1", OUI: "001122", LastInform: now.Add(-179 * time.Second)}
	if err := db.Create(&device).Error; err != nil { t.Fatal(err) }
	got, err := new(CommandService).commandTargetByID(device.ID, now)
	if err != nil { t.Fatal(err) }
	if got != "001122-SN-1" { t.Fatalf("got %q", got) }
}

func TestDeviceParameterSyncParamsAreExactlyDeviceRoot(t *testing.T) {
	want := map[string]interface{}{"paths": []string{"Device."}}
	if got := deviceParameterSyncParams(); !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v want %#v", got, want)
	}
}
```

- [ ] **Step 2: Run the focused tests and verify RED**

Run:

```bash
cd server
go test ./plugin/tr069/service -run 'CommandTarget|DeviceParameterSync' -count=1
```

Expected: FAIL because the sentinel errors, `commandTargetByID`, and `deviceParameterSyncParams` do not exist.

- [ ] **Step 3: Implement the admission boundary and queue errors**

In `server/plugin/tr069/service/command.go`, add exact shared errors and the 180-second rule:

```go
var (
	ErrDeviceOffline          = errors.New("device offline")
	ErrCommandQueueUnavailable = errors.New("command queue unavailable")
)

const commandOnlineThreshold = 180 * time.Second

func deviceParameterSyncParams() map[string]interface{} {
	return map[string]interface{}{"paths": []string{"Device."}}
}

func (s *CommandService) commandTargetByID(deviceID uint, now time.Time) (string, error) {
	if !adapter.DBAvailable() {
		return "", errors.New("db not initialized")
	}
	var device model.Device
	if err := global.GVA_DB.Select("id", "oui", "serial_number", "last_inform").First(&device, deviceID).Error; err != nil {
		return "", err
	}
	if device.OUI == "" || device.SerialNumber == "" {
		return "", fmt.Errorf("device missing oui/serialNumber: %d", deviceID)
	}
	if device.LastInform.IsZero() || now.Sub(device.LastInform) >= commandOnlineThreshold {
		return "", ErrDeviceOffline
	}
	return fmt.Sprintf("%s-%s", device.OUI, device.SerialNumber), nil
}
```

Replace every command-producing call to `deviceKeyByID` with `commandTargetByID(deviceID, time.Now())`. Replace both Redis availability errors in `enqueue` and `enqueueImmediate` with `ErrCommandQueueUnavailable`.

Replace the multi-path method with one command:

```go
func (s *CommandService) EnqueueDeviceParameterSync(deviceID uint) (string, error) {
	deviceKey, err := s.commandTargetByID(deviceID, time.Now())
	if err != nil { return "", err }
	dedupKey := fmt.Sprintf("dm:gpv:Device.:%s:%d", deviceKey, time.Now().UnixNano())
	return s.enqueueImmediate(
		context.Background(), deviceID, deviceKey,
		"GetParameterValues", deviceParameterSyncParams(), dedupKey,
	)
}
```

Delete `EnqueueFullDataModelSync(deviceID uint, paths []string)` and the now-unused loop/console output.

- [ ] **Step 4: Map service errors at the API boundary**

In `server/plugin/tr069/api/command.go` and `datamodel.go`, add a local helper with exact messages:

```go
func commandFailureMessage(err error) string {
	switch {
	case errors.Is(err, service.ErrDeviceOffline):
		return "设备离线，无法下发命令"
	case errors.Is(err, service.ErrCommandQueueUnavailable):
		return "命令队列不可用，请检查 Redis 服务"
	default:
		return "任务下发失败"
	}
}
```

Keep a single helper in `api/command.go` (same Go package) and reuse it from `datamodel.go`. Change `FullSync` to ignore request paths and return the command ID:

```go
func (a *DataModelApi) FullSync(c *gin.Context) {
	deviceID, err := strconv.Atoi(c.Param("deviceId"))
	if err != nil || deviceID <= 0 {
		response.FailWithMessage("设备参数错误", c)
		return
	}
	commandID, err := dmService.EnqueueDeviceParameterSync(uint(deviceID))
	if err != nil {
		response.FailWithMessage(commandFailureMessage(err), c)
		return
	}
	response.OkWithDetailed(map[string]string{"commandId": commandID}, "同步任务已下发", c)
}
```

Use `commandFailureMessage(err)` for custom GPV and SPV enqueue failures as well.

- [ ] **Step 5: Verify focused and package tests**

Run:

```bash
cd server
go test ./plugin/tr069/service -run 'CommandTarget|DeviceParameterSync' -count=1
go test ./plugin/tr069/api ./plugin/tr069/service -count=1
```

Expected: PASS. The service test output must show the focused tests were executed, not “no tests to run”.

- [ ] **Step 6: Commit the backend admission change**

```bash
git add server/plugin/tr069/service/command.go \
  server/plugin/tr069/service/command_sync_test.go \
  server/plugin/tr069/api/command.go \
  server/plugin/tr069/api/datamodel.go
git commit -m "feat: gate TR-069 commands by device status"
```

---

### Task 2: Simplify the active device-list actions

**Files:**
- Create: `web/src/plugin/tr069/utils/device-actions.js`
- Create: `web/src/plugin/tr069/utils/device-actions.test.js`
- Modify: `web/src/plugin/tr069/view/device/index.vue`
- Modify: `web/src/plugin/tr069/api/command.js`

**Interfaces:**
- Consumes: device rows with `ID` and `online`, existing GPV/SPV APIs, `fullDataModelSync(deviceId)`, and the existing `DataModelViewer` component.
- Produces: `canIssueDeviceCommand(row)`, `deviceParameterSyncPayload()`, and the exact action layout `同步参数 | 参数 | 更多`.

- [ ] **Step 1: Write failing pure action-policy tests**

Create `web/src/plugin/tr069/utils/device-actions.test.js`:

```js
import test from 'node:test'
import assert from 'node:assert/strict'
import {
  canIssueDeviceCommand,
  deviceParameterSyncPayload
} from './device-actions.js'

test('commands are enabled only for online devices', () => {
  assert.equal(canIssueDeviceCommand({ online: true }), true)
  assert.equal(canIssueDeviceCommand({ online: false }), false)
  assert.equal(canIssueDeviceCommand({}), false)
})

test('quick sync always uses only Device root', () => {
  assert.deepEqual(deviceParameterSyncPayload(), { paths: ['Device.'] })
  const first = deviceParameterSyncPayload()
  first.paths.push('Other.')
  assert.deepEqual(deviceParameterSyncPayload(), { paths: ['Device.'] })
})
```

- [ ] **Step 2: Run the Node test and verify RED**

Run:

```bash
cd web
node --test src/plugin/tr069/utils/device-actions.test.js
```

Expected: FAIL because `device-actions.js` does not exist.

- [ ] **Step 3: Implement the pure policy module**

Create `web/src/plugin/tr069/utils/device-actions.js`:

```js
export const canIssueDeviceCommand = (row) => row?.online === true

export const deviceParameterSyncPayload = () => ({
  paths: ['Device.']
})
```

Run the Node test again; expected: 2 tests PASS.

- [ ] **Step 4: Make the sync API fixed-purpose**

Change `fullDataModelSync` in `web/src/plugin/tr069/api/command.js` to accept only `deviceId` and send the fixed payload internally:

```js
import { deviceParameterSyncPayload } from '@/plugin/tr069/utils/device-actions'

export const fullDataModelSync = (deviceId) => service({
  url: `/tr069/datamodel/${deviceId}/sync`,
  method: 'post',
  data: deviceParameterSyncPayload()
})
```

This keeps the API request deterministic even if a future caller bypasses the table handler.

- [ ] **Step 5: Replace the operation column**

In the active `index.vue`, replace `详情 | 参数 | 删除` with:

```js
import { canIssueDeviceCommand } from '@/plugin/tr069/utils/device-actions'
```

```vue
<el-table-column label="操作" width="300" fixed="right" align="center">
  <template #default="scope">
    <el-tooltip
      :disabled="canIssueDeviceCommand(scope.row)"
      content="设备离线，无法下发命令"
    >
      <span>
        <el-button
          type="primary"
          link
          :disabled="!canIssueDeviceCommand(scope.row)"
          :loading="syncingDeviceIds.has(scope.row.ID)"
          @click="syncDeviceParameters(scope.row)"
        >同步参数</el-button>
      </span>
    </el-tooltip>
    <el-button type="primary" link :icon="Connection" @click="openDataModelFromRow(scope.row)">参数</el-button>
    <el-dropdown trigger="click" @command="handleMoreCommand($event, scope.row)">
      <el-button type="primary" link>更多<el-icon class="el-icon--right"><ArrowDown /></el-icon></el-button>
      <template #dropdown>
        <el-dropdown-menu>
          <el-dropdown-item command="get" :disabled="!canIssueDeviceCommand(scope.row)">获取参数</el-dropdown-item>
          <el-dropdown-item command="set" :disabled="!canIssueDeviceCommand(scope.row)">配置参数</el-dropdown-item>
          <el-dropdown-item command="delete" divided>删除设备</el-dropdown-item>
        </el-dropdown-menu>
      </template>
    </el-dropdown>
  </template>
</el-table-column>
```

Use row-scoped handlers:

```js
const syncingDeviceIds = reactive(new Set())

const syncDeviceParameters = async (row) => {
  if (!canIssueDeviceCommand(row) || syncingDeviceIds.has(row.ID)) return
  syncingDeviceIds.add(row.ID)
  try {
    const res = await fullDataModelSync(row.ID)
    if (res.code === 0) ElMessage.success('同步任务已下发')
    else ElMessage.error(res.msg || '下发失败')
  } finally {
    syncingDeviceIds.delete(row.ID)
  }
}

const handleMoreCommand = (command, row) => {
  currentRow.value = row
  if (command === 'get') openGPVDialog()
  if (command === 'set') openSPVDialog()
  if (command === 'delete') deleteRow(row)
}
```

- [ ] **Step 6: Delete the overloaded detail workflow**

Remove from `index.vue`:

- The complete detail drawer template.
- `viewDetail`, `handleGetRPCMethods`, and `openFullSyncDialog`.
- `drawerVisible`, `rpcLoading`, and `fullSyncSubmitting`.
- `getRPCMethods`, `FapInfo`, and the unused detail-only icon imports.
- The TR-196 component rendering.

Keep `currentRow` solely as the selected row for GPV/SPV dialogs and the parameter-tree drawer.

- [ ] **Step 7: Verify tests and production compilation**

Run:

```bash
cd web
node --test src/plugin/tr069/utils/device-actions.test.js
npm run build
```

Expected: Node tests PASS and Vite build exits 0. If Vite regenerates only `web/src/pathInfo.json` ordering/newline, restore its pre-build content with `apply_patch`; do not commit generated-only noise.

- [ ] **Step 8: Commit the simplified device actions**

```bash
git add web/src/plugin/tr069/utils/device-actions.js \
  web/src/plugin/tr069/utils/device-actions.test.js \
  web/src/plugin/tr069/view/device/index.vue \
  web/src/plugin/tr069/api/command.js
git commit -m "feat: simplify TR-069 device actions"
```

---

### Task 3: Make the parameter tree query-only

**Files:**
- Modify: `web/src/plugin/tr069/view/device/components/data-model-viewer.vue`
- Create: `web/src/plugin/tr069/view/device/components/data-model-viewer.contract.test.js`

**Interfaces:**
- Consumes: `row.ID`, `row.online`, `getDataModelStructure`, and `getDataModelList`.
- Produces: the existing parameter-tree drawer with no command-producing imports or handlers.

- [ ] **Step 1: Write a failing query-only contract test**

Create a Node source-contract test:

```js
import test from 'node:test'
import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'

const source = await readFile(
  new URL('./data-model-viewer.vue', import.meta.url),
  'utf8'
)

test('parameter tree has no command-producing sync action', () => {
  assert.doesNotMatch(source, /fullDataModelSync/)
  assert.doesNotMatch(source, /openFullSync/)
  assert.doesNotMatch(source, />\s*全量同步\s*</)
})

test('parameter tree uses the device online field', () => {
  assert.match(source, /deviceRow\.online/)
  assert.doesNotMatch(source, /deviceRow\.status/)
})
```

- [ ] **Step 2: Run the contract test and verify RED**

Run:

```bash
cd web
node --test src/plugin/tr069/view/device/components/data-model-viewer.contract.test.js
```

Expected: FAIL because the component still imports/calls `fullDataModelSync` and reads `deviceRow.status`.

- [ ] **Step 3: Remove command behavior and fix online display**

In `data-model-viewer.vue`:

- Remove the header “全量同步” button.
- Remove `Download` from icon imports.
- Remove `fullDataModelSync` import.
- Remove `fullSyncSubmitting` and `openFullSync`.
- Change both status expressions to `deviceRow.online`.
- Preserve `refreshStructure` and `refreshValues` as database reads only.

- [ ] **Step 4: Verify the contract and build**

Run:

```bash
cd web
node --test src/plugin/tr069/view/device/components/data-model-viewer.contract.test.js
node --test src/plugin/tr069/utils/device-actions.test.js
npm run build
```

Expected: all Node tests PASS and Vite build exits 0.

- [ ] **Step 5: Commit the query-only parameter tree**

```bash
git add web/src/plugin/tr069/view/device/components/data-model-viewer.vue \
  web/src/plugin/tr069/view/device/components/data-model-viewer.contract.test.js
git commit -m "refactor: make TR-069 parameter tree query only"
```

---

### Task 4: Enable Redis locally and verify the complete BS-to-UI workflow

**Files:**
- Modify locally, do not commit: `server/config.local.yaml`
- Verify: `server/plugin/tr069/...`
- Verify: `web/src/plugin/tr069/...`

**Interfaces:**
- Consumes: local Redis at `127.0.0.1:6379`, MySQL `gva_acs`, GVA ports `8888/7458`, Web port `8080`, and the BS/OAM container.
- Produces: a running local environment in which one online device can enqueue one `GPV(Device.)` and later display persisted values.

- [ ] **Step 1: Change only the ignored local runtime configuration**

Using `apply_patch`, set in `server/config.local.yaml`:

```yaml
system:
    use-redis: true
```

Leave the existing Redis address `127.0.0.1:6379`, DB credentials, and tracked `server/config.yaml` unchanged. Confirm:

```bash
git check-ignore -v server/config.local.yaml
git status --short
```

Expected: `server/config.local.yaml` is ignored and does not appear in status.

- [ ] **Step 2: Run backend verification**

Run:

```bash
cd server
go test ./plugin/tr069/service ./plugin/tr069/api ./plugin/tr069/handler ./plugin/tr069/engine -count=1
```

Expected: PASS. Then run the broader suite once:

```bash
go test ./plugin/tr069/... -count=1
```

Record the existing `adapter.TestDataModelHook_PersistsGPVWithXsiType` JSONB scan failure separately if it remains; no other failures are accepted.

- [ ] **Step 3: Run frontend verification**

Run:

```bash
cd web
node --test src/plugin/tr069/utils/device-actions.test.js \
  src/plugin/tr069/view/device/components/data-model-viewer.contract.test.js
npm run build
```

Expected: all tests PASS and Vite build exits 0.

- [ ] **Step 4: Restart GVA and Web with the local config**

Start GVA from `server/` with:

```bash
go run . -c config.local.yaml
```

Start Web from `web/` with:

```bash
npm run dev
```

Verify:

```bash
curl --noproxy '*' -fsS http://127.0.0.1:18888/health
curl --noproxy '*' -fsS http://127.0.0.1:18080/api/health
redis-cli -h 127.0.0.1 -p 6379 PING
```

Expected: both health calls return `"ok"`; Redis returns `PONG`.

- [ ] **Step 5: Verify live TR-069 behavior**

With the BS/OAM stack running and a recent Inform received:

1. Open `http://127.0.0.1:18080/#/layout/tr069/device`.
2. Confirm the row shows `同步参数 | 参数 | 更多` and no `详情`.
3. Click `同步参数` on the online BS.
4. Query MySQL and confirm exactly one new `tr069_commands` row with `operation='GetParameterValues'` and `params_json='{"paths":["Device."]}'`.
5. Wait for the device response and confirm `tr069_datamodel_values` timestamps/count update.
6. Open `参数` and confirm the cached tree renders without issuing another command.
7. Confirm offline rows disable sync/GPV/SPV while `参数` and delete remain available.

- [ ] **Step 6: Final repository verification and commit policy**

Run:

```bash
git status --short --branch
git log -4 --oneline
```

Expected: only the planned source/test commits exist; `server/config.local.yaml`, runtime logs, `web/node_modules`, `.codegraph`, `.superpowers`, and the nested core repository are absent from parent status. No additional commit is needed unless verification requires a source fix; any such fix receives its own focused test and commit.
