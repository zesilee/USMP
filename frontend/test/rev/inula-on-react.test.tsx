import { describe, it, expect } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
// 反向 alias 下（openinula→react），两件套应在 React 19 上工作。
import { VueI18n, I18nProvider, useIntl } from 'inula-intl/build/cjs/intl.js'
import { BrowserRouter, Route, Switch, Link, useHistory, Prompt } from 'inula-router/router/cjs/router.js'

describe('inula-intl on React 19（反向 alias 波 A 前提）', () => {
  it('VueI18n 适配器：$t 插值 + changeLanguage + change 事件', () => {
    const i18n = new VueI18n({
      locale: 'zh',
      messages: { zh: { hello: '你好 {name}' }, en: { hello: 'hi {name}' } },
    })
    expect(i18n.$t('hello', { name: 'x' })).toBe('你好 x')
    let changed = 0
    i18n.on('change', () => changed++)
    i18n.changeLanguage('en')
    expect(i18n.$t('hello', { name: 'x' })).toBe('hi x')
    expect(changed).toBeGreaterThan(0)
  })

  it('I18nProvider + useIntl 在 React 组件树内工作', () => {
    function Probe() {
      const intl = useIntl()
      return <span>{intl.formatMessage({ id: 'k' })}</span>
    }
    render(
      <I18nProvider locale="zh" messages={{ k: '词条' }}>
        <Probe />
      </I18nProvider>,
    )
    expect(screen.getByText('词条')).toBeInTheDocument()
  })
})

describe('inula-router on React 19（v5 API）', () => {
  it('Switch/Route 渲染 + Link 导航 + useHistory', async () => {
    function PageB() {
      const history = useHistory()
      return (
        <div>
          <span>page-b</span>
          <button onClick={() => history.push('/')}>back</button>
        </div>
      )
    }
    render(
      <BrowserRouter>
        <Switch>
          <Route exact path="/">
            <div>
              <span>page-a</span>
              <Link to="/b">go-b</Link>
            </div>
          </Route>
          <Route path="/b">
            <PageB />
          </Route>
        </Switch>
      </BrowserRouter>,
    )
    expect(screen.getByText('page-a')).toBeInTheDocument()
    fireEvent.click(screen.getByText('go-b'))
    await waitFor(() => expect(screen.getByText('page-b')).toBeInTheDocument())
    fireEvent.click(screen.getByText('back'))
    await waitFor(() => expect(screen.getByText('page-a')).toBeInTheDocument())
  })

  it('Prompt 挂载不崩（离开守卫桥的载体）', () => {
    render(
      <BrowserRouter>
        <Prompt when={false} message="留在原页？" />
        <span>ok</span>
      </BrowserRouter>,
    )
    expect(screen.getByText('ok')).toBeInTheDocument()
  })
})
