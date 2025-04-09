package middleware

import (
	"github.com/Xushengqwer/gateway/pkg/constant"
	"github.com/gin-gonic/gin" // Gin 框架
	"github.com/google/uuid"   // UUID 生成
	"go.uber.org/zap"          // Zap 日志
)

// RequestIDMiddleware 定义请求 ID 中间件，为请求生成或复用唯一 ID
// - 输入: logger *zap.Logger 日志实例，用于记录请求 ID
// - 输入: generateID bool，是否生成新的请求 ID（网关为 true，下游服务为 false）
// - 输出: gin.HandlerFunc 中间件函数
// - 意图: 为请求生成或复用全局请求 ID，存入上下文和请求头，透传给下游服务，并集成到日志
func RequestIDMiddleware(logger *zap.Logger, generateID bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. 检查请求头中的 X-Request-Id
		// - 获取客户端或上游服务可能已生成的请求 ID
		// - 如果存在，则复用该 ID
		requestID := c.Request.Header.Get("X-Request-Id")

		// 2. 生成新的请求 ID（如果需要）
		// - 如果 generateID 为 true（网关场景）且请求头中没有 X-Request-Id，则生成一个新的 UUID
		if generateID && requestID == "" {
			requestID = uuid.New().String()
		}

		// 3. 验证请求 ID
		// - 如果 requestID 仍为空（下游服务场景且无透传 ID），记录警告但继续处理
		if requestID == "" {
			logger.Warn("请求 ID 为空，上游未提供 ID")
			// 可选：生成临时 ID，仅用于当前服务
			// requestID = uuid.New().String()
		}

		// 4. 存储请求 ID 到上下文
		// - 将 requestID 存入 gin.Context，使用 constants.RequestIDKey 作为键
		// - 便于后续中间件或控制器使用
		c.Set(constant.RequestIDKey, requestID)

		// 5. 设置请求头
		// - 将 requestID 写入请求头的 X-Request-Id 字段
		// - 确保透传给下游服务
		if requestID != "" {
			c.Request.Header.Set("X-Request-Id", requestID)
		}

		// 6. 设置响应头
		// - 将 requestID 写入响应头的 X-Request-Id 字段
		// - 方便客户端或日志系统追踪请求
		if requestID != "" {
			c.Header("X-Request-Id", requestID)
		}

		// 7. 将请求 ID 添加到日志上下文
		// - 使用 With 创建一个新的 logger，包含 requestID 字段
		// - 确保后续日志记录包含请求 ID
		if requestID != "" {
			loggerWithID := logger.With(zap.String("requestID", requestID))
			c.Set("logger", loggerWithID)
		}

		// 8. 继续处理请求
		// - 调用 c.Next() 执行后续中间件或处理逻辑
		c.Next()
	}
}
