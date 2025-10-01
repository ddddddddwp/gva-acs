<template>
  <div>
    <div class="gva-form-box">
      <el-card class="box-card">
        <template #header>
          <div class="card-header">
            <el-button @click="goBack" link icon="ArrowLeft">返回</el-button>
            <span class="card-header-text">设备详情</span>
          </div>
        </template>
        <el-row :gutter="20">
          <el-col :span="12">
            <el-descriptions title="基本信息" :column="1" border>
              <el-descriptions-item label="设备ID">{{ device.ID }}</el-descriptions-item>
              <el-descriptions-item label="序列号">{{ device.SerialNumber }}</el-descriptions-item>
              <el-descriptions-item label="设备型号">{{ device.Model }}</el-descriptions-item>
              <el-descriptions-item label="软件版本">{{ device.SoftwareVersion }}</el-descriptions-item>
              <el-descriptions-item label="硬件版本">{{ device.HardwareVersion }}</el-descriptions-item>
              <el-descriptions-item label="状态">
                <el-tag :type="device.Status === 'online' ? 'success' : 'danger'">
                  {{ device.Status === 'online' ? '在线' : '离线' }}
                </el-tag>
              </el-descriptions-item>
              <el-descriptions-item label="最后连接时间">{{ formatDate(device.LastConnected) }}</el-descriptions-item>
            </el-descriptions>
          </el-col>
          <el-col :span="12">
            <el-descriptions title="设备统计" :column="1" border v-if="stats">
              <el-descriptions-item label="总连接次数">{{ stats.TotalConnections }}</el-descriptions-item>
              <el-descriptions-item label="总事件数">{{ stats.TotalEvents }}</el-descriptions-item>
              <el-descriptions-item label="参数设置次数">{{ stats.ParameterSets }}</el-descriptions-item>
              <el-descriptions-item label="最长在线时长">{{ formatDuration(stats.LongestSession) }}</el-descriptions-item>
              <el-descriptions-item label="平均在线时长">{{ formatDuration(stats.AverageSession) }}</el-descriptions-item>
            </el-descriptions>
          </el-col>
        </el-row>

        <el-divider />

        <el-tabs v-model="activeTab">
          <el-tab-pane label="参数配置" name="parameters">
            <div class="parameter-actions">
              <el-button type="primary" @click="openSetParameterDialog" icon="Edit">设置参数</el-button>
            </div>
            <el-table :data="parameters" style="width: 100%" v-loading="loading">
              <el-table-column prop="name" label="参数名称" min-width="250" />
              <el-table-column prop="value" label="参数值" min-width="200" />
              <el-table-column prop="type" label="类型" width="100" />
              <el-table-column prop="writable" label="可写" width="80">
                <template #default="scope">
                  <el-tag :type="scope.row.writable ? 'success' : 'info'">
                    {{ scope.row.writable ? '是' : '否' }}
                  </el-tag>
                </template>
              </el-table-column>
              <el-table-column prop="lastUpdated" label="最后更新时间" width="180">
                <template #default="scope">
                  {{ formatDate(scope.row.lastUpdated) }}
                </template>
              </el-table-column>
            </el-table>
          </el-tab-pane>
          <el-tab-pane label="最近事件" name="events">
            <div class="event-actions">
              <el-button type="primary" @click="viewAllEvents" icon="View">查看所有事件</el-button>
            </div>
            <el-table :data="events" style="width: 100%" v-loading="loading">
              <el-table-column prop="ID" label="事件ID" width="80" />
              <el-table-column prop="Type" label="事件类型" width="150" />
              <el-table-column prop="Description" label="事件描述" min-width="250" />
              <el-table-column prop="CreatedAt" label="发生时间" width="180">
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
            </el-table>
          </el-tab-pane>
        </el-tabs>
      </el-card>
    </div>

    <!-- 设置参数对话框 -->
    <el-dialog v-model="setParameterDialogVisible" title="设置设备参数" width="600px">
      <el-form :model="parameterForm" label-width="120px">
        <el-form-item label="参数名称" required>
          <el-input v-model="parameterForm.name" placeholder="请输入参数名称，例如：InternetGatewayDevice.DeviceInfo.ProvisioningCode" />
        </el-form-item>
        <el-form-item label="参数值" required>
          <el-input v-model="parameterForm.value" placeholder="请输入参数值" />
        </el-form-item>
        <el-form-item label="参数类型">
          <el-select v-model="parameterForm.type" placeholder="请选择参数类型">
            <el-option label="字符串" value="string" />
            <el-option label="整数" value="int" />
            <el-option label="布尔值" value="boolean" />
            <el-option label="无符号整数" value="unsignedInt" />
            <el-option label="日期时间" value="dateTime" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <span class="dialog-footer">
          <el-button @click="setParameterDialogVisible = false">取消</el-button>
          <el-button type="primary" @click="setParameter">确定</el-button>
        </span>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, onMounted, reactive } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { getDeviceById, getDeviceEvents, setDeviceParameters, getDeviceStats } from '@/api/tr069Device'
import { formatDate } from '@/utils/format'

const route = useRoute()
const router = useRouter()
const deviceId = route.query.id

// 数据定义
const device = ref({})
const stats = ref(null)
const parameters = ref([])
const events = ref([])
const loading = ref(false)
const activeTab = ref('parameters')
const setParameterDialogVisible = ref(false)
const parameterForm = reactive({
  name: '',
  value: '',
  type: 'string'
})

