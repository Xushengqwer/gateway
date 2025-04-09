package middleware

import (
	"fmt"
	"github.com/Xushengqwer/gateway/internal/core"
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ErrorHandlingMiddleware 定义 Gin 的全局错误处理中间件，用于捕获和处理 Panic
// - 输入: logger ZapLogger 实例，用于记录错误日志
// - 输出: gin.HandlerFunc 中间件函数
func ErrorHandlingMiddleware(logger *core.ZapLogger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. 设置 Panic 捕获逻辑
		// - 使用 defer 确保在 Panic 发生时执行恢复操作
		// - 捕获任何未预期的错误并记录日志
		defer func() {
			if err := recover(); err != nil {
				// 2. 记录 Panic 错误日志
				// - 使用 ZapLogger 记录详细的错误信息，包括堆栈跟踪
				// - 包含请求路径、方法和客户端 IP 等上下文信息
				logger.Error("Panic recovered",
					zap.Any("error", err),                           // 错误内容
					zap.String("errorType", fmt.Sprintf("%T", err)), // 错误类型
					zap.String("stack", string(debug.Stack())),      // 堆栈跟踪
					zap.String("path", c.Request.URL.Path),          // 请求路径
					zap.String("method", c.Request.Method),          // 请求方法
					zap.String("clientIP", c.ClientIP()),            // 客户端 IP
				)

				// 3. 返回错误响应
				// - 向客户端返回统一的服务器错误信息
				// - 使用 HTTP 状态码 500 表示内部错误
				c.JSON(http.StatusInternalServerError, gin.H{
					"message": "服务器故障，请稍后再试.",
				})
				c.Abort() // 终止请求处理
			}
		}()

		// 4. 继续处理后续请求
		// - 调用 c.Next() 执行链中的下一个中间件或处理函数
		c.Next()
	}
}
