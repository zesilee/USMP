# WSL Ubuntu · 企业内网部署指南

在 **WSL2 Ubuntu + 企业内网（TLS 拦截代理）** 环境用 `docker compose` 全栈部署 uSMP
（simulator + backend + frontend）。已在真实拦截代理内网实测跑通。

> 普通环境（无拦截代理、无网段冲突）请直接看 README 的 `make setup` / `make staging-up`。
> 本文只覆盖内网环境额外要处理的三件事：**TLS 拦截、代理地址、Docker 网段冲突**。

---

## 0. 前置

- WSL2 里可用的 Docker（原生 Engine + systemd，或 Docker Desktop WSL 集成）。
- 已 `git clone` 本仓库（如为私有仓库，用 `gh auth login` 或 PAT 鉴权）。
- 构建**无 submodule 依赖**（YANG 模型源 `snd/ce6866p-yang` 已入库，ygot 生成物已提交），**不需要** `.env`。
- 记下你的 WSL eth0 IP 和本地代理端口：
  ```bash
  ip addr show eth0 | grep 'inet '     # 记下 eth0 的 IPv4
  env | grep -i proxy                  # 记下本地代理端口
  ```
  下文用 `<WSL_IP>` 和 `<PORT>` 代指你的实际值。

> ⚠️ 本地代理常监听 `127.0.0.1:<PORT>`，**容器里的 127.0.0.1 指向容器自己**，够不到宿主代理。
> 只要该代理监听在 `0.0.0.0`（多数是），容器就能用**宿主 IP** `<WSL_IP>:<PORT>` 访问它。
> 若企业主代理需账号密码、不适合免交互构建，用本地免鉴权代理转发即可。

---

## 1. 处理 TLS 拦截（改三个 Dockerfile，仅本地）

内网代理会拦截 HTTPS 并换成企业自签根 CA，容器不信任 → apk `certificate not trusted`、
npm `SELF_SIGNED_CERT_IN_CHAIN`、go 证书错。构建期关掉校验即可（本地 staging，安全可接受）。

**`frontend/Dockerfile`** — `npm ci` 之前加：
```dockerfile
ENV NODE_TLS_REJECT_UNAUTHORIZED=0
ENV NPM_CONFIG_STRICT_SSL=false
```

**`backend/Dockerfile` 与 `backend/Dockerfile.simulator`** — `WORKDIR /app` 之后加
（`GOPROXY` 换成你所在网络可直连的 Go module 镜像）：
```dockerfile
ENV GOPROXY=<企业内部 GOPROXY 镜像>
ENV GOSUMDB=off
ENV GOINSECURE=*
```

**apk（三个 Dockerfile 的每个 `apk add` 之前）** 把源改 http 绕过证书校验：
```dockerfile
RUN sed -i 's|https://|http://|g' /etc/apk/repositories 2>/dev/null; \
    sed -i 's|https://|http://|g' /etc/apk/repositories.d/* 2>/dev/null; \
    apk add --no-cache <原有的包>
```

**另加 `frontend/.dockerignore`**（挡住宿主 node_modules 被 `COPY . .` 灌进镜像，
否则宿主平台的原生二进制会覆盖容器内 `npm ci` 结果）：
```
node_modules
dist
.git
coverage
storybook-static
```

> 🚫 **这些改动禁止 commit 进 main**：属内网环境特定 hack，会触发 CI 合规检查、污染正常环境。
> 只留本地（`git stash` 或单独的 `local-only` 分支）。

---

## 2. 腾出 `192.168.1.0/24` 网段

compose 的 `usmp-net` 固定用 `192.168.1.0/24`、simulator 落 `192.168.1.1`
（对齐 backend 硬编码种子设备，**不可改**）。若 Docker 默认网桥 docker0 也占了这个段
（现象：`up` 时报 `Pool overlaps with other one on this address space`），把 docker0 挪走。

先看宿主已占哪些段，挑一个不冲突的给 docker0：
```bash
ip -o -4 addr show      # 列出宿主已占用的网段
docker network inspect bridge --format '{{range .IPAM.Config}}{{.Subnet}}{{end}}'
```

