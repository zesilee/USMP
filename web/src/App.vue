<template>
  <div id="app">
    <el-container>
      <el-header>
        <h1>交换机设备管理平台</h1>
      </el-header>
      <el-container>
        <el-aside width="250px">
          <DeviceTree
            :devices="devices"
            @node-selected="onNodeSelected"
          />
        </el-aside>
        <el-main>
          <div v-if="currentDevice && currentYangPath" class="config-form">
            <h2>
              {{ currentDevice.ip }} - {{ currentYangName }}
            </h2>
            <DynamicForm
              :device-ip="currentDevice.ip"
              :yang-path="currentYangPath"
              :key="formKey"
            />
          </div>
          <div v-else class="empty-state">
            <el-empty description="请从左侧选择设备和配置节点"></el-empty>
          </div>
        </el-main>
      </el-container>
    </el-container>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import DeviceTree from './components/DeviceTree.vue'
import DynamicForm from './components/DynamicForm.vue'
import { DeviceInfo } from './types/yang'

const devices = ref<DeviceInfo[]>([])
const currentDevice = ref<DeviceInfo | null>(null)
const currentYangPath = ref<string>('')
const currentYangName = ref<string>('')
const formKey = ref(0)

const onNodeSelected = (device: DeviceInfo, yangPath: string, name: string) => {
  currentDevice.value = device
  currentYangPath.value = yangPath
  currentYangName.value = name
  formKey.value++
}
</script>

<style>
#app {
  height: 100vh;
}

el-header {
  background-color: #409eff;
  color: white;
  line-height: 60px;
}

el-header h1 {
  margin: 0;
  font-size: 20px;
}

.el-aside {
  background-color: #f5f7fa;
  border-right: 1px solid #dcdfe6;
}

.el-main {
  background-color: #ffffff;
  min-height: calc(100vh - 60px);
}

.config-form {
  max-width: 900px;
  margin: 0 auto;
}

.empty-state {
  display: flex;
  justify-content: center;
  align-items: center;
  height: 400px;
}
</style>
