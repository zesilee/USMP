// @nce/icon-plus 宽松声明（真包无 d.ts）：纯具名导出 ESM（E2E 首跑实证：
// 无 default 导出，rolldown 生产构建严格拦默认导入）。命名空间导入取用；
// 实名形态=IconPlus 前缀（校准 R16：2608 图标 22 语义名全命中）。
declare module '@nce/icon-plus' {
  import type { ComponentType } from 'react'
  const icons: Record<string, ComponentType<Record<string, unknown>>>
  export = icons
}
