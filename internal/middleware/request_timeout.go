package middleware

import (
	"gateway/internal/core"
	"gateway/pkg/constant"
	"gateway/pkg/middleware"
	"github.com/gin-gonic/gin"
)

// router.Use(middleware.RequestTimeoutMiddleware(logger.Logger(), 10*time.Second))

// RequestTimeoutMiddleware 网关服务请求超时中间件
func RequestTimeoutMiddleware(logger *core.ZapLogger) gin.HandlerFunc {
	return middleware.RequestTimeoutMiddleware(logger, constant.RequestTimeout)
}
