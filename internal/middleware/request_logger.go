package middleware

import (
	"gateway/internal/core" // 网关的 ZapLogger
	"gateway/pkg/middleware"
	"github.com/gin-gonic/gin"
)

// RequestLoggerMiddleware 适配网关服务的请求日志中间件
// - 输入: logger *core.ZapLogger 网关的日志实例
// - 输出: gin.HandlerFunc 中间件函数
func RequestLoggerMiddleware(logger *core.ZapLogger) gin.HandlerFunc {
	// 网关服务，设置 isGateway 为 true
	return middleware.RequestLoggerMiddleware(logger.Logger(), true)
}
