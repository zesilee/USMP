## 1. 测试设计先行（红灯，T05/T01）

- [x] 1.1 触发 `yang-config-test-design`，产出 acl group/group6 标量+枚举边界完备矩阵清单
- [x] 1.2 schema 驱动断言：group/group6 直属 config-true leaf（标量 + 枚举 type/match-order）覆盖本波次字段集、深层 rule-* 为推迟容器（红）
- [x] 1.3 描述符谓词表格驱动测试：命中 `/acl:acl/...`、负路径不误命中、`Lookup` 可达性 ok=true（红）
- [x] 1.4 xmlcodec 往返单测：group(identity+type enum+标量)+group6 编→解等值，枚举值域名真值断言，根 namespace（红）
- [x] 1.5 B2 集成测试 `*_integration_test.go`（`testing.Short()` 跳过）：下发→回读→收敛（含 enum 真值）+ 幂等（红）

## 2. 驱动描述符 + reconciler（绿）

- [x] 2.1 `internal/drivers/huawei.go` 新增 acl namespace 常量 `urn:huawei:yang:huawei-acl`
- [x] 2.2 新增 `driver.Descriptor{Vendor:"huawei", Module:"acl"}`：谓词精确锚定 `/acl:acl` + 显式 Namespace + SchemaTree 入口 `HuaweiAcl_Acl`
- [x] 2.3 `internal/controller/acl` 容器根 reconciler（镜像 routingpolicy，单条整根 MODIFY 收敛，deviceClient.Get 走 DecoderFor + JSON 路径 deviceRoot.Acl）
- [x] 2.4 编译通过 + 1.3 谓词/可达性测试转绿

## 3. 通用引擎缺口（仅实测暴露才补）

- [x] 3.1 跑 1.4 往返：枚举 leaf(type/match-order)编解码**暴露平台级缺口**——引擎发整数非值域名，已拆独立 change xmlcodec-enum-name(XC-08,#158)先修并合入 main，acl 复用之
- [x] 3.2 1.4 往返测试转绿

## 4. B2 端到端（绿）

- [x] 4.1 netconfsim 模型无关，B2 复用既有 RFC edit-config 通道（零方言，同波次①②③；枚举名回读由 #158 query.go enumInt 提供）
- [x] 4.2 1.5 B2 集成（下发→回读→收敛含 enum + 幂等）转绿

## 5. 完备矩阵补全与门禁

- [x] 5.1 补齐矩阵剩余项：枚举往返、并发-race、边界/负路径（缺 mandatory type）、删除语义、schema 形状锁死深层 rule-* 推迟边界
- [x] 5.2 `go test ./... -race` 全绿；覆盖率不低于基线（T08）
- [x] 5.3 `go-code-review-check` 技能通过（T04）

## 6. 提交与合入

- [x] 6.1 What/Why/How 三段式提交（≤500 行/commit，超限拆描述符+codec / reconciler+B2）
- [x] 6.2 `/opsx:sync`：delta spec → 主 spec
- [x] 6.3 `/opsx:archive`：归档 change
- [ ] 6.4 push + PR（≤1000 行），CI required checks 全绿后合入
