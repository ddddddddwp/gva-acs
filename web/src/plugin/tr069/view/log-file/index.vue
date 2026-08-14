<template>
  <div>
    <div class="gva-search-box">
      <el-form :inline="true" :model="searchInfo">
        <el-form-item label="设备 ID">
          <el-input
            v-model.trim="searchInfo.serialNumber"
            placeholder="请输入完整设备序列号"
            clearable
          />
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
      <el-table v-loading="loading" :data="tableData" row-key="fileId">
        <el-table-column prop="fileId" label="文件 ID" width="100" align="center" />
        <el-table-column label="设备 ID" min-width="160">
          <template #default="scope">{{ scope.row.serialNumber || '-' }}</template>
        </el-table-column>
        <el-table-column prop="originalName" label="文件名" min-width="200" show-overflow-tooltip>
          <template #default="scope">{{ scope.row.originalName || `${scope.row.fileId}.bin` }}</template>
        </el-table-column>
        <el-table-column label="来源" width="110" align="center">
          <template #default="scope">
            <el-tag :type="sourceView(scope.row.source).type">{{ sourceView(scope.row.source).label }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="大小" width="100" align="right">
          <template #default="scope">{{ formatBytes(scope.row.size) }}</template>
        </el-table-column>
        <el-table-column label="接收完成时间" width="180">
          <template #default="scope">{{ formatTime(scope.row.receivedAt) }}</template>
        </el-table-column>
        <el-table-column label="操作" width="100" fixed="right" align="center">
          <template #default="scope">
            <el-button
              type="primary"
              link
              :icon="Download"
              :disabled="!canDownload(scope.row)"
              :loading="downloadingId === scope.row.fileId"
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
  sourceView,
  triggerBlobDownload
} from './log-file-view'

defineOptions({ name: 'Tr069LogFiles' })

const loading = ref(false)
const downloadingId = ref(0)
const tableData = ref([])
const page = ref(1)
const pageSize = ref(10)
const total = ref(0)
const searchInfo = reactive({ serialNumber: '', createdRange: [] })

const formatTime = value => value ? new Date(value).toLocaleString('zh-CN', { hour12: false }) : '-'

const getTableData = async () => {
  loading.value = true
  try {
    const res = await getLogArtifactList({
      page: page.value,
      pageSize: pageSize.value,
      serialNumber: searchInfo.serialNumber,
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
  Object.assign(searchInfo, { serialNumber: '', createdRange: [] })
  page.value = 1
  getTableData()
}

const download = async row => {
  if (!canDownload(row) || downloadingId.value) return
  downloadingId.value = row.fileId
  try {
    const res = await downloadLogArtifact(row.fileId)
    const blob = res?.data instanceof Blob ? res.data : res
    if (!(blob instanceof Blob)) throw new Error('invalid artifact response')
    const disposition = res?.headers?.['content-disposition'] || ''
    const fallback = row.originalName || `${row.fileId}.bin`
    triggerBlobDownload(blob, filenameFromDisposition(disposition, fallback))
    ElMessage.success('下载成功')
  } catch {
    ElMessage.error('日志文件下载失败')
  } finally {
    downloadingId.value = 0
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

.toolbar-hint {
  color: var(--el-text-color-secondary);
  font-size: 12px;
}
</style>
