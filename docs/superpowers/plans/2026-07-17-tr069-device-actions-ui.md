# TR-069 Device Actions UI Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Reorganize the TR-069 device-list actions into a parameter drawer entry, two clearly grouped RPC menus, and an independent device-delete action while preserving the GVA visual language.

**Architecture:** Keep the RPC registry as the single source of truth and add a two-menu presentation mapping on top of its four semantic groups. Move parameter synchronization into the parameter drawer so the drawer owns its own loading state and the device list only selects the active device. Use contract tests for the Vue templates and executable unit tests for the action registry before changing production code.

**Tech Stack:** Vue 3 SFC, Element Plus 2.13, JavaScript ES modules, Node test runner, Vite 8.

## Global Constraints

- The device-list operation row contains exactly four top-level entries: `参数`, `查询与配置`, `文件与维护`, and independent danger-styled `删除`.
- `查询与配置` contains the `查询` and `配置` groups; `文件与维护` contains the `文件` and `维护` groups.
- `DeleteObject` is labeled `删除对象实例`; its key, method, permission, payload, and confirmation behavior remain unchanged.
- Parameter synchronization sends the existing `Device.` payload and moves into the parameter drawer as `同步设备参数`.
- `刷新本地数据` reads stored data only; it does not create an RPC command.
- Offline devices may inspect local data but cannot start parameter synchronization or another CWMP command.
- Existing capability checks, dangerous confirmations, RPC forms, APIs, command lifecycle, and record pages remain unchanged.
- UI colors and surfaces continue to use Element Plus/GVA theme variables; no light-only or dark-only hard-coded colors are added.

---

### Task 1: Reorganize the device actions and move synchronization into the parameter drawer

**Files:**
- Modify: `web/src/plugin/tr069/utils/device-actions.js`
- Test: `web/src/plugin/tr069/utils/device-actions.test.js`
- Modify: `web/src/plugin/tr069/view/device/index.vue`
- Test: `web/src/plugin/tr069/view/device/device-rpc-actions.contract.test.js`
- Modify: `web/src/plugin/tr069/view/device/components/data-model-viewer.vue`
- Test: `web/src/plugin/tr069/view/device/components/data-model-viewer.contract.test.js`

**Interfaces:**
- Consumes: `fullDataModelSync(deviceId: number): Promise<Response>` from `web/src/plugin/tr069/api/command.js` and `canIssueDeviceCommand(row): boolean` from `device-actions.js`.
- Produces: `RPC_ACTION_MENUS`, an array of `{ key: string, label: string, groups: RPCActionGroup[] }`; the existing `RPC_ACTION_GROUPS`, `findRPCAction`, and capability functions remain compatible.
- Produces: `syncDeviceParameters(): Promise<void>` inside `data-model-viewer.vue`; it calls `fullDataModelSync(props.row.ID)` once while `syncingParameters` is false.

- [ ] **Step 1: Write failing registry and template contract tests**

Update `device-actions.test.js` to import `RPC_ACTION_MENUS`, assert the two top-level menu labels and their group mapping, and change the expected `DeleteObject` label:

```js
import {
  RPC_ACTION_GROUPS,
  RPC_ACTION_MENUS,
  canIssueDeviceCommand,
  canIssueRPCAction,
  deviceParameterSyncPayload,
  findRPCAction
} from './device-actions.js'

test('device actions expose four semantic groups through two menus', () => {
  assert.deepEqual(RPC_ACTION_GROUPS.map(group => group.label), ['查询', '配置', '文件', '维护'])
  assert.deepEqual(RPC_ACTION_MENUS.map(menu => menu.label), ['查询与配置', '文件与维护'])
  assert.deepEqual(
    RPC_ACTION_MENUS.map(menu => menu.groups.map(group => group.label)),
    [['查询', '配置'], ['文件', '维护']]
  )

  const actions = RPC_ACTION_GROUPS.flatMap(group => group.actions)
  assert.equal(actions.length, 12)
  assert.deepEqual(actions.map(action => action.label), [
    '查询设备能力', '获取参数', '获取参数名称', '获取参数属性',
    '配置参数', '配置参数属性', '添加对象', '删除对象实例',
    '下载文件', '上传文件', '重启设备', '恢复出厂设置'
  ])
  assert.equal(new Set(actions.map(action => action.key)).size, 12)
  assert.equal(actions.some(action => action.key === 'deleteDevice'), false)
})
```

