---
description: 任务断点续做 — 查看、恢复、同步、归档 openspec/tasks/ 下的持久化任务
---

# 任务命令（Task Command）

管理 `openspec/tasks/` 下的持久化任务，支持跨会话和跨机器续做。

## 子命令

用户输入 `/task` 后跟子命令，你**必须**根据子命令执行对应操作：

### `/task list` — 列出所有任务

1. 扫描 `openspec/tasks/` 下所有 `.md` 文件（排除 `archive/` 目录）
2. 解析每个文件的 YAML frontmatter，提取 `id`、`title`、`status`、`priority`、`branch`
3. 按状态分组显示，顺序：`in_progress` → `blocked` → `pending` → `completed` → `deleted`
4. 输出格式：

```
📋 任务列表 (openspec/tasks/)

🔴 进行中 (1):
  • add-gnmi-subscription [high] — feature/gnmi-subscription

🟡 待处理 (2):
  • fix-cache-race [medium]
  • update-api-docs [low]

✅ 已完成 (0)

归档: /task archive <slug>
恢复: /task resume <slug>
```

### `/task resume <id-or-slug>` — 恢复任务到当前会话

1. 根据 id 或 slug（文件名中的日期后部分）定位文件
2. 读取文件，解析 frontmatter
3. 使用 TaskCreate 工具恢复任务到当前会话：
   - `subject`: 使用 frontmatter 中的 `title`
   - `description`: 使用文件正文中的"目标"和"上下文恢复提示"部分
   - `status`: 使用 frontmatter 中的 `status`
4. 读取"恢复指令"部分，按步骤执行上下文恢复：
   - 如有 `worktree` 记录且目录存在 → 建议用户 `EnterWorktree`
   - 如有 `branch` 记录且当前不在该分支 → 建议用户 `git checkout`
   - 如有 `plan` 记录且文件存在 → 读取计划文件了解完整上下文
5. 打印恢复摘要：

```
✅ 任务已恢复: 添加 gNMI 订阅支持
   状态: in_progress | 优先级: high
   分支: feature/gnmi-subscription
   Worktree: .claude/worktrees/gnmi-sub
   
   📌 下一步: 完成 gnmi_subscribe.go 的 reconnect 逻辑
```

### `/task sync` — 双向同步任务状态

1. 获取当前会话中所有内置 Task（使用 TaskList）
2. 扫描 `openspec/tasks/` 下所有文件
3. 对于每个文件中的任务：
   - 如果内置 Task 中存在同 id → 以内置 Task 状态更新文件
   - 如果内置 Task 中不存在 → 使用 TaskCreate 从文件恢复到内置
4. 对于每个内置 Task：
   - 如果文件中不存在同 id → 创建新文件
5. 输出同步结果摘要

### `/task archive <id-or-slug>` — 归档已完成任务

1. 定位任务文件
2. 验证 `status` 为 `completed`（非已完成则警告）
3. 将文件移到 `openspec/tasks/archive/`
4. 如果内置 Task 中存在该任务，使用 TaskUpdate 标记 `status: completed`
5. 输出确认

### `/task status` — 当前会话任务概览

1. 使用 TaskList 获取当前会话任务
2. 扫描 `openspec/tasks/` 获取持久化任务
3. 对比两者，标出差异：
   - 仅在文件中 → 需要resume
   - 仅在内置 → 需要sync
   - 两边都有但状态不同 → 需要sync

## 无参数时的行为

如果用户只输入 `/task` 不带子命令，等同于 `/task status`。

## 重要提示

- 所有文件操作路径基于项目根目录
- 文件名 slug 为 kebab-case，来自 frontmatter title
- frontmatter 中 `status` 字段为唯一权威状态源
- `archive/` 下的文件仅做历史记录，不参与 list/sync
