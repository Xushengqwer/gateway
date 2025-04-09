# 构建阶段
FROM golang:1.23-alpine AS builder

# 设置工作目录
WORKDIR /app

# 复制 go.mod 和 go.sum 文件
COPY go.mod go.sum ./

# 下载依赖
RUN go mod download

# 复制项目文件
COPY . .

# 构建项目
RUN go build -o gateway ./main.go

# 运行阶段
FROM alpine:latest

# 设置工作目录
WORKDIR /app

# 从构建阶段复制二进制文件
COPY --from=builder /app/gateway .

# 复制配置文件（如果有）
COPY config/config.yaml /app/config/config.yaml

# 暴露端口
EXPOSE 8080

# 运行命令
CMD ["./gateway"]