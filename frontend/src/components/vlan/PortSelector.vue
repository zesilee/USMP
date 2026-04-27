<template>
  <div class="port-selector">
    <el-popover
      ref="popoverRef"
      placement="bottom-start"
      width="500"
      trigger="click"
    >
      <template #reference>
        <div class="selector-trigger">
          <template v-if="modelValue.length > 0">
            <span v-for="port in displayPorts" :key="port" class="selected-tag">
              {{ formatPortName(port) }}
            </span>
            <span v-if="modelValue.length > 3" class="more-tag">
              +{{ modelValue.length - 3 }}
            </span>
          </template>
          <span v-else class="placeholder-text">{{ placeholder }}</span>
          <el-icon class="arrow-icon"><ArrowDown /></el-icon>
        </div>
      </template>

      <div class="selector-panel">
        <div class="panel-search">
          <el-input
            v-model="searchText"
            placeholder="搜索端口..."
            size="small"
            clearable
          >
            <template #prefix>
              <el-icon><Search /></el-icon>
            </template>
          </el-input>
        </div>

        <div class="panel-groups">
          <div
            v-for="group in groupedPorts"
            :key="group.type"
            class="port-group"
          >
            <div class="group-title">{{ group.type }}</div>
            <div class="port-list">
              <el-tag
                v-for="port in group.ports"
                :key="port.name"
                :type="isSelected(port.name) ? 'primary' : 'info'"
                :effect="isSelected(port.name) ? 'dark' : 'plain'"
                class="port-item"
                @click="togglePort(port.name)"
              >
                {{ formatPortName(port.name) }}
              </el-tag>
            </div>
          </div>
        </div>

        <div class="panel-actions">
          <el-button size="small" @click="handleClear">清空</el-button>
          <el-button size="small" type="primary" @click="handleClose">确定</el-button>
        </div>
      </div>
    </el-popover>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { ArrowDown, Search } from '@element-plus/icons-vue'
import type { PortInfo } from '../../types/vlan'

interface Props {
  modelValue: string[]
  deviceIp: string
  placeholder?: string
}

const props = withDefaults(defineProps<Props>(), {
  placeholder: '选择端口'
})

const emit = defineEmits<{
  'update:modelValue': [ports: string[]]
}>()

const popoverRef = ref()
const searchText = ref('')

// Mock 端口数据
const allPorts = ref<PortInfo[]>([
  { name: 'GigabitEthernet0/1', type: 'GE', status: 'UP' },
  { name: 'GigabitEthernet0/2', type: 'GE', status: 'UP' },
  { name: 'GigabitEthernet0/3', type: 'GE', status: 'DOWN' },
  { name: 'GigabitEthernet0/4', type: 'GE', status: 'UP' },
  { name: 'GigabitEthernet0/5', type: 'GE', status: 'UP' },
  { name: 'GigabitEthernet0/6', type: 'GE', status: 'UP' },
  { name: 'GigabitEthernet0/7', type: 'GE', status: 'DOWN' },
  { name: 'GigabitEthernet0/8', type: 'GE', status: 'UP' },
  { name: 'TenGigabitEthernet1/1', type: '10GE', status: 'UP' },
  { name: 'TenGigabitEthernet1/2', type: '10GE', status: 'UP' },
])

const filteredPorts = computed(() => {
  if (!searchText.value) return allPorts.value
  return allPorts.value.filter(p =>
    p.name.toLowerCase().includes(searchText.value.toLowerCase())
  )
})

const groupedPorts = computed(() => {
  const groups: Record<string, PortInfo[]> = {}
  filteredPorts.value.forEach(port => {
    if (!groups[port.type]) groups[port.type] = []
    groups[port.type].push(port)
  })
  return Object.entries(groups).map(([type, ports]) => ({ type, ports }))
})

const displayPorts = computed(() => props.modelValue.slice(0, 3))

const isSelected = (portName: string) => props.modelValue.includes(portName)

const togglePort = (portName: string) => {
  const newValue = [...props.modelValue]
  const index = newValue.indexOf(portName)
  if (index === -1) {
    newValue.push(portName)
  } else {
    newValue.splice(index, 1)
  }
  emit('update:modelValue', newValue)
}

const handleClear = () => {
  emit('update:modelValue', [])
}

const handleClose = () => {
  popoverRef.value?.hide()
}

const formatPortName = (name: string) => {
  return name
    .replace('GigabitEthernet', 'GE')
    .replace('TenGigabitEthernet', '10GE')
    .replace('Ethernet', 'Eth')
}
</script>

<style lang="scss" scoped>
@import '../../styles/variables.scss';

.port-selector {
  width: 100%;
}

.selector-trigger {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 4px;
  min-height: 36px;
  padding: 4px 8px;
  background-color: $bg-elevated;
  border: 1px solid $border-color;
  border-radius: $radius-md;
  cursor: pointer;
  transition: all $transition-fast;

  &:hover {
    border-color: $border-color-light;
  }

  .selected-tag {
    padding: 2px 8px;
    background-color: $color-primary-bg;
    color: $color-primary-light;
    border-radius: $radius-sm;
    font-size: $font-size-xs;
    font-weight: $font-weight-medium;
  }

  .more-tag {
    font-size: $font-size-xs;
    color: $text-secondary;
  }

  .placeholder-text {
    color: $text-tertiary;
    font-size: $font-size-sm;
  }

  .arrow-icon {
    margin-left: auto;
    color: $text-tertiary;
    font-size: 14px;
  }
}

.selector-panel {
  .panel-search {
    margin-bottom: $spacing-md;
  }

  .panel-groups {
    max-height: 300px;
    overflow-y: auto;
    margin-bottom: $spacing-md;

    .port-group {
      margin-bottom: $spacing-md;

      &:last-child {
        margin-bottom: 0;
      }

      .group-title {
        font-size: $font-size-xs;
        color: $text-tertiary;
        font-weight: $font-weight-medium;
        margin-bottom: $spacing-sm;
        text-transform: uppercase;
        letter-spacing: 0.5px;
      }

      .port-list {
        display: flex;
        flex-wrap: wrap;
        gap: 6px;
      }
    }
  }

  .panel-actions {
    display: flex;
    justify-content: flex-end;
    gap: $spacing-sm;
    padding-top: $spacing-md;
    border-top: 1px solid $border-color;
  }
}

.port-item {
  cursor: pointer;
  transition: all $transition-fast;
  user-select: none;

  &:hover {
    transform: translateY(-1px);
  }
}

:deep(.el-popper) {
  --el-popover-bg-color: #{$bg-card};
  --el-border-color: #{$border-color};
}
</style>
