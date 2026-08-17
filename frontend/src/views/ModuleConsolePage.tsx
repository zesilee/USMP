import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { useParams, useSearchParams, useBlocker } from 'react-router'
import { Alert, Breadcrumb, Empty, Select, Tabs, Tag, Tooltip, confirm } from '../ui'
import { i18n, useLocale } from '../i18n'
import { getYangSchema, getOwnership } from '../api'
import { localizeFields, localizeRpcs } from '../composables/useFieldLabels'
import { useMenuStore } from '../stores/menu'
import { useChangesetStore } from '../stores/changeset'
import { useDeviceStore } from '../stores/device'
import type { Field } from '../utils/crdSchemaParser'
import { deriveTabs, deriveRpcTabs, type ConsoleTab, type RpcDef } from '../utils/moduleConsole'
import ModuleListTab from '../components/config/ModuleListTab'
import ModuleFormTab from '../components/config/ModuleFormTab'
import RpcExecuteTab from '../components/config/RpcExecuteTab'
import BatchToolbar from '../components/config/BatchToolbar'
import BatchCommitDialog from '../components/config/BatchCommitDialog'
import './ModuleConsolePage.scss'

// ModuleConsolePage（FE-10/18/19/24 宿主）：通用模块控制台——schema 拉取（?device=
// 附不支持预标记 CN-05）→ 标签本地化（UI-03 懒加载原位替换，rootName 守卫防
// 竞态）→ deriveTabs 一级 Tab（rpc 不进 Tab 栏，导航落点=左树）→ 各 Tab 组件。
// rpc 直达模式仅渲染该 rpc 面板（未知名明确报错 R08）。攒批工具栏与提交编排随
// tasks 11 组在 header-actions 挂载（consoleEpoch 重挂机制已就位）。
const t = (k: string, p?: Record<string, unknown>) => i18n.global.t(k, p)

