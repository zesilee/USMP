# translation-engine — 行为契约（反向还原）

> 反向还原自 `backend/pkg/translator/`，忠实 as-built。详见 `design.md`。仅 Huawei 实现。

## 能力概述

把业务 CRD 意图 Spec 翻译为厂商 ygot YANG 结构，按厂商插件化，供 Stack A 控制器下发前调用。

## 行为契约

### TE-01 厂商派发
- **Given** `vendor` + `configType` + spec
- **When** `TranslateConfig(vendor, type, spec)`
- **Then** `GetTranslator(vendor)` 取翻译器后按 type 分派；未注册厂商返回「未找到厂商」错误

### TE-02 VLAN 翻译
- **Given** `bizv1.BusinessVlanSpec`
- **When** `HuaweiTranslator.TranslateVlan`
- **Then** 产出 `*huawei.HuaweiVlan_Vlan_Vlans`，以 uint16 VLAN ID 为键，admin/type/mac-learning → 数字枚举

### TE-03 Interface 翻译
- **Given** `bizv1.BusinessInterfaceSpec`
- **When** `HuaweiTranslator.TranslateInterface`
- **Then** 产出 `*huawei.HuaweiIfm_Ifm_Interfaces`，以接口名为键

### TE-04 校验
- **Given** 输入 spec
- **When** `Validate`
- **Then** 校验 CIDR/IP 等，不合法返回 `TranslateError`；不支持项返回 `Unsupported`

## 契约缺口（详见 design.md §4）

- 仅 Huawei 注册；Route 翻译返回裸 map（未完成）；System 翻译不支持；接口以 `interface{}` 传参（R04 张力）。

## 关联
- `design.md`、`business-crd/spec.md`（意图来源）、`actor-transaction/spec.md`（下发消费）、`yang-ygot-generate` 技能。
