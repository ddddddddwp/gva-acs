import test from 'node:test'
import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'

const devicePageURL = new URL('./index.vue', import.meta.url)

test('device page delegates light and dark surfaces to GVA theme classes', async () => {
  const source = await readFile(devicePageURL, 'utf8')

  assert.match(source, /class="gva-search-box"/)
  assert.match(source, /class="gva-table-box"/)
  assert.doesNotMatch(source, /background-color:\s*#fff/i)
})
