// UI 适配层 · 命令式反馈（FA-03，design D7）：轻提示 toast() 与 Promise 化
// confirm()，可在任意函数（非组件上下文）调用——业务代码保持
// `if (await confirm(...))` 的旧写法，SHALL NOT 各自持有组件库实例。
//
// 实现：UiProvider 挂载时经 __bindFeedback 注入 App.useApp() 实例（带主题与
// locale 上下文）；未挂 Provider（如极早期调用）降级 antd 静态 API（R08 不崩，
// 仅失去主题上下文）。换库时只改本文件。
import { message as staticMessage, Modal as StaticModal } from 'antd'
import type { MessageInstance } from 'antd/es/message/interface'
import type { ModalStaticFunctions } from 'antd/es/modal/confirm'

type ToastKind = 'success' | 'error' | 'warning' | 'info'

let boundMessage: MessageInstance | null = null
let boundModal: Omit<ModalStaticFunctions, 'warn'> | null = null

/** 由 UiProvider 挂载/卸载时调用（适配层内部契约，业务代码勿用）；传 null 解绑。 */
export function __bindFeedback(
  message: MessageInstance | null,
  modal: Omit<ModalStaticFunctions, 'warn'> | null,
): void {
  boundMessage = message
  boundModal = modal
}

/** 轻提示：toast('已下发') / toast('失败', 'error')。 */
export function toast(content: string, kind: ToastKind = 'success'): void {
  const m = boundMessage ?? staticMessage
  m[kind](content)
}

export interface ConfirmOptions {
  /** 标题。缺省时整段 content 充当标题（对齐旧 ElMessageBox 单段形态，
   *  避免硬编码默认标题违反 UI-02 零硬编码）。 */
  title?: string
  /** 危险操作样式（删除/高危 rpc）：确认钮红色。 */
  danger?: boolean
  okText?: string
  cancelText?: string
}

/**
 * 确认框：resolve(true)=确认，resolve(false)=取消/关闭。永不 reject——
 * 调用方统一 `if (await confirm(...))` 分支，无需 try/catch（R08）。
 */
export function confirm(content: string, opts: ConfirmOptions = {}): Promise<boolean> {
  const modal = boundModal ?? StaticModal
  return new Promise<boolean>((resolve) => {
    modal.confirm({
      title: opts.title ?? content,
      content: opts.title ? content : undefined,
      okText: opts.okText,
      cancelText: opts.cancelText,
      okButtonProps: opts.danger ? { danger: true } : undefined,
      onOk: () => resolve(true),
      onCancel: () => resolve(false),
    })
  })
}
