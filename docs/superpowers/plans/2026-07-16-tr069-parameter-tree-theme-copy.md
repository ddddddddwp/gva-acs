# TR-069 Parameter Tree Theme and Copy Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the TR-069 parameter-tree drawer follow GVA light/dark themes and make complete parameter names directly and reliably copyable.

**Architecture:** Keep the existing `data-model-viewer.vue` component and backend interfaces. Replace fixed Tailwind color utilities with component-scoped semantic classes backed by Element Plus theme variables, then make parameter-name and parameter-value copying explicit through one awaited clipboard helper.

**Tech Stack:** Vue 3 `<script setup>`, Element Plus, VueUse `useClipboard`, Node.js built-in test runner, Vite.

## Global Constraints

- Keep the parameter tree query-only.
- Do not modify backend APIs or data structures.
- Do not change the parameter-tree entry point, split layout, or existing device/RPC behavior.
- Parameter names must remain natively text-selectable.
- Clipboard failures must be reported instead of showing a false success message.
- Work directly on the existing `dev` branch; do not create a worktree or subagent task.

---

### Task 1: Adapt the parameter-tree drawer to GVA themes

**Files:**
- Modify: `web/src/plugin/tr069/view/device/components/data-model-viewer.contract.test.js`
- Modify: `web/src/plugin/tr069/view/device/components/data-model-viewer.vue`

**Interfaces:**
- Consumes: Element Plus CSS variables already defined by the active GVA theme.
- Produces: semantic component classes `dm-surface`, `dm-subtle-surface`, `dm-primary-text`, `dm-secondary-text`, `dm-border-bottom`, and `dm-border-right`.

- [ ] **Step 1: Write the failing theme contract test**

Append this test to `data-model-viewer.contract.test.js`:

```js
test('parameter tree delegates light and dark colors to Element theme variables', () => {
  assert.doesNotMatch(source, /\b(?:bg-white|bg-gray-[^\s"']+|text-gray-[^\s"']+|border-gray-[^\s"']+)/)
  for (const variable of [
    '--el-bg-color',
    '--el-fill-color-light',
    '--el-text-color-primary',
    '--el-text-color-secondary',
    '--el-border-color-light',
    '--el-color-primary'
  ]) {
    assert.match(source, new RegExp(variable))
  }
})
```

- [ ] **Step 2: Run the test and verify RED**

Run:

```bash
cd web
node --test src/plugin/tr069/view/device/components/data-model-viewer.contract.test.js
```

Expected: FAIL because the component still contains `bg-white`, `bg-gray-*`, `text-gray-*`, and `border-gray-*`, and does not contain the required theme variables.

- [ ] **Step 3: Replace fixed colors with semantic classes**

In the template, retain sizing and layout utilities but replace fixed color utilities as follows:

```vue
<div class="dm-container h-full flex flex-col">
  <div class="dm-header dm-surface dm-border-bottom flex justify-between items-center px-6 py-4">
    <span class="dm-title text-lg font-bold tracking-wide font-mono">...</span>
  </div>

  <div class="dm-sidebar dm-surface dm-border-right w-1/3 min-w-[300px] flex flex-col">
    <div class="dm-border-bottom p-3">...</div>
    <span class="dm-node-label truncate font-medium" :title="node.label">...</span>
    <span class="dm-secondary-text text-xs ml-2">...</span>
  </div>

  <div class="dm-main dm-subtle-surface flex-1 flex flex-col">
    <div class="dm-path-header dm-surface dm-border-bottom p-4 flex justify-between items-center shadow-sm z-10">
      <span class="dm-secondary-text text-xs">当前路径</span>
      <span class="dm-path font-mono text-sm font-semibold break-all">{{ currentPath }}</span>
    </div>
  </div>
</div>
```

Use the same semantic text classes for loading, timestamp, empty-state, and secondary labels. Add the scoped styles:

```css
.dm-container,
.dm-surface {
  color: var(--el-text-color-primary);
  background: var(--el-bg-color);
}
.dm-subtle-surface {
  background: var(--el-fill-color-light);
}
.dm-primary-text,
.dm-title,
.dm-node-label {
  color: var(--el-text-color-primary);
}
.dm-secondary-text {
  color: var(--el-text-color-secondary);
}
.dm-path {
  color: var(--el-color-primary);
}
.dm-border-bottom {
  border-bottom: 1px solid var(--el-border-color-light);
}
.dm-border-right {
  border-right: 1px solid var(--el-border-color-light);
}
.custom-scrollbar::-webkit-scrollbar-thumb {
  background: var(--el-border-color-light);
}
```

- [ ] **Step 4: Run the test and verify GREEN**

Run:

```bash
cd web
node --test src/plugin/tr069/view/device/components/data-model-viewer.contract.test.js
```

Expected: all parameter-tree contract tests PASS.

- [ ] **Step 5: Commit the theme fix**

```bash
git add web/src/plugin/tr069/view/device/components/data-model-viewer.vue \
  web/src/plugin/tr069/view/device/components/data-model-viewer.contract.test.js
git commit -m "fix(tr069): adapt parameter tree to GVA theme"
```

### Task 2: Make parameter names directly copyable

**Files:**
- Modify: `web/src/plugin/tr069/view/device/components/data-model-viewer.contract.test.js`
- Modify: `web/src/plugin/tr069/view/device/components/data-model-viewer.vue`

**Interfaces:**
- Consumes: `useClipboard(): { copy, isSupported }` from `@vueuse/core`.
- Produces: `copyText(text, target): Promise<void>`, where `target` is `参数名` or `参数值`.

