import { describe, it, expect, vi } from 'vitest'
import { useState } from 'react'
import { render, screen, fireEvent, waitFor, cleanup } from '../gate/inula-testing'
import { afterEach } from 'vitest'
import { Button, Input, Tag, Alert, Select } from 'antd'

afterEach(cleanup)

// 翻转波路线判定探针：antd 6 组件在 openinula+18 级垫片 上的存活性。
// 绿 = "运行时先切、antd 暂留、EviewUI 逐步替换"的渐进路径成立。

describe('antd on openinula+shim', () => {
  it('Button 渲染与点击', async () => {
    const onClick = vi.fn()
    render(<Button type="primary" onClick={onClick}>hi</Button>)
    fireEvent.click(screen.getByRole('button'))
    expect(onClick).toHaveBeenCalled()
  })

  it('Input 受控输入链', async () => {
    function Host() {
      const [v, setV] = useState('a')
      return <Input value={v} onChange={(e) => setV(e.target?.value ?? (e.currentTarget as any)?.value)} />
    }
    const { container } = render(<Host />)
    const input = container.querySelector('input')!
    fireEvent.change(input, { target: { value: 'ab' } })
    await waitFor(() => expect(input.value).toBe('ab'))
  })

  it('Tag/Alert 静态渲染', () => {
    render(
      <div>
        <Tag color="success">ok</Tag>
        <Alert type="info" message="msg" showIcon />
      </div>,
    )
    expect(screen.getByText('ok')).toBeInTheDocument()
    expect(screen.getByText('msg')).toBeInTheDocument()
  })

  it('Select 弹层（useId/teleport 重度用户）', async () => {
    render(
      <Select
        open
        options={[
          { label: 'alpha', value: 'a' },
          { label: 'beta', value: 'b' },
        ]}
      />,
    )
    await waitFor(() => {
      const opts = document.body.querySelectorAll('.ant-select-item-option')
      expect(opts.length).toBeGreaterThan(0)
    })
  })
})
