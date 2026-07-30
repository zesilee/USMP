<template>
  <template v-for="(node, i) in nodes" :key="`${indexPrefix}-${i}`">
    <!-- 模块级子节点（LT-02/LT-03）：container 路由控制台、rpc 路由直达执行页，
         与 YANG 模块顶层同级平铺；高危 rpc 带警示图标（R12 真实图标）。 -->
    <el-menu-item
      v-if="node.kind === 'container'"
      :index="`/module/${node.name}`"
      :title="label(node)"
      :data-test="`lefttree-node-${node.name}`"
    >
      <span>{{ label(node) }}</span>
    </el-menu-item>
    <el-menu-item
      v-else-if="node.kind === 'rpc'"
      :index="`/module/${moduleContext}/rpc/${node.name}`"
      :title="label(node)"
      :data-test="`lefttree-rpc-${moduleContext}-${node.name}`"
    >
      <span>{{ label(node) }}</span>
      <el-icon v-if="node.highRisk" class="rpc-high-risk"><WarningFilled /></el-icon>
    </el-menu-item>
    <!-- 模块叶（已接入且带模块级子节点）：可展开分组，锚点保留在叶上（LT-03） -->
    <el-sub-menu
      v-else-if="node.sourceModule && node.available && node.children?.length"
      :index="`${indexPrefix}-${i}`"
      :data-test="`lefttree-leaf-${node.sourceModule}`"
    >
      <template #title>{{ label(node) }}</template>
      <LeftTreeMenu
        :nodes="node.children"
        :index-prefix="`${indexPrefix}-${i}`"
        :module-context="node.module"
      />
    </el-sub-menu>
    <!-- 模块叶（无子节点载荷回退直达 / 未接入禁用 + 提示，全树+占位） -->
    <el-menu-item
      v-else-if="node.sourceModule"
      :index="node.available && node.module ? `/module/${node.module}` : `${indexPrefix}-${i}-na`"
      :disabled="!node.available"
      :title="node.available ? label(node) : `${label(node)} (${t('nav.notOnboarded')})`"
      :data-test="`lefttree-leaf-${node.sourceModule}`"
    >
      <span>{{ label(node) }}</span>
      <span v-if="!node.available" class="na-tag">{{ t('nav.notOnboarded') }}</span>
    </el-menu-item>
    <!-- 分组：递归渲染子层 -->
    <el-sub-menu v-else :index="`${indexPrefix}-${i}`" :data-test="`lefttree-group-${node.zh}`">
      <template #title>{{ label(node) }}</template>
      <LeftTreeMenu :nodes="node.children || []" :index-prefix="`${indexPrefix}-${i}`" />
    </el-sub-menu>
  </template>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { WarningFilled } from '@element-plus/icons-vue'
import type { LeftTreeNode } from '../../stores/menu'

defineOptions({ name: 'LeftTreeMenu' })

defineProps<{
  nodes: LeftTreeNode[]
  indexPrefix: string
  // 模块叶的路由 module（首个已加载根容器）：rpc 子节点路由前缀用。
  moduleContext?: string
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

.rpc-high-risk {
  margin-left: 6px;
  color: var(--el-color-warning, #e6a23c);
}
</style>
