---
name: snd-integration-program
description: snd 驱动包融合四期计划：做任何 YANG 基线/左树/i18n/能力协商工作前必读——分期拍板、snd 包结构、基线漂移实测、yang-models 依赖触点清单
metadata: 
  node_type: memory
  type: project
  originSessionId: 1c015416-7746-4d30-8363-c35061c4e505
---

**用户需求（2026-07-18 拍板）**：/home/leezesi/snd（华为 NCE 风格 SND 驱动包，CE6866P）整包融合进 USMP：①以 snd 为 YANG 基线并删除 yang-models submodule 依赖 → ②设备角色（DCGW/EOR）+能力协商（统一 YANG 对接多版本交换机）→ ③原生配置左树重构为 webui/template/left-tree.json → ④界面中英文。**每期独立 change+PR，顺序执行。** **①期已交付（2026-07-18）**：PR#204 snd 入库+切源+测试适配（漂移适配清单：hwfix 删 TagMode/NetworkLayerStatus/ProtocolUpDelayTime、rtp 示范标量迁 nodes/node/match-mode、Af key 改 E_HuaweiBgp_AfTypeDeviations、CE ImportFilterPolicy 只有 AclNameOrNum+Ipv4PrefixFilter、BGP shape 锁 41→34/29→28、ifm class operation-exclude 只剩 update）+ PR#205 submodule 删除 + PR#206 收官归档。pr-size/commit-msg 已排除 snd/**（两处同口径）。**②期已交付（2026-07-18）**：PR#207 设备角色+能力协商（Device CRD spec.role ≤32字符[A-Za-z0-9_-]、/yang/modules?device= 协商子集+negotiated标记+未注册404+离线降级、tools/blacklistgen 构建期黑名单→blacklisted注解只注解不裁剪、netconfsim SetCapabilities 已有）+ PR#208 收官归档。关键实现事实：运行期 Module.Namespace() 恒空（ygot 装载不带 ns）→ 能力匹配靠名称子串启发、blacklist 键=根容器名（goyang 构建期解析）；能力=ClientPool 活跃连接 ServerCapabilities() 可选接口断言（连接即缓存）。**③期已交付（2026-07-18）**：PR#209 左树重构（lefttreegen 构建期 codegen→lefttree.gen.go、GET /yang/left-tree available/module/?device= supported、前端 LeftTreeMenu 递归组件+未接入禁用占位+失败回退 category 分组、smoke 改 lefttree-* 锚点）。体积口径新增排除 **/*.gen.go（commit-msg+pr-size 两处）。**④期已交付（2026-07-19，四期全部完成）**：PR#211 i18n（vue-i18n+locale store+ElConfigProvider、252 键双语词表 38 文件抽取、useFieldLabels res 查表懒加载+回退、make sync-snd-i18n 前端入库副本、smoke 本地化断言适配）。踩坑存档：F3 browser 套件独立配置需单独挂 i18n setup（CI 兜出）；smoke 断言被列头同名+固定列隐藏副本坑过（限定 .el-drawer）；res 懒加载须先渲染原始标签再原位重标（Tab 派生时序）。**snd 融合四期计划全部交付**：PR#204-#211。覆盖率棘轮 70.0。后续可选债：65 模块渐进生成 structs（③期占位变可点）、blacklist 硬裁剪语义澄清、category 分组降级路径退役评估。

**三项用户拍板**：左树=全树展示 65 模块+未接入占位（BR-04 降级）+按业务优先级渐进生成 structs；角色=Device CRD 增 spec.role 人工指定（角色只做展示/策略标签不参与裁剪）+模块能力由 NETCONF hello 实测协商；节奏=①→②→③→④。

**snd 包结构**：ce6866p-yang/ 143 个 .yang（137 huawei + ietf×2 + openconfig-telemetry 系×4）+ blacklist.xml（23 模块屏蔽，22 个文件仍在）+ domain.xml("ce6866p-snd")；resources/netconf-driver.xml（保活模板）+ CliPassthroughCommands.xml + i18n/{zh-cn,en-us} 各 93 个 res.json（键=YANG 数据路径→{"name":本地化名}，中英完全对称）；webui/template/left-tree.json（14 顶层组/3 层/65 模块 xpath，全部有对应 yang）。

**基线漂移实测（2026-07-18）**：ygot 对 snd 目录生成现有 7 模块一次通过；ifm/bgp 是更老 revision（CE vs NE）；171 个类型消失但业务代码仅 1 处引用（hwfix 夹具）；108 个在用类型中 15 个缺字段（VLAN MemberPort.TagMode、IFM Ipv4/Ipv6/NetworkLayerStatus 等），使用点全在测试/夹具/模拟器，零生产逻辑受影响。4 个关键 extension（task-name/support-filter/operation-exclude/dynamic-default）snd 全有（huawei-extension.yang 共 25 个 ext，21 个未消费）。

**yang-models submodule 依赖触点（删除时全清）**：huawei gen.conf yang_path、scripts/gen-yang.sh 校验提示、yangschema/load.go go:generate tasknamegen -path、compliance.yml 3 处（path 触发/变更检测 grep/submodule init）、.githooks/pre-commit 提示语、.gitmodules、generated/huawei/doc.go、generated/README.md、docs/DEPLOY-WSL-CN.md、openspec/specs/{yang-codegen-pipeline,yang-api} 条款。

**能力协商现状**：NarrowModulesByCapabilities 已实现已测但未接生产链路；/yang/modules 不分设备返回全量；client.ServerCapabilities() 已读 hello 但只喂 confirmed-commit 判断。Device CRD 无 role/版本/型号字段。

**前端现状**：零 i18n（无 vue-i18n、52/56 文件硬编码中文、ElementPlus locale 未配）；FieldDef.label=YANG 英文节点名；左树=category(task-name) 单级分组。

相关：[[arch-optimization-roadmap]]、[[snd-driver-registry]]、[[vlan-config-stackb]]。
