---
id: crd-to-yang-codegen
title: 北向 CRD→YANG 反向代码生成工具（crd2yang，crdgen 镜像）
status: in_progress
priority: medium
branch: worktree-crd-to-yang-codegen
worktree: .claude/worktrees/crd-to-yang-codegen
change: openspec/changes/crd-to-yang-codegen（C2Y-01~06）
updated: 2026-08-04
origin: 用户 2026-08-04 提出：北向「网络自动化服务」（K8s）以 CRD 发布业务契约，USMP 代码生成工具需增 CRD→YANG 能力，生成的 YANG 作为业务网络配置；不用仓库现有 CRD，自建对齐华为 YANG 规范的示例 CRD 验证
---

## 上下文恢复提示

- 工作方式：**所有代码检视都启独立 agent**（用户 2026-08-04 拍板，本任务已执行）。
- 范围拍板：只做转换工具，不照 refactor-by-crd.md 整体重构（Scrapligo/前端直连 K8s/Stack A 旧 Kind 三处撞红线，均剪出）；示例模型不接线运行期 load.go（业务控制台单 Kind 写死，挂菜单会出死页面）。
- 进度：任务组 1（工具+示例 CRD+golden+负路径+race）代码完成，独立 agent 检视出 4 个必须修复项（int64 溢出腐蚀 range / 属性名未做 YANG 标识符校验 / 跨类型约束静默丢弃 / guard 漏 PatternProperties·Dependencies·Definitions），修复中；随后按 tasks.md 组 2（往返对账）→组 3（businessdemo ygot+fixture+前端黄金）→组 4（Makefile+CI 门禁）→组 5（收尾）。
- 提交拆分：commit-msg 500 行限制，实现已拆 crd2yang.go（解析/元信息）+ emit.go（渲染/映射），分 3 commit 入库。
- ⚠️ 主仓库工作区留有 openspec/changes/crd-to-yang-codegen 与本文件的未跟踪副本（为跨会话可见），**PR 合并回 main 前须先删掉主仓库副本再 pull**，否则 checkout 冲突。

## 恢复指令

1. `cd /home/leezesi/USMP/.claude/worktrees/crd-to-yang-codegen`
2. 看 `openspec/changes/crd-to-yang-codegen/tasks.md` 勾选进度
3. `cd backend && go test ./tools/crd2yang/ -race` 确认现状
