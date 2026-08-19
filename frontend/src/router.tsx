import { createBrowserRouter } from './router/compat'
import MainLayout from './components/layout/MainLayout'
import ModuleConsolePage from './views/ModuleConsolePage'
import Dashboard from './views/Dashboard'
import Devices from './views/Devices'
import Logs from './views/Logs'
import Settings from './views/Settings'
import BusinessConsolePage from './views/BusinessConsolePage'

// 路由表（FE-13/17/19，对齐旧 vue-router 契约）：legacy 路由一律不存在
// （/native/:module、/config/* 兼容期已结束）。占位页随 tasks 10 组逐页做实。
export const router = createBrowserRouter([
  {
    path: '/',
    element: <MainLayout />,
    children: [
      { index: true, element: <Dashboard /> },
      { path: 'devices', element: <Devices /> },
      // 业务网络配置控制台（FE-17）：平台作用域，与设备作用域 /module/:module 并列。
      { path: 'business/:module', element: <BusinessConsolePage /> },
      // 通用模块控制台（FE-10）：零 per-module props，Tab/列/表单全部由 schema 派生。
      { path: 'module/:module', element: <ModuleConsolePage /> },
      // rpc 直达（FE-19）：左树 rpc 节点入口，仅渲染该 rpc 执行面板。
      { path: 'module/:module/rpc/:rpcName', element: <ModuleConsolePage /> },
      { path: 'logs', element: <Logs /> },
      { path: 'settings', element: <Settings /> },
    ],
  },
])
