package middleware

import (
	"github.com/Xushengqwer/gateway/pkg/constant"
	"time" // 时间处理

	"github.com/gin-gonic/gin" // Gin 框架
	"go.uber.org/zap"          // Zap 日志
)

// RequestLoggerMiddleware 定义请求日志中间件，用于记录每个请求的关键信息
// - 输入: logger *zap.Logger 日志实例，用于记录日志
// - 输入: isGateway bool，是否为网关服务（网关需要分解时长）
// - 输出: gin.HandlerFunc 中间件函数
// - 意图: 记录请求的详细信息，支持网关和下游服务，网关可分解时长
func RequestLoggerMiddleware(logger *zap.Logger, isGateway bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. 记录请求开始时间
		// - 使用 time.Now() 获取当前时间，作为请求处理的起点
		startTime := time.Now()

		// 2. 处理后续请求
		// - 调用 c.Next() 执行后续中间件或控制器逻辑
		// - 网关服务记录转发前的时间
		var gatewayProcessingTime time.Duration
		if isGateway {
			gatewayStart := time.Now()
			c.Next()
			gatewayProcessingTime = time.Now().Sub(gatewayStart)
		} else {
			c.Next()
		}

		// 3. 计算总处理时长
		// - 获取请求结束时间并计算与开始时间的差值
		endTime := time.Now()
		totalLatency := endTime.Sub(startTime)

		// 4. 从上下文中获取请求信息
		// - 提取请求方法、路径、状态码、客户端 IP 和用户代理
		method := c.Request.Method
		path := c.Request.URL.Path
		statusCode := c.Writer.Status()
		clientIP := c.ClientIP()
		userAgent := c.Request.UserAgent()
		rid, exists := c.Get(constant.RequestIDKey)
		requestID, _ := rid.(string)

		// 5. 检查请求 ID 是否存在
		// - 如果 requestID 为空，记录警告
		if !exists || requestID == "" {
			logger.Warn("Request ID is missing in context")
			requestID = "unknown"
		}

		// 6. 构建日志字段
		// - 包含请求 ID、方法、路径、状态码、客户端 IP、用户代理和处理时长
		logFields := []zap.Field{
			zap.String("request_id", requestID),         // 请求唯一标识
			zap.String("method", method),                // 请求方法（如 GET、POST）
			zap.String("path", path),                    // 请求路径
			zap.Int("status", statusCode),               // 响应状态码
			zap.String("client_ip", clientIP),           // 客户端 IP 地址
			zap.String("user_agent", userAgent),         // 用户代理信息
			zap.Duration("total_latency", totalLatency), // 总请求处理时长
		}

		// 7. 网关服务添加额外字段
		// - 如果是网关服务，记录网关自身的处理时长
		if isGateway {
			logFields = append(logFields, zap.Duration("gateway_latency", gatewayProcessingTime))
		}

		// 8. 记录请求日志
		// - 使用 ZapLogger 记录请求的详细信息
		logger.Info("HTTP 请求", logFields...)
	}
}
