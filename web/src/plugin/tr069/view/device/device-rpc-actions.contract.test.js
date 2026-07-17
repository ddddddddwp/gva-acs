import test from 'node:test'
import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'

const devicePageURL = new URL('./index.vue', import.meta.url)
const commandAPIURL = new URL('../../api/command.js', import.meta.url)

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
