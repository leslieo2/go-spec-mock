package observability

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Logger struct {
	*zap.Logger
}

type LogConfig struct {
	Level      string `json:"level" yaml:"level"`
	Format     string `json:"format" yaml:"format"`
	Output     string `json:"output" yaml:"output"`
	Development bool   `json:"development" yaml:"development"`
}

func NewLogger(config LogConfig) (*Logger, error) {
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

func DefaultLogConfig() LogConfig {
	return LogConfig{
		Level:       "info",
		Format:      "json",
		Output:      "stdout",
		Development: false,
	}
}