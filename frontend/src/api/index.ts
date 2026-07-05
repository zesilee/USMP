import axios from 'axios'
import type { ApiResponse } from '../types/yang'
import type { ApiEnvelope, DeviceListData, DeviceConnStatus } from '../types/api'

// API 基础配置
const API_BASE = import.meta.env.VITE_API_URL || 'http://localhost:8080/api/v1'

const api = axios.create({
  baseURL: API_BASE,
  timeout: 15000,
})

// Device API —— 响应类型由后端 OpenAPI 契约生成（见 types/api.ts），
// 写错字段（res.data.devices）会编译报错。
export const listDevices = () => {
  return api.get<ApiEnvelope<DeviceListData>>('/devices')
}

export const getDeviceStatus = (ip: string) => {
  return api.get<ApiEnvelope<DeviceConnStatus>>(`/devices/${ip}/status`)
}

// Config API - 通用 YANG 配置接口
export const getConfig = (ip: string, path: string, forceRefresh = false) => {
  // 移除 path 开头的斜杠
  const cleanPath = path.startsWith('/') ? path.slice(1) : path
  return api.get<ApiResponse<any>>(`/config/${ip}/${cleanPath}`, {
    params: { force_refresh: forceRefresh }
  })
}

export const setConfig = (ip: string, path: string, data: any) => {
  const cleanPath = path.startsWith('/') ? path.slice(1) : path
  return api.post<ApiResponse<void>>(`/config/${ip}/${cleanPath}`, data)
}

// Schema API - 获取 YANG 模型定义
export const getSchema = (path: string) => {
  return api.get<ApiResponse<any>>(`/schema/${path}`)
}

export default api
