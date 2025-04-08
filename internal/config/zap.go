package config

// ZapConfig 定义 Zap 日志框架的配置参数，用于控制日志行为
// 当前设计:
//   - 仅支持 stdout 和 stderr 输出，普通日志（低于 Error 级别）输出到 stdout，错误日志（Error 及以上）输出到 stderr
//   - 原因: 适配 K8S 环境，K8S 通过 Node Agent（如 Fluentd）收集 stdout 和 stderr 日志，无需文件输出和轮转
//
// 未来计划:
//   - 在 K8S 环境中，日志将由 Node Agent 收集并发送到集中式日志系统（如 Elasticsearch 或 Loki）
//   - 可根据需求扩展支持其他输出目标（如文件），但需确保与 K8S 日志收集机制兼容
type ZapConfig struct {
	Level       string `mapstructure:"level" yaml:"level"`               // 日志级别，支持 "debug"、"info"、"warn"、"error"、"fatal"，控制日志详细程度
	Encoding    string `mapstructure:"encoding" yaml:"encoding"`         // 编码格式，支持 "json"（生产推荐）或 "console"（开发推荐）
	OutputPath  string `mapstructure:"output_path" yaml:"output_path"`   // 普通日志输出路径，仅支持 "stdout"，若为空则默认使用 stdout
	ErrorOutput string `mapstructure:"error_output" yaml:"error_output"` // 错误日志输出路径，仅支持 "stderr"，若为空则默认使用 stderr
}
