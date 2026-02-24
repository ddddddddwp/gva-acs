<template>
  <div class="fap-info-container">
    <div class="header">
      <span class="title">基站无线参数 (TR-196)</span>
      <div class="actions">
        <el-button type="primary" size="small" :icon="Refresh" @click="handleSync" :loading="syncLoading">
          刷新/同步
        </el-button>
        <el-button type="warning" size="small" :icon="Edit" @click="handleEdit">
          修改配置
        </el-button>
      </div>
    </div>

    <el-descriptions border :column="2" size="medium">
      <el-descriptions-item label="物理小区ID (PCI)">
        {{ fapInfo.pci || '-' }}
      </el-descriptions-item>
      <el-descriptions-item label="小区标识 (CellID)">
        {{ fapInfo.cellId || '-' }}
      </el-descriptions-item>
      <el-descriptions-item label="下行频点 (EARFCN-DL)">
        {{ fapInfo.earfcnDl || '-' }}
      </el-descriptions-item>
      <el-descriptions-item label="上行频点 (EARFCN-UL)">
        {{ fapInfo.earfcnUl || '-' }}
      </el-descriptions-item>
      <el-descriptions-item label="下行带宽">
        {{ fapInfo.dlBandwidth || '-' }}
      </el-descriptions-item>
      <el-descriptions-item label="上行带宽">
        {{ fapInfo.ulBandwidth || '-' }}
      </el-descriptions-item>
      <el-descriptions-item label="发射功率">
        {{ fapInfo.txPower || '-' }}
      </el-descriptions-item>
      <el-descriptions-item label="运行状态">
        <el-tag :type="fapInfo.opState ? 'success' : 'danger'">
          {{ fapInfo.opState ? 'Enable' : 'Disable' }}
        </el-tag>
      </el-descriptions-item>
    </el-descriptions>

    <!-- 配置修改弹窗 -->
    <el-dialog title="修改基站配置" v-model="dialogVisible" width="500px" append-to-body>
      <el-form :model="form" label-width="120px">
        <el-form-item label="物理小区ID (PCI)">
          <el-input v-model.number="form.pci" type="number" placeholder="0-503"></el-input>
        </el-form-item>
        <el-form-item label="发射功率 (dBm)">
          <el-input v-model="form.txPower" placeholder="e.g. 20"></el-input>
        </el-form-item>
      </el-form>
      <template #footer>
        <span class="dialog-footer">
          <el-button @click="dialogVisible = false">取 消</el-button>
          <el-button type="primary" @click="handleSubmit" :loading="submitLoading">确 定</el-button>
        </span>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { getFAPInfo, syncFAPInfo, configureFAP } from '@/plugin/tr069/api/fap'

// Props
const props = defineProps({
  deviceId: {
    type: Number,
    required: true
  }
})

// 响应式数据
const fapInfo = ref({})
const syncLoading = ref(false)
const submitLoading = ref(false)
const dialogVisible = ref(false)
const form = reactive({
  pci: null,
  txPower: ''
})

// 获取数据
const fetchData = async () => {
  try {
    const res = await getFAPInfo(props.deviceId)
    if (res.code === 0) {
      fapInfo.value = res.data || {}
    }
  } catch (error) {
    ElMessage.error('获取基站信息失败')
  }
}

// 同步
const handleSync = async () => {
  syncLoading.value = true
  try {
    const res = await syncFAPInfo(props.deviceId)
    if (res.code === 0) {
      ElMessage.success('同步任务已下发，请稍后刷新查看最新数据')
    } else {
      ElMessage.error(res.msg || '同步失败')
    }
  } finally {
    syncLoading.value = false
  }
}

// 编辑
const handleEdit = () => {
  form.pci = fapInfo.value.pci
  form.txPower = fapInfo.value.txPower
  dialogVisible.value = true
}

// 提交
const handleSubmit = async () => {
  submitLoading.value = true
  try {
    const res = await configureFAP(props.deviceId, { ...form })
    if (res.code === 0) {
      ElMessage.success('配置任务已下发')
      dialogVisible.value = false
      fetchData()
    } else {
      ElMessage.error(res.msg || '配置失败')
    }
  } finally {
    submitLoading.value = false
  }
}

// 监听设备ID变化
watch(() => props.deviceId, (val) => {
  if (val) {
    fetchData()
  }
}, { immediate: true })
</script>

<style scoped>
.fap-info-container {
  padding: 20px;
  background-color: #fff;
}
.header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}
.title {
  font-size: 16px;
  font-weight: bold;
  color: #303133;
}
</style>
