package middleware

import (
	"github.com/Xushengqwer/gateway/internal/config"
	"github.com/Xushengqwer/go-common/constants"
	"github.com/Xushengqwer/go-common/models/enums"
	"github.com/Xushengqwer/go-common/response"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// MatchRoute 检查请求是否匹配给定的路由规则 (已导出)
func MatchRoute(route config.RouteConfig, requestPath, requestMethod string) (bool, int) {
	// 1. 检查 HTTP 方法
	if len(route.Methods) > 0 {
		methodMatch := false
		for _, m := range route.Methods {
			if strings.EqualFold(m, requestMethod) { // 使用不区分大小写的比较
				methodMatch = true
				break
			}
		}
		if !methodMatch {
			return false, 0 // 方法不匹配
		}
	}

	// 2. 检查路径
	routeSegments := strings.Split(strings.Trim(route.Path, "/"), "/")
	requestSegments := strings.Split(strings.Trim(requestPath, "/"), "/")

	// 处理根路径 "/" 的情况 (包括 / vs "")
	isRouteRoot := route.Path == "/" || (len(routeSegments) == 1 && routeSegments[0] == "")
	isRequestRoot := requestPath == "/" || (len(requestSegments) == 1 && requestSegments[0] == "")

	if isRouteRoot && isRequestRoot {
		return true, 1 // 都代表根路径，匹配
	}

	if len(routeSegments) != len(requestSegments) {
		return false, 0 // 段数必须相同
	}

	matchScore := 0
	for i, segment := range routeSegments {
		if strings.HasPrefix(segment, ":") || strings.HasPrefix(segment, "{") {
			// 如果请求段为空，则参数不匹配（除非我们允许空参数，但这里不允许）
			if requestSegments[i] == "" {
				return false, 0
			}
			matchScore++ // 参数匹配，分数较低
		} else if segment == requestSegments[i] {
			matchScore += 2 // 静态匹配，分数较高
		} else {
			return false, 0 // 任何段不匹配则整个路径不匹配
		}
	}

	return true, matchScore // 返回匹配成功和得分
}

// FindBestMatchingRoute 找到最匹配的路由规则 (已导出)
func FindBestMatchingRoute(routes []config.RouteConfig, relativePath, method string) (*config.RouteConfig, bool) {
	var bestMatch *config.RouteConfig
	bestScore := -1 // 初始化为 -1，确保任何匹配都优于它

	for i := range routes {
		route := routes[i]                                        // 使用索引来获取值，避免闭包问题
		matches, score := MatchRoute(route, relativePath, method) // 使用导出的函数
		if matches {
			currentSegments := strings.Split(strings.Trim(route.Path, "/"), "/")
			currentLength := len(currentSegments)

			bestLength := 0
			if bestMatch != nil {
				bestSegments := strings.Split(strings.Trim(bestMatch.Path, "/"), "/")
				bestLength = len(bestSegments)
			}

			if score > bestScore || (score == bestScore && currentLength > bestLength) {
				bestScore = score
				bestMatch = &route
			}
		}
	}

	return bestMatch, bestMatch != nil
}

// PermissionMiddleware 定义权限中间件 (重构版)
func PermissionMiddleware(cfg *config.GatewayConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		// --- 获取用户状态和角色 ---
		statusVal, exists := c.Get(constants.StatusContextKey)
		if !exists {
			response.RespondError(c, http.StatusForbidden, response.ErrCodeClientForbidden, "状态获取失败")
			c.Abort()
			return
		}
		status, ok := statusVal.(enums.UserStatus)
		if !ok {
			response.RespondError(c, http.StatusForbidden, response.ErrCodeClientForbidden, "状态信息无效")
			c.Abort()
			return
		}

		if status == enums.StatusBlacklisted {
			response.RespondError(c, http.StatusForbidden, response.ErrCodeClientForbidden, "用户已被拉黑")
			c.Abort()
			return
		}

		roleValue, exists := c.Get(constants.RoleContextKey)
		if !exists {
			response.RespondError(c, http.StatusForbidden, response.ErrCodeClientForbidden, "权限不足 (无法获取角色)")
			c.Abort()
			return
		}

		role, ok := roleValue.(enums.UserRole)
		if !ok {
			response.RespondError(c, http.StatusForbidden, response.ErrCodeClientForbidden, "角色信息无效")
			c.Abort()
			return
		}

		// --- 匹配逻辑 ---
		path := c.Request.URL.Path
		method := c.Request.Method

		for _, svc := range cfg.Services {
			if strings.HasPrefix(path, svc.Prefix) {
				relativePath := strings.TrimPrefix(path, svc.Prefix)
				if !strings.HasPrefix(relativePath, "/") && relativePath != "" {
					relativePath = "/" + relativePath
				} else if relativePath == "" {
					relativePath = "/"
				}

				bestRoute, found := FindBestMatchingRoute(svc.Routes, relativePath, method) // 使用导出的函数

				if !found {
					// 理论上不应发生，因为 createProxyHandler 已确保这是私有路由
					response.RespondError(c, http.StatusForbidden, response.ErrCodeClientForbidden, "无权访问该路径或路径未定义 (Perm)")
					c.Abort()
					return
				}

				hasPermission := false
				for _, allowedRole := range bestRoute.AllowedRoles {
					if role == allowedRole {
						hasPermission = true
						break
					}
				}

				if hasPermission {
					c.Next()
					return
				} else {
					response.RespondError(c, http.StatusForbidden, response.ErrCodeClientForbidden, "权限不足")
					c.Abort()
					return
				}
			}
		}

		response.RespondError(c, http.StatusNotFound, response.ErrCodeClientResourceNotFound, "服务未找到 (Perm)")
		c.Abort()
	}
}
