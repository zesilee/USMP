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
      fit
    >
      <el-table-column
        v-for="col in columns"
        :key="col.name"
        :prop="col.name"
        :label="col.description || col.name"
        :min-width="getColumnWidth(col)"
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
        :min-width="140"
        align="center"
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
      :title="dialogTitle"
      width="600px"
    >
      <el-form
        ref="formRef"
        :model="editForm"
        label-width="120px"
        class="vlan-form"
      >
        <YangField
          v-for="child in childNodes"
          :key="child.path"
          :node="child"
          v-model="editForm[child.name]"
          @validate="(r) => handleFieldValidate(child.name, r)"
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
import { ref, computed, reactive, nextTick } from 'vue'
import { Plus } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import YangField from './YangField.vue'
import { validateField, kebabToCamel, convertKeysToCamel, convertKeysToKebab, type YangNode, type FormData, type FieldValue } from '../../types/yang-schema'
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

const dialogTitle = computed(() => {
  const name = props.node.name || ''
  let itemName = '项'
  if (name.includes('vlan')) itemName = 'VLAN'
  if (name.includes('interface')) itemName = '接口'
  if (name.includes('port')) itemName = '端口'
  return isEditing.value ? `编辑 ${itemName}` : `新建 ${itemName}`
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

// 收集所有子节点的验证结果
const validationResults = ref<Record<string, boolean>>({})

const handleFieldValidate = (fieldName: string, result: any) => {
  validationResults.value[fieldName] = result.valid
}

const hasValidationErrors = computed(() => {
  return Object.values(validationResults.value).some(v => v === false)
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

const getColumnWidth = (col: YangNode): number => {
  if (col.type === 'boolean') return 80
  if (col.type === 'uint' || col.type === 'int') return 100
  if (col.type === 'enum') return 120
  if (col.name === 'name' || col.name === 'description') return 180
  return 150
}

const getEnumLabel = (col: YangNode, value: FieldValue): string => {
  const opt = col.enumOptions?.find(o => o.value === value)
  return opt?.name || String(value || '-')
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
  validationResults.value = {}
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
  validationResults.value = {}
  Object.keys(editForm).forEach(key => delete editForm[key])
  // 确保键名格式一致，转换为 kebab-case
  const normalizedRow = convertKeysToKebab(row)
  Object.assign(editForm, normalizedRow)
  dialogVisible.value = true
}

const handleDelete = (index: number) => {
  const newValue = [...tableData.value]
  newValue.splice(index, 1)
  emit('update:modelValue', newValue)
}

const handleSave = async () => {
  // 触发表单验证 - 先触发所有字段的验证
  let hasErrors = false
  childNodes.value.forEach(node => {
    const result = validateField(node, editForm[node.name])
    if (!result.valid) {
      hasErrors = true
      validationResults.value[node.name] = false
    }
  })

  if (hasErrors) {
    ElMessage.error('请检查表单中的错误')
    return
  }

  // 保存前确保键名格式一致
  const formDataKebab = convertKeysToKebab({ ...editForm })
  const newValue = [...tableData.value]
  if (isEditing.value && editIndex.value >= 0) {
    newValue[editIndex.value] = formDataKebab
  } else {
    newValue.push(formDataKebab)
  }
  emit('update:modelValue', newValue)
  dialogVisible.value = false
  ElMessage.success('保存成功')
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
    white-space: nowrap;
  }
}

:deep(.el-table .cell) {
  white-space: nowrap;
  padding-left: 12px;
  padding-right: 12px;
}

:deep(.el-dialog) {
  background-color: var(--bg-card);
  border: 1px solid var(--border-color);
}

:deep(.el-table th .cell) {
  font-weight: var(--font-weight-semibold);
}

:deep(.el-table__body-wrapper) {
  overflow-x: auto;
}

/* 表格容器添加横向滚动支持，小屏幕下可滚动 */
.yang-table {
  overflow-x: auto;
}


:deep(.el-dialog__overlay) {
  background-color: rgba(0, 0, 0, 0.7);
  backdrop-filter: blur(4px);
}

.vlan-form {
  :deep(.el-form-item) {
    margin-bottom: 16px;
  }

  :deep(.el-input__wrapper) {
    width: 100%;
  }

  :deep(.el-select) {
    width: 100%;
  }

  :deep(.el-input-number) {
    width: 100%;
  }

  :deep(.el-input-number .el-input__inner) {
    text-align: left;
  }
}
</style>
