<template>
  <div>
    <div class="gva-search-box">
      <el-form :inline="true" :model="searchInfo" class="demo-form-inline">
        <el-form-item label="设备序列号">
          <el-input v-model="searchInfo.serialNumber" placeholder="搜索序列号" />
        </el-form-item>
        <el-form-item label="状态">
          <el-select v-model="searchInfo.status" placeholder="请选择" clearable>
            <el-option label="活跃" value="Active" />
            <el-option label="已清除" value="Cleared" />
          </el-select>
        </el-form-item>
        <el-form-item label="级别">
          <el-select v-model="searchInfo.severity" placeholder="请选择" clearable>
            <el-option label="Critical" value="Critical" />
            <el-option label="Major" value="Major" />
            <el-option label="Minor" value="Minor" />
            <el-option label="Warning" value="Warning" />
            <el-option label="Indeterminate" value="Indeterminate" />
          </el-select>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" icon="search" @click="onSubmit">查询</el-button>
          <el-button icon="refresh" @click="onReset">重置</el-button>
        </el-form-item>
      </el-form>
    </div>
    <div class="gva-table-box">
      <el-table :data="tableData" row-key="ID" stripe style="width: 100%">
        <el-table-column label="发生时间" width="180">
          <template #default="scope">{{ formatDate(scope.row.startTime) }}</template>
        </el-table-column>
        <el-table-column label="设备序列号" prop="serialNumber" min-width="150" />
        <el-table-column label="告警ID" prop="alarmIdentifier" min-width="180" show-overflow-tooltip />
        <el-table-column label="级别" width="120">
          <template #default="scope">
            <el-tag :type="getSeverityType(scope.row.severity)">{{ scope.row.severity }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="具体问题" prop="specificProblem" min-width="200" show-overflow-tooltip />
        <el-table-column label="状态" width="100">
           <template #default="scope">
             <el-tag :type="scope.row.status === 'Active' ? 'danger' : 'info'">{{ scope.row.status }}</el-tag>
           </template>
        </el-table-column>
        <el-table-column label="清除时间" width="180">
          <template #default="scope">{{ formatDate(scope.row.endTime) }}</template>
        </el-table-column>
        <el-table-column label="操作" width="100" fixed="right">
          <template #default="scope">
            <el-button type="primary" link icon="view" @click="openDetail(scope.row)">详情</el-button>
          </template>
        </el-table-column>
      </el-table>
      <div class="gva-pagination">
        <el-pagination
          layout="total, sizes, prev, pager, next, jumper"
          :current-page="page"
          :page-size="pageSize"
          :page-sizes="[10, 30, 50, 100]"
          :total="total"
          @current-change="handleCurrentChange"
          @size-change="handleSizeChange"
        />
      </div>
    </div>

    <el-dialog v-model="dialogVisible" title="告警详情" width="50%">
        <el-descriptions :column="1" border>
            <el-descriptions-item label="设备序列号">{{ detailData.serialNumber }}</el-descriptions-item>
            <el-descriptions-item label="告警ID">{{ detailData.alarmIdentifier }}</el-descriptions-item>
            <el-descriptions-item label="状态">{{ detailData.status }}</el-descriptions-item>
            <el-descriptions-item label="级别">{{ detailData.severity }}</el-descriptions-item>
            <el-descriptions-item label="事件类型">{{ detailData.eventType }}</el-descriptions-item>
            <el-descriptions-item label="可能原因">{{ detailData.probableCause }}</el-descriptions-item>
            <el-descriptions-item label="具体问题">{{ detailData.specificProblem }}</el-descriptions-item>
            <el-descriptions-item label="附加文本">{{ detailData.additionalText }}</el-descriptions-item>
            <el-descriptions-item label="附加信息">{{ detailData.addInfo }}</el-descriptions-item>
            <el-descriptions-item label="发生时间">{{ formatDate(detailData.startTime) }}</el-descriptions-item>
            <el-descriptions-item label="清除时间">{{ formatDate(detailData.endTime) }}</el-descriptions-item>
        </el-descriptions>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { getAlarmList } from '@/plugin/tr069/api/alarm'
import { formatDate } from '@/utils/format'

defineOptions({
  name: 'Tr069Alarm'
})

const searchInfo = ref({})
const page = ref(1)
const pageSize = ref(10)
const total = ref(0)
const tableData = ref([])
const dialogVisible = ref(false)
const detailData = ref({})

const getSeverityType = (severity) => {
    switch (severity) {
        case 'Critical': return 'danger'
        case 'Major': return 'warning'
        case 'Minor': return 'warning'
        case 'Warning': return 'info' 
        default: return ''
    }
}

const getTableData = async() => {
  const table = await getAlarmList({ page: page.value, pageSize: pageSize.value, ...searchInfo.value })
  if (table.code === 0) {
    tableData.value = table.data.list
    total.value = table.data.total
    page.value = table.data.page
    pageSize.value = table.data.pageSize
  }
}

getTableData()

const onSubmit = () => {
  page.value = 1
  getTableData()
}

const onReset = () => {
  searchInfo.value = {}
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
</script>
