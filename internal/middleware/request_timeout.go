package middleware

import (
	"github.com/Xushengqwer/gateway/internal/core"
	"github.com/Xushengqwer/gateway/pkg/constant"
	"github.com/Xushengqwer/gateway/pkg/middleware"
	"github.com/gin-gonic/gin"
)

// router.Use(middleware.RequestTimeoutMiddleware(logger.Logger(), 10*time.Second))

// RequestTimeoutMiddleware 网关服务请求超时中间件
func RequestTimeoutMiddleware(logger *core.ZapLogger) gin.HandlerFunc {
	return middleware.RequestTimeoutMiddleware(logger, constant.RequestTimeout)
}
