import { useEffect, useRef } from 'react'
import { Alert, Button, Modal } from '../../ui'
import { i18n } from '../../i18n'
import { useChangesetSubmit } from '../../hooks/useChangesetSubmit'
import ReconcileSteps from './ReconcileSteps'
import './BatchCommitDialog.scss'

// 提交进度弹窗（FE-03 攒批）：打开即执行提交编排（commit→回读→轮询对账），
// ReconcileSteps 呈现 pushing→reading→终局。失败如实展示且变更集保留；成功
// onCommitted 供页面刷新（consoleEpoch 重挂）。进行中禁止关闭。
const t = (k: string) => i18n.global.t(k)

export default function BatchCommitDialog({
  open,
  device,
  onClose,
  onCommitted,
}: {
  open: boolean
  device: string
  onClose: () => void
  onCommitted: () => void
}) {
  const flow = useChangesetSubmit()
  const { reset, run, phaseRef } = flow
  const startedRef = useRef(false)

  const done =
    flow.progress.done || flow.timedOut || flow.phase === 'error' || flow.phase === 'idle'

  useEffect(() => {
    if (!open) {
      startedRef.current = false
      return
    }
    if (startedRef.current) return
    startedRef.current = true
    reset()
    void (async () => {
      const committed = await run(device)
      if (committed) onCommitted()
      if (phaseRef.current === 'idle') onClose() // 空集/用户取消 force：静默收起
    })()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, device])

  return (
    <Modal
      open={open}
      title={t('console.batch.commit')}
      width={560}
      maskClosable={false}
      closable={done}
      footer={
        <Button type="primary" data-test="commit-close" disabled={!done} onClick={onClose}>
          {done ? t('common.close') : t('console.reconciling')}
        </Button>
      }
      onCancel={() => {
        if (done) onClose() // 进行中不允许关闭
      }}
    >
      {flow.error && <Alert data-test="commit-error" type="error" showIcon message={flow.error} />}
      <ReconcileSteps progress={flow.progress} timedOut={flow.timedOut} />
    </Modal>
  )
}
