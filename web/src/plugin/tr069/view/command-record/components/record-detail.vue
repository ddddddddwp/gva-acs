<template>
  <el-drawer v-model="visible" title="RPC 命令详情" size="78%" append-to-body>
    <div v-loading="loading" class="record-detail">
      <template v-if="command.commandId">
        <div class="detail-toolbar">
          <el-tag :type="statusView(command.status).type">{{ statusView(command.status).label }}</el-tag>
          <el-button v-if="canRetryRecord(command)" type="primary" :loading="retrying" @click="handleRetry">重新下发</el-button>
        </div>

        <el-descriptions title="命令概览" :column="3" border>
          <el-descriptions-item label="Command ID" :span="2">{{ command.commandId }}</el-descriptions-item>
          <el-descriptions-item label="功能">{{ operationLabel(command.operation) }}</el-descriptions-item>
          <el-descriptions-item label="设备">{{ command.deviceKey || command.deviceId }}</el-descriptions-item>
          <el-descriptions-item label="当前状态">{{ statusView(command.status).label }}</el-descriptions-item>
          <el-descriptions-item label="当前截止时间">{{ formatTime(command.phaseDeadlineAt) }}</el-descriptions-item>
          <el-descriptions-item label="Request ID">{{ command.requestId || '-' }}</el-descriptions-item>
          <el-descriptions-item label="CommandKey">{{ command.commandKey || '-' }}</el-descriptions-item>
          <el-descriptions-item label="重试来源">{{ command.retryOf || '-' }}</el-descriptions-item>
          <el-descriptions-item label="提交时间">{{ formatTime(command.createdAt) }}</el-descriptions-item>
          <el-descriptions-item label="发送时间">{{ formatTime(command.sentAt) }}</el-descriptions-item>
          <el-descriptions-item label="完成时间">{{ formatTime(command.finishedAt) }}</el-descriptions-item>
        </el-descriptions>

        <el-alert
          v-if="command.failureStage || command.faultCode || command.faultString"
          class="detail-section"
          :title="`失败阶段：${command.failureStage || '-'}；FaultCode：${command.faultCode || '-'}`"
          :description="command.faultString || '命令执行失败'"
          type="error"
          show-icon
          :closable="false"
        />

        <el-tabs class="detail-section">
          <el-tab-pane label="请求参数">
            <pre class="json-block">{{ formatJSON(command.params) }}</pre>
          </el-tab-pane>
          <el-tab-pane label="结构化结果">
            <pre class="json-block">{{ formatJSON(command.result) }}</pre>
          </el-tab-pane>
          <el-tab-pane :label="`状态时间线 (${events.length})`">
            <el-empty v-if="events.length === 0" description="暂无状态事件" />
            <el-timeline v-else>
              <el-timeline-item
                v-for="event in events"
                :key="event.id"
                :timestamp="formatTime(event.createdAt)"
                :type="statusView(event.toStatus).type"
                placement="top"
              >
                <el-card shadow="never">
                  <div class="event-title">
                    {{ event.eventType || '状态变化' }}
                    <el-tag v-if="event.toStatus" size="small" :type="statusView(event.toStatus).type">{{ statusView(event.toStatus).label }}</el-tag>
                  </div>
                  <div v-if="event.fromStatus || event.toStatus" class="event-meta">{{ event.fromStatus || '-' }} → {{ event.toStatus || '-' }}</div>
                  <div v-if="event.stage" class="event-meta">阶段：{{ event.stage }}</div>
                  <div v-if="event.message">{{ event.message }}</div>
                  <pre v-if="event.payload" class="json-block compact">{{ formatJSON(event.payload) }}</pre>
                </el-card>
              </el-timeline-item>
            </el-timeline>
          </el-tab-pane>
          <el-tab-pane :label="`完整 XML (${xmlRecords.length})`">
            <el-empty v-if="xmlRecords.length === 0" description="XML 已清理或尚未生成" />
            <el-tabs v-else tab-position="left">
              <el-tab-pane v-for="record in xmlRecords" :key="record.id" :label="`${directionLabel(record.direction)} · ${record.method}`">
                <el-descriptions :column="2" size="small" border>
                  <el-descriptions-item label="方向">{{ directionLabel(record.direction) }}</el-descriptions-item>
                  <el-descriptions-item label="时间">{{ formatTime(record.createdAt) }}</el-descriptions-item>
                  <el-descriptions-item label="CWMP ID">{{ record.cwmpId || '-' }}</el-descriptions-item>
                  <el-descriptions-item label="Request ID">{{ record.requestId || '-' }}</el-descriptions-item>
                </el-descriptions>
                <pre class="xml-block">{{ record.xml }}</pre>
              </el-tab-pane>
            </el-tabs>
          </el-tab-pane>
        </el-tabs>
      </template>
      <el-empty v-else-if="!loading" description="未找到命令详情" />
    </div>
  </el-drawer>
