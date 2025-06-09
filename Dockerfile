# Dockerfile (使用 COPY . . 的简洁版本)

# --- 阶段 1: Builder ---
FROM golang:1.23-alpine AS builder

WORKDIR /app

# 1. 复制并下载依赖 (利用缓存)
COPY go.mod go.sum ./
RUN go mod download

# 2. 复制所有文件
# 这种方式更简单，但缓存效率较低
COPY . .

# 3. 构建应用
ENV CGO_ENABLED=0 GOOS=linux
RUN go build -o /app/gateway_server ./gateway/main.go

# --- 阶段 2: 最终镜像 ---
FROM alpine:3.18

RUN addgroup -S appgroup && adduser -S appuser -G appgroup
WORKDIR /app

# 从 builder 阶段复制编译好的二进制文件和生产环境配置
COPY --from=builder /app/gateway_server .
COPY --from=builder /app/gateway/config/config.production.yaml ./config/config.production.yaml

USER appuser
EXPOSE 8080
CMD ["./gateway_server", "--config", "./config/config.production.yaml"]