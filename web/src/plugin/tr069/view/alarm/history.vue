<template>
  <div>
    <div class="gva-search-box">
      <el-form :inline="true" :model="searchInfo" class="demo-form-inline">
        <el-form-item label="设备序列号">
          <el-input v-model="searchInfo.serialNumber" placeholder="搜索序列号" clearable />
        </el-form-item>
        <el-form-item label="告警来源">
          <el-select v-model="searchInfo.source" placeholder="请选择" clearable>
            <el-option label="HistoryEvent" value="HistoryEvent" />
            <el-option label="ExpeditedEvent" value="ExpeditedEvent" />
          </el-select>
        </el-form-item>
        <el-form-item label="严重程度">
          <el-select v-model="searchInfo.severity" placeholder="请选择" clearable>
            <el-option label="Critical" value="Critical" />
            <el-option label="Major" value="Major" />
            <el-option label="Minor" value="Minor" />
            <el-option label="Warning" value="Warning" />
          </el-select>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" icon="search" @click="onSubmit">查询</el-button>
          <el-button icon="refresh" @click="onReset">重置</el-button>
        </el-form-item>
      </el-form>
    </div>
    <div class="gva-table-box">
      <el-table :data="tableData" v-loading="loading" row-key="ID" stripe style="width: 100%">
        <el-table-column label="清除时间" width="180">
          <template #default="scope">{{ formatDate(scope.row.endTime) }}</template>
        </el-table-column>
        <el-table-column label="发生时间" width="180">
          <template #default="scope">{{ formatDate(scope.row.startTime) }}</template>
        </el-table-column>
        <el-table-column label="设备序列号" prop="serialNumber" min-width="150" />
        <el-table-column label="告警来源" prop="source" width="140">
          <template #default="scope">
            <el-tag :type="scope.row.source === 'HistoryEvent' ? 'info' : 'warning'">
              {{ scope.row.source }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="严重程度" width="120">
          <template #default="scope">
            <el-tag :type="getSeverityType(scope.row.perceivedSeverity)">
              {{ scope.row.perceivedSeverity }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="具体问题" prop="specificProblem" min-width="200" show-overflow-tooltip />
        <el-table-column label="操作" width="100" fixed="right">
          <template #default="scope">
            <el-button type="primary" link icon="view" @click="openDetail(scope.row)">详情</el-button>
          </template>
        </el-table-column>
      </el-table>
      <div class="gva-pagination">
        <el-pagination
          layout="total, sizes, prev, pager, next, jumper"
          v-model:current-page="page"
          v-model:page-size="pageSize"
          :page-sizes="[10, 30, 50, 100]"
          :total="total"
          @current-change="handleCurrentChange"
          @size-change="handleSizeChange"
        />
      </div>
    </div>

    <el-dialog v-model="dialogVisible" title="告警详情" width="60%">
      <el-descriptions :column="2" border>
        <el-descriptions-item label="设备序列号">{{ detailData.serialNumber }}</el-descriptions-item>
        <el-descriptions-item label="OUI">{{ detailData.oui }}</el-descriptions-item>
        <el-descriptions-item label="告警来源">{{ detailData.source }}</el-descriptions-item>
        <el-descriptions-item label="严重程度">
          <el-tag :type="getSeverityType(detailData.perceivedSeverity)">{{ detailData.perceivedSeverity }}</el-tag>
        </el-descriptions-item>
        <el-descriptions-item label="事件类型">{{ detailData.eventType }}</el-descriptions-item>
        <el-descriptions-item label="通知类型">{{ detailData.notificationType }}</el-descriptions-item>
        <el-descriptions-item label="可能原因" :span="2">{{ detailData.probableCause }}</el-descriptions-item>
        <el-descriptions-item label="具体问题" :span="2">{{ detailData.specificProblem }}</el-descriptions-item>
        <el-descriptions-item label="附加文本">{{ detailData.additionalText }}</el-descriptions-item>
        <el-descriptions-item label="管理对象">{{ detailData.managedObjectInstance }}</el-descriptions-item>
        <el-descriptions-item label="附加信息" :span="2">{{ detailData.additionalInformation }}</el-descriptions-item>
        <el-descriptions-item label="发生时间">{{ formatDate(detailData.startTime) }}</el-descriptions-item>
        <el-descriptions-item label="清除时间">{{ formatDate(detailData.endTime) }}</el-descriptions-item>
      </el-descriptions>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { getHistoryAlarms } from '@/plugin/tr069/api/alarm'
import { formatDate } from '@/utils/format'

defineOptions({
  name: 'Tr069HistoryAlarm'
})

// 响应式数据
const loading = ref(false)
const searchInfo = reactive({
  serialNumber: '',
  source: '',
  severity: ''
})
const page = ref(1)
const pageSize = ref(10)
const total = ref(0)
const tableData = ref([])
const dialogVisible = ref(false)
const detailData = ref({})

// 获取严重程度类型
const getSeverityType = (severity) => {
  switch (severity) {
    case 'Critical': return 'danger'
    case 'Major': return 'warning'
    case 'Minor': return 'warning'
    case 'Warning': return 'info'
    default: return ''
  }
}

// 获取表格数据
const getTableData = async () => {
  loading.value = true
  try {
    const res = await getHistoryAlarms({ page: page.value, pageSize: pageSize.value, ...searchInfo })
    if (res.code === 0) {
      tableData.value = res.data.list || []
      total.value = res.data.total || 0
    } else {
      ElMessage.error(res.msg || '获取数据失败')
    }
  } catch (error) {
    ElMessage.error('获取历史告警列表失败')
  } finally {
    loading.value = false
  }
}

const onSubmit = () => {
  page.value = 1
  getTableData()
}

const onReset = () => {
  searchInfo.serialNumber = ''
  searchInfo.source = ''
  searchInfo.severity = ''
  page.value = 1
  getTableData()
}

const handleSizeChange = (val) => {
  pageSize.value = val
  getTableData()
}

const handleCurrentChange = (val) => {
  page.value = val
  getTableData()
}

const openDetail = (row) => {
  detailData.value = row
  dialogVisible.value = true
}

// 生命周期
onMounted(() => {
  getTableData()
})
</script>
