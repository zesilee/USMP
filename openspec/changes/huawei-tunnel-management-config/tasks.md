## 1. 测试设计先行（红灯，T05/T01）

- [ ] 1.1 触发 `yang-config-test-design`，产出 tunnel-management 标量边界完备测试矩阵清单（全属性/端到端/并发/边界/幂等/负路径/删除语义），落到本 change
- [ ] 1.2 schema 驱动断言：枚举 `HuaweiTunnelManagement_TunnelManagement` 标量 config-true leaf（tunnel-policy name/description、tunnel-down-switch enable）对照 fixture，计数防遗漏（红）
- [ ] 1.3 描述符谓词表格驱动测试：`MatchRoute`/`MatchDecode`/`MatchEncode` 命中 `/tunnel-management/...`、负路径不误命中、`Lookup` 注册可达性 ok=true（红）
- [ ] 1.4 xmlcodec 往返单测：tunnel-policy(name+description)+tunnel-down-switch 编→解等值，根元素 namespace `urn:huawei:yang:huawei-tunnel-management` 真值断言，边界长度（name 1..39 / description 1..80）（红）
- [ ] 1.5 B2 集成测试骨架 `*_integration_test.go`（`testing.Short()` 跳过）：下发→回读→收敛 + 幂等（红）

## 2. 驱动描述符（绿）

- [ ] 2.1 `backend/internal/drivers/huawei.go` 新增 tunnel-management namespace 常量 `urn:huawei:yang:huawei-tunnel-management`
- [ ] 2.2 新增 `driver.Descriptor{Vendor:"huawei", Module:"tunnel-management"}`：谓词精确锚定 `tunnel-management` 根 token + 显式 Namespace + SchemaTree 入口闭包返回 `&HuaweiTunnelManagement_TunnelManagement{}`
- [ ] 2.3 编译通过 + 1.3 谓词/可达性测试转绿

## 3. 通用引擎缺口（仅实测暴露才补）

- [ ] 3.1 跑 1.4 往返：若容器根 Encode/Decode 或 namespace 暴露缺口，按 TDD 补 `pkg/yang-runtime/xmlcodec` delta 并回填 proposal Modified Capabilities；否则记录「复用 XC-05/XC-06 无新增」
- [ ] 3.2 1.4 往返测试转绿

## 4. 模拟网元方言与 B2（绿）

- [ ] 4.1 `simulator/netconfsim` 增加 tunnel-management edit-config（整树替换）/get-config 方言，Decode 锚定 `<tunnel-management>` 顶层容器
- [ ] 4.2 1.5 B2 集成（下发→回读→收敛 + 幂等）转绿

## 5. 完备矩阵补全与门禁

- [ ] 5.1 补齐 1.1 矩阵剩余项：并发-race、边界越界负路径、删除语义（声明式 subset + DELETE 通道推迟债登记）
- [ ] 5.2 `go test ./... -race` 全绿；覆盖率不低于基线（T08，补测后按需上调 `.coverage-baseline`）
- [ ] 5.3 `go-code-review-check` 技能通过（T04）

## 6. 提交与合入

- [ ] 6.1 What/Why/How 三段式提交（≤500 行/commit，原子功能）
- [ ] 6.2 `/opsx:sync`：delta spec → 主 spec
- [ ] 6.3 `/opsx:archive`：归档 change
- [ ] 6.4 `finishing-a-development-branch`：push + PR（≤1000 行），CI required checks 全绿后合入
