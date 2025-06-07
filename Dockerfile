# Dockerfile

# ---- 第一阶段：构建阶段 ----
FROM golang:1.23-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o gateway ./main.go

# ---- 第二阶段：运行阶段 ----
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

COPY --from=builder /app/gateway .

# 关键修改：将配置文件复制到容器内一个固定的、明确的路径
COPY config/development.yaml /app/config/config.yaml

# 暴露端口
EXPOSE 8080

# 关键修改：CMD 中明确指定我们复制进去的配置文件路径
CMD ["./gateway", "--config=/app/config/config.yaml"]