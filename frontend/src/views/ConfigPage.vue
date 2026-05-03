<template>
  <div class="config-page">
    <h1>Config Page: {{ module }}</h1>
  </div>
</template>

<script setup lang="ts">
defineProps<{ module?: string }>()
</script>

<style scoped>
.config-page {
  padding: 20px;
}
</style>
