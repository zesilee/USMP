## 1. 基线与红灯（TDD）

- [x] 1.1 worktree 内基线 `go test -race ./pkg/yang-runtime/xmlcodec/... ./internal/drivers/... ./internal/controller/networkinstance/...` 全绿
- [x] 1.2 红灯先行：写「peer 完备枚举计数 + 下发 namespace 真值 + af-type 枚举 key 往返」失败用例（对照 BN-01/02/03）

## 2. 编解码验证（B1，缺口则补）

- [x] 2.1 peer 往返真值单测：peer config-true 标量 + timer/graceful-restart/bfd-parameter，经 ni Spec 编码（XC-06 namespace）→解码→DeepEqual
- [x] 2.2 完备枚举 + 计数断言：schema 驱动枚举 peer 子树 config-true 标量（config 继承 + 排除策略/状态清单），计数锁定（BN-01）
- [x] 2.3 af-type 枚举 key 往返：list-under-list + 枚举 key 编解码（BN-02）——若暴露缺口进 2b'
- [x] 2.4 namespace 真值断言：`<bgp xmlns="…huawei-bgp">`、peer/address/remote-as 继承、ni 原生不另发（BN-01）

## 2b'. （条件）xmlcodec 深层嵌套/枚举 key 缺口修复

- [x] 2b'.1 **仅当 2.1/2.3 暴露缺口**：补 `pkg/yang-runtime/xmlcodec` + golden/往返，若触发补 yang-xml-codec delta

## 3. B2 集成（下发→回读→收敛）

- [x] 3.1 `internal/controller/networkinstance` 集成用例：ni + instance[_public_] + peers（+timer/graceful-restart）→ 下发→回读→二次收敛 Changes==0；断言下发 XML namespace 真值（BN-01）
- [x] 3.2 幂等：连续两次 reconcile 第二次无 change（BN-03）
- [x] 3.3 确认既有 ni 描述符/reconciler/main.go 无需改动（复用链路，D1）——若需改动记录并补测

## 4. 完备矩阵补齐（BN-03/BN-04）

- [x] 4.1 并发-race：多协程并发含 peers 的 reconcile 无竞态无 panic
- [x] 4.2 边界与负路径：peer 无 remote-as → ΛValidate 拒绝；address/域越界；下发失败缓存不更新
- [x] 4.3 负路径：策略子容器/config-false 状态字段不出现在下发报文（BN-04 防越序）
- [x] 4.4 声明式删除 subset 契约：移除 peer 不下发删除（BN-04）
- [x] 4.5 覆盖率不下降（T08），达标上调基线

## 5. 评审与提交

- [ ] 5.1 `go-code-review-check` 通过
- [ ] 5.2 `git-what-why-how-commit` 三段式，≤500 行/commit（测试为主，按功能拆）
- [ ] 5.3 push + PR（CI required checks 全绿，PR ≤1000 行）

## 6. sync + archive（合入后）

- [ ] 6.1 `/opsx:sync`：BN-01~04 合入主 spec（新建 `openspec/specs/huawei-bgp-neighbor-config`；若 2b' 触发同步 yang-xml-codec）
- [ ] 6.2 `/opsx:archive`
- [ ] 6.3 更新记忆 `bgp-onboarding-deps`：2a 基础邻居已交付，2b 策略属性门控 routing-policy/xpl/acl/tunnel-management
