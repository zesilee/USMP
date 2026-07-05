#!/usr/bin/env bash
#
# setup-mac-runner.sh —— 在 macOS 上安装并注册 USMP 的 GitHub 自托管 Runner。
#
# 用途：为 .github/workflows/e2e-staging.yml 准备一台带标签 `macos-staging` 的自托管 Runner，
#       并以「登录自启服务」方式常驻，让 merge→main 后自动部署 + 常驻本地 staging。
#
# 前置（本脚本会检查）：
#   • Docker Desktop 已安装且在运行（建议在其设置里开启 "Start Docker Desktop when you sign in"）
#   • Node ≥ 20（前端 npm ci / Playwright 需要）
#
# 注册 token 必须由你在 GitHub 现取（无法代取）：
#   仓库 → Settings → Actions → Runners → New self-hosted runner → macOS
#   复制页面上 `./config.sh --url ... --token XXXX` 里的 token。
#
# 用法：
#   scripts/setup-mac-runner.sh <RUNNER_TOKEN> [REPO_URL]
#   REPO_URL 默认 https://github.com/zesilee/USMP
#
# 安全：本仓库已转为 PRIVATE —— 这是自托管 Runner 的前提（公开仓库会让 fork PR 在本机 RCE）。

set -euo pipefail

REPO_URL="${2:-https://github.com/zesilee/USMP}"
RUNNER_TOKEN="${1:-}"
RUNNER_DIR="${RUNNER_DIR:-$HOME/actions-runner-usmp}"
RUNNER_NAME="${RUNNER_NAME:-$(hostname -s)-usmp}"
LABELS="macos-staging"

die() { echo "❌ $*" >&2; exit 1; }
info() { echo "▶ $*"; }

[ -n "$RUNNER_TOKEN" ] || die "缺少注册 token。用法: $0 <RUNNER_TOKEN> [REPO_URL]（token 从 GitHub Settings→Actions→Runners 获取）"
[ "$(uname -s)" = "Darwin" ] || die "本脚本仅用于 macOS（自托管 Runner 目标机）。"

# ── 前置依赖检查 ───────────────────────────────
info "检查依赖…"
command -v docker >/dev/null 2>&1 || die "未找到 docker。请先安装 Docker Desktop：https://www.docker.com/products/docker-desktop"
docker info >/dev/null 2>&1 || die "Docker 守护进程未运行。请启动 Docker Desktop 后重试。"
command -v node >/dev/null 2>&1 || die "未找到 node。请安装 Node ≥ 20（brew install node）。"
NODE_MAJOR="$(node -v | sed 's/^v\([0-9]*\).*/\1/')"
[ "$NODE_MAJOR" -ge 20 ] || die "Node 版本过低（$(node -v)），前端需 ≥ 20。"
echo "  ✅ docker / node 就绪"

# ── 选择 Runner 架构 ──────────────────────────
case "$(uname -m)" in
  arm64) RUNNER_ARCH="arm64" ;;
  x86_64) RUNNER_ARCH="x64" ;;
  *) die "未知架构 $(uname -m)" ;;
esac

# ── 获取最新 Runner 版本 ──────────────────────
info "查询最新 actions-runner 版本…"
LATEST="$(curl -fsSL https://api.github.com/repos/actions/runner/releases/latest \
  | grep -o '"tag_name": *"v[^"]*"' | head -1 | sed 's/.*"v\([^"]*\)".*/\1/')"
[ -n "$LATEST" ] || die "无法获取 Runner 版本（网络？）。"
TARBALL="actions-runner-osx-${RUNNER_ARCH}-${LATEST}.tar.gz"
URL="https://github.com/actions/runner/releases/download/v${LATEST}/${TARBALL}"
echo "  版本 v${LATEST}（osx-${RUNNER_ARCH}）"

# ── 下载并解包 ────────────────────────────────
mkdir -p "$RUNNER_DIR"
cd "$RUNNER_DIR"
if [ ! -f config.sh ]; then
  info "下载 $TARBALL …"
  curl -fsSL -o "$TARBALL" "$URL"
  tar xzf "$TARBALL"
  rm -f "$TARBALL"
else
  info "已存在 Runner 安装（$RUNNER_DIR），跳过下载。"
fi

# ── 配置（幂等：已配置则先移除）───────────────
if [ -f .runner ]; then
  info "检测到已有配置，先移除旧注册…"
  ./config.sh remove --token "$RUNNER_TOKEN" || true
fi
info "注册 Runner（labels=$LABELS）…"
./config.sh --unattended \
  --url "$REPO_URL" \
  --token "$RUNNER_TOKEN" \
  --name "$RUNNER_NAME" \
  --labels "$LABELS" \
  --replace

# ── 安装为登录自启服务 ────────────────────────
info "安装为后台服务（登录自启）…"
./svc.sh install
./svc.sh start

echo ""
echo "✅ 完成。Runner「$RUNNER_NAME」已注册并作为服务运行（标签：$LABELS）。"
echo "   验证：仓库 → Settings → Actions → Runners 应显示为 Idle。"
echo "   之后 merge→main 会自动触发 e2e-staging，部署并常驻本地 staging："
echo "     前端 http://localhost:3002   后端 http://localhost:8080/api/v1"
echo ""
echo "   管理：cd $RUNNER_DIR && ./svc.sh {status|stop|start|uninstall}"
