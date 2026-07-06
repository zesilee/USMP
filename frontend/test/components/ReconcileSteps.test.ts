import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import ReconcileSteps from '../../src/components/config/ReconcileSteps.vue'
import { deriveReconcileProgress } from '../../src/utils/reconcileProgress'

describe('ReconcileSteps · 对账进度', () => {
  it('渲染三步，reading 阶段第三步 active、无终局徽标', () => {
    const w = mount(ReconcileSteps, { props: { progress: deriveReconcileProgress('reading') } })
    const steps = w.findAll('.rstep')
    expect(steps).toHaveLength(3)
    expect(steps[0].classes()).toContain('done')
    expect(steps[2].classes()).toContain('active')
    expect(w.find('.recon-result').exists()).toBe(false)
  })

  it('converged → 三步 done + 已收敛徽标', () => {
    const w = mount(ReconcileSteps, { props: { progress: deriveReconcileProgress('converged') } })
    expect(w.findAll('.rstep.done')).toHaveLength(3)
    const chip = w.find('.recon-result')
    expect(chip.classes()).toContain('conv')
    expect(chip.text()).toContain('已收敛')
  })

  it('drifted → 漂移徽标', () => {
    const w = mount(ReconcileSteps, { props: { progress: deriveReconcileProgress('drifted') } })
    expect(w.find('.recon-result').classes()).toContain('drift')
    expect(w.find('.recon-result').text()).toContain('已漂移')
  })

  it('error → 第三步 error + 失败徽标', () => {
    const w = mount(ReconcileSteps, { props: { progress: deriveReconcileProgress('error') } })
    expect(w.findAll('.rstep')[2].classes()).toContain('error')
    expect(w.find('.recon-result').classes()).toContain('error')
    expect(w.find('.recon-result').text()).toContain('下发失败')
  })

  it('timedOut → 诚实标注仍在对账（非成功）', () => {
    const w = mount(ReconcileSteps, { props: { progress: deriveReconcileProgress('reading'), timedOut: true } })
    const chip = w.find('.recon-result')
    expect(chip.classes()).toContain('recon')
    expect(chip.text()).toContain('对账仍在进行')
  })
})
