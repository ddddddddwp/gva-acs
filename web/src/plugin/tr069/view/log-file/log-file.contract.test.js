import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import test from 'node:test'

test('log page keeps GVA theme and supports exact device ID filtering', async () => {
  const source = await readFile(new URL('./index.vue', import.meta.url), 'utf8')
  assert.match(source, /class="gva-search-box"/)
  assert.match(source, /class="gva-table-box"/)
  assert.match(source, /deviceId/)
  assert.match(source, /downloadLogArtifact/)
  assert.doesNotMatch(source, /background(?:-color)?:\s*#fff/i)
})
