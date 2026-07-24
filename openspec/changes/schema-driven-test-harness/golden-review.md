# 首次黄金人工审阅记录（GD-04 / R2 唯一一次语义确认）

> 黄金门禁只保证「派生确定、未非预期变化」，不保证「派生结果对用户合理」。
> 首次生成时的这一次人工审阅是唯一一次语义确认，此后黄金只承担回归防线（GD-04）。

## 抽样口径（Open-Q1）

分层抽样 16 个模块：EASY ×2（vlan/acl）、EASY\* ×2（evpn/telemetry-system）、
PATTERN 全 7（dsa/ecc/rsa/sm2/snmp/time-range/vxlan-path-detect）、COND 2（arp/routing）、
规模 top3（network-instance/qos/ifm）。分类口径见探索阶段 schema 难度分析。

## 结论

**16/16 抽样模块的派生结论结构合理**，无破损派生。逐项确认：

- Tab 派生：list/form 划分符合 schema 顶层结构；顶层散叶聚合为 `__basic__` 表单 Tab；
  只读子树（config false）照常出 Tab 且整 Tab 标只读（FE-14），符合预期。
- 主键派生：抽样 list 的 keyField 均落在派生列内，无回退到非主键叶的异常。
- 列派生：分层取列（key→create-only→when→enum→其余）+ cap=9 截断行为一致。
- tree 派生：kind/isConfig/isReadonly 与 schema 的 config-false 标注一致
  （如 vlan default-instance 全只读、qos 只读占比高）。
- COND 模块（arp/routing）派生正常——黄金派生不需要满足 when/must（那是实例生成的
  约束，属后续设备一致性矩阵 change），故 COND 难度不影响本层。

## Follow-up 观察（不在本 change 修，转后续）

**F1 — deriveColumns cap=9 对宽列表的截断值得 UX 复核。**
多个宽 list 触顶 9 列：`vlan/vlans`、`snmp/{target-hosts,usm-users}`、`arp/*`、
`ifm/interfaces`。分层顺序把 enum 排在「其余标量」前，可能把 `name`/`description`
这类字符串列挤出前 9。这是**既有生产行为**（非本 change 引入），黄金只是首次把它
显性化。是否调整 cap 或分层顺序是纯 UX 判断，需单独评估——**本 change 不改派生逻辑**
（改了会震动全部黄金，超出「钉住现状」的范围）。

> 结论：无阻断项。黄金如实反映当前派生行为，可作为回归基线。F1 作为独立 UX
> 议题记录，不进本 change。
