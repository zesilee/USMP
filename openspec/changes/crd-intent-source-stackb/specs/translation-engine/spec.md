## ADDED Requirements

### Requirement: 翻译消费方改为 CRD 意图源

翻译入口 `translator.TranslateConfig(vendor, configType, spec)` 契约保持不变，但其消费方 SHALL 从 Actor（Stack A）改为 Stack B 的 CRD 意图源：CRD Spec → `TranslateConfig` → 厂商 ygot desired → ConfigStore。翻译输出 SHALL 为 ygot 生成结构（R04）。

#### Scenario: CRD 源调用翻译
- **WHEN** CRD 意图源处理一个业务 CR
- **THEN** SHALL 调 `TranslateConfig` 得厂商 ygot desired，写入 ConfigStore

#### Scenario: Route/System 翻译缺口显式化
- **WHEN** 迁移 BusinessRoute/Switch（当前 Route 翻译返回裸 map、System 不支持）
- **THEN** SHALL 补齐为 ygot 翻译，或显式标注该 configType 为受限/未支持（不静默产出裸 map）
