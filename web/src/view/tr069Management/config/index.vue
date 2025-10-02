<template>
  <div class="config-profile-container">
    <div class="gva-search-box">
      <el-form :inline="true" :model="searchInfo">
        <el-form-item label="配置名称">
          <el-input v-model="searchInfo.name" placeholder="请输入配置名称" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="getTableData">查询</el-button>
          <el-button @click="resetSearch">重置</el-button>
        </el-form-item>
      </el-form>
    </div>
    
    <div class="gva-table-box">
      <div class="gva-btn-list">
        <el-button type="primary" @click="openDialog('add')">新增配置</el-button>
      </div>
      
      <el-table
        :data="tableData"
        style="width: 100%"
        tooltip-effect="dark"
        row-key="ID"
      >
        <el-table-column label="配置名称" prop="name" width="180" />
        <el-table-column label="描述" prop="description" width="300" show-overflow-tooltip />
        <el-table-column label="参数数量" prop="parameterCount" width="120" />
        <el-table-column label="创建时间" width="180">
          <template #default="scope">
            {{ formatDate(scope.row.createdAt) }}
          </template>
        </el-table-column>
        <el-table-column label="操作">
          <template #default="scope">
            <el-button type="primary" link @click="openDialog('edit', scope.row)">编辑</el-button>
            <el-button type="primary" link @click="openApplyDialog(scope.row)">应用</el-button>
            <el-popconfirm title="确定要删除此配置吗?" @confirm="deleteConfigProfile(scope.row)">
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
    
    <!-- 配置表单对话框 -->
    <el-dialog v-model="dialogFormVisible" :title="dialogTitle" width="700px">
      <el-form ref="configFormRef" :model="configForm" label-width="100px" :rules="rules">
        <el-form-item label="配置名称" prop="name">
          <el-input v-model="configForm.name" placeholder="请输入配置名称" />
        </el-form-item>
        <el-form-item label="描述" prop="description">
          <el-input v-model="configForm.description" type="textarea" placeholder="请输入描述" />
        </el-form-item>
        <el-form-item label="参数配置" prop="parameters">
          <el-table :data="configForm.parameters" style="width: 100%">
            <el-table-column label="参数名称" width="300">
              <template #default="scope">
                <el-input v-model="scope.row.name" placeholder="参数名称" />
              </template>
            </el-table-column>
            <el-table-column label="参数值" width="200">
              <template #default="scope">
                <el-input v-model="scope.row.value" placeholder="参数值" />
              </template>
            </el-table-column>
            <el-table-column>
              <template #default="scope">
                <el-button type="danger" link @click="removeParameter(scope.$index)">删除</el-button>
              </template>
            </el-table-column>
          </el-table>
          <div style="margin-top: 10px;">
            <el-button type="primary" @click="addParameter">添加参数</el-button>
          </div>
        </el-form-item>
      </el-form>
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="dialogFormVisible = false">取消</el-button>
          <el-button type="primary" @click="submitForm">确定</el-button>
        </div>
      </template>
    </el-dialog>
    
    <!-- 应用配置对话框 -->
    <el-dialog v-model="applyDialogVisible" title="应用配置" width="500px">
      <el-form ref="applyFormRef" :model="applyForm" label-width="120px" :rules="applyRules">
        <el-form-item label="应用方式" prop="applyType">
          <el-radio-group v-model="applyForm.applyType">
            <el-radio label="device">单个设备</el-radio>
            <el-radio label="group">设备组</el-radio>
          </el-radio-group>
        </el-form-item>
        
        <el-form-item v-if="applyForm.applyType === 'device'" label="设备序列号" prop="deviceSerialNumber">
          <el-input v-model="applyForm.deviceSerialNumber" placeholder="请输入设备序列号" />
        </el-form-item>
        
        <el-form-item v-if="applyForm.applyType === 'group'" label="设备组" prop="groupID">
          <el-select v-model="applyForm.groupID" placeholder="请选择设备组">
            <el-option 
              v-for="item in groupOptions" 
              :key="item.value" 
              :label="item.label" 
              :value="item.value" 
            />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="applyDialogVisible = false">取消</el-button>
          <el-button type="primary" @click="applyConfig">确定</el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { formatDate } from '@/utils/format'
