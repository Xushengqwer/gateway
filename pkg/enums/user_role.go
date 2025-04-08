package enums

// UserRole 定义用户角色的枚举类型
type UserRole uint

// 用户角色枚举值
const (
	RoleAdmin UserRole = iota // 0 - 管理员，具有最高权限
	RoleUser                  // 1 - 普通用户，标准用户角色
	RoleGuest                 // 2 - 访客，限制性访问权限
)

// String 将 UserRole 转换为字符串表示
// - 输入: r UserRole，待转换的用户角色
// - 输出: string 角色对应的字符串表示
// - 意图: 便于日志记录或调试时显示用户角色的可读形式
func (r UserRole) String() string {
	switch r {
	case RoleAdmin:
		return "admin"
	case RoleUser:
		return "user"
	case RoleGuest:
		return "guest"
	default:
		return "unknown"
	}
}
