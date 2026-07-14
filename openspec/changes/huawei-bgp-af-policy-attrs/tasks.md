## 1. 测试设计先行（红灯，T05/T01）

- [ ] 1.1 触发 `yang-config-test-design`，产出 AF import-filter-policy 策略属性完备矩阵清单
- [ ] 1.2 `internal/drivers` 编码断言：af[ipv4-unicast]/ipv4-unicast/import-filter-policy 的 acl-name-or-num+filter-name+filter-parameter 经 ni 描述符编码含值 + `<bgp xmlns=huawei-bgp>` + 命中 ni（非新描述符）（红）
- [ ] 1.3 B2 集成 `*_integration_test.go`（`testing.Short()` 跳过）：经 ni reconciler 下发含 AF 策略属性 → 回读 → 收敛 + 幂等（红）
- [ ] 1.4 负路径/并发：空 import-filter-policy 跳过不 panic；`-race` 并发编码（红/骨架）

## 2. 经既有链路转绿（预期零生产代码）

- [ ] 2.1 跑 1.2/1.3：经既有 ni 描述符 + XC-06 namespace 即绿（sanity 已证）；若暴露缺口按 TDD 补并回填 proposal Modified
- [ ] 2.2 编码断言 + B2 转绿

## 3. 完备矩阵补全与门禁

- [ ] 3.1 补齐矩阵：幂等、并发-race、负路径、leafref 编排依赖登记（目标先配，spec AFPOL-02）
- [ ] 3.2 `go test ./... -race` 全绿；覆盖率不低于基线（T08）
- [ ] 3.3 `go-code-review-check` 通过（T04）

## 4. 提交与合入（BGP 2b 收官）

- [ ] 4.1 What/Why/How 三段式提交（≤500 行/commit）
- [ ] 4.2 `/opsx:sync`：AFPOL delta → 主 spec
- [ ] 4.3 `/opsx:archive`：归档 change
- [ ] 4.4 push + PR，CI 全绿后合入 → BGP 2b 完成
