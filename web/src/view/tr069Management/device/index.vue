<template>
  <div class="device-list-container">
    <div class="gva-search-box">
      <el-form :inline="true" :model="searchInfo">
        <el-form-item label="设备序列号">
          <el-input v-model="searchInfo.serialNumber" placeholder="请输入设备序列号" />
        </el-form-item>
        <el-form-item label="设备状态">
          <el-select v-model="searchInfo.status" placeholder="请选择设备状态" clearable>
            <el-option label="在线" value="online" />
            <el-option label="离线" value="offline" />
            <el-option label="未知" value="unknown" />
          </el-select>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="getTableData">查询</el-button>
          <el-button @click="resetSearch">重置</el-button>
        </el-form-item>
      </el-form>
    </div>
    
    <div class="gva-table-box">
      <div class="gva-btn-list">
        <el-button type="primary" @click="openDialog('add')">新增</el-button>
        <el-button type="danger" @click="batchDeleteDevices">批量删除</el-button>
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
        <el-table-column label="序列号" prop="serialNumber" width="180" />
        <el-table-column label="厂商" prop="manufacturer" width="120" />
        <el-table-column label="型号" prop="modelName" width="120" />
        <el-table-column label="状态" width="100">
          <template #default="scope">
            <el-tag :type="scope.row.status === 'online' ? 'success' : scope.row.status === 'offline' ? 'danger' : 'info'">
              {{ scope.row.status === 'online' ? '在线' : scope.row.status === 'offline' ? '离线' : '未知' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="IP地址" prop="ipAddress" width="140" />
        <el-table-column label="最后通信时间" width="180">
          <template #default="scope">
            {{ formatDate(scope.row.lastInform) }}
          </template>
        </el-table-column>
        <el-table-column label="操作">
          <template #default="scope">
            <el-button type="primary" link @click="viewDeviceDetail(scope.row)">查看</el-button>
            <el-button type="primary" link @click="openDialog('edit', scope.row)">编辑</el-button>
            <el-button type="primary" link @click="viewDeviceParameters(scope.row)">参数</el-button>
            <el-popconfirm title="确定要删除此设备吗?" @confirm="deleteDevice(scope.row)">
              <template #reference>
                <el-button type="primary" link>删除</el-button>
              </template>
            </el-popconfirm>
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
    
    <!-- 设备表单对话框 -->
    <el-dialog v-model="dialogFormVisible" :title="dialogTitle" width="600px">
      <el-form ref="deviceFormRef" :model="deviceForm" label-width="120px" :rules="rules">
        <el-form-item label="序列号" prop="serialNumber">
          <el-input v-model="deviceForm.serialNumber" placeholder="请输入设备序列号" />
        </el-form-item>
        <el-form-item label="厂商" prop="manufacturer">
          <el-input v-model="deviceForm.manufacturer" placeholder="请输入厂商名称" />
        </el-form-item>
        <el-form-item label="型号" prop="modelName">
          <el-input v-model="deviceForm.modelName" placeholder="请输入设备型号" />
        </el-form-item>
        <el-form-item label="状态" prop="status">
          <el-select v-model="deviceForm.status" placeholder="请选择设备状态">
            <el-option label="在线" value="online" />
            <el-option label="离线" value="offline" />
            <el-option label="未知" value="unknown" />
          </el-select>
        </el-form-item>
        <el-form-item label="IP地址" prop="ipAddress">
          <el-input v-model="deviceForm.ipAddress" placeholder="请输入IP地址" />
        </el-form-item>
        <el-form-item label="描述" prop="description">
          <el-input v-model="deviceForm.description" type="textarea" placeholder="请输入设备描述" />
        </el-form-item>
      </el-form>
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="closeDialog">取消</el-button>
          <el-button type="primary" @click="submitForm">确定</el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useRouter } from 'vue-router'
import { getDeviceList, createDevice, updateDevice, deleteDevice as apiDeleteDevice, batchDeleteDevices as apiBatchDeleteDevices } from '@/api/tr069Management'

const router = useRouter()

// 表格数据
const tableData = ref([])
const page = ref(1)
const pageSize = ref(10)
const total = ref(0)
const searchInfo = reactive({
  serialNumber: '',
  status: ''
})

// 多选相关
const multipleSelection = ref([])
const multipleTable = ref(null)

// 对话框相关
const dialogFormVisible = ref(false)
const dialogTitle = ref('新增设备')
const deviceForm = reactive({
  ID: 0,
  serialNumber: '',
  manufacturer: '',
  modelName: '',
  status: 'unknown',
  ipAddress: '',
  description: ''
})
const deviceFormRef = ref(null)
const rules = reactive({
  serialNumber: [{ required: true, message: '请输入设备序列号', trigger: 'blur' }],
  manufacturer: [{ required: true, message: '请输入厂商名称', trigger: 'blur' }],
  modelName: [{ required: true, message: '请输入设备型号', trigger: 'blur' }]
})

// 获取表格数据
const getTableData = async () => {
  const params = {
    page: page.value,
    pageSize: pageSize.value,
    ...searchInfo
  }
  try {
    const res = await getDeviceList(params)
    if (res.code === 0) {
      tableData.value = res.data.list
      total.value = res.data.total
    }
  } catch (err) {
    console.error(err)
  }
}

// 重置搜索
const resetSearch = () => {
  searchInfo.serialNumber = ''
  searchInfo.status = ''
  getTableData()
}

// 处理多选变化
const handleSelectionChange = (val) => {
  multipleSelection.value = val
}

// 批量删除设备
const batchDeleteDevices = async () => {
  if (multipleSelection.value.length === 0) {
    ElMessage({
      type: 'warning',
      message: '请选择要删除的设备'
    })
    return
  }
  
  ElMessageBox.confirm('确定要删除选中的设备吗?', '提示', {
    confirmButtonText: '确定',
    cancelButtonText: '取消',
    type: 'warning'
  }).then(async () => {
    const ids = multipleSelection.value.map(item => item.ID)
    try {
      const res = await apiBatchDeleteDevices({ ids })
      if (res.code === 0) {
        ElMessage({
          type: 'success',
          message: '删除成功'
        })
        getTableData()
      }
    } catch (err) {
      console.error(err)
    }
  }).catch(() => {})
}

// 删除单个设备
const deleteDevice = async (row) => {
  try {
    const res = await apiDeleteDevice(row.ID)
    if (res.code === 0) {
      ElMessage({
        type: 'success',
        message: '删除成功'
      })
      getTableData()
    }
  } catch (err) {
    console.error(err)
  }
}

// 打开对话框
const openDialog = (type, row) => {
  if (type === 'add') {
    dialogTitle.value = '新增设备'
    Object.keys(deviceForm).forEach(key => {
      if (key !== 'status') {
        deviceForm[key] = ''
      } else {
        deviceForm[key] = 'unknown'
      }
    })
    deviceForm.ID = 0
  } else {
    dialogTitle.value = '编辑设备'
    Object.keys(deviceForm).forEach(key => {
      deviceForm[key] = row[key]
    })
  }
  dialogFormVisible.value = true
}

// 关闭对话框
const closeDialog = () => {
  dialogFormVisible.value = false
  if (deviceFormRef.value) {
    deviceFormRef.value.resetFields()
  }
}

// 提交表单
const submitForm = async () => {
  if (!deviceFormRef.value) return
  
  await deviceFormRef.value.validate(async (valid) => {
    if (valid) {
      try {
        let res
        if (deviceForm.ID === 0) {
          res = await createDevice(deviceForm)
        } else {
          res = await updateDevice(deviceForm)
        }
        
        if (res.code === 0) {
          ElMessage({
            type: 'success',
            message: deviceForm.ID === 0 ? '添加成功' : '更新成功'
          })
          closeDialog()
          getTableData()
        }
      } catch (err) {
        console.error(err)
      }
    }
  })
}

// 查看设备详情
const viewDeviceDetail = (row) => {
  router.push({
    path: `/tr069Management/deviceDetail/${row.ID}`
  })
}

// 查看设备参数
const viewDeviceParameters = (row) => {
  router.push({
    path: `/tr069Management/deviceParameters/${row.ID}`
  })
}

// 格式化日期
const formatDate = (date) => {
  if (!date) return '-'
  return new Date(date).toLocaleString()
}

// 分页处理
const handleSizeChange = (val) => {
  pageSize.value = val
  getTableData()
}

const handleCurrentChange = (val) => {
  page.value = val
  getTableData()
}

// 页面加载时获取数据
onMounted(() => {
  getTableData()
})
</script>

<style lang="scss" scoped>
.device-list-container {
  padding: 10px;
  
  .gva-search-box {
    padding: 24px;
    background-color: #fff;
    border-radius: 4px;
    margin-bottom: 12px;
  }
  
  .gva-table-box {
    padding: 24px;
    background-color: #fff;
    border-radius: 4px;
    
    .gva-btn-list {
      margin-bottom: 12px;
    }
    
    .gva-pagination {
      display: flex;
      justify-content: flex-end;
      margin-top: 15px;
    }
  }
}
</style>