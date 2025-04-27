package config

// CorsConfig 定义 CORS 相关配置
type CorsConfig struct {
	AllowOrigins     []string `mapstructure:"allow_origins" yaml:"allow_origins"` // 允许的来源列表
	AllowMethods     []string `mapstructure:"allow_methods" yaml:"allow_methods"`
	AllowHeaders     []string `mapstructure:"allow_headers" yaml:"allow_headers"`
	AllowCredentials bool     `mapstructure:"allow_credentials" yaml:"allow_credentials"`
	MaxAge           int      `mapstructure:"max_age" yaml:"max_age"` // 预检请求缓存时间 (秒)
}
