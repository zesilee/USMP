<template>
  <template v-for="(node, i) in nodes" :key="`${indexPrefix}-${i}`">
    <!-- 叶子：已接入可点（路由通用模块控制台）；未接入禁用 + 提示（LT-03 全树+占位） -->
    <el-menu-item
      v-if="node.sourceModule"
      :index="node.available && node.module ? `/module/${node.module}` : `${indexPrefix}-${i}-na`"
      :disabled="!node.available"
      :title="node.available ? node.zh : `${node.zh}（未接入）`"
      :data-test="`lefttree-leaf-${node.sourceModule}`"
    >
      <span>{{ node.zh }}</span>
      <span v-if="!node.available" class="na-tag">未接入</span>
    </el-menu-item>
    <!-- 分组：递归渲染子层（left-tree ≤3 层） -->
    <el-sub-menu v-else :index="`${indexPrefix}-${i}`" :data-test="`lefttree-group-${node.zh}`">
      <template #title>{{ node.zh }}</template>
      <LeftTreeMenu :nodes="node.children || []" :index-prefix="`${indexPrefix}-${i}`" />
    </el-sub-menu>
  </template>
</template>

<script setup lang="ts">
import type { LeftTreeNode } from '../../stores/menu'

defineOptions({ name: 'LeftTreeMenu' })

defineProps<{
  nodes: LeftTreeNode[]
  indexPrefix: string
}>()
</script>

<style scoped>
.na-tag {
  margin-left: 6px;
  font-size: 11px;
  color: var(--el-text-color-placeholder, #a8abb2);
}
</style>
