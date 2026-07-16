import test from 'node:test'
import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'

const source = await readFile(new URL('./data-model-viewer.vue', import.meta.url), 'utf8')

test('parameter tree is query-only', () => {
  assert.doesNotMatch(source, /fullDataModelSync/)
  assert.doesNotMatch(source, /openFullSync/)
  assert.doesNotMatch(source, /全量同步/)
})

test('parameter tree uses the device online boolean', () => {
  assert.match(source, /deviceRow\.online/)
  assert.doesNotMatch(source, /deviceRow\.status/)
})

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
