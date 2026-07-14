import axios from 'axios'
import type { ApiResponse } from '../types/yang'
import type {
  ApiEnvelope,
  DeviceListData,
  DeviceConnStatus,
  FleetReconcileData,
  DeviceReconcileData,
  AuditListData,
} from '../types/api'

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

// Reconcile API —— 车队/单设备对账结局（desired↔actual 收敛），供概览大盘消费。
export const getFleetReconcile = () => {
  return api.get<ApiEnvelope<FleetReconcileData>>('/reconcile/status')
}

export const getDeviceReconcile = (ip: string) => {
  return api.get<ApiEnvelope<DeviceReconcileData>>(`/devices/${ip}/reconcile`)
}

// 操作日志 API —— 配置下发审计 + 当前对账结局（PR-B4，真数据端点）。
export const getLogs = (params: { device?: string; status?: string; limit?: number; offset?: number } = {}) => {
  return api.get<ApiEnvelope<AuditListData>>('/logs', { params })
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

// 行删除（FE-16，命令语义）：key 为条目主键（vlan→id、interface→name），经 query 承载
// （接口名含斜杠，axios params 自动 URL 编码）。
export const deleteConfig = (ip: string, path: string, key: string | number) => {
  const cleanPath = path.startsWith('/') ? path.slice(1) : path
  return api.delete<ApiResponse<any>>(`/config/${ip}/${cleanPath}`, { params: { key } })
}

// Schema API - 获取 YANG 模型定义
export const getSchema = (path: string) => {
  return api.get<ApiResponse<any>>(`/schema/${path}`)
}

// YANG 模块列表（原生配置菜单驱动，FE-13）。必须走 api 客户端（绝对 baseURL）：
// staging 下 nginx 不代理 /api，裸相对 fetch('/api/...') 会命中 SPA fallback 返回
// index.html，res.json() 抛「Unexpected token '<'」。
export const listYangModules = () => {
  return api.get<ApiResponse<any[]>>('/yang/modules')
}

// YANG 模块动态表单 schema。走 api 客户端（绝对 baseURL，staging 下 nginx 不代理 /api，
// 故不能用裸相对 fetch）。form='nested' 返回嵌套树（保留 member-ports 等 list-in-list）。
export const getYangSchema = (module: string, form?: 'nested') => {
  return api.get<ApiResponse<any>>(`/yang/schema/${module}`, {
    params: form ? { form } : {},
  })
}

export default api
