# frontend/Dockerfile
# 构建阶段
# Node 版本与 .nvmrc / frontend-ci / 本地开发统一为 22（vitest4 要求 ≥20.12，锁 22 保证本地构建==出厂构建）
FROM node:22-alpine AS builder

WORKDIR /app

# 安装依赖
COPY package*.json ./
RUN npm install --registry=https://registry.npmmirror.com

# 复制源码
COPY . .

# 构建（组 7.2：USMP_UI_BACKEND=antd 时走 antd 测试镜像——外网 e2e-local
# 全栈门禁用；缺省 eview 真身=内网/交付构建）
ARG USMP_UI_BACKEND=
ENV USMP_UI_BACKEND=${USMP_UI_BACKEND}
RUN npm run build

# 运行阶段
FROM nginx:alpine

# 复制 Nginx 配置
COPY nginx.conf /etc/nginx/nginx.conf

# 复制构建产物
COPY --from=builder /app/dist /usr/share/nginx/html

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]
