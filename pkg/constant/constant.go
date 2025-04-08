package constant

import "time"

// RequestTimeout 全局请求超时设置
// - 输入: 无 (常量，直接使用 10 * time.Second)
// - 输出: time.Duration 类型，表示所有 HTTP 请求的默认超时时间，当前为 10 秒
const RequestTimeout = 10 * time.Second

// RoleContextKey 上下文中的用户角色键名
// - 输入: 无 (常量，直接使用 "role")
// - 输出: string 类型，用于在请求上下文中存储和获取用户角色的键名
const RoleContextKey = "role"

// StatusContextKey 上下文中的状态键名
// - 输入: 无 (常量，直接使用 "status")
// - 输出: string 类型，用于在请求上下文中存储和获取状态信息的键名
const StatusContextKey = "status"

// RequestIDKey 上下文中的请求 ID 键名
// - 输入: 无 (常量，直接使用 "RequestID")
// - 输出: string 类型，用于在请求上下文中存储和获取唯一请求 ID 的键名
const RequestIDKey = "RequestID"
