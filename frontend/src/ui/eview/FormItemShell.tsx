// FormItemShell（FA-06 / 组 3.6）：表单项外壳——label + 必填星 + 受控错误态。
// 适配层自有实现，SHALL NOT 使用组件库表单容器的内部 store 与校验器
// （校验权威在自研约束引擎）；错误文案由外部受控传入（error prop），
// 与 antd 后端时代的 validateStatus/help 受控语义对等。
// 结构与类名对齐 SchemaForm 栅格样式（组 5 接线已顶替 antd Form.Item）。
import type { ReactNode } from 'react'

export interface FormItemShellProps {
  label?: ReactNode
  /** 必填星（仅展示，不产生任何校验行为）。 */
  required?: boolean
  /** 受控错误文案：非空即错误态（红框由 .fis-error 类联动子控件样式）。 */
  error?: string
  /** label 右侧扩展位（字段级清除钮等）。 */
  labelExtra?: ReactNode
  /** 测试锚点（FA-05：wrapper 直接承载 data-test）。 */
  'data-test'?: string
  /** 布局类透传（SchemaForm 的 fi-span-full 栅格控制）。 */
  className?: string
  children?: ReactNode
}

export default function FormItemShell(props: FormItemShellProps) {
  const { label, required, error, labelExtra, children } = props
  const cls = ['form-item-shell', error ? 'fis-error' : '', props.className ?? ''].filter(Boolean).join(' ')
  return (
    <div className={cls} data-test={props['data-test']}>
      {(label != null || labelExtra != null) && (
        <label className="fis-label">
          <span className="fis-label-text">
            {required && (
              <span className="fis-required" aria-hidden="true">
                *
              </span>
            )}
            {label}
          </span>
          {labelExtra}
        </label>
      )}
      <div className="fis-control">{children}</div>
      {error ? (
        <div className="fis-error-msg" role="alert">
          {error}
        </div>
      ) : null}
    </div>
  )
}
