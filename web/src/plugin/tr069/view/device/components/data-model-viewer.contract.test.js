import test from 'node:test'
import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'

const source = await readFile(new URL('./data-model-viewer.vue', import.meta.url), 'utf8')

test('parameter drawer separates local refresh from device synchronization', () => {
  assert.match(source, /刷新本地数据/)
  assert.match(source, /同步设备参数/)
  assert.match(source, /import \{ fullDataModelSync \} from '@\/plugin\/tr069\/api\/command'/)
  assert.match(source, /const syncingParameters = ref\(false\)/)
  assert.match(source, /:disabled="!deviceRow\.online"/)
  assert.match(source, /:loading="syncingParameters"/)
  assert.match(source, /if \(!props\.row\.ID \|\| !deviceRow\.value\.online \|\| syncingParameters\.value\) return/)
  assert.match(source, /await fullDataModelSync\(props\.row\.ID\)/)
  assert.doesNotMatch(source, /commandId=/)
  assert.match(source, /参数同步已下发/)
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

test('parameter table keeps only the compact parameter-name copy action', () => {
  assert.match(source, /content="复制参数名"/)
  assert.match(source, /aria-label="复制参数名"/)
  assert.match(source, /copyParameterName\(scope\.row\.name\)/)
  assert.match(source, /class="parameter-name-text/)
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
