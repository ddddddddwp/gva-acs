<template>
  <div class="group-list-container">
    <div class="gva-search-box">
      <el-form :inline="true" :model="searchInfo">
        <el-form-item label="组名称">
          <el-input v-model="searchInfo.name" placeholder="请输入组名称" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="getTableData">查询</el-button>
          <el-button @click="resetSearch">重置</el-button>
        </el-form-item>
      </el-form>
    </div>
    
    <div class="gva-table-box">
      <div class="gva-btn-list">
        <el-button type="primary" @click="openDialog('add')">新增设备组</el-button>
      </div>
      
      <el-table
        :data="tableData"
        style="width: 100%"
        tooltip-effect="dark"
        row-key="ID"
      >
        <el-table-column label="组名称" prop="name" width="180" />
        <el-table-column label="描述" prop="description" width="300" show-overflow-tooltip />
        <el-table-column label="设备数量" prop="deviceCount" width="120" />
        <el-table-column label="创建时间" width="180">
          <template #default="scope">
            {{ formatDate(scope.row.createdAt) }}
          </template>
        </el-table-column>
        <el-table-column label="操作">
          <template #default="scope">
            <el-button type="primary" link @click="openDialog('edit', scope.row)">编辑</el-button>
            <el-button type="primary" link @click="openDeviceManagement(scope.row)">管理设备</el-button>
            <el-popconfirm title="确定要删除此设备组吗?" @confirm="deleteGroup(scope.row)">
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
    
    <!-- 设备组表单对话框 -->
    <el-dialog v-model="dialogFormVisible" :title="dialogTitle" width="500px">
      <el-form ref="groupFormRef" :model="groupForm" label-width="100px" :rules="rules">
        <el-form-item label="组名称" prop="name">
          <el-input v-model="groupForm.name" placeholder="请输入组名称" />
        </el-form-item>
        <el-form-item label="描述" prop="description">
          <el-input v-model="groupForm.description" type="textarea" placeholder="请输入描述" />
        </el-form-item>
      </el-form>
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="dialogFormVisible = false">取消</el-button>
          <el-button type="primary" @click="submitForm">确定</el-button>
        </div>
      </template>
    </el-dialog>
    
    <!-- 设备管理对话框 -->
    <el-dialog v-model="deviceManagementVisible" title="设备组管理" width="800px">
      <div class="device-management-container">
        <div class="device-management-header">
          <h3>{{ currentGroup.name }} - 设备管理</h3>
          <p>{{ currentGroup.description }}</p>
        </div>
        
        <div class="device-search">
          <el-input
            v-model="deviceSearchInput"
            placeholder="请输入设备序列号或型号搜索"
            clearable
            @input="filterDevices"
          >
            <template #append>
              <el-button @click="filterDevices">搜索</el-button>
            </template>
          </el-input>
        </div>
        
        <div class="device-lists">
          <div class="device-list">
            <h4>可用设备</h4>
            <el-table
              :data="availableDevices"
              style="width: 100%"
              height="300"
              @selection-change="handleAvailableSelectionChange"
            >
              <el-table-column type="selection" width="55" />
              <el-table-column label="序列号" prop="serialNumber" width="180" />
              <el-table-column label="型号" prop="modelName" />
            </el-table>
            <div class="device-list-footer">
              <el-button 
                type="primary" 
                :disabled="selectedAvailableDevices.length === 0"
                @click="addDevicesToGroup"
              >
                添加到组 <el-icon><ArrowRight /></el-icon>
              </el-button>
            </div>
          </div>
          
          <div class="device-list">
            <h4>组内设备</h4>
            <el-table
              :data="groupDevices"
              style="width: 100%"
              height="300"
              @selection-change="handleGroupSelectionChange"
            >
              <el-table-column type="selection" width="55" />
              <el-table-column label="序列号" prop="serialNumber" width="180" />
              <el-table-column label="型号" prop="modelName" />
            </el-table>
            <div class="device-list-footer">
              <el-button 
                type="danger" 
                :disabled="selectedGroupDevices.length === 0"
                @click="removeDevicesFromGroup"
              >
                <el-icon><ArrowLeft /></el-icon> 从组中移除
              </el-button>
            </div>
          </div>
        </div>
      </div>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { ArrowRight, ArrowLeft } from '@element-plus/icons-vue'
