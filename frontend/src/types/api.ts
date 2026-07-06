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

// 对账（desired↔actual 收敛）契约（PR-B1 端点 + PR-B3 注解）。
export type ReconcileOutcome = NonNullable<Schemas['status.Outcome']>
export type ReconcileStatus =
  Schemas['github_com_leezesi_usmp_backend_pkg_yang-runtime_status.Status']
export type DeviceRollup = Schemas['api.DeviceRollup']
export type FleetReconcileData = Schemas['api.FleetReconcileData']
export type DeviceReconcileData = Schemas['api.DeviceReconcileData']

// 操作日志（配置下发审计 + 当前对账结局 live-join）契约（PR-B4 端点）。
export type AuditListData = Schemas['api.AuditListData']
export type LogEntry = Schemas['api.LogEntry']
