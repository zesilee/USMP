# ownership-hard-lock — 设计

## Context

一期数据面已就绪：`intent.OwnershipIndex`（进程内注册表，intent Reconciler 维护，`Owners(device, path)` 双向前缀匹配）、`config_handler.ownershipWarningFor`（POST/DELETE 响应附警告）、`GET /ownership/:device`、前端徽标 + ElMessage.warning。信封惯例：HTTP 恒 200，`{code,message,data,success}`，`Error()` 现不携带 data。

## Goals / Non-Goals

**Goals:**

- POST/DELETE 命中认领路径缺省 409 拒绝，错误响应携带认领意图列表（前端可渲染）。
- force=true 放行 + 审计可辨识（谁在什么路径上覆盖了哪个意图的认领）。
- 前端阻断确认流：409 → 确认框 → force 重发。

**Non-Goals:**

- 不做鉴权/角色（force 无权限区分，Actor 仍恒 "system"——无鉴权是平台现状）。
- 不改归属索引数据面与意图收敛逻辑。
- 不锁 GET 读路径。

## Decisions

### D1 拒绝响应形态：`Error` 不动，新增带 data 的错误助手

`Error(c, code, msg)` 是全 API 面共用签名，加参数会波及所有调用点。新增 `ErrorWithData(c, code, msg, data)`（同信封，data 携 `{intents: []string}`），仅硬锁使用。信封码取 409（语义=冲突：与意图声明的期望态冲突），HTTP 仍 200（既有惯例，前端拦截器按 success/code 分支）。

### D2 force 逃生：query 参数 + 审计强留痕

`force=true`（query，与 `force_refresh` 命名风格一致）放行。放行后：响应仍附 `ownershipWarning`（BR-11 一期行为在 force 分支保留）；审计记录 `Forced=true, ForcedOwners=认领意图`。**为什么不做「先自动改意图」**：意图是声明式期望态，系统替用户改意图等于猜意图，越权且不可审计；force 的语义是「我知道会被覆盖，仍要临时手改」。

### D3 审计字段扩展而非 Summary 前缀

`audit.Record` + `Forced bool` + `ForcedOwners []string`（omitempty）。CRD manifest 增可选属性（向后兼容：旧 CR 无该字段 = zero value）；CRDStore spec 映射与 /logs DTO 同步透出。**备选** Summary 前缀 "[force]"——不可结构化查询、契约含糊，拒绝。

### D4 门禁位置：handler 入口早失败，在编解码之前

SetConfig/DeleteConfig 入口处（解析 path/key 后、编解码与建连之前）查 `Owners()`：early-reject 不浪费编解码/设备 IO；被拒请求按 OA-01 既有语义**不产生审计记录**（force 放行的才记录，且带标记）。

### D5 前端确认流复用 ElMessageBox，两处调用点收敛到一个助手

`useConfigSubmit.ts` 与 `ModuleFormTab.vue` 两处下发分支都可能命中 409。抽 `confirmOwnershipOverride(err): Promise<boolean>` 助手（`src/composables/` 下）：识别信封 `code===409 && data.intents`，`ElMessageBox.confirm`（列意图名 + 覆盖警告文案，确认按钮「强制下发」），确认返回 true 由调用方带 `force=true` 重发，取消则中止流程且不报错态。**F3 真浏览器层不需要**：ElMessageBox 在 happy-dom 可 mock 断言调用参数（F2 覆盖），无 el-select 弹层/teleport 交互。

### D6 setConfig/deleteConfig API 签名

前端 `setConfig(ip, path, body, force?)` / `deleteConfig(ip, path, key, force?)` 追加可选参数拼 query；后端 swagger 注解补 `force` query 参数，契约生成物再生（1a 漂移门禁）。

## Risks / Trade-offs

- **[风险] 硬锁误伤：前缀匹配过宽把邻近路径也锁死** → 复用一期已交付且有测试的 `Owners()` 双向前缀匹配语义，不另造匹配；B3 补「兄弟路径不受锁」负路径用例。
- **[风险] 意图删除后索引未及时清理导致死锁** → 索引由 intent Reconciler 维护（Remove on delete，一期已有）；force 通道本身就是最终逃生门。
- **[取舍] 无鉴权环境下 force 人人可用** → 平台现状即无鉴权（Actor=system）；本期价值是「显式确认 + 审计留痕」而非权限控制，权限属未来鉴权 change。
- **[风险] BREAKING：依赖「警告但放行」行为的脚本/自动化被 409 打断** → 内部平台无外部 API 消费者；前端同 change 内适配；PR 描述显式标注行为收紧。

## Migration Plan

单 PR 交付（预估 <600 行）。部署零迁移：CRD 新增可选字段向后兼容；回滚 = revert PR（force 期间产生的审计记录字段被旧代码忽略，无害）。

## Open Questions

（无——force 参数名、409 码值、确认文案均按既有惯例定，无需用户先决。）
