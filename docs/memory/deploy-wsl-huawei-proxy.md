---
name: deploy-wsl-huawei-proxy
description: 在华为内网 WSL Ubuntu 上 docker compose 部署 uSMP 的踩坑与解法（TLS 拦截/代理/网段冲突）
metadata: 
  node_type: memory
  type: project
  originSessionId: 014f7a79-ea21-4958-af4f-6af2e70396f4
---

2026-07-07 在另一台 WSL Ubuntu（华为内网，主机名 DESKTOP-1CJ11AT，clone 到 /home/leezesi/code/USMP）用 docker compose 全栈部署 uSMP（simulator+backend+frontend，`make staging-up` 那套），跑通。全程在**独立 clone**，与主 vultr 机器不同步——改动要各自做。

**部署形态**：Docker 全栈 staging（前端 3002 / 后端 8080 / simulator 固定 192.168.1.1:830，对齐后端硬编码种子设备）。构建不需要 yang-models 子模块（生成物已提交），不需要 .env。

**核心坑=华为内网**，三镜像逐个 `DOCKER_BUILDKIT=0 docker build --build-arg HTTP_PROXY/HTTPS_PROXY/NO_PROXY -t usmp-{frontend,controller,simulator}:latest`，再 `docker compose up -d --pull never`：
1. **TLS 拦截**：代理 MITM 换华为自签根 CA，容器不信任 → apk `certificate not trusted`、npm `SELF_SIGNED_CERT_IN_CHAIN`。解：apk 源改 http；前端加 `ENV NODE_TLS_REJECT_UNAUTHORIZED=0` + `NPM_CONFIG_STRICT_SSL=false`；go 用 `GOPROXY=http://mirrors.tools.huawei.com/goproxy/`+`GOSUMDB=off`+`GOINSECURE=*`。
2. **代理地址**：宿主 `http(s)_proxy=127.0.0.1:3131`（本地代理，容器够不到）；`proxyjp.huawei.com:8080` 要账号密码弃用。容器构建用**宿主 eth0 IP `<wsl-ip>:3131`**（3131 监听 0.0.0.0，bridge 容器可路由到宿主 IP）。busybox wget 不支持 https-over-proxy(CONNECT)，测试要用 `npm view` 别用 wget。
3. **网段冲突**：docker0 默认网桥占了 `192.168.1.0/24`，与 compose 的 usmp-net 撞（`Pool overlaps`）。解：`/etc/docker/daemon.json` 设 `{"bip":"172.28.0.1/24"}`（避开宿主 172.24.160.0/20、企业 172.30.x、k8s 10.96/12、minikube 192.168.39/49/59/76），清残留 `ip link delete docker0`，重启 docker，腾出 192.168.1.0/24。

**两条铁律**：① 这些 Dockerfile hack（TLS 关校验/GOINSECURE/apk http）是**内网环境特定**，禁止 commit 进 main（触发 CI 合规 + 污染正常环境），只留本地。② WSL 重启后 eth0 IP 变，运行期不受影响，但**重新构建**镜像要换 build-arg 里的代理 IP；daemon.json 的 bip 持久。

相关：[[frontend-ci-gotchas]] [[cicd-self-hosted]]
