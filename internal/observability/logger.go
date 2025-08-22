package observability

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/leslieo2/go-spec-mock/internal/config"
)

type Logger struct {
	*zap.Logger
}

func NewLogger(config config.LoggingConfig) (*Logger, error) {
	var zapConfig zap.Config

	if config.Development {
		zapConfig = zap.NewDevelopmentConfig()
	} else {
		zapConfig = zap.NewProductionConfig()
	}

	// Set log level
	level, err := zapcore.ParseLevel(config.Level)
	if err != nil {
		level = zapcore.InfoLevel
	}
	zapConfig.Level = zap.NewAtomicLevelAt(level)

	// Set output format
	if config.Format == "json" {
		zapConfig.Encoding = "json"
	} else {
		zapConfig.Encoding = "console"
	}

	logger, err := zapConfig.Build()
	if err != nil {
		return nil, err
	}

	return &Logger{logger}, nil
}

func (l *Logger) Sync() error {
	return l.Logger.Sync()
}
