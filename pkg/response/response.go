package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// APIResponse 定义通用的泛型响应结构体，用于标准化所有 API 响应
// - T: 泛型类型参数，表示 Data 字段的具体类型
type APIResponse[T any] struct {
	Code    int    `json:"code" example:"0"`                    // 响应状态码，0 表示成功，其他值表示错误
	Message string `json:"message,omitempty" example:"success"` // 可选的响应消息，若为空则不输出
	Data    T      `json:"data,omitempty"`                      // 响应数据，类型由 T 指定，若无数据则为 nil，且在 JSON 中省略
}

// RespondSuccess 发送成功的 HTTP 响应，包含可选的自定义消息和数据
// - c: Gin 上下文，用于发送响应
// - data: 响应数据，类型为 T，可为 nil
// - message: 可选的自定义消息，默认为 "success"
func RespondSuccess[T any](c *gin.Context, data T, message ...string) {
	msg := "success" // 默认成功消息
	if len(message) > 0 {
		msg = message[0] // 如果提供了自定义消息，则使用它
	}
	c.JSON(http.StatusOK, APIResponse[T]{
		Code:    Success, // 成功状态码，通常为 0
		Message: msg,     // 自定义或默认消息
		Data:    data,    // 响应数据，可选
	})
}

// RespondError 发送错误的 HTTP 响应，包含指定的状态码、错误码和消息
// - c: Gin 上下文，用于发送响应
// - statusCode: HTTP 状态码，例如 400、500
// - code: 应用特定的错误码，例如 ErrCodeClientInvalidInput
// - message: 错误的具体描述
func RespondError(c *gin.Context, statusCode int, code int, message string) {
	c.JSON(statusCode, APIResponse[any]{
		Code:    code,    // 应用特定的错误码
		Message: message, // 错误消息
		Data:    nil,     // 数据字段为空，在 JSON 中会被省略
	})
}
