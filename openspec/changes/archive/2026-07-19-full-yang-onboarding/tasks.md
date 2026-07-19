# Tasks: full-yang-onboarding

> TDD（T05/T07）：每项先红后绿。改动类型→必补层（§5.6）：管线→B1、注册/编解码→B1+B2、API→B3、导航→F4。

## 1. 管线与生成（CG-04）

- [x] 1.1 gen-yang.sh：yang_path 逗号分隔多目录 + `-ignore_unsupported`（试验已验证）
- [x] 1.2 `usmp-deviations.yang`：五条豁免各注原因与影响面（试验已验证可生成）
- [x] 1.3 gen.conf modules 7→58，pic 延期注释；`make gen-yang` 再生成零漂移（CI regen-and-diff 兜底）
- [x] 1.4 taskname go:generate 模块清单同步 + 再生成 taskname.gen.go
- [x] 1.5 B1 红灯→绿：schema 加载断言 67 根容器全集（含新模块代表样本）

## 2. 表驱动注册与路径约定（DR-06）

- [x] 2.1 B1 红灯：注册表不变量参数化测试（namespace 非空唯一 / SchemaTree 入口存在 / 根名路径 route·decode·encode 三谓词命中 / -race）
- [x] 2.2 B1 红灯：tnlm/rtp 根名路径命中（断链回归）；ni 双口径（根名 + `/ni:` 兼容）命中
- [x] 2.3 实现：`plainModules` 表 49+4 行 + `registerPlain` 循环；tnlm/xpl/rtp/acl 迁表；ni 双谓词；绿
- [x] 2.4 B1 红灯→绿：参数化编解码往返（每模块 schema 采样最小实例 Encode→Decode 相等）

## 3. 泛型控制器（D4）

- [x] 3.1 B1 红灯：plainmodule reconciler 单测（Get 解码 / 整根收敛 / Set 变更映射，mock client）
- [x] 3.2 实现：`internal/controller/plainmodule` + main.go 描述符循环批量注册；绿
- [x] 3.3 B2 红灯→绿：sim 集成——代表模块（每任务域≥1：ntp/lldp/mstp/vrrp/sflow/ospfv2/arp/evpn/hwtacacs/qos 等）写→回读→收敛（`testing.Short` 跳过）

## 4. API 与导航（B3/LT-04）

- [x] 4.1 B3 红灯→绿：每模块根路径 convertConfig 编包成功（参数化）
- [x] 4.2 B1 红灯→绿：LT-04 基线——左树恰 60 叶可用、pic 占位（缩水即红）
- [x] 4.3 F4：staging smoke 增「左树全量可用 + 新模块（如 ntp）控制台 Tab 渲染」冒烟；`make e2e-local` 全绿

## 5. 收口

- [x] 5.1 后端全量 `go test ./...`（含 -race）+ 覆盖率棘轮不降；前端套件全绿（零前端代码改动）
- [x] 5.2 code review + What/Why/How 提交 + PR + CI 全绿
- [x] 5.3 `/opsx:sync` 三 delta → 主 spec；`/opsx:archive`；memory 更新
