<template>
  <div class="settings">
    <div class="page-header">
      <h2>系统设置</h2>
      <div class="sub">连接与缓存策略 · 无数据库，运行配置实时读取（R03）</div>
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
      以上为平台架构固定策略（NETCONF/gNMI 端口、缓存 TTL/LRU），非运行时可改项；配置的下发在「配置下发」页完成。
    </div>
  </div>
</template>

<script setup lang="ts">
// 只读架构事实（非可编辑设置——无设置持久化后端，展示系统实际策略更诚实）。
// 数值与后端一致：runningCache=TTL 30s/LRU 4096（manager.go）；端口见 CLAUDE.md §1/§3。
const cards = [
  {
    title: '协议连接',
    meta: '',
    rows: [
      { k: 'NETCONF 端口', hint: 'SSH over 830', v: '830', muted: false },
      { k: 'gNMI 端口', hint: 'gRPC 遥测订阅（规划能力，当前未实现）', v: '9339 / 9340', muted: true },
      { k: '断线重连', hint: 'ClientPool 自动重试', v: '启用', muted: false },
      { k: '连接超时', hint: 'NETCONF 单次请求上限', v: '10s', muted: false },
    ],
  },
  {
    title: '缓存策略',
    meta: 'TTL + LRU · 内存',
    rows: [
      { k: '缓存 TTL', hint: '过期自动重拉设备配置', v: '30s', muted: false },
      { k: 'LRU 容量', hint: 'Key = 设备IP + YANG路径', v: '4096 条', muted: false },
      { k: '下发后失效', hint: 'edit-config 成功即清缓存', v: '启用', muted: false },
      { k: '持久化', hint: '运行配置不落库（R03）', v: '禁用', muted: true },
    ],
  },
]
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
