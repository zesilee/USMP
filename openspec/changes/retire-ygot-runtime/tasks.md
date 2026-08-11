# Tasks: retire-ygot-runtime

> 7 阶段渐进迁移（design.md Migration Plan），每阶段独立可合入、可回退，全程主线可发布。
> 每步：旧链路保留 + 新链路并行 + 双路径对拍绿 → 切换 → 删旧（§5.3 存量改造军规）。
> 单 PR ≤1000 行（生成物豁免口径沿用 regen-and-diff），测试先行（T05/T06）。

## 1. Schema IR 自立（YN-03，先删 goyang 运行时大头）

- [x] 1.1 定义 Schema IR 自有序列化格式（含版本号字段、namespace/list-key 元数据），先写格式 round-trip + 版本不匹配快速失败测试（红）
- [x] 1.2 构建期工具 `tools/schemagen`：复用 `schema/entry.go` 转换逻辑（goyang Entry→内部 Node 模型）迁入工具，产出 IR 生成物入库；工具 B1 表格驱动测试
- [x] 1.3 运行期 IR 加载器（`pkg/yang-runtime/schema`），并行于旧 `AddYgotSchema` 链路；schema 通道 golden 对拍测试（`/yang/schema` 全模块新旧逐字节，复用 68 模块黄金口径）
- [x] 1.4 对拍绿后 `yangschema.Load()` 切换到 IR 数据源；前端派生黄金全量刷新核对（GD-01/SF-04）
- [x] 1.5 删除运行时旧链路：`schema/entry.go`、`schema/loader.go` 迁出运行时包（测试消费方改用 IR 或迁 tools）

## 2. 自研生成器（CG-01 修订，YN-01 类型约定）

- [x] 2.1 `Object`/`KeyedObject`/`Enum` 接口族 + 指针 helper 运行库包（先写接口契约测试：key 元数据/枚举映射与 ygot 等价，红绿）
- [x] 2.2 生成器 `tools/yanggen` 骨架：gen.conf 解析、结构体/枚举/union 生成（命名、`path`+`module` tag、map-list、`...Key`、枚举标识符合法化内建），B1 表格驱动测试覆盖命名/tag/union 分支
- [x] 2.3 确定性保证（无序集合稳定排序内建）+ split_count 拆分布局；连续两次生成字节一致测试
- [x] 2.4 全模块生成 `internal/generated/native/*`（与旧生成物并存），结构约定 diff 对拍：字段名/tag/类型集合与 ygot 生成物逐一比对的守护测试
- [x] 2.5 `make gen-yang` 接入自研生成器（VENDOR= 口径保持），CG-03 regen-and-diff 门禁覆盖新生成物；存量 deviation 集合沿用验证（CG-04 修订）

## 3. RFC7951 JSON 通道（YN-02）

- [ ] 3.1 生成器扩展：per-type `MarshalJSON`/`UnmarshalJSON` 方法生成（list map↔数组、(u)int64 字符串化、枚举值域名、union 试探、模块限定 key 兼容）
- [ ] 3.2 JSON 通道 golden 对拍测试：全模块 fixture + union/int64/枚举/嵌套 list 构造样本，新旧 解码→编码 往返逐字节比对；负路径（未知字段/类型不符/非法枚举）行为对齐
- [ ] 3.3 对拍绿后切换：driver 注册表签名换 `Object`（DR-01 修订）、`internal/drivers` 描述符换新类型与生成方法、`config_codec`/`changeset_handler`/`config_delete` 的 `EmitJSON`/`Unmarshal` 调用切换
- [ ] 3.4 存量 B2/B3 全绿 + e2e smoke 验证对外 API 形状零变化

## 4. XML 通道（XC-01/02/03/08 修订）

- [ ] 4.1 xmlcodec 触点置换：`ygot.GoStruct`→`Object`、`KeyHelperGoStruct`→`KeyedObject`、`EnumName`/`ΛMap`→`Enum` 映射、schema 入口从 `huawei.SchemaTree`（yang.Entry）改读 Schema IR
- [ ] 4.2 XML 通道 golden 对拍：同一配置树新旧类型经 encode/decode/delete 编码逐字节比对（含 empty/枚举/多键/嵌套/per-node namespace 全形态）；既有 xmlcodec golden 全量保持
- [ ] 4.3 对拍绿后 reconciler/intent 层 Change 值切换到新类型，B2 模拟网元端到端（下发→回读→二次收敛 Changes==0）全绿

## 5. 服务端校验（YN-04）

- [ ] 5.1 先写 `intent/cr.go` 现状 `Validate()` 行为快照测试（接受/拒绝用例基线，红绿基线）
- [ ] 5.2 IR 驱动校验器（pattern/range/length/enum，must/when 明确不做），快照用例双跑一致后切换，删除 `Validate()` 调用

## 6. 胶水清扫与旧链路删除

- [ ] 6.1 清零剩余运行时 ygot 引用：`intent/{expand,cleanup}`、`api/config_delete`、指针 helper 全量换自研；simulator/testutil 迁新类型（豁免面留守护测试白名单）
- [ ] 6.2 businessdemo 处置拍板执行（随迁 or 删除，视北向 demo 存续）
- [ ] 6.3 删除旧生成物 `internal/generated/{huawei,business}` ygot 版与三通道对拍脚手架（golden 文件保留为回归基线）；全量测试绿

## 7. 收尾与门禁（YN-05 / SC-07）

- [ ] 7.1 守护测试：`go list -deps` 断言 `usmp-backend` import 闭包零 ygot/goyang（tools/测试/simulator 豁免），仿 NC-01 口径
- [ ] 7.2 go.mod 整理：两库降为工具/测试依赖；验收口径（二进制不链接 vs tools 拆独立 module）与用户拍板执行
- [ ] 7.3 文档同步：CLAUDE.md R04 表述、§3 技术栈依赖行、TEAM_HANDBOOK 相关条目；覆盖率棘轮按新增测试上调（T08）
- [ ] 7.4 发布验证：`scripts/build-release.sh` 出包 + 干净容器冒烟 + `go version -m` 依赖审计留证
