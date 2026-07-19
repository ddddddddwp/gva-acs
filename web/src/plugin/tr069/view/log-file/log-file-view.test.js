import assert from 'node:assert/strict'
import test from 'node:test'
import {
  canDownload,
  filenameFromDisposition,
  formatBytes,
  sourceView,
  triggerBlobDownload
} from './log-file-view.js'

test('log file view formats metadata and download state', () => {
  assert.equal(formatBytes(0), '0 B')
  assert.equal(formatBytes(1024), '1 KB')
  assert.equal(formatBytes(20 * 1024 * 1024), '20 MB')
  assert.equal(formatBytes(Math.round(20.35 * 1024 * 1024)), '20.35 MB')
  assert.deepEqual(sourceView('ACTIVE'), { label: '主动采集', type: 'primary' })
  assert.deepEqual(sourceView('PERIODIC'), { label: '周期上传', type: 'success' })
  assert.equal(canDownload({ fileId: 101 }), true)
  assert.equal(canDownload({ fileId: 0 }), false)
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
