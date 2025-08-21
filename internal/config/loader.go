package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/leslieo2/go-spec-mock/internal/security"
	"gopkg.in/yaml.v3"
)

// LoadConfig loads configuration with precedence:
// 1. CLI flags (override everything)
// 2. Environment variables
// 3. Configuration file values
// 4. Default values
func LoadConfig(configFile string, cliFlags *CLIFlags) (*Config, error) {
	// Start with default configuration
	config := DefaultConfig()

	// Load from configuration file if provided
	if configFile != "" {
		fileConfig, err := loadFromFile(configFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load config file: %w", err)
		}
		mergeConfig(config, fileConfig)
	}

	// Load from environment variables
	loadFromEnv(config)

	// Override with CLI flags
	if cliFlags != nil {
		overrideWithCLI(config, cliFlags)
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return config, nil
}

// CLIFlags contains CLI flag values that can override configuration
// This struct is used to pass CLI flag values without using the flag package directly
type CLIFlags struct {
	Host              *string
	Port              *string
	MetricsPort       *string
	SpecFile          *string
	ReadTimeout       *time.Duration
	WriteTimeout      *time.Duration
	IdleTimeout       *time.Duration
	MaxRequestSize    *int64
	ShutdownTimeout   *time.Duration
	AuthEnabled       *bool
	RateLimitEnabled  *bool
	RateLimitStrategy *string
	RateLimitRPS      *int
	GenerateKey       *string
}

// loadFromFile loads configuration from a YAML or JSON file
func loadFromFile(filePath string) (*Config, error) {
	// Normalize path
	if !filepath.IsAbs(filePath) {
		filePath = filepath.Clean(filePath)
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", filePath, err)
	}

	config := &Config{}
	ext := filepath.Ext(filePath)
	switch strings.ToLower(ext) {
	case ".yaml", ".yml":
		err = yaml.Unmarshal(data, config)
	case ".json":
		err = json.Unmarshal(data, config)
	default:
		return nil, fmt.Errorf("unsupported config file format: %s", ext)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to parse config file %s: %w", filePath, err)
	}

	return config, nil
}

// loadFromEnv loads configuration from environment variables
func loadFromEnv(config *Config) {
	// Server configuration
	if val := os.Getenv("GO_SPEC_MOCK_HOST"); val != "" {
		config.Server.Host = val
	}
	if val := os.Getenv("GO_SPEC_MOCK_PORT"); val != "" {
		config.Server.Port = val
	}
	if val := os.Getenv("GO_SPEC_MOCK_METRICS_PORT"); val != "" {
		config.Server.MetricsPort = val
	}
	if val := os.Getenv("GO_SPEC_MOCK_READ_TIMEOUT"); val != "" {
		if duration, err := time.ParseDuration(val); err == nil {
			config.Server.ReadTimeout = duration
		}
	}
	if val := os.Getenv("GO_SPEC_MOCK_WRITE_TIMEOUT"); val != "" {
		if duration, err := time.ParseDuration(val); err == nil {
			config.Server.WriteTimeout = duration
		}
	}
	if val := os.Getenv("GO_SPEC_MOCK_IDLE_TIMEOUT"); val != "" {
		if duration, err := time.ParseDuration(val); err == nil {
			config.Server.IdleTimeout = duration
		}
	}
	if val := os.Getenv("GO_SPEC_MOCK_MAX_REQUEST_SIZE"); val != "" {
		if size, err := strconv.ParseInt(val, 10, 64); err == nil {
			config.Server.MaxRequestSize = size
		}
	}
	if val := os.Getenv("GO_SPEC_MOCK_SHUTDOWN_TIMEOUT"); val != "" {
		if duration, err := time.ParseDuration(val); err == nil {
			config.Server.ShutdownTimeout = duration
		}
	}
	if val := os.Getenv("GO_SPEC_MOCK_SPEC_FILE"); val != "" {
		config.SpecFile = val
	}
}

// overrideWithCLI overrides configuration with CLI flag values
func overrideWithCLI(config *Config, flags *CLIFlags) {
	if flags.Host != nil && *flags.Host != "" {
		config.Server.Host = *flags.Host
	}
	if flags.Port != nil && *flags.Port != "" {
		config.Server.Port = *flags.Port
	}
	if flags.MetricsPort != nil && *flags.MetricsPort != "" {
		config.Server.MetricsPort = *flags.MetricsPort
	}
	if flags.ReadTimeout != nil && *flags.ReadTimeout > 0 {
		config.Server.ReadTimeout = *flags.ReadTimeout
	}
	if flags.WriteTimeout != nil && *flags.WriteTimeout > 0 {
		config.Server.WriteTimeout = *flags.WriteTimeout
	}
	if flags.IdleTimeout != nil && *flags.IdleTimeout > 0 {
		config.Server.IdleTimeout = *flags.IdleTimeout
	}
	if flags.MaxRequestSize != nil && *flags.MaxRequestSize > 0 {
		config.Server.MaxRequestSize = *flags.MaxRequestSize
	}
	if flags.ShutdownTimeout != nil && *flags.ShutdownTimeout > 0 {
		config.Server.ShutdownTimeout = *flags.ShutdownTimeout
	}
	if flags.AuthEnabled != nil {
		config.Security.Auth.Enabled = *flags.AuthEnabled
	}
	// AuthConfig field was removed - security config is now part of main config file
	if flags.RateLimitEnabled != nil {
		config.Security.RateLimit.Enabled = *flags.RateLimitEnabled
	}
	if flags.RateLimitStrategy != nil && *flags.RateLimitStrategy != "" {
		config.Security.RateLimit.Strategy = *flags.RateLimitStrategy
	}
	if flags.RateLimitRPS != nil && *flags.RateLimitRPS > 0 {
		if config.Security.RateLimit.Global == nil {
			config.Security.RateLimit.Global = &security.GlobalRateLimit{
				RequestsPerSecond: *flags.RateLimitRPS,
				BurstSize:         200,
				WindowSize:        time.Minute,
			}
		} else {
			config.Security.RateLimit.Global.RequestsPerSecond = *flags.RateLimitRPS
		}
	}
	if flags.SpecFile != nil && *flags.SpecFile != "" {
		config.SpecFile = *flags.SpecFile
	}
}

// mergeConfig merges file configuration into the base configuration
func mergeConfig(base *Config, file *Config) {
	if file.Server.Host != "" {
		base.Server.Host = file.Server.Host
	}
	if file.Server.Port != "" {
		base.Server.Port = file.Server.Port
	}
	if file.Server.MetricsPort != "" {
		base.Server.MetricsPort = file.Server.MetricsPort
	}
	if file.Server.ReadTimeout > 0 {
		base.Server.ReadTimeout = file.Server.ReadTimeout
	}
	if file.Server.WriteTimeout > 0 {
		base.Server.WriteTimeout = file.Server.WriteTimeout
	}
	if file.Server.IdleTimeout > 0 {
		base.Server.IdleTimeout = file.Server.IdleTimeout
	}
	if file.Server.MaxRequestSize > 0 {
		base.Server.MaxRequestSize = file.Server.MaxRequestSize
	}
	if file.Server.ShutdownTimeout > 0 {
		base.Server.ShutdownTimeout = file.Server.ShutdownTimeout
	}

	// Merge security configuration
	if file.Security.Auth.Enabled != base.Security.Auth.Enabled {
		base.Security.Auth = file.Security.Auth
	}
	if file.Security.RateLimit.Enabled != base.Security.RateLimit.Enabled {
		base.Security.RateLimit = file.Security.RateLimit
	}

	// Merge observability configuration
	if file.Observability.Logging.Level != "" {
		base.Observability.Logging.Level = file.Observability.Logging.Level
	}
	if file.Observability.Logging.Format != "" {
		base.Observability.Logging.Format = file.Observability.Logging.Format
	}
	if file.Observability.Metrics.Enabled != base.Observability.Metrics.Enabled {
		base.Observability.Metrics = file.Observability.Metrics
	}
	if file.Observability.Metrics.Path != "" {
		base.Observability.Metrics.Path = file.Observability.Metrics.Path
	}
	if file.Observability.Tracing.Enabled != base.Observability.Tracing.Enabled {
		base.Observability.Tracing = file.Observability.Tracing
	}

	if file.SpecFile != "" {
		base.SpecFile = file.SpecFile
	}
}
