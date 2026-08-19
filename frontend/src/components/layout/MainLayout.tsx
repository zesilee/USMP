import { Outlet, useLocation } from '../../router/compat'
import Sidebar from './Sidebar'
import Header from './Header'
import './MainLayout.scss'

// MainLayout：侧栏 + 顶栏 + 内容区。内容按路由 path 加 key 重建——同一组件被
// 相邻路由复用时（ModuleConsolePage 服务 /module/vlan 与 /module/ifm），否则
// effects 不重跑 → schema 不重载（旧 Vue :key=$route.path 同语义）。
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
