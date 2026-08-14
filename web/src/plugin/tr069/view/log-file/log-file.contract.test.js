import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import test from 'node:test'

test('log page uses numeric file IDs and exact alphanumeric serial number filtering', async () => {
  const source = await readFile(new URL('./index.vue', import.meta.url), 'utf8')
  assert.match(source, /class="gva-search-box"/)
  assert.match(source, /class="gva-table-box"/)
  assert.match(source, /<el-input\s/)
  assert.doesNotMatch(source, /el-input-number/)
  assert.match(source, /serialNumber/)
  assert.match(source, /fileId/)
  assert.match(source, /downloadLogArtifact/)
  assert.doesNotMatch(source, /searchInfo\.status/)
  assert.doesNotMatch(source, /label="状态"/)
  assert.doesNotMatch(source, /SHA-256|shortSHA256|scope\.row\.oui|scope\.row\.deviceId/)
  assert.doesNotMatch(source, /scope\.row\.createdAt/)
  assert.doesNotMatch(source, /background(?:-color)?:\s*#fff/i)
})
