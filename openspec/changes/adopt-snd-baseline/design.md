## Context

当前华为 YANG 构建期源 = yang-models submodule（NE40E 8.20.10，仅构建期、运行期零依赖，生成物入库）。snd 包（/home/leezesi/snd）是 NCE 风格 CE6866P 驱动包。已实测（2026-07-18 探索）：

- ygot 对 snd 目录生成现有 7 模块**一次通过**（导入闭包完整：ietf-inet-types/ietf-yang-types + openconfig-telemetry 系共 6 个非 huawei 文件）
- 类型面：171 个现有类型消失，业务代码仅 1 处引用（hwfix `E_HuaweiIfm_NetworkLayerState`）
- 字段面：108 个在用类型中 15 个缺字段，使用点全在 `hwfix`、`vlan_hardening_integration_test`、`ifm reconciler_integration_test`、`netconf_vlan_test`、`netconfsim/{query,asserts}`——零生产 Reconciler/API 逻辑
- 4 个管线依赖 extension（task-name/support-filter/operation-exclude/dynamic-default）snd 全有
- yang-models 依赖触点 14 处（gen.conf、gen-yang.sh、go:generate、compliance.yml×3、pre-commit、.gitmodules、文档×4、spec×2）

## Goals / Non-Goals

**Goals:**
- `snd/` 整包入库为仓库顶层目录，`snd/ce6866p-yang` 成为华为模型唯一构建期源
- 7 模块生成物 + taskname.gen.go 按新基线 regen，测试全绿
- yang-models submodule 物理删除，14 触点全清
- spec 契约同步（yang-codegen-pipeline / yang-api / 新增 snd-package）

**Non-Goals:**
- 不消费 i18n res.json、left-tree.json、blacklist.xml、netconf-driver.xml（②③④ 期）
- 不新增生成模块（65 模块渐进生成属 ③ 期后续批次）
- 不动 Device CRD / 能力协商（② 期）
- 不做前端任何改动

## Decisions

**D1：snd 放仓库顶层 `snd/` 原样入库（保持包目录结构），而非拆散进 backend/internal/yang/。** 理由：snd 是一个有自身语义的驱动包整体（domain.xml/template.json 标识包身份），原样保留使后续升级（换新版 snd 包）成为整目录替换 + regen；backend/internal/yang/models 继续只放 usmp 业务模型，两者职责不混。备选「按资源类型拆散」被否：破坏包完整性，升级时要人工重新分拣。

**D2：漂移字段的测试改造 = 删除断言而非改模拟器造假。** CE 基线上不存在的字段（TagMode/Ipv4…），netconfsim 数据树与断言一并删除对应节点，不保留「模拟器有但模型没有」的僵尸数据。golden 文件重生成后人工 diff 抽查关键模块（vlan/ifm）确认只有预期字段消失。

**D3：blacklist.xml 本期入库不生效。** 黑名单里含 huawei-system（现有 system 控制器在用）与 huawei-xpl（BGP 2b 波次目标），语义是 NCE 对特定 revision 的已知问题屏蔽，是否/如何映射到 USMP 的模块可见性属 ② 期能力协商设计，本期不预设。

**D4：compliance CI 的 regen-and-diff 保留且简化。** 删 submodule init 步骤；path 触发器 `yang-models` 改 `snd/ce6866p-yang`；变更检测 grep 同步。gen-yang.sh 的目录存在性校验保留，提示语从「git submodule update --init」改为「snd 目录应随仓库存在」。

**D5：openconfig-telemetry 系 4 文件随包入库但不生成、不进 schema 树。** 它们是 left-tree 中 telemetry 特性的模型与 import 依赖，BR-11 允许「构建期编译依赖」存在；是否为其生成 structs 由 ③ 期渐进批次按需决定。BR-11 的 vendor 边界（运行期仅 huawei/usmp）不变。

**D6：单 change 拆 2-3 个 PR 交付。** PR-1：snd 入库 + gen.conf/tasknamegen 切源 + regen + 测试适配（核心，行数主要是生成物与 vendored yang，二者均在 pr-size 排除/豁免口径内需验证——vendored snd 不在现有排除清单，需在 PR-1 中把 `snd/**` 加入 pr-size 排除并在 commit 说明，属门禁配置变更）；PR-2：删 submodule + 14 触点清理 + 文档；（如超限再拆 PR-3）。

## Risks / Trade-offs

- [snd/** 体积 ~10MB 打爆 pr-size] → D6：把 `snd/**` 加入 pr-size 排除清单（vendored 第三方基线与 generated 同理不算手写代码），随 PR-1 一并提交并在 PR 描述明示
- [netconfsim 数据树与新 schema 不一致导致集成测试隐性漂移] → B2 全量集成测试 + golden 重生成后人工抽查 vlan/ifm
- [tasknamegen 对 CE 版模型的 task-name 覆盖不同（分组变化）] → regen 后 diff taskname.gen.go，如 category 键变化同步前端断言（menu/Sidebar 测试用真实模块名）
- [未来需要 NE 路由器支持] → gen-yang 管线目录驱动，加 `snd-ne40e/` 类目录 + gen.conf 即可恢复，本期删除不封路
- [ifm/bgp 字段面缩小影响已交付表单功能] → 消失字段在 CE 设备上本就下发不了（实测使用点全在测试面），属纠错；PR 描述中列全 15 类型清单供 review

## Migration Plan

worktree 隔离，顺序：spec delta 先行（R17）→ snd 入库 + pr-size 排除 → gen.conf/tasknamegen 切源 regen → 红灯收集（编译错/测试挂全量清单）→ 测试面适配到全绿 → 删 submodule + 触点 → 全量验证（build/test -race/regen-and-diff/覆盖率棘轮）。回滚 = git revert（生成物与源同库原子回退）。

## Open Questions

（无——三项用户拍板已齐，技术未知点已实测。）
