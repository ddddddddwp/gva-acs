<template>
  <div class="device-detail-container">
    <el-card class="device-info-card">
      <template #header>
        <div class="card-header">
          <span>设备基本信息</span>
          <el-tag v-if="deviceInfo.online" type="success">在线</el-tag>
          <el-tag v-else type="danger">离线</el-tag>
        </div>
      </template>
      <el-descriptions :column="3" border>
        <el-descriptions-item label="设备序列号">{{ deviceInfo.serialNumber }}</el-descriptions-item>
        <el-descriptions-item label="设备型号">{{ deviceInfo.modelName }}</el-descriptions-item>
        <el-descriptions-item label="厂商">{{ deviceInfo.manufacturer }}</el-descriptions-item>
        <el-descriptions-item label="软件版本">{{ deviceInfo.softwareVersion }}</el-descriptions-item>
        <el-descriptions-item label="硬件版本">{{ deviceInfo.hardwareVersion }}</el-descriptions-item>
        <el-descriptions-item label="MAC地址">{{ deviceInfo.macAddress }}</el-descriptions-item>
        <el-descriptions-item label="IP地址">{{ deviceInfo.ipAddress }}</el-descriptions-item>
        <el-descriptions-item label="最后连接时间">{{ formatDate(deviceInfo.lastConnectTime) }}</el-descriptions-item>
        <el-descriptions-item label="设备分组">{{ deviceInfo.groupName || '未分组' }}</el-descriptions-item>
      </el-descriptions>
      
      <div class="action-buttons">
        <el-button type="primary" @click="triggerAction('Reboot')">重启设备</el-button>
        <el-button type="warning" @click="triggerAction('FactoryReset')">恢复出厂设置</el-button>
        <el-button @click="refreshDeviceInfo">刷新信息</el-button>
      </div>
    </el-card>
    
    <el-tabs v-model="activeTab" class="detail-tabs">
      <el-tab-pane label="设备参数" name="parameters">
        <div class="search-box">
          <el-form :inline="true" :model="paramSearch">
            <el-form-item label="参数名称">
              <el-input v-model="paramSearch.name" placeholder="请输入参数名称" />
            </el-form-item>
            <el-form-item label="参数类别">
              <el-select v-model="paramSearch.category" placeholder="请选择类别" clearable>
                <el-option v-for="item in categoryOptions" :key="item" :label="item" :value="item" />
              </el-select>
            </el-form-item>
            <el-form-item>
              <el-button type="primary" @click="getParameterList">查询</el-button>
              <el-button @click="resetParamSearch">重置</el-button>
            </el-form-item>
          </el-form>
        </div>
        
        <el-table :data="parameterList" style="width: 100%" v-loading="loading.parameters">
          <el-table-column label="参数名称" prop="name" width="350" show-overflow-tooltip />
          <el-table-column label="参数值" prop="value" width="200" show-overflow-tooltip />
          <el-table-column label="类型" prop="type" width="100" />
          <el-table-column label="类别" prop="category" width="120" />
          <el-table-column label="读写权限" prop="writable" width="100">
            <template #default="scope">
              {{ scope.row.writable ? '可写' : '只读' }}
            </template>
          </el-table-column>
          <el-table-column label="更新时间" width="180">
            <template #default="scope">
              {{ formatDate(scope.row.updatedAt) }}
            </template>
          </el-table-column>
          <el-table-column label="操作" width="120">
            <template #default="scope">
              <el-button v-if="scope.row.writable" type="primary" link @click="openEditParamDialog(scope.row)">
                修改
              </el-button>
            </template>
          </el-table-column>
        </el-table>
        
        <div class="pagination">
          <el-pagination
            layout="total, sizes, prev, pager, next, jumper"
            :current-page="paramPagination.page"
            :page-size="paramPagination.pageSize"
            :page-sizes="[10, 20, 50, 100]"
            :total="paramPagination.total"
            @current-change="handleParamPageChange"
            @size-change="handleParamSizeChange"
          />
        </div>
      </el-tab-pane>
      
      <el-tab-pane label="设备事件" name="events">
        <el-table :data="eventsList" style="width: 100%" v-loading="loading.events">
          <el-table-column label="事件代码" prop="eventCode" width="150" />
          <el-table-column label="事件类型" prop="eventType" width="150" />
          <el-table-column label="事件描述" prop="description" show-overflow-tooltip />
          <el-table-column label="发生时间" width="180">
            <template #default="scope">
              {{ formatDate(scope.row.createdAt) }}
            </template>
          </el-table-column>
        </el-table>
        
        <div class="pagination">
          <el-pagination
            layout="total, sizes, prev, pager, next, jumper"
            :current-page="eventsPagination.page"
            :page-size="eventsPagination.pageSize"
            :page-sizes="[10, 20, 50, 100]"
            :total="eventsPagination.total"
            @current-change="handleEventsPageChange"
            @size-change="handleEventsSizeChange"
          />
        </div>
      </el-tab-pane>
      
      <el-tab-pane label="会话记录" name="sessions">
        <el-table :data="sessionsList" style="width: 100%" v-loading="loading.sessions">
          <el-table-column label="会话ID" prop="sessionID" width="280" show-overflow-tooltip />
          <el-table-column label="会话类型" prop="sessionType" width="120" />
          <el-table-column label="状态" prop="status" width="100">
            <template #default="scope">
              <el-tag :type="scope.row.status === 'active' ? 'success' : 'info'">
                {{ scope.row.status === 'active' ? '活跃' : '已结束' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="客户端IP" prop="clientIP" width="150" />
          <el-table-column label="开始时间" width="180">
            <template #default="scope">
              {{ formatDate(scope.row.startTime) }}
            </template>
          </el-table-column>
          <el-table-column label="结束时间" width="180">
            <template #default="scope">
              {{ scope.row.endTime ? formatDate(scope.row.endTime) : '-' }}
            </template>
          </el-table-column>
        </el-table>
        
        <div class="pagination">
          <el-pagination
            layout="total, sizes, prev, pager, next, jumper"
            :current-page="sessionsPagination.page"
            :page-size="sessionsPagination.pageSize"
            :page-sizes="[10, 20, 50, 100]"
            :total="sessionsPagination.total"
            @current-change="handleSessionsPageChange"
            @size-change="handleSessionsSizeChange"
          />
        </div>
      </el-tab-pane>
      
      <el-tab-pane label="操作日志" name="logs">
        <el-table :data="logsList" style="width: 100%" v-loading="loading.logs">
          <el-table-column label="操作类型" prop="operationType" width="120" />
          <el-table-column label="操作目标" prop="operationTarget" width="150" show-overflow-tooltip />
          <el-table-column label="操作内容" prop="operationContent" show-overflow-tooltip />
          <el-table-column label="操作结果" prop="operationResult" width="120" />
          <el-table-column label="操作人" prop="operatorName" width="120" />
          <el-table-column label="操作时间" width="180">
            <template #default="scope">
              {{ formatDate(scope.row.createdAt) }}
            </template>
          </el-table-column>
        </el-table>
        
        <div class="pagination">
          <el-pagination
            layout="total, sizes, prev, pager, next, jumper"
            :current-page="logsPagination.page"
            :page-size="logsPagination.pageSize"
            :page-sizes="[10, 20, 50, 100]"
            :total="logsPagination.total"
            @current-change="handleLogsPageChange"
            @size-change="handleLogsSizeChange"
          />
        </div>
      </el-tab-pane>
    </el-tabs>
    
    <!-- 修改参数对话框 -->
    <el-dialog v-model="editParamDialogVisible" title="修改参数" width="500px">
      <el-form ref="editParamFormRef" :model="editParamForm" label-width="100px">
        <el-form-item label="参数名称">
          <el-input v-model="editParamForm.name" disabled />
        </el-form-item>
        <el-form-item label="参数值">
          <el-input v-model="editParamForm.value" />
        </el-form-item>
        <el-form-item label="参数类型">
          <el-input v-model="editParamForm.type" disabled />
        </el-form-item>
      </el-form>
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="editParamDialogVisible = false">取消</el-button>
          <el-button type="primary" @click="submitEditParam">确定</el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted, watch } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { formatDate } from '@/utils/format'
import { 
  getDeviceDetail, 
  getDeviceParameters, 
  setDeviceParameter 
} from '@/api/tr069Management'
import {
  getDeviceStatus,
  triggerDeviceAction,
  getDeviceEvents,
  getDeviceSessions,
  getDeviceOperationLogs
} from '@/api/tr069Management'

const route = useRoute()
const serialNumber = ref('')
const deviceInfo = ref({})
const activeTab = ref('parameters')

// 加载状态
const loading = reactive({
  device: false,
  parameters: false,
  events: false,
  sessions: false,
  logs: false
})

// 参数相关
const parameterList = ref([])
const categoryOptions = ref([])
const paramSearch = reactive({
  name: '',
  category: ''
})
const paramPagination = reactive({
  page: 1,
  pageSize: 10,
  total: 0
})

// 事件相关
const eventsList = ref([])
const eventsPagination = reactive({
  page: 1,
  pageSize: 10,
  total: 0
})

// 会话相关
const sessionsList = ref([])
const sessionsPagination = reactive({
  page: 1,
  pageSize: 10,
  total: 0
})

// 日志相关
const logsList = ref([])
const logsPagination = reactive({
  page: 1,
  pageSize: 10,
  total: 0
})

// 修改参数相关
const editParamDialogVisible = ref(false)
const editParamForm = reactive({
  name: '',
  value: '',
  type: ''
})

// 初始化
onMounted(() => {
  serialNumber.value = route.params.id
  if (serialNumber.value) {
    getDeviceInfo()
    getParameterList()
  }
})

// 监听标签页切换
watch(activeTab, (newVal) => {
  if (newVal === 'events' && eventsList.value.length === 0) {
    getEventsList()
  } else if (newVal === 'sessions' && sessionsList.value.length === 0) {
    getSessionsList()
  } else if (newVal === 'logs' && logsList.value.length === 0) {
    getLogsList()
  }
})

// 获取设备信息
const getDeviceInfo = async () => {
  loading.device = true
  try {
    const res = await getDeviceDetail({ serialNumber: serialNumber.value })
    if (res.code === 0) {
      deviceInfo.value = res.data
      
      // 获取设备在线状态
      const statusRes = await getDeviceStatus({ serialNumber: serialNumber.value })
      if (statusRes.code === 0) {
        deviceInfo.value.online = statusRes.data
      }
    }
  } catch (err) {
    console.error('获取设备信息失败:', err)
  } finally {
    loading.device = false
  }
}

// 刷新设备信息
const refreshDeviceInfo = () => {
  getDeviceInfo()
  ElMessage.success('设备信息已刷新')
}

// 触发设备操作
const triggerAction = async (action) => {
  const actionMap = {
    'Reboot': '重启',
    'FactoryReset': '恢复出厂设置'
  }
  
  try {
    await ElMessageBox.confirm(
      `确定要对设备执行${actionMap[action]}操作吗？`,
      '操作确认',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )
    
    const res = await triggerDeviceAction({
      serialNumber: serialNumber.value,
      action: action
    })
    
    if (res.code === 0) {
      ElMessage.success(`${actionMap[action]}操作已提交`)
    }
  } catch (err) {
    if (err !== 'cancel') {
      console.error('触发设备操作失败:', err)
    }
  }
}

// 获取参数列表
const getParameterList = async () => {
  loading.parameters = true
  try {
    const res = await getDeviceParameters({
      serialNumber: serialNumber.value,
      name: paramSearch.name,
      category: paramSearch.category,
      page: paramPagination.page,
      pageSize: paramPagination.pageSize
    })
    
    if (res.code === 0) {
      parameterList.value = res.data.list
      paramPagination.total = res.data.total
      
      // 提取参数类别
      const categories = new Set()
      res.data.list.forEach(item => {
        if (item.category) {
          categories.add(item.category)
        }
      })
      categoryOptions.value = Array.from(categories)
    }
  } catch (err) {
    console.error('获取参数列表失败:', err)
  } finally {
    loading.parameters = false
  }
}

// 重置参数搜索
const resetParamSearch = () => {
  paramSearch.name = ''
  paramSearch.category = ''
  paramPagination.page = 1
  getParameterList()
}

// 获取事件列表
const getEventsList = async () => {
  loading.events = true
  try {
    const res = await getDeviceEvents({
      serialNumber: serialNumber.value,
      page: eventsPagination.page,
      pageSize: eventsPagination.pageSize
    })
    
    if (res.code === 0) {
      eventsList.value = res.data.list
      eventsPagination.total = res.data.total
    }
  } catch (err) {
    console.error('获取事件列表失败:', err)
  } finally {
    loading.events = false
  }
}

// 获取会话列表
const getSessionsList = async () => {
  loading.sessions = true
  try {
    const res = await getDeviceSessions({
      serialNumber: serialNumber.value,
      page: sessionsPagination.page,
      pageSize: sessionsPagination.pageSize
    })
    
    if (res.code === 0) {
      sessionsList.value = res.data.list
      sessionsPagination.total = res.data.total
    }
  } catch (err) {
    console.error('获取会话列表失败:', err)
  } finally {
    loading.sessions = false
  }
}

// 获取日志列表
const getLogsList = async () => {
  loading.logs = true
  try {
    const res = await getDeviceOperationLogs({
      serialNumber: serialNumber.value,
      page: logsPagination.page,
      pageSize: logsPagination.pageSize
    })
    
    if (res.code === 0) {
      logsList.value = res.data.list
      logsPagination.total = res.data.total
    }
  } catch (err) {
    console.error('获取日志列表失败:', err)
  } finally {
    loading.logs = false
  }
}

// 打开修改参数对话框
const openEditParamDialog = (row) => {
  editParamForm.name = row.name
  editParamForm.value = row.value
  editParamForm.type = row.type
  editParamDialogVisible.value = true
}

// 提交修改参数
const submitEditParam = async () => {
  try {
    const res = await setDeviceParameter({
      serialNumber: serialNumber.value,
      name: editParamForm.name,
      value: editParamForm.value
    })
    
    if (res.code === 0) {
      ElMessage.success('参数修改成功')
      editParamDialogVisible.value = false
      getParameterList()
    }
  } catch (err) {
    console.error('修改参数失败:', err)
  }
}

// 分页处理 - 参数
const handleParamPageChange = (page) => {
  paramPagination.page = page
  getParameterList()
}

const handleParamSizeChange = (size) => {
  paramPagination.pageSize = size
  paramPagination.page = 1
  getParameterList()
}

// 分页处理 - 事件
const handleEventsPageChange = (page) => {
  eventsPagination.page = page
  getEventsList()
}

const handleEventsSizeChange = (size) => {
  eventsPagination.pageSize = size
  eventsPagination.page = 1
  getEventsList()
}

// 分页处理 - 会话
const handleSessionsPageChange = (page) => {
  sessionsPagination.page = page
  getSessionsList()
}

const handleSessionsSizeChange = (size) => {
  sessionsPagination.pageSize = size
  sessionsPagination.page = 1
  getSessionsList()
}

// 分页处理 - 日志
const handleLogsPageChange = (page) => {
  logsPagination.page = page
  getLogsList()
}

const handleLogsSizeChange = (size) => {
  logsPagination.pageSize = size
  logsPagination.page = 1
  getLogsList()
}
</script>

<style lang="scss" scoped>
.device-detail-container {
  padding: 20px;
  background-color: #f5f7fa;
  min-height: calc(100vh - 120px);
  
  .device-info-card {
    margin-bottom: 20px;
    
    .card-header {
      display: flex;
      justify-content: space-between;
      align-items: center;
    }
    
    .action-buttons {
      margin-top: 20px;
      display: flex;
      justify-content: flex-end;
      gap: 10px;
    }
  }
  
  .detail-tabs {
    background-color: #fff;
    padding: 20px;
    border-radius: 4px;
    
    .search-box {
      margin-bottom: 20px;
    }
    
    .pagination {
      margin-top: 20px;
      display: flex;
      justify-content: flex-end;
    }
  }
}
</style>