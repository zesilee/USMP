# Tasks — device-role-capability

> spec delta 已先行（R17）。worktree 隔离。目标单 PR（手写面超 1000 行则按 design 拆后端/前端两 PR）。

## 1. 前置：模拟网元 hello 注入（D5）

- [ ] 1.1 netconfsim 增 `SetHelloCapabilities([]string)`（默认不变，零回归）；B1 单测：注入后 hello 报文携带定制能力集

## 2. 能力协商链路（CN-01/CN-02）

- [ ] 2.1 红灯：B1 表格驱动（含 race）——按 DeviceID 查能力、重连刷新；B2 集成——sim 只声明 vlan/ifm → `?device=` 返回子集 + `negotiated:true`；离线降级全量 + `negotiated:false`；未注册 404
- [ ] 2.2 实现：ClientPool/连接层能力缓存暴露（CN-01）；`yang_handler.go` `ListModules` 支持 `device` 参数（CN-02 + BR-12）
- [ ] 2.3 门禁：`go test ./internal/api/... ./pkg/yang-runtime/client/... -race` 全绿

## 3. blacklist 注解（CN-03）

- [ ] 3.1 `tools/blacklistgen`（tasknamegen 同模式）→ `yangschema/blacklist.gen.go`；go:generate 接线；B1：解析/匹配（模块名+revision）表格驱动
- [ ] 3.2 `ListModules` 模块项附 `blacklisted:true`（omitempty）；B3 断言 system 命中注解且仍在列表
- [ ] 3.3 regen-and-diff 口径确认（生成物入库、R04 门禁覆盖）

## 4. 设备角色（BR-14）

- [ ] 4.1 红灯：B3——注册带 role 透传/CRD 落库、非法 role 400、缺省 omitempty；device store CRD 后端往返
- [ ] 4.2 实现：`device_types.go` +`spec.role`（校验 marker）→ `make gen-crd` 重生成 CRD yaml；`device_handler.go` 透传
- [ ] 4.3 前端：`stores/device.ts` +role（F1）；设备页 role 列 + 表单字段（常用值提示下拉+可自由输入，F2 含校验错误态）；`npm test` 全绿

## 5. 收官

- [ ] 5.1 全量 `go test ./... -race` + 前端全绿 + 覆盖率棘轮校验（留 0.1 余量）
- [ ] 5.2 `go-code-review-check` → What/Why/How 提交 → PR → CI 绿直接合（已授权）
- [ ] 5.3 `/opsx:sync` + `/opsx:archive`；更新 [[snd-integration-program]] ②期状态、③期入口
