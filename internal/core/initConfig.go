package core

import (
	"flag"
	"fmt"
	"log"
	"os"
)

// InitConfig 初始化配置加载
// 设计目标：根据环境变量和命令行参数决定配置加载方式和路径，提供灵活的配置来源
// 返回值：
//   - configPath: 配置文件路径（如果从文件加载）
//   - configSource: 配置来源（"file" 或 "env"）
//
// 注意事项：
//   - CONFIG_SOURCE 环境变量决定加载方式，优先级：环境变量 > 命令行参数 > 默认值 "file"。
//   - K8S_ENV 环境变量和命令行参数决定是否在 Kubernetes 环境中运行。
//   - 如果未设置 ENV 变量或命令行参数，默认使用 "development" 环境。
//   - 命令行参数通过 flag 包解析，需在程序启动时调用 flag.Parse()。
//   - 日志记录使用 Go 标准库的 log 包，确保在配置加载前可用。
func InitConfig() (configPath string, configSource string) {
	// 获取配置来源
	configSource = os.Getenv("CONFIG_SOURCE")
	if configSource == "" {
		flag.StringVar(&configSource, "config-source", "file", "配置来源，可选 'file' 或 'env'")
		flag.Parse()
		if configSource == "" {
			configSource = "file" // 默认从文件加载
		}
		log.Printf("未设置 CONFIG_SOURCE 环境变量，从命令行参数或默认值获取: %s", configSource)
	}

	if configSource == "file" {
		// 判断是否在 Kubernetes 环境中
		k8sEnv := os.Getenv("K8S_ENV")
		if k8sEnv == "" {
			flag.StringVar(&k8sEnv, "k8s-env", "false", "是否在 K8S 环境中运行，'true' 或 'false'")
			flag.Parse()
		}
		if k8sEnv == "true" {
			configPath = "/etc/config/config.yaml" // K8S ConfigMap/Secret 挂载路径
			log.Printf("检测到 K8S 环境，从挂载路径加载配置文件: %s", configPath)
		} else {
			// 单机环境下的配置路径
			env := os.Getenv("ENV")
			if env == "" {
				flag.StringVar(&env, "env", "development", "运行环境，例如 'development' 或 'production'")
				flag.Parse()
				if env == "" {
					env = "development" // 默认环境
				}
			}
			configPath = fmt.Sprintf("config/config.%s.yaml", env)
			log.Printf("单机环境，从本地路径加载配置文件: %s", configPath)
		}
	} else if configSource == "env" {
		// 从环境变量加载，configPath 留空
		configPath = ""
		log.Println("配置来源设置为环境变量，将从环境变量加载")
	}

	return configPath, configSource
}
