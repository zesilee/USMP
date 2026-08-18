// 运行时守卫统一安装点（组 7 E2E 定案）：必须在应用入口第一行 import——
// 生产 bundle 模块序下 provider 安装时机晚于库模块初始化会被绕过
// （dev 按需加载无此问题）。
import { installFindDOMNodePolyfill } from './finddomnode-polyfill'
import { installAttachShadowGuard } from './attachshadow-guard'

installFindDOMNodePolyfill()
installAttachShadowGuard()
