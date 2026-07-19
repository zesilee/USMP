<template>
  <template v-for="(node, i) in nodes" :key="`${indexPrefix}-${i}`">
    <!-- 叶子：已接入可点（路由通用模块控制台）；未接入禁用 + 提示（LT-03 全树+占位） -->
    <el-menu-item
      v-if="node.sourceModule"
      :index="node.available && node.module ? `/module/${node.module}` : `${indexPrefix}-${i}-na`"
      :disabled="!node.available"
      :title="node.available ? label(node) : `${label(node)} (${t('nav.notOnboarded')})`"
      :data-test="`lefttree-leaf-${node.sourceModule}`"
    >
      <span>{{ label(node) }}</span>
      <span v-if="!node.available" class="na-tag">{{ t('nav.notOnboarded') }}</span>
    </el-menu-item>
    <!-- 分组：递归渲染子层（left-tree ≤3 层） -->
    <el-sub-menu v-else :index="`${indexPrefix}-${i}`" :data-test="`lefttree-group-${node.zh}`">
      <template #title>{{ label(node) }}</template>
      <LeftTreeMenu :nodes="node.children || []" :index-prefix="`${indexPrefix}-${i}`" />
    </el-sub-menu>
  </template>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import type { LeftTreeNode } from '../../stores/menu'

defineOptions({ name: 'LeftTreeMenu' })

defineProps<{
  nodes: LeftTreeNode[]
  indexPrefix: string
}>()

const { t, locale } = useI18n()

// UI-02：左树双语（③期载荷已带 zh/en）；对应语言字段缺失回退另一语言（R08）。
function label(node: LeftTreeNode): string {
  return locale.value === 'en-us' ? node.en || node.zh : node.zh || node.en
}
</script>

<style scoped>
.na-tag {
  margin-left: 6px;
  font-size: 11px;
  color: var(--el-text-color-placeholder, #a8abb2);
}
</style>
