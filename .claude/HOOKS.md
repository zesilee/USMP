# Hook 配置说明

## 目录结构
```
.claude/
├── settings.json          # 主配置文件（简洁版）
├── HOOKS.md               # 本文档（详细说明）
├── hooks/
│   ├── pre-commit-review.md    # 提交前代码检视流程
│   └── post-test-review.md     # 测试后代码检视流程
└── ...
```

## Hook 列表

### 1. Stop Hook - 提交前强制代码检视
**触发时机**：会话停止时

**执行流程**：
1. 检查 Git 状态，确认是否有未提交的 Go 代码变更
2. 如有变更，使用 `go-code-review-check` 技能进行全面检视
3. 生成检视报告（问题清单、严重程度、修改建议）
4. 检视不通过 → 自动修复 → 重新检视（循环直到通过）
5. 检视通过 → 使用 `git-what-why-how-commit` 规范自动提交

**检视项**：
- 并发安全（goroutine 泄漏、锁使用正确性）
- ygot 类型安全（类型转换、空指针）
- 内存缓存安全（TTL、LRU、并发访问）
- NETCONF 异常处理（重连、超时、错误恢复）
- 架构合规性（符合 yang-controller-runtime 规范）
- 代码规范（命名、注释、错误处理）

---

### 2. PostToolUse Hook - 测试后代码检视
**触发时机**：执行 `go test` 命令后

**执行流程**：
1. 列出所有变更的 `.go` 文件
2. 使用 `go-code-review-check` 技能进行检视
3. 输出检视报告
4. 存在严重问题 → 标记并列出需修复项
5. 检视通过 → 标记可安全提交

---

## 权限配置
允许的命令：
- `Bash(git status *)`
- `Bash(git diff *)`
- `Bash(git log *)`
- `Bash(go fmt *)`
- `Bash(go vet *)`

---

## 提交规范
使用 `git-what-why-how-commit` 三段式：
- **What**：做了什么
- **Why**：为什么做
- **Impact**：影响范围

自动添加 Claude Code 作为 Co-Author。
