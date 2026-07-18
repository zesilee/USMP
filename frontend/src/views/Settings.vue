<template>
  <div class="settings">
    <div class="page-header">
      <h2>{{ t('settings.title') }}</h2>
      <div class="sub">{{ t('settings.subtitle') }}</div>
    </div>

    <div class="set-grid">
      <div v-for="card in cards" :key="card.title" class="card">
        <div class="card-h">
          <h3>{{ card.title }}</h3>
          <span v-if="card.meta" class="meta">{{ card.meta }}</span>
        </div>
        <div class="card-b">
          <div v-for="row in card.rows" :key="row.k" class="set-row">
            <div class="k"><b>{{ row.k }}</b><span>{{ row.hint }}</span></div>
            <div class="v" :class="{ muted: row.muted }">{{ row.v }}</div>
          </div>
        </div>
      </div>
    </div>

    <div class="footnote">
      {{ t('settings.footnote') }}
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

// 只读架构事实（非可编辑设置——无设置持久化后端，展示系统实际策略更诚实）。
// 数值与后端一致：runningCache=TTL 30s/LRU 4096（manager.go）；端口见 CLAUDE.md §1/§3。
const cards = computed(() => [
  {
    title: t('settings.protocolCard'),
    meta: '',
    rows: [
      { k: t('settings.netconfPort'), hint: t('settings.netconfPortHint'), v: '830', muted: false },
      { k: t('settings.gnmiPort'), hint: t('settings.gnmiPortHint'), v: '9339 / 9340', muted: true },
      { k: t('settings.reconnect'), hint: t('settings.reconnectHint'), v: t('common.enabled'), muted: false },
      { k: t('settings.connTimeout'), hint: t('settings.connTimeoutHint'), v: '10s', muted: false },
    ],
  },
  {
    title: t('settings.cacheCard'),
    meta: t('settings.cacheMeta'),
    rows: [
      { k: t('settings.cacheTtl'), hint: t('settings.cacheTtlHint'), v: '30s', muted: false },
      { k: t('settings.lruCapacity'), hint: t('settings.lruCapacityHint'), v: t('settings.lruCapacityValue'), muted: false },
      { k: t('settings.invalidateOnPush'), hint: t('settings.invalidateOnPushHint'), v: t('common.enabled'), muted: false },
      { k: t('settings.persistence'), hint: t('settings.persistenceHint'), v: t('common.disabled'), muted: true },
    ],
  },
])
</script>

<style scoped>
.settings {
  padding: 20px;
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.page-header h2 {
  margin: 0;
  font-size: 20px;
  font-weight: 600;
  color: var(--ink, #1f2d3d);
}

.page-header .sub {
  margin-top: 4px;
  font-size: 12.5px;
  color: var(--ink-3, #93a2b1);
}

.set-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 16px;
}

.card {
  background: var(--bg-card, #fff);
  border: 1px solid var(--line, #e6ebf0);
  border-radius: var(--r-card, 12px);
  overflow: hidden;
}

.card-h {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 15px 18px;
  border-bottom: 1px solid var(--line, #e6ebf0);
}

.card-h h3 {
  margin: 0;
  font-size: 14.5px;
  font-weight: 600;
  color: var(--ink, #1f2d3d);
}

.card-h .meta {
  font-size: 12px;
  color: var(--ink-3, #93a2b1);
}

.card-b {
  padding: 4px 18px 10px;
}

.set-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 14px 0;
  border-bottom: 1px solid var(--line, #e6ebf0);
}

.set-row:last-child {
  border-bottom: none;
}

.set-row .k b {
  display: block;
  font-size: 13.5px;
  font-weight: 600;
  color: var(--ink, #1f2d3d);
}

.set-row .k span {
  font-size: 12px;
  color: var(--ink-3, #93a2b1);
}

.set-row .v {
  font-family: var(--f-mono, monospace);
  font-size: 13px;
  color: var(--ink, #1f2d3d);
}

.set-row .v.muted {
  color: var(--ink-3, #93a2b1);
}

.footnote {
  font-size: 11.5px;
  line-height: 1.6;
  color: var(--ink-3, #93a2b1);
}

@media (max-width: 768px) {
  .set-grid {
    grid-template-columns: 1fr;
  }
}
</style>
