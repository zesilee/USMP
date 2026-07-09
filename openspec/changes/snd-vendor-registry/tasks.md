# tasks — snd-vendor-registry

> TDD 红绿循环（T01/T05）：每组先测试（红）再实现（绿）。测试层按 §5.6：涉编解码/Reconciler 链路 → B1+B2（B2 为存量回归，D5）。
> 单 commit ≤500 行、原子功能；What/Why/How 三段式。

## 1. S1 Vendor 贯穿设备连接层（B1+B3，DS-01/DS-03/devices-api BR-03/BR-04）

- [x] 1.1 B1 红：`DeviceConnectionInfo.Vendor` 字段——Store Set/Get 透传、零值缺省 huawei 语义、并发读写 race
- [x] 1.2 B3 红：注册 API——带 vendor 写入、缺省 huawei、未知厂商 400、存量无 vendor 请求行为不变
- [x] 1.3 绿：client.go 字段 + store 透传 + device_handler 注册入口/swagger 注解；`make gen-contract` 再生成
- [x] 1.4 全量回归：`go test ./...`（含存量 B2 集成）

## 2. S2 translator 编译期自注册（B1，TE-01/TE-02）

- [x] 2.1 B1 红：init 自注册后 GetTranslator(huawei) 可得、未注册厂商明确报错、RegisterTranslator 并发 race、vendorOf 解析（store 命中/miss 降级 huawei）
- [x] 2.2 绿：huawei.go init() + factory.go 删 once.Do 硬注册 + crdsource 两调用点按设备 Vendor 解析
- [x] 2.3 全量回归（crdsource 双路等价性存量测试全绿）

## 3. S3 驱动描述符注册表 + 查表化（B1，DR-01/DR-02/DR-03）

- [x] 3.1 B1 红：driver 包——Register/Lookup（vendor+path 前缀匹配、未命中、重复注册、并发 race）；现有全部路径（system:/vlan:/ifm: 及各别名）查表结果与原 Contains 链逐一对拍的表格用例
- [x] 3.2 绿：新包 pkg/yang-runtime/driver + 三模块描述符注册
- [x] 3.3 红→绿：manager.go 路径→控制器路由改查表（保留未命中 fallback 行为）
- [x] 3.4 红→绿：config_codec.go 编解码表改查表（decode/encode 两处）
- [x] 3.5 B2 回归：存量 netconfsim 集成套件全绿（行为等价性证明，D5）

## 4. 收口

- [ ] 4.1 全量验证：`go test ./... -race`、`make gen-contract` 漂移、前端单测（api.gen.ts 变更）
- [ ] 4.2 覆盖率对齐棘轮（后端 58.3），补测后按需上调
- [ ] 4.3 `go-code-review-check` + What/Why/How 提交整理 + PR 体积自检（≤1000 行）
