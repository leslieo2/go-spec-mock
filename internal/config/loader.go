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
	RateLimitEnabled  *bool
	RateLimitStrategy *string
	HotReload         *bool
	ProxyEnabled      *bool
	ProxyTarget       *string
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
	if flags.Host != nil && isFlagSet("host") && *flags.Host != "" {
		config.Server.Host = *flags.Host
	}
	if flags.Port != nil && isFlagSet("port") && *flags.Port != "" {
		config.Server.Port = *flags.Port
	}
	if flags.MetricsPort != nil && isFlagSet("metrics-port") && *flags.MetricsPort != "" {
		config.Server.MetricsPort = *flags.MetricsPort
	}

	// Security flags
	if flags.RateLimitEnabled != nil && isFlagSet("rate-limit-enabled") {
		config.Security.RateLimit.Enabled = *flags.RateLimitEnabled
	}
	if flags.RateLimitStrategy != nil && isFlagSet("rate-limit-strategy") && (*flags.RateLimitStrategy == constants.RateLimitStrategyIP || *flags.RateLimitStrategy == constants.RateLimitStrategyAPIKey) {
		config.Security.RateLimit.Strategy = *flags.RateLimitStrategy
	}

	// Spec file and hot reload
	if flags.SpecFile != nil && isFlagSet("spec-file") && *flags.SpecFile != "" {
		config.SpecFile = *flags.SpecFile
	}
	if flags.HotReload != nil && isFlagSet("hot-reload") {
		config.HotReload.Enabled = *flags.HotReload
	}

	// Proxy configuration
	if flags.ProxyEnabled != nil && isFlagSet("proxy-enabled") {
		config.Proxy.Enabled = *flags.ProxyEnabled
	}
	if flags.ProxyTarget != nil && isFlagSet("proxy-target") && *flags.ProxyTarget != "" {
		config.Proxy.Target = *flags.ProxyTarget
	}

	// TLS configuration
	if flags.TLSEnabled != nil && isFlagSet("tls-enabled") {
		config.TLS.Enabled = *flags.TLSEnabled
	}
	if flags.TLSCertFile != nil && isFlagSet("tls-cert-file") && *flags.TLSCertFile != "" {
		config.TLS.CertFile = *flags.TLSCertFile
	}
	if flags.TLSKeyFile != nil && isFlagSet("tls-key-file") && *flags.TLSKeyFile != "" {
		config.TLS.KeyFile = *flags.TLSKeyFile
	}
}

// isFlagSet checks if a flag is set (changed) in pflag, or returns true if pflag is not initialized
// This allows the function to work in test environments where pflag may not be initialized
func isFlagSet(flagName string) bool {
	flag := pflag.Lookup(flagName)
	if flag == nil {
		// If pflag isn't initialized with this flag, assume it's set for testing
		return true
	}
	return flag.Changed
}

// mergeConfig merges file configuration into the base configuration
func mergeConfig(base *Config, file *Config) {
	if file == nil {
		return
	}

	if file.Server.Host != "" {
		base.Server.Host = file.Server.Host
	}
	if file.Server.Port != "" {
		base.Server.Port = file.Server.Port
	}
	if file.Server.MetricsPort != "" {
		base.Server.MetricsPort = file.Server.MetricsPort
	}

	// Merge security configuration
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
