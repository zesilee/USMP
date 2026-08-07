---
name: go-122-pin
description: Go 版本已因交付要求钉死 1.22——升依赖/加语法/改CI/查覆盖率数值前必读
metadata:
  type: project
---

# Go 1.22 交付钉死（2026-08-03 拍板）

- **交付硬性要求基于 Go 1.22**，全链已钉：go.mod `go 1.22.0`、Docker 双镜像 golang:1.22-alpine、compliance/contract-drift CI 走 go-version-file、build-release.sh 缺省 `GOTOOLCHAIN=go1.22.12`（离线构建机 `USMP_GOTOOLCHAIN=local`）。
- **升级/新增依赖前必查** `go list -m -json mod@ver` 的 GoVersion ≤1.22，否则 go get 会静默把 go 指令抬上去。当前钉住的天花板：ygot v0.29.20（v0.29.22 起就要 1.23.4）、x/crypto v0.33.0（仍含 CVE-2024-45337/CVE-2025-22869 两个 SSH 修复；再往后的补丁吃不到）、x/sys v0.30、x/net v0.35、otel v1.35、x/exp@20241009（gnmi 同款）、swag v1.16.4（勿让 tidy 抬到 v1.16.6，契约生成物会漂）、beego v2.3.0（v2.3.1 起 go.mod 即 1.24.x，tidy 解析到最新版会连带抬 go 指令与 x/crypto，详见 [[gin-to-beego-migration]]）。
- **禁用 1.23+ 语法/标准库**（range-over-func、iter、unique 等）；go.mod 的 `tool` 指令（1.24+）不可用，构建工具锁定走 `backend/tools/tools.go`（build-tag `tools` 空白导入）+ `go run <pkg>`，Makefile gen-contract 与 gen-yang.sh 已改。
- **ygot v0.29.20 与 v0.34 生成物逐字节一致**（仅版本戳注释差异），regen 后放心 diff。
- **覆盖率口径坑**：Go 1.22 的 `-coverprofile ./...` 把无测试包（含巨型 generated）计入分母，总数假跌 72→13；compliance CI 已加「go list 取带测试包 + awk 过滤 profile」归一化，**本地手测覆盖率务必用同口径**，基线现值 72.5（nce-batch 记忆里的 72.3 已过时）。
- 降级只动了 go.mod/go.sum + 生成物版本戳，无任何业务代码改动；1.22.12 下 vet+全量 -race+容器冒烟+golang:1.22-alpine 镜像构建全部实测通过。
- 相关：[[release-packaging]]（打包脚本被一并钉 1.22）。
