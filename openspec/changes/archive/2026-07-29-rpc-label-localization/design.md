# Design — rpc-label-localization

## 背景与既有机制

UI-03 的本地化已建立完整机制（`useFieldLabels.ts`）：
- `loadFieldRes(locale, sourceModule)`：vite glob 懒加载 `snd-i18n/{locale}/{sourceModule}-res.json`，缺文件返回 `null`（R08）。
- `sourceModuleFor(root, leftTree)`：根容器名 → 源模块名（左树命中优先，否则 `huawei-<root>` 约定回退）。
- `localizeFields(fields, root, locale, leftTree)`：对配置字段树按 `resKeyFor = /<sourceModule>:<path>` 查表重标，缺键保留原 label。

`ModuleConsolePage.vue` 已有 `relabelFields()`：首帧渲染原始标签（不阻塞 Tab 派生），res 就绪后原位替换；`watch(locale)` 触发即时重查。rpc 从 `/yang/schema` 响应的 `rpcs` 字段拿到，经 `deriveRpcTabs` 派生成与容器平级的 Tab，`RpcExecuteTab` 读 `tab.rpc.label` 与 `tab.rpc.input[].label`。

## 键规律（实测 snd res）

配置叶键含根容器：`/huawei-ifm:ifm/interfaces/interface/name`。
rpc 是**模块顶层语句**，键中**无根容器段**：
- rpc 标签：`/<sourceModule>:<rpcName>`，如 `/huawei-ifm:restart-if` →「重启接口」。
- input 叶：`/<sourceModule>:<rpcName>/input/<leafName>`，如 `/huawei-ifm:restart-if/input/if-name` →「重启接口名」。

（res 另有 `/rpcs/<sourceModule>:<root>/<rpcName>` 冗余段，但它**只含 rpc 顶层标签、不含 input 叶**；直接键两者都覆盖，故采用直接键，与 `resKeyFor` 语义一致，仅去掉根容器前缀。）

后端 `RPCSchema`：`Name`=rpc 节点名，`Label`=同 Name（原始），`Input[].Path`=input 叶名（如 `if-name`，无 `/input/` 前缀）。故查 input 键时用 `/<sourceModule>:<rpcName>/input/<叶名>`，叶名取 `Path` 末段。

## 方案

新增 `localizeRpcs(rpcs, root, locale, leftTree)`：
1. `sourceModuleFor` + `loadFieldRes`，缺 res 整树原样返回（R08）。
2. 每个 rpc：查 `/<sourceModule>:<rpcName>` 命中则替换 `label`；对 `input[]` 每叶查 `/<sourceModule>:<rpcName>/input/<叶名>` 命中则替换 `label`。
3. 不改入参，返回新对象树（与 `localizeFields` 同风格）。

`ModuleConsolePage`：保留 `rawRpcs`（后端原始），`relabelFields` 内并行 `localizeFields` + `localizeRpcs`，就绪后同时写回 `schemaFields` 与 `rpcs`（同一 `rootName` 守卫防竞态）。首帧仍写原始 rpcs 保证 Tab 即时可见。

## 为何不动后端

后端 `RPCSchema.Label` 目前 = rpc 原始名。让后端查 res 需把 snd 翻译打进运行镜像（违背「运行镜像零 snd 依赖」，snd 是仅构建期来源），且 i18n 是**展示层**关注点、随 locale 切换——只能在前端做。与 UI-03 配置字段本地化完全同构，复用同一层。

## 测试

- F1（`useFieldLabels.test.ts`）：真实 huawei-ifm res 副本——`restart-if` 命中中文标签 + input `if-name` 命中「重启接口名」；缺键 rpc 回退原名；缺 res 文件整树回退；en-us 取英文。
- F2（`ModuleConsolePage` 组件测）：mock schema 返回含 rpc，断言 rpc Tab 标签渲染为本地化中文（而非原始节点名）。

## 风险与回退

- 缺翻译 rpc → 回退原名（R08），无空标签、无报错。
- locale 无 res 文件（未来新语言）→ 整树回退。
- 无契约/数据变更，纯展示层增强，风险面限于前端标签文案。
