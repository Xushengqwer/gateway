package core

import (
	"fmt"
	"gateway/internal/config"
	"log"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

// LoadConfig 加载配置的核心函数
// 设计目标：支持从文件和环境变量加载配置，支持文件配置的热加载，灵活适配单机和 K8S 环境
// - 该函数根据运行环境加载对应的配置文件（如 config.development.yaml），并解析到 Config 结构体返回
// - 遵循单一职责原则，仅加载和解析配置，不应该关心如何获取 env（环境变量或命令行），将参数解析放到 main.go 中，保持函数职责清晰
// 参数：
//   - configPath: 配置文件路径（如果为空，则尝试从环境变量加载）
//
// 返回值：
//   - *config.GatewayConfig: 解析后的配置结构体
//   - error: 如果加载或解析失败，返回错误
//
// 注意事项：
//   - 优先级：文件 > 环境变量。如果 configPath 不为空且文件加载成功，则使用文件配置。
//   - 如果文件加载失败，会回退到环境变量加载。
//   - 热加载仅在文件加载时生效，通过 viper.WatchConfig() 实现，环境变量加载不支持热加载。
//   - 日志记录使用 Go 标准库的 log 包，确保在配置加载前可用。
func LoadConfig(configPath string) (*config.GatewayConfig, error) {
	v := viper.New()

	// 步骤 1：尝试从文件加载配置
	if configPath != "" {
		v.SetConfigFile(configPath)
		v.SetConfigType("yaml")
		if err := v.ReadInConfig(); err == nil {
			log.Printf("成功从文件加载配置: %s", configPath)
			cfg, err := unmarshalConfig(v)
			if err != nil {
				return nil, err
			}
			// 启用热加载
			// 为什么使用 WatchConfig：
			//   - viper 的 WatchConfig 方法可以监听配置文件变化，适用于动态更新场景。
			//   - 在 K8S 中，若 ConfigMap 更新并同步到挂载文件，热加载也能生效。
			v.WatchConfig()
			v.OnConfigChange(func(e fsnotify.Event) {
				log.Printf("配置文件发生变化: %s", e.Name)
				if err := v.Unmarshal(&cfg); err != nil {
					log.Printf("重新解析配置文件失败: %v", err)
				} else {
					log.Printf("配置已更新: %+v", cfg)
				}
			})
			return cfg, nil
		} else {
			log.Printf("从文件加载配置失败: %s, 错误: %v", configPath, err)
		}
	}

	// 步骤 2：回退到环境变量加载
	// 注意：环境变量加载不支持热加载，因为环境变量更新需要重启程序。
	v.AutomaticEnv()
	log.Println("从环境变量加载配置")
	return unmarshalConfig(v)
}

// unmarshalConfig 将 viper 的配置解析到结构体
// 设计目标：将配置解析逻辑抽取为独立函数，提高复用性和可测试性
// 参数：
//   - v: viper 实例，包含已加载的配置数据
//
// 返回值：
//   - *config.GatewayConfig: 解析后的配置结构体
//   - error: 如果解析失败，返回错误
func unmarshalConfig(v *viper.Viper) (*config.GatewayConfig, error) {
	var cfg config.GatewayConfig
	if err := v.Unmarshal(&cfg); err != nil {
		log.Printf("解析配置失败: %v", err)
		return nil, fmt.Errorf("无法解析配置: %v", err)
	}
	log.Printf("配置解析成功: %+v", cfg)
	return &cfg, nil
}
