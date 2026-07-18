<template>
  <div>
    <div class="gva-search-box">
      <el-form :inline="true" :model="searchInfo">
        <el-form-item label="设备">
          <el-input v-model="searchInfo.deviceSerial" placeholder="序列号或 Device Key" clearable />
        </el-form-item>
        <el-form-item label="功能">
          <el-select v-model="searchInfo.operation" placeholder="全部功能" clearable style="width: 180px">
            <el-option v-for="item in operationOptions" :key="item.value" :label="item.label" :value="item.value" />
          </el-select>
        </el-form-item>
        <el-form-item label="状态">
          <el-select v-model="searchInfo.status" placeholder="全部状态" clearable style="width: 150px">
            <el-option v-for="item in statusOptions" :key="item.value" :label="item.label" :value="item.value" />
          </el-select>
        </el-form-item>
        <el-form-item label="提交时间">
          <el-date-picker
            v-model="searchInfo.createdRange"
            type="datetimerange"
            value-format="YYYY-MM-DD HH:mm:ss"
            range-separator="至"
            start-placeholder="开始时间"
            end-placeholder="结束时间"
          />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :icon="Search" @click="onSearch">查询</el-button>
          <el-button :icon="Refresh" @click="onReset">重置</el-button>
        </el-form-item>
      </el-form>
    </div>

    <div class="gva-table-box">
      <div class="record-toolbar">
        <div class="refresh-hint">页面可见时每 5 秒自动刷新</div>
        <el-button :icon="Refresh" :loading="loading" @click="getTableData()">立即刷新</el-button>
      </div>
      <el-table :data="tableData" v-loading="loading" row-key="commandId">
        <el-table-column prop="deviceKey" label="设备" min-width="180" show-overflow-tooltip />
        <el-table-column label="功能" min-width="150">
          <template #default="scope">
            <div>{{ operationLabel(scope.row.operation) }}</div>
            <div class="cell-secondary">{{ scope.row.operation }}</div>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="130" align="center">
          <template #default="scope">
            <el-tag :type="statusView(scope.row.status).type">{{ statusView(scope.row.status).label }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="当前阶段" min-width="150">
          <template #default="scope">{{ phaseLabel(scope.row) }}</template>
        </el-table-column>
        <el-table-column label="阶段截止" width="180">
          <template #default="scope">{{ formatTime(scope.row.phaseDeadlineAt) }}</template>
        </el-table-column>
        <el-table-column label="提交时间" width="180">
          <template #default="scope">{{ formatTime(scope.row.createdAt) }}</template>
        </el-table-column>
        <el-table-column label="发送时间" width="180">
          <template #default="scope">{{ formatTime(scope.row.sentAt) }}</template>
        </el-table-column>
        <el-table-column label="完成时间" width="180">
          <template #default="scope">{{ formatTime(scope.row.finishedAt) }}</template>
        </el-table-column>
        <el-table-column label="耗时" width="100">
          <template #default="scope">{{ formatDuration(scope.row.createdAt, scope.row.finishedAt || scope.row.updatedAt) }}</template>
        </el-table-column>
        <el-table-column label="操作" width="90" fixed="right" align="center">
          <template #default="scope">
            <el-button type="primary" link @click="openDetail(scope.row)">详情</el-button>
          </template>
        </el-table-column>
      </el-table>

      <div class="gva-pagination">
        <el-pagination
          v-model:current-page="page"
          v-model:page-size="pageSize"
          :page-sizes="[10, 30, 50, 100]"
          :total="total"
          layout="total, sizes, prev, pager, next, jumper"
          @current-change="getTableData()"
          @size-change="getTableData()"
        />
      </div>
    </div>

    <record-detail
      v-model="detailVisible"
      :command-id="currentCommandId"
      :refresh-key="detailRefreshKey"
      @retried="handleRetried"
    />
  </div>
</template>

<script setup>
import { onBeforeUnmount, onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { getCommandRecordList } from '@/plugin/tr069/api/command-record'
import RecordDetail from './components/record-detail.vue'
import {
  formatDuration,
  operationLabel,
  operationOptions,
  statusOptions,
  statusView
} from './record-view'

const AUTO_REFRESH_MS = 5000
const loading = ref(false)
const tableData = ref([])
const page = ref(1)
const pageSize = ref(10)
const total = ref(0)
const detailVisible = ref(false)
const currentCommandId = ref('')
const detailRefreshKey = ref(0)
const searchInfo = reactive({
  deviceSerial: '',
  operation: '',
  status: '',
  createdRange: []
})
let autoRefreshTimer

const formatTime = value => value ? new Date(value).toLocaleString('zh-CN', { hour12: false }) : '-'

const phaseLabel = row => {
  if (row.failureStage) return row.failureStage
  const phases = {
    QUEUED: '等待前序命令',
    WAITING_DEVICE: '等待设备连接',
    BUILDING: '构造请求 XML',
    SENT: '等待 RPC 响应',
    WAITING_TRANSFER: '等待 TransferComplete',
    WAITING_REBOOT: '等待设备重新上线',
    COMPLETED: '已完成',
    FAILED: '执行失败',
    TIMEOUT: '等待超时'
  }
  return phases[row.status] || '-'
}

const getTableData = async (silent = false) => {
  if (loading.value) return
  loading.value = true
  try {
    const res = await getCommandRecordList({
      page: page.value,
      pageSize: pageSize.value,
      deviceSerial: searchInfo.deviceSerial,
      operation: searchInfo.operation,
      status: searchInfo.status,
      createdFrom: searchInfo.createdRange?.[0] || '',
      createdTo: searchInfo.createdRange?.[1] || ''
    })
    if (res.code === 0) {
      tableData.value = res.data?.list || []
      total.value = res.data?.total || 0
      detailRefreshKey.value += 1
    } else if (!silent) {
      ElMessage.error(res.msg || '获取 RPC 记录失败')
    }
  } catch {
    if (!silent) ElMessage.error('获取 RPC 记录失败')
  } finally {
    loading.value = false
  }
}

const onSearch = () => {
  page.value = 1
  getTableData()
}

const onReset = () => {
  Object.assign(searchInfo, { deviceSerial: '', operation: '', status: '', createdRange: [] })
  page.value = 1
  getTableData()
}

const openDetail = row => {
  currentCommandId.value = row.commandId
  detailVisible.value = true
}

const handleRetried = () => getTableData()

onMounted(() => {
  getTableData()
  autoRefreshTimer = window.setInterval(() => {
    if (document.visibilityState === 'visible') getTableData(true)
  }, AUTO_REFRESH_MS)
})

onBeforeUnmount(() => {
  if (autoRefreshTimer) window.clearInterval(autoRefreshTimer)
})
</script>

<style scoped>
.record-toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
}
.refresh-hint,
.cell-secondary {
  color: var(--el-text-color-secondary);
  font-size: 12px;
}
</style>
