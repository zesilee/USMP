<template>
  <div class="yang-list-editor">
    <!-- 标签列表 -->
    <div class="tags-container">
      <el-tag
        v-for="(item, index) in localValue"
        :key="index"
        closable
        :disabled="disabled"
        @close="handleRemove(index)"
        class="item-tag"
      >
        {{ getItemDisplay(item) }}
      </el-tag>
    </div>

    <!-- 添加输入框 -->
    <div v-if="!disabled" class="add-item">
      <el-input
        v-model="newItemValue"
        size="small"
        placeholder="输入并按回车添加"
        @keyup.enter="handleAdd"
        class="add-input"
      />
      <el-button
        size="small"
        type="primary"
        @click="handleAdd"
        :disabled="!newItemValue.trim()"
      >
        添加
      </el-button>
    </div>

    <!-- 空状态 -->
    <div v-if="!localValue || localValue.length === 0" class="empty-hint">
      <span v-if="disabled">暂无数据</span>
      <span v-else>暂无数据，输入后按回车添加</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import type { YangNode } from '../../types/yang-schema'

interface Props {
  modelValue: any[]
  node: YangNode
  disabled?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  disabled: false
})

const emit = defineEmits<{
  'update:modelValue': [value: any[]]
}>()

const newItemValue = ref('')

const localValue = computed(() => {
  if (!props.modelValue) return []
  return Array.isArray(props.modelValue) ? props.modelValue : [props.modelValue]
})

// 获取 list item 的值字段
const getItemFieldName = (): string => {
  const firstChild = props.node.children?.[0]
  if (firstChild) {
    return firstChild.name
  }
  return 'value'
}

const getItemDisplay = (item: any): string => {
  if (typeof item === 'string' || typeof item === 'number') {
    return String(item)
  }
  if (typeof item === 'object' && item !== null) {
    const fieldName = getItemFieldName()
    return String(item[fieldName] || item.value || item.name || '')
  }
  return ''
}

const handleAdd = () => {
  const value = newItemValue.value.trim()
  if (!value) return

  // 判断是简单数组还是对象数组
  const firstChild = props.node.children?.[0]
  let newValue: any
  if (firstChild?.type === 'string' || firstChild?.type === 'uint') {
    // 有子节点定义，创建对象
    newValue = { [firstChild.name]: value }
  } else {
    // 简单数组
    newValue = value
  }

  emit('update:modelValue', [...localValue.value, newValue])
  newItemValue.value = ''
}

const handleRemove = (index: number) => {
  const newValue = [...localValue.value]
  newValue.splice(index, 1)
  emit('update:modelValue', newValue)
}
</script>

<style lang="scss" scoped>
.yang-list-editor {
  width: 100%;
  min-height: 40px;
}

.tags-container {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-bottom: 12px;
}

.item-tag {
  max-width: 200px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.add-item {
  display: flex;
  gap: 8px;
  align-items: center;
}

.add-input {
  flex: 1;
  max-width: 300px;
}

.empty-hint {
  font-size: 12px;
  color: var(--text-tertiary);
  padding: 8px 0;
}
</style>
