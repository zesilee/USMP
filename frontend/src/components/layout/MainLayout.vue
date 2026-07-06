<template>
  <div class="main-layout">
    <div class="sidebar-wrapper">
      <Sidebar />
    </div>

    <div class="content-wrapper">
      <div class="header-wrapper">
        <Header />
      </div>

      <main class="main-content">
        <slot />
      </main>
    </div>
  </div>
</template>

<script setup lang="ts">
import Sidebar from './Sidebar.vue'
import Header from './Header.vue'
</script>

<style scoped>
/* 外壳：侧栏 + 内容两栏，内容区独立滚动（浅色 iMaster） */
.main-layout {
  display: flex;
  height: 100vh;
  overflow: hidden;
  background: var(--paper);
}

.sidebar-wrapper {
  flex-shrink: 0;
}

.content-wrapper {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  min-width: 0;
}

.main-content {
  flex: 1;
  padding: 22px;
  overflow-y: auto;
  background: var(--paper);
}
</style>
