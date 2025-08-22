package observability

import (
	"bytes"
	"encoding/json"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/leslieo2/go-spec-mock/internal/config"
)

func TestNewLogger(t *testing.T) {
	tests := []struct {
		name    string
		config  config.LoggingConfig
		wantErr bool
	}{
		{
			name: "default configuration",
			config: config.LoggingConfig{
				Level:       "info",
				Format:      "json",
				Output:      "stdout",
				Development: false,
			},
			wantErr: false,
		},
		{
			name: "development mode",
			config: config.LoggingConfig{
				Level:       "debug",
				Format:      "console",
				Output:      "stdout",
				Development: true,
			},
			wantErr: false,
		},
		{
			name: "invalid log level",
			config: config.LoggingConfig{
				Level:       "invalid",
				Format:      "json",
				Output:      "stdout",
				Development: false,
			},
			wantErr: false, // Should default to info level
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, err := NewLogger(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewLogger() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && logger == nil {
				t.Error("NewLogger() returned nil logger")
			}
			if logger != nil {
				_ = logger.Sync()
			}
		})
	}
}

func TestLogger_JSONOutput(t *testing.T) {
	var buf bytes.Buffer

	config := zap.NewProductionConfig()
	config.OutputPaths = []string{"stdout"}
	config.EncoderConfig.TimeKey = "timestamp"
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(config.EncoderConfig),
		zapcore.AddSync(&buf),
		zapcore.InfoLevel,
	)

	logger := &Logger{zap.New(core)}
	defer func() { _ = logger.Sync() }()

	logger.Info("test message", zap.String("key", "value"))

	var logEntry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
		t.Fatalf("Failed to unmarshal log entry: %v", err)
	}

	if logEntry["msg"] != "test message" {
		t.Errorf("Expected message 'test message', got %v", logEntry["msg"])
	}

	if logEntry["key"] != "value" {
		t.Errorf("Expected key 'value', got %v", logEntry["key"])
	}
}

func TestLogger_DefaultLogConfig(t *testing.T) {
	cfg := config.DefaultLoggingConfig()

	expected := config.LoggingConfig{
		Level:       "info",
		Format:      "json",
		Output:      "stdout",
		Development: false,
	}

	if cfg != expected {
		t.Errorf("DefaultLoggingConfig() = %+v, want %+v", cfg, expected)
	}
}

func TestLogger_Sync(t *testing.T) {
	logger, err := NewLogger(config.DefaultLoggingConfig())
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	// Sync might fail with stderr in tests, but that's acceptable
	_ = logger.Sync()
}
