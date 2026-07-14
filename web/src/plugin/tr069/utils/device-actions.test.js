import test from 'node:test'
import assert from 'node:assert/strict'

import { canIssueDeviceCommand, deviceParameterSyncPayload } from './device-actions.js'

test('only explicitly online devices may issue TR-069 commands', () => {
  assert.equal(canIssueDeviceCommand({ online: true }), true)
  assert.equal(canIssueDeviceCommand({ online: false }), false)
  assert.equal(canIssueDeviceCommand({ status: 'online' }), false)
  assert.equal(canIssueDeviceCommand(undefined), false)
})

test('quick parameter sync always requests the shallow Device root', () => {
  assert.deepEqual(deviceParameterSyncPayload(), { paths: ['Device.'] })
})
