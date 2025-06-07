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
	flag.StringVar(&configFile, "config", "config/development.yaml", "Path to configuration file")
	flag.Parse()

	// 1. 加载配置
	var cfg config.GatewayConfig
	if err := sharedCore.LoadConfig(configFile, &cfg); err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// --- DEBUGGING: 打印从文件加载后的完整配置 ---
	log.Println("--- DEBUG START: 打印从文件加载后的初始配置 ---")

	// 使用 JSON 格式化打印，结构更清晰
	configBytes, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		log.Printf("DEBUG: 无法将配置序列化为 JSON 进行打印: %v", err)
	} else {
		// 打印 Viper 解析后的完整配置结构体
		log.Printf("DEBUG: Viper 加载的配置内容如下:\n%s\n", string(configBytes))
	}

	// 专门检查几个关键字段的值
	log.Printf("DEBUG: 加载后 cfg.Server.ListenAddr 的值是: '%s'", cfg.Server.ListenAddr)
	log.Printf("DEBUG: 加载后 cfg.Services 切片的长度是: %d", len(cfg.Services))
	if len(cfg.Services) > 0 {
		log.Printf("DEBUG: 第一个服务 (Services[0]) 的名称是: '%s'", cfg.Services[0].Name)
	} else {
		log.Println("DEBUG: 警告！cfg.Services 列表为空，这会导致网关无法注册任何代理路由！")
	}
	log.Println("--- DEBUG END ---")
	// --- 结束 DEBUGGING ---

	// 注意：我们暂时不执行环境变量覆盖，以便观察最纯粹的文件加载结果
	// log.Println("检查环境变量以覆盖 Gateway 的文件配置...")
	// ... (所有 if os.Getenv(...) 的代码块暂时被跳过) ...

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
		// 这里我们做一个保护，如果 ListenAddr 仍然是空的，就 panic，以便在日志中看到明确的失败点
		if srv.Addr == "" {
			logger.Fatal("HTTP 服务器启动失败：监听地址 (Addr) 为空！")
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
