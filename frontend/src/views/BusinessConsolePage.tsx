import { useCallback, useEffect, useState } from 'react'
import { Alert, Breadcrumb, Button, Drawer, Input, Table, Tag, confirm, toast, type TableColumnType } from '../ui'
import { i18n } from '../i18n'
import {
  getYangSchema,
  listBusinessVlanServices,
  applyBusinessVlanService,
  deleteBusinessVlanService,
  type BusinessVlanServiceItem,
} from '../api'
import { useConfigForm } from '../hooks/useConfigForm'
import type { Field } from '../utils/crdSchemaParser'
import SchemaForm from '../components/config/SchemaForm'
import './BusinessConsolePage.scss'

// 平台作用域业务控制台（FE-17）：与设备作用域模块控制台并列——无设备选择器，
// 一个意图实例管理 spec.devices 里的 N 台设备。表单由意图 YANG schema 自动
// 渲染（R05；devices 为嵌套 list）。语义自旧 Vue 版逐段平移。
const MODULE = 'business-vlan-service'
const t = (k: string, p?: Record<string, unknown>) => i18n.global.t(k, p)

// 收敛状态按语义分档：tag 颜色取自档位而非文案，避免与 i18n 耦合。
type ConvergeKind = 'validationFailed' | 'pending' | 'converged' | 'partialFailed' | 'converging'

function condition(row: BusinessVlanServiceItem, type: string): any {
  return (row.status?.conditions || []).find((c: any) => c?.type === type)
}
function deviceStates(row: BusinessVlanServiceItem): any[] {
  return row.status?.deviceStates || []
}
function claims(row: BusinessVlanServiceItem): any[] {
  return row.status?.claims || []
}
function convergeKind(row: BusinessVlanServiceItem): ConvergeKind {
  const v = condition(row, 'Validated')
  if (v && v.status === 'False') return 'validationFailed'
  const c = condition(row, 'Converged')
  if (!c) return 'pending'
  if (c.status === 'True') return 'converged'
  const failed = deviceStates(row).filter((s: any) => s.phase === 'failed').length
  if (failed > 0) return 'partialFailed'
  return 'converging'
}
function convergeText(row: BusinessVlanServiceItem): string {
  const kind = convergeKind(row)
  switch (kind) {
    case 'validationFailed':
      return t('business.stateValidationFailed')
    case 'pending':
      return t('business.statePending')
    case 'converged':
      return t('common.state.conv')
    case 'partialFailed': {
      const states = deviceStates(row)
      const failed = states.filter((s: any) => s.phase === 'failed').length
      return t('business.statePartialFailed', { ok: states.length - failed, total: states.length })
    }
    default:
      return t('common.state.recon')
  }
}
function convergeColor(row: BusinessVlanServiceItem): string {
  const kind = convergeKind(row)
  if (kind === 'converged') return 'green'
  if (kind === 'validationFailed' || kind === 'partialFailed') return 'red'
  return 'default'
}
function phaseColor(phase: string): string {
  if (phase === 'synced') return 'green'
  if (phase === 'failed') return 'red'
  return 'default'
}

