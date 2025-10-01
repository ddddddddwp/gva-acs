<template>
  <div>
    <div class="gva-search-box">
      <el-form :inline="true" :model="searchInfo" class="demo-form-inline">
        <el-form-item label="设备序列号">
          <el-input v-model="searchInfo.serialNumber" placeholder="请输入设备序列号" />
        </el-form-item>
        <el-form-item label="设备状态">
          <el-select v-model="searchInfo.status" placeholder="请选择设备状态" clearable>
            <el-option label="在线" value="online" />
            <el-option label="离线" value="offline" />
          </el-select>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" icon="search" @click="onSubmit">查询</el-button>
          <el-button icon="refresh" @click="onReset">重置</el-button>
        </el-form-item>
      </el-form>
    </div>
    <div class="gva-table-box">
      <div class="gva-btn-list">
        <el-button type="danger" icon="delete" size="small" :disabled="!multipleSelection.length" @click="deleteSelected">批量删除</el-button>
        <el-button type="primary" icon="refresh" size="small" @click="getTableData">刷新</el-button>
      </div>
      <el-table
        ref="multipleTable"
        :data="tableData"
        style="width: 100%"
        tooltip-effect="dark"
        row-key="ID"
        @selection-change="handleSelectionChange"
      >
        <el-table-column type="selection" width="55" />
        <el-table-column align="left" label="设备ID" min-width="100" prop="ID" />
        <el-table-column align="left" label="设备序列号" min-width="150" prop="serialNumber" />
        <el-table-column align="left" label="设备型号" min-width="150" prop="model" />
        <el-table-column align="left" label="软件版本" min-width="120" prop="softwareVersion" />
        <el-table-column align="left" label="硬件版本" min-width="120" prop="hardwareVersion" />
        <el-table-column align="left" label="状态" min-width="100">
          <template #default="scope">
            <el-tag :type="scope.row.status === 'online' ? 'success' : 'danger'">
              {{ scope.row.status === 'online' ? '在线' : '离线' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column align="left" label="最后连接时间" min-width="180" prop="lastConnectTime" />
        <el-table-column align="left" label="操作" min-width="250">
          <template #default="scope">
            <el-button type="primary" link icon="view" size="small" @click="viewDetails(scope.row)">查看详情</el-button>
            <el-button type="primary" link icon="list" size="small" @click="viewEvents(scope.row)">查看事件</el-button>
            <el-button type="primary" link icon="setting" size="small" @click="setParameters(scope.row)">设置参数</el-button>
            <el-button type="danger" link icon="delete" size="small" @click="deleteRow(scope.row)">删除</el-button>
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
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { getDeviceList, deleteDevice } from '@/api/tr069Device'
import { useRouter } from 'vue-router'

const router = useRouter()

// 列表数据
const tableData = ref([])
const page = ref(1)
const pageSize = ref(10)
const total = ref(0)
const searchInfo = reactive({
  serialNumber: '',
  status: ''
})
const multipleSelection = ref([])

// 获取列表数据
const getTableData = async() => {
  const table = await getDeviceList({
    page: page.value,
    pageSize: pageSize.value,
    serialNumber: searchInfo.serialNumber,
    status: searchInfo.status
  })
  if (table.code === 0) {
    tableData.value = table.data.list
    total.value = table.data.total
    page.value = table.data.page
    pageSize.value = table.data.pageSize
  }
}

// 查询
const onSubmit = () => {
  page.value = 1
  getTableData()
}

// 重置
const onReset = () => {
  searchInfo.serialNumber = ''
  searchInfo.status = ''
  page.value = 1
  getTableData()
}

// 多选
const handleSelectionChange = (val) => {
  multipleSelection.value = val
}

// 批量删除
const deleteSelected = async() => {
  ElMessageBox.confirm('确定要删除选中的设备吗?', '提示', {
    confirmButtonText: '确定',
    cancelButtonText: '取消',
    type: 'warning'
  }).then(async() => {
    const ids = multipleSelection.value.map(item => item.ID)
    const res = await Promise.all(ids.map(id => deleteDevice(id)))
    if (res.every(item => item.code === 0)) {
      ElMessage({
        type: 'success',
        message: '删除成功'
      })
      getTableData()
    }
  })
}

// 删除单行
const deleteRow = async(row) => {
  ElMessageBox.confirm('确定要删除该设备吗?', '提示', {
    confirmButtonText: '确定',
    cancelButtonText: '取消',
    type: 'warning'
  }).then(async() => {
    const res = await deleteDevice(row.ID)
    if (res.code === 0) {
      ElMessage({
        type: 'success',
        message: '删除成功'
      })
      getTableData()
    }
  })
}

// 查看详情
const viewDetails = (row) => {
  router.push({
    name: 'tr069DeviceDetail',
    query: {
      id: row.ID
    }
  })
}

// 查看事件
const viewEvents = (row) => {
  router.push({
    name: 'tr069DeviceEvents',
    query: {
      id: row.ID,
      serialNumber: row.serialNumber
    }
  })
}

// 设置参数
const setParameters = (row) => {
  router.push({
    name: 'tr069DeviceParams',
    query: {
      id: row.ID,
      serialNumber: row.serialNumber
    }
  })
}

// 分页
const handleSizeChange = (val) => {
  pageSize.value = val
  getTableData()
}

const handleCurrentChange = (val) => {
  page.value = val
  getTableData()
}

onMounted(() => {
  getTableData()
})
</script>

<style scoped>
.gva-search-box {
  padding: 24px;
  background-color: var(--el-bg-color);
  border-radius: 4px;
  margin-bottom: 12px;
}

.gva-btn-list {
  margin-bottom: 12px;
}

.gva-table-box {
  padding: 24px;
  background-color: var(--el-bg-color);
  border-radius: 4px;
}

.gva-pagination {
  display: flex;
  justify-content: flex-end;
  margin-top: 12px;
}
</style>