Replace the first test in `device-rpc-actions.contract.test.js` with assertions for the four-entry layout, two menu mapping, grouped titles, independent delete button, and removal of list-level synchronization:

```js
test('device list renders four aligned top-level actions and two grouped RPC menus', async () => {
  const source = await readFile(devicePageURL, 'utf8')

  assert.match(source, /class="device-row-actions"/)
  assert.match(source, />参数<\/el-button>/)
  assert.match(source, /RPC_ACTION_MENUS/)
  assert.match(source, /v-for="menu in RPC_ACTION_MENUS"/)
  assert.match(source, /\{\{ menu\.label \}\}/)
  assert.match(source, /menu\.groups/)
  assert.match(source, /groupIndex > 0/)
  assert.match(source, /type="danger"[\s\S]*@click="deleteRow\(scope\.row\)"[\s\S]*>删除<\/el-button>/)
  assert.doesNotMatch(source, /command="deleteDevice"/)
  assert.doesNotMatch(source, /syncParameters\(scope\.row\)/)
})
```

Replace the `parameter tree is query-only` test in `data-model-viewer.contract.test.js` with synchronization ownership and local-refresh assertions:

```js
test('parameter drawer separates local refresh from device synchronization', () => {
  assert.match(source, /刷新本地数据/)
  assert.match(source, /同步设备参数/)
  assert.match(source, /import \{ fullDataModelSync \} from '@\/plugin\/tr069\/api\/command'/)
  assert.match(source, /const syncingParameters = ref\(false\)/)
  assert.match(source, /:disabled="!deviceRow\.online"/)
  assert.match(source, /:loading="syncingParameters"/)
  assert.match(source, /if \(!props\.row\.ID \|\| !deviceRow\.value\.online \|\| syncingParameters\.value\) return/)
  assert.match(source, /await fullDataModelSync\(props\.row\.ID\)/)
  assert.match(source, /commandId=/)
})
```

- [ ] **Step 2: Run the focused tests and verify RED**

Run:

```bash
cd web
node --test \
  src/plugin/tr069/utils/device-actions.test.js \
  src/plugin/tr069/view/device/device-rpc-actions.contract.test.js \
  src/plugin/tr069/view/device/components/data-model-viewer.contract.test.js
```

Expected: FAIL because `RPC_ACTION_MENUS`, `device-row-actions`, the two menu labels, the independent delete link, and parameter-drawer synchronization do not yet exist; the old label remains `删除对象`.

- [ ] **Step 3: Add the two-menu mapping and precise DeleteObject label**

In `device-actions.js`, change only the `deleteObject` label and add the presentation mapping after `RPC_ACTION_GROUPS`:

```js
{ key: 'deleteObject', label: '删除对象实例', method: 'DeleteObject', confirm: 'danger' }

export const RPC_ACTION_MENUS = [
  {
    key: 'query-config',
    label: '查询与配置',
    groups: RPC_ACTION_GROUPS.slice(0, 2)
  },
  {
    key: 'file-maintenance',
    label: '文件与维护',
    groups: RPC_ACTION_GROUPS.slice(2, 4)
  }
]
```

Keep `const RPC_ACTIONS = RPC_ACTION_GROUPS.flatMap(...)` unchanged so lookup, capability, and confirmation behavior still uses the original 12-action registry.

- [ ] **Step 4: Replace the device-list action cell with the four-entry GVA layout**

In `index.vue`:

1. Import `RPC_ACTION_MENUS` instead of using `RPC_ACTION_GROUPS` directly in the template.
2. Remove the `fullDataModelSync` import, `syncingDeviceIds`, and `syncParameters` function.
3. Change the fixed operation-column width to `360`.
4. Replace the operation slot with:

```vue
<div class="device-row-actions">
  <el-button type="primary" link :icon="Connection" @click="openDataModelFromRow(scope.row)">
    参数
  </el-button>

  <el-dropdown
    v-for="menu in RPC_ACTION_MENUS"
    :key="menu.key"
    trigger="click"
    @command="command => handleMoreCommand(command, scope.row)"
  >
    <el-button type="primary" link>
      {{ menu.label }}<el-icon class="el-icon--right"><ArrowDown /></el-icon>
    </el-button>
    <template #dropdown>
      <el-dropdown-menu>
        <template v-for="(group, groupIndex) in menu.groups" :key="group.label">
          <el-dropdown-item
            disabled
            :divided="groupIndex > 0"
            class="rpc-group-title"
          >
            {{ group.label }}
          </el-dropdown-item>
          <el-dropdown-item
            v-for="action in group.actions"
            :key="action.key"
            :command="action.key"
            :disabled="!canIssueRPCAction(scope.row, action)"
          >
            {{ action.label }}
          </el-dropdown-item>
        </template>
      </el-dropdown-menu>
    </template>
  </el-dropdown>

  <el-button type="danger" link @click="deleteRow(scope.row)">删除</el-button>
</div>
```

