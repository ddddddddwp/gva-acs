<template>
  <div class="device-list-container">
    <div class="search-box">
      <el-form :inline="true" :model="searchInfo" class="demo-form-inline">
        <el-form-item label="序列号">
          <el-input v-model="searchInfo.serialNumber" placeholder="请输入序列号" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" icon="el-icon-search" @click="onSubmit">查询</el-button>
          <el-button icon="el-icon-refresh" @click="onReset">重置</el-button>
        </el-form-item>
      </el-form>
    </div>

    <div class="table-box">
      <div class="btn-list">
        <el-button type="primary" icon="el-icon-plus" @click="openDialog">录入设备 (白名单)</el-button>
      </div>

      <el-table :data="tableData" style="width: 100%" v-loading="loading">
        <el-table-column prop="ID" label="ID" width="60" />
        <el-table-column prop="serialNumber" label="序列号" min-width="150" />
        <el-table-column prop="oui" label="OUI" width="100" />
        <el-table-column prop="productClass" label="产品类别" width="120" />
        <el-table-column prop="softwareVer" label="软件版本" width="120" />
        <el-table-column prop="ip" label="IP地址" width="130" />
        <el-table-column label="在线状态" width="100">
          <template #default="scope">
            <el-tag :type="scope.row.status === 'online' ? 'success' : 'info'">
              {{ scope.row.status === 'online' ? '在线' : '离线' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="200" fixed="right">
          <template #default="scope">
            <el-button type="text" size="small" @click="viewDetail(scope.row)">详情/配置</el-button>
            <el-button type="text" size="small" class="delete-btn" @click="deleteRow(scope.row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>

      <div class="pagination">
        <el-pagination
          background
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

    <!-- 录入设备弹窗 -->
    <el-dialog title="录入设备 (白名单)" v-model="dialogFormVisible" width="500px">
      <el-form :model="formData" ref="addForm" :rules="rules" label-width="100px">
        <el-form-item label="序列号" prop="serialNumber">
          <el-input v-model="formData.serialNumber" autocomplete="off" />
        </el-form-item>
        <el-form-item label="OUI" prop="oui">
          <el-input v-model="formData.oui" autocomplete="off" />
        </el-form-item>
        <el-form-item label="备注">
          <el-input v-model="formData.remark" type="textarea" />
        </el-form-item>
      </el-form>
      <div slot="footer" class="dialog-footer">
        <el-button @click="dialogFormVisible = false">取 消</el-button>
        <el-button type="primary" @click="enterDevice">确 定</el-button>
      </div>
    </el-dialog>

    <!-- 设备详情抽屉 -->
    <el-drawer
      title="设备详情与配置"
      v-model="drawerVisible"
      direction="rtl"
      size="50%">
      <div class="drawer-content" v-if="currentRow.ID">
        <!-- 基础信息 -->
        <el-descriptions title="基础信息" :column="2" border class="mb-20">
          <el-descriptions-item label="序列号">{{ currentRow.serialNumber }}</el-descriptions-item>
          <el-descriptions-item label="OUI">{{ currentRow.oui }}</el-descriptions-item>
          <el-descriptions-item label="IP地址">{{ currentRow.ip }}</el-descriptions-item>
          <el-descriptions-item label="最后上线">{{ currentRow.lastOnline | formatDate }}</el-descriptions-item>
        </el-descriptions>

        <div class="mb-20">
          <el-button type="primary" size="small" @click="handleGetRPCMethods" :loading="rpcLoading">GetRPCMethods</el-button>
          <el-button type="primary" size="small" @click="openGPVDialog">GetParameterValues</el-button>
          <el-button type="primary" size="small" @click="openSPVDialog">SetParameterValues</el-button>
          <el-button type="primary" size="small" @click="openFullSyncDialog">全量同步</el-button>
        </div>

        <!-- 基站无线参数组件 -->
        <fap-info :device-id="currentRow.ID" />
      </div>
    </el-drawer>

    <el-dialog title="GetParameterValues" v-model="gpvDialogVisible" width="720px" append-to-body>
      <el-form :model="gpvForm" label-width="140px">
        <el-form-item label="参数路径(一行一个)">
          <el-input v-model="gpvForm.pathsText" type="textarea" :rows="10" />
        </el-form-item>
      </el-form>
      <div slot="footer" class="dialog-footer">
        <el-button @click="gpvDialogVisible = false">取 消</el-button>
        <el-button type="primary" @click="submitGPV" :loading="gpvSubmitting">确 定</el-button>
      </div>
    </el-dialog>

    <el-dialog title="SetParameterValues" v-model="spvDialogVisible" width="820px" append-to-body>
      <el-form :model="spvForm" label-width="140px">
        <el-form-item label="ParameterKey">
          <el-input v-model="spvForm.parameterKey" />
        </el-form-item>
        <el-form-item label="参数列表">
          <el-table :data="spvForm.parameters" size="small">
            <el-table-column label="Name" min-width="360">
              <template #default="scope">
                <el-input v-model="scope.row.name" />
              </template>
            </el-table-column>
            <el-table-column label="Type" min-width="160">
              <template #default="scope">
                <el-input v-model="scope.row.type" />
              </template>
            </el-table-column>
            <el-table-column label="Value" min-width="220">
              <template #default="scope">
                <el-input v-model="scope.row.value" />
              </template>
            </el-table-column>
            <el-table-column label="" width="80">
              <template #default="scope">
                <el-button type="text" size="small" @click="removeSPVRow(scope.$index)">删除</el-button>
              </template>
            </el-table-column>
          </el-table>
          <div class="mt-3">
            <el-button type="primary" size="small" @click="addSPVRow">新增一行</el-button>
          </div>
        </el-form-item>
      </el-form>
      <div slot="footer" class="dialog-footer">
        <el-button @click="spvDialogVisible = false">取 消</el-button>
        <el-button type="primary" @click="submitSPV" :loading="spvSubmitting">确 定</el-button>
      </div>
    </el-dialog>

    <el-dialog title="全量同步数据模型" v-model="fullSyncVisible" width="520px" append-to-body>
      <el-form :model="fullSyncForm" label-width="140px">
        <el-form-item label="最大深度">
          <el-input-number v-model="fullSyncForm.maxDepth" :min="1" :max="64" />
        </el-form-item>
      </el-form>
      <div slot="footer" class="dialog-footer">
        <el-button @click="fullSyncVisible = false">取 消</el-button>
        <el-button type="primary" @click="submitFullSync" :loading="fullSyncSubmitting">确 定</el-button>
      </div>
    </el-dialog>
  </div>
</template>

<script>
import { getDeviceList, createDevice, deleteDevice } from '@/plugin/tr069/api/device'
import { fullDataModelSync, getParameterValues, getRPCMethods, setParameterValues } from '@/plugin/tr069/api/command'
import FapInfo from './components/fap-info.vue'
import { formatTimeToStr } from '@/utils/date'
import { ElMessage, ElMessageBox } from 'element-plus'

export default {
  name: 'DeviceList',
  components: {
    FapInfo
  },
  filters: {
    formatDate(time) {
      if (time && time !== '0001-01-01T00:00:00Z') {
        return formatTimeToStr(time)
      }
      return '-'
    }
  },
  data() {
    return {
      loading: false,
      tableData: [],
      page: 1,
      pageSize: 10,
      total: 0,
      searchInfo: {
        serialNumber: ''
      },
      dialogFormVisible: false,
      formData: {
        serialNumber: '',
        oui: '',
        remark: ''
      },
      rules: {
        serialNumber: [{ required: true, message: '请输入序列号', trigger: 'blur' }],
        oui: [{ required: true, message: '请输入OUI', trigger: 'blur' }]
      },
      drawerVisible: false,
      currentRow: {},
      rpcLoading: false,
      gpvDialogVisible: false,
      gpvSubmitting: false,
      gpvForm: {
        pathsText: ''
      },
      spvDialogVisible: false,
      spvSubmitting: false,
      spvForm: {
        parameterKey: '',
        parameters: [{ name: '', type: 'xsd:string', value: '' }]
      },
      fullSyncVisible: false,
      fullSyncSubmitting: false,
      fullSyncForm: {
        maxDepth: 16
      }
    }
  },
  created() {
    this.getTableData()
  },
  methods: {
    async getTableData() {
      this.loading = true
      const res = await getDeviceList({ page: this.page, pageSize: this.pageSize, ...this.searchInfo })
      if (res.code === 0) {
        this.tableData = res.data.list
        this.total = res.data.total
      }
      this.loading = false
    },
    onSubmit() {
      this.page = 1
      this.getTableData()
    },
    onReset() {
      this.searchInfo = { serialNumber: '' }
      this.getTableData()
    },
    handleCurrentChange(val) {
      this.page = val
      this.getTableData()
    },
    handleSizeChange(val) {
      this.pageSize = val
      this.getTableData()
    },
    openDialog() {
      this.formData = { serialNumber: '', oui: '', remark: '' }
      this.dialogFormVisible = true
    },
    enterDevice() {
      this.$refs.addForm.validate(async valid => {
        if (valid) {
          const res = await createDevice(this.formData)
          if (res.code === 0) {
            ElMessage.success('录入成功')
            this.dialogFormVisible = false
            this.getTableData()
          }
        }
      })
    },
    deleteRow(row) {
      ElMessageBox.confirm('确定要删除该设备吗?', '提示', {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      }).then(async () => {
        const res = await deleteDevice(row.ID)
        if (res.code === 0) {
          ElMessage.success('删除成功')
          this.getTableData()
        }
      })
    },
    viewDetail(row) {
      this.currentRow = row
      this.drawerVisible = true
      this.gpvForm = { pathsText: 'Device.DeviceInfo.SerialNumber' }
      this.spvForm = { parameterKey: '', parameters: [{ name: '', type: 'xsd:string', value: '' }] }
    },
    async handleGetRPCMethods() {
      if (!this.currentRow.ID) return
      this.rpcLoading = true
      try {
        const res = await getRPCMethods(this.currentRow.ID)
        if (res.code === 0) {
          ElMessage.success(`已下发，commandId=${res.data?.commandId || '-'}`)
        } else {
          ElMessage.error(res.msg || '下发失败')
        }
      } finally {
        this.rpcLoading = false
      }
    },
    openGPVDialog() {
      this.gpvDialogVisible = true
    },
    async submitGPV() {
      if (!this.currentRow.ID) return
      const paths = (this.gpvForm.pathsText || '')
        .split('\n')
        .map(s => s.trim())
        .filter(Boolean)
      if (paths.length === 0) {
        ElMessage.warning('请填写参数路径')
        return
      }
      this.gpvSubmitting = true
      try {
        const res = await getParameterValues(this.currentRow.ID, { paths })
        if (res.code === 0) {
          ElMessage.success(`已下发，commandId=${res.data?.commandId || '-'}`)
          this.gpvDialogVisible = false
        } else {
          ElMessage.error(res.msg || '下发失败')
        }
      } finally {
        this.gpvSubmitting = false
      }
    },
    openSPVDialog() {
      this.spvDialogVisible = true
    },
    addSPVRow() {
      this.spvForm.parameters = [...this.spvForm.parameters, { name: '', type: 'xsd:string', value: '' }]
    },
    removeSPVRow(idx) {
      this.spvForm.parameters = this.spvForm.parameters.filter((_, i) => i !== idx)
    },
    async submitSPV() {
      if (!this.currentRow.ID) return
      const payload = {
        parameterKey: this.spvForm.parameterKey,
        parameters: (this.spvForm.parameters || [])
          .map(p => ({ ...p, name: (p.name || '').trim(), type: (p.type || '').trim() }))
          .filter(p => p.name)
      }
      if (payload.parameters.length === 0) {
        ElMessage.warning('请至少填写一条参数')
        return
      }
      this.spvSubmitting = true
      try {
        const res = await setParameterValues(this.currentRow.ID, payload)
        if (res.code === 0) {
          ElMessage.success(`已下发，commandId=${res.data?.commandId || '-'}`)
          this.spvDialogVisible = false
        } else {
          ElMessage.error(res.msg || '下发失败')
        }
      } finally {
        this.spvSubmitting = false
      }
    },
    openFullSyncDialog() {
      this.fullSyncVisible = true
    },
    async submitFullSync() {
      if (!this.currentRow.ID) return
      this.fullSyncSubmitting = true
      try {
        const res = await fullDataModelSync(this.currentRow.ID, { maxDepth: this.fullSyncForm.maxDepth })
        if (res.code === 0) {
          ElMessage.success('已下发，等待设备上报')
          this.fullSyncVisible = false
        } else {
          ElMessage.error(res.msg || '下发失败')
        }
      } finally {
        this.fullSyncSubmitting = false
      }
    }
  }
}
</script>

<style scoped>
.device-list-container {
  padding: 20px;
  background-color: #fff;
}
.search-box {
  margin-bottom: 20px;
}
.table-box {
  background-color: #fff;
}
.btn-list {
  margin-bottom: 10px;
}
.delete-btn {
  color: #f56c6c;
}
.drawer-content {
  padding: 20px;
}
.mb-20 {
  margin-bottom: 20px;
}
</style>
