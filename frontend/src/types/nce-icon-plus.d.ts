// @nce/icon-plus 宽松声明（真包 d.ts 未随 vendor 采集；具名导出形态由内网
// 校准侦察用例验证——eview/icons.tsx 运行时按候选名自适应取用）。
declare module '@nce/icon-plus' {
  import type { ComponentType } from 'react'
  const icons: Record<string, ComponentType<Record<string, unknown>>>
  export default icons
}
