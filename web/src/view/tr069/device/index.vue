<template>
  <div class="gva-table-box">
    <div class="gva-search-box">
      <el-form :inline="true" :model="searchInfo">
        <el-form-item label="序列号">
          <el-input v-model="searchInfo.serialNumber" placeholder="序列号" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" icon="search" @click="onSearch">查询</el-button>
          <el-button icon="refresh" @click="onReset">重置</el-button>
        </el-form-item>
      </el-form>
    </div>

    <el-table :data="tableData" row-key="ID">
      <el-table-column align="left" label="ID" min-width="80" prop="ID" />
      <el-table-column align="left" label="SN" min-width="160" prop="serialNumber" />
      <el-table-column align="left" label="OUI" min-width="120" prop="oui" />
      <el-table-column align="left" label="状态" min-width="100" prop="status" />
      <el-table-column align="left" label="IP" min-width="140" prop="ip" />
      <el-table-column label="操作" min-width="480">
        <template #default="scope">
          <el-button type="primary" link @click="handleGetRPCMethods(scope.row)">GetRPCMethods</el-button>
          <el-button type="primary" link @click="openGPV(scope.row)">GetParameterValues</el-button>
          <el-button type="primary" link @click="openSPV(scope.row)">SetParameterValues</el-button>
          <el-button type="primary" link @click="openFullSync(scope.row)">全量同步</el-button>
          <el-button type="primary" link @click="openDataModel(scope.row)">参数树</el-button>
        </template>
      </el-table-column>
    </el-table>

    <div class="gva-pagination">
      <el-pagination
        v-model:current-page="page"
        v-model:page-size="pageSize"
        :total="total"
        layout="total, sizes, prev, pager, next, jumper"
        :page-sizes="[10, 20, 50, 100]"
        @current-change="getTableData"
        @size-change="getTableData"
      />
    </div>

    <el-dialog v-model="gpvDialogVisible" title="GetParameterValues" width="720px">
      <el-form :model="gpvForm" label-width="120px">
        <el-form-item label="参数路径(一行一个)">
          <el-input v-model="gpvForm.pathsText" type="textarea" :rows="10" placeholder="Device.DeviceInfo.SerialNumber&#10;Device.Services.FAPService.1." />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="gpvDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="gpvSubmitting" @click="submitGPV">下发</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="spvDialogVisible" title="SetParameterValues" width="820px">
      <el-form :model="spvForm" label-width="120px">
        <el-form-item label="ParameterKey">
          <el-input v-model="spvForm.parameterKey" placeholder="可选" />
        </el-form-item>
        <el-form-item label="参数列表">
          <el-table :data="spvForm.parameters" size="small">
            <el-table-column label="Name" min-width="360">
              <template #default="scope">
                <el-input v-model="scope.row.name" placeholder="Device.X.Y" />
              </template>
            </el-table-column>
            <el-table-column label="Type" min-width="140">
              <template #default="scope">
                <el-input v-model="scope.row.type" placeholder="xsd:string" />
              </template>
            </el-table-column>
            <el-table-column label="Value" min-width="220">
              <template #default="scope">
                <el-input v-model="scope.row.value" placeholder="值" />
              </template>
            </el-table-column>
            <el-table-column label="" width="80">
              <template #default="scope">
                <el-button type="primary" link icon="delete" @click="removeSPVRow(scope.$index)">删除</el-button>
              </template>
            </el-table-column>
          </el-table>
          <div class="mt-3">
            <el-button type="primary" icon="plus" @click="addSPVRow">新增一行</el-button>
          </div>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="spvDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="spvSubmitting" @click="submitSPV">下发</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="fullSyncVisible" title="全量同步数据模型" width="520px">
      <el-form :model="fullSyncForm" label-width="120px">
        <el-form-item label="最大深度">
          <el-input-number v-model="fullSyncForm.maxDepth" :min="1" :max="64" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="fullSyncVisible = false">取消</el-button>
        <el-button type="primary" :loading="fullSyncSubmitting" @click="submitFullSync">下发</el-button>
      </template>
    </el-dialog>

    <el-drawer v-model="dmDrawerVisible" title="设备数据模型参数树" size="80%">
      <div class="flex h-full">
        <!-- 左侧树 -->
        <div class="w-1/3 h-full border-r pr-4 overflow-auto">
          <el-tree
            ref="dmTreeRef"
            :data="dmTreeData"
            :props="{ label: 'label', children: 'children', isLeaf: 'isLeaf' }"
            node-key="id"
            highlight-current
            accordion
            @node-click="handleNodeClick"
          >
            <template #default="{ node, data }">
              <span class="custom-tree-node">
                <span>{{ node.label }}</span>
              </span>
            </template>
          </el-tree>
        </div>
        <!-- 右侧值列表 -->
        <div class="w-2/3 h-full pl-4 flex flex-col">
          <div class="mb-2 font-bold text-lg" v-if="currentPath">{{ currentPath }}</div>
          <el-table :data="dmTableData" stripe style="width: 100%" height="100%">
            <el-table-column prop="name" label="参数名" min-width="300" show-overflow-tooltip />
            <el-table-column prop="valueJson" label="值" min-width="200" show-overflow-tooltip>
               <template #default="scope">
                 {{ formatValue(scope.row.valueJson) }}
               </template>
            </el-table-column>
            <el-table-column prop="valueType" label="类型" width="120" />
            <el-table-column prop="lastCollectedAt" label="采集时间" width="180">
              <template #default="scope">
                {{ scope.row.lastCollectedAt ? new Date(scope.row.lastCollectedAt).toLocaleString() : '-' }}
              </template>
            </el-table-column>
          </el-table>
        </div>
      </div>
    </el-drawer>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { ElMessage } from 'element-plus'
