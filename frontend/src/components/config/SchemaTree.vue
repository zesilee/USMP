<template>
  <div class="schema-tree">
    <div class="tree-h">
      {{ t('console.tree.title') }}<span v-if="moduleLabel" class="mod mono"> · {{ moduleLabel }}</span>
    </div>

    <div v-if="!nodes.length" class="tree-empty">{{ t('console.tree.loading') }}</div>

    <div
      v-for="node in nodes"
      :key="node.path"
      class="ynode"
      :class="{ cfg: node.isConfig, ro: node.isReadonly }"
      :style="{ paddingLeft: 10 + node.depth * 14 + 'px' }"
      @click="$emit('node-click', node)"
    >
      <span class="kind" :class="node.kind">{{ kindLabel[node.kind] }}</span>
      <span class="nm">{{ node.name }}</span>
      <span class="rt">
        <span v-if="node.isKey" class="ty keyt">key</span>
        <span v-if="node.kind === 'list' && itemCounts[node.path] != null" class="count-pill">{{ itemCounts[node.path] }}</span>
        <span v-if="node.dataType" class="ty">{{ node.dataType }}</span>
      </span>
    </div>

    <div v-if="nodes.length" class="tree-foot">{{ t('console.tree.legend') }}</div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { Field } from '../../utils/crdSchemaParser'
import { deriveSchemaTree } from '../../utils/schemaTree'

const { t } = useI18n()

const props = withDefaults(
  defineProps<{
    fields: Field[]
    keyField?: string
    moduleLabel?: string
    itemCounts?: Record<string, number>
  }>(),
  { keyField: undefined, moduleLabel: '', itemCounts: () => ({}) },
)

defineEmits<{ (e: 'node-click', node: ReturnType<typeof deriveSchemaTree>[number]): void }>()

const kindLabel = computed<Record<'container' | 'list' | 'leaf', string>>(() => ({
  container: t('console.tree.kindContainer'),
  list: t('console.tree.kindList'),
  leaf: t('console.tree.kindLeaf'),
}))

const nodes = computed(() => deriveSchemaTree(props.fields, { keyField: props.keyField }))
</script>

<style scoped>
.schema-tree {
  padding: 8px;
  background: var(--bg-card, #fff);
  border: 1px solid var(--line, #e6ebf0);
  border-radius: var(--r-card, 12px);
  overflow-x: auto;
}

.tree-h {
  font-size: 11px;
  letter-spacing: 0.09em;
  text-transform: uppercase;
  color: var(--ink-3, #93a2b1);
  font-weight: 600;
  padding: 8px 10px;
}

.tree-h .mod {
  color: var(--ink, #1f2d3d);
  text-transform: none;
  letter-spacing: 0;
}

.tree-empty {
  padding: 12px;
  font-size: 12.5px;
  color: var(--ink-3, #93a2b1);
}

.ynode {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 5px 10px;
  border-radius: 6px;
  font-size: 12.5px;
  color: var(--ink-2, #52627a);
  width: max-content;
  min-width: 100%;
  cursor: default;
}

.ynode:hover {
  background: var(--sunken, #f4f6f9);
}

.ynode .kind {
  font-size: 9.5px;
  letter-spacing: 0.02em;
  font-weight: 700;
  padding: 1px 5px;
  border-radius: 4px;
  flex-shrink: 0;
}

.ynode .kind.container {
  background: #eaeef3;
  color: var(--ink-3, #93a2b1);
}

.ynode .kind.list {
  background: var(--primary-weak, #e6effb);
  color: var(--primary-ink, #1c4e93);
}

.ynode .kind.leaf {
  background: var(--st-conv-bg, #e4f2e8);
  color: var(--st-conv, #2f8a4c);
}

.ynode .nm {
  font-family: var(--f-mono, monospace);
  color: var(--ink, #1f2d3d);
  font-size: 12px;
  white-space: nowrap;
}

.ynode .rt {
  display: inline-flex;
  align-items: center;
  flex-wrap: nowrap;
  gap: 6px;
  flex-shrink: 0;
}

.ynode .ty {
  font-family: var(--f-mono, monospace);
  font-size: 10.5px;
  color: var(--ink-3, #93a2b1);
}

.ynode .ty.keyt {
  background: var(--primary-weak, #e6effb);
  color: var(--primary-ink, #1c4e93);
  font-weight: 700;
  padding: 0 5px;
  border-radius: 4px;
  font-size: 9.5px;
  letter-spacing: 0.02em;
}

.count-pill {
  font-family: var(--f-mono, monospace);
  font-size: 11px;
  background: var(--sunken, #f4f6f9);
  color: var(--ink-2, #52627a);
  border-radius: 999px;
  padding: 1px 7px;
}

.ynode.ro {
  opacity: 0.5;
}

.ynode.ro .kind.leaf {
  background: #eaeef3;
  color: var(--ink-3, #93a2b1);
}

.ynode.cfg .nm {
  color: var(--primary-ink, #1c4e93);
  font-weight: 600;
}

.tree-foot {
  font-size: 10.5px;
  color: var(--ink-3, #93a2b1);
  padding: 10px 12px 4px;
  line-height: 1.5;
  border-top: 1px solid var(--line, #e6ebf0);
  margin-top: 6px;
}
</style>