import { formatDate } from '@/utils/format'
import { 
  getGroupList, 
  getGroupByID, 
  createGroup, 
  updateGroup, 
  deleteGroup as deleteGroupAPI,
  addDeviceToGroup as addDeviceAPI,
  removeDeviceFromGroup as removeDeviceAPI
} from '@/api/tr069Management'

// 数据列表
const tableData = ref([])
const page = ref(1)
const pageSize = ref(10)
const total = ref(0)
const searchInfo = reactive({
  name: ''
})

// 表单相关
const dialogFormVisible = ref(false)
const dialogTitle = ref('新增设备组')
const groupFormRef = ref(null)
const groupForm = reactive({
  id: '',
  name: '',
  description: ''
})
const rules = reactive({
  name: [{ required: true, message: '请输入组名称', trigger: 'blur' }]
})

// 设备管理相关
const deviceManagementVisible = ref(false)
const currentGroup = reactive({
  id: '',
  name: '',
  description: ''
})
const deviceSearchInput = ref('')
const availableDevices = ref([])
const groupDevices = ref([])
const selectedAvailableDevices = ref([])
const selectedGroupDevices = ref([])

// 初始化
onMounted(() => {
  getTableData()
})

// 获取表格数据
const getTableData = async () => {
  try {
    const params = {
      page: page.value,
      pageSize: pageSize.value,
      name: searchInfo.name
    }
    const res = await getGroupList(params)
    if (res.code === 0) {
      tableData.value = res.data.list
      total.value = res.data.total
    }
  } catch (err) {
    console.error('获取设备组列表失败:', err)
  }
}

// 重置搜索
const resetSearch = () => {
  searchInfo.name = ''
  getTableData()
}

// 打开对话框
const openDialog = (type, row) => {
  if (type === 'add') {
    dialogTitle.value = '新增设备组'
    groupForm.id = ''
    groupForm.name = ''
    groupForm.description = ''
  } else {
    dialogTitle.value = '编辑设备组'
    groupForm.id = row.ID
    groupForm.name = row.name
    groupForm.description = row.description
  }
  dialogFormVisible.value = true
}

// 提交表单
const submitForm = async () => {
  if (!groupFormRef.value) return
  
  await groupFormRef.value.validate(async (valid) => {
    if (valid) {
      try {
        let res
        if (groupForm.id) {
          // 编辑
          res = await updateGroup({
            id: groupForm.id,
            name: groupForm.name,
            description: groupForm.description
          })
        } else {
          // 新增
          res = await createGroup({
            name: groupForm.name,
            description: groupForm.description
          })
        }
        
        if (res.code === 0) {
          ElMessage.success(groupForm.id ? '更新成功' : '创建成功')
          dialogFormVisible.value = false
          getTableData()
        }
      } catch (err) {
        console.error(groupForm.id ? '更新设备组失败:' : '创建设备组失败:', err)
      }
    }
  })
}

// 删除设备组
const deleteGroup = async (row) => {
  try {
    const res = await deleteGroupAPI({ id: row.ID })
    if (res.code === 0) {
      ElMessage.success('删除成功')
      if (tableData.value.length === 1 && page.value > 1) {
        page.value--
      }
      getTableData()
    }
  } catch (err) {
    console.error('删除设备组失败:', err)
  }
}

// 打开设备管理
const openDeviceManagement = async (row) => {
  currentGroup.id = row.ID
  currentGroup.name = row.name
  currentGroup.description = row.description
  
  // 获取组内设备和可用设备
  await loadGroupDevices()
  await loadAvailableDevices()
  
  deviceManagementVisible.value = true
}