export default function ModuleConsolePage() {
  const params = useParams()
  const [searchParams] = useSearchParams()
  const moduleName = String(params.module || '')
  const rpcName = String(params.rpcName || '')
  const rpcMode = !!rpcName

  const locale = useLocale()
  const menuStore = useMenuStore()
  const devices = useDeviceStore((s) => s.devices)
  const selectedDeviceIp = useDeviceStore((s) => s.selectedDeviceIp)
  const selectDevice = useDeviceStore((s) => s.selectDevice)
  const fetchDevices = useDeviceStore((s) => s.fetchDevices)

  useEffect(() => {
    void fetchDevices()
  }, [fetchDevices])

  // 入页优先级 query > store：深链显式指定则覆盖全局上下文；重复参数取首个。
  useEffect(() => {
    const ip = searchParams.get('device')
    if (ip) selectDevice(ip)
  }, [searchParams, selectDevice])

  const [schemaError, setSchemaError] = useState('')
  const [title, setTitle] = useState('')
  const [vendor, setVendor] = useState('')
  const [rootName, setRootName] = useState('')
  const [schemaFields, setSchemaFields] = useState<Field[]>([])
  const [rpcs, setRpcs] = useState<RpcDef[]>([])
  const [unsupportedTabs, setUnsupportedTabs] = useState<string[]>([])
  const [activeTab, setActiveTab] = useState('')
  // 重置/提交成功 → 重挂 Tab 内容组件：表单回设备实际态、列表标记行还原。
  const [consoleEpoch, setConsoleEpoch] = useState(0)
  const [commitOpen, setCommitOpen] = useState(false)
  const changeset = useChangesetStore()

  // 提交编排（FE-03/FE-23）：确认 → 提交进度弹窗。
  const onCommitRequest = useCallback(async () => {
    const n = changeset.countFor(selectedDeviceIp)
    if (!n) return
    const ok = await confirm(t('console.batch.commitConfirm', { count: n }), {
      title: t('console.batch.commit'),
    })
    if (ok) setCommitOpen(true)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [changeset, selectedDeviceIp])

  // 路由离开确认（FE-23 负路径）：存在未提交变更提示；取消停留、变更集保留。
  const blocker = useBlocker(() => changeset.countFor(selectedDeviceIp) > 0)
  useEffect(() => {
    if (blocker.state !== 'blocked') return
    void (async () => {
      const ok = await confirm(t('console.batch.leaveConfirm'), { title: t('console.batch.changes') })
      if (ok) blocker.proceed()
      else blocker.reset()
    })()
  }, [blocker])

  const rawRef = useRef<{ fields: Field[]; rpcs: RpcDef[] }>({ fields: [], rpcs: [] })
  // schema 载入代际：loadSchema 重跑（如选设备后带 ?device= 重取预标记）会以原始
  // fields 覆盖已本地化版本，而 rootName/locale/leftTree 均未变 → relabel effect
  // 不会重跑（真机 E2E 回归：Tab 停在原始名）。代际计数强制 relabel 跟跑。
  const [schemaEpoch, setSchemaEpoch] = useState(0)

  const loadSchema = useCallback(async () => {
    setSchemaError('')
    setSchemaFields([])
    try {
      // 带设备取 schema：零额外请求拿到该设备的不支持预标记（CN-05）。
      const res = await getYangSchema(moduleName, 'nested', selectedDeviceIp || undefined)
      const data = res.data?.data
      setUnsupportedTabs(((data?.unsupported ?? []) as string[]) || [])
      rawRef.current = { fields: data?.fields ?? [], rpcs: (data?.rpcs ?? []) as RpcDef[] }
      setTitle(data?.title || moduleName)
      setVendor(data?.vendor || '')
      // 运行时配置路径根段 = 模块根容器名（schema title 即 root.Name()）。
      setRootName(data?.title || moduleName)
      setSchemaFields(rawRef.current.fields)
      setRpcs(rawRef.current.rpcs)
      const tabs0 = deriveTabs(data?.fields ?? [])
      setActiveTab(tabs0[0]?.name || '')
      setSchemaEpoch((n) => n + 1)
    } catch (e: any) {
      // schema 拉取失败降级：页面不崩，明确报错（R08/§9）。
      setSchemaError(e?.response?.data?.message || e?.message || t('console.schemaLoadFailed'))
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [moduleName, selectedDeviceIp])

  useEffect(() => {
    void loadSchema()
  }, [loadSchema])

  // 标签本地化（UI-03）：res 懒加载异步，首帧原始标签；rootName 守卫防快速切模块竞态。
  useEffect(() => {
    if (!rootName) return
    let alive = true
    void (async () => {
      const [lf, lr] = await Promise.all([
        localizeFields(rawRef.current.fields, rootName, locale, menuStore.leftTree),
        localizeRpcs(rawRef.current.rpcs as any, rootName, locale, menuStore.leftTree),
      ])
      if (!alive) return
      setSchemaFields(lf as Field[])
      setRpcs(lr as RpcDef[])
    })()
    return () => {
      alive = false
    }
  }, [rootName, locale, menuStore.leftTree, schemaEpoch])

  // 软归属（FE-18）：查询失败静默降级为无徽标（R08）。
  const [ownershipIntents, setOwnershipIntents] = useState<string[]>([])
  useEffect(() => {
    setOwnershipIntents([])
    if (!selectedDeviceIp || !moduleName) return
    let alive = true
    void (async () => {
      try {
        const res = await getOwnership(selectedDeviceIp)
        const claims: any[] = res.data?.data?.claims || []
        const intents = new Set<string>()
        for (const c of claims) {
          if (c?.module === moduleName && c?.intent) intents.add(c.intent)
        }
        if (alive) setOwnershipIntents([...intents].sort())
      } catch {
        /* 无徽标即可 */
      }
    })()
    return () => {
      alive = false
    }
  }, [selectedDeviceIp, moduleName])

  const tabs = useMemo<ConsoleTab[]>(() => deriveTabs(schemaFields), [schemaFields])
  const activeRpcTab = useMemo(
    () => deriveRpcTabs(rpcs).find((x) => x.rpc?.name === rpcName),
    [rpcs, rpcName],
  )
  const schemaLoaded = !!rootName
  const activeTabLabel = rpcMode
    ? activeRpcTab?.label || ''
    : tabs.find((x) => x.name === activeTab)?.label || ''

  // 运行中学习回同步（FE-24）：Tab 头淡化即时呈现。
  const onTabUnsupportedChange = useCallback((name: string, un: boolean) => {
    setUnsupportedTabs((prev) => {
      const has = prev.includes(name)
      if (un && !has) return [...prev, name]
      if (!un && has) return prev.filter((n) => n !== name)
      return prev
    })
  }, [])

  return (
    <div className="module-console" data-test="module-console">
      <div className="page-header">
        <Breadcrumb
          separator=">"
          items={[
            { title: t('console.breadcrumbConfig') },
            ...(vendor ? [{ title: vendor }] : []),
            { title },
            ...(activeTabLabel ? [{ title: activeTabLabel }] : []),
          ]}
        />
        <div className="header-actions">
          {/* 攒批工具栏（FE-23）：变更内容/试运行/重置/提交配置。 */}
          {selectedDeviceIp && (
            <BatchToolbar
              device={selectedDeviceIp}
              onReset={() => setConsoleEpoch((n) => n + 1)}
              onCommitRequest={() => void onCommitRequest()}
            />
          )}
          {ownershipIntents.length > 0 && (
            <Tooltip title={t('console.ownedTooltip', { intents: ownershipIntents.join('、') })}>
              <Tag color="orange" data-test="ownership-badge">
                {t('console.ownedBadge', { n: ownershipIntents.length })}
              </Tag>
            </Tooltip>
          )}
          {/* 全局设备上下文（FE-10）：选一次跨模块保持。 */}
          <Select
            style={{ width: 220 }}
            placeholder={t('console.selectDevicePlaceholder')}
            value={selectedDeviceIp || undefined}
            onChange={(v) => selectDevice(v)}
            options={devices.map((d) => ({ label: d.ip, value: d.ip }))}
            data-test="device-select"
          />
        </div>
      </div>

      {schemaError && <Alert type="error" showIcon message={schemaError} />}

      {!schemaError && !selectedDeviceIp && (
        <Empty data-test="select-device-empty" description={t('console.selectDeviceFirst')} />
      )}

      {!schemaError && selectedDeviceIp && rpcMode && (
        <>
          {activeRpcTab?.rpc ? (
            <div className="console-tabs">
              <RpcExecuteTab rpc={activeRpcTab.rpc} module={rootName} device={selectedDeviceIp} />
            </div>
          ) : schemaLoaded ? (
            <Alert
              data-test="rpc-not-found"
              type="error"
              showIcon
              message={t('console.rpcNotFound', { rpc: rpcName })}
            />
          ) : (
            <Empty description={t('console.schemaLoading')} />
          )}
        </>
      )}

      {!schemaError && selectedDeviceIp && !rpcMode && tabs.length > 0 && (
        <Tabs
          className="console-tabs"
          activeKey={activeTab}
          onChange={setActiveTab}
          items={tabs.map((tab) => ({
            key: tab.name,
            // FE-24 Tab 头淡化：设备不支持诚实透出（不隐藏），仅视觉降级。
            label: (
              <span
                className={unsupportedTabs.includes(tab.name) ? 'tab-unsupported' : undefined}
                data-test={unsupportedTabs.includes(tab.name) ? 'tab-unsupported' : undefined}
              >
                {tab.label}
              </span>
            ),
            children:
              tab.kind === 'list' ? (
                <ModuleListTab
                  key={consoleEpoch}
                  tab={tab}
                  rootName={rootName}
                  device={selectedDeviceIp}
                  unsupported={unsupportedTabs.includes(tab.name)}
                  onUnsupportedChange={(un) => onTabUnsupportedChange(tab.name, un)}
                />
              ) : (
                <ModuleFormTab
                  key={consoleEpoch}
                  tab={tab}
                  rootName={rootName}
                  device={selectedDeviceIp}
                  unsupported={unsupportedTabs.includes(tab.name)}
                  onUnsupportedChange={(un) => onTabUnsupportedChange(tab.name, un)}
                />
              ),
          }))}
        />
      )}

      {!schemaError && selectedDeviceIp && !rpcMode && tabs.length === 0 && (
        <Empty description={t('console.schemaLoading')} />
      )}

      <BatchCommitDialog
        open={commitOpen}
        device={selectedDeviceIp}
        onClose={() => setCommitOpen(false)}
        onCommitted={() => setConsoleEpoch((n) => n + 1)}
      />
    </div>
  )
}
