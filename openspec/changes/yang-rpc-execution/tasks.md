## 阶段① 构建期提取 + 列 rpc（后端地基，一 PR）

### 1. Worktree 与基线

- [ ] 1.1 创建 worktree，验证 `.gitignore`，跑基线 `make compliance` 全绿
- [x] 1.2 扫全模块 rpc：列出各模块 rpc 名与 input 结构，定稿高危关键词清单（Open-Q3）

### 2. rpcgen 构建期提取（B1，RPC-01）

- [x] 2.1 **先写测试**：`backend/tools/rpcgen` B1——(a) huawei-ifm 提取出全部 rpc；(b) `reset-if-counters-by-name` 的 input `if-name` 带 leafref 目标 + mandatory；(c) 模块动态发现不硬编码；(d) 提取确定性（两次逐字节一致）。红灯先行
- [x] 2.2 实现 rpcgen：goyang 解析 → `Entry.RPC` 提取 rpc 名 + input 叶树（含 leafref/mandatory/units/range/pattern）→ 生成 `internal/yangschema/rpc.gen.go`。形状对齐 lefttreegen/tasknamegen。转绿
- [x] 2.3 高危分类（RPC-04）：名称启发式（restart/reboot/reload/reset/clear/delete…）打 highRisk 标，B1 断言 restart-if 高危、reset-if-counters 按清单归类
- [x] 2.4 `Makefile` 加 `gen-rpc`；执行生成 rpc.gen.go 入库

### 3. 列 rpc API（B3，RPC-02/05）

- [x] 3.1 **先写测试**：`/yang/schema/:module` 响应含 rpcs 数组——B3 断言 huawei-ifm 含 rpc 列表、input 为 FieldDef、if-name mandatory+leafref；无 rpc 模块 rpcs 为空不报错
- [x] 3.2 schema 响应加 `rpcs` 字段（input 复用 FieldDef 构建逻辑）；厂商边界 BR-11 一致（仅 huawei/usmp）
- [x] 3.3 `make gen-contract` 重生成 api.gen.ts；契约漂移门禁

### 4. 门禁 + 提交

- [x] 4.1 CI regen-and-diff 校验 rpc.gen.go 零漂移（RPC-01，并入 compliance）；pr-size 排除生成物（若需）
- [ ] 4.2 覆盖率棘轮不下降（T08）；`go-code-review-check` 过；What/Why/How 提交；完成分支 PR

## 阶段② NETCONF 执行通道（后端，一 PR）

### 5. device-protocol ExecuteRPC（B1，DP-10）

- [x] 5.1 **先写测试**：ExecuteRPC 编解码 B1——input→`<rpc>` payload（命名空间=模块 ns）编码正确；`<rpc-reply>` 的 ok/数据/`<rpc-error>` 解析正确；负路径不 panic
- [x] 5.2 实现 `ExecuteRPC(ctx, module, rpc, inputs)`：scrapligo `Driver.RPC` 发送、解析 reply；断线重试/超时复用既有语义；get/edit-config 路径不动
- [x] 5.3 B1 断言读写路径行为不变（回归）

### 6. 模拟网元 custom rpc（B2，NS-09）

- [x] 6.1 **先写测试**：netconfsim B2——classifyRPC 识别 custom rpc；执行 reset-if-counters-by-name(if-name=存在接口)→校验通过→记录→ok；leafref 不存在→rpc-error；未识别 rpc→ok
- [x] 6.2 实现 custom-rpc 分发：input 校验（mandatory/leafref 存在性）+ 调用记录 + 结果注入
- [x] 6.3 **端到端集成测试**（T02）：client ExecuteRPC → sim 校验/记录 → 结果回读，覆盖成功 + 负路径（缺 mandatory / leafref 不存在）

### 7. 执行 API（B3，RPC-03）

- [ ] 7.1 **先写测试**：`POST /rpc/:ip/:module/:rpc` B3——成功执行返回结果；缺 mandatory input 拒绝不下发；设备 rpc-error 明确回传；断言**不写配置缓存/不触发对账**（D4）
- [ ] 7.2 实现 `rpc_handler.go` 执行端点；`make gen-contract`；契约漂移
- [ ] 7.3 覆盖率棘轮 + review + 提交 + PR

## 阶段③ 前端渲染执行（前端，一 PR）

### 8. rpc 渲染与执行（F1/F2，FE-19）

- [ ] 8.1 **先写测试**：rpc 派生纯函数 F1——从 schema.rpcs 派生 rpc 导航条目（与 container 平级）；执行 payload 组装（input 值→请求体）
- [ ] 8.2 模块控制台加 rpc 区（与 container Tab 平级，D6）；点 rpc 开执行面板
- [ ] 8.3 **F2 组件**：执行面板由 input FieldDef 渲染（复用 FieldRenderer）；mandatory 校验拦截执行；执行回显结果/错误
- [ ] 8.4 **F3 真浏览器**：if-name leafref 下拉（el-select 弹层）真实交互——选接口、校验态

### 9. 高危确认（F2，FE-20）

- [ ] 9.1 **先写测试**：F2——普通 rpc 执行前弹基础确认（rpc 名/input/目标设备）；高危 rpc 升级警示；取消不下发
- [ ] 9.2 实现确认对话（基础 + 高危升级样式）；接执行 API
- [ ] 9.3 leafref 下拉数据源：执行面板打开时拉目标列表（Open-Q4）；设备离线降级手工输入（R08）

### 10. 端到端 + 收尾

- [ ] 10.1 **F4 staging-smoke**：huawei-ifm 控制台出现 rpc「按接口名清除统计」、执行面板渲染 if-name、校验拦截（冒烟）
- [ ] 10.2 前端覆盖率棘轮 + typecheck + `make e2e-local` 全绿
- [ ] 10.3 review + 提交 + PR；合入后 `/opsx:sync` + `/opsx:archive`

## 收尾（跨阶段）

- [ ] 11.1 首模块 huawei-ifm 端到端真机抽验（发布前，模拟网元绿≠真机绿边界，R4）
- [ ] 11.2 文档：TESTING.md / 相关 README 补 rpc 能力定位；确认管线泛化到其余模块 rpc 自动可用
