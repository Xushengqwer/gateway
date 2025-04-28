package core

import (
	"errors"
	"fmt"
	"github.com/Xushengqwer/gateway/internal/config"
	"github.com/Xushengqwer/go-common/core"
	"github.com/Xushengqwer/go-common/models/enums"

	"go.uber.org/zap"

	"github.com/golang-jwt/jwt/v5"
)

// JWTUtilityInterface 定义 JWT 工具的接口
// - 用于解析 JWT 令牌，验证访问令牌和刷新令牌的有效性
// - 网关层仅需解析功能，生成 token 的功能由用户服务或其他认证服务负责
type JWTUtilityInterface interface {
	// ParseAccessToken 解析并验证访问令牌
	// - 输入: tokenString 待解析的令牌字符串
	// - 输出: 解析后的 CustomClaims 和可能的错误
	ParseAccessToken(tokenString string) (*CustomClaims, error)
}

// CustomClaims 定义 JWT 的声明结构体，包含标准字段和自定义字段
type CustomClaims struct {
	UserID               string           `json:"user_id"`  // 用户ID，唯一标识用户
	Role                 enums.UserRole   `json:"role"`     // 用户角色，例如管理员或普通用户
	Status               enums.UserStatus `json:"status"`   // 用户状态，例如活跃或禁用
	Platform             enums.Platform   `json:"platform"` // 客户端平台，例如 Web 或微信小程序
	jwt.RegisteredClaims                  // 嵌入 JWT v5 的标准声明字段
}

// JWTUtility 实现 JWTUtilityInterface 接口的结构体
type JWTUtility struct {
	cfg    *config.GatewayConfig // JWT 配置，包含密钥、发行者等信息
	logger *core.ZapLogger       // 日志记录器，用于记录解析错误
}

// NewJWTUtility 创建 JWTUtility 实例，通过依赖注入初始化
// - 输入: cfg JWT 配置实例, logger ZapLogger 实例
// - 输出: JWTUtilityInterface 接口实例
func NewJWTUtility(cfg *config.GatewayConfig, logger *core.ZapLogger) JWTUtilityInterface {
	return &JWTUtility{cfg: cfg, logger: logger}
}

// ParseAccessToken 解析并验证访问令牌
// - 输入: tokenString 待解析的令牌字符串
// - 输出: 解析后的 CustomClaims 和可能的错误
func (ju *JWTUtility) ParseAccessToken(tokenString string) (*CustomClaims, error) {
	secret := []byte(ju.cfg.JWTConfig.SecretKey)

	// 创建解析器，启用 v5 的严格验证选项
	parser := jwt.NewParser(
		jwt.WithExpirationRequired(),            // 强制要求令牌包含过期时间
		jwt.WithIssuer(ju.cfg.JWTConfig.Issuer), // 验证发行者是否匹配配置中的值
	)

	// 解析令牌
	claims, err := ju.parseToken(tokenString, secret, parser)
	if err != nil {
		ju.logger.Error("解析访问令牌失败", zap.String("token", tokenString), zap.Error(err))
		return nil, err
	}

	// 验证用户状态
	if claims.Status != enums.StatusActive {
		ju.logger.Warn("用户状态无效", zap.String("user_id", claims.UserID), zap.String("status", claims.Status.String()))
		return nil, errors.New("用户状态无效")
	}

	// 验证平台
	if !enums.IsValidPlatform(claims.Platform) {
		ju.logger.Warn("无效的平台类型", zap.String("platform", string(claims.Platform)))
		return nil, errors.New("无效的平台类型")
	}

	return claims, nil
}

// parseToken 辅助函数，用于解析和验证 JWT 令牌
// - 输入: tokenString 待解析的令牌字符串, secret 签名密钥, parser v5 的解析器实例
// - 输出: 解析后的 CustomClaims 和可能的错误
func (ju *JWTUtility) parseToken(tokenString string, secret []byte, parser *jwt.Parser) (*CustomClaims, error) {
	// 使用 v5 的 Parser 解析令牌
	token, err := parser.ParseWithClaims(tokenString, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		// 验证签名算法是否为 HS256
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("签名算法不匹配: %v", token.Header["alg"])
		}
		return secret, nil
	})

	// 如果解析失败，返回错误
	if err != nil {
		return nil, err
	}

	// 类型断言并验证令牌有效性
	claims, ok := token.Claims.(*CustomClaims)
	if !ok || !token.Valid {
		return nil, errors.New("无效的JWT声明")
	}

	return claims, nil
}
