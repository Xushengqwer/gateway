package middleware

import (
	"github.com/Xushengqwer/gateway/internal/core"
	"github.com/Xushengqwer/gateway/pkg/middleware"
	"github.com/gin-gonic/gin"
)

// ErrorHandlingMiddleware 适配网关服务的错误处理中间件
func ErrorHandlingMiddleware(logger *core.ZapLogger) gin.HandlerFunc {
	return middleware.ErrorHandlingMiddleware(logger)
}
