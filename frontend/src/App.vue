<template>
  <MainLayout>
    <!-- :key 按路由路径重建组件：同一组件被相邻路由复用时（如 DeviceConfigPage 同时服务
         /config/vlan 与 /config/interface），否则 setup/onMounted 不重跑 → schema 不重载。 -->
    <router-view :key="$route.path" />
  </MainLayout>
</template>

<script setup lang="ts">
import MainLayout from './components/layout/MainLayout.vue'
</script>

<style>
* {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}

html, body, #app {
  height: 100%;
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
}
</style>
