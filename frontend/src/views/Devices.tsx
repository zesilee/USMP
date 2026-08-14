import { useCallback, useEffect, useMemo, useState } from 'react'
import { useNavigate } from 'react-router'
import { Button, Input, Modal, Select, Table, Tag, confirm, icons, toast, type TableColumnType } from '../ui'
import { i18n } from '../i18n'
import { useDeviceStore } from '../stores/device'
import { addDevice, getFleetReconcile, removeDevice } from '../api'
import { deriveDeviceRows, type DeviceRow } from '../utils/deviceRows'
import { validateDeviceForm, toAddDevicePayload, type DeviceFormInput } from '../utils/deviceForm'
import type { FleetInput } from '../composables/useFleetOverview'
import ReconcileChip from '../components/dashboard/ReconcileChip'
import Sparkline from '../components/common/Sparkline'
import './Devices.scss'

// Devices 页：设备台账（事实+会话态+对账真数据 join，离线优先）——搜索/状态/
// 厂商筛选、分页、添加（validateDeviceForm 纯函数闸门）、删除确认、探活。
// 语义自旧 Vue 版逐段平移（含「对账失败不阻断设备表」的 allSettled 边界）。
const t = (k: string, p?: Record<string, unknown>) => i18n.global.t(k, p)

const EMPTY_FORM: DeviceFormInput = { ip: '', port: '', username: '', password: '', vendor: '', role: '' }

