---
name: snd-driver-registry
description: "加新厂商/新YANG模块前必读：全量接入已完成(#216,60/65叶可用)、加模块=plainModules表加一行+gen.conf加模块名、根名路径口径军规、deviation豁免机制、goyang/ygot生成器破绽清单、超限PR拆法"
metadata: 
  node_type: memory
  type: project
  originSessionId: 6b56e476-f149-4620-a2db-3ee139fb8dc6
---

P5-1 `snd-vendor-registry`（#134/#135）+ P5-2 `snd-xml-codec`（#136/#137/#138/#139）+ P5-3 `snd-ygot-pipeline`（#140，2026-07-09）已合入 main。插拔性已拍板：**纯 Go 编译期**（③注册表→①声明式数据驱动终态），不做 gRPC 插件。

- **YANG 生成已 manifest 化（P5-3）**：`make gen-yang`（`VENDOR=<pkg>` 单包）扫 `backend/internal/generated/*/gen.conf`；**加厂商生成侧 = 建目录+gen.conf 零脚本改动**；`backend/tools/genfix` 跨平台后处理（枚举 `|`→`_OR_`+头部机器路径规范化）；R04 门禁已从冻结进化为 regen-and-diff（CI compliance + 本地 pre-commit 对称）——生成物改动合法当且仅当 `make gen-yang` 零漂移，**别再手改 generated/ 或想豁免**；pr-size/commit-msg 体积已排除 generated/ 目录。spec 见 yang-codegen-pipeline（CG-01~03）。
- **生成物已按文件拆分（PR#162，2026-07-15）**：huawei 包不再是单文件 all.gen.go，而是 `structs-0..7.go + enum.go + enum_map.go + union.go + schema.go`（gen.conf `split_count=8` 驱动，未设置的包如 openconfig 仍单文件）；blob 独占 schema.go，struct/blob diff 分离。⚠️ ygot `-output_dir` 给每文件写同一份 import 块，未用 import 编译失败——拆分模式收尾是 `go tool goimports`（go.mod tool 指令锁版）而非 gofmt，**别把拆分包的收尾改回 gofmt**。加模块流程不变（gen.conf 加模块名+重生成），文件数/分桶由 generator 确定。

- **全量接入已完成（PR#216，2026-07-19）**：左树 65 叶 60 可用；例外=pic（goyang 不识跨模块 submodule typedef，延期）+4 个 augment-only 叶（ethernet/ip/gre/nvo3-statistics，入闭包并进宿主树但无独立控制台）。**加新 YANG 模块 = `huawei_modules.go` 的 `plainModules` 表加一行（module/ns/构造子）+ gen.conf 加模块名 + 重生成**，谓词/锚点/控制器全自动（main.go 按 driver.All() 循环装配泛型 plainmodule reconciler）。**路径口径军规（DR-06）**：运行时配置路径前缀=根容器名（`/<root>:<root>/...`，前端 configPathFor 口径），YANG prefix 只属 XML namespace 层——2b 波次曾锚 `/tnlm:`/`/rtp:` 致控制台断链已修；vlan/ifm/system 裸子串谓词已收紧防误吞（manager 路由也已改精确名匹配 `<vendor>-<token>`，触发按锚点入队）。**ygot 生成器破绽与对策（usmp-deviations.yang，CG-04）**：bits 带 default→deviate delete default；anydata→not-supported；binary key list→not-supported；leafref 穿 choice/case augment→deviate replace type string；跨模块 submodule typedef 引用→无解（pic 延期）。参数化 T02b 矩阵在 drivers（往返）/plainmodule（sim 端到端）/api（编包+LT-04 基线），加模块自动被矩阵覆盖，冒烟左树断言是相对不变式（available=total−5，改例外数才要动）。schema 闭包 13→67 根后 Load() 已 sync.Once 记忆化、pre-commit 包测试 120s/pre-push race 420s。
- **（历史）逐模块手写描述符** = 在 `backend/internal/drivers/huawei.go` 注册一条 `driver.Descriptor`（现仅 system/vlan/ifm/bgp/ni 五个手写块）——**零 XML 代码**：encode/decode/键式 delete 全部由通用引擎 `pkg/yang-runtime/xmlcodec` 按 ygot 生成物数据（`path:` tag/SchemaTree/ΛListKeyMap）驱动。⚠️ namespace 必须显式登记：内嵌 gzip schema 的 `Entry.Namespace()` 实测返回空，不可派生。⚠️ 引擎不做 config-false 过滤：华为模型把在发字段（member-ports 整树、is-l2-switch）都标 config-false，按 schema 过滤会破坏行为等价。
- **注册可达性**：注册靠空白导入 internal/drivers 触发——api、vlan/ifm reconciler 生产文件、actor 测试二进制已带；**新增二进制/独立测试包若不带，注册表为空→编码落 xml.Marshal 兜底对 map 报错**（actor 曾因此翻车，评审抓回）。
- **golden 方法论**（改编码语义前必用）：`xmlcodec.Canonicalize`（同级全排序+相同同级去重+namespace 前缀不敏感）冻结现状→新实现逐 fixture 对拍；fixture/golden 在 `internal/testutil/hwfix`（`-args -update-golden` 刷新）；抹序后 key-first/根 NS 需在原始输出上定向断言。历史回归用例原文经 test-only shim 走新路径 = 等价性持续证明。
- **Vendor 数据通路**：注册 API 可选 `vendor`（缺省 huawei、未知 400）→ DeviceStore → crdsource 按设备解析 translator。边界用 `translator.VendorFromString` 大小写无关规范化。
- **P5-4 已剪出（用户拍板 2026-07-10，不需要接入非华为设备）**：路线图完结归档（openspec/tasks/archive/，PR #142）。多厂商能力零成本待命：未来若需接新厂商 = 加 gen.conf + 注册 driver.Descriptor + netconfsim 方言，无需架构立项。遗留低优先机械债（无载体、按需另立小 change）：mergeConfig/config_delete 的 per-model type-switch 收敛（internal/api，华为模型自身也受益）；xmlcodec Decode 顶层按 Local 名扫描条目的方言观察项（仅真接新厂商时相关）。~~ygot 生成管线 sed 四连击~~ P5-3 已消灭。
- **坑**：单 commit ≤500 行、PR >20 文件限 3000 行——超限按「并行→切换→删除」拆顺序 PR（#136 纯增量→#137 切换→#138 删除），**逐个合完再开下一个**（#129 堆叠翻车教训）；本机 gh 无 `pr update-branch`，BEHIND 时本地 merge origin/main 再推（W07 禁 force push）；EnterWorktree 从本地 origin/main 建分支——先 `git fetch`；自托管 runner 卡住时 commit-lint/sensitive-files 会 15min cancelled，`gh run rerun --failed` 即可。

**Why:** 防止后续迭代绕过注册表/通用引擎重新手写 XML 胶水、或在注册可达性/namespace/config-false 上重踩坑。
**How to apply:** 动 translator/驱动/编解码前先读 `openspec/specs/device-driver-registry/spec.md`、`yang-xml-codec/spec.md`（XC-01~04）与 `translation-engine/spec.md`（TE-05/06）。

> 2026-07-17（rfc7951-input-contract）：Descriptor 新增 `EncodeAnchor`（NewStruct 容器规范路径，DR-05）——config-api 写路径据此包裹子路径载荷；加新模块登记描述符时**必须带锚点**，否则写路径 400。谓词声明式化的第一块数据（MatchEncode 可由锚点派生退役）。
