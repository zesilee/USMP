<template>
  <span class="chip" :class="variant.cls">
    <span class="glyph" aria-hidden="true"></span>
    {{ variant.label }}
  </span>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { DisplayState } from '../../composables/useFleetOverview'

// 对账四态 chip（收敛/收敛中/漂移/失败/离线/未对账）。原型 chip 口径 + 令牌配色。
const props = defineProps<{
  state: DisplayState
}>()

const VARIANTS: Record<DisplayState, { label: string; cls: string }> = {
  conv: { label: '已收敛', cls: 'conv' },
  recon: { label: '收敛中', cls: 'recon' },
  drift: { label: '已漂移', cls: 'drift' },
  error: { label: '下发失败', cls: 'error' },
  off: { label: '离线', cls: 'off' },
  unknown: { label: '未对账', cls: 'unknown' },
}

const variant = computed(() => VARIANTS[props.state] ?? VARIANTS.unknown)
</script>

<style scoped>
.chip {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  height: 23px;
  padding: 0 9px 0 8px;
  border-radius: var(--r-chip);
  font-size: 12px;
  font-weight: 600;
  white-space: nowrap;
}
.chip .glyph {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  flex-shrink: 0;
}

.chip.conv {
  background: var(--st-conv-bg);
  color: var(--st-conv);
}
.chip.conv .glyph {
  background: var(--st-conv);
}

.chip.recon {
  background: var(--st-recon-bg);
  color: var(--st-recon);
}
.chip.recon .glyph {
  background: var(--st-recon);
  animation: blink 1.1s steps(2, jump-none) infinite;
}

.chip.drift {
  background: var(--st-drift-bg);
  color: var(--st-drift);
}
.chip.drift .glyph {
  border-radius: 1px;
  transform: rotate(45deg);
  background: var(--st-drift);
}

.chip.error,
.chip.off {
  background: var(--st-off-bg);
  color: var(--st-off);
}
.chip.error .glyph,
.chip.off .glyph {
  background: var(--st-off);
}
.chip.off .glyph {
  border-radius: 50%;
  box-shadow: 0 0 0 2px var(--st-off-bg);
}

.chip.unknown {
  background: var(--sunken);
  color: var(--ink-3);
}
.chip.unknown .glyph {
  background: var(--line-strong);
}

@keyframes blink {
  0% { opacity: 1; }
  50% { opacity: 0.35; }
  100% { opacity: 1; }
}
@media (prefers-reduced-motion: reduce) {
  .chip .glyph { animation: none; }
}
</style>
