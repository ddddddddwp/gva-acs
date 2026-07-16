import test from 'node:test'
import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'

const devicePageURL = new URL('./index.vue', import.meta.url)
const commandAPIURL = new URL('../../api/command.js', import.meta.url)

test('device list renders grouped RPC actions and keeps device deletion separate', async () => {
  const source = await readFile(devicePageURL, 'utf8')

  assert.match(source, /RPC_ACTION_GROUPS/)
  assert.match(source, /group\.actions/)
  assert.match(source, /command="deleteDevice"/)
  assert.doesNotMatch(source, /command="deleteObject"[^>]*>删除设备/)
})

test('frontend exposes one fixed API function for every RPC operation', async () => {
  const source = await readFile(commandAPIURL, 'utf8')
  const functions = [
    'getRPCMethods', 'getParameterValues', 'getParameterNames', 'getParameterAttributes',
    'setParameterValues', 'setParameterAttributes', 'addObject', 'deleteObject',
    'downloadFile', 'uploadFile', 'rebootDevice', 'factoryResetDevice'
  ]

  for (const name of functions) {
    assert.match(source, new RegExp(`export const ${name} =`), `missing typed command API ${name}`)
  }
})
