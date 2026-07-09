# YANG Generated Code

本目录存放 ygot 从 YANG 模型生成的 Go 强类型结构体（R04：禁止手写）。
生成由**厂商 manifest 驱动**，入口统一为仓库根的 `make gen-yang`。

## 目录结构

```
<vendor-pkg>/          每厂商一个 Go 包（目录名 = package 名）
├── gen.conf           声明式生成配置（YANG 路径 + 模块列表 + 选项）
├── all.gen.go         ygot 生成物（单文件，勿手改）
└── doc.go             包文档（手写，可选）
```

当前厂商包：`huawei/`（huawei-vlan / huawei-ifm / huawei-system / huawei-pub-type / huawei-extension）、
`openconfig/`（openconfig-vlan / openconfig-interfaces，schema 离线回退源）。

## 重新生成

```bash
make gen-yang                # 全量
make gen-yang VENDOR=huawei  # 单厂商包
```

管线：`scripts/gen-yang.sh` 扫描 `*/gen.conf` → ygot generator（版本由
`backend/go.mod` 锁定）→ `backend/tools/genfix` 后处理（跨平台修复枚举
标识符 `|` + 规范化生成头部机器路径）→ gofmt。

前置：huawei 需要 yang-models submodule（`git submodule update --init yang-models`）。

## 新增厂商 / 新增模块

- **新增模块**（已有厂商）：把模块名加进该厂商 `gen.conf` 的 `modules=`，跑 `make gen-yang VENDOR=<pkg>`。
- **新增厂商**：新建 `internal/generated/<vendor>/` 目录 + `gen.conf`，跑 `make gen-yang`——脚本与 Makefile 零改动。
  接入下发链路另需注册驱动描述符，见 `backend/internal/drivers/` 与 `openspec/specs/device-driver-registry/spec.md`。

`gen.conf` 键（`yang_path` 相对仓库根）：

```
yang_path=yang-models/network-router/8.20.10/ne40e-x8x16
modules=huawei-vlan huawei-ifm huawei-system huawei-pub-type huawei-extension
generate_fakeroot=true
compress_paths=false
```

## 约束与背景

1. **生成物勿手改**：CI 以 regen-and-diff 验证（重跑 `make gen-yang` 断言零漂移）——
   生成物改动合法当且仅当可由管线复现。改 YANG 模型或 `gen.conf` 后重新生成提交。
2. **单文件输出**：ygot 跨模块类型引用/全局枚举映射/合并 gzip schema 要求单编译单元，
   每包只有一个 `all.gen.go`（openconfig 生态标准做法）。
3. **每包唯一 Device 根**：`generate_fakeroot=true` 只能出现一次，新模块并入既有 `modules=` 列表而非另开生成。
4. **枚举 `|` 修复**：华为 YANG 枚举名含 `|`（非法 Go 标识符），由 `tools/genfix` 跨平台修复为 `_OR_`
   （YANG 原值字符串映射保持原样）。
