import { Outlet, useLocation } from '@app-router'
import Sidebar from './Sidebar'
import Header from './Header'
import './MainLayout.scss'

// MainLayout：侧栏 + 顶栏 + 内容区。内容区按路由 path 加 key 强制重建——相邻
// 路由复用同一组件时（ModuleConsolePage 同时服务 /module/vlan 与 /module/ifm），
// 不换 key 则 effects 不重跑、schema 不重载（旧 Vue :key=$route.path 同语义）。
export default function MainLayout() {
  const location = useLocation()
  return (
    <div className="main-layout">
      <Sidebar />
      <div className="main-body">
        <Header />
        <main className="main-content">
          <Outlet key={location.pathname} />
        </main>
      </div>
    </div>
  )
}
