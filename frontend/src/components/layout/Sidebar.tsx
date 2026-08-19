import { useEffect, useMemo, useState } from 'react'
import { useLocation, useNavigate } from '@app-router'
import { Input, Menu, icons } from '../../ui'
import { i18n, useLocale } from '../../i18n'
import { useMenuStore, type LeftTreeNode } from '../../stores/menu'
import { filterLeftTree } from '../../utils/leftTreeFilter'
import './Sidebar.scss'

// Sidebar（LT-03/FE-13/FE-17）：SND 左树驱动 14 组/3 层导航（container→
// /module/:m、rpc→/module/:m/rpc/:name、未接入 available=false 禁用占位）；
// left-tree 失败回退任务域分组（R08 导航不消失）；业务模块 category 分桶自动
// 出业务组；节点名搜索（LT-05，双语命中）。antd Menu items 配置化承接。
const t = (k: string, p?: Record<string, unknown>) => i18n.global.t(k, p)

type MenuItem = Required<Parameters<typeof Menu>[0]>['items'][number]

function nodeLabel(n: LeftTreeNode, locale: string): string {
  return (locale === 'en-us' ? n.en : n.zh) || n.zh || n.en
}

// 左树 → antd Menu items（LT-02/03）：分组递归子树；叶（sourceModule）带模块级
// children（container/rpc 平铺）；available=false 禁用。
// data-test 命名沿用旧 LeftTreeMenu 契约（E2E 选择器基准）：group=zh、
// leaf=sourceModule、node=name（container）、rpc=<module>-<name>。
function treeToItems(nodes: LeftTreeNode[], locale: string, prefix: string): MenuItem[] {
  return nodes.map((n, i) => {
    const key = `${prefix}-${i}`
    const label = nodeLabel(n, locale)
    if (n.children?.length && !n.sourceModule) {
      return {
        key,
        label: <span data-test={`lefttree-group-${n.zh}`}>{label}</span>,
        children: treeToItems(n.children, locale, key),
      }
    }
    if (n.sourceModule) {
      if (!n.available) {
        return { key, label: <span data-test={`lefttree-leaf-${n.sourceModule}`}>{label}</span>, disabled: true }
      }
      const kids = (n.children ?? []).map((c) => ({
        key:
          c.kind === 'rpc'
            ? `/module/${n.module}/rpc/${c.name}`
            : `/module/${c.name || n.module}`,
        // 高危 rpc 真实图标标记（R12 禁 emoji）。
        label:
          c.kind === 'rpc' ? (
            <span data-test={`lefttree-rpc-${n.module}-${c.name}`} className={c.highRisk ? 'lt-rpc-highrisk' : undefined}>
              {nodeLabel(c, locale)} {c.highRisk && <icons.WarningFilledIcon />}
            </span>
          ) : (
            <span data-test={`lefttree-node-${c.name || n.module}`}>{nodeLabel(c, locale)}</span>
          ),
      }))
      if (kids.length) {
        return {
          key,
          label: <span data-test={`lefttree-leaf-${n.sourceModule}`}>{label}</span>,
          children: kids,
        }
      }
      return { key: `/module/${n.module}`, label: <span data-test={`lefttree-leaf-${n.sourceModule}`}>{label}</span> }
    }
    return { key, label }
  })
}

