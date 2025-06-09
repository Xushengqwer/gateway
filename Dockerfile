# Dockerfile (已根据你的仓库结构修正路径)

# --- 阶段 1: Builder ---
FROM golang:1.23-alpine AS builder

ENV CGO_ENABLED=0 GOOS=linux
WORKDIR /app

# 1. 复制模块文件并下载依赖
# go.mod 和 go.sum 位于仓库根目录
COPY go.mod go.sum ./
RUN go mod download

# 2. 复制所有源代码和配置文件
# main.go, internal/, config/ 等都位于仓库根目录
COPY main.go .
COPY internal ./internal
COPY config ./config
# 如果你的项目有 pkg 目录，也一并复制
# COPY pkg ./pkg

# 3. 构建应用
# 此时，我们已经在 /app 目录下，可以直接构建
RUN go build -o /app/gateway_server .


# --- 阶段 2: 最终镜像 ---
FROM alpine:3.18

RUN addgroup -S appgroup && adduser -S appuser -G appgroup
USER appuser

WORKDIR /app

# 从 builder 阶段复制编译好的二进制文件
COPY --from=builder /app/gateway_server .

# 从 builder 阶段复制 config 目录
# 这样可以确保任何在构建阶段可能生成或修改的配置也能被包含进来
COPY --from=builder /app/config ./config

EXPOSE 8080

# CMD 默认加载生产配置文件
# 容器内的路径是固定的 /app/config/
CMD ["./gateway_server", "--config", "./config/config.production.yaml"]