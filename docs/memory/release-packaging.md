---
name: release-packaging
description: 一键编译打包发布 zip 的入口、包结构与设计约束——做交付/部署/改打包链路前必读
metadata:
  type: project
---

# 发布打包（2026-08-03 交付，PR#255）

- 入口：`./scripts/build-release.sh` → `release/usmp-release-<git短sha>.zip`（`/release/` 已 gitignore，注意写的是根锚定 `/release/`，裸 `release/` 会误伤 `scripts/release/`）。
- 包结构：`usmp/{bin/usmp-backend, bin/usmp-web, web/, start.sh, VERSION}`；用法 = 解压后 `sh usmp/start.sh`（前端 :3002 可用 USMP_WEB_PORT 改，后端固定 :8080，USMP_SEED_DEVICE 透传）。
- **前端托管不用 nginx**：目标 docker 镜像不预装任何东西，静态站是自研零依赖 Go 二进制 `cmd/staticserver` + `internal/staticweb`（行为对齐 nginx try_files：SPA fallback + /healthz；前端本就直连 :8080 绝对 baseURL，无需 /api 代理）。
- `start.sh` 是 POSIX sh（alpine 无 bash），可直接当容器 ENTRYPOINT：开头 chmod 自愈可执行位（python zipfile 解压会丢 +x）、任一子进程退出→整体非零退出交编排重启、TERM/INT 优雅回收（alpine 容器实测 exit=0）。
- 打包兜底链：`zip` 命令缺失时用 python3 zipfile（ZipInfo.from_file 保留可执行位）；打包后强制完整性 + 必备条目双重校验。
- 校验手法：交叉编译产物用 `go version -m <bin> | grep GOOS=` 验目标平台，比 `file` 可靠且不依赖额外工具。
- 目标平台缺省 linux/amd64，USMP_GOOS/USMP_GOARCH 可覆盖。
- 冒烟验证法：本机 8080/3002 常被常驻 staging（或 [[test-server-process-leak]]）占用，别杀——把 zip 解压挂进干净 alpine 容器里跑 start.sh 验证即可（同时正好复现真实交付场景）。
