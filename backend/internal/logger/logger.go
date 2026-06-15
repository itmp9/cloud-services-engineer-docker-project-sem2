package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var Log = zap.NewNop()

func Setup(level string) {
	logLevel := zapcore.InfoLevel
	if level != "" {
		_ = logLevel.Set(level)
	}

	cfg := &zap.Config{
		Level:       zap.NewAtomicLevelAt(logLevel),
		Development: false,
		Sampling: &zap.SamplingConfig{
			Initial:    100,
			Thereafter: 100,
		},
		Encoding:         "json",
		EncoderConfig:    zap.NewProductionEncoderConfig(),
		OutputPaths:      []string{"stderr"},
		ErrorOutputPaths: []string{"stderr"},
	}
	logger, err := cfg.Build()
	if err == nil {
		Log = logger
	}
}
