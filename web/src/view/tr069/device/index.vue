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
      <el-table-column label="操作" min-width="420" fixed="right">
        <template #default="scope">
          <el-button type="primary" link @click="handleGetRPCMethods(scope.row)">GetRPCMethods</el-button>
          <el-button type="primary" link @click="openGPV(scope.row)">GetParameterValues</el-button>
          <el-button type="primary" link @click="openSPV(scope.row)">SetParameterValues</el-button>
          <el-button type="primary" link @click="openFullSync(scope.row)">全量同步</el-button>
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
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { ElMessage } from 'element-plus'
import { getTR069DeviceList, tr069FullDataModelSync, tr069GetParameterValues, tr069GetRPCMethods, tr069SetParameterValues } from '@/api/tr069'

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

getTableData()
</script>

