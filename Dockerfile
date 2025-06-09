# Dockerfile (最终修正版 - 已修正路径)

# --- 阶段 1: Builder ---
FROM golang:1.23-alpine AS builder

WORKDIR /app

# 1. 复制并下载依赖 (利用缓存)
# 这一步保持不变
COPY go.mod go.sum ./
RUN go mod download

# 2. 复制所有文件
# 将项目根目录（即 gateway 目录）的所有内容复制到 /app
COPY . .

# 3. 构建应用
ENV CGO_ENABLED=0 GOOS=linux
# [核心修正] 构建命令的目标是当前目录 "."，因为 main.go 就在 /app 目录下
RUN go build -o /app/gateway_server .

# --- 阶段 2: 最终镜像 ---
FROM alpine:3.18

RUN addgroup -S appgroup && adduser -S appuser -G appgroup
WORKDIR /app

# 从 builder 阶段复制编译好的二进制文件和生产环境配置
COPY --from=builder /app/gateway_server .
# [核心修正] 配置文件路径也需要修正，/app 后面直接就是 config 目录
COPY --from=builder /app/config/config.production.yaml ./config/config.production.yaml

USER appuser
EXPOSE 8080
CMD ["./gateway_server", "--config", "./config/config.production.yaml"]