import { 
  getConfigProfileList, 
  getConfigProfileByID, 
  createConfigProfile, 
  updateConfigProfile, 
  deleteConfigProfile as deleteConfigAPI,
  applyConfigProfile as applyConfigAPI,
  getGroupList
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
const dialogTitle = ref('新增配置')
const configFormRef = ref(null)
const configForm = reactive({
  id: '',
  name: '',
  description: '',
  parameters: []
})
const rules = reactive({
  name: [{ required: true, message: '请输入配置名称', trigger: 'blur' }]
})

// 应用配置相关
const applyDialogVisible = ref(false)
const applyFormRef = ref(null)
const applyForm = reactive({
  configID: '',
  applyType: 'device',
  deviceSerialNumber: '',
  groupID: ''
})
const applyRules = reactive({
  applyType: [{ required: true, message: '请选择应用方式', trigger: 'change' }],
  deviceSerialNumber: [{ required: true, message: '请输入设备序列号', trigger: 'blur' }],
  groupID: [{ required: true, message: '请选择设备组', trigger: 'change' }]
})
const groupOptions = ref([])

// 初始化
onMounted(() => {
  getTableData()
  loadGroups()
})

// 获取表格数据
const getTableData = async () => {
  try {
    const params = {
      page: page.value,
      pageSize: pageSize.value,
      name: searchInfo.name
    }
    const res = await getConfigProfileList(params)
    if (res.code === 0) {
      tableData.value = res.data.list
      total.value = res.data.total
    }
  } catch (err) {
    console.error('获取配置列表失败:', err)
  }
}

// 加载设备组
const loadGroups = async () => {
  try {
    const res = await getGroupList({ page: 1, pageSize: 100 })
    if (res.code === 0) {
      groupOptions.value = res.data.list.map(item => ({
        label: item.name,
        value: item.ID
      }))
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
const openDialog = async (type, row) => {
  if (type === 'add') {
    dialogTitle.value = '新增配置'
    configForm.id = ''
    configForm.name = ''
    configForm.description = ''
    configForm.parameters = [{ name: '', value: '' }]
  } else {
    dialogTitle.value = '编辑配置'
    configForm.id = row.ID
    
    // 获取配置详情
    try {
      const res = await getConfigProfileByID({ id: row.ID })
      if (res.code === 0) {
        configForm.name = res.data.name
        configForm.description = res.data.description
        configForm.parameters = res.data.parameters || []
        
        // 确保至少有一个参数
        if (configForm.parameters.length === 0) {
          configForm.parameters = [{ name: '', value: '' }]
        }
      }
    } catch (err) {
      console.error('获取配置详情失败:', err)
    }
  }
  dialogFormVisible.value = true
}

// 添加参数
const addParameter = () => {
  configForm.parameters.push({ name: '', value: '' })
}

// 移除参数
const removeParameter = (index) => {
  configForm.parameters.splice(index, 1)
}

// 提交表单
const submitForm = async () => {
  if (!configFormRef.value) return
  
  await configFormRef.value.validate(async (valid) => {
    if (valid) {
      // 验证参数是否都填写完整
      const invalidParams = configForm.parameters.some(p => !p.name || !p.value)
      if (invalidParams) {
        ElMessage.warning('请填写完整的参数名称和值')
        return
      }
      
      try {
        let res
        if (configForm.id) {
          // 编辑
          res = await updateConfigProfile({
            id: configForm.id,
            name: configForm.name,
            description: configForm.description,
            parameters: configForm.parameters
          })
        } else {
          // 新增
          res = await createConfigProfile({
            name: configForm.name,
            description: configForm.description,
            parameters: configForm.parameters
          })
        }
        
        if (res.code === 0) {
          ElMessage.success(configForm.id ? '更新成功' : '创建成功')
          dialogFormVisible.value = false
          getTableData()
        }
      } catch (err) {
        console.error(configForm.id ? '更新配置失败:' : '创建配置失败:', err)
      }
    }
  })
}

// 删除配置
const deleteConfigProfile = async (row) => {
  try {
    const res = await deleteConfigAPI({ id: row.ID })
    if (res.code === 0) {
      ElMessage.success('删除成功')
      if (tableData.value.length === 1 && page.value > 1) {
        page.value--
      }
      getTableData()
    }
  } catch (err) {
    console.error('删除配置失败:', err)
  }
}

// 打开应用配置对话框
const openApplyDialog = (row) => {
  applyForm.configID = row.ID
  applyForm.applyType = 'device'
  applyForm.deviceSerialNumber = ''
  applyForm.groupID = ''
  applyDialogVisible.value = true
}

// 应用配置
const applyConfig = async () => {
  if (!applyFormRef.value) return
  
  await applyFormRef.value.validate(async (valid) => {
    if (valid) {
      try {
        const params = {
          configID: applyForm.configID
        }
        
        if (applyForm.applyType === 'device') {
          params.deviceSerialNumber = applyForm.deviceSerialNumber
        } else {
          params.groupID = applyForm.groupID
        }
        
        const res = await applyConfigAPI(params)
        if (res.code === 0) {
          ElMessage.success('配置应用成功')
          applyDialogVisible.value = false
        }
      } catch (err) {
        console.error('应用配置失败:', err)
      }
    }
  })
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
.config-profile-container {
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
</style>