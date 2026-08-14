<template>
  <el-dialog
    v-model="visible"
    :title="action?.label || '设备操作'"
    width="820px"
    append-to-body
    destroy-on-close
  >
    <el-alert
      v-if="action"
      :title="`${action.label}（${action.method}）`"
      type="info"
      :closable="false"
      class="rpc-command-tip"
    />

    <el-form v-if="action" :model="form" label-width="150px">
      <template v-if="action.key === 'getRPCMethods'">
        <el-empty description="无需参数，将查询设备声明支持的 CWMP 方法" :image-size="80" />
      </template>

      <template v-else-if="action.key === 'getParameterValues'">
        <el-form-item label="参数路径">
          <el-input v-model="form.pathsText" type="textarea" :rows="9" placeholder="每行一个路径，例如 Device.DeviceInfo.SerialNumber" />
        </el-form-item>
      </template>

      <template v-else-if="action.key === 'getParameterNames'">
        <el-form-item label="对象路径">
          <el-input v-model="form.parameterPath" placeholder="例如 Device." />
        </el-form-item>
        <el-form-item label="只查询下一层">
          <el-switch v-model="form.nextLevel" />
        </el-form-item>
      </template>

      <template v-else-if="action.key === 'getParameterAttributes'">
        <el-form-item label="参数名称">
          <el-input v-model="form.parameterNamesText" type="textarea" :rows="9" placeholder="每行一个完整参数名称" />
        </el-form-item>
      </template>

      <template v-else-if="action.key === 'setParameterValues'">
        <el-form-item label="ParameterKey">
          <el-input v-model="form.parameterKey" />
        </el-form-item>
        <el-form-item label="参数列表">
          <div class="rpc-command-table">
            <el-table :data="form.parameters" size="small" border>
              <el-table-column label="参数名称" min-width="280">
                <template #default="scope"><el-input v-model="scope.row.name" /></template>
              </el-table-column>
              <el-table-column label="类型" width="180">
                <template #default="scope">
                  <el-select v-model="scope.row.type" style="width: 100%">
                    <el-option v-for="type in xsdTypes" :key="type" :label="type" :value="type" />
                  </el-select>
                </template>
              </el-table-column>
              <el-table-column label="值" min-width="180">
                <template #default="scope"><el-input v-model="scope.row.value" /></template>
              </el-table-column>
              <el-table-column width="70" align="center">
                <template #default="scope"><el-button link type="danger" @click="removeRow('parameters', scope.$index)">删除</el-button></template>
              </el-table-column>
            </el-table>
            <el-button class="rpc-add-row" plain type="primary" @click="addParameter">新增参数</el-button>
          </div>
        </el-form-item>
      </template>

      <template v-else-if="action.key === 'setParameterAttributes'">
        <el-form-item label="属性列表">
          <div class="rpc-command-table">
            <div v-for="(item, index) in form.parameterAttributes" :key="index" class="attribute-card">
              <el-input v-model="item.name" placeholder="完整参数名称" />
              <div class="attribute-options">
                <el-checkbox v-model="item.notificationChange">修改通知级别</el-checkbox>
                <el-select v-model="item.notification" :disabled="!item.notificationChange" style="width: 120px">
                  <el-option label="0 - 关闭" :value="0" />
                  <el-option label="1 - 被动" :value="1" />
                  <el-option label="2 - 主动" :value="2" />
                </el-select>
                <el-checkbox v-model="item.accessListChange">修改访问列表</el-checkbox>
                <el-input v-model="item.accessListText" :disabled="!item.accessListChange" placeholder="逗号分隔，例如 Subscriber" />
                <el-button link type="danger" @click="removeRow('parameterAttributes', index)">删除</el-button>
              </div>
            </div>
            <el-button class="rpc-add-row" plain type="primary" @click="addAttribute">新增属性</el-button>
          </div>
        </el-form-item>
      </template>

      <template v-else-if="action.key === 'addObject' || action.key === 'deleteObject'">
        <el-form-item label="对象路径">
          <el-input v-model="form.objectName" :placeholder="action.key === 'addObject' ? '例如 Device.WiFi.SSID.' : '例如 Device.WiFi.SSID.7.'" />
        </el-form-item>
        <el-form-item label="ParameterKey">
          <el-input v-model="form.parameterKey" />
        </el-form-item>
        <el-alert
          v-if="action.key === 'deleteObject'"
          title="这里只删除设备数据模型中的对象实例，不会删除 GVA 设备记录。"
          type="warning"
          :closable="false"
        />
      </template>

      <template v-else-if="action.key === 'download'">
        <el-form-item label="文件类型">
          <el-input v-model="form.fileType" placeholder="CWMP FileType，例如 1 Firmware Upgrade Image" />
        </el-form-item>
        <el-form-item label="文件 URL">
          <el-input v-model="form.url" placeholder="http(s):// 或 ftp://" />
        </el-form-item>
        <el-form-item label="用户名"><el-input v-model="form.username" /></el-form-item>
        <el-form-item label="密码"><el-input v-model="form.password" type="password" show-password /></el-form-item>
        <el-form-item label="延迟秒数"><el-input-number v-model="form.delaySeconds" :min="0" /></el-form-item>
        <el-form-item label="文件大小"><el-input-number v-model="form.fileSize" :min="0" /></el-form-item>
        <el-form-item label="目标文件名"><el-input v-model="form.targetFileName" /></el-form-item>
        <el-form-item label="成功回调 URL"><el-input v-model="form.successURL" /></el-form-item>
        <el-form-item label="失败回调 URL"><el-input v-model="form.failureURL" /></el-form-item>
      </template>

      <template v-else-if="action.key === 'upload'">
        <el-form-item label="文件类型">
          <el-input v-model="form.fileType" placeholder="CWMP FileType，例如 Vendor Log File" />
        </el-form-item>
        <el-form-item label="延迟秒数"><el-input-number v-model="form.delaySeconds" :min="0" /></el-form-item>
      </template>

      <template v-else-if="action.key === 'factoryReset'">
        <el-alert title="恢复出厂设置会清除设备配置，提交后需要等待设备重新连接。" type="error" :closable="false" show-icon />
      </template>
    </el-form>

    <template #footer>
      <el-button @click="visible = false">取消</el-button>
      <el-button type="primary" :loading="submitting" :disabled="!supported" @click="submit">提交命令</el-button>
    </template>
  </el-dialog>
