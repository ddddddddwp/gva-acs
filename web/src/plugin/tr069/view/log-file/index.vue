<template>
  <div>
    <div class="gva-search-box">
      <el-form :inline="true" :model="searchInfo">
        <el-form-item label="设备 ID">
          <el-input-number
            v-model="searchInfo.deviceId"
            :min="1"
            :controls="false"
            placeholder="精确设备 ID"
            clearable
          />
        </el-form-item>
        <el-form-item label="状态">
          <el-select v-model="searchInfo.status" placeholder="全部状态" clearable style="width: 140px">
            <el-option label="可下载" value="AVAILABLE" />
            <el-option label="接收中" value="RECEIVING" />
            <el-option label="接收失败" value="FAILED" />
          </el-select>
        </el-form-item>
        <el-form-item label="接收时间">
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
      <div class="log-file-toolbar">
        <span class="toolbar-hint">基站上传的 LOG 制品</span>
        <el-button :icon="Refresh" :loading="loading" @click="getTableData">刷新</el-button>
      </div>
      <el-table v-loading="loading" :data="tableData" row-key="artifactId">
        <el-table-column label="设备" min-width="170">
          <template #default="scope">
            <div>{{ scope.row.serialNumber || '-' }}</div>
            <div class="cell-secondary">ID {{ scope.row.deviceId }} · OUI {{ scope.row.oui || '-' }}</div>
          </template>
        </el-table-column>
        <el-table-column prop="originalName" label="文件名" min-width="200" show-overflow-tooltip>
          <template #default="scope">{{ scope.row.originalName || `${scope.row.artifactId}.bin` }}</template>
        </el-table-column>
        <el-table-column label="来源" width="110" align="center">
          <template #default="scope">
            <el-tag :type="sourceView(scope.row.source).type">{{ sourceView(scope.row.source).label }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="110" align="center">
          <template #default="scope">
            <el-tag :type="statusView(scope.row.status).type">{{ statusView(scope.row.status).label }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="大小" width="100" align="right">
          <template #default="scope">{{ formatBytes(scope.row.size) }}</template>
        </el-table-column>
        <el-table-column label="SHA-256" width="150">
          <template #default="scope">
            <el-tooltip :content="scope.row.sha256 || '-'" placement="top">
              <span class="digest">{{ shortSHA256(scope.row.sha256) }}</span>
            </el-tooltip>
          </template>
        </el-table-column>
        <el-table-column label="接收完成时间" width="180">
          <template #default="scope">{{ formatTime(scope.row.receivedAt || scope.row.createdAt) }}</template>
        </el-table-column>
        <el-table-column label="操作" width="100" fixed="right" align="center">
          <template #default="scope">
            <el-button
              type="primary"
              link
              :icon="Download"
              :disabled="!canDownload(scope.row)"
              :loading="downloadingId === scope.row.artifactId"
              @click="download(scope.row)"
            >下载</el-button>
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
          @current-change="getTableData"
          @size-change="getTableData"
        />
      </div>
    </div>
  </div>
</template>

<script setup>
import { onMounted, reactive, ref } from 'vue'
import { Download, Refresh, Search } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { downloadLogArtifact, getLogArtifactList } from '@/plugin/tr069/api/log-file'
import {
  canDownload,
  filenameFromDisposition,
  formatBytes,
  shortSHA256,
  sourceView,
  statusView,
  triggerBlobDownload
} from './log-file-view'

defineOptions({ name: 'Tr069LogFiles' })

const loading = ref(false)
const downloadingId = ref('')
const tableData = ref([])
const page = ref(1)
const pageSize = ref(10)
const total = ref(0)
const searchInfo = reactive({ deviceId: undefined, status: '', createdRange: [] })

const formatTime = value => value ? new Date(value).toLocaleString('zh-CN', { hour12: false }) : '-'

const getTableData = async () => {
  loading.value = true
  try {
    const res = await getLogArtifactList({
      page: page.value,
      pageSize: pageSize.value,
      deviceId: searchInfo.deviceId,
      channel: 'LOG',
      status: searchInfo.status,
      createdFrom: searchInfo.createdRange?.[0] || '',
      createdTo: searchInfo.createdRange?.[1] || ''
    })
    if (res.code === 0) {
      tableData.value = res.data?.list || []
      total.value = res.data?.total || 0
    } else {
      ElMessage.error(res.msg || '获取日志文件失败')
    }
  } catch {
    ElMessage.error('获取日志文件失败')
  } finally {
    loading.value = false
  }
}

const onSearch = () => {
  page.value = 1
  getTableData()
}

const onReset = () => {
  Object.assign(searchInfo, { deviceId: undefined, status: '', createdRange: [] })
  page.value = 1
  getTableData()
}

const download = async row => {
  if (!canDownload(row) || downloadingId.value) return
  downloadingId.value = row.artifactId
  try {
    const res = await downloadLogArtifact(row.artifactId)
    const blob = res?.data instanceof Blob ? res.data : res
    if (!(blob instanceof Blob)) throw new Error('invalid artifact response')
    const disposition = res?.headers?.['content-disposition'] || ''
    const fallback = row.originalName || `${row.artifactId}.bin`
    triggerBlobDownload(blob, filenameFromDisposition(disposition, fallback))
    ElMessage.success('下载成功')
  } catch {
    ElMessage.error('日志文件下载失败')
  } finally {
    downloadingId.value = ''
  }
}

onMounted(getTableData)
</script>

<style scoped>
.log-file-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 12px;
}

.toolbar-hint,
.cell-secondary {
  color: var(--el-text-color-secondary);
  font-size: 12px;
}

.digest {
  color: var(--el-text-color-regular);
  cursor: help;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
}
</style>
