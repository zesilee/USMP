// 路由兼容层（波 C 2.4 收口，2026-08-19）：业务与装配代码 SHALL 只从本文件
// 导入路由 API，SHALL NOT 直接 import 'react-router'（守护测试拦截）。
// 现阶段=react-router 直通 re-export；波 C 翻转时本文件内部换 inula-router
// （v5 API 形态）：useNavigate/useSearchParams 以 history/location 薄包装、
// useBlocker 以 Prompt + getUserConfirmation 桥实现（blocker.state/proceed/
// reset 契约保持——RT-03 Scenario 钉住）、createBrowserRouter/RouterProvider
// 以 BrowserRouter + Switch/Route JSX 树组合表达。调用点零改动是本层的存在
// 意义——翻转改动被收束在此单点。
export {
  createBrowserRouter,
  RouterProvider,
  Outlet,
  useLocation,
  useNavigate,
  useParams,
  useSearchParams,
  useBlocker,
} from 'react-router'
