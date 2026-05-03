<template>
  <div class="sidebar" :class="{ collapsed: isCollapsed }">
    <div class="logo-area">
      <span v-if="!isCollapsed">USMP</span>
      <span v-else>U</span>
    </div>

    <el-menu
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

      <el-sub-menu index="business-config">
        <template #title>
          <el-icon><Connection /></el-icon>
          <span>业务网络配置</span>
        </template>
        <el-menu-item index="/config/interface">接口配置</el-menu-item>
        <el-menu-item index="/config/vlan">VLAN配置</el-menu-item>
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

    <div class="collapse-btn" @click="toggleCollapse">
      <el-icon v-if="isCollapsed"><Expand /></el-icon>
      <el-icon v-else><Fold /></el-icon>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { useMenuStore } from '../../stores/menu'
import { DataLine, Monitor, Connection, Setting, Document, Tools, Loading, Fold, Expand } from '@element-plus/icons-vue'

const route = useRoute()
const menuStore = useMenuStore()

const activeMenu = computed(() => route.path)
const isCollapsed = computed(() => menuStore.isCollapsed)
const nativeModels = computed(() => menuStore.nativeModels)
const nativeMenuLoaded = computed(() => menuStore.nativeMenuLoaded)

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
  width: 240px;
  height: 100vh;
  background: #fff;
  border-right: 1px solid #e5e7eb;
  display: flex;
  flex-direction: column;
  transition: width 0.3s;
}

.sidebar.collapsed {
  width: 64px;
}

.logo-area {
  height: 64px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 20px;
  font-weight: bold;
  color: #409eff;
  border-bottom: 1px solid #e5e7eb;
}

.el-menu {
  flex: 1;
  border-right: none;
}

.collapse-btn {
  height: 48px;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  border-top: 1px solid #e5e7eb;
  color: #909399;
}

.collapse-btn:hover {
  color: #409eff;
}
</style>
