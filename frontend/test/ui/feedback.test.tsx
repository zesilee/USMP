import { describe, it, expect, vi } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { UiProvider } from '../../src/ui'
import { toast, confirm } from '../../src/ui/feedback'

// FA-03（design D7）：命令式反馈的 Promise 语义与非组件上下文可调用性。
// 挂 UiProvider 后 toast/confirm 走带主题上下文的实例；确认/取消分别 resolve
// true/false，永不 reject。
function mount() {
  return render(
    <UiProvider>
      <div data-testid="host" />
    </UiProvider>,
  )
}

describe('feedback 适配（FA-03）', () => {
  it('未挂 UiProvider 时降级静态 API 不崩（R08 负路径）', async () => {
    // 干净模块副本：绕开其它用例已完成的 __bindFeedback 绑定。
    vi.resetModules()
    const fresh = await import('../../src/ui/feedback')
    expect(() => fresh.toast('降级提示', 'info')).not.toThrow()
    const p = fresh.confirm('降级确认？')
    const cancel = await screen.findByRole('button', { name: /Cancel|取\s*消/ })
    await userEvent.click(cancel)
    await expect(p).resolves.toBe(false)
  })

  it('toast 在普通函数中调用即弹出且不抛错（R08）', async () => {
    mount()
    expect(() => toast('已下发')).not.toThrow()
    await waitFor(() => expect(document.body.textContent).toContain('已下发'))
  })

  it('confirm 点确认 resolve(true)', async () => {
    mount()
    const p = confirm('确认删除该行？', { danger: true })
    const ok = await screen.findByRole('button', { name: /OK|确\s*定/ })
    await userEvent.click(ok)
    await expect(p).resolves.toBe(true)
  })

  it('confirm 点取消 resolve(false)（负路径，永不 reject）', async () => {
    mount()
    const p = confirm('确认下发？')
    const cancel = await screen.findByRole('button', { name: /Cancel|取\s*消/ })
    await userEvent.click(cancel)
    await expect(p).resolves.toBe(false)
  })
})
