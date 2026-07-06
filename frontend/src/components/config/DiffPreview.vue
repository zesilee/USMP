<template>
  <div class="diff-preview">
    <div class="preview-head">
      下发预览 · <b>{{ diff.length }}</b> 项改动
    </div>
    <div v-if="diff.length" class="dva">
      <div v-for="d in diff" :key="d.key" class="dva-row">
        <div class="dk">{{ d.label }}</div>
        <div class="dv changed">
          <template v-if="d.isNew">
            <span class="now">{{ fmt(d.now) }}</span>
            <span class="tag-new">新增</span>
          </template>
          <template v-else>
            <span class="was">{{ fmt(d.was) }}</span>
            <span class="arrow">→</span>
            <span class="now">{{ fmt(d.now) }}</span>
          </template>
        </div>
      </div>
    </div>
    <div v-else class="preview-empty">尚无改动 · 修改字段后在此预览下发差异</div>
  </div>
</template>

<script setup lang="ts">
import type { DiffEntry } from '../../utils/configDiff'

defineProps<{ diff: DiffEntry[] }>()

// 空值显示占位；其余原样（数值/字符串）。
const fmt = (v: any): string => (v === '' || v == null ? '—' : String(v))
</script>

<style scoped>
.preview-head {
  display: flex;
  align-items: center;
  gap: 4px;
  background: var(--sunken, #f4f6f9);
  border: 1px solid var(--line, #e6ebf0);
  border-radius: var(--r-ctl, 8px);
  padding: 11px 14px;
  font-size: 12.5px;
  font-weight: 600;
  color: var(--ink, #1f2d3d);
  margin-bottom: 10px;
}

.preview-head b {
  font-family: var(--f-mono, monospace);
  color: var(--primary, #2266cc);
}

.dva {
  border: 1px solid var(--line, #e6ebf0);
  border-radius: var(--r-ctl, 8px);
  overflow: hidden;
}

.dva-row {
  display: grid;
  grid-template-columns: 120px 1fr;
  font-size: 12.5px;
  border-bottom: 1px solid var(--line, #e6ebf0);
}

.dva-row:last-child {
  border-bottom: none;
}

.dva-row .dk {
  padding: 9px 12px;
  background: var(--sunken, #f4f6f9);
  color: var(--ink-2, #52627a);
  font-weight: 500;
}

.dva-row .dv {
  padding: 9px 12px;
  font-family: var(--f-mono, monospace);
  display: flex;
  align-items: center;
  gap: 8px;
}

.dva-row .dv.changed {
  background: #fbf7ed;
}

.dva .now {
  color: var(--ink, #1f2d3d);
  font-weight: 600;
}

.dva .was {
  color: var(--st-drift, #b26a00);
  text-decoration: line-through;
  opacity: 0.7;
}

.arrow {
  color: var(--ink-3, #93a2b1);
  margin: 0 5px;
}

.tag-new {
  font-family: var(--f-sans, sans-serif); /* 父级 .dv 为等宽，标签回到 sans（对齐原型） */
  font-size: 10px;
  font-weight: 700;
  color: var(--st-conv, #2f8a4c);
  background: var(--st-conv-bg, #e4f2e8);
  padding: 1px 6px;
  border-radius: 4px;
  letter-spacing: 0.03em;
}

.preview-empty {
  padding: 16px;
  text-align: center;
  color: var(--ink-3, #93a2b1);
  font-size: 12.5px;
  border: 1px dashed var(--line-strong, #cfd8e3);
  border-radius: var(--r-ctl, 8px);
}
</style>
