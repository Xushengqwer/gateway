// 在 gateway/main.go
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"github.com/Xushengqwer/gateway/internal/constant"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Xushengqwer/gateway/internal/config"
	gatewayCore "github.com/Xushengqwer/gateway/internal/core"    // 网关内部 core (JWT)
	"github.com/Xushengqwer/gateway/internal/router"              // 网关内部 router
	sharedCore "github.com/Xushengqwer/go-common/core"            // 共享库 core (Logger, Config)
	sharedTracing "github.com/Xushengqwer/go-common/core/tracing" // 共享库 tracing

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	// 导入 OTel HTTP Client Instrumentation
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

func main() {
	var configFile string
	flag.StringVar(&configFile, "config", "config/development.yaml", "Path to configuration file")
	flag.Parse()

	// --- 1. 加载配置 (修正) ---
	var cfg config.GatewayConfig
	if err := sharedCore.LoadConfig(configFile, &cfg); err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}
	fmt.Printf("Loaded config: %+v\n", &cfg) // 打印指针地址内容

	// --- 2. 初始化 Logger (修正) ---
	logger, err := sharedCore.NewZapLogger(cfg.ZapConfig)
	if err != nil {
		log.Fatalf("初始化 ZapLogger 失败: %v", err)
	}
	defer func() {
		if err := logger.Logger().Sync(); err != nil {
			logger.Error("ZapLogger Sync 失败", zap.Error(err))
		}
	}()

	// --- 3. 初始化 TracerProvider (如果启用) ---
	var otelTransport http.RoundTripper = http.DefaultTransport
	if cfg.TracerConfig.Enabled {
		shutdownTracer, err := sharedTracing.InitTracerProvider(constant.ServiceName, constant.ServiceVersion, cfg.TracerConfig)
		if err != nil {
			logger.Fatal("初始化 TracerProvider 失败", zap.Error(err))
		}
		defer func() {
			if err := shutdownTracer(context.Background()); err != nil {
				logger.Error("关闭 TracerProvider 失败", zap.Error(err))
			}
		}()
		logger.Info("分布式追踪已初始化")

		// --- 关键: 创建 OTel-instrumented Transport ---
		// 这个 transport 将用于反向代理发出的请求
		otelTransport = otelhttp.NewTransport(http.DefaultTransport)
	} else {
		logger.Info("分布式追踪已禁用")
	}

	// --- 4. 初始化 Gin 引擎 ---
	gin.SetMode(gin.ReleaseMode) // 或根据环境设置
	r := gin.New()               // 使用 New() 以便完全控制中间件

	// --- 5. 初始化 JWT 工具 ---
	jwtUtility := gatewayCore.NewJWTUtility(&cfg, logger)

	// --- 6. 设置路由和所有中间件 ---
	// 将依赖项传递给路由设置函数
	router.SetupRouter(r, &cfg, logger, jwtUtility, otelTransport) // <--- 调用新的设置函数

	// --- 7. 创建 HTTP 服务器 ---
	// 确保 cfg.Server.ListenAddr 在 GatewayConfig 中定义
	srv := &http.Server{
		Addr:    cfg.Server.ListenAddr,
		Handler: r,
		// 可以添加 ReadTimeout, WriteTimeout 等设置
	}

	// --- 8. 启动和优雅关闭 (保持不变) ---
	go func() {
		logger.Info("Starting gateway server", zap.String("addr", cfg.Server.ListenAddr))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("Shutting down gateway server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second) // 使用 cfg.Server.ShutdownTimeout
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("Server shutdown failed", zap.Error(err))
	}
	logger.Info("Gateway server exited")
}
