<template>
  <div class="card conv">
    <div class="conv-top">
      <div class="conv-lead">
        <div class="conv-pct mono">{{ overview.convergenceRate }}<small>%</small></div>
        <div class="conv-cap">{{ t('dashboard.hero.fleetPrefix') }}<b>{{ t('dashboard.hero.rateWord') }}</b><br />{{ t('dashboard.hero.rateDesc') }}</div>
      </div>
      <ReconcileChip v-if="overview.pendingCount > 0" state="recon" />
      <ReconcileChip v-else-if="overview.total > 0" state="conv" />
    </div>

    <div class="segbar" :title="t('dashboard.hero.segbarTitle')">
      <span
        v-for="seg in visibleSegments"
        :key="seg.key"
        :class="`s-${seg.key}`"
        :style="{ flexGrow: seg.grow }"
      ></span>
      <span v-if="visibleSegments.length === 0" class="s-empty" style="flex-grow: 1"></span>
    </div>

    <div class="legend">
      <div v-for="seg in overview.segments" :key="seg.key" class="legend-row">
        <span class="k" :class="`k-${seg.key}`"></span>{{ seg.label }}
        <span class="n mono">{{ seg.count }}</span>
      </div>
    </div>
    <div v-if="overview.unknownCount > 0" class="legend-foot">
      {{ t('dashboard.hero.unknownPre') }}<b class="mono">{{ overview.unknownCount }}</b>{{ t('dashboard.hero.unknownPost') }}
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { Overview } from '../../composables/useFleetOverview'
import ReconcileChip from './ReconcileChip.vue'

const { t } = useI18n()

// 车队收敛率 hero：大号收敛率 + 分段条 + 四态图例。纯展示，全部输入经 overview prop。
const props = defineProps<{
  overview: Overview
}>()

// 分段条只渲染 count>0 的段，避免零宽细缝。
const visibleSegments = computed(() => props.overview.segments.filter((s) => s.count > 0))
</script>

<style scoped>
.card {
  background: var(--surface);
  border: 1px solid var(--line);
  border-radius: var(--r-card);
  box-shadow: var(--sh-1);
}
.conv {
  padding: 20px 22px;
}
.conv-top {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  margin-bottom: 16px;
  gap: 12px;
}
.conv-lead {
  display: flex;
  align-items: baseline;
  gap: 14px;
}
.conv-pct {
  font-size: 52px;
  font-weight: 600;
  letter-spacing: -0.03em;
  line-height: 0.9;
  color: var(--ink);
}
.conv-pct small {
  font-size: 22px;
  color: var(--ink-3);
  font-weight: 500;
}
.conv-cap {
  font-size: 12.5px;
  color: var(--ink-2);
  max-width: 180px;
}
.conv-cap b {
  color: var(--ink);
  font-weight: 600;
}

.segbar {
  display: flex;
  height: 14px;
  border-radius: 5px;
  overflow: hidden;
  gap: 2px;
  background: var(--sunken);
  margin-bottom: 14px;
}
.segbar span {
  display: block;
  transition: flex-grow 0.6s;
}
.s-conv { background: var(--st-conv); }
.s-recon {
  background: repeating-linear-gradient(45deg, var(--st-recon), var(--st-recon) 4px, #3f7cc0 4px, #3f7cc0 8px);
}
.s-attention { background: var(--st-drift); }
.s-off { background: var(--st-off); }
.s-empty { background: transparent; }

.legend {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 8px 20px;
}
.legend-row {
  display: flex;
  align-items: center;
  gap: 9px;
  font-size: 13px;
  color: var(--ink-2);
}
.legend-row .k {
  width: 9px;
  height: 9px;
  border-radius: 3px;
  flex-shrink: 0;
}
.k-conv { background: var(--st-conv); }
.k-recon { background: var(--st-recon); }
.k-attention { background: var(--st-drift); }
.k-off { background: var(--st-off); }
.legend-row .n {
  font-weight: 600;
  color: var(--ink);
  margin-left: auto;
  font-size: 14px;
}
.legend-foot {
  margin-top: 12px;
  font-size: 12px;
  color: var(--ink-3);
}
.legend-foot b { color: var(--ink-2); }
</style>
