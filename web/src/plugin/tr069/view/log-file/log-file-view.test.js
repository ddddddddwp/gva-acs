import assert from 'node:assert/strict'
import test from 'node:test'
import {
  canDownload,
  filenameFromDisposition,
  formatBytes,
  shortSHA256,
  sourceView,
  statusView,
  triggerBlobDownload
} from './log-file-view.js'

test('log file view formats metadata and download state', () => {
  assert.equal(formatBytes(0), '0 B')
  assert.equal(formatBytes(1024), '1 KB')
  assert.equal(formatBytes(20 * 1024 * 1024), '20 MB')
  assert.equal(shortSHA256('0123456789abcdef'), '01234567…cdef')
  assert.deepEqual(sourceView('ACTIVE'), { label: '主动采集', type: 'primary' })
  assert.deepEqual(sourceView('PERIODIC'), { label: '周期上传', type: 'success' })
  assert.deepEqual(statusView('AVAILABLE'), { label: '可下载', type: 'success' })
  assert.equal(canDownload({ status: 'AVAILABLE' }), true)
  assert.equal(canDownload({ status: 'RECEIVING' }), false)
})

test('download helpers use server filename and always revoke object URL', () => {
  assert.equal(filenameFromDisposition(`attachment; filename="bs-log.tar.gz"`, 'fallback.bin'), 'bs-log.tar.gz')
  assert.equal(filenameFromDisposition(`attachment; filename*=UTF-8''BS%20log.tar.gz`, 'fallback.bin'), 'BS log.tar.gz')

  const calls = []
  const anchor = {
    click: () => calls.push('click'),
    remove: () => calls.push('remove')
  }
  const documentAPI = {
    body: { appendChild: () => calls.push('append') },
    createElement: () => anchor
  }
  const urlAPI = {
    createObjectURL: () => 'blob:test-log',
    revokeObjectURL: value => calls.push(`revoke:${value}`)
  }
  triggerBlobDownload(new Blob(['log']), 'bs-log.tar.gz', urlAPI, documentAPI)
  assert.equal(anchor.href, 'blob:test-log')
  assert.equal(anchor.download, 'bs-log.tar.gz')
  assert.deepEqual(calls, ['append', 'click', 'remove', 'revoke:blob:test-log'])
})
