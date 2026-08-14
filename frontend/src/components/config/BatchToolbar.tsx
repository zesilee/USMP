import { useEffect, useRef, useState } from 'react'
import { Alert, Badge, Button, confirm } from '../../ui'
import { i18n } from '../../i18n'
import { useChangesetStore } from '../../stores/changeset'
import ChangesContentDialog from './ChangesContentDialog'
import DryRunDialog from './DryRunDialog'
import './BatchToolbar.scss'

// 攒批工具栏（FE-23）：变更内容（徽标=未提交条目数）/试运行/重置/提交配置。
// 空集禁用后三者；有变更展示提示条（可关闭，清零后再攒重新出现）。提交编排由
// 页面层承担（onCommitRequest）。语义自旧版逐段平移。
const t = (k: string, p?: Record<string, unknown>) => i18n.global.t(k, p)

export default function BatchToolbar({
  device,
  onReset,
  onCommitRequest,
}: {
  device: string
  onReset: () => void
  onCommitRequest: () => void
}) {
  const changeset = useChangesetStore()
  const count = changeset.countFor(device)

  const [changesOpen, setChangesOpen] = useState(false)
  const [dryRunOpen, setDryRunOpen] = useState(false)

  // 提示条：有变更即显；关闭后保持隐藏，直到清零后再次攒入（0→>0 复位）。
  const [hintClosed, setHintClosed] = useState(false)
  const prevCount = useRef(count)
  useEffect(() => {
    if (prevCount.current === 0 && count > 0) setHintClosed(false)
    prevCount.current = count
  }, [count])
  const showHint = count > 0 && !hintClosed

  // 重置（FE-23）：确认后清空当前设备变更集；页面层收 reset 恢复表单/标记行。
  const handleReset = async () => {
    const ok = await confirm(t('console.batch.resetConfirm', { count }), {
      title: t('console.batch.resetConfirmTitle'),
    })
    if (!ok) return
    changeset.clear(device)
    onReset()
  }

  return (
    <div className="batch-toolbar">
      {showHint && (
        <Alert
          data-test="batch-hint"
          className="batch-hint"
          type="info"
          showIcon
          closable
          message={t('console.batch.hint')}
          onClose={() => setHintClosed(true)}
        />
      )}
      <Badge count={count} size="small">
        <Button data-test="batch-changes" onClick={() => setChangesOpen(true)}>
          {t('console.batch.changes')}
        </Button>
      </Badge>
      <Button data-test="batch-dryrun" disabled={count === 0} onClick={() => setDryRunOpen(true)}>
        {t('console.batch.dryRun')}
      </Button>
      <Button data-test="batch-reset" disabled={count === 0} onClick={() => void handleReset()}>
        {t('console.batch.reset')}
      </Button>
      <Button data-test="batch-commit" type="primary" disabled={count === 0} onClick={onCommitRequest}>
        {t('console.batch.commit')}
      </Button>

      <ChangesContentDialog open={changesOpen} device={device} onClose={() => setChangesOpen(false)} />
      <DryRunDialog open={dryRunOpen} device={device} onClose={() => setDryRunOpen(false)} />
    </div>
  )
}