// 初始化加载
onMounted(async () => {
  if (!deviceId) {
    ElMessage.error('设备ID不能为空')
    goBack()
    return
  }
  
  await loadDeviceDetail()
  await loadDeviceStats()
  await loadRecentEvents()
  
  // 这里应该加载设备参数，但API中没有定义获取参数列表的方法
  // 暂时使用模拟数据
  loadMockParameters()
})

// 加载设备详情
const loadDeviceDetail = async () => {
  loading.value = true
  try {
    const res = await getDeviceById(deviceId)
    if (res.code === 0 && res.data) {
      device.value = res.data
    } else {
      ElMessage.error(res.msg || '获取设备详情失败')
    }
  } catch (error) {
    console.error('获取设备详情出错:', error)
    ElMessage.error('获取设备详情失败')
  } finally {
    loading.value = false
  }
}

// 加载设备统计信息
const loadDeviceStats = async () => {
  try {
    const res = await getDeviceStats(deviceId)
    if (res.code === 0 && res.data) {
      stats.value = res.data
    }
  } catch (error) {
    console.error('获取设备统计信息出错:', error)
  }
}

// 加载最近事件
const loadRecentEvents = async () => {
  try {
    const res = await getDeviceEvents(deviceId, { page: 1, pageSize: 5 })
    if (res.code === 0 && res.data) {
      events.value = res.data.list || []
    }
  } catch (error) {
    console.error('获取设备事件出错:', error)
  }
}

// 模拟参数数据（实际项目中应该通过API获取）
const loadMockParameters = () => {
  parameters.value = [
    { name: 'InternetGatewayDevice.DeviceInfo.Manufacturer', value: 'Example Corp', type: 'string', writable: false, lastUpdated: new Date() },
    { name: 'InternetGatewayDevice.DeviceInfo.ModelName', value: device.value.Model, type: 'string', writable: false, lastUpdated: new Date() },
    { name: 'InternetGatewayDevice.DeviceInfo.SerialNumber', value: device.value.SerialNumber, type: 'string', writable: false, lastUpdated: new Date() },
    { name: 'InternetGatewayDevice.DeviceInfo.HardwareVersion', value: device.value.HardwareVersion, type: 'string', writable: false, lastUpdated: new Date() },
    { name: 'InternetGatewayDevice.DeviceInfo.SoftwareVersion', value: device.value.SoftwareVersion, type: 'string', writable: false, lastUpdated: new Date() },
    { name: 'InternetGatewayDevice.DeviceInfo.ProvisioningCode', value: 'ABC123', type: 'string', writable: true, lastUpdated: new Date() },
    { name: 'InternetGatewayDevice.ManagementServer.URL', value: 'https://acs.example.com/tr069', type: 'string', writable: true, lastUpdated: new Date() },
    { name: 'InternetGatewayDevice.ManagementServer.PeriodicInformEnable', value: 'true', type: 'boolean', writable: true, lastUpdated: new Date() },
    { name: 'InternetGatewayDevice.ManagementServer.PeriodicInformInterval', value: '86400', type: 'unsignedInt', writable: true, lastUpdated: new Date() }
  ]
}

// 格式化时长
const formatDuration = (seconds) => {
  if (!seconds) return '0秒'
  
  const hours = Math.floor(seconds / 3600)
  const minutes = Math.floor((seconds % 3600) / 60)
  const remainingSeconds = seconds % 60
  
  let result = ''
  if (hours > 0) result += `${hours}小时`
  if (minutes > 0) result += `${minutes}分钟`
  if (remainingSeconds > 0 || result === '') result += `${remainingSeconds}秒`
  
  return result
}

// 获取事件状态类型
const getEventStatusType = (status) => {
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

// 打开设置参数对话框
const openSetParameterDialog = () => {
  parameterForm.name = ''
  parameterForm.value = ''
  parameterForm.type = 'string'
  setParameterDialogVisible.value = true
}

// 设置参数
const setParameter = async () => {
  if (!parameterForm.name || !parameterForm.value) {
    ElMessage.warning('参数名称和值不能为空')
    return
  }
  
  try {
    const params = {
      name: parameterForm.name,
      value: parameterForm.value,
      type: parameterForm.type
    }
    
    const res = await setDeviceParameters(deviceId, params)
    if (res.code === 0) {
      ElMessage.success('参数设置成功')
      setParameterDialogVisible.value = false
      
      // 更新参数列表
      const index = parameters.value.findIndex(p => p.name === parameterForm.name)
      if (index >= 0) {
        parameters.value[index].value = parameterForm.value
        parameters.value[index].lastUpdated = new Date()
      } else {
        parameters.value.push({
          name: parameterForm.name,
          value: parameterForm.value,
          type: parameterForm.type,
          writable: true,
          lastUpdated: new Date()
        })
      }
    } else {
      ElMessage.error(res.msg || '参数设置失败')
    }
  } catch (error) {
    console.error('设置参数出错:', error)
    ElMessage.error('参数设置失败')
  }
}

// 查看所有事件
const viewAllEvents = () => {
  router.push({
    path: '/tr069/device/events',
    query: { deviceId }
  })
}

// 返回列表页
const goBack = () => {
  router.push('/tr069Device')
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
.parameter-actions, .event-actions {
  margin-bottom: 16px;
  display: flex;
  justify-content: flex-end;
}
.el-divider {
  margin: 24px 0;
}
</style>