export default function Devices() {
  const navigate = useNavigate()
  const devices = useDeviceStore((s) => s.devices)
  const fetchDevices = useDeviceStore((s) => s.fetchDevices)
  const selectDevice = useDeviceStore((s) => s.selectDevice)
  const testConnection = useDeviceStore((s) => s.testConnection)

  const [fleet, setFleet] = useState<FleetInput>({})
  const [loading, setLoading] = useState(false)
  const [searchKeyword, setSearchKeyword] = useState('')
  const [statusFilter, setStatusFilter] = useState('')
  const [vendorFilter, setVendorFilter] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)

  const load = useCallback(async () => {
    setLoading(true)
    try {
      // 设备与对账聚合并行；对账失败不阻断设备表（收敛态降级 unknown/off）。
      const [, fleetRes] = await Promise.allSettled([fetchDevices(), getFleetReconcile()])
      setFleet(fleetRes.status === 'fulfilled' ? ((fleetRes.value.data?.data as FleetInput) ?? {}) : {})
    } finally {
      setLoading(false)
    }
  }, [fetchDevices])

  useEffect(() => {
    void load()
  }, [load])

  const rows = useMemo<DeviceRow[]>(() => deriveDeviceRows(devices, fleet), [devices, fleet])
  const vendors = useMemo(() => [...new Set(rows.map((r) => r.vendor).filter(Boolean))].sort(), [rows])

  // 筛选/搜索变化回第一页，避免停在越界空页。
  useEffect(() => setPage(1), [searchKeyword, statusFilter, vendorFilter])

  const filteredRows = useMemo(() => {
    let result = rows
    if (searchKeyword) {
      const kw = searchKeyword.toLowerCase()
      result = result.filter((r) => r.ip.toLowerCase().includes(kw) || r.name.toLowerCase().includes(kw))
    }
    if (statusFilter) result = result.filter((r) => (statusFilter === 'online' ? r.online : !r.online))
    if (vendorFilter) result = result.filter((r) => r.vendor === vendorFilter)
    return result
  }, [rows, searchKeyword, statusFilter, vendorFilter])

  const goToConfig = (row: DeviceRow) => {
    // 同步写全局设备上下文（FE-10）；query 携带保证 URL 可分享（双写幂等）。
    selectDevice(row.ip)
    navigate(`/module/ifm?device=${row.ip}`)
  }

  // ── 添加设备（validateDeviceForm 纯函数为权威闸门；行内错误受控展示）──
  const [addOpen, setAddOpen] = useState(false)
  const [adding, setAdding] = useState(false)
  const [addForm, setAddForm] = useState<DeviceFormInput>(EMPTY_FORM)
  const [addErrors, setAddErrors] = useState<Partial<Record<keyof DeviceFormInput, string>>>({})

  const setF = (k: keyof DeviceFormInput, v: string) => setAddForm((prev) => ({ ...prev, [k]: v }))

  const submitAddDevice = async () => {
    const errors = validateDeviceForm(addForm)
    setAddErrors(errors)
    if (Object.keys(errors).length > 0) {
      const firstKey = Object.values(errors)[0]
      if (firstKey) toast(t(firstKey), 'error')
      return
    }
    setAdding(true)
    try {
      await addDevice(toAddDevicePayload(addForm))
      toast(t('devices.addSuccess', { ip: addForm.ip.trim() }))
      setAddOpen(false)
      await load()
    } catch (e: any) {
      // 后端拒绝（连不上/参数不合规）：不关框，展示原因供改后重试。
      toast(t('devices.addFailed', { reason: e?.response?.data?.message || e?.message || e }), 'error')
    } finally {
      setAdding(false)
    }
  }

  const handleDelete = async (row: DeviceRow) => {
    const ok = await confirm(t('devices.deleteConfirm', { ip: row.ip }), {
      title: t('common.delete'),
      danger: true,
    })
    if (!ok) return
    try {
      await removeDevice(row.ip)
      toast(t('devices.deleteSuccess', { ip: row.ip }))
      await load()
    } catch (e: any) {
      toast(t('devices.deleteFailed', { reason: e?.response?.data?.message || e?.message || e }), 'error')
    }
  }

  const handleTest = async (row: DeviceRow) => {
    const result = await testConnection(row.id)
    if (result.success) toast(t('devices.connTestSuccess', { name: row.name || row.ip }))
    else toast(`${row.name || row.ip} ${result.message}`, 'error')
  }

  const columns: TableColumnType<DeviceRow>[] = [
    {
      title: t('devices.colIp'),
      width: 150,
      render: (_, r) => <span className="mono strong">{r.ip}</span>,
    },
    { title: t('devices.colName'), width: 170, render: (_, r) => r.name },
    {
      title: t('devices.colVendorModel'),
      render: (_, r) => [r.vendor, r.model].filter(Boolean).join(' / ') || '—',
    },
    {
      title: t('devices.colRole'),
      width: 100,
      render: (_, r) => (r.role ? <Tag>{r.role}</Tag> : '—'),
    },
    {
      title: t('devices.colSession'),
      width: 120,
      render: (_, r) => (
        <Tag color={r.online ? 'green' : 'default'}>{r.online ? t('devices.online') : t('devices.offline')}</Tag>
      ),
    },
    { title: t('devices.colLoad'), width: 110, render: (_, r) => <Sparkline points={r.load} /> },
    {
      title: t('devices.colReconcile'),
      width: 120,
      render: (_, r) => <ReconcileChip state={r.reconcileState} />,
    },
    {
      title: t('devices.colLastSync'),
      render: (_, r) => <span className="mono dim">{r.lastSync || '—'}</span>,
    },
    {
      title: t('common.actions'),
      width: 200,
      fixed: 'right',
      render: (_, r) => (
        <>
          <Button type="link" size="small" onClick={() => goToConfig(r)}>
            {t('devices.viewConfig')}
          </Button>
          <Button type="link" size="small" onClick={() => void handleTest(r)}>
            {t('devices.testConnection')}
          </Button>
          <Button type="link" size="small" danger data-test="device-delete-btn" onClick={() => void handleDelete(r)}>
            {t('common.delete')}
          </Button>
        </>
      ),
    },
  ]

  const fieldError = (k: keyof DeviceFormInput) => (addErrors[k] ? t(addErrors[k]!) : undefined)

  return (
    <div className="devices" data-test="devices-page">
      <div className="page-header">
        <h1>{t('devices.title')}</h1>
        <div className="header-actions">
          <Input
            allowClear
            style={{ width: 200 }}
            placeholder={t('devices.searchPlaceholder')}
            prefix={<icons.SearchIcon />}
            value={searchKeyword}
            onChange={(e) => setSearchKeyword(e.target.value)}
          />
          <Select
            allowClear
            style={{ width: 120 }}
            placeholder={t('devices.filterStatus')}
            value={statusFilter || undefined}
            onChange={(v) => setStatusFilter(v ?? '')}
            options={[
              { label: t('devices.online'), value: 'online' },
              { label: t('devices.offline'), value: 'offline' },
            ]}
          />
          <Select
            allowClear
            style={{ width: 140 }}
            placeholder={t('devices.filterVendor')}
            value={vendorFilter || undefined}
            onChange={(v) => setVendorFilter(v ?? '')}
            options={vendors.map((v) => ({ label: v, value: v }))}
          />
          <Button icon={<icons.RefreshIcon />} onClick={() => void load()}>
            {t('common.refresh')}
          </Button>
          <Button type="primary" icon={<icons.PlusIcon />} data-test="add-device-btn" onClick={() => { setAddForm(EMPTY_FORM); setAddErrors({}); setAddOpen(true) }}>
            {t('devices.addDevice')}
          </Button>
        </div>
      </div>

      <Table
        size="small"
        rowKey="id"
        columns={columns}
        dataSource={filteredRows}
        loading={loading}
        pagination={{
          current: page,
          pageSize,
          total: filteredRows.length,
          pageSizeOptions: [10, 20, 50],
          showSizeChanger: true,
          onChange: (p, ps) => {
            setPage(ps !== pageSize ? 1 : p)
            setPageSize(ps)
          },
        }}
        locale={{ emptyText: loading ? t('devices.loadingEllipsis') : t('devices.emptyNone') }}
      />

      <Modal
        open={addOpen}
        title={t('devices.addDeviceTitle')}
        width={460}
        confirmLoading={adding}
        okText={t('common.confirm')}
        onOk={() => void submitAddDevice()}
        onCancel={() => setAddOpen(false)}
        destroyOnHidden
      >
        <div className="add-device-form" data-test="add-device-form">
          {(
            [
              ['ip', 'devices.fieldIp', true],
              ['port', 'devices.fieldPort', false],
              ['username', 'devices.fieldUsername', true],
              ['password', 'devices.fieldPassword', true],
              ['vendor', 'devices.fieldVendor', false],
              ['role', 'devices.fieldRole', false],
            ] as [keyof DeviceFormInput, string, boolean][]
          ).map(([key, labelKey, required]) => (
            <label key={key} className={`adf-item${fieldError(key) ? ' has-error' : ''}`}>
              <span className="adf-label">
                {t(labelKey)}
                {required && <i className="req">*</i>}
              </span>
              <Input
                type={key === 'password' ? 'password' : 'text'}
                value={addForm[key]}
                onChange={(e) => setF(key, e.target.value)}
                data-test={`add-${key}`}
              />
              {fieldError(key) && <span className="adf-err">{fieldError(key)}</span>}
            </label>
          ))}
        </div>
      </Modal>
    </div>
  )
}
