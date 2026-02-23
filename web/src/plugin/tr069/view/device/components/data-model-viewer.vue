<template>
  <el-drawer
    v-model="visible"
    title="设备参数树 (Data Model)"
    size="85%"
    direction="rtl"
    :before-close="handleClose"
    destroy-on-close
    class="dm-drawer"
  >
    <div class="dm-container h-full flex flex-col">
      <!-- Top Actions Bar -->
      <div class="dm-header flex justify-between items-center px-6 py-4 border-b border-gray-200 bg-white">
        <div class="flex items-center gap-3">
          <span class="text-lg font-bold text-gray-800 tracking-wide font-mono">{{ deviceRow.serialNumber || 'Unknown Device' }}</span>
          <el-tag :type="deviceRow.status === 'online' ? 'success' : 'info'" effect="dark" size="small" class="ml-2 rounded-full px-3">
            {{ deviceRow.status === 'online' ? '在线' : '离线' }}
          </el-tag>
        </div>
        <div class="flex gap-2">
          <el-button type="primary" plain size="small" :icon="Refresh" @click="refreshStructure" :loading="loadingStructure">
            刷新结构
          </el-button>
          <el-button type="success" plain size="small" :icon="Download" @click="openFullSync">
            全量同步
          </el-button>
        </div>
      </div>

      <!-- Main Content Split -->
      <div class="dm-content flex-1 flex overflow-hidden">
        <!-- Left: Tree Structure -->
        <div class="dm-sidebar w-1/3 min-w-[300px] border-r border-gray-100 flex flex-col bg-white">
          <div class="p-3 border-b border-gray-50">
            <el-input
              v-model="filterText"
              placeholder="搜索参数节点..."
              :prefix-icon="Search"
              clearable
            />
          </div>
          <div class="flex-1 overflow-auto p-2 custom-scrollbar">
            <el-tree
              ref="treeRef"
              :data="treeData"
              :props="defaultProps"
              node-key="id"
              highlight-current
              :filter-node-method="filterNode"
              @node-click="handleNodeClick"
              :expand-on-click-node="false"
              empty-text="暂无数据结构"
            >
              <template #default="{ node, data }">
                <div class="custom-tree-node flex items-center text-sm py-1">
                  <!-- User requested to remove icons to save space -->
                  <span class="truncate font-medium text-gray-700" :title="node.label">{{ node.label }}</span>
                  <span v-if="data.children && data.children.length > 0" class="text-gray-400 text-xs ml-2">({{ data.children.length }})</span>
                </div>
              </template>
            </el-tree>
            
            <div v-if="loadingStructure" class="py-10 text-center text-gray-400">
              <el-icon class="is-loading text-xl mb-2"><Loading /></el-icon>
              <p class="text-xs">正在加载数据结构...</p>
            </div>
          </div>
        </div>

        <!-- Right: Parameter Values -->
        <div class="dm-main flex-1 flex flex-col bg-gray-50/30">
          <div v-if="currentPath" class="h-full flex flex-col">
            <!-- Breadcrumb / Path Header -->
            <div class="p-4 bg-white border-b border-gray-100 flex justify-between items-center shadow-sm z-10">
              <div class="flex flex-col gap-1">
                <span class="text-xs text-gray-400">当前路径</span>
                <span class="font-mono text-sm font-semibold text-primary break-all">{{ currentPath }}</span>
              </div>
              <el-button type="primary" text bg size="small" :icon="RefreshRight" @click="refreshValues" :loading="loadingValues">
                刷新数值
              </el-button>
            </div>

            <!-- Table -->
            <div class="flex-1 p-4 overflow-hidden">
              <el-card shadow="never" class="h-full border-0 flex flex-col" :body-style="{ padding: '0', height: '100%', display: 'flex', flexDirection: 'column' }">
                <el-table
                  :data="tableData"
                  style="width: 100%"
                  height="100%"
                  v-loading="loadingValues"
                  stripe
                >
                  <el-table-column prop="name" label="参数名" min-width="200" show-overflow-tooltip sortable>
                     <template #default="scope">
                        <span class="font-mono text-xs">{{ scope.row.name.replace(currentPath, '') }}</span>
                        <span class="text-gray-400 text-xs ml-2">({{ scope.row.name }})</span>
                     </template>
                  </el-table-column>
                  <el-table-column prop="valueJson" label="值" min-width="150" show-overflow-tooltip>
                    <template #default="scope">
                      <div class="flex items-center justify-between group">
                        <span class="font-mono text-sm truncate">{{ formatValue(scope.row.valueJson) }}</span>
                        <el-icon class="cursor-pointer opacity-0 group-hover:opacity-100 text-gray-400 hover:text-primary transition-opacity" @click="copyValue(formatValue(scope.row.valueJson))">
                          <CopyDocument />
                        </el-icon>
                      </div>
                    </template>
                  </el-table-column>
                  <el-table-column prop="valueType" label="类型" width="100">
                    <template #default="scope">
                      <el-tag size="small" type="info" effect="plain">{{ scope.row.valueType }}</el-tag>
                    </template>
                  </el-table-column>
                  <el-table-column prop="updatedAt" label="更新时间" width="160">
                    <template #default="scope">
                      <span class="text-xs text-gray-500">{{ formatDate(scope.row.UpdatedAt) }}</span>
                    </template>
                  </el-table-column>
                </el-table>
              </el-card>
            </div>
          </div>

          <!-- Empty State -->
          <div v-else class="h-full flex flex-col items-center justify-center text-gray-400">
            <el-empty description="请从左侧选择一个节点查看参数" :image-size="120" />
          </div>
        </div>
      </div>
    </div>
    
  </el-drawer>