export default function Sidebar() {
  const navigate = useNavigate()
  const location = useLocation()
  const locale = useLocale()
  const isCollapsed = useMenuStore((s) => s.isCollapsed)
  const leftTree = useMenuStore((s) => s.leftTree)
  const nativeModules = useMenuStore((s) => s.nativeModules)
  const loadLeftTree = useMenuStore((s) => s.loadLeftTree)
  const loadNativeModules = useMenuStore((s) => s.loadNativeModules)

  useEffect(() => {
    void loadLeftTree()
    void loadNativeModules()
  }, [loadLeftTree, loadNativeModules])

  const [ltQuery, setLtQuery] = useState('')
  const filteredTree = useMemo(() => filterLeftTree(leftTree, ltQuery), [leftTree, ltQuery])
  const noMatch = ltQuery.trim() !== '' && filteredTree.length === 0

  const menu = useMenuStore()
  const businessModules = menu.businessModules()
  const nativeGroups = menu.nativeGroups()
  const nativeGrouped = nativeGroups.some((g) => g.category)

  const nativeChildren: MenuItem[] = leftTree.length
    ? treeToItems(filteredTree, locale, 'lt')
    : nativeGrouped
      ? nativeGroups.map((g) => ({
          key: `grp-${g.category || '__default__'}`,
          type: 'group' as const,
          label: g.category || t('nav.otherGroup'),
          children: g.modules.map((m) => ({ key: `/module/${m.name}`, label: m.title })),
        }))
      : nativeModules.map((m) => ({ key: `/module/${m.name}`, label: m.title }))

  const items: MenuItem[] = [
    { key: '/', icon: <icons.DataLineIcon />, label: t('nav.overview') },
    { key: '/devices', icon: <icons.MonitorIcon />, label: t('nav.devices') },
    {
      key: 'native-config',
      icon: <icons.ConnectionIcon />,
      label: t('nav.nativeConfig'),
      children: nativeChildren,
    },
    ...(businessModules.length
      ? [
          {
            key: 'business-config',
            icon: <icons.ShareIcon />,
            label: <span data-test="business-group">{t('nav.businessConfig')}</span>,
            children: businessModules.map((m) => ({
              key: `/business/${m.name}`,
              label: <span data-test={`business-item-${m.name}`}>{m.title}</span>,
            })),
          } as MenuItem,
        ]
      : []),
    { key: '/logs', icon: <icons.DocumentIcon />, label: t('nav.logs') },
    { key: '/settings', icon: <icons.SettingIcon />, label: t('nav.settings') },
  ]

  const [openKeys, setOpenKeys] = useState<string[]>(['native-config'])
  // 搜索时自动展开命中路径（LT-05 简化形态：展开全部分组层）。
  useEffect(() => {
    if (!ltQuery.trim()) return
    const keys: string[] = ['native-config']
    const walk = (nodes: LeftTreeNode[], prefix: string) => {
      nodes.forEach((n, i) => {
        const key = `${prefix}-${i}`
        if (n.children?.length && !n.sourceModule) {
          keys.push(key)
          walk(n.children, key)
        } else if (n.sourceModule && n.children?.length) {
          keys.push(key)
        }
      })
    }
    walk(filteredTree, 'lt')
    setOpenKeys(keys)
  }, [ltQuery, filteredTree])

  return (
    <div className={`sidebar${isCollapsed ? ' collapsed' : ''}`} data-test="sidebar">
      <div className="brand">
        <div className="brand-mark" aria-hidden="true" />
        {!isCollapsed && (
          <div className="brand-text">
            <div className="brand-name">USMP</div>
            <div className="brand-sub">Switch Mgmt</div>
          </div>
        )}
      </div>

      <div className="lt-toolbar">
        <Input
          size="small"
          allowClear
          data-test="lefttree-search"
          placeholder={t('nav.treeSearchPlaceholder')}
          prefix={<icons.SearchIcon />}
          value={ltQuery}
          onChange={(e) => setLtQuery(e.target.value)}
        />
      </div>
      {noMatch && (
        <div className="lt-empty" data-test="lefttree-no-match">
          {t('nav.searchNoMatch')}
        </div>
      )}

      <Menu
        className="nav"
        mode="inline"
        inlineCollapsed={isCollapsed}
        selectedKeys={[location.pathname]}
        openKeys={isCollapsed ? undefined : openKeys}
        onOpenChange={(keys) => setOpenKeys(keys as string[])}
        items={items}
        onClick={({ key }) => {
          if (String(key).startsWith('/')) navigate(String(key))
        }}
      />
    </div>
  )
}
