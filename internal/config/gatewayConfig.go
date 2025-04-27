package config

import (
	"github.com/Xushengqwer/go-common/config"
)

// GatewayConfig 定义网关服务的全局配置
// - 包含 JWT、Zap 日志、速率限制、服务列表等配置
type GatewayConfig struct {
	ZapConfig    config.ZapConfig    `mapstructure:"zapConfig" json:"zapConfig" yaml:"zapConfig"`          // Zap 日志配置
	TracerConfig config.TracerConfig `mapstructure:"tracerConfig" json:"tracerConfig" yaml:"tracerConfig"` // 分布式追踪配置
	Server       config.ServerConfig `mapstructure:"server" json:"server" yaml:"server"`

	JWTConfig       JWTConfig        `mapstructure:"jwtConfig" json:"jwtConfig" yaml:"jwtConfig"`                   // JWT 认证配置
	RateLimitConfig *RateLimitConfig `mapstructure:"rateLimitConfig" json:"rateLimitConfig" yaml:"rateLimitConfig"` // 速率限制配置
	Services        []ServiceConfig  `mapstructure:"services" json:"services" yaml:"services"`                      // 下游服务配置列表
	Cors            CorsConfig       `mapstructure:"cors" yaml:"cors"`                                              // **新增 CORS 配置段**
}
