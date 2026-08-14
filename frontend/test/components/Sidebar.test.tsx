import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { MemoryRouter } from 'react-router'
import Sidebar from '../../src/components/layout/Sidebar'
import Header from '../../src/components/layout/Header'
import { UiProvider } from '../../src/ui'
import { useMenuStore } from '../../src/stores/menu'
import { useLocaleStore } from '../../src/stores/locale'
import * as apiModule from '../../src/api'

// Sidebar/Header F2（LT-03/05 + UI-01）：左树 items 派生（分组/叶/模块级
// children/rpc/禁用）、搜索过滤、business 分桶、语言切换入口。
vi.mock('../../src/api')

const tree = [
  {
    zh: '以太网交换', en: 'Ethernet Switching',
    children: [
      {
        zh: 'VLAN', en: 'VLAN',
        children: [
          {
            zh: 'huawei-vlan', en: 'huawei-vlan', sourceModule: 'huawei-vlan', available: true, module: 'vlan',
            children: [
              { zh: 'VLAN配置', en: 'VLAN Config', kind: 'container' as const, name: 'vlan' },
              { zh: '重启VLAN', en: 'Restart VLAN', kind: 'rpc' as const, name: 'restart-vlan', highRisk: true },
            ],
          },
          { zh: 'huawei-evpl', en: 'huawei-evpl', sourceModule: 'huawei-evpl', available: false },
        ],
      },
    ],
  },
]

function mount() {
  return render(
    <UiProvider>
      <MemoryRouter initialEntries={['/']}>
        <Sidebar />
        <Header />
      </MemoryRouter>
    </UiProvider>,
  )
}

describe('Sidebar · SND 左树导航（LT-03）', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    localStorage.clear()
    useLocaleStore.getState().__resetForTest()
    useMenuStore.setState({
      leftTree: tree as any, leftTreeLoaded: true,
      nativeModules: [
        { name: 'vlan', title: 'VLAN', vendor: 'huawei' },
        { name: 'bvs', title: '业务VLAN', vendor: 'usmp', category: 'business-network' },
      ] as any,
      nativeLoaded: true, isCollapsed: false,
    })
    vi.mocked(apiModule.getLeftTree).mockResolvedValue({ data: { data: tree } } as any)
    vi.mocked(apiModule.listYangModules).mockResolvedValue({ data: { data: [] } } as any)
  })

  it('左树分组→子树、叶模块级 children（container/rpc）、未接入禁用', async () => {
    mount()
    fireEvent.click(await screen.findByText('以太网交换'))
    fireEvent.click(await screen.findByText('VLAN', { selector: '.ant-menu-submenu-title *' }))
    fireEvent.click(await screen.findByText('huawei-vlan'))
    expect(await screen.findByText('VLAN配置')).toBeInTheDocument()
    expect(screen.getByText(/重启VLAN/)).toBeInTheDocument()
    // 未接入叶禁用
    const disabled = screen.getByText('huawei-evpl').closest('li')
    expect(disabled?.className).toContain('disabled')
  })

  it('搜索过滤：命中子树保留、不命中提示（LT-05）', async () => {
    mount()
    const input = document.querySelector('[data-test="lefttree-search"] input, input[data-test="lefttree-search"]') as HTMLInputElement
    fireEvent.change(input, { target: { value: 'VLAN' } })
    expect(screen.queryByText(/no-match/)).toBeNull()
    fireEvent.change(input, { target: { value: 'zzz-none' } })
    await waitFor(() => expect(document.querySelector('[data-test="lefttree-no-match"]')).toBeTruthy())
  })

  it('业务模块自动出业务组（FE-17 category 分桶零硬编码）', async () => {
    mount()
    expect(document.querySelector('[data-test="sidebar"]')!.textContent).toContain('业务')
  })

  it('语言切换（UI-01）：Header 入口切 en-us，左树标签换英文', async () => {
    mount()
    fireEvent.click(document.querySelector('[data-test="locale-switch"]')!)
    fireEvent.click(await screen.findByText('English'))
    await waitFor(() => expect(screen.getByText('Ethernet Switching')).toBeInTheDocument())
    expect(localStorage.getItem('usmp-locale')).toBe('en-us')
    // 收尾切回，避免污染同文件后续用例
    fireEvent.click(document.querySelector('[data-test="locale-switch"]')!)
    fireEvent.click(await screen.findByText('中文'))
  })
})
