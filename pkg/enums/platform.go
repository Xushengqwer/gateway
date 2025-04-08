package enums

import "errors"

// Platform 定义客户端平台的枚举类型
type Platform string

// 平台枚举值
const (
	PlatformWeb    Platform = "web"    // 网站
	PlatformWechat Platform = "wechat" // 微信小程序
	PlatformApp    Platform = "app"    // 移动应用
)

// PlatformFromString 将字符串转换为 Platform 类型
// - 输入: s string，待转换的字符串
// - 输出: Platform 转换后的平台类型
// - 输出: error 如果字符串不是有效的平台类型，则返回错误
// - 意图: 确保传入的平台字符串是预定义的有效值，避免无效平台类型导致的错误
func PlatformFromString(s string) (Platform, error) {
	switch Platform(s) {
	case PlatformWeb, PlatformWechat, PlatformApp:
		return Platform(s), nil
	default:
		return "", errors.New("无效的平台类型")
	}
}

// IsValidPlatform 验证平台类型是否有效
// - 输入: p Platform，待验证的平台类型
// - 输出: bool 如果平台类型是预定义的有效值，则返回 true，否则返回 false
// - 意图: 用于在解析 JWT 或其他场景中验证平台字段的合法性
func IsValidPlatform(p Platform) bool {
	_, err := PlatformFromString(string(p))
	return err == nil
}
