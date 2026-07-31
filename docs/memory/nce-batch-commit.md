---
name: nce-batch-commit
description: 攒批提交二期全交付：变更集/试运行/批量原子提交/即时下发退役；含 8 个交付坑（PR拆分/worktree前端环境/vitest红灯机制）
metadata:
  type: project
---

# 攒批提交工作流（二期，已交付 2026-07-31）

**拍板**（用户确认 2026-07-31）：①完全改攒批——表单「确定」只入变更集，右上「提交配置」才下发，即时下发退役；②失败整体回退（candidate 两阶段，全成或全退）；③导出/配置项本期不做。目标截图 3 张（变更内容/试运行/网元差异）在 `~/ui/`。

**交付**（#245 提案 → #244 纯函数 → #247 preview → #248 commit → #249 store → #250 工具栏弹窗 → #251 链路切换 → PR-6 收尾）：

- 后端 `config-changeset`（CS-01~05）：`POST /config/changeset/preview`（正向/回滚报文+diff 树，纯计算零副作用，基线三级链 desired→cache→实时回读并标注来源；无 XML 通道模块如实降级）+ `/commit`（复用 intent.TxCoordinator，Fragment 加 Op/RawXML；**desired 在设备 commit 成功后才写**——先写会被周期对账绕过 2PC 复推失败变更；归属硬锁 409 整体拦截）。
- 纯函数地基：`client.EncodeChangeXML`（ErrNoXMLEncoder 哨兵）、`diff.Invert`（回滚=ADD↔DELETE 互换/MODIFY 换值，幂等回环）、`xmlcodec.EncodeLeafDelete`（叶级 nc:operation=delete，五重防线）。
- 前端：`stores/changeset`（按设备隔离、upsert 合并保首次 baseline、删待创建=移除、编辑保 create、toRequest 序列化）；工具栏四按钮+徽标+提示条；变更内容/试运行/提交进度三弹窗；三路「确定」入集、删除/批量删除入集+标记合成视图+取消删除；字段级清除=删除意图（基线有值才算，tooltip 按基线区分）；`useConfigSubmit` 物理退役。
- D9 债清偿：YANG length → FieldDef minLength/maxLength → 「合法长度」占位自动生效。
- 草稿为**会话态不持久化**（刷新即丢；离开确认+提示条兜底，D1 诚实边界）。

**Why:** NCE 商用网管的核对-预演-提交闭环；误操作不再直达设备。

**How to apply / 踩坑（后续 PR 交付必读）：**
- **pr-size 不豁免 openspec 文档**：提案三件套+代码同车易超 1000；提案文档拆独立分支先行 PR（R17 本来就该 spec 先落 main）。误推超限内容且禁 force push 时：从 origin/main 摘干净分支重开 PR，主分支 merge 收敛（#246→#247 套路）。
- **worktree 前端环境（升级版口径）**：pre-push e2e 会在 worktree `npm install` 出**真目录** node_modules——之后**不要再 symlink**（对已存在目录 `ln -s` 会在里面造 `node_modules/node_modules` 自引用 → resolveComponent 告警+`reading 'ce'` 崩，象征双 Vue 实例）；真目录直接用即可。`npm run test`（.bin 符号链接入口）在 symlink 目录下也会双实例——症状同上时先查 `ls node_modules/node_modules`。
- **vitest 4 会因 Unhandled Rejection 红整个套件**（578 全过仍 exit≠0）：async watch 回调里未捕获的 reject 是常客；组合式 run() 必须整体 try/catch 兜底（R08 本来要求）。
- **8080 socket hang up**：staging 网络被拆但容器还活着（docker-proxy 半死）→ 单测 fetch 被 RST；`docker ps | grep usmp` 全停再跑。ECONNREFUSED 是正常路径（config 注释有账）。
- gin 1.10 静态段（/config/changeset/…）与 `:ip` 参数段可共存（实验验证）。
- 变更集条目 path **带前导斜杠**（后端锚点前缀匹配），与 REST URL 剥斜杠形态不同——`'/' + configPathFor(...)`。
- 测试 pinia：mount 插件与测试内 useStore 必须**同实例**（beforeEach 建共享 pinia），否则断言到空 store。
- coverage 阈值要用 staging 停机后的干净值定（灌水风险 vitest.config 注释有账）；现值 **86/79/80/87**、后端棘轮 **72.3**。

**后续期待办**：导出/配置项按钮语义；跨设备联合变更集；变更内容弹窗补「确定」钮（现 X 关闭）；试运行差异对比嵌套树呈现（现平铺行）；NCE 三级回滚/网络变更模拟（研究底稿参考来源）；真机 leaf-delete 语义验证（netconfsim 已过，华为真机待窗口）。
