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
        <el-table-column label="操作" width="260" fixed="right" align="center">
          <template #default="scope">
            <el-button
              type="primary"
              link
              :disabled="!canIssueDeviceCommand(scope.row)"
              :loading="syncingDeviceIds.has(scope.row.ID)"
              @click="syncParameters(scope.row)"
            >同步参数</el-button>
            <el-button type="primary" link :icon="Connection" @click="openDataModelFromRow(scope.row)">参数</el-button>
            <el-dropdown trigger="click" @command="command => handleMoreCommand(command, scope.row)">
              <el-button type="primary" link>
                更多<el-icon class="el-icon--right"><ArrowDown /></el-icon>
              </el-button>
              <template #dropdown>
                <el-dropdown-menu>
                  <template v-for="group in RPC_ACTION_GROUPS" :key="group.label">
                    <el-dropdown-item disabled class="rpc-group-title">{{ group.label }}</el-dropdown-item>
                    <el-dropdown-item
                      v-for="action in group.actions"
                      :key="action.key"
                      :command="action.key"
                      :disabled="!canIssueRPCAction(scope.row, action)"
                    >{{ action.label }}</el-dropdown-item>
                  </template>
                  <el-dropdown-item command="deleteDevice" divided class="delete-device-action">删除设备</el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
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

    <rpc-command-dialog
      v-model="rpcDialogVisible"
      :row="currentRow"
      :action="currentRPCAction"
    />

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
import { fullDataModelSync } from '@/plugin/tr069/api/command'
import {
  RPC_ACTION_GROUPS,
  canIssueDeviceCommand,
  canIssueRPCAction,
  findRPCAction
} from '@/plugin/tr069/utils/device-actions'
import DataModelViewer from './components/data-model-viewer.vue'
import RpcCommandDialog from './components/rpc-command-dialog.vue'

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
const currentRow = ref({})
const currentRPCAction = ref()
const rpcDialogVisible = ref(false)
const syncingDeviceIds = ref(new Set())
const dmDrawerVisible = ref(false)

// 表单校验规则
const rules = {
  serialNumber: [{ required: true, message: '请输入序列号', trigger: 'blur' }],
  oui: [{ required: true, message: '请输入OUI', trigger: 'blur' }]
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

const syncParameters = async (row) => {
  if (!canIssueDeviceCommand(row) || syncingDeviceIds.value.has(row.ID)) return
  syncingDeviceIds.value = new Set(syncingDeviceIds.value).add(row.ID)
  try {
    const res = await fullDataModelSync(row.ID)
    if (res.code === 0) {
      ElMessage.success(`参数同步已下发，commandId=${res.data?.commandId || '-'}`)
    } else {
      ElMessage.error(res.msg || '下发失败')
    }
  } finally {
    const next = new Set(syncingDeviceIds.value)
    next.delete(row.ID)
    syncingDeviceIds.value = next
  }
}

const handleMoreCommand = (command, row) => {
  if (command === 'deleteDevice') {
    deleteRow(row)
    return
  }
  const action = findRPCAction(command)
  if (!action || !canIssueRPCAction(row, action)) return
  currentRow.value = row
  currentRPCAction.value = action
  rpcDialogVisible.value = true
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
:deep(.rpc-group-title) {
  font-size: 12px;
  font-weight: 600;
  color: var(--el-text-color-secondary);
  cursor: default;
  opacity: 1;
}
:deep(.delete-device-action) {
  color: var(--el-color-danger);
}
</style>
