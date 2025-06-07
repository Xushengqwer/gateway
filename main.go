package main

import (
	"context"
	"encoding/json" // <-- 新增导入，用于格式化打印
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
	// CMD 中指定的路径是 /app/config/config.yaml，这里保持默认值以防万一
	flag.StringVar(&configFile, "config", "/app/config/config.yaml", "Path to configuration file")
	flag.Parse()

	// 1. 加载配置
	var cfg config.GatewayConfig
	if err := sharedCore.LoadConfig(configFile, &cfg); err != nil {
		// 增加一个明确的文件存在性检查，让错误更清晰
		if _, statErr := os.Stat(configFile); os.IsNotExist(statErr) {
			log.Fatalf("FATAL: 配置文件未在指定路径找到 (File Not Found at '%s'): %v", configFile, statErr)
		}
		log.Fatalf("FATAL: 加载配置文件失败 (%s): %v", configFile, err)
	}

	// --- DEBUGGING: 打印从文件加载后的完整配置 ---
	log.Println("--- DEBUG START: 打印从文件加载后的初始配置 ---")
	configBytes, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		log.Printf("DEBUG: 无法将配置序列化为 JSON 进行打印: %v", err)
	} else {
		log.Printf("DEBUG: Viper 加载的配置内容如下:\n%s\n", string(configBytes))
	}
	log.Printf("DEBUG: 加载后 cfg.Server.ListenAddr 的值是: '%s'", cfg.Server.ListenAddr)
	log.Printf("DEBUG: 加载后 cfg.Services 切片的长度是: %d", len(cfg.Services))
	if len(cfg.Services) > 0 {
		log.Printf("DEBUG: 第一个服务 (Services[0]) 的名称是: '%s'", cfg.Services[0].Name)
	} else {
		log.Println("DEBUG: 警告！cfg.Services 列表为空，这很可能是导致 404 或 panic 的原因！")
	}
	log.Println("--- DEBUG END ---")
	// --- 结束 DEBUGGING ---

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
