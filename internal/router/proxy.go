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

// setupProxyRoutesInternal 处理实际的反向代理逻辑
func setupProxyRoutesInternal(r *gin.Engine, cfg *config.GatewayConfig, logger *sharedCore.ZapLogger, jwtUtil gatewayCore.JWTUtilityInterface, otelTransport http.RoundTripper) {
	// 创建一个 map 用于快速查找服务的公开路径
	publicPathsMap := make(map[string]map[string]bool) // map[servicePrefix][subPath] = true
	for _, svc := range cfg.Services {
		if _, exists := publicPathsMap[svc.Prefix]; !exists {
			publicPathsMap[svc.Prefix] = make(map[string]bool)
		}
		for _, p := range svc.PublicPaths {
			// 确保 public path 以 / 开头
			if !strings.HasPrefix(p, "/") {
				p = "/" + p
			}
			publicPathsMap[svc.Prefix][p] = true
			logger.Info("Marking public path", zap.String("prefix", svc.Prefix), zap.String("subPath", p))
		}
	}

	// 遍历配置中的服务
	for _, svc := range cfg.Services {
		// --- 构建 targetURL (保持不变) ---
		var targetHost string
		var port int
		scheme := svc.Scheme
		if scheme == "" {
			scheme = "http"
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
				logger.Fatal("Invalid service config for non-K8s mode", zap.String("service", svc.Name))
			}
			targetHost = svc.Host
			port = svc.Port
		}
		targetURL, err := url.Parse(fmt.Sprintf("%s://%s:%d", scheme, targetHost, port))
		if err != nil {
			logger.Fatal("Invalid service URL", zap.String("service", svc.Name), zap.Error(err))
		}

		// --- 创建反向代理实例 (保持不变) ---
		proxy := httputil.NewSingleHostReverseProxy(targetURL)
		proxy.Transport = otelTransport
		proxy.Director = func(req *http.Request) {
			req.URL.Scheme = targetURL.Scheme
			req.URL.Host = targetURL.Host
			originalPath := req.URL.Path
			newPath := strings.TrimPrefix(originalPath, svc.Prefix)
			if newPath == "" && originalPath == svc.Prefix {
				newPath = "/"
			}
			if !strings.HasPrefix(newPath, "/") && newPath != "" {
				newPath = "/" + newPath
			}
			req.URL.Path = newPath
			req.Header.Set("X-Forwarded-Host", req.Header.Get("Host"))
			req.Host = targetURL.Host
		}
		proxy.ErrorHandler = func(rw http.ResponseWriter, req *http.Request, err error) {
			// ... (ErrorHandler 逻辑保持不变, 考虑使用 response 包) ...
			logger.Error("Proxy error", zap.Error(err), zap.String("target", targetURL.String()), zap.String("path", req.URL.Path))
			rw.Header().Set("Content-Type", "application/json")
			rw.WriteHeader(http.StatusBadGateway)
			fmt.Fprintf(rw, `{"code": 50201, "message": "Bad Gateway", "detail": "下游服务不可用"}`)
		}

		// --- 注册统一的处理器 ---
		// 为整个服务前缀注册一个 ANY 方法的处理器
		// 使用 /*action 匹配该前缀下的所有子路径
		proxyPath := svc.Prefix + "/*action"
		logger.Info("Registering proxy handler", zap.String("path", proxyPath), zap.String("target", targetURL.String()))

		// 将服务配置和依赖注入到处理函数中
		handler := createProxyHandler(svc, cfg, logger, jwtUtil, proxy, publicPathsMap[svc.Prefix])
		r.Any(proxyPath, handler)

		// 如果 Prefix 本身也需要代理 (例如 /api/user 代理到下游 /)
		if svc.Prefix != "" {
			r.Any(svc.Prefix, handler) // 也注册根前缀
		}
	}
}

// createProxyHandler 创建一个 Gin 处理函数，该函数内部处理公开/私有逻辑
func createProxyHandler(svc config.ServiceConfig, cfg *config.GatewayConfig, logger *sharedCore.ZapLogger, jwtUtil gatewayCore.JWTUtilityInterface, proxy *httputil.ReverseProxy, servicePublicPaths map[string]bool) gin.HandlerFunc {
	// 提前创建中间件实例 (如果它们有状态的话，否则可以直接调用函数)
	// 注意：这里假设 AuthMiddleware 和 PermissionMiddleware 返回 gin.HandlerFunc
	// 更好的做法是将它们的 *核心逻辑* 提取为可单独调用的函数
	authHandler := mymiddleware.AuthMiddleware(jwtUtil)
	permHandler := mymiddleware.PermissionMiddleware(cfg)

	return func(c *gin.Context) {
		// 获取相对于服务前缀的子路径
		subPath := strings.TrimPrefix(c.FullPath(), svc.Prefix) // 使用 FullPath 获取注册的路径模板
		if !strings.HasPrefix(subPath, "/") && subPath != "" {
			subPath = "/" + subPath
		}
		// 对于完全匹配 Prefix 的情况，subPath 可能是空的，也视为根 "/"
		if subPath == "" {
			subPath = "/"
		}

		// 检查是否是公开路径
		isPublic := servicePublicPaths[subPath]
		// 也可以添加对 c.Param("action") 的检查，如果需要更精确匹配

		traceIDVal, _ := c.Get("traceID") // 假设 traceID 在 context 中
		logger.Debug("Checking path authorization",
			zap.String("prefix", svc.Prefix),
			zap.String("subPath", subPath),
			zap.Bool("isPublic", isPublic),
			zap.Any("traceID", traceIDVal), // 添加 traceID 方便调试
		)

		if isPublic {
			// 如果是公开路径，直接调用代理
			logger.Debug("Public path, proxying directly", zap.String("path", c.Request.URL.Path))
			proxy.ServeHTTP(c.Writer, c.Request)
		} else {
			// 如果是私有路径，按顺序执行中间件，然后代理
			logger.Debug("Private path, applying middleware", zap.String("path", c.Request.URL.Path))
			// 1. 执行认证中间件
			authHandler(c)
			if c.IsAborted() { // 检查认证中间件是否中止了请求
				logger.Warn("Request aborted by AuthMiddleware", zap.String("path", c.Request.URL.Path))
				return
			}

			// 2. 执行权限中间件
			permHandler(c)
			if c.IsAborted() { // 检查权限中间件是否中止了请求
				logger.Warn("Request aborted by PermissionMiddleware", zap.String("path", c.Request.URL.Path))
				return
			}

			// 3. 如果所有中间件通过，执行代理
			logger.Debug("Middleware passed, proxying request", zap.String("path", c.Request.URL.Path))
			proxy.ServeHTTP(c.Writer, c.Request)
		}
	}
}
