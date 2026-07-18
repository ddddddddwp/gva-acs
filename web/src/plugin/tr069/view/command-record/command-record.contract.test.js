import test from 'node:test'
import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'

const listSource = await readFile(new URL('./index.vue', import.meta.url), 'utf8')
const detailSource = await readFile(new URL('./components/record-detail.vue', import.meta.url), 'utf8')

test('RPC command UI hides internal Command ID and legacy Request ID', () => {
  assert.doesNotMatch(listSource, /label="Command ID"|prop="commandId"/)
  assert.doesNotMatch(detailSource, /label="Command ID"|label="Request ID"|commandId=/)
  assert.doesNotMatch(detailSource, /\{\{\s*command\.retryOf\s*\|\|/)
  assert.match(detailSource, /label="CWMP ID"/)
})

test('RPC command detail keeps Command ID only as an opaque API parameter', () => {
  assert.match(listSource, /row-key="commandId"/)
  assert.match(listSource, /currentCommandId\.value\s*=\s*row\.commandId/)
  assert.match(detailSource, /getCommandRecordDetail\(props\.commandId\)/)
  assert.match(detailSource, /retryCommandRecord\(command\.value\.commandId\)/)
})

test('XML metadata exposes only its real CWMP ID', () => {
  assert.match(detailSource, /record\.cwmpId/)
  assert.doesNotMatch(detailSource, /record\.requestId/)
})
