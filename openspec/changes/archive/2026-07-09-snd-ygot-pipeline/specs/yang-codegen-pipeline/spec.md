# yang-codegen-pipeline — ygot 生成管线（delta）

## ADDED Requirements

### Requirement: CG-01 厂商 manifest 驱动的可复现生成

系统 SHALL 提供 `make gen-yang` 生成入口：扫描 `backend/internal/generated/*/gen.conf`（每厂商包一份声明式生成配置：YANG 模型路径、模块列表、fakeroot/compress 选项），对每包执行 ygot generator（版本由 go.mod 锁定）→ 跨平台后处理 → gofmt，输出该包唯一生成物 `all.gen.go`。`make gen-yang VENDOR=<pkg>` SHALL 仅重生成指定包。新增厂商 SHALL 只需新增目录 + `gen.conf`，零脚本/Makefile 改动。管线 SHALL 可复现且机器无关：同一仓库状态下重复执行输出字节一致，生成物 SHALL NOT 包含生成机器特定内容（如生成器绝对路径头部注释——由后处理规范化）。

#### Scenario: 全量重生成零漂移
- **WHEN** 在干净工作区执行 `make gen-yang`
- **THEN** `git diff backend/internal/generated/` SHALL 为空（生成物与仓库一致）

#### Scenario: 单厂商重生成
- **WHEN** 执行 `make gen-yang VENDOR=huawei`
- **THEN** SHALL 仅重生成 `backend/internal/generated/huawei/all.gen.go`，其他包不动

#### Scenario: submodule 未初始化时可操作报错
- **WHEN** `gen.conf` 的 `yang_path` 目录不存在或为空（yang-models submodule 未初始化）时执行 `make gen-yang`
- **THEN** SHALL 以非零码退出，并输出含 `git submodule update --init yang-models` 的修复指引，SHALL NOT 产生半成品输出

### Requirement: CG-02 跨平台枚举标识符后处理

生成管线 SHALL 使用 Go 实现的后处理器（`scripts/genfix`）修复 ygot 生成的含非法字符 `|` 的枚举标识符（如 `PortType_50|100GE` → `PortType_50_OR_100GE`），SHALL 在 Linux 与 macOS 上行为一致（不依赖平台 sed 方言）。修复 SHALL 幂等：对已修复或无匹配的输入执行为 no-op；SHALL NOT 改动枚举标识符之外的内容（含 YANG 原值字符串映射）。

#### Scenario: 非法字符修复
- **WHEN** 生成物包含标识符 `HuaweiIfm_PortType_50|100GE`
- **THEN** 后处理后 SHALL 为 `HuaweiIfm_PortType_50_OR_100GE`，且对应 YANG 原值映射字符串（如 `"50|100GE"`）SHALL 保持原样

#### Scenario: 幂等 no-op
- **WHEN** 对同一文件执行后处理两次
- **THEN** 第二次执行 SHALL 不产生任何变更

### Requirement: CG-03 生成物漂移 CI 门禁（R04 可验证形态）

CI SHALL 以 regen-and-diff 验证生成物：当 PR 变更触及 `backend/internal/generated/**`、生成脚本/后处理器或 yang-models submodule 指针时，SHALL 重跑 `make gen-yang` 并断言 `git diff --exit-code backend/internal/generated/` 为空——生成物改动合法当且仅当可由管线复现（取代无条件冻结 `generated/` 的旧检查）。未触及上述路径的 PR SHALL 跳过该验证。本地 pre-commit 钩子 SHALL 以同口径对称拦截（T09）：暂存触及生成物/manifest（纯文档除外）时本地 regen + diff 校验。

#### Scenario: 手改生成物被拦截
- **WHEN** PR 直接手工编辑 `all.gen.go` 而未经管线生成
- **THEN** CI regen-and-diff SHALL fail

#### Scenario: 管线产物合法通过
- **WHEN** PR 通过修改 `gen.conf` 并执行 `make gen-yang` 提交生成物变更
- **THEN** CI regen-and-diff SHALL pass

#### Scenario: 无关 PR 跳过
- **WHEN** PR 未触及生成物、生成脚本与 yang-models 指针
- **THEN** SHALL 跳过 regen 验证（不消耗生成耗时）
