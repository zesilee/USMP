# 构建阶段
FROM golang:1.21-alpine AS builder

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

# 暴露健康检查
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8081/healthz || exit 1

# 非 root 用户运行
RUN adduser -D usmp
USER usmp

# 启动命令
ENTRYPOINT ["/manager"]
