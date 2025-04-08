package enums

// UserStatus 定义用户状态的枚举类型
type UserStatus uint

// 用户状态枚举值
const (
	StatusActive      UserStatus = 0 // 活跃，用户可以正常操作
	StatusBlacklisted UserStatus = 1 // 拉黑，用户被禁止访问
)

// String 将 UserStatus 转换为字符串表示
// - 输入: s UserStatus，待转换的用户状态
// - 输出: string 状态对应的字符串表示
// - 意图: 便于日志记录或调试时显示用户状态的可读形式
func (s UserStatus) String() string {
	switch s {
	case StatusActive:
		return "active"
	case StatusBlacklisted:
		return "blacklisted"
	default:
		return "unknown"
	}
}
