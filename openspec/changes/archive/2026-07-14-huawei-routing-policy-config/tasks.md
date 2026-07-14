## 1. 测试设计先行（红灯，T05/T01）

- [x] 1.1 触发 `yang-config-test-design`，产出 rtp policy-definition 标量边界完备矩阵清单
- [x] 1.2 schema 驱动断言：policy-definition 直属 config-true 标量恰好 2（name+address-family-mismatch-deny）、深层 nodes 为推迟容器（红）
- [x] 1.3 描述符谓词表格驱动测试：命中 `/rtp:routing-policy/...`、负路径不误命中、`Lookup` 可达性 ok=true（红）
- [x] 1.4 xmlcodec 往返单测：policy-definition(name+address-family-mismatch-deny) 编→解等值，根 namespace 真值断言（红）
- [x] 1.5 B2 集成测试 `*_integration_test.go`（`testing.Short()` 跳过）：下发→回读→收敛 + 幂等（红）

## 2. 驱动描述符 + reconciler（绿）

- [x] 2.1 `internal/drivers/huawei.go` 新增 rtp namespace 常量 `urn:huawei:yang:huawei-routing-policy`
- [x] 2.2 新增 `driver.Descriptor{Vendor:"huawei", Module:"routing-policy"}`：谓词精确锚定 `/rtp:routing-policy` + 显式 Namespace + SchemaTree 入口 `HuaweiRoutingPolicy_RoutingPolicy`
- [x] 2.3 `internal/controller/routingpolicy` 容器根 reconciler（镜像 xpl，单条整根 MODIFY 收敛，deviceClient.Get 走 DecoderFor + JSON 路径 deviceRoot.RoutingPolicy）
- [x] 2.4 编译通过 + 1.3 谓词/可达性测试转绿

## 3. 通用引擎缺口（仅实测暴露才补）

- [x] 3.1 跑 1.4 往返：容器根编解码**无新缺口**，复用 XC-05/XC-06
- [x] 3.2 1.4 往返测试转绿

## 4. B2 端到端（绿）

- [x] 4.1 netconfsim 模型无关，B2 复用既有 RFC edit-config 通道（零方言，同波次①②，实测收敛）
- [x] 4.2 1.5 B2 集成（下发→回读→收敛 + 幂等）转绿

## 5. 完备矩阵补全与门禁

- [x] 5.1 补齐矩阵剩余项：并发-race、边界/负路径、删除语义（声明式 subset + DELETE 通道推迟债登记）、schema 形状锁死深层 nodes 推迟边界
- [x] 5.2 `go test ./... -race` 全绿；覆盖率不低于基线（T08）
- [x] 5.3 `go-code-review-check` 技能通过（T04）

## 6. 提交与合入

- [x] 6.1 What/Why/How 三段式提交（≤500 行/commit，超限拆描述符+codec / reconciler+B2）
- [x] 6.2 `/opsx:sync`：delta spec → 主 spec
- [x] 6.3 `/opsx:archive`：归档 change
- [ ] 6.4 push + PR（≤1000 行），CI required checks 全绿后合入
