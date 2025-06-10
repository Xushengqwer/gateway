package router

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/Xushengqwer/gateway/internal/config"
	"github.com/Xushengqwer/gateway/internal/constant"
	gatewayCore "github.com/Xushengqwer/gateway/internal/core"
	mymiddleware "github.com/Xushengqwer/gateway/internal/middleware"
	sharedCore "github.com/Xushengqwer/go-common/core"
	sharedMiddleware "github.com/Xushengqwer/go-common/middleware"
	"github.com/Xushengqwer/go-common/response" // <-- 确保已导入

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.uber.org/zap"
)

// SetupRouter 设置网关的所有路由和全局中间件。
// (函数保持不变)
func SetupRouter(r *gin.Engine, cfg *config.GatewayConfig, logger *sharedCore.ZapLogger, jwtUtil gatewayCore.JWTUtilityInterface, otelTransport http.RoundTripper) {
	logger.Info("开始设置网关路由及全局中间件...")

	// --- 1. 应用全局中间件 (按执行顺序排列) ---
	if cfg.TracerConfig.Enabled {
		r.Use(otelgin.Middleware(constant.ServiceName))
		logger.Info("OpenTelemetry 中间件已启用。")
	}
	r.Use(sharedMiddleware.ErrorHandlingMiddleware(logger))
	if baseLogger := logger.Logger(); baseLogger != nil {
		r.Use(sharedMiddleware.RequestLoggerMiddleware(baseLogger))
	} else {
		logger.Warn("无法获取底层的 *zap.Logger，跳过 RequestLoggerMiddleware 注册")
	}
	r.Use(sharedMiddleware.RequestTimeoutMiddleware(logger, cfg.Server.RequestTimeout))
	if cfg.RateLimitConfig != nil {
		r.Use(mymiddleware.RateLimitMiddleware(logger, cfg.RateLimitConfig))
		logger.Info("全局限流中间件已启用。")
	} else {
		logger.Info("全局限流配置未提供，跳过限流中间件。")
	}
	r.Use(mymiddleware.CorsMiddleware(cfg.Cors))
	logger.Info("CORS 中间件已启用。")

	// --- 2. 设置健康检查路由 ---
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})
	logger.Info("健康检查路由 /health 已注册。")

	// --- 3. 设置代理路由和特定中间件 ---
	setupProxyRoutesInternal(r, cfg, logger, jwtUtil, otelTransport)
	logger.Info("所有代理路由已设置完成。")
}

// matchPublicPath 检查请求子路径是否匹配给定的公共路径模式 (支持参数)
// (函数保持不变)
func matchPublicPath(publicPathPattern string, requestSubPath string) bool {
	patternSegments := strings.Split(strings.Trim(publicPathPattern, "/"), "/")
	requestSegments := strings.Split(strings.Trim(requestSubPath, "/"), "/")

	isPatternRoot := publicPathPattern == "/" || (len(patternSegments) == 1 && patternSegments[0] == "")
	isRequestRoot := requestSubPath == "/" || (len(requestSegments) == 1 && requestSegments[0] == "")

	if isPatternRoot && isRequestRoot {
		return true
	}

	if len(patternSegments) != len(requestSegments) {
		return false
	}

	for i, pSegment := range patternSegments {
		rSegment := requestSegments[i]
		if strings.HasPrefix(pSegment, ":") || strings.HasPrefix(pSegment, "{") {
			if rSegment == "" {
				return false
			}
			continue
		}
		if pSegment != rSegment {
			return false
		}
	}
	return true
}

// setupProxyRoutesInternal 根据配置文件中的服务列表，设置反向代理路由。
// (函数保持不变, 除了 handler 创建部分)
func setupProxyRoutesInternal(r *gin.Engine, cfg *config.GatewayConfig, logger *sharedCore.ZapLogger, jwtUtil gatewayCore.JWTUtilityInterface, otelTransport http.RoundTripper) {

	for _, svc := range cfg.Services {
		serviceConfig := svc // 避免闭包问题

		var targetHost string
		var port int
		scheme := serviceConfig.Scheme
		if scheme == "" {
			scheme = "http"
		}

		if serviceConfig.ServiceName != "" {
			namespace := serviceConfig.Namespace
			if namespace == "" {
				namespace = "default"
			}
			targetHost = fmt.Sprintf("%s.%s.svc.cluster.local", serviceConfig.ServiceName, namespace)
			port = serviceConfig.Port
			if port == 0 {
				port = 80
			}
		} else {
			if serviceConfig.Host == "" || serviceConfig.Port == 0 {
				logger.Fatal("非K8s模式下服务配置无效：Host 或 Port 未指定",
					zap.String("serviceName", serviceConfig.Name))
			}
			targetHost = serviceConfig.Host
			port = serviceConfig.Port
		}

		targetURL, err := url.Parse(fmt.Sprintf("%s://%s:%d", scheme, targetHost, port))
		if err != nil {
			logger.Fatal("解析目标服务URL失败",
				zap.String("serviceName", serviceConfig.Name),
				zap.String("targetHost", targetHost),
				zap.Int("port", port),
				zap.Error(err))
		}
		logger.Info("构建目标服务URL成功",
			zap.String("serviceName", serviceConfig.Name),
			zap.String("targetURL", targetURL.String()))

		proxy := httputil.NewSingleHostReverseProxy(targetURL)
		if otelTransport != nil {
			proxy.Transport = otelTransport
		}

		proxy.Director = func(req *http.Request) {
			req.URL.Scheme = targetURL.Scheme
			req.URL.Host = targetURL.Host
			req.Header.Set("X-Forwarded-Host", req.Header.Get("Host"))
			req.Host = targetURL.Host
			logger.Debug("正在代理请求，包含以下头部信息",
				zap.String("serviceName", serviceConfig.Name),
				zap.Any("headers", req.Header),
				zap.String("path", req.URL.Path))
		}

		proxy.ErrorHandler = func(rw http.ResponseWriter, req *http.Request, err error) {
			logger.Error("反向代理错误",
				zap.Error(err),
				zap.String("targetService", serviceConfig.Name),
				zap.String("targetURL", targetURL.String()),
				zap.String("requestPath", req.URL.Path),
			)
			rw.Header().Set("Content-Type", "application/json")
			rw.WriteHeader(http.StatusBadGateway)
			fmt.Fprintf(rw, `{"code": 50201, "message": "Bad Gateway", "detail": "下游服务不可用或响应错误"}`)
		}

		proxyPath := serviceConfig.Prefix
		if !strings.HasSuffix(proxyPath, "/") {
			proxyPath += "/"
		}
		proxyPath += "*action"

		logger.Info("为服务注册代理处理器",
			zap.String("serviceName", serviceConfig.Name),
			zap.String("ginRoutePath", proxyPath),
			zap.String("targetURL", targetURL.String()))

		// --- 修改: 不再传递 PublicPaths, 而是整个 svcCfg ---
		handler := createProxyHandler(serviceConfig, cfg, logger, jwtUtil, proxy)

		r.Any(proxyPath, handler)
		if serviceConfig.Prefix != "" {
			r.Any(serviceConfig.Prefix, handler)
			logger.Info("同时为服务根前缀注册处理器",
				zap.String("serviceName", serviceConfig.Name),
				zap.String("exactPrefixPath", serviceConfig.Prefix))
		}
	}
}

