# system-architecture — 行为契约（反向还原）

> 反向还原自实现（Brownfield），忠实 as-built。详细架构见同目录 `design.md`。

## 能力概述

系统级架构不变量与红线契约：约束整个平台"无数据库、模型驱动、NETCONF/gNMI、yang-controller-runtime"的边界。本契约以**可观测的系统级行为**表述；子系统行为契约见各能力 `spec.md`。

## 系统级契约

### SC-01 单一权威栈
- **Given** 代码存在 Stack A(K8s CRD+Actor) 与 Stack B(yang-controller-runtime) 双栈
- **When** 判定架构权威性
- **Then** 依 R01，Stack B 为权威；Stack A 标注 `legacy` 待退役（现状：生产入口仍为 Stack A，属已知迁移债）

### SC-02 无数据库
- **Given** 需要存储期望配置
- **When** 系统运行
- **Then** 仅用 TTL+LRU 内存 + 本地 JSON 元信息；不得引入 MySQL/Redis/SQLite（R03）。Stack A 的 etcd 依赖为与 R03 张力的 legacy 项。

### SC-03 仅现代协议
- **Given** 需要与设备通信
- **When** 下发/读取配置
- **Then** 仅 NETCONF(SSH 830)/gNMI；禁止 Telnet/SNMP（R02）。`NativeDeviceConfig` 的 `format: CLI` 透传为软张力项。

### SC-04 模型驱动渲染
- **Given** 新增 YANG/CRD 模块
- **When** 前端展示配置表单
- **Then** 由模型 schema 自动渲染，零手写固定表单（R05）；YANG 结构由 ygot 自动生成（R04）。

### SC-05 异常降级不崩溃
- **Given** 设备离线/缓存过期/下发失败/校验失败
- **When** 异常发生
- **Then** 每种异常均有降级路径（重连/重拉/保留原配置/行内提示），进程不崩溃（R08）。

## 红线合规现状

见 `design.md` §6 合规矩阵。当前 R01/R03 因双栈处于**部分违反**（legacy 栈所致），R02/R04/R05 合规。

## 关联
- `design.md`（本能力）、各子系统 `spec.md`/`design.md`、`CLAUDE.md` §2 红线。