</template>

<script setup>
import { computed, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { getCommandRecordDetail, retryCommandRecord } from '@/plugin/tr069/api/command-record'
import {
  canRetryRecord,
  directionLabel,
  formatJSON,
  isDangerousOperation,
  operationLabel,
  statusView
} from '../record-view'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
  commandId: { type: String, default: '' },
  refreshKey: { type: Number, default: 0 }
})

const emit = defineEmits(['update:modelValue', 'retried'])
const visible = computed({
  get: () => props.modelValue,
  set: value => emit('update:modelValue', value)
})
const loading = ref(false)
const retrying = ref(false)
const detail = ref({ command: {}, events: [], xml: [] })
const command = computed(() => detail.value.command || {})
const events = computed(() => detail.value.events || [])
const xmlRecords = computed(() => detail.value.xml || [])

const formatTime = value => value ? new Date(value).toLocaleString('zh-CN', { hour12: false }) : '-'

const loadDetail = async () => {
  if (!visible.value || !props.commandId || loading.value) return
  loading.value = true
  try {
    const res = await getCommandRecordDetail(props.commandId)
    if (res.code === 0) detail.value = res.data || { command: {}, events: [], xml: [] }
  } finally {
    loading.value = false
  }
}

watch(() => [props.modelValue, props.commandId, props.refreshKey], ([open]) => {
  if (open) loadDetail()
}, { immediate: true })

const handleRetry = async () => {
  const dangerous = isDangerousOperation(command.value.operation)
  try {
    await ElMessageBox.confirm(
      dangerous
        ? `重新下发“${operationLabel(command.value.operation)}”可能影响设备业务，确定继续吗？`
        : `确定重新下发“${operationLabel(command.value.operation)}”吗？`,
      dangerous ? '危险操作确认' : '重新下发确认',
      { type: dangerous ? 'error' : 'warning', confirmButtonText: '确认下发', cancelButtonText: '取消' }
    )
  } catch {
    return
  }
  retrying.value = true
  try {
    const res = await retryCommandRecord(command.value.commandId)
    if (res.code !== 0) {
      ElMessage.error(res.msg || '重新下发失败')
      return
    }
    ElMessage.success(`命令已重新提交，commandId=${res.data?.commandId || '-'}`)
    emit('retried', res.data)
  } finally {
    retrying.value = false
  }
}
</script>

<style scoped>
.record-detail {
  min-height: 300px;
}
.detail-toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}
.detail-section {
  margin-top: 20px;
}
.json-block,
.xml-block {
  overflow: auto;
  max-height: 520px;
  padding: 14px;
  margin: 12px 0 0;
  color: var(--el-text-color-primary);
  background: var(--el-fill-color-light);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 4px;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 12px;
  line-height: 1.6;
  white-space: pre-wrap;
  word-break: break-all;
}
.json-block.compact {
  max-height: 180px;
  padding: 8px;
}
.event-title {
  display: flex;
  gap: 8px;
  align-items: center;
  font-weight: 600;
}
.event-meta {
  margin-top: 6px;
  color: var(--el-text-color-secondary);
}
</style>
