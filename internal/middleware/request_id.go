package middleware

import (
	"github.com/Xushengqwer/gateway/internal/core"
	"github.com/Xushengqwer/gateway/pkg/middleware"
	"github.com/gin-gonic/gin"
)

// RequestIDMiddleware 适配网关服务的请求 ID 中间件
// - 输入: logger *core.ZapLogger 网关的日志实例
// - 输出: gin.HandlerFunc 中间件函数
func RequestIDMiddleware(logger *core.ZapLogger) gin.HandlerFunc {
	// 网关需要生成请求 ID，设置 generateID 为 true
	return middleware.RequestIDMiddleware(logger.Logger(), true)
}
