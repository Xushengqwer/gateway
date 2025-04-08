package config

import "time"

// RateLimitConfig 定义速率限制的相关配置。
// 例如：
//
//	Capacity: 100        // 令牌桶容量
//	RefillInterval: 1s   // 令牌补充间隔
//	CleanupInterval: 5m  // 定期清理不活跃IP的间隔
//	IdleTimeout: 10m     // IP多久不访问就算不活跃
type RateLimitConfig struct {
	Capacity        int           `mapstructure:"capacity" json:"capacity" yaml:"capacity"`
	RefillInterval  time.Duration `mapstructure:"refill_interval" json:"refill_interval" yaml:"refill_interval"`
	CleanupInterval time.Duration `mapstructure:"cleanup_interval" json:"cleanup_interval" yaml:"cleanup_interval"`
	IdleTimeout     time.Duration `mapstructure:"idle_timeout" json:"idle_timeout" yaml:"idle_timeout"`
}
