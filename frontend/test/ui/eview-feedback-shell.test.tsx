import { describe, it, expect, vi, afterEach } from 'vitest'
import { render, screen, fireEvent, waitFor, cleanup } from '@testing-library/react'
import FormItemShell from '@bridge/FormItemShell'

// EviewUI 组件替身（混合模式：实现仅内网，F2 以 d.ts+gate 实测行为为规格）。
// DivMessage：渲染 text；onClose 经关闭钮触发。MessageDialog：渲染 title/
// content 与 ok/cancel 两钮（buttons 仅 ok/cancel，gate 定案）。
vi.mock('@nce/eview-react/DivMessage', () => ({
  default: (p: any) => (
    <div data-stub="divmessage" data-type={p.type}>
      <span>{p.text}</span>
      {p.closeIconDisplay && <button onClick={p.onClose}>x</button>}
    </div>
  ),
}))
vi.mock('@nce/eview-react/MessageDialog', () => ({
  default: (p: any) =>
    p.isOpen ? (
      <div data-stub="messagedialog" data-type={p.type}>
        <b>{p.title}</b>
        {p.content && <p>{p.content}</p>}
        <button onClick={p.buttons?.ok?.onClick}>{p.buttons?.ok?.text ?? 'OK'}</button>
        <button onClick={p.buttons?.cancel?.onClick}>{p.buttons?.cancel?.text ?? 'CANCEL'}</button>
        <button aria-label="close" onClick={p.onClose}>×</button>
      </div>
    ) : null,
}))

import { toast, confirm } from '@bridge/feedback'

afterEach(() => {
  cleanup()
  document.body.innerHTML = ''
  vi.useRealTimers()
})

describe('eview feedback · toast（自养挂载点）', () => {
  it('渲染到 body、info 映射 default、3s 自动卸载', async () => {
    vi.useFakeTimers()
    toast('已下发', 'info')
    const host = document.body.querySelector('.usmp-feedback-host')!
    expect(host).toBeTruthy()
    expect(host.textContent).toContain('已下发')
    expect(host.querySelector('[data-type="default"]')).toBeTruthy()
    vi.advanceTimersByTime(3100)
    expect(document.body.querySelector('.usmp-feedback-host')).toBeNull()
  })

  it('点关闭钮即时卸载（不等 3s）', async () => {
    toast('msg', 'error')
    const host = document.body.querySelector('.usmp-feedback-host')!
    expect(host.querySelector('[data-type="error"]')).toBeTruthy()
    fireEvent.click(host.querySelector('button')!)
    await waitFor(() => expect(document.body.querySelector('.usmp-feedback-host')).toBeNull())
  })
})

describe('eview feedback · confirm（Promise 化）', () => {
  it('确认 resolve(true) 并卸载', async () => {
    const p = confirm('删除该行？', { okText: '删除', danger: true })
    const dlg = document.body.querySelector('[data-stub="messagedialog"]')!
    expect(dlg.getAttribute('data-type')).toBe('risk')
    expect(dlg.textContent).toContain('删除该行？')
    fireEvent.click(screen.getByText('删除'))
    await expect(p).resolves.toBe(true)
    expect(document.body.querySelector('.usmp-feedback-host')).toBeNull()
  })

  it('取消与右上关闭均 resolve(false)', async () => {
    const p1 = confirm('确认？')
    fireEvent.click(screen.getByText('CANCEL'))
    await expect(p1).resolves.toBe(false)
    const p2 = confirm('确认？', { title: '标题', okText: 'y', cancelText: 'n' })
    const dlg = document.body.querySelector('[data-stub="messagedialog"]')!
    expect(dlg.querySelector('b')!.textContent).toBe('标题')
    expect(dlg.querySelector('p')!.textContent).toBe('确认？')
    fireEvent.click(screen.getByLabelText('close'))
    await expect(p2).resolves.toBe(false)
  })
})

describe('FormItemShell（FA-06 受控错误态）', () => {
  it('label/必填星/data-test 锚点渲染', () => {
    const { container } = render(
      <FormItemShell label="VLAN 标识" required data-test="fi-id">
        <input />
      </FormItemShell>,
    )
    expect(screen.getByText('VLAN 标识')).toBeInTheDocument()
    expect(container.querySelector('.fis-required')).toBeTruthy()
    expect(container.querySelector('[data-test="fi-id"]')).toBeTruthy()
  })

  it('error 受控：出现即错误态+role=alert，清空即消除', () => {
    const { container, rerender } = render(
      <FormItemShell label="l" error="必填项">
        <input />
      </FormItemShell>,
    )
    expect(container.querySelector('.fis-error')).toBeTruthy()
    expect(screen.getByRole('alert').textContent).toBe('必填项')
    rerender(
      <FormItemShell label="l">
        <input />
      </FormItemShell>,
    )
    expect(container.querySelector('.fis-error')).toBeNull()
    expect(container.querySelector('.fis-error-msg')).toBeNull()
  })

  it('无 label 时不渲染 label 结构；children 恒在控制区', () => {
    const { container } = render(
      <FormItemShell>
        <input data-test="raw" />
      </FormItemShell>,
    )
    expect(container.querySelector('.fis-label')).toBeNull()
    expect(container.querySelector('.fis-control [data-test="raw"]')).toBeTruthy()
  })
})
