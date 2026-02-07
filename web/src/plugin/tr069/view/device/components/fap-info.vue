<template>
  <div class="fap-info-container">
    <div class="header">
      <span class="title">基站无线参数 (TR-196)</span>
      <div class="actions">
        <el-button type="primary" size="small" icon="el-icon-refresh" @click="handleSync" :loading="syncLoading">
          刷新/同步
        </el-button>
        <el-button type="warning" size="small" icon="el-icon-edit" @click="handleEdit">
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
      <span slot="footer" class="dialog-footer">
        <el-button @click="dialogVisible = false">取 消</el-button>
        <el-button type="primary" @click="handleSubmit" :loading="submitLoading">确 定</el-button>
      </span>
    </el-dialog>
  </div>
</template>

<script>
import { getFAPInfo, syncFAPInfo, configureFAP } from '@/plugin/tr069/api/fap'
import { ElMessage } from 'element-plus'

export default {
  name: 'FapInfo',
  props: {
    deviceId: {
      type: Number,
      required: true
    }
  },
  data() {
    return {
      fapInfo: {},
      syncLoading: false,
      submitLoading: false,
      dialogVisible: false,
      form: {
        pci: null,
        txPower: ''
      }
    }
  },
  watch: {
    deviceId: {
      handler(val) {
        if (val) {
          this.fetchData()
        }
      },
      immediate: true
    }
  },
  methods: {
    async fetchData() {
      try {
        const res = await getFAPInfo(this.deviceId)
        if (res.code === 0) {
          this.fapInfo = res.data || {}
        }
      } catch (error) {
        console.error('Fetch FAP Info Error:', error)
      }
    },
    async handleSync() {
      this.syncLoading = true
      try {
        const res = await syncFAPInfo(this.deviceId)
        if (res.code === 0) {
          ElMessage.success('同步任务已下发，请稍后刷新查看最新数据')
        }
      } finally {
        this.syncLoading = false
      }
    },
    handleEdit() {
      this.form = {
        pci: this.fapInfo.pci,
        txPower: this.fapInfo.txPower
      }
      this.dialogVisible = true
    },
    async handleSubmit() {
      this.submitLoading = true
      try {
        const res = await configureFAP(this.deviceId, this.form)
        if (res.code === 0) {
          ElMessage.success('配置任务已下发')
          this.dialogVisible = false
        }
      } finally {
        this.submitLoading = false
      }
    }
  }
}
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
