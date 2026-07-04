# translation-engine — 翻译引擎架构设计（反向还原）

> **权威性**：属 Stack A 配置链路的一环；**仅 Huawei 实现**，Cisco/H3C/Juniper 为枚举占位 stub。
> **还原基准**：`main@b1cfbae`，代码根 `backend/pkg/translator/`。

## 1. 职责

把业务 CRD 意图 Spec（`bizv1.Business*Spec`，来自 `backend/api/v1`）翻译为**厂商 ygot YANG 结构**（当前为 Huawei）。是一个按厂商插件化的映射层，供 Stack A 的 CRD 控制器在下发前调用。

## 2. 组件

### 2.1 核心接口 `Translator` — `translator.go:31`
方法：`TranslateVlan / TranslateInterface / TranslateRoute / TranslateSystem`（`spec interface{}` → 厂商 YANG `interface{}`）+ `Vendor()` + `Validate()`。

### 2.2 厂商与工厂
- 声明厂商 `translator.go:12`：`Huawei, Cisco, H3C, Juniper, Unknown`。**仅 Huawei 实现并注册**。
- 注册/工厂 `factory.go:8`：`GetTranslator` 经 `once.Do` **只注册 Huawei**（`factory.go:21`）；其余返回「未找到厂商」错误。
- 调度入口 `TranslateConfig` `factory.go:57`：validate → `switch ConfigType`。

### 2.3 Huawei 翻译器 `huawei.go:8`（组合 `vlanTrans` + `ifaceTrans`）
- **VLAN** `huawei_vlan.go:22`：`Translate(bizv1.BusinessVlanSpec) → *huawei.HuaweiVlan_Vlan_Vlans`，构建以 `uint16` VLAN ID 为键的 map，admin/type/mac-learning → 数字枚举（`:32-97`）。
- **Interface** `huawei_interface.go:22`：`Translate(bizv1.BusinessInterfaceSpec) → *huawei.HuaweiIfm_Ifm_Interfaces`，以名为键（`:30`）。
- **Route** `huawei.go:54`：**非** YANG 结构，返回 `map[string]interface{}`（注释 CLI/huawei-ip，未完成）。
- **System** `translator.go:135`：基类 `TranslateSystem` 返回 `NewUnsupportedError`，Huawei 未覆盖 → 不支持。
- 错误：`TranslateError`（含 `Unsupported` 标志，`translator.go:56`）；CIDR/IP 助手在 `utils.go`。

## 3. 数据流

```
controllers/BusinessVlanReconciler.Reconcile
   → translator.TranslateConfig(VendorHuawei, ConfigTypeVlan, bizv1.BusinessVlanSpec)
        factory.GetTranslator(Huawei) → HuaweiTranslator.TranslateVlan
   → *huawei.HuaweiVlan_Vlan_Vlans   （ygot 结构）
   → ActorManager … TranslateCmd/Prepare/Commit → NETCONF   （见 actor-transaction）
```

## 4. as-built 缺口

| 缺口 | 位置 | 影响 |
|------|------|------|
| 多厂商仅 Huawei | `factory.go:21` | Cisco/H3C/Juniper 仅枚举占位，无 translator 文件（迁移债 D8） |
| Route 翻译未完成 | `huawei.go:54` | 返回裸 map 而非 ygot 结构 |
| System 翻译不支持 | `translator.go:135` | Huawei 未实现 |

## 5. 红线对照

- **R04（ygot 生成、禁手写、禁滥用 interface{}）**：VLAN/Interface 输出为 ygot 结构 ✅；但接口签名用 `interface{}` 传 spec/结果、Route 返回裸 map 是 R04 张力点。

## 6. 关联
- `business-crd/design.md`（意图来源）、`actor-transaction/design.md`（下发消费方）、`yang-ygot-generate` 技能（厂商 YANG 结构生成）。
