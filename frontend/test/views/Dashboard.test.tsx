import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter } from 'react-router'
import Dashboard from '../../src/views/Dashboard'
import { UiProvider } from '../../src/ui'
import * as apiModule from '../../src/api'

// Dashboard F2：fleet 概览渲染（收敛率/统计栈/台账/最近对账）+ 失败降级。
vi.mock('../../src/api')

function mount() {
  return render(
    <UiProvider>
      <MemoryRouter>
        <Dashboard />
      </MemoryRouter>
    </UiProvider>,
  )
}

describe('Dashboard · 设备总览', () => {
  beforeEach(() => vi.clearAllMocks())

  it('概览渲染：收敛率/在线数/台账行与最近对账', async () => {
    vi.mocked(apiModule.listDevices).mockResolvedValue({
      data: { success: true, data: { devices: [
        { ip: '10.0.0.1', online: true },
        { ip: '10.0.0.2', online: false },
      ] } },
    } as any)
    vi.mocked(apiModule.getFleetReconcile).mockResolvedValue({
      data: { data: { devices: [{ device_id: '10.0.0.1', outcome: 'converged', last_run: '2026-08-14T10:00:00Z' }] } },
    } as any)
    mount()
    await waitFor(() => expect(document.querySelector('[data-test="convergence-hero"]')).toBeTruthy())
    expect(screen.getByText('50')).toBeInTheDocument() // 1/2 收敛率
    // 离线设备进台账
    expect(await screen.findByText('10.0.0.2')).toBeInTheDocument()
    // 最近对账含已收敛设备
    expect(screen.getAllByText('10.0.0.1').length).toBeGreaterThan(0)
  })

  it('取数失败：错误条+空概览不崩（R08）', async () => {
    vi.mocked(apiModule.listDevices).mockRejectedValue(new Error('fleet down'))
    vi.mocked(apiModule.getFleetReconcile).mockRejectedValue(new Error('fleet down'))
    mount()
    await waitFor(() => expect(document.querySelector('.ant-alert-error')).toBeTruthy())
    expect(document.querySelector('[data-test="convergence-hero"]')).toBeTruthy()
  })
})
