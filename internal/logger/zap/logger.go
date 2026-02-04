package logger

import (
	uzap "go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// New создает новый экземпляр логгера
// level: "debug", "info", "error"
func New(level string) (*uzap.Logger, error) {
	lvl, err := uzap.ParseAtomicLevel(level)
	if err != nil {
		return nil, err
	}

	cfg := uzap.Config{
		Level:       lvl,
		Development: level == "debug",
		Encoding:    "json", // Структурированные логи в JSON
		EncoderConfig: zapcore.EncoderConfig{
			TimeKey:        "ts",
			LevelKey:       "level",
			NameKey:        "logger",
			CallerKey:      "caller",
			FunctionKey:    zapcore.OmitKey,
			MessageKey:     "msg",
			StacktraceKey:  "stacktrace",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    zapcore.LowercaseLevelEncoder,
			EncodeTime:     zapcore.ISO8601TimeEncoder,
			EncodeDuration: zapcore.SecondsDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		},
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}

	return cfg.Build()
}