</template>

<script setup>
import { computed, reactive, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  addObject,
  deleteObject,
  downloadFile,
  factoryResetDevice,
  getParameterAttributes,
  getParameterNames,
  getParameterValues,
  getRPCMethods,
  rebootDevice,
  setParameterAttributes,
  setParameterValues,
  uploadFile
} from '@/plugin/tr069/api/command'
import { canIssueRPCAction } from '@/plugin/tr069/utils/device-actions'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
  row: { type: Object, default: () => ({}) },
  action: { type: Object, default: undefined }
})

const emit = defineEmits(['update:modelValue', 'submitted'])

const visible = computed({
  get: () => props.modelValue,
  set: value => emit('update:modelValue', value)
})
const supported = computed(() => canIssueRPCAction(props.row, props.action))
const submitting = ref(false)
const xsdTypes = ['xsd:string', 'xsd:int', 'xsd:unsignedInt', 'xsd:boolean', 'xsd:dateTime', 'xsd:base64Binary', 'xsd:long', 'xsd:unsignedLong', 'xsd:double']

const defaultForm = () => ({
  pathsText: 'Device.',
  parameterPath: 'Device.',
  nextLevel: true,
  parameterNamesText: 'Device.DeviceInfo.Manufacturer',
  parameterKey: '',
  parameters: [{ name: '', type: 'xsd:string', value: '' }],
  parameterAttributes: [{ name: '', notificationChange: true, notification: 0, accessListChange: false, accessListText: '' }],
  objectName: '',
  fileType: '',
  url: '',
  username: '',
  password: '',
  fileSize: 0,
  targetFileName: '',
  delaySeconds: 0,
  successURL: '',
  failureURL: ''
})
const form = reactive(defaultForm())

watch(() => [props.modelValue, props.action?.key], ([open]) => {
  if (open) Object.assign(form, defaultForm())
})

const lines = (value) => (value || '').split('\n').map(item => item.trim()).filter(Boolean)

const addParameter = () => form.parameters.push({ name: '', type: 'xsd:string', value: '' })
const addAttribute = () => form.parameterAttributes.push({ name: '', notificationChange: true, notification: 0, accessListChange: false, accessListText: '' })
const removeRow = (field, index) => {
  form[field].splice(index, 1)
}

