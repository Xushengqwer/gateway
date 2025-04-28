package middleware

import (
	"github.com/Xushengqwer/gateway/internal/config"
	"github.com/Xushengqwer/go-common/constants"
	"github.com/Xushengqwer/go-common/models/enums"
	"github.com/Xushengqwer/go-common/response"
	"net/http" // HTTP 状态码和请求处理
	"strings"  // 字符串操作，用于路径匹配

	"github.com/gin-gonic/gin" // Gin 框架
)

// PermissionMiddleware 定义权限中间件，用于检查用户角色是否满足请求路径的权限要求
// - 输入: cfg *config.Config，网关服务的配置，包含服务路径和允许角色的映射
// - 输出: gin.HandlerFunc 中间件函数
// - 意图: 在网关层根据配置文件中的权限规则，拦截不符合角色要求的请求
func PermissionMiddleware(cfg *config.GatewayConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取用户状态，从上下文读取 StatusContextKey 的值
		// 如果不存在，返回禁止访问错误
		status, exists := c.Get(constants.StatusContextKey)
		if !exists {
			response.RespondError(c, http.StatusForbidden, response.ErrCodeClientForbidden, "状态获取失败")
			c.Abort() // 终止请求
			return
		}

		// 检查用户是否被拉黑
		// 如果状态为 StatusBlacklisted，返回用户被拉黑的错误
		if status == enums.StatusBlacklisted {
			response.RespondError(c, http.StatusForbidden, response.ErrCodeClientForbidden, "用户已被拉黑")
			c.Abort() // 终止请求
			return
		}

		// 获取用户角色，从上下文读取 RoleContextKey 的值
		// 如果不存在，返回权限不足错误
		roleList, exists := c.Get(constants.RoleContextKey)
		if !exists {
			response.RespondError(c, http.StatusForbidden, response.ErrCodeClientForbidden, "权限不足")
			c.Abort() // 终止请求
			return
		}

		// 将 roleList 转换为 enums.UserRole 类型
		// 如果类型转换失败，返回角色无效错误
		role, ok := roleList.(enums.UserRole)
		if !ok {
			response.RespondError(c, http.StatusForbidden, response.ErrCodeClientForbidden, "角色信息无效")
			c.Abort() // 终止请求
			return
		}

		// 获取请求路径，用于匹配配置文件中的服务
		path := c.Request.URL.Path

		// 遍历配置文件中的服务，匹配路径前缀
		for _, svc := range cfg.Services {
			if strings.HasPrefix(path, svc.Prefix) {
				// 计算相对于 Prefix 的路径
				relativePath := strings.TrimPrefix(path, svc.Prefix)

				// 检查 Routes 中的路径规则
				for _, route := range svc.Routes {
					// 检查请求路径是否以 route.Path 开头
					if strings.HasPrefix(relativePath, route.Path) {
						// 检查用户角色是否在该路径的 AllowedRoles 中
						for _, allowedRole := range route.AllowedRoles {
							if role == allowedRole {
								c.Next() // 角色匹配，允许继续处理请求
								return
							}
						}
						// 角色不在允许列表中，返回权限不足错误
						response.RespondError(c, http.StatusForbidden, response.ErrCodeClientForbidden, "权限不足")
						c.Abort() // 终止请求
						return
					}
				}

				// 如果没有匹配到任何 Route，默认拒绝访问
				response.RespondError(c, http.StatusForbidden, response.ErrCodeClientForbidden, "未定义的路径")
				c.Abort() // 终止请求
				return
			}
		}

		// 如果没有匹配的服务，返回服务未找到错误
		response.RespondError(c, http.StatusNotFound, response.ErrCodeClientResourceNotFound, "服务未找到")
		c.Abort() // 终止请求
	}
}
