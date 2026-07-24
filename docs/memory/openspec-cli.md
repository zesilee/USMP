---
name: openspec-cli
description: OpenSpec CLI 正确包名是 @fission-ai/openspec（裸 openspec 是空壳占位）；需 Node≥20，系统当前 v18 有警告但能跑
metadata: 
  node_type: memory
  type: reference
  originSessionId: 7bbe5a84-45aa-4158-85b4-61c82d05eb0f
---

USMP 用 OpenSpec 做设计/需求管理。CLI 安装要点：

- **正确包名**：`@fission-ai/openspec`（`npm i -g @fission-ai/openspec`，提供 `openspec` 命令）。**裸 `openspec` 包是 `0.0.0` 空壳占位,勿装。**
- 已全局装于 `/usr/local/bin/openspec`（v1.5.0）。
- **Node**：CLI 要求 `^20.20 || >=22.22`,系统当前 v18.19.1 → 有 EBADENGINE 警告,但 `--help`/`list`/`change` 实测可用。
- CLI 认得本仓库现有 `openspec/specs` + `openspec/changes` 布局(无需 `openspec init`)。仓库另有自定义 `openspec/.openspec.yaml`(非官方格式,手写约定)。
- **注意**：官方 CLI 按 `#### Scenario:` 固定小标题解析 spec 的需求;本仓库手写 specs 用 `### 行为契约 + Given-When-Then` 风格,故 `openspec list --specs` 显示 requirements 0(非报错)。要让 CLI 统计需求需对齐小标题格式。
- 本仓库没有 `openspec new`/`instructions` 用的 templates 目录;`/opsx:propose` 等 skill 假设官方 CLI 流程,实际是手写三件套。相关：[[dual-stack-migration]]。
