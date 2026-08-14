import { i18n } from '../../i18n'
import { formatXml } from '../../utils/xmlFormat'
import './XmlViewer.scss'

// NETCONF 报文查看器（FE-23）：带行号的缩进着色视图。格式化纯属**展示层**，
// 报文内容零改动（试运行的意义就在于「所见即将发」）。token 经 JSX 文本渲染而
// 非 innerHTML——报文含设备侧任意文本，绝不拼 HTML（XSS 免疫）。
const t = (k: string) => i18n.global.t(k)

export default function XmlViewer({ xml }: { xml?: string }) {
  const lines = formatXml(xml ?? '')
  if (!lines.length) return <div className="xml-viewer xml-empty" data-test="xml-viewer">{t('console.batch.noPayload')}</div>
  return (
    <div className="xml-viewer" data-test="xml-viewer">
      <ol className="xml-lines">
        {lines.map((line, i) => (
          <li key={i} className="xml-line">
            <span className="ln" aria-hidden="true">
              {i + 1}
            </span>
            <code className="code" style={{ paddingLeft: `${line.indent * 1.25}em` }}>
              {line.tokens.map((tk, j) => (
                <span key={j} className={`tk-${tk.kind}`}>
                  {tk.text}
                </span>
              ))}
            </code>
          </li>
        ))}
      </ol>
    </div>
  )
}
