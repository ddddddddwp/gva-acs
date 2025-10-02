<template>
  <div class="parameter-list-container">
    <div class="gva-search-box">
      <el-form :inline="true" :model="searchInfo">
        <el-form-item label="设备序列号">
          <el-input v-model="searchInfo.deviceSerialNumber" placeholder="请输入设备序列号" />
        </el-form-item>
        <el-form-item label="参数名称">
          <el-input v-model="searchInfo.name" placeholder="请输入参数名称" />
        </el-form-item>
        <el-form-item label="参数类别">
          <el-select v-model="searchInfo.category" placeholder="请选择参数类别" clearable>
            <el-option 
              v-for="item in categoryOptions" 
              :key="item.value" 
              :label="item.label" 
              :value="item.value" 
            />
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
        <el-button type="primary" @click="openSetParametersDialog">设置参数</el-button>
        <el-button type="primary" @click="refreshParameters">刷新参数</el-button>
      </div>
      
      <el-table
        :data="tableData"
        style="width: 100%"
        tooltip-effect="dark"
        row-key="ID"
      >
        <el-table-column label="设备序列号" prop="deviceSerialNumber" width="180" />
        <el-table-column label="参数名称" prop="name" width="300" show-overflow-tooltip />
        <el-table-column label="参数值" prop="value" width="200" show-overflow-tooltip />
        <el-table-column label="类型" prop="type" width="100" />
        <el-table-column label="类别" prop="category" width="120" />
        <el-table-column label="可写" width="80">
          <template #default="scope">
            <el-tag :type="scope.row.writable ? 'success' : 'info'">
              {{ scope.row.writable ? '是' : '否' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="更新时间" width="180">
          <template #default="scope">
            {{ formatDate(scope.row.updatedAt) }}
          </template>
        </el-table-column>
        <el-table-column label="操作">
          <template #default="scope">
            <el-button 
              type="primary" 
              link 
              :disabled="!scope.row.writable" 
              @click="openEditDialog(scope.row)"
            >
              编辑
            </el-button>
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
    
    <!-- 编辑参数对话框 -->
    <el-dialog v-model="editDialogVisible" title="编辑参数" width="500px">
      <el-form ref="paramFormRef" :model="paramForm" label-width="120px" :rules="rules">
        <el-form-item label="参数名称">
          <el-input v-model="paramForm.name" disabled />
        </el-form-item>
        <el-form-item label="当前值">
          <el-input v-model="paramForm.currentValue" disabled />
        </el-form-item>
        <el-form-item label="新值" prop="newValue">
          <el-input v-model="paramForm.newValue" placeholder="请输入新的参数值" />
        </el-form-item>
      </el-form>
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="editDialogVisible = false">取消</el-button>
          <el-button type="primary" @click="updateParameter">确定</el-button>
        </div>
      </template>
    </el-dialog>
    
    <!-- 批量设置参数对话框 -->
    <el-dialog v-model="setParamsDialogVisible" title="设置设备参数" width="600px">
      <el-form ref="setParamsFormRef" :model="setParamsForm" label-width="120px" :rules="setParamsRules">
        <el-form-item label="设备序列号" prop="deviceSerialNumber">
          <el-input v-model="setParamsForm.deviceSerialNumber" placeholder="请输入设备序列号" />
        </el-form-item>
        <el-form-item label="参数列表" prop="parameters">
          <el-table :data="setParamsForm.parameters" style="width: 100%">
            <el-table-column label="参数名称" width="300">
              <template #default="scope">
                <el-input v-model="scope.row.name" placeholder="参数名称" />
              </template>
            </el-table-column>
            <el-table-column label="参数值" width="150">
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
          <el-button @click="setParamsDialogVisible = false">取消</el-button>
          <el-button type="primary" @click="setParameters">确定</el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { formatDate } from '@/utils/format'
import { getParametersByDevice, getParameterCategories, setDeviceParameters } from '@/api/tr069Management'

// 数据列表
const tableData = ref([])
const page = ref(1)
const pageSize = ref(10)
const total = ref(0)
const searchInfo = reactive({
  deviceSerialNumber: '',
  name: '',
  category: ''
})
const categoryOptions = ref([])

// 编辑参数相关
const editDialogVisible = ref(false)
const paramFormRef = ref(null)
const paramForm = reactive({
  id: '',
  name: '',
  currentValue: '',
  newValue: '',
  deviceSerialNumber: ''
})
const rules = reactive({
  newValue: [{ required: true, message: '请输入参数值', trigger: 'blur' }]
})

// 批量设置参数相关
const setParamsDialogVisible = ref(false)
const setParamsFormRef = ref(null)
const setParamsForm = reactive({
  deviceSerialNumber: '',
  parameters: []
})
const setParamsRules = reactive({
  deviceSerialNumber: [{ required: true, message: '请输入设备序列号', trigger: 'blur' }]
})

// 初始化
onMounted(() => {
  getTableData()
  getCategories()
})

// 获取参数类别
const getCategories = async () => {
  try {
    const res = await getParameterCategories()
    if (res.code === 0) {
      categoryOptions.value = res.data.map(item => ({
        label: item,
        value: item
      }))
    }
  } catch (err) {
    console.error('获取参数类别失败:', err)
  }
}

// 获取表格数据
const getTableData = async () => {
  try {
    const params = {
      page: page.value,
      pageSize: pageSize.value,
      deviceSerialNumber: searchInfo.deviceSerialNumber,
      name: searchInfo.name,
      category: searchInfo.category
    }
    const res = await getParametersByDevice(params)
    if (res.code === 0) {
      tableData.value = res.data.list
      total.value = res.data.total
    }
  } catch (err) {
    console.error('获取参数列表失败:', err)
  }
}

// 重置搜索
const resetSearch = () => {
  searchInfo.deviceSerialNumber = ''
  searchInfo.name = ''
  searchInfo.category = ''
  getTableData()
}

// 刷新参数
const refreshParameters = () => {
  if (!searchInfo.deviceSerialNumber) {
    ElMessage.warning('请先输入设备序列号')
    return
  }
  
  ElMessageBox.confirm('确定要刷新设备参数吗?', '提示', {
    confirmButtonText: '确定',
    cancelButtonText: '取消',
    type: 'warning'
  }).then(() => {
    // 这里应该调用刷新参数的API，目前先重新获取数据
    getTableData()
    ElMessage.success('参数刷新成功')
  }).catch(() => {})
}

// 打开编辑对话框
const openEditDialog = (row) => {
  paramForm.id = row.ID
  paramForm.name = row.name
  paramForm.currentValue = row.value
  paramForm.newValue = row.value
  paramForm.deviceSerialNumber = row.deviceSerialNumber
  editDialogVisible.value = true
}

// 更新参数
const updateParameter = async () => {
  if (!paramFormRef.value) return
  
  await paramFormRef.value.validate(async (valid) => {
    if (valid) {
      try {
        const params = {
          deviceSerialNumber: paramForm.deviceSerialNumber,
          parameters: [
            {
              name: paramForm.name,
              value: paramForm.newValue
            }
          ]
        }
        
        const res = await setDeviceParameters(params)
        if (res.code === 0) {
          ElMessage.success('参数更新成功')
          editDialogVisible.value = false
          getTableData()
        }
      } catch (err) {
        console.error('更新参数失败:', err)
      }
    }
  })
}

// 打开设置参数对话框
const openSetParametersDialog = () => {
  setParamsForm.deviceSerialNumber = searchInfo.deviceSerialNumber || ''
  setParamsForm.parameters = [{ name: '', value: '' }]
  setParamsDialogVisible.value = true
}

// 添加参数
const addParameter = () => {
  setParamsForm.parameters.push({ name: '', value: '' })
}

// 移除参数
const removeParameter = (index) => {
  setParamsForm.parameters.splice(index, 1)
}

// 设置参数
const setParameters = async () => {
  if (!setParamsFormRef.value) return
  
  await setParamsFormRef.value.validate(async (valid) => {
    if (valid) {
      // 验证参数是否都填写完整
      const invalidParams = setParamsForm.parameters.some(p => !p.name || !p.value)
      if (invalidParams) {
        ElMessage.warning('请填写完整的参数名称和值')
        return
      }
      
      try {
        const params = {
          deviceSerialNumber: setParamsForm.deviceSerialNumber,
          parameters: setParamsForm.parameters
        }
        
        const res = await setDeviceParameters(params)
        if (res.code === 0) {
          ElMessage.success('参数设置成功')
          setParamsDialogVisible.value = false
          // 如果当前搜索的设备与设置的设备相同，则刷新数据
          if (searchInfo.deviceSerialNumber === setParamsForm.deviceSerialNumber) {
            getTableData()
          }
        }
      } catch (err) {
        console.error('设置参数失败:', err)
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
.parameter-list-container {
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