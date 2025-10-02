<template>
  <div class="firmware-container">
    <div class="gva-search-box">
      <el-form :inline="true" :model="searchInfo">
        <el-form-item label="固件名称">
          <el-input v-model="searchInfo.name" placeholder="请输入固件名称" />
        </el-form-item>
        <el-form-item label="版本号">
          <el-input v-model="searchInfo.version" placeholder="请输入版本号" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="getTableData">查询</el-button>
          <el-button @click="resetSearch">重置</el-button>
        </el-form-item>
      </el-form>
    </div>
    
    <div class="gva-table-box">
      <div class="gva-btn-list">
        <el-button type="primary" @click="openUploadDialog">上传固件</el-button>
      </div>
      
      <el-table
        :data="tableData"
        style="width: 100%"
        tooltip-effect="dark"
        row-key="ID"
      >
        <el-table-column label="固件名称" prop="name" width="180" />
        <el-table-column label="版本号" prop="version" width="120" />
        <el-table-column label="型号" prop="modelName" width="150" />
        <el-table-column label="文件大小" width="120">
          <template #default="scope">
            {{ formatFileSize(scope.row.fileSize) }}
          </template>
        </el-table-column>
        <el-table-column label="描述" prop="description" width="200" show-overflow-tooltip />
        <el-table-column label="上传时间" width="180">
          <template #default="scope">
            {{ formatDate(scope.row.createdAt) }}
          </template>
        </el-table-column>
        <el-table-column label="操作">
          <template #default="scope">
            <el-button type="primary" link @click="openUpgradeDialog(scope.row)">升级</el-button>
            <el-popconfirm title="确定要删除此固件吗?" @confirm="deleteFirmware(scope.row)">
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
    
    <!-- 上传固件对话框 -->
    <el-dialog v-model="uploadDialogVisible" title="上传固件" width="600px">
      <el-form ref="uploadFormRef" :model="uploadForm" label-width="100px" :rules="uploadRules">
        <el-form-item label="固件名称" prop="name">
          <el-input v-model="uploadForm.name" placeholder="请输入固件名称" />
        </el-form-item>
        <el-form-item label="版本号" prop="version">
          <el-input v-model="uploadForm.version" placeholder="请输入版本号" />
        </el-form-item>
        <el-form-item label="适用型号" prop="modelName">
          <el-input v-model="uploadForm.modelName" placeholder="请输入适用设备型号" />
        </el-form-item>
        <el-form-item label="描述" prop="description">
          <el-input v-model="uploadForm.description" type="textarea" placeholder="请输入描述" />
        </el-form-item>
        <el-form-item label="固件文件" prop="file">
          <el-upload
            class="firmware-upload"
            :action="uploadUrl"
            :headers="uploadHeaders"
            :on-success="handleUploadSuccess"
            :on-error="handleUploadError"
            :before-upload="beforeUpload"
            :limit="1"
            :auto-upload="false"
            ref="uploadRef"
          >
            <template #trigger>
              <el-button type="primary">选择文件</el-button>
            </template>
            <template #tip>
              <div class="el-upload__tip">
                请上传固件文件，文件大小不超过100MB
              </div>
            </template>
          </el-upload>
        </el-form-item>
      </el-form>
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="uploadDialogVisible = false">取消</el-button>
          <el-button type="primary" @click="submitUpload">上传</el-button>
        </div>
      </template>
    </el-dialog>
    
    <!-- 升级固件对话框 -->
    <el-dialog v-model="upgradeDialogVisible" title="升级固件" width="500px">
      <el-form ref="upgradeFormRef" :model="upgradeForm" label-width="120px" :rules="upgradeRules">
        <el-form-item label="升级方式" prop="upgradeType">
          <el-radio-group v-model="upgradeForm.upgradeType">
            <el-radio label="device">单个设备</el-radio>
            <el-radio label="group">设备组</el-radio>
          </el-radio-group>
        </el-form-item>
        
        <el-form-item v-if="upgradeForm.upgradeType === 'device'" label="设备序列号" prop="deviceSerialNumber">
          <el-input v-model="upgradeForm.deviceSerialNumber" placeholder="请输入设备序列号" />
        </el-form-item>
        
        <el-form-item v-if="upgradeForm.upgradeType === 'group'" label="设备组" prop="groupID">
          <el-select v-model="upgradeForm.groupID" placeholder="请选择设备组">
            <el-option 
              v-for="item in groupOptions" 
              :key="item.value" 
              :label="item.label" 
              :value="item.value" 
            />
          </el-select>
        </el-form-item>
        
        <el-form-item label="升级时间" prop="scheduleTime">
          <el-radio-group v-model="upgradeForm.scheduleType">
            <el-radio label="now">立即升级</el-radio>
            <el-radio label="schedule">定时升级</el-radio>
          </el-radio-group>
          
          <el-date-picker
            v-if="upgradeForm.scheduleType === 'schedule'"
            v-model="upgradeForm.scheduleTime"
            type="datetime"
            placeholder="选择升级时间"
            style="width: 100%; margin-top: 10px;"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="upgradeDialogVisible = false">取消</el-button>
          <el-button type="primary" @click="upgradeFirmware">确定</el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted, computed } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { formatDate } from '@/utils/format'
import { useUserStore } from '@/pinia/modules/user'
import { 
  getFirmwareList, 
  deleteFirmware as deleteFirmwareAPI,
  uploadFirmware as uploadFirmwareAPI,
  upgradeFirmware as upgradeFirmwareAPI,
  getGroupList
} from '@/api/tr069Management'

