<template>
  <div class="device-list-container">
    <div class="search-box">
      <el-form :inline="true" :model="searchInfo" class="demo-form-inline">
        <el-form-item label="序列号">
          <el-input v-model="searchInfo.serialNumber" placeholder="请输入序列号" clearable />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :icon="Search" @click="onSubmit">查询</el-button>
          <el-button :icon="Refresh" @click="onReset">重置</el-button>
        </el-form-item>
      </el-form>
    </div>

    <div class="table-box">
      <div class="btn-list">
        <el-button type="primary" :icon="Plus" @click="openDialog">录入设备 (白名单)</el-button>
      </div>

      <el-table :data="tableData" style="width: 100%" v-loading="loading">
        <el-table-column prop="ID" label="ID" width="80" />
        <el-table-column prop="serialNumber" label="序列号" min-width="180" show-overflow-tooltip />
        <el-table-column prop="oui" label="OUI" width="120" show-overflow-tooltip />
        <el-table-column prop="productClass" label="产品类别" min-width="150" show-overflow-tooltip />
        <el-table-column prop="softwareVer" label="软件版本" min-width="150" show-overflow-tooltip />
        <el-table-column prop="ip" label="IP地址" width="140" show-overflow-tooltip />
        <el-table-column label="在线状态" width="100" align="center">
          <template #default="scope">
            <el-tag :type="scope.row.online ? 'success' : 'info'" effect="light">
              {{ scope.row.online ? '在线' : '离线' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="220" fixed="right" align="center">
          <template #default="scope">
            <el-button type="primary" link :icon="View" @click="viewDetail(scope.row)">详情</el-button>
            <el-button type="primary" link :icon="Connection" @click="openDataModelFromRow(scope.row)">参数</el-button>
            <el-button type="danger" link :icon="Delete" class="delete-btn" @click="deleteRow(scope.row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>

      <div class="pagination">
        <el-pagination
          background
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

    <!-- 录入设备弹窗 -->
    <el-dialog title="录入设备 (白名单)" v-model="dialogFormVisible" width="500px">
      <el-form :model="formData" ref="addFormRef" :rules="rules" label-width="100px">
        <el-form-item label="序列号" prop="serialNumber">
          <el-input v-model="formData.serialNumber" autocomplete="off" />
        </el-form-item>
        <el-form-item label="OUI" prop="oui">
          <el-input v-model="formData.oui" autocomplete="off" />
        </el-form-item>
        <el-form-item label="备注">
          <el-input v-model="formData.remark" type="textarea" />
        </el-form-item>
      </el-form>
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="dialogFormVisible = false">取 消</el-button>
          <el-button type="primary" @click="enterDevice">确 定</el-button>
        </div>
      </template>
    </el-dialog>

    <!-- 设备详情抽屉 -->
    <el-drawer
      title="设备详情与配置"
      v-model="drawerVisible"
      direction="rtl"
      size="50%">
      <div class="drawer-content" v-if="currentRow.ID">
        <!-- 基础信息 -->
        <el-descriptions title="基础信息" :column="2" border class="mb-20">
          <el-descriptions-item label="序列号">{{ currentRow.serialNumber }}</el-descriptions-item>
          <el-descriptions-item label="OUI">{{ currentRow.oui }}</el-descriptions-item>
          <el-descriptions-item label="IP地址">{{ currentRow.ip }}</el-descriptions-item>
          <el-descriptions-item label="最后上线">{{ formatDate(currentRow.lastOnline) }}</el-descriptions-item>
        </el-descriptions>

        <div class="mb-4 flex flex-wrap gap-2">
          <el-button-group>
            <el-button type="primary" plain size="small" :icon="Refresh" @click="handleGetRPCMethods" :loading="rpcLoading">刷新RPC</el-button>
            <el-button type="primary" plain size="small" :icon="Download" @click="openGPVDialog">获取参数</el-button>
            <el-button type="primary" plain size="small" :icon="Edit" @click="openSPVDialog">设置参数</el-button>
          </el-button-group>
          <el-button-group class="ml-2">
             <el-button type="success" plain size="small" :icon="RefreshRight" @click="openFullSyncDialog">全量同步</el-button>
             <el-button type="primary" size="small" :icon="Connection" @click="openDataModelFromRow(currentRow)">参数树</el-button>
          </el-button-group>
        </div>

        <!-- 基站无线参数组件 -->
        <fap-info :device-id="currentRow.ID" />
      </div>
    </el-drawer>

    <el-dialog title="GetParameterValues" v-model="gpvDialogVisible" width="720px" append-to-body>
      <el-form :model="gpvForm" label-width="140px">
        <el-form-item label="参数路径(一行一个)">
          <el-input v-model="gpvForm.pathsText" type="textarea" :rows="10" />
        </el-form-item>
      </el-form>
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="gpvDialogVisible = false">取 消</el-button>
          <el-button type="primary" @click="submitGPV" :loading="gpvSubmitting">确 定</el-button>
        </div>
      </template>
    </el-dialog>

    <el-dialog title="SetParameterValues" v-model="spvDialogVisible" width="820px" append-to-body>
      <el-form :model="spvForm" label-width="140px">
        <el-form-item label="ParameterKey">
          <el-input v-model="spvForm.parameterKey" />
        </el-form-item>
        <el-form-item label="参数列表">
          <el-table :data="spvForm.parameters" size="small">
            <el-table-column label="Name" min-width="360">
              <template #default="scope">
                <el-input v-model="scope.row.name" />
              </template>
            </el-table-column>
            <el-table-column label="Type" min-width="160">
              <template #default="scope">
                <el-input v-model="scope.row.type" />
              </template>
            </el-table-column>
            <el-table-column label="Value" min-width="220">
              <template #default="scope">
                <el-input v-model="scope.row.value" />
              </template>
            </el-table-column>
            <el-table-column label="" width="80">
              <template #default="scope">
                <el-button type="text" size="small" @click="removeSPVRow(scope.$index)">删除</el-button>
              </template>
            </el-table-column>
          </el-table>
          <div class="mt-3">
            <el-button type="primary" size="small" @click="addSPVRow">新增一行</el-button>
          </div>
        </el-form-item>
      </el-form>
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="spvDialogVisible = false">取 消</el-button>
          <el-button type="primary" @click="submitSPV" :loading="spvSubmitting">确 定</el-button>
        </div>
      </template>
    </el-dialog>

    <data-model-viewer
      v-model="dmDrawerVisible"
      :row="currentRow"
    />
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { getDeviceList, createDevice, deleteDevice } from '@/plugin/tr069/api/device'
import { fullDataModelSync, getParameterValues, getRPCMethods, setParameterValues } from '@/plugin/tr069/api/command'
import { formatTimeToStr } from '@/utils/date'
import FapInfo from './components/fap-info.vue'
import DataModelViewer from './components/data-model-viewer.vue'

// 响应式数据
const loading = ref(false)
const tableData = ref([])
const page = ref(1)
const pageSize = ref(10)
const total = ref(0)
const searchInfo = reactive({ serialNumber: '' })
const dialogFormVisible = ref(false)
const formData = reactive({
  serialNumber: '',
  oui: '',
  remark: ''
})
const addFormRef = ref(null)
const drawerVisible = ref(false)
const currentRow = ref({})
const rpcLoading = ref(false)
const gpvDialogVisible = ref(false)
const gpvSubmitting = ref(false)
const gpvForm = reactive({ pathsText: '' })
const spvDialogVisible = ref(false)
const spvSubmitting = ref(false)
const spvForm = reactive({
  parameterKey: '',
  parameters: [{ name: '', type: 'xsd:string', value: '' }]
})
const fullSyncSubmitting = ref(false)
const dmDrawerVisible = ref(false)

// 表单校验规则
const rules = {
  serialNumber: [{ required: true, message: '请输入序列号', trigger: 'blur' }],
  oui: [{ required: true, message: '请输入OUI', trigger: 'blur' }]
}

// 格式化日期
const formatDate = (time) => {
  if (time && time !== '0001-01-01T00:00:00Z') {
    return formatTimeToStr(time)
  }
  return '-'
}

// 获取表格数据
const getTableData = async () => {
  loading.value = true
  try {
    const res = await getDeviceList({ page: page.value, pageSize: pageSize.value, ...searchInfo })
    if (res.code === 0) {
      tableData.value = res.data.list || []
      total.value = res.data.total || 0
    }
  } catch (error) {
    ElMessage.error('获取设备列表失败')
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
  getTableData()
}

const handleCurrentChange = (val) => {
  page.value = val
  getTableData()
}

const handleSizeChange = (val) => {
  pageSize.value = val
  getTableData()
}

const openDialog = () => {
  formData.serialNumber = ''
  formData.oui = ''
  formData.remark = ''
  dialogFormVisible.value = true
}

const enterDevice = async () => {
  try {
    await addFormRef.value.validate()
    const res = await createDevice({ ...formData })
    if (res.code === 0) {
      ElMessage.success('录入成功')
      dialogFormVisible.value = false
      getTableData()
    } else {
      ElMessage.error(res.msg || '录入失败')
    }
  } catch (error) {
    // 表单验证失败
  }
}

const deleteRow = async (row) => {
  try {
    await ElMessageBox.confirm('确定要删除该设备吗?', '提示', { type: 'warning' })
    const res = await deleteDevice(row.ID)
    if (res.code === 0) {
      ElMessage.success('删除成功')
      getTableData()
    } else {
      ElMessage.error(res.msg || '删除失败')
    }
  } catch (error) {
    // 用户取消
  }
}

const viewDetail = (row) => {
  currentRow.value = row
  drawerVisible.value = true
  gpvForm.pathsText = 'Device.DeviceInfo.SerialNumber'
  spvForm.parameterKey = ''
  spvForm.parameters = [{ name: '', type: 'xsd:string', value: '' }]
}

const handleGetRPCMethods = async () => {
  if (!currentRow.value.ID) return
  rpcLoading.value = true
  try {
    const res = await getRPCMethods(currentRow.value.ID)
    if (res.code === 0) {
      ElMessage.success(`已下发，commandId=${res.data?.commandId || '-'}`)
    } else {
      ElMessage.error(res.msg || '下发失败')
    }
  } finally {
    rpcLoading.value = false
  }
}

const openGPVDialog = () => {
  gpvDialogVisible.value = true
}

const submitGPV = async () => {
  if (!currentRow.value.ID) return
  const paths = (gpvForm.pathsText || '')
    .split('\n')
    .map(s => s.trim())
    .filter(Boolean)
  if (paths.length === 0) {
    ElMessage.warning('请填写参数路径')
    return
  }
  gpvSubmitting.value = true
  try {
    const res = await getParameterValues(currentRow.value.ID, { paths })
    if (res.code === 0) {
      ElMessage.success(`已下发，commandId=${res.data?.commandId || '-'}`)
      gpvDialogVisible.value = false
    } else {
      ElMessage.error(res.msg || '下发失败')
    }
  } finally {
    gpvSubmitting.value = false
  }
}

const openSPVDialog = () => {
  spvDialogVisible.value = true
}

const addSPVRow = () => {
  spvForm.parameters = [...spvForm.parameters, { name: '', type: 'xsd:string', value: '' }]
}

const removeSPVRow = (idx) => {
  spvForm.parameters = spvForm.parameters.filter((_, i) => i !== idx)
}

const submitSPV = async () => {
  if (!currentRow.value.ID) return
  const payload = {
    parameterKey: spvForm.parameterKey,
    parameters: (spvForm.parameters || [])
      .map(p => ({ ...p, name: (p.name || '').trim(), type: (p.type || '').trim() }))
      .filter(p => p.name)
  }
  if (payload.parameters.length === 0) {
    ElMessage.warning('请至少填写一条参数')
    return
  }
  spvSubmitting.value = true
  try {
    const res = await setParameterValues(currentRow.value.ID, payload)
    if (res.code === 0) {
      ElMessage.success(`已下发，commandId=${res.data?.commandId || '-'}`)
      spvDialogVisible.value = false
    } else {
      ElMessage.error(res.msg || '下发失败')
    }
  } finally {
    spvSubmitting.value = false
  }
}

const openFullSyncDialog = async () => {
  if (!currentRow.value?.ID) return
  if (fullSyncSubmitting.value) return
  fullSyncSubmitting.value = true
  try {
    const res = await fullDataModelSync(currentRow.value.ID, { paths: ['Device.'] })
    if (res.code === 0) {
      ElMessage.success('已下发全量同步请求')
    } else {
      ElMessage.error(res.msg || '下发失败')
    }
  } finally {
    fullSyncSubmitting.value = false
  }
}

const openDataModelFromRow = (row) => {
  currentRow.value = row
  dmDrawerVisible.value = true
}

// 生命周期
onMounted(() => {
  getTableData()
})
</script>

<style scoped>
.device-list-container {
  padding: 20px;
  background-color: #fff;
}
.search-box {
  margin-bottom: 20px;
}
.table-box {
  background-color: #fff;
}
.btn-list {
  margin-bottom: 10px;
}
.delete-btn {
  color: #f56c6c;
}
.drawer-content {
  padding: 20px;
}
.mb-20 {
  margin-bottom: 20px;
}
</style>
