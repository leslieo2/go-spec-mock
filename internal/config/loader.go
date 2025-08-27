package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/leslieo2/go-spec-mock/internal/constants"
	"github.com/spf13/pflag"
	"gopkg.in/yaml.v3"
)

// LoadConfig loads configuration with precedence:
// 1. Explicit CLI flags (highest priority)
// 2. Environment variables
// 3. Configuration file values
// 4. CLI flag default values
// 5. Default configuration values (lowest priority)
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

	// Override with CLI flags (including defaults)
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
	HotReload         *bool
	HotReloadDebounce *time.Duration
	ProxyEnabled      *bool
	ProxyTarget       *string
	ProxyTimeout      *time.Duration
	TLSEnabled        *bool
	TLSCertFile       *string
	TLSKeyFile        *string
}

// loadFromFile loads configuration from a YAML or JSON file
func loadFromFile(filePath string) (*Config, error) {
	// Normalize path to absolute for consistency
	if !filepath.IsAbs(filePath) {
		absPath, err := filepath.Abs(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to get absolute path for %s: %w", filePath, err)
		}
		filePath = absPath
	}

	// Validate file path to prevent directory traversal
	if err := validateFilePath(filePath); err != nil {
		return nil, fmt.Errorf("invalid config file path %s: %w", filePath, err)
	}

	data, err := os.ReadFile(filePath) // #nosec G304 - file path validated by validateFilePath()
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
	if val := os.Getenv(constants.EnvHost); val != "" {
		config.Server.Host = val
	}
	if val := os.Getenv(constants.EnvPort); val != "" {
		config.Server.Port = val
	}
	if val := os.Getenv(constants.EnvMetricsPort); val != "" {
		config.Server.MetricsPort = val
	}
	if val := os.Getenv(constants.EnvReadTimeout); val != "" {
		if duration, err := time.ParseDuration(val); err == nil {
			config.Server.ReadTimeout = duration
		}
	}
	if val := os.Getenv(constants.EnvWriteTimeout); val != "" {
		if duration, err := time.ParseDuration(val); err == nil {
			config.Server.WriteTimeout = duration
		}
	}
	if val := os.Getenv(constants.EnvIdleTimeout); val != "" {
		if duration, err := time.ParseDuration(val); err == nil {
			config.Server.IdleTimeout = duration
		}
	}
	if val := os.Getenv(constants.EnvMaxRequestSize); val != "" {
		if size, err := strconv.ParseInt(val, 10, 64); err == nil {
			config.Server.MaxRequestSize = size
		}
	}
	if val := os.Getenv(constants.EnvShutdownTimeout); val != "" {
		if duration, err := time.ParseDuration(val); err == nil {
			config.Server.ShutdownTimeout = duration
		}
	}
	if val := os.Getenv(constants.EnvSpecFile); val != "" {
		config.SpecFile = val
	}
	if val := os.Getenv(constants.EnvHotReload); val != "" {
		if enabled, err := strconv.ParseBool(val); err == nil {
			config.HotReload.Enabled = enabled
		}
	}
	if val := os.Getenv(constants.EnvHotReloadDebounce); val != "" {
		if duration, err := time.ParseDuration(val); err == nil {
			config.HotReload.Debounce = duration
		}
	}
	if val := os.Getenv(constants.EnvProxyEnabled); val != "" {
		if enabled, err := strconv.ParseBool(val); err == nil {
			config.Proxy.Enabled = enabled
		}
	}
	if val := os.Getenv(constants.EnvProxyTarget); val != "" {
		config.Proxy.Target = val
	}
	if val := os.Getenv(constants.EnvProxyTimeout); val != "" {
		if duration, err := time.ParseDuration(val); err == nil {
			config.Proxy.Timeout = duration
		}
	}
	if val := os.Getenv(constants.EnvTLSEnabled); val != "" {
		if enabled, err := strconv.ParseBool(val); err == nil {
			config.TLS.Enabled = enabled
		}
	}
	if val := os.Getenv(constants.EnvTLSCertFile); val != "" {
		config.TLS.CertFile = val
	}
	if val := os.Getenv(constants.EnvTLSKeyFile); val != "" {
		config.TLS.KeyFile = val
	}
}