import { 
  getTR069DeviceList, 
  tr069FullDataModelSync, 
  tr069GetParameterValues, 
  tr069GetRPCMethods, 
  tr069SetParameterValues,
  tr069GetDataModelStructure,
  tr069GetDataModelList
} from '@/api/tr069'

defineOptions({
  name: 'TR069Device'
})

const searchInfo = ref({
  serialNumber: ''
})

const page = ref(1)
const pageSize = ref(10)
const total = ref(0)
const tableData = ref([])

const getTableData = async () => {
  const res = await getTR069DeviceList({
    page: page.value,
    pageSize: pageSize.value,
    serialNumber: searchInfo.value.serialNumber
  })
  if (res.code === 0) {
    tableData.value = res.data.list
    total.value = res.data.total
  }
}

const onSearch = async () => {
  page.value = 1
  await getTableData()
}

const onReset = async () => {
  searchInfo.value.serialNumber = ''
  page.value = 1
  await getTableData()
}

const currentDeviceId = ref(null)

const handleGetRPCMethods = async (row) => {
  const res = await tr069GetRPCMethods(row.ID)
  if (res.code === 0) {
    ElMessage.success(`已下发，commandId=${res.data?.commandId || '-'}`)
  } else {
    ElMessage.error(res.msg || '下发失败')
  }
}

const gpvDialogVisible = ref(false)
const gpvSubmitting = ref(false)
const gpvForm = ref({
  pathsText: ''
})

const openGPV = (row) => {
  currentDeviceId.value = row.ID
  gpvForm.value.pathsText = 'Device.DeviceInfo.SerialNumber'
  gpvDialogVisible.value = true
}

const submitGPV = async () => {
  const deviceId = currentDeviceId.value
  if (!deviceId) return
  const paths = gpvForm.value.pathsText
    .split('\n')
    .map((s) => s.trim())
    .filter(Boolean)
  if (paths.length === 0) {
    ElMessage.warning('请填写参数路径')
    return
  }
  gpvSubmitting.value = true
  const res = await tr069GetParameterValues(deviceId, { paths })
  gpvSubmitting.value = false
  if (res.code === 0) {
    ElMessage.success(`已下发，commandId=${res.data?.commandId || '-'}`)
    gpvDialogVisible.value = false
  } else {
    ElMessage.error(res.msg || '下发失败')
  }
}

const spvDialogVisible = ref(false)
const spvSubmitting = ref(false)
const spvForm = ref({
  parameterKey: '',
  parameters: [{ name: '', type: 'xsd:string', value: '' }]
})

