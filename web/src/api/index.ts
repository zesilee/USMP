import axios from 'axios'
import { ApiResponse, DeviceInfo, YangModuleInfo, GetConfigResponse } from '../types/yang'

const api = axios.create({
  baseURL: '/api/v1',
  timeout: 15000,
})

// Device API
export const listDevices = () => {
  return api.get<ApiResponse<DeviceInfo[]>>('/devices')
}

export const addDevice = (device: DeviceInfo) => {
  return api.post<ApiResponse<void>>('/devices', device)
}

export const removeDevice = (ip: string) => {
  return api.delete<ApiResponse<void>>(`/devices/${ip}`)
}

export const getDeviceStatus = (ip: string) => {
  return api.get<ApiResponse<{ running: boolean; connected: boolean }>>(`/devices/${ip}/status`)
}

// YANG API
export const listYangModules = () => {
  return api.get<ApiResponse<YangModuleInfo[]>>('/yang/modules')
}

// Config API
export const getConfig = (ip: string, path: string, forceRefresh = false) => {
  return api.get<ApiResponse<GetConfigResponse>>(`/config/${ip}/${path.slice(1)}`, {
    params: { force_refresh: forceRefresh }
  })
}

export const setConfig = (ip: string, path: string, data: any) => {
  return api.post<ApiResponse<void>>(`/config/${ip}/${path.slice(1)}`, data)
}

export default api
