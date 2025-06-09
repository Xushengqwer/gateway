# gateway/Dockerfile

# --- 阶段 1: Builder ---
FROM golang:1.23-alpine AS builder

ENV CGO_ENABLED=0 GOOS=linux
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY ./gateway /app/gateway
COPY ./go-common /app/go-common # 假设 go-common 在同级目录

WORKDIR /app/gateway
RUN go build -o /app/gateway_server ./main.go

# --- 阶段 2: 最终镜像 ---
FROM alpine:3.18

RUN addgroup -S appgroup && adduser -S appuser -G appgroup
USER appuser

WORKDIR /app

# 从 builder 阶段复制编译好的二进制文件
COPY --from=builder /app/gateway_server .

# [关键] 复制整个 config 目录到容器中
# 这样无论 development.yaml 还是 production.yaml 都在容器的 /app/config/ 路径下
COPY ./gateway/config ./config

EXPOSE 8080

# [关键] CMD 默认加载生产配置文件
# 在线上环境中，这个路径是固定的。
# 如果需要，你仍然可以在 docker run 时覆盖整个 CMD。
CMD ["./gateway_server", "--config", "./config/config.production.yaml"]