// UI 适配层 · 主题令牌（FA-04）：取值**逐项对齐 styles/variables.scss**（PR-0a
// 浅色 iMaster 令牌，#83）——四组核心值（主色/中性色/圆角/间距，design D9 拍板
// 粒度）。业务组件 SHALL 从这里取令牌，SHALL NOT 硬编码色值；换库时本文件是
// 唯一的主题对接点。改这里前先改 variables.scss，两处同源（评审防线：脚手架
// 首版曾误用 Arco 默认色板致双色系，已按实值重钉）。
/** 品牌主色（= $color-primary #0C5EA6，深海蓝 ink 系）。 */
export const colorPrimary = '#0C5EA6'

/** 语义色（= $color-success / $color-warning / $color-error）。 */
export const colorSuccess = '#10814A'
export const colorWarning = '#B26A00'
export const colorError = '#C7000B'

/** 中性色（= $text-1/2/3 与 $border、$bg-page、$bg-elevated）。 */
export const textPrimary = '#16222E'
export const textSecondary = '#57697A'
export const textTertiary = '#93A2B1'
export const borderColor = '#DBE2EA'
export const bgLayout = '#EAEEF3'
export const bgElevated = '#F4F6F9'

/** 圆角（= $radius-md 6；控件圆角 $radius-ctl 7 由 antd 按比例派生）与基准
 *  字号（14px = 行业事实标准，iView/NCE 调研双重印证）。 */
export const borderRadius = 6
export const fontSize = 14

// antd 主题装配已随组 5 接线挪至 antd-backend/provider（测试后端镜像）；
// 生产主题注入 = eview/theme.ts 的 CSS 变量覆盖（UiProvider 装配）。
