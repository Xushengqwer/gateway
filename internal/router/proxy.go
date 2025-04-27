package router

import (
	"fmt"
	"github.com/Xushengqwer/gateway/internal/constant"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/Xushengqwer/gateway/internal/config"
	gatewayCore "github.com/Xushengqwer/gateway/internal/core"        // 网关内部 core (JWT)
	mymiddleware "github.com/Xushengqwer/gateway/internal/middleware" // 网关内部中间件 (Auth, Perm)
	sharedCore "github.com/Xushengqwer/go-common/core"                // 共享库 core (Logger)
	sharedMiddleware "github.com/Xushengqwer/go-common/middleware"    // 共享库中间件

	"github.com/gin-gonic/gin"                                                     // Gin 框架
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin" // OTel Gin
	"go.uber.org/zap"
)

// SetupRouter 设置网关的所有路由和全局中间件
func SetupRouter(r *gin.Engine, cfg *config.GatewayConfig, logger *sharedCore.ZapLogger, jwtUtil gatewayCore.JWTUtilityInterface, otelTransport http.RoundTripper) {

	// --- 1. 应用全局中间件 (按顺序) ---

	// OTel Gin 中间件 (如果启用追踪)
	if cfg.TracerConfig.Enabled {
		r.Use(otelgin.Middleware(constant.ServiceName))
	}

	// Panic 恢复
	r.Use(sharedMiddleware.ErrorHandlingMiddleware(logger))

	// Trace Info 提取 (可选)
	r.Use(sharedMiddleware.RequestIDMiddleware()) // 使用共享改造后的

	// 请求日志 (包含 Trace/Span ID)
	r.Use(sharedMiddleware.RequestLoggerMiddleware(logger.Logger())) // 使用共享改造后的

	// 请求超时 (使用配置中的超时)
	r.Use(sharedMiddleware.RequestTimeoutMiddleware(logger, cfg.Server.RequestTimeout))

	// 全局限流中间件
	r.Use(mymiddleware.RateLimitMiddleware(logger, cfg.RateLimitConfig))

	// 7. CORS 跨域处理中间件
	r.Use(mymiddleware.CorsMiddleware(cfg.Cors))

	// --- 2. 设置健康检查路由 ---
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})

	// --- 3. 设置代理路由和特定中间件 ---
	setupProxyRoutesInternal(r, cfg, logger, jwtUtil, otelTransport) // 调用内部函数处理代理

}

// setupProxyRoutesInternal 处理实际的反向代理逻辑和路由组特定中间件
// (这是你原来 SetupProxyRoutes 函数的核心逻辑，稍作修改)
func setupProxyRoutesInternal(r *gin.Engine, cfg *config.GatewayConfig, logger *sharedCore.ZapLogger, jwtUtil gatewayCore.JWTUtilityInterface, otelTransport http.RoundTripper) {
	// 遍历配置中的服务
	for _, svc := range cfg.Services {
		// ... (构建 targetHost, port, scheme 的逻辑保持不变) ...
		var targetHost string
		var port int
		scheme := svc.Scheme
		if scheme == "" {
			scheme = "http" // 默认协议为 http
		}
		if svc.ServiceName != "" { /* K8s logic */
			namespace := svc.Namespace
			if namespace == "" {
				namespace = "default"
			}
			targetHost = fmt.Sprintf("%s.%s.svc.cluster.local", svc.ServiceName, namespace)
			port = svc.Port
			if port == 0 {
				port = 80
			}
		} else { /* Non-K8s logic */
			if svc.Host == "" || svc.Port == 0 { /* Error handling */
				logger.Fatal("Invalid service config for non-K8s mode", zap.String("service", svc.Name)) // ... more fields ...
			}
			targetHost = svc.Host
			port = svc.Port
		}

		targetURL, err := url.Parse(fmt.Sprintf("%s://%s:%d", scheme, targetHost, port))
		if err != nil {
			logger.Fatal("Invalid service URL", zap.String("service", svc.Name), zap.Error(err))
		}

		// 创建反向代理实例
		proxy := httputil.NewSingleHostReverseProxy(targetURL)

		// --- 关键: 设置 OTel Transport ---
		// 这样通过代理发出的请求就会被追踪并携带上下文
		proxy.Transport = otelTransport

		// 修改 Director (移除手动设置 X-Request-Id)
		proxy.Director = func(req *http.Request) {
			req.URL.Scheme = targetURL.Scheme
			req.URL.Host = targetURL.Host
			originalPath := req.URL.Path // 保存原始路径供可能的日志记录或权限判断
			newPath := strings.TrimPrefix(originalPath, svc.Prefix)
			// 如果 TrimPrefix 后为空，可能需要设置为 "/"
			if newPath == "" && originalPath == svc.Prefix {
				newPath = "/"
			} else if !strings.HasPrefix(newPath, "/") && newPath != "" {
				newPath = "/" + newPath // 确保以 / 开头
			}
			req.URL.Path = newPath

			// 保留原始 Host 或设置 X-Forwarded-Host (可选，根据下游需要)
			req.Header.Set("X-Forwarded-Host", req.Header.Get("Host"))
			req.Host = targetURL.Host // 设置请求的 Host 头为目标服务的 Host
		}

		// ErrorHandler (保持不变)
		proxy.ErrorHandler = func(rw http.ResponseWriter, req *http.Request, err error) {
			logger.Error("Proxy error", zap.Error(err), zap.String("path", req.URL.Path))
			rw.WriteHeader(http.StatusBadGateway)
			// 考虑使用共享的 RespondError
			// sharedResponse.RespondError(rw, http.StatusBadGateway, sharedResponse.ErrCodeServiceNotFound, "下游服务不可用")
			fmt.Fprintf(rw, `{"code": 50201, "message": "Bad Gateway", "detail": "下游服务不可用"}`) // 使用统一格式
		}

		// 创建路由组
		serviceGroup := r.Group(svc.Prefix)
		{ // 使用花括号明确作用域

			// --- 在路由组级别应用网关特定的中间件 ---
			// (这些中间件在全局中间件之后，代理处理之前运行)

			// 10. 应用认证和权限中间件到需要保护的路径
			// 注意：这里我们将认证和权限应用到整个 group
			// 如果有公开路径，它们的注册方式需要调整
			serviceGroup.Use(mymiddleware.AuthMiddleware(jwtUtil))   // JWT 认证
			serviceGroup.Use(mymiddleware.PermissionMiddleware(cfg)) // 权限校验

			// 11. 代理所有 /prefix/* 的请求
			// 使用 /*action 匹配组内的所有路径
			serviceGroup.Any("/*action", gin.WrapH(proxy))
		}

		// --- 处理公开路径 (PublicPaths) ---
		// 公开路径不需要 Auth 和 Permission，直接在 r 上注册
		for _, publicPath := range svc.PublicPaths {
			// 确保 publicPath 以 / 开头
			if !strings.HasPrefix(publicPath, "/") {
				publicPath = "/" + publicPath
			}
			// 完整路径应该是 Prefix + PublicPath
			// 例如 Prefix=/user, PublicPath=/register -> /user/register
			fullPublicPath := svc.Prefix + publicPath
			logger.Info("Registering public proxy route", zap.String("path", fullPublicPath))
			// 对公开路径应用代理，但不应用 serviceGroup 的中间件
			r.Any(fullPublicPath, gin.WrapH(proxy))
		}
	}
}
