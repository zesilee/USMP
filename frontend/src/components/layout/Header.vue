<template>
  <div class="header">
    <div class="header-left">
      <el-input
        v-model="searchText"
        placeholder="搜索设备、配置..."
        style="width: 300px"
      >
        <template #prefix>
          <el-icon><Search /></el-icon>
        </template>
      </el-input>
    </div>

    <div class="header-right">
      <div class="device-status">
        <span class="status-dot online"></span>
        <span>{{ onlineCount }}/{{ totalCount }} 在线</span>
      </div>

      <el-badge :value="notificationCount" :hidden="notificationCount === 0" class="notification-icon">
        <el-button circle><el-icon><Bell /></el-icon></el-button>
      </el-badge>

      <div class="user-area">
        <el-avatar :size="32">Admin</el-avatar>
      </div>

      <el-dropdown>
        <el-button circle><el-icon><MoreFilled /></el-icon></el-button>
        <template #dropdown>
          <el-dropdown-menu>
            <el-dropdown-item>切换主题</el-dropdown-item>
            <el-dropdown-item>系统设置</el-dropdown-item>
            <el-dropdown-item divided>退出登录</el-dropdown-item>
          </el-dropdown-menu>
        </template>
      </el-dropdown>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { Bell, MoreFilled, Search } from '@element-plus/icons-vue'

const searchText = ref('')
const onlineCount = ref(10)
const totalCount = ref(12)
const notificationCount = ref(3)
</script>

<style scoped>
.header {
  height: 64px;
  background: #fff;
  border-bottom: 1px solid #e5e7eb;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 24px;
}

.header-right {
  display: flex;
  align-items: center;
  gap: 16px;
}

.device-status {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;
  color: #606266;
}

.status-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
}

.status-dot.online {
  background: #67c23a;
}

.notification-icon {
  cursor: pointer;
}

.user-area {
  cursor: pointer;
}
</style>
