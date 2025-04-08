package middleware

import (
	"gateway/internal/config"
	"gateway/internal/core"
	"gateway/pkg/response"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// RateLimiter 定义令牌桶限流器，用于存储每个 IP 的令牌数据和访问时间
type RateLimiter struct {
	capacity       int           // 令牌桶的最大容量
	tokens         int           // 当前可用令牌数
	refillInterval time.Duration // 令牌补充的时间间隔
	lastRefill     time.Time     // 上次补充令牌的时间
	lastAccessed   time.Time     // 上次访问时间，用于清理不活跃 IP
	mu             sync.Mutex    // 互斥锁，确保并发安全
}

// NewRateLimiter 创建新的 RateLimiter 实例
// - 输入: capacity 令牌桶容量, refillInterval 令牌补充间隔
// - 输出: RateLimiter 实例指针
func NewRateLimiter(capacity int, refillInterval time.Duration) *RateLimiter {
	now := time.Now()
	return &RateLimiter{
		capacity:       capacity,       // 设置令牌桶容量
		tokens:         capacity,       // 初始化时令牌桶满
		refillInterval: refillInterval, // 设置补充间隔
		lastRefill:     now,            // 初始化上次补充时间
		lastAccessed:   now,            // 初始化上次访问时间
	}
}

// Allow 判断是否允许获取令牌
// - 输出: bool 值，true 表示允许请求，false 表示超出限流
func (rl *RateLimiter) Allow() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	// 1. 计算令牌补充
	// - 根据当前时间与上次补充时间之差，计算可补充的令牌数
	now := time.Now()
	elapsed := now.Sub(rl.lastRefill)
	refillCount := int(elapsed / rl.refillInterval)
	if refillCount > 0 {
		newTokens := rl.tokens + refillCount
		if newTokens > rl.capacity {
			newTokens = rl.capacity // 确保不超过容量
		}
		rl.tokens = newTokens
		rl.lastRefill = now // 更新上次补充时间
	}

	// 2. 检查并消耗令牌
	// - 如果有可用令牌，消耗一个并允许请求
	if rl.tokens > 0 {
		rl.tokens--
		rl.lastAccessed = now // 更新访问时间
		return true
	}
	return false
}

// GetLastAccessed 获取上次访问时间
// - 输出: time.Time 表示最近一次访问时间，用于清理不活跃 IP
func (rl *RateLimiter) GetLastAccessed() time.Time {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	return rl.lastAccessed
}

// RateLimitMiddleware 定义限流中间件，用于限制每个 IP 的请求频率
// - 输入: logger ZapLogger 实例用于日志记录, cfg RateLimitConfig 配置限流参数
// - 输出: gin.HandlerFunc 中间件函数
func RateLimitMiddleware(logger *core.ZapLogger, cfg *config.RateLimitConfig) gin.HandlerFunc {
	// 1. 初始化 IP 到 RateLimiter 的映射
	// - 使用 sync.Map 存储每个 IP 的限流器实例
	var rateLimiters sync.Map

	// 2. 启动后台清理协程
	// - 定期检查并移除不活跃的 IP
	go func() {
		ticker := time.NewTicker(cfg.CleanupInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				now := time.Now()
				rateLimiters.Range(func(key, value interface{}) bool {
					ip := key.(string)
					rl := value.(*RateLimiter)
					if now.Sub(rl.GetLastAccessed()) > cfg.IdleTimeout {
						// - 超过空闲时间，删除该 IP 的限流器
						rateLimiters.Delete(ip)
						logger.Info("已移除不活跃的 IP 从限流映射中",
							zap.String("ip", ip),
						)
					}
					return true
				})
			}
		}
	}()

	// 3. 返回中间件主逻辑
	return func(c *gin.Context) {
		// 4. 获取客户端 IP
		// - 从 gin.Context 中提取客户端 IP 地址
		clientIP := getClientIP(c)

		// 5. 获取或创建 RateLimiter
		// - 检查该 IP 是否已有限流器，若无则创建并存储
		val, ok := rateLimiters.Load(clientIP)
		if !ok {
			newRL := NewRateLimiter(cfg.Capacity, cfg.RefillInterval)
			val, _ = rateLimiters.LoadOrStore(clientIP, newRL)
		}
		rl := val.(*RateLimiter)

		// 6. 检查是否允许请求
		// - 调用 Allow 方法判断是否可获取令牌
		if !rl.Allow() {
			// 7. 限流超出处理
			// - 记录警告日志
			logger.Warn("请求频率超出限制",
				zap.String("client_ip", clientIP),
				zap.String("path", c.Request.URL.Path),
			)

			// - 设置 Retry-After 头并返回错误响应
			retryAfter := cfg.RefillInterval.Seconds() // 下一个令牌的补充时间
			c.Header("Retry-After", formatFloatToString(retryAfter, 1))
			response.RespondError(c, http.StatusTooManyRequests, response.ErrCodeClientRateLimitExceeded, "请求频率超出限制，请稍后重试")
			c.Abort()
			return
		}

		// 8. 允许请求继续
		// - 调用 c.Next() 处理后续请求
		c.Next()
	}
}

// getClientIP 从 gin.Context 中获取客户端 IP
// - 输入: c Gin 上下文对象
// - 输出: string 表示客户端 IP 地址
func getClientIP(c *gin.Context) string {
	// 1. 获取客户端 IP
	// - 使用 c.ClientIP() 获取 IP 地址
	clientIP := c.ClientIP()

	// 2. 验证 IP 有效性
	// - 如果无法解析为有效 IP，返回 "invalid_ip"
	if ip := net.ParseIP(clientIP); ip == nil {
		return "invalid_ip"
	}
	return clientIP
}

// formatFloatToString 将浮点数转换为字符串，用于 Retry-After 头
// - 输入: f 浮点数, precision 小数点后精度
// - 输出: string 表示格式化后的字符串
func formatFloatToString(f float64, precision int) string {
	// 使用 strconv.FormatFloat 格式化浮点数
	return strconv.FormatFloat(f, 'f', precision, 64)
}