设 `/etc/docker/daemon.json`（`172.28.0.0/24` 需避开你所在网络已占用的网段：
WSL 宿主段 / 企业内网段 / 10.96.0.0/12 k8s / 192.168.39·49·59·76 minikube；按你机器实际调整）：
```bash
sudo tee /etc/docker/daemon.json >/dev/null <<'EOF'
{ "bip": "172.28.0.1/24" }
EOF

# 清掉可能残留的旧 docker0 网卡（忽略报错）
sudo ip link set docker0 down 2>/dev/null; sudo ip link delete docker0 2>/dev/null; true

sudo service docker restart      # 或 sudo systemctl restart docker
docker network inspect bridge --format '{{range .IPAM.Config}}{{.Subnet}}{{end}}'
# 应打印 172.28.0.0/24 → 192.168.1.0/24 已空出
```
> 若 docker 起不来：`sudo dockerd 2>&1 | head -30` 看报错，多半是选的段又撞了，换一个。

---

## 3. 构建三个镜像

```bash
cd <仓库根>
PX=http://<WSL_IP>:<PORT>                                   # 你的宿主 IP + 本地代理端口
NOPX=localhost,127.0.0.1,<企业内部直连域名后缀>,<GOPROXY 镜像主机>

DOCKER_BUILDKIT=0 docker build \
  --build-arg HTTP_PROXY=$PX --build-arg HTTPS_PROXY=$PX --build-arg NO_PROXY=$NOPX \
  -t usmp-frontend:latest ./frontend

DOCKER_BUILDKIT=0 docker build \
  --build-arg HTTP_PROXY=$PX --build-arg HTTPS_PROXY=$PX --build-arg NO_PROXY=$NOPX \
  -t usmp-controller:latest -f backend/Dockerfile ./backend

DOCKER_BUILDKIT=0 docker build \
  --build-arg HTTP_PROXY=$PX --build-arg HTTPS_PROXY=$PX --build-arg NO_PROXY=$NOPX \
  -t usmp-simulator:latest -f backend/Dockerfile.simulator ./backend
```

- `HTTP_PROXY/HTTPS_PROXY/NO_PROXY` 是 Docker 预置 build-arg，自动注入构建 `RUN` 环境，
  apk/npm/go 都会走代理，无需在 Dockerfile 声明 ARG。
- `NO_PROXY` 含 GOPROXY 镜像主机 → go 走内部镜像直连、不绕外部代理。
- 三个 tag 正好是 `docker-compose.yml` 里 `image:` 的名字。

**验证代理能用**（busybox wget 不支持 https-over-proxy，用 npm 测）：
```bash
docker run --rm -e http_proxy=$PX -e https_proxy=$PX -e no_proxy=$NOPX \
  node:22-alpine sh -c 'npm view vue version --registry=https://registry.npmjs.org'
# 打印版本号 = 代理通；SELF_SIGNED_CERT_IN_CHAIN = TLS 拦截（靠第 1 节的关校验解决）
```

---

## 4. 起栈

```bash
docker compose up -d --pull never       # 只用本地已建镜像，不联网拉取
docker compose ps                        # 三个容器均 healthy
curl -s http://localhost:8080/api/v1/yang/modules | head
```

- 前端：http://localhost:3002
- 后端 API：http://localhost:8080/api/v1
- simulator：`192.168.1.1:830`（对齐后端种子设备，设备显示**在线**）

---

## 5. 复发排查速查

| 现象 | 根因 | 处理 |
|------|------|------|
| apk `certificate not trusted` | TLS 拦截 | apk 源改 http（§1） |
| npm `SELF_SIGNED_CERT_IN_CHAIN` | TLS 拦截 | `NODE_TLS_REJECT_UNAUTHORIZED=0`（§1） |
| npm `ETIMEDOUT` 全部 tgz | 容器没走代理 / 代理地址是 127.0.0.1 | build-arg 用 `<WSL_IP>:<PORT>`（§3） |
| npm `Exit handler never called!` | 上游 ETIMEDOUT 后 npm 崩 | 同上，先解决网络 |
| `Pool overlaps ... address space` | docker0 占了 192.168.1.0/24 | 挪 docker0 的 bip（§2） |
| `pull access denied for usmp-*` | 本地没有该镜像 / tag 没打上 | 确认 `docker images \| grep usmp`，`up` 加 `--pull never` |

## 6. 重要提醒

1. **WSL 重启后 eth0 IP 会变**。运行期容器不用代理，不受影响；但**重新构建**镜像时要
   `ip addr show eth0` 拿新 IP 换掉 `PX`。`daemon.json` 的 `bip` 是持久的，不用重设。
2. **第 1 节的 Dockerfile 改动只留本地，禁止合入 main。** 正常/CI 环境用原始 Dockerfile。
3. 代理只在**构建期**需要；`docker compose up` 用已建镜像，不再联网。
