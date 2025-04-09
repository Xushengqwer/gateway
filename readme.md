---

# 网关服务

![GitHub go.mod Go 版本](https://img.shields.io/github/go-mod/go-version/your-username/gateway-service)
![GitHub 许可证](https://img.shields.io/github/license/your-username/gateway-service)
![GitHub 最后提交](https://img.shields.io/github/last-commit/your-username/gateway-service)

**网关服务** 是一个高性能的 API 网关服务，基于 Go 语言开发，旨在为微服务架构提供统一的入口。它支持请求路由、认证授权、速率限制、日志记录,跨域请求、全局请求 ID 和简易的请求链路追踪，能够查看客户端请求的详细信息及服务间请求耗时等功能，同时，它天然支持多端场景（如 web、app、微信），适用于云服务器单机部署和 Kubernetes（K8S）环境。

## 目录

- [概述](#概述)
- [架构与原理](#架构与原理)
- [功能](#功能)
- [使用方法](#使用方法)
- [先决条件](#先决条件)
- [安装](#安装)
- [配置](#配置)
- [运行服务](#运行服务)
- [使用注意事项](#使用注意事项)
- [适用环境](#适用环境)
- [贡献](#贡献)
- [许可证](#许可证)

## 概述

网关服务 是一个轻量级、高性能的 API 网关，设计目标是为微服务架构提供统一的入口点。它通过反向代理将客户端请求路由到下游服务，同时提供跨域，认证、授权、速率限制、日志记录，全局请求链路追踪等功能。网关服务基于 Go 语言和 Gin 框架开发，结合 Zap 日志、JWT 认证等技术，适用于单机部署和 K8S 集群环境。

## 架构与原理

### 架构
网关服务作为客户端与下游微服务之间的反向代理，其架构包括以下组件：

- **请求路由**：根据 URL 前缀将传入请求路由到适当的下游服务。
- **认证中间件**：验证 JWT 令牌，支持多端令牌验证，确保只有授权用户可以访问受保护的端点。
- **授权中间件**：根据用户角色（RBAC）强制执行对服务的访问控制。
- **跨域支持**：允许 web 客户端跨域请求。
- **请求链路追踪**：记录请求流转详情，查看客户端请求信息及服务间耗时。
- **速率限制**：使用令牌桶算法限制每个 IP 的请求速率。
- **日志记录**：使用 Zap 进行结构化日志记录，普通日志输出到 `stdout`，错误日志输出到 `stderr`。
- **配置管理**：从 YAML 文件或环境变量加载配置，支持 K8S 中的 ConfigMap 并支持热更新。

### 原理
- **单一入口点**：所有客户端请求都通过网关，简化客户端逻辑并集中控制。
- **模块化中间件**：认证、速率限制等中间件模块化且可配置，便于扩展。
- **K8S 兼容性**：设计与 K8S 无缝集成，支持 ConfigMap、Secret 和动态配置更新。
- **高性能**：基于 Go 和 Gin 构建，延迟低、吞吐量高。

## 功能

- **请求路由**：根据 URL 前缀（如 `/api/user`）路由到下游服务（如 `user-service`）。
- **JWT 认证**：验证访问令牌，支持多平台场景（例如 web、app、微信）。
- **基于角色的授权**：根据用户角色（如 `admin`、`user`）限制服务访问。
- **速率限制**：使用令牌桶算法限制请求速率。
- **结构化日志**：以 JSON 格式记录请求和错误，兼容 K8S 日志系统（如 stdout/stderr）。
- **动态配置**：支持从 YAML 文件、环境变量和 K8S ConfigMap 加载配置，具有热加载能力。
- **健康检查**：提供 `/health` 端点用于监控服务状态。

## 使用方法

### 先决条件
- **Go**：版本 1.20 或更高。
- **Docker**：用于容器化部署（可选）。
- **Kubernetes**：用于 K8S 部署（可选）。
- **kubectl**：用于管理 K8S 资源（可选）。
- **Git**：用于克隆仓库。

### 安装
1. **克隆仓库**：
```bash
git clone https://github.com/your-username/gateway-service.git
cd gateway-service
```

2. **安装依赖**：
```bash
go mod tidy
```

3. **构建二进制文件**（可选，用于本地部署）：
```bash
go build -o gateway ./cmd/gateway
```

### 配置
网关服务使用 YAML 配置文件定义其行为。提供了一个示例配置文件（`config/config.development.yaml`）：

```yaml
listenAddr: ":8080"
timeout: 10s
defaultPlatform: "web"
jwtConfig:
secret_key: "your-access-secret"
issuer: "gateway"
refresh_secret: "your-refresh-secret"
zapConfig:
level: "info"
encoding: "json"
output_paths: ["stdout"]
error_outputs: ["stderr"]
rateLimitConfig:
capacity: 100
refill_interval: 1s
cleanup_interval: 5m
idle_timeout: 10m
services:
- name: "user-service"
host: "user-service"
port: 8080
prefix: "/api/user"
allowedRoles:
- "admin"
- "user"
- name: "post-service"
host: "post-service"
port: 8081
prefix: "/api/post"
allowedRoles:
- "admin"
```

- **环境变量**：
- `APP_ENV`：指定运行环境（例如 `development`、`production`）。默认：`development`。
- `CONFIG_PATH`：指定配置文件路径（例如 K8S 中的 `/etc/config/config.yaml`）。

### 运行服务

#### **本地部署（单机环境）**
1. **运行服务**：
```bash
go run ./cmd/gateway
```
或，若已构建二进制文件：
```bash
./gateway
```

2. **访问服务**：
- 服务默认监听 `http://localhost:8080`。
- 检查健康端点：
```bash
curl http://localhost:8080/health
```

#### **Docker 部署**
1. **构建 Docker 镜像**：
```bash
docker build -t gateway:latest .
```

2. **运行容器**：
```bash
docker run -d -p 8080:8080 \
-v $(pwd)/config:/app/config \
-e APP_ENV=development \
gateway:latest
```

#### **Kubernetes 部署**
1. **准备 K8S 资源**：
- 使用提供的 `k8s-config` 目录，其中包含 `ConfigMap` 和 `Deployment` 定义。
- 示例 `ConfigMap`（`k8s-config/base/gateway-config.yaml`）：
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
name: gateway-config
data:
config.yaml: |
listenAddr: ":8080"
timeout: 10s
# ... (同上)
```

2. **应用到 K8S 集群**：
```bash
kubectl apply -k k8s-config/overlays/dev
```

3. **验证部署**：
```bash
kubectl get pods -n dev -l app=gateway
```

## 使用注意事项

- **配置路径**：
- 在单机环境中，确保配置文件位于预期路径（例如 `config/config.development.yaml`）。
- 在 K8S 中，使用 `CONFIG_PATH` 环境变量指定 ConfigMap 挂载路径（例如 `/etc/config/config.yaml`）。
- **敏感数据**：
- 避免在 `ConfigMap` 中存储敏感数据（例如 `jwtConfig.secret_key`）。使用 K8S `Secret` 并通过环境变量加载。
- **动态更新**：
- 服务支持 K8S 中使用 ConfigMap 时的热加载。确保 `onConfigChange` 回调正确实现，特别是在更新日志配置等关键组件时。
- **日志记录**：
- 日志输出到 `stdout` 和 `stderr`，兼容 K8S 日志系统。使用日志收集器（例如 Fluentd、Loki）在 K8S 中聚合日志。
- **速率限制**：
- 谨慎配置 `rateLimitConfig`，避免阻塞合法流量。在生产环境中监控速率限制指标。

## 适用环境

- **云服务器（单机环境）**：
- 适合小型部署或开发环境。
- 使用二进制文件或 Docker 容器部署。
- 确保下游服务（`user-service`、`post-service`）通过网络可访问。
- **Kubernetes（K8S）**：
- 推荐用于生产环境。
- 支持 ConfigMap、Secret 和动态配置更新。
- 使用 Kustomize 或 Helm 管理环境特定配置（dev、qa、prod）。
- 确保下游服务部署为 K8S 服务，并正确解析 DNS（例如 `user-service.default.svc.cluster.local`）。

## 贡献

欢迎贡献！请遵循以下步骤：
1. Fork 该仓库。
2. 创建新分支（`git checkout -b feature/你的功能`）。
3. 进行更改并提交（`git commit -m "添加你的功能"`）。
4. 推送分支（`git push origin feature/你的功能`）。
5. 打开 Pull Request。

## 许可证

该项目采用 MIT 许可证。详情请参阅 [LICENSE](LICENSE) 文件。