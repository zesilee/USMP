// 契约类型出口。
//
// 从 ./api.gen.ts（生成物，勿手改）抽取常用类型别名。api.gen.ts 由后端 OpenAPI
// 规格生成（swag 注解 → openapi3 → openapi-typescript），后端是唯一真源。
// 端点/字段写错（如 res.data.devices 而非 res.data.data.devices）会在此类型下变编译错误。
import type { components } from './api.gen'

type Schemas = components['schemas']

// 统一响应信封 {code,message,data,success}；data 的形状由各端点后端 DTO 决定。
export interface ApiEnvelope<T> {
  code?: number
  message?: string
  success?: boolean
  data?: T
}

export type DeviceStatusDTO = Schemas['api.DeviceStatus']
export type DeviceListData = Schemas['api.DeviceListData']
export type DeviceConnStatus = Schemas['api.DeviceConnStatus']