export default function BusinessConsolePage() {
  const [pageTitle, setPageTitle] = useState(t('business.defaultTitle'))
  const [schemaError, setSchemaError] = useState('')
  const [listError, setListError] = useState('')
  const [loading, setLoading] = useState(false)
  const [items, setItems] = useState<BusinessVlanServiceItem[]>([])
  const [schemaFields, setSchemaFields] = useState<Field[]>([])

  const form = useConfigForm(schemaFields)
  const { resetForm } = form

  const [drawerOpen, setDrawerOpen] = useState(false)
  const [detail, setDetail] = useState<BusinessVlanServiceItem | null>(null)
  const [instanceName, setInstanceName] = useState('')
  const [editingName, setEditingName] = useState('')

  const loadSchema = useCallback(async () => {
    try {
      const res = await getYangSchema(MODULE, 'nested')
      const payload = res.data?.data
      const fields = payload?.fields || []
      setSchemaFields(fields)
      if (payload?.title) setPageTitle((payload as any).description || t('business.defaultTitle'))
      if (!fields.length) setSchemaError(t('business.noRenderableFields', { module: MODULE }))
    } catch (e: any) {
      setSchemaError(e?.response?.data?.message || e?.message || t('business.loadModelFailed'))
    }
  }, [])

  const loadList = useCallback(async () => {
    setLoading(true)
    setListError('')
    try {
      const res = await listBusinessVlanServices()
      if (res.data?.success === false) {
        // 后端信封错误（HTTP 恒 200）：如未连接集群的 503 降级。
        setListError(res.data?.message || t('business.unavailable'))
        setItems([])
        return
      }
      setItems(res.data?.data?.items || [])
    } catch (e: any) {
      setListError(e?.response?.data?.message || e?.message || t('business.listFailed'))
      setItems([])
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    void loadSchema()
    void loadList()
  }, [loadSchema, loadList])

  const openCreate = () => {
    setEditingName('')
    setInstanceName('')
    resetForm()
    setDrawerOpen(true)
  }
  const openEdit = (row: BusinessVlanServiceItem) => {
    setEditingName(row.name)
    setInstanceName(row.name)
    resetForm(row.spec || {})
    setDrawerOpen(true)
  }

  const submit = async () => {
    if (form.blocked || !instanceName) return
    try {
      await applyBusinessVlanService(instanceName, form.visiblePayload())
      toast(t('business.submitted'))
      setDrawerOpen(false)
      await loadList()
    } catch (e: any) {
      toast(e?.response?.data?.message || e?.message || t('business.submitFailed'), 'error')
    }
  }

  const remove = async (row: BusinessVlanServiceItem) => {
    const ok = await confirm(t('business.removeConfirm', { name: row.name }), {
      title: t('common.confirmDelete'),
      danger: true,
    })
    if (!ok) return
    try {
      await deleteBusinessVlanService(row.name)
      toast(t('business.removeAccepted'))
      await loadList()
    } catch (e: any) {
      toast(e?.response?.data?.message || e?.message || t('business.removeFailed'), 'error')
    }
  }

  const columns: TableColumnType<BusinessVlanServiceItem>[] = [
    { title: t('business.colInstance'), dataIndex: 'name' },
    { title: 'VLAN', width: 90, render: (_, r) => r.spec?.['vlan-id'] ?? '-' },
    { title: t('business.colDeviceCount'), width: 90, render: (_, r) => (r.spec?.devices || []).length },
    {
      title: t('business.colConverge'),
      render: (_, r) => (
        <Tag color={convergeColor(r)} data-test={`converge-${r.name}`}>
          {convergeText(r)}
        </Tag>
      ),
    },
    {
      title: t('common.actions'),
      width: 210,
      render: (_, r) => (
        <>
          <Button type="link" size="small" onClick={() => setDetail(r)}>
            {t('business.detail')}
          </Button>
          <Button type="link" size="small" data-test={`business-edit-${r.name}`} onClick={() => openEdit(r)}>
            {t('common.edit')}
          </Button>
          <Button type="link" size="small" danger data-test={`business-remove-${r.name}`} onClick={() => void remove(r)}>
            {t('common.delete')}
          </Button>
        </>
      ),
    },
  ]

  return (
    <div className="business-console" data-test="business-console">
      <Breadcrumb separator="/" items={[{ title: t('nav.businessConfig') }, { title: pageTitle }]} />

      {schemaError && <Alert type="warning" showIcon message={schemaError} />}
      {!schemaError && listError && (
        <Alert type="warning" showIcon message={listError} data-test="business-unavailable" />
      )}

      {!schemaError && (
        <>
          <div className="toolbar">
            <Button type="primary" data-test="business-create" onClick={openCreate}>
              {t('business.create')}
            </Button>
            <span className="tip">{t('business.tip')}</span>
          </div>

          <Table
            size="small"
            rowKey="name"
            columns={columns}
            dataSource={items}
            loading={loading}
            data-test="business-table"
          />

          {/* 新建/编辑抽屉：表单由意图 YANG schema 自动渲染（R05；devices 嵌套 list）。 */}
          <Drawer
            open={drawerOpen}
            width={560}
            title={editingName ? t('business.editTitle', { name: editingName }) : t('business.create')}
            onClose={() => setDrawerOpen(false)}
          >
            <div className="biz-name-item">
              <label>
                {t('business.nameLabel')}
                <i className="req">*</i>
              </label>
              <Input
                value={instanceName}
                disabled={!!editingName}
                placeholder={t('business.namePlaceholder')}
                data-test="business-name-input"
                onChange={(e) => setInstanceName(e.target.value)}
              />
            </div>
            <SchemaForm fields={schemaFields} form={form} />
            <div className="drawer-actions">
              <Button onClick={() => setDrawerOpen(false)}>{t('common.cancel')}</Button>
              <Button
                type="primary"
                disabled={form.blocked || !instanceName}
                data-test="business-submit"
                onClick={() => void submit()}
              >
                {t('common.submit')}
              </Button>
            </div>
          </Drawer>

          {/* 详情抽屉：每设备收敛状态与失败原因（BIC-04 deviceStates）。 */}
          <Drawer
            open={!!detail}
            width={480}
            title={t('business.detailTitle', { name: detail?.name || '' })}
            onClose={() => setDetail(null)}
          >
            {detail && (
              <>
                <h4>{t('business.perDeviceState')}</h4>
                <Table
                  size="small"
                  rowKey="device"
                  data-test="device-states"
                  dataSource={deviceStates(detail)}
                  pagination={false}
                  columns={[
                    { title: t('common.device'), dataIndex: 'device', width: 130 },
                    {
                      title: t('common.status'),
                      width: 90,
                      render: (_, r: any) => <Tag color={phaseColor(r.phase)}>{r.phase}</Tag>,
                    },
                    { title: t('business.colReason'), dataIndex: 'reason', ellipsis: true },
                  ]}
                />
                <h4>{t('business.claimedNative')}</h4>
                <Table
                  size="small"
                  rowKey={(r: any) => `${r.device}-${r.path}`}
                  dataSource={claims(detail)}
                  pagination={false}
                  columns={[
                    { title: t('common.device'), dataIndex: 'device', width: 130 },
                    { title: t('business.colPath'), dataIndex: 'path', ellipsis: true },
                  ]}
                />
              </>
            )}
          </Drawer>
        </>
      )}
    </div>
  )
}
