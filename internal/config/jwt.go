package config

// JWTConfig 定义JWT认证功能的相关配置，包含密钥、过期时间等信息，用于生成和验证JWT。
type JWTConfig struct {
	SecretKey     string `mapstructure:"secret_key" yaml:"secret_key"`         // 用于签名Access Token的密钥
	RefreshSecret string `mapstructure:"refresh_secret" yaml:"refresh_secret"` // 用于签名Refresh Token的密钥
	Issuer        string `mapstructure:"issuer" yaml:"issuer"`                 // JWT的签发者
}
