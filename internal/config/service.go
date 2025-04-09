package config

import "github.com/Xushengqwer/gateway/pkg/enums"

//  todo 每个服务的接口需要在这里写需要什么角色才能访问，需要写一个配置文件

// RouteConfig 定义基于路径的路由规则
type RouteConfig struct {
	Path         string           `yaml:"path"`         // 资源路径（根据资源路径来选择权限）
	AllowedRoles []enums.UserRole `yaml:"allowedRoles"` // 该路径允许的角色
}

// ServiceConfig 定义单个服务的配置
type ServiceConfig struct {
	Name        string        `yaml:"name"`                  // 服务名称
	Host        string        `yaml:"host,omitempty"`        // 单机部署的主机地址（可选）
	Port        int           `yaml:"port,omitempty"`        // 服务端口（单机用必填，K8s 用可选）
	ServiceName string        `yaml:"serviceName,omitempty"` // K8s Service 名称（K8s 用必填）
	Namespace   string        `yaml:"namespace,omitempty"`   // K8s 命名空间（可选，默认与网关相同）
	Scheme      string        `yaml:"scheme,omitempty"`      // 协议（http 或 https，默认 http）
	Prefix      string        `yaml:"prefix"`                // 服务路径前缀，示例api/v1
	Routes      []RouteConfig `yaml:"routes,omitempty"`      // 基于路径的权限（可选）
	PublicPaths []string      `yaml:"publicPaths,omitempty"` // 新增字段
}

// Config 定义网关的整体配置
type Config struct {
	Services []ServiceConfig `yaml:"services"` // 服务配置列表
}
