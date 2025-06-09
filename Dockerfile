# Dockerfile (此版本路径正确，无需再修改)

# --- 阶段 1: Builder ---
FROM golang:1.23-alpine AS builder

ENV CGO_ENABLED=0 GOOS=linux
WORKDIR /app

# 1. 复制模块文件
COPY go.mod go.sum ./

# 2. 复制你的所有源代码和配置文件
COPY main.go .
COPY internal ./internal
COPY config ./config

# 3. 下载依赖
# 这一步将会在配置好授权的 CI/CD 环境中成功运行
RUN go mod download

# 4. 构建应用
RUN go build -o /app/gateway_server .


# --- 阶段 2: 最终镜像 ---
FROM alpine:3.18

RUN addgroup -S appgroup && adduser -S appuser -G appgroup
USER appuser

WORKDIR /app

# 从 builder 阶段复制编译好的二进制文件和配置
COPY --from=builder /app/gateway_server .
COPY --from=builder /app/config ./config

EXPOSE 8080

CMD ["./gateway_server", "--config", "./config/config.production.yaml"]