<template>
  <div class="sidebar" :class="{ collapsed: isCollapsed }">
    <div class="brand">
      <div class="brand-mark" aria-hidden="true"></div>
      <div v-if="!isCollapsed" class="brand-text">
        <div class="brand-name">USMP</div>
        <div class="brand-sub">Switch Mgmt</div>
      </div>
    </div>

    <el-menu
      class="nav"
      :default-active="activeMenu"
      :collapse="isCollapsed"
      router
    >
      <el-menu-item index="/">
        <el-icon><DataLine /></el-icon>
        <template #title>概览</template>
      </el-menu-item>

      <el-menu-item index="/devices">
        <el-icon><Monitor /></el-icon>
        <template #title>设备管理</template>
      </el-menu-item>

      <!-- 业务配置菜单：/yang/modules 模型驱动，指向通用模块控制台（FE-13）。
           加载失败回退内置项（R08）；路由配置为 legacy 独立流，保留静态项。 -->
      <el-sub-menu index="business-config">
        <template #title>
          <el-icon><Connection /></el-icon>
          <span>业务网络配置</span>
        </template>
        <!-- 任务域分组（FE-13）：任一模块带 category 时按组渲染，未标注归「其他」；
             全部未标注则平铺（等价旧形态）。 -->
        <template v-if="businessGrouped">
          <el-menu-item-group
            v-for="g in businessGroups"
            :key="g.category || '__default__'"
            :title="g.category || '其他'"
          >
            <el-menu-item
              v-for="m in g.modules"
              :key="m.name"
              :index="`/module/${m.name}`"
              :data-test="`module-item-${m.name}`"
            >
              {{ m.title }}
            </el-menu-item>
          </el-menu-item-group>
        </template>
        <template v-else>
          <el-menu-item
            v-for="m in businessModules"
            :key="m.name"
            :index="`/module/${m.name}`"
            :data-test="`module-item-${m.name}`"
          >
            {{ m.title }}
          </el-menu-item>
        </template>
        <el-menu-item index="/config/route">路由配置</el-menu-item>
      </el-sub-menu>

      <el-sub-menu index="native-config" @click="handleNativeMenuClick">
        <template #title>
          <el-icon><Setting /></el-icon>
          <span>原生配置</span>
        </template>
        <el-menu-item
          v-for="model in nativeModels"
          :key="model.name"
          :index="`/native/${model.name}`"
        >
          {{ model.title }}
        </el-menu-item>
        <el-menu-item v-if="!nativeMenuLoaded" index="loading" disabled>
          <el-icon class="is-loading"><Loading /></el-icon>
          加载中...
        </el-menu-item>
      </el-sub-menu>

      <el-menu-item index="/logs">
        <el-icon><Document /></el-icon>
        <template #title>操作日志</template>
      </el-menu-item>

      <el-menu-item index="/settings">
        <el-icon><Tools /></el-icon>
        <template #title>系统设置</template>
      </el-menu-item>
    </el-menu>

    <div v-if="!isCollapsed" class="side-foot">
      <span class="pulse-dot" aria-hidden="true"></span>
      <span>无数据库 · TTL 缓存</span>
    </div>

    <div class="collapse-btn" @click="toggleCollapse">
      <el-icon v-if="isCollapsed"><Expand /></el-icon>
      <el-icon v-else><Fold /></el-icon>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { useMenuStore } from '../../stores/menu'
import { DataLine, Monitor, Connection, Setting, Document, Tools, Loading, Fold, Expand } from '@element-plus/icons-vue'

const route = useRoute()
const menuStore = useMenuStore()

const activeMenu = computed(() => route.path)
const isCollapsed = computed(() => menuStore.isCollapsed)
const nativeModels = computed(() => menuStore.nativeModels)
const nativeMenuLoaded = computed(() => menuStore.nativeMenuLoaded)
const businessModules = computed(() => menuStore.businessModules)
const businessGroups = computed(() => menuStore.businessGroups)
const businessGrouped = computed(() => businessGroups.value.some((g) => g.category))

onMounted(() => {
  menuStore.loadBusinessModules()
})

function toggleCollapse() {
  menuStore.toggleCollapse()
}

function handleNativeMenuClick() {
  if (!nativeMenuLoaded.value) {
    menuStore.loadNativeModels()
  }
}
</script>

<style scoped>
.sidebar {
  width: var(--sidebar-w);
  height: 100vh;
  background: var(--surface);
  border-right: 1px solid var(--line);
  display: flex;
  flex-direction: column;
  transition: width 0.25s var(--ease, cubic-bezier(0.4, 0, 0.2, 1));
}
.sidebar.collapsed {
  width: 64px;
}

/* 品牌区 */
.brand {
  height: var(--topbar-h);
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 0 18px;
  border-bottom: 1px solid var(--line);
  flex-shrink: 0;
}
.brand-mark {
  width: 26px;
  height: 26px;
  border-radius: 6px;
  background: var(--ink);
  position: relative;
  flex-shrink: 0;
  display: grid;
  place-items: center;
}
.brand-mark::before {
  content: '';
  width: 12px;
  height: 12px;
  border: 2px solid var(--brand);
  border-radius: 3px;
  transform: rotate(45deg);
}
.brand-name {
  font-weight: 700;
  letter-spacing: 0.06em;
  font-size: 15px;
  color: var(--ink);
  line-height: 1.1;
}
.brand-sub {
  font-size: 10.5px;
  color: var(--ink-3);
  letter-spacing: 0.12em;
  text-transform: uppercase;
}

/* 导航（覆盖 el-menu 为浅色 iMaster 风格） */
.nav {
  flex: 1;
  border-right: none;
  padding: 10px;
  overflow-y: auto;
  background: transparent;
}
.nav :deep(.el-menu-item),
.nav :deep(.el-sub-menu__title) {
  height: 38px;
  line-height: 38px;
  border-radius: var(--r-ctl);
  margin: 2px 0;
  color: var(--ink-2);
  font-weight: 500;
  font-size: 13.5px;
}
.nav :deep(.el-menu-item .el-icon),
.nav :deep(.el-sub-menu__title .el-icon) {
  color: currentColor;
}
.nav :deep(.el-menu-item:hover),
.nav :deep(.el-sub-menu__title:hover) {
  background: var(--sunken);
  color: var(--ink);
}
.nav :deep(.el-menu-item.is-active) {
  background: var(--primary-weak);
  color: var(--primary-ink);
  font-weight: 600;
}
.nav :deep(.el-sub-menu .el-menu-item) {
  min-width: 0;
  padding-left: 44px !important;
}

/* 侧栏底部状态 */
.side-foot {
  border-top: 1px solid var(--line);
  padding: 12px 16px;
  font-size: 11.5px;
  color: var(--ink-3);
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
}
.pulse-dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: var(--st-conv);
  animation: pulse 2.4s infinite;
}
@keyframes pulse {
  0% { box-shadow: 0 0 0 0 rgba(16, 129, 74, 0.45); }
  70% { box-shadow: 0 0 0 6px rgba(16, 129, 74, 0); }
  100% { box-shadow: 0 0 0 0 rgba(16, 129, 74, 0); }
}

.collapse-btn {
  height: 44px;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  border-top: 1px solid var(--line);
  color: var(--ink-3);
  flex-shrink: 0;
}
.collapse-btn:hover {
  color: var(--primary);
  background: var(--sunken);
}

@media (prefers-reduced-motion: reduce) {
  .pulse-dot { animation: none; }
}
</style>
