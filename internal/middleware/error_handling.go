package middleware

import (
	"gateway/internal/core"
	"gateway/pkg/middleware"
	"github.com/gin-gonic/gin"
)

// ErrorHandlingMiddleware 适配网关服务的错误处理中间件
func ErrorHandlingMiddleware(logger *core.ZapLogger) gin.HandlerFunc {
	return middleware.ErrorHandlingMiddleware(logger)
}
