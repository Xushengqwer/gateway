package middleware

import (
	"github.com/Xushengqwer/gateway/internal/config" // 导入配置包
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"time" // 需要导入 time
)

// CorsMiddleware 设置跨域中间件 (从配置加载)
func CorsMiddleware(cfg config.CorsConfig) gin.HandlerFunc { // 接收 CorsConfig
	// 如果配置中未提供值，可以使用默认值
	if len(cfg.AllowOrigins) == 0 {
		// 警告或设置一个非常严格的默认值，或者允许所有（不推荐用于生产）
		// cfg.AllowOrigins = []string{"*"} // 谨慎使用
		cfg.AllowOrigins = []string{} // 或者不允许任何跨域作为默认？
	}
	if len(cfg.AllowMethods) == 0 {
		cfg.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	}
	if len(cfg.AllowHeaders) == 0 {
		cfg.AllowHeaders = []string{"Origin", "Content-Type", "Authorization", "X-Requested-With"}
	}
	maxAgeDuration := time.Duration(cfg.MaxAge) * time.Second
	if cfg.MaxAge == 0 {
		maxAgeDuration = 12 * time.Hour // 默认 12 小时
	}

	return cors.New(cors.Config{
		AllowOrigins:     cfg.AllowOrigins,     // 使用配置的值
		AllowMethods:     cfg.AllowMethods,     // 使用配置的值
		AllowHeaders:     cfg.AllowHeaders,     // 使用配置的值
		AllowCredentials: cfg.AllowCredentials, // 使用配置的值
		MaxAge:           maxAgeDuration,       // 使用计算后的 Duration
	})
}