const openSPV = (row) => {
  currentDeviceId.value = row.ID
  spvForm.value = {
    parameterKey: '',
    parameters: [{ name: '', type: 'xsd:string', value: '' }]
  }
  spvDialogVisible.value = true
}

const addSPVRow = () => {
  spvForm.value.parameters = [...spvForm.value.parameters, { name: '', type: 'xsd:string', value: '' }]
}

const removeSPVRow = (idx) => {
  spvForm.value.parameters = spvForm.value.parameters.filter((_, i) => i !== idx)
}

const submitSPV = async () => {
  const deviceId = currentDeviceId.value
  if (!deviceId) return
  const payload = {
    parameterKey: spvForm.value.parameterKey,
    parameters: spvForm.value.parameters
      .map((p) => ({ ...p, name: (p.name || '').trim(), type: (p.type || '').trim() }))
      .filter((p) => p.name)
  }
  if (payload.parameters.length === 0) {
    ElMessage.warning('请至少填写一条参数')
    return
  }
  spvSubmitting.value = true
  const res = await tr069SetParameterValues(deviceId, payload)
  spvSubmitting.value = false
  if (res.code === 0) {
    ElMessage.success(`已下发，commandId=${res.data?.commandId || '-'}`)
    spvDialogVisible.value = false
  } else {
    ElMessage.error(res.msg || '下发失败')
  }
}

const fullSyncVisible = ref(false)
const fullSyncSubmitting = ref(false)
const fullSyncForm = ref({
  maxDepth: 16
})

const openFullSync = (row) => {
  currentDeviceId.value = row.ID
  fullSyncVisible.value = true
}

const submitFullSync = async () => {
  const deviceId = currentDeviceId.value
  if (!deviceId) return
  fullSyncSubmitting.value = true
  const res = await tr069FullDataModelSync(deviceId, { maxDepth: fullSyncForm.value.maxDepth })
  fullSyncSubmitting.value = false
  if (res.code === 0) {
    ElMessage.success('已下发，等待设备上报')
    fullSyncVisible.value = false
  } else {
    ElMessage.error(res.msg || '下发失败')
  }
}

// Data Model Tree Logic
const dmDrawerVisible = ref(false)
const dmTreeData = ref([])
const dmTableData = ref([])
const currentPath = ref('')

const openDataModel = async (row) => {
  currentDeviceId.value = row.ID
  dmDrawerVisible.value = true
  dmTreeData.value = []
  dmTableData.value = []
  currentPath.value = ''
  
  // Load structure
  const res = await tr069GetDataModelStructure(row.ID)
  if (res.code === 0 && res.data) {
    dmTreeData.value = buildTree(res.data)
  }
}

// Convert flat paths to tree structure
const buildTree = (paths) => {
  const root = []
  // Ensure paths are sorted to process parent before children
  paths.sort()
  
  // Helper to find or create node
  const findOrCreate = (nodes, part, fullPath) => {
    let node = nodes.find(n => n.label === part)
    if (!node) {
      node = {
        id: fullPath,
        label: part,
        children: [],
        isLeaf: false
      }
      nodes.push(node)
    }
    return node
  }

  paths.forEach(path => {
    // Split "Device.DeviceInfo." -> ["Device", "DeviceInfo"]
    const parts = path.split('.').filter(Boolean)
    let currentLevel = root
    let currentPath = ''
    
    parts.forEach((part, index) => {
      currentPath += part + '.'
      const node = findOrCreate(currentLevel, part, currentPath)
      currentLevel = node.children
    })
  })
  
  return root
}

const handleNodeClick = async (data) => {
  currentPath.value = data.id
  const res = await tr069GetDataModelList(currentDeviceId.value, { prefix: data.id })
  if (res.code === 0 && res.data) {
    // API returns { list: [...] }
    dmTableData.value = res.data.list || []
  }
}

const formatValue = (jsonVal) => {
  if (jsonVal === null || jsonVal === undefined) return ''
  // If it's a JSON string, try to parse it, otherwise return as is
  // But usually it's already a JS object/value if axios parsed JSON response
  if (typeof jsonVal === 'object') return JSON.stringify(jsonVal)
  return String(jsonVal)
}

getTableData()
</script>

