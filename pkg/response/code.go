package response

// 定义应用程序中使用的错误码
// 这些错误码用于标准化 API 响应中的 code 字段，确保所有服务和网关的错误码一致
const (
	// Success 操作成功
	Success = 0

	// 4xx 客户端错误

	ErrCodeClientInvalidInput        = 40001 // 输入参数无效
	ErrCodeClientUnauthorized        = 40101 // 未授权
	ErrCodeClientAccessTokenExpired  = 40102 // Access Token 过期
	ErrCodeClientRefreshTokenExpired = 40103 // Refresh Token 过期
	ErrCodeClientForbidden           = 40301 // 客户端被禁止访问
	ErrCodeClientResourceNotFound    = 40401 // 未找到指定资源
	ErrCodeClientRateLimitExceeded   = 42901 // 请求频率超出限制

	// 5xx 服务器错误

	ErrCodeServerInternal  = 50001 // 服务器内部错误
	ErrCodeServerTimeout   = 50002 // 操作超时
	ErrCodeServiceNotFound = 50003 // 服务未找到（网关专用）
)