// overrideWithCLI overrides configuration with CLI flag values
// Only explicitly set CLI flags override other configuration sources
func overrideWithCLI(config *Config, flags *CLIFlags) {
	if flags == nil {
		return
	}

	// Server configuration
	if flags.Host != nil && pflag.Lookup("host") != nil && pflag.Lookup("host").Changed {
		config.Server.Host = *flags.Host
	}
	if flags.Port != nil && pflag.Lookup("port") != nil && pflag.Lookup("port").Changed {
		config.Server.Port = *flags.Port
	}
	if flags.MetricsPort != nil && pflag.Lookup("metrics-port") != nil && pflag.Lookup("metrics-port").Changed {
		config.Server.MetricsPort = *flags.MetricsPort
	}
	if flags.ReadTimeout != nil && pflag.Lookup("read-timeout") != nil && pflag.Lookup("read-timeout").Changed {
		config.Server.ReadTimeout = *flags.ReadTimeout
	}
	if flags.WriteTimeout != nil && pflag.Lookup("write-timeout") != nil && pflag.Lookup("write-timeout").Changed {
		config.Server.WriteTimeout = *flags.WriteTimeout
	}
	if flags.IdleTimeout != nil && pflag.Lookup("idle-timeout") != nil && pflag.Lookup("idle-timeout").Changed {
		config.Server.IdleTimeout = *flags.IdleTimeout
	}
	if flags.MaxRequestSize != nil && pflag.Lookup("max-request-size") != nil && pflag.Lookup("max-request-size").Changed {
		config.Server.MaxRequestSize = *flags.MaxRequestSize
	}
	if flags.ShutdownTimeout != nil && pflag.Lookup("shutdown-timeout") != nil && pflag.Lookup("shutdown-timeout").Changed {
		config.Server.ShutdownTimeout = *flags.ShutdownTimeout
	}

	// Security flags
	if flags.AuthEnabled != nil && pflag.Lookup("auth-enabled") != nil && pflag.Lookup("auth-enabled").Changed {
		config.Security.Auth.Enabled = *flags.AuthEnabled
	}
	if flags.RateLimitEnabled != nil && pflag.Lookup("rate-limit-enabled") != nil && pflag.Lookup("rate-limit-enabled").Changed {
		config.Security.RateLimit.Enabled = *flags.RateLimitEnabled
	}
	if flags.RateLimitStrategy != nil && pflag.Lookup("rate-limit-strategy") != nil && pflag.Lookup("rate-limit-strategy").Changed {
		config.Security.RateLimit.Strategy = *flags.RateLimitStrategy
	}
	if flags.RateLimitRPS != nil && pflag.Lookup("rate-limit-rps") != nil && pflag.Lookup("rate-limit-rps").Changed {
		if config.Security.RateLimit.Global == nil {
			config.Security.RateLimit.Global = &GlobalRateLimit{
				RequestsPerSecond: *flags.RateLimitRPS,
				BurstSize:         200,
				WindowSize:        time.Minute,
			}
		} else {
			config.Security.RateLimit.Global.RequestsPerSecond = *flags.RateLimitRPS
		}
	}

	// Spec file and hot reload
	if flags.SpecFile != nil && pflag.Lookup("spec-file") != nil && pflag.Lookup("spec-file").Changed {
		config.SpecFile = *flags.SpecFile
	}
	if flags.HotReload != nil && pflag.Lookup("hot-reload") != nil && pflag.Lookup("hot-reload").Changed {
		config.HotReload.Enabled = *flags.HotReload
	}
	if flags.HotReloadDebounce != nil && pflag.Lookup("hot-reload-debounce") != nil && pflag.Lookup("hot-reload-debounce").Changed {
		config.HotReload.Debounce = *flags.HotReloadDebounce
	}

	// Proxy configuration
	if flags.ProxyEnabled != nil && pflag.Lookup("proxy-enabled") != nil && pflag.Lookup("proxy-enabled").Changed {
		config.Proxy.Enabled = *flags.ProxyEnabled
	}
	if flags.ProxyTarget != nil && pflag.Lookup("proxy-target") != nil && pflag.Lookup("proxy-target").Changed {
		config.Proxy.Target = *flags.ProxyTarget
	}
	if flags.ProxyTimeout != nil && pflag.Lookup("proxy-timeout") != nil && pflag.Lookup("proxy-timeout").Changed {
		config.Proxy.Timeout = *flags.ProxyTimeout
	}

	// TLS configuration
	if flags.TLSEnabled != nil && pflag.Lookup("tls-enabled") != nil && pflag.Lookup("tls-enabled").Changed {
		config.TLS.Enabled = *flags.TLSEnabled
	}
	if flags.TLSCertFile != nil && pflag.Lookup("tls-cert-file") != nil && pflag.Lookup("tls-cert-file").Changed {
		config.TLS.CertFile = *flags.TLSCertFile
	}
	if flags.TLSKeyFile != nil && pflag.Lookup("tls-key-file") != nil && pflag.Lookup("tls-key-file").Changed {
		config.TLS.KeyFile = *flags.TLSKeyFile
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

	// Merge hot reload configuration
	if file.HotReload.Enabled != base.HotReload.Enabled {
		base.HotReload.Enabled = file.HotReload.Enabled
	}
	if file.HotReload.Debounce > 0 {
		base.HotReload.Debounce = file.HotReload.Debounce
	}

	// Merge TLS configuration
	if file.TLS.Enabled != base.TLS.Enabled {
		base.TLS.Enabled = file.TLS.Enabled
	}
	if file.TLS.CertFile != "" {
		base.TLS.CertFile = file.TLS.CertFile
	}
	if file.TLS.KeyFile != "" {
		base.TLS.KeyFile = file.TLS.KeyFile
	}
}

// validateFilePath checks if the file path is safe to read
// Prevents directory traversal attacks and ensures the file is within expected locations
func validateFilePath(filePath string) error {
	// Get absolute path and clean it
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Clean the path to remove any .. or . components
	cleanPath := filepath.Clean(absPath)

	// Ensure the path doesn't contain any suspicious patterns
	if strings.Contains(cleanPath, "..") {
		return fmt.Errorf("path contains directory traversal attempts")
	}

	return nil
}
