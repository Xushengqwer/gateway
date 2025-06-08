package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/Xushengqwer/gateway/internal/config"
	"github.com/Xushengqwer/gateway/internal/constant"
	gatewayCore "github.com/Xushengqwer/gateway/internal/core"
	"github.com/Xushengqwer/gateway/internal/router"
	sharedCore "github.com/Xushengqwer/go-common/core"
	sharedTracing "github.com/Xushengqwer/go-common/core/tracing"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.uber.org/zap"
)

func main() {
	var configFile string
	flag.StringVar(&configFile, "config", "/app/config/config.yaml", "Path to configuration file")
	flag.Parse()

	// 1. 加载配置
	var cfg config.GatewayConfig
	if err := sharedCore.LoadConfig(configFile, &cfg); err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// --- 手动从环境变量覆盖关键配置 (最终生产版) ---
	log.Println("检查环境变量以覆盖文件配置...")

	if key := os.Getenv("JWTCONFIG_SECRET_KEY"); key != "" {
		cfg.JWTConfig.SecretKey = key
		log.Println("通过环境变量覆盖了 JWTConfig.SecretKey")
	}
	if key := os.Getenv("JWTCONFIG_REFRESH_SECRET"); key != "" {
		cfg.JWTConfig.RefreshSecret = key
		log.Println("通过环境变量覆盖了 JWTConfig.RefreshSecret")
	}
	if origins := os.Getenv("PROD_CORS_ALLOW_ORIGINS"); origins != "" {
		cfg.Cors.AllowOrigins = strings.Split(origins, ",")
		log.Printf("通过环境变量覆盖了 CORS AllowOrigins: %v\n", cfg.Cors.AllowOrigins)
	}

	// 动态覆盖下游服务地址
	for i := range cfg.Services {
		serviceName := cfg.Services[i].Name
		var newHost string
		var newPort int

		switch serviceName {
		case "user-hub-service":
			newHost = "user-hub-app"
			newPort = 8081
		case "post-service":
			newHost = "post-app"
			newPort = 8082
		case "post-search-service":
			newHost = "post-search-app"
			newPort = 8083
		}

		if newHost != "" {
			cfg.Services[i].Host = newHost
			cfg.Services[i].Port = newPort
			log.Printf("生产环境: 服务 %s 将被代理到 -> %s:%d\n", serviceName, newHost, newPort)
		}
	}
	// --- 结束环境变量覆盖 ---

	// 2. 初始化 Logger
	logger, err := sharedCore.NewZapLogger(cfg.ZapConfig)
	if err != nil {
		log.Fatalf("初始化 ZapLogger 失败: %v", err)
	}
	defer func() {
		if err := logger.Logger().Sync(); err != nil {
			logger.Error("ZapLogger Sync 失败", zap.Error(err))
		}
	}()
	logger.Info("Logger 初始化成功")

	// ... (main 函数的其余部分保持不变) ...
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
		otelTransport = otelhttp.NewTransport(http.DefaultTransport)
	} else {
		logger.Info("分布式追踪已禁用")
	}

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()

	jwtUtility := gatewayCore.NewJWTUtility(&cfg, logger)
	router.SetupRouter(r, &cfg, logger, jwtUtility, otelTransport)

	srv := &http.Server{
		Addr:    cfg.Server.ListenAddr,
		Handler: r,
	}

	go func() {
		if srv.Addr == "" {
			logger.Fatal("HTTP 服务器启动失败：监听地址 (Addr) 为空！请检查配置加载。")
		}
		logger.Info("Starting gateway server", zap.String("addr", cfg.Server.ListenAddr))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("Shutting down gateway server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("Server shutdown failed", zap.Error(err))
	}
	logger.Info("Gateway server exited")
}
