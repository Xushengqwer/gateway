package core

import (
	"fmt"
	"os"

	"gateway/internal/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// ZapLogger 封装了 zap 的日志记录器，提供统一的日志接口
// - logger: 底层的 *zap.Logger 实例，用于执行实际的日志记录操作
type ZapLogger struct {
	logger *zap.Logger
}

// NewZapLogger 创建并初始化一个 ZapLogger 实例，用于日志记录
// 该函数根据提供的配置（ZapConfig）设置日志级别、编码格式、输出目标等，支持普通日志和错误日志的分离输出
// 当前设计:
//   - 普通日志（低于 Error 级别）输出到 stdout，错误日志（Error 及以上级别）输出到 stderr
//   - 原因: 适配 K8S 环境，K8S 通过 Node Agent（如 Fluentd）收集 stdout 和 stderr 日志，无需文件输出和轮转
//
// 未来计划:
//   - 在 K8S 环境中，日志将由 Node Agent 收集并发送到集中式日志系统（如 Elasticsearch 或 Loki）
//   - 可根据需求扩展支持其他输出目标（如文件），但需确保与 K8S 日志收集机制兼容
//
// 参数:
//   - cfg: ZapConfig 结构体，包含日志级别、编码格式、输出路径等配置项
//
// 返回值:
//   - *ZapLogger: 初始化完成的 ZapLogger 实例
//   - error: 如果配置解析失败或输出目标无效，则返回具体错误信息
func NewZapLogger(cfg config.ZapConfig) (*ZapLogger, error) {
	// 解析日志级别，确保输入有效
	var level zapcore.Level
	if err := level.UnmarshalText([]byte(cfg.Level)); err != nil {
		return nil, fmt.Errorf("无效的日志级别 '%s'，支持的级别包括: debug, info, warn, error, fatal", cfg.Level)
	}

	// 定义日志编码配置，控制日志输出的结构和样式
	// 这些字段和编码器是硬编码的，确保日志格式在整个项目中一致
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",                         // 时间字段键名，格式为 ISO8601
		LevelKey:       "level",                        // 日志级别字段键名，大写显示
		NameKey:        "logger",                       // 日志记录器名称字段键名
		CallerKey:      "caller",                       // 调用者信息字段键名，简短显示
		MessageKey:     "msg",                          // 日志消息字段键名
		StacktraceKey:  "stacktrace",                   // 堆栈跟踪字段键名
		LineEnding:     zapcore.DefaultLineEnding,      // 默认行结束符
		EncodeLevel:    zapcore.CapitalLevelEncoder,    // 日志级别大写编码（如 INFO、ERROR）
		EncodeTime:     zapcore.ISO8601TimeEncoder,     // 时间格式为 ISO8601（如 2023-10-10T12:00:00Z）
		EncodeDuration: zapcore.SecondsDurationEncoder, // 持续时间以秒为单位
		EncodeCaller:   zapcore.ShortCallerEncoder,     // 调用者信息简化为文件和行号
	}

	// 根据配置选择编码器，支持 JSON 或 Console 格式
	// - JSON: 适合生产环境，便于日志收集和解析
	// - Console: 适合开发环境，便于直接阅读
	var encoder zapcore.Encoder
	if cfg.Encoding == "json" {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	}

	// 设置普通日志输出目标，强制使用 stdout
	// 如果配置了其他值，打印警告并忽略，确保适配 K8S
	if cfg.OutputPath != "" && cfg.OutputPath != "stdout" {
		fmt.Printf("警告: OutputPath 仅支持 'stdout'，当前值 '%s' 已被忽略\n", cfg.OutputPath)
	}
	regularWS := zapcore.AddSync(os.Stdout)

	// 设置错误日志输出目标，强制使用 stderr
	// 如果配置了其他值，打印警告并忽略，确保适配 K8S
	if cfg.ErrorOutput != "" && cfg.ErrorOutput != "stderr" {
		fmt.Printf("警告: ErrorOutput 仅支持 'stderr'，当前值 '%s' 已被忽略\n", cfg.ErrorOutput)
	}
	errorWS := zapcore.AddSync(os.Stderr)

	// 定义日志级别过滤器，用于分离普通日志和错误日志
	// - lowPriority: 过滤低于 Error 级别的日志（如 Debug、Info、Warn）
	lowPriority := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		return lvl >= level && lvl < zapcore.ErrorLevel
	})
	// - highPriority: 过滤 Error 及以上级别的日志（如 Error、Fatal）
	highPriority := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		return lvl >= zapcore.ErrorLevel
	})

	// 创建普通日志的 Core，负责编码和输出
	regularCore := zapcore.NewCore(
		encoder,
		zapcore.Lock(regularWS), // 加锁确保线程安全
		lowPriority,
	)

	// 创建错误日志的 Core，负责编码和输出
	errorCore := zapcore.NewCore(
		encoder,
		zapcore.Lock(errorWS), // 加锁确保线程安全
		highPriority,
	)

	// 使用 Tee 合并普通日志和错误日志的 Core
	core := zapcore.NewTee(regularCore, errorCore)

	// 构建底层的 zap.Logger 实例，添加调用者信息并跳过一层调用栈
	logger := zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))

	return &ZapLogger{logger: logger}, nil
}

// Debug 记录 Debug 级别的日志
// - msg: 日志消息
// - fields: 可选的附加字段，用于提供上下文信息
func (z *ZapLogger) Debug(msg string, fields ...zap.Field) {
	z.logger.Debug(msg, fields...)
}

// Info 记录 Info 级别的日志
// - msg: 日志消息
// - fields: 可选的附加字段，用于提供上下文信息
func (z *ZapLogger) Info(msg string, fields ...zap.Field) {
	z.logger.Info(msg, fields...)
}

// Warn 记录 Warn 级别的日志
// - msg: 日志消息
// - fields: 可选的附加字段，用于提供上下文信息
func (z *ZapLogger) Warn(msg string, fields ...zap.Field) {
	z.logger.Warn(msg, fields...)
}

// Error 记录 Error 级别的日志
// - msg: 日志消息
// - fields: 可选的附加字段，用于提供上下文信息
func (z *ZapLogger) Error(msg string, fields ...zap.Field) {
	z.logger.Error(msg, fields...)
}

// Fatal 记录 Fatal 级别的日志，并终止程序
// - msg: 日志消息
// - fields: 可选的附加字段，用于提供上下文信息
func (z *ZapLogger) Fatal(msg string, fields ...zap.Field) {
	z.logger.Fatal(msg, fields...)
}

// Logger 返回底层的 *zap.Logger 实例，便于高级用法
func (z *ZapLogger) Logger() *zap.Logger {
	return z.logger
}
