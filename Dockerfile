# Dockerfile (最终修正版)

# --- 阶段 1: Builder ---
# 使用官方的 Go alpine 镜像作为构建环境
FROM golang:1.23-alpine AS builder

# 设置必要的环境变量
ENV CGO_ENABLED=0 GOOS=linux
WORKDIR /app

# 1. 复制模块文件
# 单独复制以利用 Docker 的层缓存机制
COPY go.mod go.sum ./

# 2. 复制所有源代码和配置文件
# 这是最关键的一步，确保所有需要的模块都被复制
COPY main.go .
COPY internal ./internal
COPY config ./config
# [核心修复] 将 go-common 模块也复制进来，这样 go.mod 中的 replace 指令才能生效
COPY go-common ./go-common

# 3. 下载依赖
# 由于 go-common 文件夹已存在，这一步会正确使用本地模块
RUN go mod download

# 4. 构建应用
# 将编译后的二进制文件输出到 /app/gateway_server
RUN go build -o /app/gateway_server .


# --- 阶段 2: 最终镜像 ---
# 使用轻量的 alpine 镜像作为最终的运行环境
FROM alpine:3.18

# 1. 创建非 root 用户和用户组，增强安全性
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

# 2. 创建应用目录
WORKDIR /app

# 3. 从 builder 阶段复制必要的文件
# 只复制编译好的二进制文件
COPY --from=builder /app/gateway_server .
# [优化] 只复制生产环境需要的配置文件，减小镜像体积
COPY --from=builder /app/config/config.production.yaml ./config/config.production.yaml

# 4. 切换到非 root 用户运行
USER appuser

# 5. 暴露端口
EXPOSE 8080

# 6. 容器启动命令
# 指定使用生产环境的配置文件
CMD ["./gateway_server", "--config", "./config/config.production.yaml"]