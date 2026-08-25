// NETCONF 报文格式化（FE-23 试运行可读性）：后端返回的是**零换行的紧凑 XML**
// （xmlcodec.Encode 的真实下发形态），直接塞进 <pre> 会糊成一整行。此处仅做
// **展示层**折行与缩进，绝不改动报文内容本身（报文保真是试运行的意义所在）。
//
// 输出为结构化 token 而非 HTML 字符串：渲染侧逐 token 出 span 着色，天然免疫
// XSS（报文里含设备返回的任意文本，绝不能走 innerHTML 类通道）。

/** token 类别，驱动着色。 */
export type XmlTokenKind = 'punct' | 'tag' | 'attr-name' | 'attr-value' | 'text' | 'comment'

export interface XmlToken {
  text: string
  kind: XmlTokenKind
}

export interface XmlLine {
  /** 缩进层级（每级 2 空格，由渲染侧决定视觉宽度）。 */
  indent: number
  tokens: XmlToken[]
}

/** 词法节点：标签或文本。 */
interface Node {
  raw: string
  kind: 'open' | 'close' | 'self' | 'meta' | 'text'
  name: string
}

/**
 * 切分 XML 为标签/文本节点。尊重引号——属性值内的 `>` 不得被当作标签结束
 * （如 xpath 型 must 表达式）。非法/残缺输入按文本吞下，不抛错（R08）。
 */
function tokenizeNodes(raw: string): Node[] {
  const nodes: Node[] = []
  let i = 0
  const n = raw.length
  while (i < n) {
    const lt = raw.indexOf('<', i)
    if (lt < 0) {
      pushText(nodes, raw.slice(i))
      break
    }
    if (lt > i) pushText(nodes, raw.slice(i, lt))

    // 找匹配的 '>'（跳过引号内内容）
    let j = lt + 1
    let quote: string | null = null
    while (j < n) {
      const c = raw[j]
      if (quote) {
        if (c === quote) quote = null
      } else if (c === '"' || c === "'") {
        quote = c
      } else if (c === '>') {
        break
      }
      j++
    }
    if (j >= n) {
      // 未闭合的 '<'：当纯文本处理，不丢字符
      pushText(nodes, raw.slice(lt))
      break
    }
    const tag = raw.slice(lt, j + 1)
    nodes.push({ raw: tag, kind: tagKind(tag), name: tagName(tag) })
    i = j + 1
  }
  return nodes
}

function pushText(nodes: Node[], s: string) {
  const t = s.trim()
  if (t) nodes.push({ raw: t, kind: 'text', name: '' })
}

function tagKind(tag: string): Node['kind'] {
  if (tag.startsWith('<?') || tag.startsWith('<!')) return 'meta'
  if (tag.startsWith('</')) return 'close'
  if (tag.endsWith('/>')) return 'self'
  return 'open'
}

function tagName(tag: string): string {
  const m = tag.match(/^<\/?\s*([^\s/>]+)/)
  return m ? m[1] : ''
}

/**
 * 把紧凑 XML 折成带缩进的行。相邻的 `<a>` + 文本 + `</a>` 合并为单行
 * （`<id>10</id>` 不拆三行）。缩进不为负（多余闭合标签容错）。
 */
export function formatXml(raw: string): XmlLine[] {
  if (!raw || !raw.trim()) return []
  const nodes = tokenizeNodes(raw)
  const lines: XmlLine[] = []
  let depth = 0

  for (let i = 0; i < nodes.length; i++) {
    const node = nodes[i]
    switch (node.kind) {
      case 'open': {
        // 纯文本叶合并：<a>text</a>
        const next = nodes[i + 1]
        const after = nodes[i + 2]
        if (next?.kind === 'text' && after?.kind === 'close' && after.name === node.name) {
          lines.push({ indent: depth, tokens: lineTokens(node.raw + next.raw + after.raw) })
          i += 2
          break
        }
        // 空元素对：<a></a> 保持两行（与设备报文形态一致，不擅自改写为自闭合）
        lines.push({ indent: depth, tokens: lineTokens(node.raw) })
        depth++
        break
      }
      case 'close':
        depth = Math.max(0, depth - 1)
        lines.push({ indent: depth, tokens: lineTokens(node.raw) })
        break
      case 'self':
      case 'meta':
      case 'text':
        lines.push({ indent: depth, tokens: lineTokens(node.raw) })
        break
    }
  }
  return lines
}

/**
 * 单行着色分词。无损：所有 token 文本拼接必须等于入参（渲染不吞字符）。
 */
export function lineTokens(line: string): XmlToken[] {
  if (line.startsWith('<!--')) return [{ text: line, kind: 'comment' }]

  const out: XmlToken[] = []
  const re = /<\/?[^>]*>/g
  let last = 0
  let m: RegExpExecArray | null
  while ((m = re.exec(line))) {
    if (m.index > last) out.push({ text: line.slice(last, m.index), kind: 'text' })
    out.push(...tagTokens(m[0]))
    last = m.index + m[0].length
  }
  if (last < line.length) out.push({ text: line.slice(last), kind: 'text' })
  return out.length ? out : [{ text: line, kind: 'text' }]
}

/** 拆一个标签为 标点/标签名/属性名/属性值 token。 */
function tagTokens(tag: string): XmlToken[] {
  if (tag.startsWith('<?') || tag.startsWith('<!')) return [{ text: tag, kind: 'meta' as XmlTokenKind }]

  const out: XmlToken[] = []
  const m = tag.match(/^(<\/?)\s*([^\s/>]+)/)
  if (!m) return [{ text: tag, kind: 'punct' }]
  out.push({ text: m[1], kind: 'punct' })
  out.push({ text: m[2], kind: 'tag' })

  const rest = tag.slice(m[0].length)
  // 属性：name="value" / name='value'，其余（空白、/、>）为标点
  const attrRe = /([^\s=/>]+)(\s*=\s*)("[^"]*"|'[^']*')/g
  let last = 0
  let a: RegExpExecArray | null
  while ((a = attrRe.exec(rest))) {
    if (a.index > last) out.push({ text: rest.slice(last, a.index), kind: 'punct' })
    out.push({ text: a[1], kind: 'attr-name' })
    out.push({ text: a[2], kind: 'punct' })
    out.push({ text: a[3], kind: 'attr-value' })
    last = a.index + a[0].length
  }
  if (last < rest.length) out.push({ text: rest.slice(last), kind: 'punct' })
  return out
}