const userStore = useUserStore()

// 数据列表
const tableData = ref([])
const page = ref(1)
const pageSize = ref(10)
const total = ref(0)
const searchInfo = reactive({
  name: '',
  version: ''
})

// 上传相关
const uploadDialogVisible = ref(false)
const uploadFormRef = ref(null)
const uploadRef = ref(null)
const uploadForm = reactive({
  name: '',
  version: '',
  modelName: '',
  description: '',
  file: null
})
const uploadRules = reactive({
  name: [{ required: true, message: '请输入固件名称', trigger: 'blur' }],
  version: [{ required: true, message: '请输入版本号', trigger: 'blur' }],
  modelName: [{ required: true, message: '请输入适用设备型号', trigger: 'blur' }]
})
const uploadUrl = '/tr069-management/firmware/upload'
const uploadHeaders = computed(() => {
  return { 'x-token': userStore.token }
})

// 升级相关
const upgradeDialogVisible = ref(false)
const upgradeFormRef = ref(null)
const upgradeForm = reactive({
  firmwareID: '',
  upgradeType: 'device',
  deviceSerialNumber: '',
  groupID: '',
  scheduleType: 'now',
  scheduleTime: null
})
const upgradeRules = reactive({
  upgradeType: [{ required: true, message: '请选择升级方式', trigger: 'change' }],
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
      name: searchInfo.name,
      version: searchInfo.version
    }
    const res = await getFirmwareList(params)
    if (res.code === 0) {
      tableData.value = res.data.list
      total.value = res.data.total
    }
  } catch (err) {
    console.error('获取固件列表失败:', err)
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
  searchInfo.version = ''
  getTableData()
}

// 格式化文件大小
const formatFileSize = (size) => {
  if (!size) return '0 B'
  
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let i = 0
  while (size >= 1024 && i < units.length - 1) {
    size /= 1024
    i++
  }
  
  return `${size.toFixed(2)} ${units[i]}`
}

// 打开上传对话框
const openUploadDialog = () => {
  uploadForm.name = ''
  uploadForm.version = ''
  uploadForm.modelName = ''
  uploadForm.description = ''
  uploadForm.file = null
  
  if (uploadRef.value) {
    uploadRef.value.clearFiles()
  }
  
  uploadDialogVisible.value = true
}

// 上传前检查
const beforeUpload = (file) => {
  const maxSize = 100 * 1024 * 1024 // 100MB
  if (file.size > maxSize) {
    ElMessage.error('文件大小不能超过100MB')
    return false
  }
  
  uploadForm.file = file
  return true
}

// 上传成功处理
const handleUploadSuccess = (res) => {
  if (res.code === 0) {
    ElMessage.success('固件上传成功')
    uploadDialogVisible.value = false
    getTableData()
  } else {
    ElMessage.error(res.msg || '上传失败')
  }
}

// 上传失败处理
const handleUploadError = () => {
  ElMessage.error('固件上传失败')
}

// 提交上传
const submitUpload = async () => {
  if (!uploadFormRef.value) return
  
  await uploadFormRef.value.validate(async (valid) => {
    if (valid) {
      if (!uploadForm.file) {
        ElMessage.warning('请选择固件文件')
        return
      }
      
      // 创建FormData
      const formData = new FormData()
      formData.append('name', uploadForm.name)
      formData.append('version', uploadForm.version)
      formData.append('modelName', uploadForm.modelName)
      formData.append('description', uploadForm.description)
      formData.append('file', uploadForm.file)
      
      try {
        const res = await uploadFirmwareAPI(formData)
        if (res.code === 0) {
          ElMessage.success('固件上传成功')
          uploadDialogVisible.value = false
          getTableData()
        }
      } catch (err) {
        console.error('上传固件失败:', err)
      }
    }
  })
}

// 删除固件
const deleteFirmware = async (row) => {
  try {
    const res = await deleteFirmwareAPI({ id: row.ID })
    if (res.code === 0) {
      ElMessage.success('删除成功')
      if (tableData.value.length === 1 && page.value > 1) {
        page.value--
      }
      getTableData()
    }
  } catch (err) {
    console.error('删除固件失败:', err)
  }
}

// 打开升级对话框
const openUpgradeDialog = (row) => {
  upgradeForm.firmwareID = row.ID
  upgradeForm.upgradeType = 'device'
  upgradeForm.deviceSerialNumber = ''
  upgradeForm.groupID = ''
  upgradeForm.scheduleType = 'now'
  upgradeForm.scheduleTime = null
  upgradeDialogVisible.value = true
}

// 升级固件
const upgradeFirmware = async () => {
  if (!upgradeFormRef.value) return
  
  await upgradeFormRef.value.validate(async (valid) => {
    if (valid) {
      try {
        const params = {
          firmwareID: upgradeForm.firmwareID
        }
        
        if (upgradeForm.upgradeType === 'device') {
          params.deviceSerialNumber = upgradeForm.deviceSerialNumber
        } else {
          params.groupID = upgradeForm.groupID
        }
        
        if (upgradeForm.scheduleType === 'schedule' && upgradeForm.scheduleTime) {
          params.scheduleTime = upgradeForm.scheduleTime
        }
        
        const res = await upgradeFirmwareAPI(params)
        if (res.code === 0) {
          ElMessage.success('固件升级任务已创建')
          upgradeDialogVisible.value = false
        }
      } catch (err) {
        console.error('创建固件升级任务失败:', err)
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
.firmware-container {
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
  
  .firmware-upload {
    width: 100%;
  }
}
</style>