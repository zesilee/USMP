// 手写补充声明（真包未附带 locales 的 d.ts）：zh 语言包=intl messages 字典。
// UiProvider 静态引入；外网测试经 vitest 别名到空 stub。
declare const messages: Record<string, string>
export default messages
