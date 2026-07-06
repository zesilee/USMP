<template>
  <div class="header">
    <div class="header-left">
      <nav class="crumb" aria-label="面包屑">
        <span class="crumb-root">车队</span>
        <span class="crumb-sep">/</span>
        <b>{{ crumbLabel }}</b>
      </nav>

      <div class="field-search">
        <el-input
          v-model="searchText"
          placeholder="搜索设备、配置…"
          size="small"
        >
          <template #prefix>
            <el-icon><Search /></el-icon>
          </template>
        </el-input>
      </div>
    </div>

    <div class="header-right">
      <div class="device-status" :title="`${onlineCount}/${totalCount} 设备在线`">
        <span class="status-dot online"></span>
        <span>{{ onlineCount }}/{{ totalCount }} 在线</span>
      </div>

      <FreshnessRing
        :age-seconds="ageSeconds"
        :ttl-seconds="ttlSeconds"
        :has-data="hasData"
        :source="source"
      />

      <el-badge :value="notificationCount" :hidden="notificationCount === 0" class="notification-icon">
        <el-button circle size="small"><el-icon><Bell /></el-icon></el-button>
      </el-badge>

      <div class="user-area">
        <div class="avatar">A</div>
        <div class="user-meta">
          <b>admin</b>
          <span>网络管理员</span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRoute } from 'vue-router'
import { Bell, Search } from '@element-plus/icons-vue'
import FreshnessRing from './FreshnessRing.vue'
import { useLiveFreshness } from '../../composables/useFreshness'

// 顶栏：面包屑（随路由）+ 搜索 + 设备在线（真数据待接）+ 缓存新鲜度环（真数据）+ 告警 + 用户。
const route = useRoute()

// 路由名 → 中文面包屑标签。缺省回退路径末段，避免出现空标题。
const CRUMB_LABELS: Record<string, string> = {
  dashboard: '车队概览',
  devices: '设备管理',
  interface: '接口配置',
  vlan: 'VLAN 配置',
  route: '路由配置',
  native: '原生配置',
  logs: '操作日志',
  settings: '系统设置',
}
const crumbLabel = computed(() => {
  const name = route.name as string | undefined
  if (name && CRUMB_LABELS[name]) return CRUMB_LABELS[name]
  const seg = route.path.split('/').filter(Boolean).pop()
  return seg ?? '概览'
})

// 缓存新鲜度环：消费 store 真数据 + 本地每秒时钟。无活跃缓存时环显示空态。
const { ageSeconds, ttlSeconds, hasData, source } = useLiveFreshness()

const searchText = ref('')
// TODO(PR-1/PR-3): 在线数接 /devices + /status 真数据；当前为占位。
const onlineCount = ref(10)
const totalCount = ref(12)
const notificationCount = ref(3)
</script>

<style scoped>
.header {
  height: var(--topbar-h);
  background: var(--surface);
  border-bottom: 1px solid var(--line);
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 22px;
  gap: 18px;
}

.header-left {
  display: flex;
  align-items: center;
  gap: 16px;
  min-width: 0;
}

.crumb {
  display: flex;
  align-items: center;
  gap: 7px;
  font-size: 13px;
  color: var(--ink-3);
  white-space: nowrap;
}
.crumb b {
  color: var(--ink);
  font-weight: 600;
}
.crumb-sep {
  color: var(--line-strong);
}

.field-search {
  width: 240px;
  max-width: 32vw;
}

.header-right {
  display: flex;
  align-items: center;
  gap: 14px;
}

.device-status {
  display: flex;
  align-items: center;
  gap: 7px;
  font-size: 13px;
  color: var(--ink-2);
  white-space: nowrap;
}

.status-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  flex-shrink: 0;
}
.status-dot.online {
  background: var(--st-conv);
  box-shadow: 0 0 0 3px var(--st-conv-bg);
}

.notification-icon {
  cursor: pointer;
}

.user-area {
  display: flex;
  align-items: center;
  gap: 9px;
  cursor: pointer;
  padding-left: 4px;
}
.avatar {
  width: 30px;
  height: 30px;
  border-radius: 8px;
  background: var(--primary);
  color: #fff;
  display: grid;
  place-items: center;
  font-weight: 600;
  font-size: 13px;
  flex-shrink: 0;
}
.user-meta {
  font-size: 12.5px;
  line-height: 1.25;
}
.user-meta b {
  display: block;
  font-weight: 600;
  color: var(--ink);
}
.user-meta span {
  color: var(--ink-3);
  font-size: 11px;
}

@media (max-width: 720px) {
  .field-search,
  .user-meta {
    display: none;
  }
}
</style>