</template>

<script setup>
import { ref, watch, computed } from 'vue'
import { Monitor, Folder, Document, Search, Refresh, Download, RefreshRight, CopyDocument, Loading } from '@element-plus/icons-vue'
import { getDataModelStructure, getDataModelList } from '@/plugin/tr069/api/datamodel'
import { fullDataModelSync } from '@/plugin/tr069/api/command'
import { formatTimeToStr } from '@/utils/date'
import { ElMessage } from 'element-plus'
import { useClipboard } from '@vueuse/core'

const props = defineProps({
  modelValue: {
    type: Boolean,
    default: false
  },
  row: {
    type: Object,
    default: () => ({})
  }
})

const emit = defineEmits(['update:modelValue'])

const visible = computed({
  get: () => props.modelValue,
  set: (val) => emit('update:modelValue', val)
})

const deviceRow = computed(() => props.row || {})
const treeRef = ref(null)
const filterText = ref('')
const treeData = ref([])
const tableData = ref([])
const currentPath = ref('')
const loadingStructure = ref(false)
const loadingValues = ref(false)

const fullSyncSubmitting = ref(false)

const defaultProps = {
  children: 'children',
  label: 'label'
}

const { copy } = useClipboard()

watch(filterText, (val) => {
  treeRef.value?.filter(val)
})

watch(() => props.modelValue, (val) => {
  if (val && props.row?.ID) {
    refreshStructure()
    currentPath.value = ''
    tableData.value = []
  }
})

watch(() => props.row?.ID, (newId, oldId) => {
  if (props.modelValue && newId && newId !== oldId) {
    refreshStructure()
    currentPath.value = ''
    tableData.value = []
  }
})

const filterNode = (value, data) => {
  if (!value) return true
  return data.label.toLowerCase().includes(value.toLowerCase())
}

const handleClose = (done) => {
  done()
}

const refreshStructure = async () => {
  if (!props.row.ID) return
  loadingStructure.value = true
  treeData.value = []
  try {
    const res = await getDataModelStructure(props.row.ID)
    if (res.code === 0 && Array.isArray(res.data)) {
      treeData.value = buildDMTree(res.data)
    } else {
      ElMessage.error(res.msg || '获取参数树结构失败')
    }
  } catch (error) {
    console.error(error)
  } finally {
    loadingStructure.value = false
  }
}

const buildDMTree = (paths) => {
  const root = []
  const findOrCreate = (nodes, part, fullPath) => {
    let node = nodes.find(n => n.label === part)
    if (!node) {
      node = {
        id: fullPath,
        label: part,
        children: []
      }
      nodes.push(node)
    }
    return node
  }
  
  // Sort paths to ensure parent nodes are processed before children if possible, 
  // though the logic handles it regardless.
  const sortedPaths = (paths || []).slice().sort()
  
  sortedPaths.forEach(path => {
    const parts = String(path).split('.').filter(Boolean)
    let currentLevel = root
    let currentPath = ''
    parts.forEach(part => {
      currentPath += part + '.'
      const node = findOrCreate(currentLevel, part, currentPath)
      currentLevel = node.children
    })
  })
  return root
}

const handleNodeClick = async (node) => {
  if (!node || !node.id) return
  currentPath.value = node.id
  await refreshValues()
}

const refreshValues = async () => {
  if (!props.row.ID || !currentPath.value) return
  loadingValues.value = true
  try {
    const res = await getDataModelList(props.row.ID, { prefix: currentPath.value, page: 1, pageSize: 2000 })
    if (res.code === 0 && res.data) {
      tableData.value = res.data.list || []
    }
  } catch (error) {
    ElMessage.error('获取参数值失败')
  } finally {
    loadingValues.value = false
  }
}

const formatValue = (v) => {
  if (v === null || v === undefined) return ''
  if (typeof v === 'object') return JSON.stringify(v)
  return String(v)
}

const formatDate = (time) => {
  if (time && time !== '0001-01-01T00:00:00Z') {
    return formatTimeToStr(time)
  }
  return '-'
}

const copyValue = (text) => {
  if (!text) return
  copy(text)
  ElMessage.success('已复制')
}

const openFullSync = async () => {
  if (fullSyncSubmitting.value) return
  fullSyncSubmitting.value = true
  try {
    const res = await fullDataModelSync(props.row.ID, { paths: ['Device.'] })
    if (res.code === 0) {
      ElMessage.success('已下发全量同步请求')
    } else {
      ElMessage.error(res.msg || '下发失败')
    }
  } catch (error) {
    ElMessage.error('系统错误')
  } finally {
    fullSyncSubmitting.value = false
  }
}
</script>

<style scoped>
.custom-scrollbar::-webkit-scrollbar {
  width: 6px;
  height: 6px;
}
.custom-scrollbar::-webkit-scrollbar-thumb {
  background: #e5e7eb;
  border-radius: 3px;
}
.custom-scrollbar::-webkit-scrollbar-track {
  background: transparent;
}
</style>
