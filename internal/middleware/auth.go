package middleware

import (
	"errors"
	"fmt"
	"github.com/Xushengqwer/gateway/internal/core"
	"github.com/Xushengqwer/go-common/constants"
	"github.com/Xushengqwer/go-common/models/enums"

	"github.com/Xushengqwer/go-common/response"

	"github.com/golang-jwt/jwt/v5"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// AuthMiddleware 定义认证中间件，用于验证请求中的访问令牌
// - 输入: jwtUtil JWT 工具实例，用于解析令牌
// - 输出: gin.HandlerFunc 中间件函数
func AuthMiddleware(jwtUtil core.JWTUtilityInterface) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. 获取请求头中的 Authorization 字段
		// - 期望格式: "Bearer <token>"
		// - 如果缺失，返回未授权错误
		authorizationHeader := c.GetHeader("Authorization")
		if authorizationHeader == "" {
			response.RespondError(c, http.StatusUnauthorized, response.ErrCodeClientUnauthorized, "缺少或不正确的令牌")
			c.Abort()
			return
		}

		// 2. 解析 Bearer Token
		// - 从 Authorization 头中提取令牌字符串
		// - 如果格式错误，返回未授权错误
		accessToken, err := parseBearerToken(authorizationHeader)
		if err != nil {
			response.RespondError(c, http.StatusUnauthorized, response.ErrCodeClientUnauthorized, "令牌格式错误")
			c.Abort()
			return
		}

		// 3. 解析并验证访问令牌
		// - 使用 JWTUtilityInterface 解析令牌，获取声明
		// - 如果解析失败，根据错误类型返回相应响应
		claims, parseErr := jwtUtil.ParseAccessToken(accessToken)
		if parseErr != nil {
			fmt.Printf("解析认证令牌错误: %v\n", parseErr)

			// 4. 检查具体错误类型（使用 v5 的错误常量）
			// - 根据错误类型返回不同的错误码和消息
			switch {
			case errors.Is(parseErr, jwt.ErrTokenExpired):
				// - 令牌过期错误
				response.RespondError(c, http.StatusUnauthorized, response.ErrCodeClientAccessTokenExpired, "访问令牌已过期")
			case errors.Is(parseErr, jwt.ErrTokenMalformed), errors.Is(parseErr, jwt.ErrTokenSignatureInvalid), errors.Is(parseErr, jwt.ErrTokenInvalidClaims):
				// - 令牌无效（格式错误、签名无效或声明无效）
				response.RespondError(c, http.StatusUnauthorized, response.ErrCodeClientUnauthorized, "无效令牌")
			default:
				// - 其他未知错误
				response.RespondError(c, http.StatusUnauthorized, response.ErrCodeClientUnauthorized, "令牌验证失败")
			}
			c.Abort()
			return
		}

		// 5. 验证平台是否匹配
		// - 根据请求路径或 header 判断预期平台，与令牌中的平台进行比较
		// - 如果不匹配，返回禁止访问错误
		expectedPlatform, err := getExpectedPlatform(c.Request)
		if err != nil {
			response.RespondError(c, http.StatusBadRequest, response.ErrCodeClientInvalidInput, err.Error())
			c.Abort()
			return
		}
		if string(claims.Platform) != expectedPlatform {
			response.RespondError(c, http.StatusForbidden, response.ErrCodeClientForbidden, "平台不匹配")
			c.Abort()
			return
		}

		// 6. 检查用户状态
		// - 如果用户被拉黑，禁止访问
		// - 返回禁止访问错误
		if claims.Status == enums.StatusBlacklisted {
			response.RespondError(c, http.StatusForbidden, response.ErrCodeClientForbidden, "用户已被拉黑")
			c.Abort()
			return
		}

		// 7. 令牌有效，将用户信息存入上下文
		// - 将 UserID、Role、Status 存入 gin.Context，供后续处理使用
		// - 继续处理请求
		// 将用户信息添加到 HTTP 头
		// 将用户状态和角色存入上下文
		c.Set(string(constants.StatusKey), claims.Status) // 假设 claims 中有 Status
		c.Set(string(constants.RoleKey), claims.Role)     // 假设 claims 中有 Role
		c.Request.Header.Set("X-User-ID", claims.UserID)
		c.Request.Header.Set("X-User-Role", claims.Role.String())
		c.Request.Header.Set("X-User-Status", claims.Status.String())
		c.Request.Header.Set("X-Platform", string(claims.Platform))
		c.Next()
	}
}

// parseBearerToken 从 Authorization 头中提取令牌
// - 输入: authHeader Authorization 头字符串，格式为 "Bearer <token>"
// - 输出: 提取的令牌字符串和可能的错误
func parseBearerToken(authHeader string) (string, error) {
	// 示例: "Bearer abc.xyz.123"
	// - 分割 Authorization 头，检查是否符合 Bearer 格式
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		return "", errors.New("invalid Authorization header format")
	}
	return parts[1], nil
}

// getExpectedPlatform 根据请求头或路径判断预期平台
// - 输入: r HTTP 请求对象
// - 输出: 预期平台字符串
// - 意图: 优先从请求头 X-Platform 获取平台信息，路径前缀作为辅助，确保平台信息准确
func getExpectedPlatform(r *http.Request) (string, error) {
	// 优先从请求头获取 X-Platform
	platform := r.Header.Get("X-Platform")
	if platform != "" {
		// 验证平台值是否有效
		if _, err := enums.PlatformFromString(platform); err == nil {
			return platform, nil
		}
		return "", fmt.Errorf("无效的 X-Platform 值: %s", platform)
	}

	// 如果请求头中没有 X-Platform，尝试从路径前缀判断
	if strings.HasPrefix(r.URL.Path, "/wechat") {
		return "wechat", nil
	} else if strings.HasPrefix(r.URL.Path, "/web") {
		return "web", nil
	} else if strings.HasPrefix(r.URL.Path, "/app") {
		return "app", nil
	}

	// 如果无法判断平台，返回错误
	return "", errors.New("无法确定预期平台，缺少 X-Platform 请求头或路径前缀")
}
