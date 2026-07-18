## Why

USMP 要对接的真实设备是 CE 系列数据中心交换机（CE6866P 等），而当前 YANG 基线来自 yang-models submodule 的 NE40E **路由器**模型（8.20.10）——模型与目标设备不匹配（ifm/bgp revision 漂移、CE 上不存在的字段被当作可配属性暴露）。用户已拍板（2026-07-18）：以 /home/leezesi/snd 驱动包（域 `ce6866p-snd`，143 个 YANG + 双语 i18n + left-tree）为**唯一基线**，删除 yang-models 依赖。这是 snd 融合四期计划的第 ①期，后续 ②设备角色+能力协商 ③左树重构 ④中英文 i18n 都建立在本期入库的资源上。

## What Changes

- 新增 `snd/` 入库目录（vendored，非 submodule）：`snd/ce6866p-yang/`（143 个 .yang + blacklist.xml + domain.xml 等）、`snd/resources/`（netconf-driver.xml、CliPassthroughCommands.xml、i18n 中英 res.json 各 93 个）、`snd/webui/template/`（left-tree.json、template.json）。**本期只入库+切生成源，i18n/left-tree/blacklist 不消费**（②③④ 期用）
- **BREAKING** `backend/internal/generated/huawei/gen.conf` 的 `yang_path` 切到 `snd/ce6866p-yang`，重新生成 7 模块结构体：ifm/bgp 降到 CE 基线 revision，15 个在用类型缺字段（VLAN `MemberPort.TagMode`、IFM `Ipv4/Ipv6/NetworkLayerStatus/ProtocolUpDelayTime`、system `Uname` 等）——CE6866P 上本不存在的属性从 schema/表单中消失（基线校正）
- 测试面适配：hwfix 夹具、vlan/ifm 集成测试、netconfsim query/asserts 中引用漂移字段的用例改造（漂移字段零生产逻辑引用，已实测）
- tasknamegen go:generate `-path` 切到 `snd/ce6866p-yang`，重新生成 taskname.gen.go
- **删除 yang-models submodule 及全部 14 处触点**：.gitmodules、gen-yang.sh 校验提示、compliance.yml 3 处（path 触发/变更检测/submodule init）、pre-commit 提示、doc.go/README/DEPLOY-WSL 文档
- CI regen-and-diff 门禁改为对入库 `snd/` 目录生效（不再有 submodule init 步骤，管线反而变简单）

## Capabilities

### New Capabilities

- `snd-package`: snd 驱动包入库资源的结构与消费契约（目录布局、ce6866p-yang 为唯一 YANG 构建期源、i18n/left-tree/blacklist 资源的存在性与格式约定、后续各期的消费接口预留）

### Modified Capabilities

- `yang-codegen-pipeline`: 生成源从 yang-models submodule 改为入库 `snd/ce6866p-yang`；「submodule 未初始化报错」「CI submodule init + regen-and-diff」条款改为入库目录语义（目录必存在，regen-and-diff 直接跑）
- `yang-api`: BR-11 中「ietf/openconfig .yang 仅允许作为构建期编译依赖（yang-models submodule）」的载体表述改为 `snd/ce6866p-yang`（ietf-inet-types/ietf-yang-types/openconfig-telemetry 系仍是华为模型 import 依赖）

## Impact

- 构建管线：gen.conf、scripts/gen-yang.sh、Makefile（不改逻辑只改文档提示）、yangschema go:generate、compliance.yml、pre-commit
- 生成物：`generated/huawei/` 全量 regen（structs 分片 diff 数千行，R04 门禁豁免 review——生成物由 regen-and-diff 守护）；taskname.gen.go regen
- 测试：hwfix、vlan_hardening_integration_test、ifm reconciler_integration_test、netconf_vlan_test、netconfsim query/asserts——漂移字段用例删改；golden 文件按新基线重生成
- 仓库体积：+~10MB（snd 入库）；删 .gitmodules 后 clone 不再需要 submodule
- 风险面：模拟网元（netconfsim）数据树与新 schema 的一致性需集成测试全绿背书；覆盖率棘轮（T08）完成后校验
- 不动：driver 注册表机制、Reconciler 逻辑、前端（本期零前端改动）
