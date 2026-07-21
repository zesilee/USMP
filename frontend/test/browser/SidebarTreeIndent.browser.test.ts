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
          { zh: 'huawei-ifm', en: 'huawei-ifm', sourceModule: 'huawei-ifm', module: 'ifm', available: true },
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

    const leaf = await vi.waitFor(() => {
      const el = Array.from(document.querySelectorAll('.el-menu-item'))
        .find((i) => i.textContent?.trim() === 'huawei-ifm') as HTMLElement
      expect(el).toBeTruthy()
      expect(el.offsetParent).not.toBeNull()
      return el
    })

    const pRoot = padLeft(titleOf('原生配置'))
    const pCat = padLeft(titleOf('接口管理'))
    const pSub = padLeft(titleOf('接口基础'))
    const pLeaf = padLeft(leaf)

    // 单调递增：每级都比上一级深（错位 bug 的形态是 pLeaf < pSub）
    expect(pCat, `分类(${pCat}) 应深于根组(${pRoot})`).toBeGreaterThan(pRoot)
    expect(pSub, `子分类(${pSub}) 应深于分类(${pCat})`).toBeGreaterThan(pCat)
    expect(pLeaf, `huawei-xx 叶子(${pLeaf}) 应深于子分类(${pSub})——错位回归点`).toBeGreaterThan(pSub)

    wrapper.unmount()
  })
})
