# Proposal: full-yang-onboarding

## Why

SND 左树 61 个能力叶中仅 11 个可用（模块已生成加载），其余 50 个显示「不可用」——设备配置管理只覆盖了功能面的一小角。注册表 + 通用 XML 编解码引擎 + manifest 管线已就绪（「加模块 = 注册一条描述符 + gen.conf 加模块名」），全量接入的边际成本已降到数据行级别，应一次收口。

## What Changes

- **全量生成**：gen.conf 由 7 模块扩到 58（+49 个叶模块 + huawei-ip 依赖 + usmp-deviations），ygot 闭包根容器 13→67
- **本地 deviation 模块**（`usmp-deviations.yang`）：精准豁免 ygot 生成器不支持的个别节点（syslog bits-default ×2、cfg anydata、qos binary-key 查询列表、lldp 穿 choice/case 的 leafref），模块本体零改动（snd 子模块只读）
- **管线扩展**：gen-yang.sh 支持逗号分隔多 YANG 目录（deviation 目录入闭包）+ `-ignore_unsupported`
- **表驱动描述符注册**：49 个「单容器根」模块以数据行注册（module/root/namespace/构造子），替代逐模块手写 18 行块
- **路径约定统一（修复存量断链）**：运行时配置路径前缀统一为**根容器名**（前端 configPathFor 的派生口径）；tunnel-management / routing-policy / network-instance 描述符从 YANG prefix 锚（`/tnlm:` `/rtp:` `/ni:`）迁到根名锚——这三个模块的控制台写链路此前因前缀口径不一致实际不可达
- **泛型 Reconciler**：提取 plain-container 泛型 reconciler（xpl/tnlm/rtp/acl 形态完全同构），main.go 以描述符循环注册新模块控制器
- **延期**：`huawei-pic`（goyang 无法解析跨模块 submodule typedef 引用 `devm:switch-status-type`，非本仓可修）——左树保持不可用标注，唯一延期项

## Capabilities

### New Capabilities

（无）

### Modified Capabilities

- `device-driver-registry`: 表驱动 plain-container 注册 + 运行时路径前缀=根容器名的约定固化
- `yang-codegen-pipeline`: 多 YANG 目录闭包 + 本地 deviation 豁免机制 + ignore_unsupported
- `left-tree-navigation`: 全量叶可用（61 叶中 60 可用，pic 唯一延期）

## Impact

- `backend/internal/generated/huawei/`：再生成（13→67 根容器，~15MB，体积门禁已豁免生成物）
- `backend/internal/yang/deviations/usmp-deviations.yang`：新增
- `scripts/gen-yang.sh`、`backend/internal/generated/huawei/gen.conf`、taskname go:generate 清单
- `backend/internal/drivers/huawei.go`：表驱动注册（+3 描述符锚点迁移）
- `backend/internal/controller/plainmodule/`（新）泛型 reconciler；`backend/main.go` 控制器循环
- 前端零代码改动（R05 模型驱动）；F4 smoke 增左树全可用断言
- 测试：参数化 T02b 矩阵（对**每个**新模块统一跑 schema/注册表/编解码往返/API 编包/sim 端到端），代替逐模块手写矩阵
