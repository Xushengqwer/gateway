package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
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
	flag.StringVar(&configFile, "config", "/config/config.development.yaml", "Path to configuration file")
	flag.Parse()

	// 1. 加载配置
	var cfg config.GatewayConfig
	if err := sharedCore.LoadConfig(configFile, &cfg); err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 2. [新增] 打印最终生效的配置以供调试
	// 使用 json 包将配置结构体格式化为可读的字符串
	configBytes, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		log.Fatalf("无法序列化配置以进行打印: %v", err)
	}
	log.Printf("✅ 配置加载成功！最终生效的配置如下:\n%s\n", string(configBytes))

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
