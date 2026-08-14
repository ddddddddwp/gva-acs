# TR-069 Parameter Name Copy Layout Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 把参数树参数表格收敛为单行完整参数名与一个悬停复制入口，并彻底移除参数值复制。

**Architecture:** 只修改现有 `data-model-viewer.vue` 的参数名单元格、复制函数和 scoped style；用 Node 源码契约测试锁定模板、无障碍和主题约束，不改变后端、接口或树结构。

**Tech Stack:** Vue 3、Element Plus、VueUse Clipboard、Node test、GVA CSS variables。

## Global Constraints

- 每行只展示一次 `scope.row.name` 完整路径。
- 参数名文本保持原生可选择；超长路径单行省略并通过 Tooltip 查看完整内容。
- 只保留参数名复制；值列不得出现复制按钮、Tooltip 或复制调用。
- 复制按钮默认弱化，行 hover 或 `focus-visible` 时显示。
- 明亮/黑暗主题继续只使用 Element Plus/GVA CSS variables。
- 不修改参数树接口、查询方式、抽屉尺寸和左右区域比例。

---

### Task 1: Simplify the parameter cell and lock the behavior with tests

**Files:**
- Modify: `web/src/plugin/tr069/view/device/components/data-model-viewer.contract.test.js`
- Modify: `web/src/plugin/tr069/view/device/components/data-model-viewer.vue`

**Interfaces:**
- Produces: `copyParameterName(name)` and `.parameter-name-copy` hover/focus interaction.
- Consumes: existing `useClipboard`, `ElMessage`, `CopyDocument`, `scope.row.name`.

- [ ] **Step 1: Replace the old copy contract with failing assertions**

Use source-contract assertions that require one name rendering and reject value copy:

```js
test('parameter table keeps only the compact parameter-name copy action', () => {
  assert.match(source, /content="复制参数名"/)
  assert.match(source, /aria-label="复制参数名"/)
  assert.match(source, /copyParameterName\(scope\.row\.name\)/)
  assert.match(source, /class="parameter-name-text"/)
  assert.doesNotMatch(source, /复制完整参数名/)
  assert.doesNotMatch(source, /复制参数值/)
  assert.doesNotMatch(source, /copyText\(/)
  assert.doesNotMatch(source, /scope\.row\.name\.replace\(currentPath/)
})

test('parameter-name copy remains selectable and keyboard accessible', () => {
  assert.match(source, /user-select:\s*text/)
  assert.match(source, /\.parameter-name-cell:hover\s+\.parameter-name-copy/)
  assert.match(source, /\.parameter-name-copy:focus-visible/)
  assert.match(source, /复制失败，请手动选择参数名复制/)
})
```

- [ ] **Step 2: Run the contract test and verify RED**

Run:

```bash
cd web && node --test src/plugin/tr069/view/device/components/data-model-viewer.contract.test.js
```

Expected: failures because the component still contains “复制完整参数名”, value copy, `copyText`, and two-line name rendering.

- [ ] **Step 3: Replace the parameter and value cell templates**

Use one flex row for the name:

```vue
<div class="parameter-name-cell">
  <el-tooltip :content="scope.row.name" placement="top" :show-after="500">
    <span class="parameter-name-text font-mono text-xs">{{ scope.row.name }}</span>
  </el-tooltip>
  <el-tooltip content="复制参数名" placement="top">
    <el-button
      class="parameter-name-copy"
      link
      type="primary"
      :icon="CopyDocument"
      aria-label="复制参数名"
      @click.stop="copyParameterName(scope.row.name)"
    />
  </el-tooltip>
</div>
```

The value column becomes:

```vue
<span class="font-mono text-sm truncate">{{ formatValue(scope.row.valueJson) }}</span>
```

- [ ] **Step 4: Replace copy logic and styles**

Use a name-only function:

```js
const copyParameterName = async (name) => {
  if (name === null || name === undefined || String(name) === '') return
  try {
    if (!isSupported.value) throw new Error('clipboard is not supported')
    await copy(String(name))
    ElMessage.success('参数名已复制')
  } catch {
    ElMessage.error('复制失败，请手动选择参数名复制')
  }
}
```

Use compact, theme-compatible CSS:

```css
.parameter-name-cell {
  display: flex;
  align-items: center;
  min-width: 0;
  gap: 6px;
}
.parameter-name-text {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  user-select: text;
}
.parameter-name-copy {
  flex: none;
  opacity: 0;
  color: var(--el-color-primary);
  transition: opacity 0.15s ease;
}
.parameter-name-cell:hover .parameter-name-copy,
.parameter-name-copy:focus-visible {
  opacity: 1;
}
```

- [ ] **Step 5: Run focused and complete front-end verification**

Run:

```bash
cd web && node --test src/plugin/tr069/view/device/components/data-model-viewer.contract.test.js
cd web && node --test src/plugin/tr069/**/*.test.js
cd web && npm run build
```

Expected: all tests and build PASS.

- [ ] **Step 6: Commit the UI change**

```bash
git add web/src/plugin/tr069/view/device/components/data-model-viewer.vue web/src/plugin/tr069/view/device/components/data-model-viewer.contract.test.js
git commit -m "fix(tr069): simplify parameter name copy layout"
```
