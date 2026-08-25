# Claude Code Hook 配置说明

> 本文档只描述 `.claude/settings.json` 中**实际注册**的钩子。git 层门禁（pre-commit/commit-msg/pre-push/post-checkout）在 `.githooks/`，与本文档无关。
> Claude Code 钩子协议：载荷以 **stdin JSON** 传入（`tool_name`/`tool_input`/`tool_response`）；PreToolUse **exit 2 才阻断**（exit 1 只是非阻断报错）。

## 目录结构
```
.claude/
├── settings.json          # 钩子注册（权限、attribution 也在这）
├── HOOKS.md               # 本文档
└── hooks/
    ├── pre-tool-use.sh    # L2 命令拦截
    ├── post-task-sync.sh  # 任务持久化同步
    └── post-task-sync.log # 同步日志（gitignore）
```

## Hook 列表

### 1. PreToolUse (Bash) — L2 命令拦截 `pre-tool-use.sh`
在 AI 执行 Bash 命令前拦截违规操作，命中即 exit 2 阻断：

| 拦截项 | 红线 |
|--------|------|
| `git push ... main`（非 hotfix） | R13 |
| `git push --force` | W07 |
| 手动编辑 `internal/generated/` | R04（走 `make gen-yang` regen-and-diff） |
| `git checkout main` | W01（用 EnterWorktree） |
| `rm -rf /` | 破坏性命令 |
| 重定向写 `.env` | R16 |

改动脚本后必跑文件末尾注释里的自测命令（历史教训：曾因读不到输入而静默放行数月）。

### 2. PostToolUse (TaskCreate|TaskUpdate|TaskDelete) — 任务同步 `post-task-sync.sh`
把会话内任务变更同步到 `openspec/tasks/*.md`（§12 会话恢复体系）：
- TaskCreate → 生成带 frontmatter 的任务文件（id/title/status/priority/branch/worktree）
- TaskUpdate → 更新对应文件 frontmatter
- TaskDelete → 标记 `status: deleted`
- 失败不阻塞主流程（`set +e`），日志见 `post-task-sync.log`

## 权限配置
见 `settings.json` 的 `permissions.allow`（git status/diff/log、go fmt/vet/test、make 等只读与构建类命令免提示）。

## 提交规范
用 `git-what-why-how-commit` 技能：`<type>: <subject>` + What/Why/How 三段式，自动附 Claude Code Co-Author（settings.json `attribution`）。
