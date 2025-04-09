# 网关服务

## 目录

- [概述](#概述)
- [目录结构](#目录结构)
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

---

## 概述

**网关服务** 是一个高性能的 API 网关服务，基于 Go 语言和 Gin 框架开发，旨在为微服务架构提供统一的入口。它通过反向代理将客户端请求路由到下游服务，同时提供请求路由、认证授权、速率限制、日志记录，全局跨域请求、全局请求 ID 和简易的请求链路追踪，能够查看客户端请求的详细信息及服务间请求耗时等功能，同时，它天然支持多端场景（如 web、app、微信），适用于云服务器部署和 Kubernetes（K8S）环境。

---

## 目录结构

`gateway` 项目采用模块化的目录结构组织，以确保可维护性、可扩展性和易于贡献。以下是目录布局的概览：

```
gateway/
├── config/
│   └── config.yaml          # 默认 YAML 配置文件
├── internal/
│   ├── config/              # 配置管理
│   │   ├── gatewayConfig.go # 网关配置解析
│   │   ├── jwt.go          # JWT 令牌处理
│   │   ├── rate_limit.go   # 速率限制实现
│   │   ├── service.go      # 下游服务管理
│   │   └── zap.go          # Zap 日志配置
│   ├── core/               # 核心初始化和工具
│   │   ├── initConfig.go   # 配置初始化
│   │   ├── jwt.go          # 核心 JWT 逻辑
│   │   ├── viper.go        # Viper 配置管理
│   │   └── zap.go          # 核心日志设置
│   ├── middleware/         # 中间件实现
│   │   ├── auth.go         # 认证中间件
│   │   ├── cors.go         # CORS 中间件
│   │   ├── error_handling.go # 错误处理中间件
│   │   ├── permission.go   # RBAC 授权中间件
│   │   ├── rate_limiting.go # 速率限制中间件
│   │   ├── request_id.go   # 请求 ID 生成中间件
│   │   ├── request_logger.go # 请求日志中间件
│   │   └── request_timeout.go # 请求超时中间件
│   └── router/             # 路由和代理逻辑
│       └── proxy.go        # 反向代理实现
├── pkg/                    # 可重用包
│   ├── constant/           # 应用程序常量
│   │   └── constant.go
│   ├── enums/              # 枚举类型
│   │   ├── platform.go     # 支持的平台（web、app、wechat）
│   │   ├── user_role.go    # 用户角色（admin、user）
│   │   └── user_status.go  # 用户状态
│   ├── middleware/         # 可重用中间件
│   │   ├── error_handling.go
│   │   ├── request_id.go
│   │   ├── request_logger.go
│   │   └── request_timeout.go
│   └── response/           # 响应处理
│       ├── code.go         # 响应代码
│       └── response.go     # 响应结构
├── go.mod                  # Go 模块定义
├── go.sum                  # 依赖校验和
├── main.go                 # 应用程序入口点
└── readme.md               # 项目文档
```

### 说明
- **`internal/`**: 包含网关核心功能的私有包，包括配置、中间件和路由逻辑，不供外部使用。
- **`pkg/`**: 包含可供其他项目导入的可重用包，如常量、枚举、中间件和响应处理程序。
- **`config/`**: 存储用于本地和开发环境的默认配置文件 (`config.yaml`)。
- **根文件**: `go.mod` 和 `go.sum` 用于依赖管理，`main.go` 为应用程序入口点，`readme.md` 提供项目文档。

这种结构支持模块化、可扩展性和 Kubernetes 兼容性，内部逻辑和可重用组件之间有清晰的关注点分离。

---

## 架构与原理

### 架构
网关服务作为客户端与下游微服务之间的反向代理，其核心组件包括：
- **请求路由**: 根据 URL 前缀将请求路由到下游服务。
- **认证中间件**: 使用 JWT 验证用户身份，支持多端场景。
- **授权中间件**: 基于角色 (RBAC) 控制服务访问。
- **跨域支持**: 为 Web 客户端提供 CORS 支持。
- **请求链路追踪**: 记录请求流转详情及服务间耗时。
- **速率限制**: 通过令牌桶算法限制请求速率。
- **日志记录**: 使用 Zap 输出结构化日志至 `stdout` 和 `stderr`。
- **配置管理**: 支持从 YAML 文件、环境变量或 K8S ConfigMap 加载配置，并具备热更新能力。

### 原理
- **单一入口点**: 所有请求通过网关处理，简化客户端逻辑。
- **模块化设计**: 中间件可独立配置和扩展。
- **K8S 兼容性**: 支持 ConfigMap、Secret 和动态配置更新。
- **高性能**: 基于 Go 和 Gin，延迟低、吞吐量高。

---

## 功能

- 请求路由到下游服务（如 `/api/user` 到 `user-service`）。
- JWT 认证支持多平台（Web、App、微信）。
- 基于角色的访问控制（RBAC）。
- 速率限制基于令牌桶算法。
- JSON 格式的结构化日志，兼容 K8S。
- 动态配置加载，支持热更新。
- 健康检查端点 `/health`。

---

## 使用方法

### 先决条件
- **Go**: 版本 1.23 或更高。
- **Docker**: 用于容器化部署（可选）。
- **Kubernetes**: 用于 K8S 部署（可选）。
- **kubectl**: 用于管理 K8S 资源（可选）。
- **Git**: 用于克隆仓库。

### 安装
1. **克隆仓库**:
   ```bash
   git clone https://github.com/your-username/gateway-service.git
   cd gateway-service
   ```
2. **安装依赖**:
   ```bash
   go mod tidy
   ```
3. **构建二进制文件**（可选，用于本地部署）:
   ```bash
   go build -o gateway ./main.go
   ```
   *注*: `main.go` 为应用程序入口点，位于根目录，详见[目录结构](#目录结构)。

### 配置
网关服务使用 YAML 配置文件定义行为，默认文件位于 `config/config.yaml`。有关文件位置的更多信息，请参阅[目录结构](#目录结构)。以下为示例配置文件：

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

- **环境变量**:
  - `APP_ENV`: 指定运行环境（默认: `development`）。
  - `CONFIG_PATH`: 指定配置文件路径（例如 K8S 中的 `/etc/config/config.yaml`）。

### 运行服务

#### 本地部署（单机环境）
1. **运行服务**:
   ```bash
   go run ./main.go
   ```
   或使用构建的二进制文件：
   ```bash
   ./gateway
   ```
2. **验证服务**:
   - 默认监听 `http://localhost:8080`。
   - 检查健康端点：
     ```bash
     curl http://localhost:8080/health
     ```

#### Docker 部署
1. **构建镜像**:
   ```bash
   docker build -t gateway:latest .
   ```
2. **运行容器**:
   ```bash
   docker run -d -p 8080:8080 \
     -v $(pwd)/config:/app/config \
     -e APP_ENV=development \
     gateway:latest
   ```

#### Kubernetes 部署
1. **准备 K8S 资源**:
   - 示例 `ConfigMap`:
     ```yaml
     apiVersion: v1
     kind: ConfigMap
     metadata:
       name: gateway-config
     data:
       config.yaml: |
         listenAddr: ":8080"
         timeout: 10s
         # ... (其余配置)
     ```
2. **部署**:
   ```bash
   kubectl apply -k k8s-config/overlays/dev
   ```
3. **验证**:
   ```bash
   kubectl get pods -n dev -l app=gateway
   ```

---

## 使用注意事项

- **配置路径**: 
  - 单机环境：确保配置文件位于 `config/` 目录（见[目录结构](#目录结构)）。
  - K8S 环境：使用 `CONFIG_PATH` 指定 ConfigMap 路径（如 `/etc/config/config.yaml`）。
- **敏感数据**: 使用 K8S `Secret` 存储 `jwtConfig.secret_key` 等敏感信息，通过环境变量加载。
- **动态更新**: 支持 ConfigMap 热加载，需确保回调逻辑正确。
- **日志记录**: 日志输出至 `stdout` 和 `stderr`，建议在 K8S 中使用日志收集器（如 Fluentd）。
- **速率限制**: 合理配置 `rateLimitConfig`，避免误限合法请求。

---

## 适用环境

- **云服务器（单机）**: 适合开发或小型部署，使用二进制或 Docker。
- **Kubernetes (K8S)**: 推荐生产环境，支持 ConfigMap、Secret 和动态配置。

---

## 贡献

欢迎贡献！请遵循以下步骤：
1. Fork 仓库。
2. 创建分支（`git checkout -b feature/你的功能`）。
3. 提交更改（`git commit -m "添加你的功能"`）。
4. 推送分支（`git push origin feature/你的功能`）。
5. 提交 Pull Request。

---

## 许可证

采用 MIT 许可证，详情见 [LICENSE](LICENSE) 文件。
