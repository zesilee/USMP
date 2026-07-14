## 1. 测试设计先行（红灯，T05/T01）

- [ ] 1.1 触发 `yang-config-test-design`，产出 xpl route-filter 完备测试矩阵清单（全属性/端到端/并发/边界/幂等/负路径/删除语义）
- [ ] 1.2 schema 驱动断言：route-filter config-true 标量恰好 2（name+content）对照 fixture，计数防遗漏（红）
- [ ] 1.3 描述符谓词表格驱动测试：命中 `/xpl:xpl/...`、负路径不误命中、`Lookup` 注册可达性 ok=true（红）
- [ ] 1.4 xmlcodec 往返单测：route-filter(name+content) 编→解等值，根元素 namespace `urn:huawei:yang:huawei-xpl` 真值断言，边界长度（content 1..16380）（红）
- [ ] 1.5 B2 集成测试 `*_integration_test.go`（`testing.Short()` 跳过）：下发→回读→收敛 + 幂等（红）

## 2. 驱动描述符 + reconciler（绿）

- [ ] 2.1 `internal/drivers/huawei.go` 新增 xpl namespace 常量 `urn:huawei:yang:huawei-xpl`
- [ ] 2.2 新增 `driver.Descriptor{Vendor:"huawei", Module:"xpl"}`：谓词精确锚定 `/xpl:xpl` + 显式 Namespace + SchemaTree 入口 `HuaweiXpl_Xpl`
- [ ] 2.3 `internal/controller/xpl` 容器根 reconciler（镜像 tunnelmgmt，单条整根 MODIFY 收敛，deviceClient.Get 走 DecoderFor + JSON 路径 deviceRoot.Xpl）
- [ ] 2.4 编译通过 + 1.3 谓词/可达性测试转绿

## 3. 通用引擎缺口（仅实测暴露才补）

- [ ] 3.1 跑 1.4 往返：若容器根/大 content 编解码暴露缺口按 TDD 补 xmlcodec delta；否则记录「复用 XC-05/XC-06 无新增」
- [ ] 3.2 1.4 往返测试转绿

## 4. B2 端到端（绿）

- [ ] 4.1 netconfsim 模型无关，B2 复用既有 RFC edit-config 通道（零方言，同波次①）
- [ ] 4.2 1.5 B2 集成（下发→回读→收敛 + 幂等）转绿

## 5. 完备矩阵补全与门禁

- [ ] 5.1 补齐矩阵剩余项：并发-race、边界越界负路径、删除语义（声明式 subset + DELETE 通道推迟债登记）
- [ ] 5.2 `go test ./... -race` 全绿；覆盖率不低于基线（T08，补测后按需上调 `.coverage-baseline`）
- [ ] 5.3 `go-code-review-check` 技能通过（T04）

## 6. 提交与合入

- [ ] 6.1 What/Why/How 三段式提交（≤500 行/commit，原子功能，超限拆描述符+codec / reconciler+B2）
- [ ] 6.2 `/opsx:sync`：delta spec → 主 spec
- [ ] 6.3 `/opsx:archive`：归档 change
- [ ] 6.4 push + PR（≤1000 行），CI required checks 全绿后合入
