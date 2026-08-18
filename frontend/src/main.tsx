// 守卫必须第一行（生产 bundle 模块序：晚装会被库模块初始化绕过——E2E 实录）。
import './runtime/install-guards'
import { createRoot } from 'react-dom/client'
import { RouterProvider } from 'react-router'
import { UiProvider } from './ui'
import { router } from './router'
import './styles/reset.scss'
import './styles/theme.scss'

// StrictMode 暂摘（组 7 E2E 诊断期排除变量；恢复评估挂组 8.1）。
// onRecoverableError：可恢复错误循环曾压崩页面（#520/#185，E2E 实录）——
// 保留钩子让此类问题在控制台自曝组件栈，不再静默循环。
createRoot(document.getElementById('app')!, {
  onRecoverableError: (err, info) => {
    const e = err as Error & { cause?: unknown }
    console.warn(
      '[RECOVERABLE]',
      String(e).slice(0, 150),
      '| cause=',
      e?.cause ? String(e.cause).slice(0, 300) : 'none',
      '| stack=',
      (info?.componentStack ?? '').split('\n').slice(0, 8).join(' <- '),
    )
  },
}).render(
  <UiProvider>
    <RouterProvider router={router} />
  </UiProvider>,
)
