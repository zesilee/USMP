# snd-package — SND 驱动包入库资源（ADDED）

## ADDED Requirements

### Requirement: SP-01 snd 目录为华为 YANG 唯一构建期源

仓库 SHALL 在顶层 `snd/` 目录以原始包结构入库 CE6866P SND 驱动包：`snd/ce6866p-yang/`（YANG 模型 + blacklist.xml + domain.xml）、`snd/resources/`（netconf-driver.xml、CliPassthroughCommands.xml、`i18n/{zh-cn,en-us}/*-res.json`）、`snd/webui/template/`（left-tree.json、template.json）。`snd/ce6866p-yang` SHALL 是华为模型 ygot 生成与 tasknamegen 的唯一 YANG 源；仓库 SHALL NOT 依赖任何 YANG 模型 submodule。包内容 SHALL 视为上游制品：升级 SHALL 以整目录替换 + 重跑生成管线完成，SHALL NOT 手工编辑包内文件（本仓库自有配置除外，如未来的裁剪清单）。

#### Scenario: 生成管线以 snd 为源
- **WHEN** 执行 `make gen-yang`（huawei 包）
- **THEN** SHALL 从 `snd/ce6866p-yang` 解析模型并生成，无需任何 submodule 初始化

#### Scenario: clone 即可构建
- **WHEN** 全新 clone 仓库后执行 `make gen-yang`
- **THEN** SHALL 直接成功（snd 随仓库存在），SHALL NOT 提示 submodule 操作

### Requirement: SP-02 包内资源的消费边界（分期启用）

`snd/` 内资源本期仅入库：`i18n/*-res.json`（键=YANG 数据路径→本地化名称，中英文件集与键集 SHALL 保持对称）、`webui/template/left-tree.json`（双语特性树，65 模块 xpath）、`ce6866p-yang/blacklist.xml`（模块屏蔽清单）SHALL 原样保留供后续能力消费；在对应能力交付前，运行期行为 SHALL NOT 因这些文件存在而改变（R08：不因未消费资源引入新失败路径）。

#### Scenario: 未消费资源不影响运行期
- **WHEN** 服务启动并对外提供 `/yang/modules`、`/yang/schema`
- **THEN** 行为 SHALL 与资源入库前一致（i18n/left-tree/blacklist 均不参与）

#### Scenario: i18n 对称性可校验
- **WHEN** 校验 `snd/resources/i18n` 下 zh-cn 与 en-us
- **THEN** 文件集合 SHALL 一一对应（同名 `*-res.json`）