const requireValue = (value, message) => {
  if (!String(value || '').trim()) throw new Error(message)
  return String(value).trim()
}

const buildPayload = () => {
  switch (props.action.key) {
    case 'getRPCMethods':
    case 'factoryReset':
      return undefined
    case 'getParameterValues': {
      const paths = lines(form.pathsText)
      if (!paths.length) throw new Error('请至少填写一个参数路径')
      return { paths }
    }
    case 'getParameterNames':
      return { parameterPath: requireValue(form.parameterPath, '请填写对象路径'), nextLevel: form.nextLevel }
    case 'getParameterAttributes': {
      const parameterNames = lines(form.parameterNamesText)
      if (!parameterNames.length) throw new Error('请至少填写一个参数名称')
      return { parameterNames }
    }
    case 'setParameterValues': {
      const parameters = form.parameters.map(item => ({ ...item, name: item.name.trim() })).filter(item => item.name)
      if (!parameters.length) throw new Error('请至少填写一条参数')
      return { parameterKey: form.parameterKey, parameters }
    }
    case 'setParameterAttributes': {
      const parameterAttributes = form.parameterAttributes.map(item => ({
        name: item.name.trim(),
        notificationChange: item.notificationChange,
        notification: item.notification,
        accessListChange: item.accessListChange,
        accessList: item.accessListChange ? item.accessListText.split(',').map(value => value.trim()).filter(Boolean) : []
      })).filter(item => item.name)
      if (!parameterAttributes.length) throw new Error('请至少填写一条参数属性')
      return { parameterAttributes }
    }
    case 'addObject':
    case 'deleteObject':
      return { objectName: requireValue(form.objectName, '请填写对象路径'), parameterKey: form.parameterKey }
    case 'download':
      return {
        fileType: requireValue(form.fileType, '请填写文件类型'),
        url: requireValue(form.url, '请填写文件 URL'),
        username: form.username,
        password: form.password,
        fileSize: form.fileSize,
        targetFileName: form.targetFileName,
        delaySeconds: form.delaySeconds,
        successURL: form.successURL,
        failureURL: form.failureURL
      }
    case 'upload':
      return {
        fileType: requireValue(form.fileType, '请填写文件类型'),
        delaySeconds: form.delaySeconds
      }
    case 'reboot':
      return undefined
    default:
      throw new Error('不支持的设备操作')
  }
}

const apiByAction = {
  getRPCMethods,
  getParameterValues,
  getParameterNames,
  getParameterAttributes,
  setParameterValues,
  setParameterAttributes,
  addObject,
  deleteObject,
  download: downloadFile,
  upload: uploadFile,
  reboot: rebootDevice,
  factoryReset: factoryResetDevice
}

const confirmSubmit = async () => {
  if (props.action.confirm === 'none') return
  const danger = props.action.confirm === 'danger'
  await ElMessageBox.confirm(
    danger ? `“${props.action.label}”可能影响设备业务，确定继续吗？` : `确定向设备下发“${props.action.label}”吗？`,
    danger ? '危险操作确认' : '操作确认',
    { type: danger ? 'error' : 'warning', confirmButtonText: '确认下发', cancelButtonText: '取消' }
  )
}

const submit = async () => {
  if (!props.row?.ID || !props.action || !supported.value) return
  let payload
  try {
    payload = buildPayload()
    await confirmSubmit()
  } catch (error) {
    if (error instanceof Error) ElMessage.warning(error.message)
    return
  }

  submitting.value = true
  try {
    const request = apiByAction[props.action.key]
    const res = await request(props.row.ID, payload)
    if (res.code !== 0) {
      ElMessage.error(res.msg || '命令下发失败')
      return
    }
    ElMessage.success('命令已提交')
    emit('submitted', { action: props.action, command: res.data })
    visible.value = false
  } finally {
    submitting.value = false
  }
}
</script>

<style scoped>
.rpc-command-tip {
  margin-bottom: 20px;
}
.rpc-command-table {
  width: 100%;
}
.rpc-add-row {
  margin-top: 12px;
}
.attribute-card {
  padding: 12px;
  margin-bottom: 12px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 4px;
}
.attribute-options {
  display: grid;
  grid-template-columns: auto 120px auto 1fr auto;
  gap: 12px;
  align-items: center;
  margin-top: 12px;
}
</style>