Add scoped layout rules without hard-coded theme colors:

```css
.device-row-actions {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  white-space: nowrap;
}
.device-row-actions :deep(.el-button) {
  margin-left: 0;
}
```

- [ ] **Step 5: Move synchronization state and behavior into the parameter drawer**

In `data-model-viewer.vue`, import the existing API:

```js
import { fullDataModelSync } from '@/plugin/tr069/api/command'
```

Add component-local state and a guarded async handler:

```js
const syncingParameters = ref(false)

const syncDeviceParameters = async () => {
  if (!props.row.ID || !deviceRow.value.online || syncingParameters.value) return
  syncingParameters.value = true
  try {
    const res = await fullDataModelSync(props.row.ID)
    if (res.code === 0) {
      ElMessage.success(`参数同步已下发，commandId=${res.data?.commandId || '-'}`)
    } else {
      ElMessage.error(res.msg || '参数同步下发失败')
    }
  } catch {
    ElMessage.error('参数同步下发失败')
  } finally {
    syncingParameters.value = false
  }
}
```

Replace the current `刷新结构` top action with the two explicit actions:

```vue
<div class="flex gap-2">
  <el-button
    type="primary"
    plain
    size="small"
    :icon="Refresh"
    :loading="loadingStructure"
    @click="refreshStructure"
  >
    刷新本地数据
  </el-button>
  <el-button
    type="primary"
    size="small"
    :disabled="!deviceRow.online"
    :loading="syncingParameters"
    @click="syncDeviceParameters"
  >
    同步设备参数
  </el-button>
</div>
```

- [ ] **Step 6: Run focused tests and verify GREEN**

Run:

```bash
cd web
node --test \
  src/plugin/tr069/utils/device-actions.test.js \
  src/plugin/tr069/view/device/device-rpc-actions.contract.test.js \
  src/plugin/tr069/view/device/components/data-model-viewer.contract.test.js
```

Expected: all focused tests PASS.

- [ ] **Step 7: Run full frontend regression and production build**

Run:

```bash
cd web
node --test \
  src/plugin/tr069/utils/device-actions.test.js \
  src/plugin/tr069/view/command-record/record-view.test.js \
  src/plugin/tr069/view/device/device-theme.contract.test.js \
  src/plugin/tr069/view/device/device-rpc-actions.contract.test.js \
  src/plugin/tr069/view/device/components/data-model-viewer.contract.test.js
npm run build
cd ..
git diff --check
```

Expected: all TR-069 frontend tests PASS, Vite build succeeds, and `git diff --check` prints no output. Existing Vite bundle-size warnings are non-blocking because this task adds no dependency or bundle-scale component.

- [ ] **Step 8: Verify the running UI in both GVA themes**

Open `http://127.0.0.1:18080/#/layout/tr069/device` and verify:

1. `参数`, `查询与配置`, `文件与维护`, and `删除` share one horizontal center line.
2. The two dropdowns show their approved gray group titles and a divider before the second group.
3. `删除对象实例` appears only under `配置`; independent `删除` remains outside both menus.
4. The parameter drawer shows `刷新本地数据` and `同步设备参数`; offline devices can inspect local data but cannot synchronize.
5. Repeat the same checks in light and dark themes; text, surface, divider, hover, disabled, and danger colors all follow GVA theme variables.

- [ ] **Step 9: Commit the implementation**

```bash
git add \
  web/src/plugin/tr069/utils/device-actions.js \
  web/src/plugin/tr069/utils/device-actions.test.js \
  web/src/plugin/tr069/view/device/index.vue \
  web/src/plugin/tr069/view/device/device-rpc-actions.contract.test.js \
  web/src/plugin/tr069/view/device/components/data-model-viewer.vue \
  web/src/plugin/tr069/view/device/components/data-model-viewer.contract.test.js
git commit -m "feat(tr069): reorganize device actions"
```
