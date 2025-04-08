package middleware

import (
	"context"
	"gateway/internal/core"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// RequestTimeoutMiddleware 定义请求超时中间件，为每个请求设置超时时间并在超时后中断处理
// - 输入: logger ZapLogger 实例用于日志记录, timeout 请求超时时间
// - 输出: gin.HandlerFunc 中间件函数
func RequestTimeoutMiddleware(logger *core.ZapLogger, timeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. 创建带超时的上下文
		// - 使用 context.WithTimeout 创建一个带有指定超时的上下文
		// - defer cancel 确保在函数退出时释放资源
		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()

		// 2. 替换请求上下文
		// - 将 gin.Context 的原始请求上下文替换为带超时的上下文
		// - 确保后续控制器或服务调用 c.Request.Context() 时具有超时限制
		c.Request = c.Request.WithContext(ctx)

		// 3. 创建完成信号通道
		// - 使用 channel 监听请求是否正常完成
		finished := make(chan struct{})

		// 4. 在协程中处理请求
		// - 调用 c.Next() 执行后续中间件或处理器
		// - 处理完成后通过 finished 通道发送信号
		go func() {
			c.Next()
			finished <- struct{}{}
		}()

		// 5. 等待请求完成或超时
		// - 使用 select 监听两个事件：上下文超时或请求完成
		select {
		case <-ctx.Done():
			// 6. 处理超时情况
			// - 如果上下文超时，记录警告日志
			err := ctx.Err() // 获取超时错误（通常为 context deadline exceeded）
			logger.Warn("请求超时",
				zap.Error(err), // 记录超时错误详情
			)

			// - 返回超时响应（HTTP 504）
			// - 使用 AbortWithStatusJSON 中断请求并返回自定义错误信息
			c.AbortWithStatusJSON(http.StatusGatewayTimeout, gin.H{
				"code":    http.StatusGatewayTimeout, // 状态码 504
				"message": "Request Timeout",         // 英文消息
				"detail":  "请求超时，请稍后重试",              // 中文详情
			})
			return

		case <-finished:
			// 7. 请求正常完成
			// - 如果收到 finished 信号，表示请求在超时前完成，无需额外处理
		}
	}
}
