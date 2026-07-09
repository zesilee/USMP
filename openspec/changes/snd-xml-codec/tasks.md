# snd-xml-codec — 任务清单

> TDD 红绿循环（T01/T05）：每组先测试后实现。存量并行→切换→删除（§5.3）。
> 硬门禁：golden 逐 fixture 对拍、存量 B1/B2/B3 全绿、`-race` 通过、PR ≤1000 行。

## 1. Golden 冻结现状（先行，红灯基线）

- [x] 1.1 建 golden 测试骨架：规范化比较器（XML 解析→list 条目按 key 排序→重序列化比对），自身单测覆盖乱序等价/属性保持
- [x] 1.2 代表性 fixture：VLAN 全字段（含 member-ports/suppression/unkown-unicast-discard/mac-aging/super-vlan）、IFM 全字段（含 damp/error-down/control-flap 全嵌套）、单字段最小体、空容器、特殊字符（xmlEscape）
- [x] 1.3 用**现有** buildHuaweiVlanVlansXML/buildHuaweiIfmInterfacesXML/marshalDeleteChange 生成并冻结 golden 文件（suppression 重复 bug 在 golden 中记录为修复后的单次输出，并注明豁免理由）
- [x] 1.4 基线验证：现有 builder 对拍 golden——除 suppression 豁免项外全绿，确认 golden 忠实

## 2. 通用引擎 xmlcodec（XC-01/02/03）

- [x] 2.1 先写 Encode 单测（表格驱动）：跳发语义（nil 叶/enum 零值/nil 容器/空 list 自闭合）、嵌套容器、list map、转义、不支持形态明确报错、并发 `-race`
- [x] 2.2 实现 `xmlcodec.Encode`（reflect + `path:` tag + 描述符 namespace），对拍 §1 golden 全绿
- [x] 2.3 先写 Decode 单测：往返恒等（Encode→Decode→ygot diff 为空）、rpc-reply/data 包裹/裸容器/namespace 前缀三形态、空回读返回非 nil 空容器、非法 XML 报错
- [x] 2.4 实现 `xmlcodec.Decode`（token 扫描 + schema 定位 list 条目），字段覆盖与 Encode 对称（XC-02）
- [x] 2.5 先写 EncodeDelete 单测：与既有 marshalDeleteChange 输出等价（DP-07 断言复用）、空条目集报错、key 为首子元素
- [x] 2.6 实现 `xmlcodec.EncodeDelete`（ΛListKeyMap 驱动），退役条件就绪

## 3. 描述符扩展与接线（DR-01/03、XC-04）

- [ ] 3.1 先写 registry 扩展测试：按 GoStruct 类型（容器型 + 内层 list map 型）查得 XML 编解码数据、未命中 ok=false、并发 `-race`
- [ ] 3.2 `driver.Descriptor` 增加 `XML *XMLCodecSpec`（Namespace/RootElem/RootSchema/类型标识）与按类型查找 API
- [ ] 3.3 `internal/drivers/huawei.go` 为 vlan/ifm 登记 XMLCodecSpec（namespace 从 SchemaTree 取，消灭 HuaweiVlanNS/HuaweiIfmNS 硬编码常量的第二来源）；DecodeXML 闭包改指通用引擎
- [ ] 3.4 `client.marshalChange` 切换：string/[]byte 直通 → 注册表按类型分发（容器/内层 map 两形态）→ openconfig 遗留分支保留 → xml.Marshal 兜底保留；`marshalDeleteChange` 切换注册表+EncodeDelete
- [ ] 3.5 reconciler（ifm/vlan）Parse 调用点改经描述符 DecodeXML；确认 netconfsim seed.go 引用不受影响

## 4. 双路径验证与退役

- [ ] 4.1 全量验证：`go test ./... -race` 全绿；B2 集成（下发→回读→收敛、删除、全字段配置→回读→diff 为空新用例）全绿
- [ ] 4.2 删除手写三件套：buildHuaweiVlanVlansXML/buildHuaweiIfmInterfacesXML/ParseHuaweiVlanVlansXML/ParseHuaweiIfmInterfacesXML/中间结构/mapToHuaweiIfmInterfaces/kebab 替换器/marshalDeleteChange 型 switch；client 移除 `internal/generated/huawei` import（XC-04 验收：grep 无引用）
- [ ] 4.3 删除后全量回归：`go test ./... -race` + B2 全绿；覆盖率不低于基线 58.3（T08，补测后同步上调 `.coverage-baseline`）
- [ ] 4.4 code review（`go-code-review-check`）通过；What/Why/How 提交（可多个原子 commit：golden→引擎→接线→退役）

## 5. 收尾

- [ ] 5.1 PR（分支 push + gh pr create，body 含 spec 对照与等价性证据）；CI 全绿后合入
- [ ] 5.2 `/opsx:sync` delta→主 spec（yang-xml-codec 新建、device-driver-registry/device-protocol 合并）+ `/opsx:archive`
- [ ] 5.3 回写 openspec/tasks/optimize-frontend-nce-insights.md 勾选 P5-2 + `/task sync`；记忆更新（snd-driver-registry 记忆中「netconf.go 45KB 三件套」剩余硬编码条目改写）