- [ ] **Step 1: Write the failing copy contract tests**

Append these tests:

```js
test('complete parameter names are selectable and directly copyable', () => {
  assert.match(source, /content="复制完整参数名"/)
  assert.match(source, /copyText\(scope\.row\.name, '参数名'\)/)
  assert.match(source, /parameter-full-name/)
  assert.match(source, /user-select:\s*text/)
})

test('parameter value copy is explicit and clipboard failures are reported', () => {
  assert.match(source, /content="复制参数值"/)
  assert.match(source, /copyText\(formatValue\(scope\.row\.valueJson\), '参数值'\)/)
  assert.match(source, /const copyText = async/)
  assert.match(source, /await copy\(String\(text\)\)/)
  assert.match(source, /复制失败，请手动选择文本复制/)
})
```

- [ ] **Step 2: Run the tests and verify RED**

Run:

```bash
cd web
node --test src/plugin/tr069/view/device/components/data-model-viewer.contract.test.js
```

Expected: the new tests FAIL because parameter names have no copy action and `copyValue` does not await or report clipboard failures.

- [ ] **Step 3: Add explicit parameter-name and value copy controls**

Replace the parameter-name cell with:

```vue
<div class="parameter-name-cell">
  <div class="parameter-name-heading">
    <span class="font-mono text-xs">{{ scope.row.name.replace(currentPath, '') }}</span>
    <el-tooltip content="复制完整参数名" placement="top">
      <el-button
        link
        type="primary"
        :icon="CopyDocument"
        aria-label="复制完整参数名"
        @click.stop="copyText(scope.row.name, '参数名')"
      />
    </el-tooltip>
  </div>
  <span
    class="parameter-full-name dm-secondary-text text-xs"
    title="点击复制完整参数名"
    role="button"
    tabindex="0"
    @click.stop="copyText(scope.row.name, '参数名')"
    @keydown.enter.stop="copyText(scope.row.name, '参数名')"
  >{{ scope.row.name }}</span>
</div>
```

Replace the value icon with an explicit control:

```vue
<el-tooltip content="复制参数值" placement="top">
  <el-button
    link
    type="primary"
    :icon="CopyDocument"
    aria-label="复制参数值"
    @click.stop="copyText(formatValue(scope.row.valueJson), '参数值')"
  />
</el-tooltip>
```

Replace `copyValue` with:

```js
const { copy, isSupported } = useClipboard()

const copyText = async (text, target) => {
  if (text === null || text === undefined || String(text) === '') return
  try {
    if (!isSupported.value) throw new Error('clipboard is not supported')
    await copy(String(text))
    ElMessage.success(`${target}已复制`)
  } catch {
    ElMessage.error('复制失败，请手动选择文本复制')
  }
}
```

Add styles that preserve native selection and keyboard focus:

```css
.parameter-name-cell {
  min-width: 0;
}
.parameter-name-heading {
  display: flex;
  align-items: center;
  gap: 4px;
}
.parameter-full-name {
  display: block;
  cursor: copy;
  user-select: text;
  overflow-wrap: anywhere;
}
.parameter-full-name:focus-visible {
  outline: 2px solid var(--el-color-primary);
  outline-offset: 2px;
}
```

- [ ] **Step 4: Run the tests and verify GREEN**

Run:

```bash
cd web
node --test src/plugin/tr069/view/device/components/data-model-viewer.contract.test.js
```

Expected: all copy and theme contract tests PASS.

- [ ] **Step 5: Commit the copy fix**

```bash
git add web/src/plugin/tr069/view/device/components/data-model-viewer.vue \
  web/src/plugin/tr069/view/device/components/data-model-viewer.contract.test.js
git commit -m "fix(tr069): make parameter names directly copyable"
```

### Task 3: Verify the complete frontend behavior

**Files:**
- Verify: `web/src/plugin/tr069/view/device/components/data-model-viewer.vue`
- Verify: `web/src/plugin/tr069/view/device/components/data-model-viewer.contract.test.js`

**Interfaces:**
- Consumes: the completed theme and copy behavior from Tasks 1 and 2.
- Produces: fresh automated and runtime evidence that the parameter tree works in both GVA themes.

- [ ] **Step 1: Run all focused TR-069 frontend tests**

```bash
cd web
node --test \
  src/plugin/tr069/utils/device-actions.test.js \
  src/plugin/tr069/view/device/device-theme.contract.test.js \
  src/plugin/tr069/view/device/device-rpc-actions.contract.test.js \
  src/plugin/tr069/view/device/components/data-model-viewer.contract.test.js \
  src/plugin/tr069/view/command-record/record-view.test.js
```

Expected: all tests PASS with zero failures.

- [ ] **Step 2: Run the production build**

```bash
cd web
npm run build
```

Expected: Vite exits with code `0` and reports `built`.

- [ ] **Step 3: Verify the running frontend**

At `http://127.0.0.1:18080/#/layout/tr069/device`:

1. Open the parameter tree in GVA light mode and verify drawer, tree, table, selected node, and scrollbars have no mismatched dark surfaces.
2. Switch to GVA dark mode and repeat the same checks.
3. Click a complete parameter path and verify the clipboard contains the exact `Device...` path.
4. Click “复制参数值” and verify only the value is copied.
5. Use keyboard focus plus Enter on the complete parameter path and verify it copies.

- [ ] **Step 4: Confirm repository state**

```bash
git diff --check
git status --short
git log -4 --oneline
```

Expected: no uncommitted implementation changes and both fix commits appear on `dev`.
