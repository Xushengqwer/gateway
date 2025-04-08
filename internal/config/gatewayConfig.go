package config

// GatewayConfig 定义网关服务的全局配置
// - 包含 JWT、Zap 日志、速率限制、服务列表等配置
type GatewayConfig struct {
	JWTConfig       JWTConfig       `mapstructure:"jwtConfig" json:"jwtConfig" yaml:"jwtConfig"`                   // JWT 认证配置
	ZapConfig       ZapConfig       `mapstructure:"zapConfig" json:"zapConfig" yaml:"zapConfig"`                   // Zap 日志配置
	RateLimitConfig RateLimitConfig `mapstructure:"rateLimitConfig" json:"rateLimitConfig" yaml:"rateLimitConfig"` // 速率限制配置
	Services        []ServiceConfig `mapstructure:"services" json:"services" yaml:"services"`                      // 下游服务配置列表
	ListenAddr      string          `mapstructure:"listenAddr" json:"listenAddr" yaml:"listenAddr"`                // 网关监听地址
}