// createProxyHandler 创建一个 Gin 处理函数 (重构版 - 核心修改)
func createProxyHandler(
	svcCfg config.ServiceConfig, // <-- 使用整个服务配置
	gatewayCfg *config.GatewayConfig,
	logger *sharedCore.ZapLogger,
	jwtUtil gatewayCore.JWTUtilityInterface,
	proxy *httputil.ReverseProxy,
) gin.HandlerFunc {
	authHandler := mymiddleware.AuthMiddleware(jwtUtil)
	permHandler := mymiddleware.PermissionMiddleware(gatewayCfg)

	return func(c *gin.Context) {
		requestPath := c.Request.URL.Path
		method := c.Request.Method
		subPathForLookup := strings.TrimPrefix(requestPath, svcCfg.Prefix)
		if !strings.HasPrefix(subPathForLookup, "/") && subPathForLookup != "" {
			subPathForLookup = "/" + subPathForLookup
		} else if subPathForLookup == "" {
			subPathForLookup = "/"
		}

		traceIDVal, _ := c.Get("traceID")
		logger.Debug("检查路径授权状态 (新)",
			zap.String("serviceName", svcCfg.Name),
			zap.String("requestPath", requestPath),
			zap.String("servicePrefix", svcCfg.Prefix),
			zap.String("subPathForLookup", subPathForLookup),
			zap.Any("traceID", traceIDVal),
		)

		// --- 1. 优先检查公开路由 ---
		isPublic := false
		for _, publicPattern := range svcCfg.PublicPaths {
			if matchPublicPath(publicPattern, subPathForLookup) {
				isPublic = true
				break
			}
		}

		if isPublic {
			// 是公开路由 -> 直接代理
			logger.Debug("公开路径，直接代理",
				zap.String("serviceName", svcCfg.Name),
				zap.String("path", requestPath))
			proxy.ServeHTTP(c.Writer, c.Request)
			return // 结束处理
		}

		// --- 2. 如果不是公开路由，再检查是否匹配私有路由 ---
		_, foundPrivate := mymiddleware.FindBestMatchingRoute(svcCfg.Routes, subPathForLookup, method)

		if foundPrivate {
			// 是私有路由 -> 走认证流程
			logger.Debug("私有路径，应用认证和权限中间件",
				zap.String("serviceName", svcCfg.Name),
				zap.String("path", requestPath))

			authHandler(c)
			if c.IsAborted() {
				logger.Warn("请求被认证中间件中止",
					zap.String("serviceName", svcCfg.Name),
					zap.String("path", requestPath),
					zap.Int("statusCode", c.Writer.Status()))
				return
			}
			logger.Debug("认证中间件通过",
				zap.String("serviceName", svcCfg.Name),
				zap.String("path", requestPath))

			permHandler(c)
			if c.IsAborted() {
				logger.Warn("请求被权限中间件中止",
					zap.String("serviceName", svcCfg.Name),
					zap.String("path", requestPath),
					zap.Int("statusCode", c.Writer.Status()))
				return
			}
			logger.Debug("权限中间件通过",
				zap.String("serviceName", svcCfg.Name),
				zap.String("path", requestPath))

			logger.Debug("所有中间件通过，执行代理",
				zap.String("serviceName", svcCfg.Name),
				zap.String("path", requestPath))
			proxy.ServeHTTP(c.Writer, c.Request)

		} else {
			// --- 3. 既不匹配公开也不匹配私有 -> 拒绝访问 ---
			logger.Warn("路径未匹配任何私有或公开路由，拒绝访问",
				zap.String("serviceName", svcCfg.Name),
				zap.String("path", requestPath),
				zap.String("subPath", subPathForLookup),
			)
			response.RespondError(c, http.StatusNotFound, response.ErrCodeClientResourceNotFound, "路径未定义或无权访问")
			c.Abort()
		}
	}
}
