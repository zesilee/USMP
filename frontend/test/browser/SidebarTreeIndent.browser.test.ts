import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { createRouter, createWebHistory } from 'vue-router'
import ElementPlus from 'element-plus'
// 断言的是 el-menu 层级缩进（padding 计算），必须加载 EP 基础样式。
import 'element-plus/dist/index.css'
import Sidebar from '../../src/components/layout/Sidebar.vue'
import { getLeftTree, listYangModules } from '../../src/api'

vi.mock('../../src/api')

// 回归（T07）：左树展开后 huawei-xx 叶子错位——历史固定 `padding-left: 44px
// !important` 把任意深度的叶子钉在 44px，三级叶子比二级分类标题（60px 级距）
// 还浅。修复后叶子缩进必须严格深于其父级标题，且逐级单调递增。
// CSS 计算是 happy-dom 伪造不了的 → F3 真浏览器（§5.6）。

const LEFT_TREE = [
  {
    zh: '接口管理', en: 'Interface Mgmt',
    children: [
      {
        zh: '接口基础', en: 'Interface Base',
        children: [
          {
            zh: 'huawei-ifm', en: 'huawei-ifm', sourceModule: 'huawei-ifm', module: 'ifm', available: true,
            // 模块级子节点（LT-02）：container 与 rpc 平级平铺，树加深到第 5 层。
            children: [
              { zh: '通用接口', en: 'Common Interface', kind: 'container', name: 'ifm' },
              { zh: '重启接口', en: 'Restart interface', kind: 'rpc', name: 'restart-if', highRisk: true },
            ],
          },
        ],
      },
    ],
  },
]

function padLeft(el: Element): number {
  return parseFloat(getComputedStyle(el).paddingLeft)
}

describe('Sidebar 左树层级缩进（F3 真浏览器）', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.mocked(getLeftTree).mockResolvedValue({ data: { data: LEFT_TREE } } as any)
    vi.mocked(listYangModules).mockResolvedValue({ data: { data: [] } } as any)
  })

  it('展开后 huawei-xx 叶子缩进严格深于父级标题，逐级单调递增', async () => {
    const router = createRouter({
      history: createWebHistory(),
      routes: [
        { path: '/', component: { template: '<div />' } },
        { path: '/module/:name', component: { template: '<div />' } },
        { path: '/module/:name/rpc/:rpcName', component: { template: '<div />' } },
      ],
    })
    const wrapper = mount(Sidebar, {
      attachTo: document.body,
      global: { plugins: [router, ElementPlus] },
    })
    await vi.waitFor(() => {
      expect(wrapper.find('[data-test="lefttree-group-接口管理"]').exists()).toBe(true)
    })

    // 逐级展开：原生配置 → 接口管理 → 接口基础
    const titles = () => Array.from(document.querySelectorAll('.el-sub-menu__title'))
    const titleOf = (text: string) => {
      const el = titles().find((t) => t.textContent?.trim() === text)
      expect(el, `sub-menu title「${text}」应存在`).toBeTruthy()
      return el as HTMLElement
    }
    titleOf('原生配置').click()
    await vi.waitFor(() => expect(titleOf('接口管理').offsetParent).not.toBeNull())
    titleOf('接口管理').click()
    await vi.waitFor(() => expect(titleOf('接口基础').offsetParent).not.toBeNull())
    titleOf('接口基础').click()
    // 模块叶现在是可展开分组（LT-03）：继续展开到模块级 container/rpc 子节点。
    await vi.waitFor(() => expect(titleOf('huawei-ifm').offsetParent).not.toBeNull())
    titleOf('huawei-ifm').click()

    const nodeItem = (text: string) =>
      Array.from(document.querySelectorAll('.el-menu-item'))
        .find((i) => i.textContent?.trim().startsWith(text)) as HTMLElement
    const containerNode = await vi.waitFor(() => {
      const el = nodeItem('通用接口')
      expect(el).toBeTruthy()
      expect(el.offsetParent).not.toBeNull()
      return el
    })
    const rpcNode = nodeItem('重启接口')
    expect(rpcNode).toBeTruthy()

    const pRoot = padLeft(titleOf('原生配置'))
    const pCat = padLeft(titleOf('接口管理'))
    const pSub = padLeft(titleOf('接口基础'))
    const pLeaf = padLeft(titleOf('huawei-ifm'))
    const pNode = padLeft(containerNode)
    const pRpc = padLeft(rpcNode)

    // 单调递增：每级都比上一级深（错位 bug 的形态是 pLeaf < pSub）
    expect(pCat, `分类(${pCat}) 应深于根组(${pRoot})`).toBeGreaterThan(pRoot)
    expect(pSub, `子分类(${pSub}) 应深于分类(${pCat})`).toBeGreaterThan(pCat)
    expect(pLeaf, `huawei-xx 叶子(${pLeaf}) 应深于子分类(${pSub})——错位回归点`).toBeGreaterThan(pSub)
    expect(pNode, `container 节点(${pNode}) 应深于模块叶(${pLeaf})`).toBeGreaterThan(pLeaf)
    // container 与 rpc 平级：同深度。
    expect(pRpc, `rpc 节点(${pRpc}) 应与 container 同深`).toBe(pNode)
    // 高危 rpc 警示图标为真实图标节点（R12）。
    expect(rpcNode.querySelector('.rpc-high-risk')).toBeTruthy()

    wrapper.unmount()
  })
})
