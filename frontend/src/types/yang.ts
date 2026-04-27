// DeviceInfo 设备信息
export interface DeviceInfo {
  ip: string
  port: number
  username: string
  password: string
}

// YangModuleInfo YANG模块信息
export interface YangModuleInfo {
  name: string
  path: string
  description: string
  type: string
}

// ApiResponse 通用API响应
export interface ApiResponse<T> {
  code: number
  message: string
  data?: T
  success: boolean
}

// GetConfigResponse 获取配置响应
export interface GetConfigResponse {
  data: any
  from_cache: boolean
}
