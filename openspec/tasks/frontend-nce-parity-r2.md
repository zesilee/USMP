---
id: frontend-nce-parity-r2
title: 前端 NCE 对齐第二轮（首轮收官后的未完项跟踪）
status: pending
priority: medium
branch: (未开工——各项开工时另立 worktree/change)
worktree: (无活跃)
change: (按项另立)
updated: 2026-08-24
r1_done: eview-visual-fixes 首轮已合入（PR#408，2026-08-24）——冻结列退役+sticky 操作列、列宽等宽/0宽修复、空标记列条件化、行内获取数据源、侧栏拖拽、左树横滚、柔和标签、编辑面字段序+控件全链等宽、e2e 双口径门禁修复、内网一键前端重部署脚本
next: 按优先级逐项开工；每项独立 change + PR
---

## 目标

首轮目视验收（NCE 参照）已收官合入。本文件跟踪剩余未完项，防跨会话漏做。
每项开工时各自走 §5 工作流（propose→apply），禁止在本任务下直接堆代码。

## 未完项清单

### A. 行级「获取数据源」精确取数（功能）
- 现状：行内入口已有，行为=整表强取（无库架构下取数粒度=整表路径）。
- 目标：NCE 语义=只取该行实例。后端已具备谓词下钻读（[[list-server-pagination]]），
  前端按主键拼谓词路径取单条+行内合并即可。
- 层次：B3（若加 API 形态）+ F2；spec delta（fetch 行为契约变更，R17 spec-first）。

### B. 列头筛选 NCE 全列形态（增强）
- 现状：桥自绘筛选菜单仅 enum/boolean 本地列（antd 语义对等）。
- 目标（可选）：NCE 每列头筛选+排序双图标形态；服务端模式列筛选需后端谓词支持。
- 先决：真实使用反馈——无人用就不做。

### C. 目视残余观感项（CSS 微调，攒批一轮做）
- sticky 操作列：悬停行背景与 sticky 底色不同步；左侧阴影浓淡。
- 变更内容对话框树表：拍平后无缩进层级。
- Dropdown 菜单选中态无高亮；菜单项 data-test 锚点债（label 文本化）。
- 展开行/Tag 圆角等杂项：用户日常使用中随手记，攒批修。

### D. 内网构建与发布完善（工程）
- 已做（首轮收官）：`scripts/inner-frontend-redeploy.sh` 前端一键重部署
  （pull→build→prebuilt 镜像→kind load→rollout→就绪等待）。
- 待做：
  - 后端/模拟器同款一键脚本（或参数化并入同一脚本 `--backend/--all`）；
  - 模拟网元种子数据漂移问题（首轮实录：VLAN 种子 1/100/101/102 一度消失、
    pod 重启后自愈）——排查 sim 状态持久化/就绪探针，必要时加种子自检；
  - 内网构建产物版本可见性：页面 footer 或 /healthz 暴露 git sha，
    杜绝「改没生效还是没部署」的判别成本（首轮至少 3 轮浪费在这上面）；
  - 摆渡依赖清单文档化（node22/浏览器包/eview 真包路径），换机可复现。

### E. 工程债（低优先）
- CI「Install Chromium」下载超时 flaky——加浏览器缓存。
- axios 清退评估（inula-request 已接管，窗口期回退通道）。
- firefox/webkit 兼容抽查（chromium 单口径拍板，有真实需求再启）。

## 方法论沉淀（本轮验证有效，后续沿用）

- eview 观感问题一律「探针先行」：diag spec dump DOM 链（类名|实宽|display）
  再改，盲改一轮=一次内网摆渡，探针一轮定案（C1-C4 实录）。
- eview CSS 对抗三板斧：!important 压制（表头 relative/树截断）、
  点名类拉伸（前缀通配会误伤同前缀按钮——「+」覆盖输入事故）、
  纯展示元素直接桥自绘收编（Tag）。
