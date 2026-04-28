<template>
  <div class="yang-table">
    <div class="table-header">
      <span class="table-title">{{ title }}</span>
      <el-button
        v-if="editable"
        type="primary"
        size="small"
        @click="handleAdd"
      >
        <el-icon><Plus /></el-icon>
        {{ addButtonText }}
      </el-button>
    </div>

    <el-table
      :data="tableData"
      style="width: 100%"
      border
    >
      <el-table-column
        v-for="col in columns"
        :key="col.name"
        :prop="col.name"
        :label="col.description || col.name"
        :width="getColumnWidth(col)"
      >
        <template #default="{ row }">
          <span v-if="col.type === 'boolean'">
            {{ getFieldValue(row, col.name) ? '是' : '否' }}
          </span>
          <span v-else-if="col.type === 'enum'">
            {{ getEnumLabel(col, getFieldValue(row, col.name)) }}
          </span>
          <span v-else-if="Array.isArray(getFieldValue(row, col.name))">
            {{ getFieldValue(row, col.name).length }} 项
          </span>
          <span v-else>
            {{ getFieldValue(row, col.name) ?? '-' }}
          </span>
        </template>
      </el-table-column>

      <el-table-column
        v-if="editable"
        label="操作"
        width="120"
        fixed="right"
      >
        <template #default="{ row, $index }">
          <el-button size="small" @click="handleEdit(row, $index)">
            编辑
          </el-button>
          <el-button size="small" type="danger" @click="handleDelete($index)">
            删除
          </el-button>
        </template>
      </el-table-column>
    </el-table>

    <!-- 编辑弹窗 -->
    <el-dialog
      v-model="dialogVisible"
      :title="isEditing ? '编辑' : '新增'"
      width="500px"
    >
      <el-form
        ref="formRef"
        :model="editForm"
        label-width="100px"
      >
        <YangField
          v-for="child in childNodes"
          :key="child.path"
          :node="child"
          v-model="editForm[child.name]"
        />
      </el-form>

      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleSave">
          保存
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, reactive } from 'vue'
import { Plus } from '@element-plus/icons-vue'
import YangField from './YangField.vue'
import type { YangNode, FormData, FieldValue } from '../../types/yang-schema'
import type { FormInstance } from 'element-plus'

interface Props {
  node: YangNode
  modelValue: Record<string, any>[]
  editable?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  editable: true
})

const emit = defineEmits<{
  'update:modelValue': [value: Record<string, any>[]]
}>()

const tableData = computed(() => props.modelValue || [])

const title = computed(() =>
  props.node.description || props.node.name
)

const addButtonText = computed(() => {
  const name = props.node.name || ''
  if (name.includes('vlan')) return '新建 VLAN'
  if (name.includes('interface')) return '新建接口'
  return '新建'
})

// 对于 VLAN schema，list node 是 /vlans/vlan，它的子节点就是字段
// 对于其他嵌套结构，需要适配
const childNodes = computed(() => {
  // 如果 node 有 children，第一个 child 通常是 list item
  const listItem = props.node.children?.[0]
  if (listItem?.children) {
    return listItem.children.filter(c => c.config !== false) || []
  }
  // 否则直接使用 node 的 children
  return props.node.children?.filter(c => c.config !== false) || []
})

const columns = computed(() => {
  const listItem = props.node.children?.[0]
  if (listItem?.children) {
    return listItem.children
  }
  return props.node.children || []
})

const editable = computed(() => props.editable && props.node.config !== false)

const dialogVisible = ref(false)
const isEditing = ref(false)
const editIndex = ref(-1)
const formRef = ref<FormInstance>()
const editForm = reactive<Record<string, FieldValue>>({})

const getColumnWidth = (col: YangNode): number | undefined => {
  if (col.type === 'boolean') return 80
  if (col.type === 'uint' || col.type === 'int') return 120
  return undefined
}

const getEnumLabel = (col: YangNode, value: FieldValue): string => {
  const opt = col.enumOptions?.find(o => o.value === value)
  return opt?.name || String(value || '-')
}

// kebab-case to camelCase: admin-status -> adminStatus
const kebabToCamel = (str: string): string => {
  return str.replace(/-([a-z])/g, (_, c) => c.toUpperCase())
}

// 兼容处理字段名：支持 schema 中的 kebab-case 和数据中的 camelCase
const getFieldValue = (row: Record<string, any>, fieldName: string): any => {
  // 先尝试直接访问
  if (row[fieldName] !== undefined) {
    return row[fieldName]
  }
  // 尝试 camelCase 版本
  const camelName = kebabToCamel(fieldName)
  if (row[camelName] !== undefined) {
    return row[camelName]
  }
  return undefined
}

const handleAdd = () => {
  isEditing.value = false
  editIndex.value = -1
  Object.keys(editForm).forEach(key => delete editForm[key])
  // 设置默认值
  childNodes.value.forEach(child => {
    if (child.default !== undefined) {
      editForm[child.name] = child.default
    }
  })
  dialogVisible.value = true
}

const handleEdit = (row: Record<string, any>, index: number) => {
  isEditing.value = true
  editIndex.value = index
  Object.keys(editForm).forEach(key => delete editForm[key])
  Object.assign(editForm, row)
  dialogVisible.value = true
}

const handleDelete = (index: number) => {
  const newValue = [...tableData.value]
  newValue.splice(index, 1)
  emit('update:modelValue', newValue)
}

const handleSave = () => {
  const newValue = [...tableData.value]
  if (isEditing.value && editIndex.value >= 0) {
    newValue[editIndex.value] = { ...editForm }
  } else {
    newValue.push({ ...editForm })
  }
  emit('update:modelValue', newValue)
  dialogVisible.value = false
}
</script>

<style lang="scss" scoped>
.yang-table {
  .table-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: var(--spacing-lg);
  }

  .table-title {
    font-size: var(--font-size-base);
    font-weight: var(--font-weight-semibold);
    color: var(--text-primary);
  }
}

:deep(.el-dialog) {
  background-color: var(--bg-card);
  border: 1px solid var(--border-color);
}

:deep(.el-dialog__overlay) {
  background-color: rgba(0, 0, 0, 0.7);
  backdrop-filter: blur(4px);
}
</style>
