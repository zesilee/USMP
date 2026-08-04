---
name: crd-to-yang-codegen
description: 碰北向对接/crd2yang/往返对账/businessdemo 前必读：工具已交付、goyang 全量 Range 陷阱、enum 字典序、demo 隔离锚点、接线剩余债
metadata:
  type: project
---

北向「网络自动化服务」（K8s）以 CRD 发布业务契约，USMP 用 `backend/tools/crd2yang`（CRD→YANG，`tools/crdgen` 的镜像）反向派生业务网络配置模型。change `crd-to-yang-codegen`（C2Y-01~06）2026-08-04 交付。

**Why:** 复用既有管线（ygot→schema 树→R05 渲染→意图编排）接北向契约；authority 边界（design D2）：USMP 自有意图模型 YANG-first（BIC-01 不变），北向拥有的 CRD 才走 CRD→YANG，同一模型禁止双向都跑。

**How to apply:**
- **元信息注解约定**（C2Y-02）：`usmp.io/yang-revision` 必填 fail-fast；module/namespace/prefix/container/task-name 可缺省确定性派生。北向正式 CRD 到位＝换 `-input` 文件，工具零改动。
- **goyang 全量 Range 陷阱**（检视轮 2 实证）：无 range 语句的整型 goyang 也附带基类型全量 Range → crdgen 对"无界"整型输出 ±2^63 边界。crd2yang 的 fullRanges 表把全量边界收敛回裸类型；往返归一化双侧删全量边界。改映射表前先想这条。
- **enum 序不保**：goyang `Enum.Names()` 按字典序返回、不保声明序；RFC7951 按名传输所以无害，往返对账双侧排序归一。
- **验证链**：golden+goyang 回读双保险 / 往返对账（示例+合成 CRD，`go run ../crdgen` 子进程，Short 跳过）/ businessdemo ygot 包（独立 gen.conf，勿并入 business——退役要整目录删）/ fixture `business-vlan-net.json`（模块名=顶层容器名）+ 前端派生黄金。
- **D7 隔离锚点**：demo 模型**不进**运行期 `load.go`——业务控制台前后端单 Kind 写死（`/business/vlan-services`、`cr.go` GVK、前端 api/index.ts）。`TestBusinessDemoNotInRuntimeLoad` 爆红=有人违规接线。**接线前置债：多 Kind 业务 API 泛化 + 意图展开规则，另立 change。**
- **门禁**：`make gen-crd2yang`（先于 gen-yang，.yang 是下游输入）；CI/pre-commit 同口径；pre-commit C2Y-06 diff 只盯产物文件（同目录有手写模型，扫全目录误伤）。
- refactor-by-crd.md（北向架构文档）三处撞红线未采纳：Scrapligo（NC-01 已删）、前端直连 K8s（Stack A 死路）、BusinessVlan 旧 Kind 名（SC-01）——示例 Kind 特意取 BusinessVlanNet 避开。
- 工作方式（用户 2026-08-04 拍板）：**所有代码检视都启独立 agent**。本次两轮独立检视共揪出 5 个必修（int64 回绕腐蚀 range、YANG 标识符注入、跨类型约束静默丢弃、guard 缺口、无界整型往返不闭环），全修复并有回归用例。

相关：[[k8s-paas-deployment-constraints]]、[[snd-driver-registry]]、[[vlan-config-stackb]]
