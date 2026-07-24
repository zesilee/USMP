---
name: spec-format-standard
description: spec 格式统一到 OpenSpec CLI 标准 + spec-first 军规；改 spec/写 spec 前必读
metadata: 
  node_type: memory
  type: feedback
  originSessionId: 014f7a79-ea21-4958-af4f-6af2e70396f4
---

用户 2026-07-07 决定统一 spec 格式与规范。

**方向**：主 spec 全部对齐 **OpenSpec CLI 标准**格式——`## Purpose` + `## Requirements` + `### Requirement: <SHALL 陈述>` + `#### Scenario: <名>`(WHEN/THEN)。原自定义中文格式(`## 能力概述`/`## 行为契约`/`### XX-NN`；config-api/devices-api 是 `## 接口定义`/`## 业务规则`)废弃。XX-NN 编号保留作 `### Requirement:` 前缀留痕。delta spec 已是 CLI 格式(`## ADDED/MODIFIED/REMOVED Requirements`)——新增需求用 ADDED（不是 MODIFIED，否则 archive 找不到同名 requirement 会失败）。

**为什么要做**：既有 12 个主 spec 用自定义格式，CLI 只认标准格式 → `openspec archive/validate --specs` 对主 spec 全失败(✗)，sync/archive 走不通(device-store change 归档就卡在这)。

**交付**：① `openspec/SPEC_CONVENTIONS.md` 规范文档 ② 填充空的 `openspec/templates/spec.md`(+design/tasks) ③ CI 加 `openspec validate <cap> --type spec` **只卡本次改动的主 spec**(未迁移的过渡期不卡，避免全红) ④ **渐进迁移**：逐能力一个 PR(禁一次性重写 12 个)。**迁移时借机修正 spec 与代码不一致处**(非忠实照搬)。

**Why / How to apply**：
- **新军规(已入 CLAUDE.md)**：**刷 spec 先于开发**——代码变更若需刷新 spec，必须**先改 spec 再启动开发**，禁止先写代码后补 spec。
- CI 校验只针对改动的 spec，不用 `--all`。
- 校验命令：`openspec validate <capability> --type spec`（单个）；`openspec validate --specs`（全部，迁移完才会全绿）。

相关：[[openspec-cli]] [[reconcile-conninfo-debt]]
