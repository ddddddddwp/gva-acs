<template>
  <div>
    <div class="gva-form-box">
      <el-card class="box-card">
        <template #header>
          <div class="card-header">
            <el-button @click="goBack" link icon="ArrowLeft">返回</el-button>
            <span class="card-header-text">设备事件记录</span>
            <span v-if="device.SerialNumber" class="device-info">
              ({{ device.SerialNumber }} - {{ device.Model }})
            </span>
          </div>
        </template>
        
        <!-- 搜索区域 -->
        <el-form :model="searchInfo" ref="searchForm" :inline="true">
          <el-form-item label="事件类型">
            <el-select v-model="searchInfo.eventType" placeholder="请选择事件类型" clearable>
              <el-option label="所有类型" value="" />
              <el-option label="连接" value="Connection" />
              <el-option label="参数变更" value="ParameterChange" />
              <el-option label="告警" value="Alarm" />
              <el-option label="传输完成" value="TransferComplete" />
              <el-option label="下载请求" value="DownloadRequest" />
              <el-option label="重启" value="Reboot" />
              <el-option label="其他" value="Other" />
            </el-select>
          </el-form-item>
          <el-form-item label="状态">
            <el-select v-model="searchInfo.status" placeholder="请选择状态" clearable>
              <el-option label="所有状态" value="" />
              <el-option label="成功" value="success" />
              <el-option label="错误" value="error" />
              <el-option label="警告" value="warning" />
              <el-option label="信息" value="info" />
              <el-option label="处理中" value="processing" />
            </el-select>
          </el-form-item>
          <el-form-item label="时间范围">
            <el-date-picker
              v-model="dateRange"
              type="daterange"
              range-separator="至"
              start-placeholder="开始日期"
              end-placeholder="结束日期"
              format="YYYY-MM-DD"
              value-format="YYYY-MM-DD"
              :shortcuts="dateShortcuts"
            />
          </el-form-item>
          <el-form-item>
            <el-button type="primary" icon="Search" @click="onSubmit">查询</el-button>
            <el-button icon="Refresh" @click="onReset">重置</el-button>
          </el-form-item>
        </el-form>
        
        <!-- 事件列表 -->
        <el-table
          :data="tableData"
          style="width: 100%"
          v-loading="loading"
          @sort-change="sortChange"
        >
          <el-table-column prop="ID" label="事件ID" width="80" sortable="custom" />
          <el-table-column prop="Type" label="事件类型" width="150" sortable="custom" />
          <el-table-column prop="Description" label="事件描述" min-width="250" show-overflow-tooltip />
          <el-table-column prop="CreatedAt" label="发生时间" width="180" sortable="custom">
            <template #default="scope">
              {{ formatDate(scope.row.CreatedAt) }}
            </template>
          </el-table-column>
          <el-table-column prop="Status" label="状态" width="100">
            <template #default="scope">
              <el-tag :type="getEventStatusType(scope.row.Status)">
                {{ scope.row.Status }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="操作" width="120">
            <template #default="scope">
              <el-button type="primary" link icon="View" @click="viewEventDetail(scope.row)">
                详情
              </el-button>
            </template>
          </el-table-column>
        </el-table>
        
        <!-- 分页 -->
        <div class="pagination-container">
          <el-pagination
            layout="total, sizes, prev, pager, next, jumper"
            :current-page="page"
            :page-size="pageSize"
            :page-sizes="[10, 20, 50, 100]"
            :total="total"
            @current-change="handleCurrentChange"
            @size-change="handleSizeChange"
          />
        </div>
      </el-card>
    </div>
    
    <!-- 事件详情对话框 -->
    <el-dialog v-model="eventDetailVisible" title="事件详情" width="700px">
      <el-descriptions :column="1" border>
        <el-descriptions-item label="事件ID">{{ currentEvent.ID }}</el-descriptions-item>
        <el-descriptions-item label="设备ID">{{ currentEvent.DeviceID }}</el-descriptions-item>
        <el-descriptions-item label="事件类型">{{ currentEvent.Type }}</el-descriptions-item>
        <el-descriptions-item label="事件描述">{{ currentEvent.Description }}</el-descriptions-item>
        <el-descriptions-item label="状态">
          <el-tag :type="getEventStatusType(currentEvent.Status)">
            {{ currentEvent.Status }}
          </el-tag>
        </el-descriptions-item>
        <el-descriptions-item label="发生时间">{{ formatDate(currentEvent.CreatedAt) }}</el-descriptions-item>
        <el-descriptions-item label="详细数据">
          <pre class="event-data">{{ formatEventData(currentEvent.Data) }}</pre>
        </el-descriptions-item>
      </el-descriptions>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { getDeviceEvents, getDeviceById } from '@/api/tr069Device'
import { formatDate } from '@/utils/format'

const route = useRoute()
const router = useRouter()
const deviceId = route.query.deviceId

// 数据定义
const device = ref({})
const loading = ref(false)
const tableData = ref([])
const page = ref(1)
const pageSize = ref(10)
const total = ref(0)
const dateRange = ref([])
const eventDetailVisible = ref(false)
const currentEvent = ref({})

// 搜索条件
const searchInfo = reactive({
  eventType: '',
  status: '',
  startDate: '',
  endDate: ''
})

// 日期快捷选项
const dateShortcuts = [
  {
    text: '最近一周',
    value: () => {
      const end = new Date()
      const start = new Date()
      start.setTime(start.getTime() - 3600 * 1000 * 24 * 7)
      return [start, end]
    }
  },
  {
    text: '最近一个月',
    value: () => {
      const end = new Date()
      const start = new Date()
      start.setTime(start.getTime() - 3600 * 1000 * 24 * 30)
      return [start, end]
    }
  },
  {
    text: '最近三个月',
    value: () => {
      const end = new Date()
      const start = new Date()
      start.setTime(start.getTime() - 3600 * 1000 * 24 * 90)
      return [start, end]
    }
  }
]

// 监听日期范围变化
const watchDateRange = computed(() => {
  if (dateRange.value && dateRange.value.length === 2) {
    searchInfo.startDate = dateRange.value[0]
    searchInfo.endDate = dateRange.value[1]
  } else {
    searchInfo.startDate = ''
    searchInfo.endDate = ''
  }
  return dateRange.value
})

// 初始化加载
onMounted(async () => {
  if (deviceId) {
    await loadDeviceInfo()
  }
  getTableData()
})

// 加载设备信息
const loadDeviceInfo = async () => {
  try {
    const res = await getDeviceById(deviceId)
    if (res.code === 0 && res.data) {
      device.value = res.data
    }
  } catch (error) {
    console.error('获取设备信息出错:', error)
  }
}

// 获取表格数据
const getTableData = async () => {
  loading.value = true
  try {
    // 构建查询参数
    const params = {
      page: page.value,
      pageSize: pageSize.value,
      ...searchInfo
    }
    
    // 如果有设备ID，则按设备ID查询
    const res = deviceId 
      ? await getDeviceEvents(deviceId, params)
      : await getDeviceEvents(null, params) // 假设API支持不传设备ID查询所有设备的事件
    
    if (res.code === 0) {
      tableData.value = res.data.list || []
      total.value = res.data.total || 0
    } else {
      ElMessage.error(res.msg || '获取事件数据失败')
    }
  } catch (error) {
    console.error('获取事件数据出错:', error)
    ElMessage.error('获取事件数据失败')
  } finally {
    loading.value = false
  }
}

// 提交查询
const onSubmit = () => {
  page.value = 1
  getTableData()
}

// 重置查询条件
const onReset = () => {
  searchInfo.eventType = ''
  searchInfo.status = ''
  dateRange.value = []
  searchInfo.startDate = ''
  searchInfo.endDate = ''
  page.value = 1
  getTableData()
}

// 处理分页变化
const handleCurrentChange = (val) => {
  page.value = val
  getTableData()
}

// 处理每页条数变化
const handleSizeChange = (val) => {
  pageSize.value = val
  page.value = 1
  getTableData()
}

// 处理排序变化
const sortChange = ({ prop, order }) => {
  if (prop && order) {
    const sortOrder = order === 'ascending' ? 'asc' : 'desc'
    searchInfo.orderKey = prop
    searchInfo.orderType = sortOrder
  } else {
    searchInfo.orderKey = ''
    searchInfo.orderType = ''
  }
  getTableData()
}

// 获取事件状态类型
const getEventStatusType = (status) => {
  if (!status) return 'info'
  
  const statusMap = {
    'success': 'success',
    'error': 'danger',
    'warning': 'warning',
    'info': 'info',
    'pending': 'info',
    'processing': 'warning'
  }
  return statusMap[status.toLowerCase()] || 'info'
}

// 查看事件详情
const viewEventDetail = (row) => {
  currentEvent.value = row
  eventDetailVisible.value = true
}

// 格式化事件数据
const formatEventData = (data) => {
  if (!data) return '无数据'
  
  try {
    if (typeof data === 'string') {
      // 尝试解析JSON字符串
      const parsed = JSON.parse(data)
      return JSON.stringify(parsed, null, 2)
    } else {
      // 已经是对象，直接格式化
      return JSON.stringify(data, null, 2)
    }
  } catch (error) {
    // 如果不是有效的JSON，直接返回原始字符串
    return data
  }
}

// 返回上一页
const goBack = () => {
  if (deviceId) {
    router.push({
      path: '/tr069/device/detail',
      query: { id: deviceId }
    })
  } else {
    router.push('/tr069Device')
  }
}
</script>

<style scoped>
.gva-form-box {
  padding: 24px;
}
.box-card {
  box-shadow: 0 1px 4px rgba(0, 21, 41, 0.08);
}
.card-header {
  display: flex;
  align-items: center;
}
.card-header-text {
  font-size: 18px;
  font-weight: bold;
  margin-left: 8px;
}
.device-info {
  margin-left: 12px;
  font-size: 14px;
  color: #606266;
}
.pagination-container {
  margin-top: 20px;
  display: flex;
  justify-content: flex-end;
}
.event-data {
  background-color: #f5f7fa;
  padding: 10px;
  border-radius: 4px;
  max-height: 300px;
  overflow-y: auto;
  font-family: monospace;
  white-space: pre-wrap;
  word-break: break-all;
}
</style>