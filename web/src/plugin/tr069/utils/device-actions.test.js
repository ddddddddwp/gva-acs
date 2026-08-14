import test from 'node:test'
import assert from 'node:assert/strict'

import {
  RPC_ACTION_GROUPS,
  RPC_ACTION_MENUS,
  canIssueDeviceCommand,
  canIssueRPCAction,
  deviceParameterSyncPayload,
  findRPCAction
} from './device-actions.js'

test('only explicitly online devices may issue TR-069 commands', () => {
  assert.equal(canIssueDeviceCommand({ online: true }), true)
  assert.equal(canIssueDeviceCommand({ online: false }), false)
  assert.equal(canIssueDeviceCommand({ status: 'online' }), false)
  assert.equal(canIssueDeviceCommand(undefined), false)
})

test('quick parameter sync always requests the shallow Device root', () => {
  assert.deepEqual(deviceParameterSyncPayload(), { paths: ['Device.'] })
})

test('device actions expose four semantic groups through two menus', () => {
  assert.deepEqual(RPC_ACTION_GROUPS.map(group => group.label), ['查询', '配置', '文件', '维护'])
  assert.deepEqual(RPC_ACTION_MENUS.map(menu => menu.label), ['查询与配置', '文件与维护'])
  assert.deepEqual(
    RPC_ACTION_MENUS.map(menu => menu.groups.map(group => group.label)),
    [['查询', '配置'], ['文件', '维护']]
  )
  const actions = RPC_ACTION_GROUPS.flatMap(group => group.actions)
  assert.equal(actions.length, 12)
  assert.deepEqual(actions.map(action => action.label), [
    '查询设备能力', '获取参数', '获取参数名称', '获取参数属性',
    '配置参数', '配置参数属性', '添加对象', '删除对象实例',
    '下载文件', '上传文件', '重启设备', '恢复出厂设置'
  ])
  assert.equal(new Set(actions.map(action => action.key)).size, 12)
  assert.equal(actions.some(action => action.key === 'deleteDevice'), false)
})

test('dangerous RPC actions use danger confirmation and device deletion is not an RPC action', () => {
  assert.equal(findRPCAction('deleteObject').confirm, 'danger')
  assert.equal(findRPCAction('reboot').confirm, 'danger')
  assert.equal(findRPCAction('factoryReset').confirm, 'danger')
  assert.equal(findRPCAction('deleteDevice'), undefined)
})

test('RPC actions require online state and advertised capability except capability discovery', () => {
  const online = { online: true, rpcMethods: ['GetParameterValues', 'Reboot'] }
  assert.equal(canIssueRPCAction(online, findRPCAction('getRPCMethods')), true)
  assert.equal(canIssueRPCAction(online, findRPCAction('getParameterValues')), true)
  assert.equal(canIssueRPCAction(online, findRPCAction('reboot')), true)
  assert.equal(canIssueRPCAction(online, findRPCAction('upload')), false)
  assert.equal(canIssueRPCAction({ ...online, online: false }, findRPCAction('getRPCMethods')), false)
})
