package middleware

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// CorsMiddleware 设置跨域中间件
// - 输出: gin.HandlerFunc 中间件函数，用于处理跨域请求
func CorsMiddleware() gin.HandlerFunc {
	return cors.New(cors.Config{
		// 允许跨域请求的来源
		// - 指定允许的域名列表，可以根据环境动态调整
		AllowOrigins: []string{
			"http://localhost:8000", // 开发环境：允许本地前端请求
			// 可添加生产环境域名，例如：
			// "https://yourproductiondomain.com",
		},
		// 允许的 HTTP 请求方法
		// - 支持常见的 RESTful 方法
		AllowMethods: []string{
			"GET",     // 查询请求
			"POST",    // 创建请求
			"PUT",     // 更新请求
			"DELETE",  // 删除请求
			"OPTIONS", // 预检请求
		},
		// 允许的请求头
		// - 指定客户端可以发送的自定义头
		AllowHeaders: []string{
			"Origin",           // 请求来源
			"Content-Type",     // 请求内容类型
			"Authorization",    // 认证令牌
			"X-Requested-With", // AJAX 请求标识
		},
		// 是否允许携带凭证
		// - 设置为 true 以支持 Cookie 或认证头跨域
		AllowCredentials: true,
		// 预检请求缓存时间（单位：秒）
		// - 设置为 12 小时（12 * 3600 秒）
		MaxAge: 12 * 3600,
	})
}
