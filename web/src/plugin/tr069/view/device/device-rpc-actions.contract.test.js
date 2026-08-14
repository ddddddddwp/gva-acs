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
  assert.match(source, /type="danger"[\s\S]*@click="deleteRow\(scope\.row\)"[\s\S]*scope\.row\.deleting \? '重试删除' : '删除'/)
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

test('reboot form keeps CommandKey server-owned', async () => {
  const dialog = await readFile(new URL('./components/rpc-command-dialog.vue', import.meta.url), 'utf8')
  const api = await readFile(commandAPIURL, 'utf8')

  assert.doesNotMatch(dialog, /GVA 将自动生成本次重启的唯一标识/)
  assert.doesNotMatch(dialog, /v-model="form\.commandKey"/)
  assert.doesNotMatch(dialog, /return\s*\{\s*commandKey:/)
  assert.match(dialog, /case 'reboot':\s*return undefined/)
  assert.match(api, /rebootDevice\s*=\s*\(deviceId\)\s*=>\s*postCommand\(deviceId,\s*'reboot'\)/)
})

test('log upload form submits only file type and delay', async () => {
  const dialog = await readFile(new URL('./components/rpc-command-dialog.vue', import.meta.url), 'utf8')

  assert.match(dialog, /v-else-if="action\.key === 'download'"/)
  assert.match(dialog, /v-else-if="action\.key === 'upload'"/)
  const uploadCase = dialog.match(/case 'upload':([\s\S]*?)case 'reboot':/)?.[1] || ''
  assert.match(uploadCase, /fileType:/)
  assert.match(uploadCase, /delaySeconds:/)
  assert.doesNotMatch(uploadCase, /url:|username:|password:/)
})

test('device deletion is permanent, row-scoped, and retryable', async () => {
  const source = await readFile(devicePageURL, 'utf8')

  assert.match(source, /永久删除该设备的参数树、RPC 记录、告警和全部日志文件/)
  assert.match(source, /deletingDeviceId/)
  assert.match(source, /scope\.row\.deleting/)
  assert.match(source, /重试删除/)
  assert.match(source, /:loading="deletingDeviceId === scope\.row\.ID"/)
  assert.match(source, /deletingDeviceId\.value = row\.ID/)
  assert.match(source, /finally\s*\{\s*deletingDeviceId\.value = 0/)
})
