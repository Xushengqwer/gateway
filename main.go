package main

import (
	"context"
	"flag"
	"fmt"
	"gateway/internal/core"
	"gateway/internal/router"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	// 1. 解析命令行参数
	flag.Parse()

	// 2. 初始化配置
	configPath, _ := core.InitConfig()

	// 3. 加载配置
	cfg, err := core.LoadConfig(configPath)
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}
	// 打印配置，验证加载是否成功
	fmt.Printf("Loaded config: %+v\n", cfg)

	// 4. 使用配置初始化 ZapLogger
	logger, err := core.NewZapLogger(cfg.ZapConfig)
	if err != nil {
		log.Fatalf("初始化 ZapLogger 失败: %v", err)
	}
	defer func() {
		if err := logger.Logger().Sync(); err != nil {
			logger.Error("ZapLogger Sync 失败: %v", zap.Error(err))
		}
	}()

	// 3. 初始化 Gin 引擎
	// - 创建 Gin 实例，设置为 Release 模式以提升性能
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()

	// 4. 初始化JWT工具
	// - 作用：解析
	jwtUtility := core.NewJWTUtility(cfg, logger)

	// 4. 设置代理路由
	// - 调用 SetupProxyRoutes 配置路由转发和中间件
	router.SetupProxyRoutes(r, cfg, logger, jwtUtility)

	// 5. 添加健康检查端点
	// - 提供 /health 端点，用于 Kubernetes 健康检查
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "healthy",
		})
	})

	// 6. 创建 HTTP 服务器
	// - 配置监听地址和处理程序
	srv := &http.Server{
		Addr:    cfg.ListenAddr, // 例如 ":8080"
		Handler: r,
	}

	// 7. 启动服务
	// - 在协程中启动 HTTP 服务器
	go func() {
		logger.Info("Starting gateway server", zap.String("addr", cfg.ListenAddr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	// 8. 优雅关闭
	// - 监听系统信号（SIGINT、SIGTERM），实现优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("Shutting down gateway server...")

	// 9. 设置关闭超时
	// - 等待 5 秒以完成正在处理的请求
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 10. 关闭服务器
	// - 调用 Shutdown 优雅关闭 HTTP 服务器
	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("Server shutdown failed", zap.Error(err))
	}
	logger.Info("Gateway server exited")
}
