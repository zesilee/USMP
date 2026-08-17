// EviewUI 后端 · 命令式反馈（FA-03 / 组 3.5，gate 定案：EviewUI 无命令式 API，
// 自养挂载点）：body 下挂容器 + createRoot 渲染 DivMessage（toast）/
// MessageDialog（confirm，Promise 化）。API 面与 antd 后端 feedback 同形，
// 组 5 接线时由 index.ts 切换导出——业务调用点零改动。
// 行为规格依据（勿凭空改）：component-matrix + gate R1/R2 实测——
// DivMessage type 无 'info'（映 'default'）、其"消失"是 display:none 不卸载
// （toast 由本模块自管卸载）；MessageDialog buttons 仅 ok/cancel、type 含 risk。
import { createRoot } from 'react-dom/client'
import { flushSync } from 'react-dom'
import { createElement, type ReactElement } from 'react'
import DivMessageMod from '@nce/eview-react/DivMessage'
import MessageDialogMod from '@nce/eview-react/MessageDialog'

// EviewUI 编译产物为 babel esModule interop（.default 承载组件）。
function pick(mod: unknown): never {
  return ((mod as { default?: unknown }).default ?? mod) as never
}
const DivMessage = pick(DivMessageMod)
const MessageDialog = pick(MessageDialogMod)

type ToastKind = 'success' | 'error' | 'warning' | 'info'
const KIND_MAP: Record<ToastKind, string> = { success: 'success', error: 'error', warning: 'warn', info: 'default' }

const TOAST_DURATION = 3000

interface EphemeralMount {
  render: (el: ReactElement) => void
  unmount: () => void
}

function mountEphemeral(): EphemeralMount {
  const host = document.createElement('div')
  host.className = 'usmp-feedback-host'
  document.body.appendChild(host)
  const root = createRoot(host)
  let alive = true
  return {
    // 命令式反馈须立即可见：flushSync 同步提交（openinula 若无此 API 则
    // 降级异步渲染，波 C 实测后定）。
    render: (el) => {
      try {
        flushSync(() => root.render(el))
      } catch {
        root.render(el)
      }
    },
    unmount: () => {
      if (!alive) return
      alive = false
      try {
        root.unmount()
      } catch {
        /* 已卸载忽略（R08） */
      }
      host.remove()
    },
  }
}

/** 轻提示（自管生命周期：渲染→3s 卸载，不依赖 DivMessage 内部 dispose）。 */
export function toast(content: string, kind: ToastKind = 'success'): void {
  const m = mountEphemeral()
  m.render(
    createElement(DivMessage, {
      text: content,
      type: KIND_MAP[kind],
      enableDisposeTimeOut: false,
      closeIconDisplay: true,
      onClose: () => m.unmount(),
    }),
  )
  setTimeout(() => m.unmount(), TOAST_DURATION)
}

export interface ConfirmOptions {
  title?: string
  danger?: boolean
  okText?: string
  cancelText?: string
}

/** 确认框：resolve(true)=确认，false=取消/关闭；永不 reject（R08）。 */
export function confirm(content: string, opts: ConfirmOptions = {}): Promise<boolean> {
  return new Promise<boolean>((resolve) => {
    const m = mountEphemeral()
    const done = (v: boolean) => {
      m.unmount()
      resolve(v)
    }
    m.render(
      createElement(MessageDialog, {
        isOpen: true,
        type: opts.danger ? 'risk' : 'confirm',
        title: opts.title ?? content,
        content: opts.title ? content : undefined,
        buttons: {
          ok: { text: opts.okText, onClick: () => done(true) },
          cancel: { text: opts.cancelText, onClick: () => done(false) },
        },
        onClose: () => done(false),
      }),
    )
  })
}