// 加载组内设备
const loadGroupDevices = async () => {
  try {
    const res = await getGroupByID({ id: currentGroup.id })
    if (res.code === 0 && res.data.devices) {
      groupDevices.value = res.data.devices
    } else {
      groupDevices.value = []
    }
  } catch (err) {
    console.error('获取组内设备失败:', err)
    groupDevices.value = []
  }
}

// 加载可用设备
const loadAvailableDevices = async () => {
  // 这里应该调用获取所有可用设备的API
  // 目前模拟一些数据
  availableDevices.value = [
    { ID: 1, serialNumber: 'DEVICE001', modelName: 'Model A' },
    { ID: 2, serialNumber: 'DEVICE002', modelName: 'Model B' },
    { ID: 3, serialNumber: 'DEVICE003', modelName: 'Model C' }
  ].filter(device => !groupDevices.value.some(gd => gd.ID === device.ID))
}

// 过滤设备
const filterDevices = () => {
  // 实现设备过滤逻辑
  if (!deviceSearchInput.value) {
    loadAvailableDevices()
    return
  }
  
  const searchTerm = deviceSearchInput.value.toLowerCase()
  // 过滤可用设备
  availableDevices.value = availableDevices.value.filter(
    device => device.serialNumber.toLowerCase().includes(searchTerm) || 
              device.modelName.toLowerCase().includes(searchTerm)
  )
}

// 处理可用设备选择变化
const handleAvailableSelectionChange = (selection) => {
  selectedAvailableDevices.value = selection
}

// 处理组内设备选择变化
const handleGroupSelectionChange = (selection) => {
  selectedGroupDevices.value = selection
}

// 添加设备到组
const addDevicesToGroup = async () => {
  if (selectedAvailableDevices.value.length === 0) return
  
  try {
    // 逐个添加设备到组
    for (const device of selectedAvailableDevices.value) {
      await addDeviceAPI({
        groupID: currentGroup.id,
        deviceID: device.ID
      })
    }
    
    ElMessage.success('添加设备成功')
    // 重新加载设备列表
    await loadGroupDevices()
    await loadAvailableDevices()
    selectedAvailableDevices.value = []
  } catch (err) {
    console.error('添加设备到组失败:', err)
  }
}

// 从组中移除设备
const removeDevicesFromGroup = async () => {
  if (selectedGroupDevices.value.length === 0) return
  
  try {
    // 逐个从组中移除设备
    for (const device of selectedGroupDevices.value) {
      await removeDeviceAPI({
        groupID: currentGroup.id,
        deviceID: device.ID
      })
    }
    
    ElMessage.success('移除设备成功')
    // 重新加载设备列表
    await loadGroupDevices()
    await loadAvailableDevices()
    selectedGroupDevices.value = []
  } catch (err) {
    console.error('从组中移除设备失败:', err)
  }
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
</script>

<style lang="scss" scoped>
.group-list-container {
  background-color: #fff;
  padding: 20px;
  border-radius: 4px;
  
  .gva-search-box {
    padding-bottom: 20px;
    border-bottom: 1px solid #dcdfe6;
    margin-bottom: 20px;
  }
  
  .gva-btn-list {
    margin-bottom: 12px;
  }
  
  .gva-pagination {
    display: flex;
    justify-content: flex-end;
    margin-top: 20px;
  }
}

.device-management-container {
  .device-management-header {
    margin-bottom: 20px;
    
    h3 {
      margin: 0 0 8px 0;
    }
    
    p {
      margin: 0;
      color: #606266;
    }
  }
  
  .device-search {
    margin-bottom: 20px;
  }
  
  .device-lists {
    display: flex;
    gap: 20px;
    
    .device-list {
      flex: 1;
      
      h4 {
        margin: 0 0 10px 0;
      }
      
      .device-list-footer {
        margin-top: 10px;
        display: flex;
        justify-content: center;
      }
    }
  }
}
</style>