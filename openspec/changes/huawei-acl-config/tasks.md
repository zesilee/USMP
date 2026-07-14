## 1. 测试设计先行（红灯，T05/T01）

- [ ] 1.1 触发 `yang-config-test-design`，产出 acl group/group6 标量+枚举边界完备矩阵清单
- [ ] 1.2 schema 驱动断言：group/group6 直属 config-true leaf（标量 + 枚举 type/match-order）覆盖本波次字段集、深层 rule-* 为推迟容器（红）
- [ ] 1.3 描述符谓词表格驱动测试：命中 `/acl:acl/...`、负路径不误命中、`Lookup` 可达性 ok=true（红）
- [ ] 1.4 xmlcodec 往返单测：group(identity+type enum+标量)+group6 编→解等值，枚举值域名真值断言，根 namespace（红）
- [ ] 1.5 B2 集成测试 `*_integration_test.go`（`testing.Short()` 跳过）：下发→回读→收敛（含 enum 真值）+ 幂等（红）

## 2. 驱动描述符 + reconciler（绿）

- [ ] 2.1 `internal/drivers/huawei.go` 新增 acl namespace 常量 `urn:huawei:yang:huawei-acl`
- [ ] 2.2 新增 `driver.Descriptor{Vendor:"huawei", Module:"acl"}`：谓词精确锚定 `/acl:acl` + 显式 Namespace + SchemaTree 入口 `HuaweiAcl_Acl`
- [ ] 2.3 `internal/controller/acl` 容器根 reconciler（镜像 routingpolicy，单条整根 MODIFY 收敛，deviceClient.Get 走 DecoderFor + JSON 路径 deviceRoot.Acl）
- [ ] 2.4 编译通过 + 1.3 谓词/可达性测试转绿

## 3. 通用引擎缺口（仅实测暴露才补）

- [ ] 3.1 跑 1.4 往返：验证枚举 leaf（type/match-order）编解码；若容器根/枚举暴露缺口按 TDD 补 xmlcodec delta；否则记录「复用既有引擎无新增」
- [ ] 3.2 1.4 往返测试转绿

## 4. B2 端到端（绿）

- [ ] 4.1 netconfsim 模型无关，B2 复用既有 RFC edit-config 通道（零方言，同波次①②③）
- [ ] 4.2 1.5 B2 集成（下发→回读→收敛含 enum + 幂等）转绿

## 5. 完备矩阵补全与门禁

- [ ] 5.1 补齐矩阵剩余项：枚举往返、并发-race、边界/负路径（缺 mandatory type）、删除语义、schema 形状锁死深层 rule-* 推迟边界
- [ ] 5.2 `go test ./... -race` 全绿；覆盖率不低于基线（T08）
- [ ] 5.3 `go-code-review-check` 技能通过（T04）

## 6. 提交与合入

- [ ] 6.1 What/Why/How 三段式提交（≤500 行/commit，超限拆描述符+codec / reconciler+B2）
- [ ] 6.2 `/opsx:sync`：delta spec → 主 spec
- [ ] 6.3 `/opsx:archive`：归档 change
- [ ] 6.4 push + PR（≤1000 行），CI required checks 全绿后合入
