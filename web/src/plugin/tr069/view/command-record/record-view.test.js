import test from 'node:test'
import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'

import {
  canRetryRecord,
  formatDuration,
  operationLabel,
  statusView
} from './record-view.js'

test('record status view uses concrete lifecycle labels and GVA tag colors', () => {
  assert.deepEqual(statusView('QUEUED'), { label: '排队中', type: 'info' })
  assert.deepEqual(statusView('WAITING_DEVICE'), { label: '等待设备', type: 'info' })
  assert.deepEqual(statusView('BUILDING'), { label: '构造中', type: 'warning' })
  assert.deepEqual(statusView('SENT'), { label: '已发送', type: 'warning' })
  assert.deepEqual(statusView('WAITING_TRANSFER'), { label: '等待传输完成', type: 'warning' })
  assert.deepEqual(statusView('WAITING_REBOOT'), { label: '设备已受理，等待重启', type: 'warning' })
  assert.deepEqual(statusView('COMPLETED'), { label: '完成', type: 'success' })
  assert.deepEqual(statusView('FAILED'), { label: '失败', type: 'danger' })
  assert.deepEqual(statusView('TIMEOUT'), { label: '超时', type: 'danger' })
})

test('record view uses concrete names for all device operations', () => {
  const methods = {
    GetRPCMethods: '查询设备能力',
    GetParameterValues: '获取参数',
    GetParameterNames: '获取参数名称',
    GetParameterAttributes: '获取参数属性',
    SetParameterValues: '配置参数',
    SetParameterAttributes: '配置参数属性',
    AddObject: '添加对象',
    DeleteObject: '删除对象',
    Download: '下载文件',
    Upload: '上传文件',
    Reboot: '重启设备',
    FactoryReset: '恢复出厂设置'
  }
  for (const [method, label] of Object.entries(methods)) {
    assert.equal(operationLabel(method), label)
  }
})

test('record duration and retry visibility follow terminal state', () => {
  assert.equal(formatDuration('2026-07-16T10:00:00Z', '2026-07-16T10:00:03Z'), '3秒')
  assert.equal(formatDuration('2026-07-16T10:00:00Z', '2026-07-16T10:02:03Z'), '2分3秒')
  assert.equal(formatDuration('', '2026-07-16T10:00:03Z'), '-')
  assert.equal(canRetryRecord({ status: 'FAILED' }), true)
  assert.equal(canRetryRecord({ status: 'TIMEOUT' }), true)
  assert.equal(canRetryRecord({ status: 'COMPLETED' }), false)
  assert.equal(canRetryRecord({ status: 'WAITING_REBOOT' }), false)
})

test('record page uses GVA theme surfaces and visible-page five-second refresh', async () => {
  const source = await readFile(new URL('./index.vue', import.meta.url), 'utf8')
  const detailSource = await readFile(new URL('./components/record-detail.vue', import.meta.url), 'utf8')

  assert.match(source, /class="gva-search-box"/)
  assert.match(source, /class="gva-table-box"/)
  assert.match(source, /AUTO_REFRESH_MS\s*=\s*5000/)
  assert.match(source, /document\.visibilityState\s*===\s*'visible'/)
  assert.match(detailSource, /<pre[^>]*>\{\{\s*record\.xml\s*\}\}<\/pre>/)
  assert.doesNotMatch(source + detailSource, /#fff|background:\s*white/i)
})
