package router

import (
	"fmt"
	"gateway/internal/config" // 网关配置包
	"gateway/internal/core"   // 核心工具包
	mymiddleware "gateway/internal/middleware"
	"gateway/pkg/constant"   // 常量包
	"gateway/pkg/middleware" // 共享中间件包
	"go.uber.org/zap"        // Zap 日志库
	"net/http"               // HTTP 相关
	"net/http/httputil"      // HTTP 工具包，用于反向代理
	"net/url"                // URL 解析
	"strings"                // 字符串操作

	"github.com/gin-gonic/gin" // Gin 框架
)

// SetupProxyRoutes 设置网关的代理路由
// - 输入: r *gin.Engine Gin 引擎实例
// - 输入: cfg *config.GatewayConfig 网关配置
// - 输入: logger *core.ZapLogger 日志实例
// - 输入: jwtUtil core.JWTUtilityInterface JWT 工具实例
// - 意图: 根据配置将请求转发到下游服务，支持单机和 K8s 环境，并应用中间件
func SetupProxyRoutes(r *gin.Engine, cfg *config.GatewayConfig, logger *core.ZapLogger, jwtUtil core.JWTUtilityInterface) {
	// 1. 注册全局中间件
	// - RequestIDMiddleware: 生成或复用请求 ID，网关始终生成新的 ID
	// - RequestLoggerMiddleware: 记录请求日志，包含请求详情
	// - RequestTimeoutMiddleware: 设置请求超时，防止下游服务响应过慢
	r.Use(middleware.RequestIDMiddleware(logger.Logger(), true))
	r.Use(middleware.RequestLoggerMiddleware(logger.Logger(), true))
	r.Use(middleware.RequestTimeoutMiddleware(logger, constant.RequestTimeout))

	// 2. 为每个服务注册代理路由
	// - 遍历配置中的服务，动态设置反向代理，支持单机和 K8s 环境
	for _, svc := range cfg.Services {
		var targetHost string
		var port int
		scheme := svc.Scheme
		if scheme == "" {
			scheme = "http" // 默认协议为 http
		}

		// 3. 判断环境并构建目标主机地址
		if svc.ServiceName != "" {
			// K8s 环境：使用 ServiceName 和 Namespace 构建 DNS 名称
			namespace := svc.Namespace
			if namespace == "" {
				namespace = "default" // 未指定命名空间时，默认使用 "default"
			}
			targetHost = fmt.Sprintf("%s.%s.svc.cluster.local", svc.ServiceName, namespace)
			port = svc.Port
			if port == 0 {
				port = 80 // K8s 环境下未指定端口时，默认使用 80
			}
		} else {
			// 单机环境：使用 Host 和 Port
			if svc.Host == "" || svc.Port == 0 {
				logger.Fatal("Invalid service config for non-K8s mode",
					zap.String("service", svc.Name),
					zap.String("host", svc.Host),
					zap.Int("port", svc.Port))
			}
			targetHost = svc.Host
			port = svc.Port
		}

		// 4. 构建下游服务 URL
		// - 使用 scheme://host:port 格式，支持 http 或 https
		targetURL, err := url.Parse(fmt.Sprintf("%s://%s:%d", scheme, targetHost, port))
		if err != nil {
			logger.Fatal("Invalid service URL",
				zap.String("service", svc.Name),
				zap.Error(err))
		}

		// 5. 创建反向代理
		// - 使用 httputil.NewSingleHostReverseProxy 创建代理实例
		proxy := httputil.NewSingleHostReverseProxy(targetURL)

		// 6. 设置代理的 Director
		// - 自定义 Director，确保请求的 Scheme、Host 和 Path 正确转发
		proxy.Director = func(req *http.Request) {
			req.URL.Scheme = targetURL.Scheme
			req.URL.Host = targetURL.Host
			req.URL.Path = strings.TrimPrefix(req.URL.Path, svc.Prefix)

			// 7. 透传请求 ID
			// - 从 gin.Context 中获取 request_id 并设置到请求头，便于下游服务追踪
			if requestID, exists := req.Context().Value(constant.RequestIDKey).(string); exists {
				req.Header.Set("X-Request-Id", requestID)
			}
		}

		// 8. 设置代理的 ErrorHandler
		// - 处理代理错误，例如下游服务不可用时返回 502 错误
		proxy.ErrorHandler = func(rw http.ResponseWriter, req *http.Request, err error) {
			logger.Error("Proxy error",
				zap.Error(err),
				zap.String("path", req.URL.Path))
			rw.WriteHeader(http.StatusBadGateway)
			fmt.Fprintf(rw, `{"error": "Bad Gateway", "detail": "下游服务不可用"}`)
		}

		// 9. 为服务路径前缀创建路由组
		// - 使用 Gin 的 Group 方法为服务路径前缀注册代理，支持任意 HTTP 方法
		serviceGroup := r.Group(svc.Prefix)

		// 10. 注册公开路径（无需认证和权限）
		for _, publicPath := range svc.PublicPaths {
			fullPath := svc.Prefix + publicPath
			r.Any(fullPath, gin.WrapH(proxy)) // 直接代理，无中间件
		}

		// 11. 为其他路径应用认证和权限中间件
		// - 其他需要认证和权限中间件的资源统一注册
		serviceGroup.Use(mymiddleware.AuthMiddleware(jwtUtil))   // JWT 认证中间件
		serviceGroup.Use(mymiddleware.PermissionMiddleware(cfg)) // 权限校验中间件
		serviceGroup.Any("/*action", gin.WrapH(proxy))           // 代理所有其他请求
	}
}
