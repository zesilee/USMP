# 构建阶段
FROM golang:1.26-alpine AS builder

WORKDIR /app

# 安装依赖
RUN apk add --no-cache git

# 复制 go mod 文件
COPY go.mod go.sum ./
RUN go mod download

# 复制源代码
COPY . .

# 构建控制器
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o manager main.go

# 运行阶段
FROM alpine:3.19

WORKDIR /

# 安装基本工具
RUN apk add --no-cache ca-certificates tzdata

# 设置时区
ENV TZ=Asia/Shanghai

# 复制二进制文件
COPY --from=builder /app/manager /manager

# 健康检查：命中真实存在的 GET 路由（服务监听 :8080，且无 /healthz）。
# 用 GET（-O /dev/null）而非 --spider 的 HEAD —— gin 仅注册了 GET，HEAD 会 404。
HEALTHCHECK --interval=30s --timeout=10s --start-period=10s --retries=3 \
  CMD wget -q -O /dev/null http://localhost:8080/api/v1/yang/modules || exit 1

# 非 root 用户运行
RUN adduser -D usmp
USER usmp

# 启动命令
ENTRYPOINT ["/manager"]
