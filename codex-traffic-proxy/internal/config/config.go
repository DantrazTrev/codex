package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

type Config struct {
	Proxy   ProxyConfig   `mapstructure:"proxy"`
	Logger  LoggerConfig  `mapstructure:"logger"`
	Parser  ParserConfig  `mapstructure:"parser"`
	Storage StorageConfig `mapstructure:"storage"`
}

type ProxyConfig struct {
	ListenAddr string `mapstructure:"listen_addr"`
	Port       int    `mapstructure:"port"`
	Verbose    bool   `mapstructure:"verbose"`
}

type LoggerConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
	Output string `mapstructure:"output"`
}

type ParserConfig struct {
	Enabled         bool     `mapstructure:"enabled"`
	ExtractAPIKeys  bool     `mapstructure:"extract_api_keys"`
	ExtractEndpoints bool    `mapstructure:"extract_endpoints"`
	ExtractTokens   bool     `mapstructure:"extract_tokens"`
	SensitivePatterns []string `mapstructure:"sensitive_patterns"`
}

type StorageConfig struct {
	Directory string `mapstructure:"directory"`
	Retention int    `mapstructure:"retention_days"`
}

var defaultConfig = Config{
	Proxy: ProxyConfig{
		ListenAddr: "127.0.0.1",
		Port:       8080,
		Verbose:    false,
	},
	Logger: LoggerConfig{
		Level:  "info",
		Format: "json",
		Output: "stdout",
	},
	Parser: ParserConfig{
		Enabled:          true,
		ExtractAPIKeys:   true,
		ExtractEndpoints: true,
		ExtractTokens:    true,
		SensitivePatterns: []string{
			"sk-[a-zA-Z0-9]{32,}",
			"Bearer [a-zA-Z0-9\\-._~+/]+=*",
			"authorization: Bearer [a-zA-Z0-9\\-._~+/]+=*",
		},
	},
	Storage: StorageConfig{
		Directory: "./logs",
		Retention: 30,
	},
}

func LoadConfig() (*Config, error) {
	viper.SetConfigName(".codex-traffic-proxy")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("$HOME")
	viper.AddConfigPath(".")

	// Set default values
	setDefaults()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found, use defaults
			return &defaultConfig, nil
		}
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	// Override with environment variables
	config.Proxy.ListenAddr = getEnvOrDefault("PROXY_LISTEN_ADDR", config.Proxy.ListenAddr)
	config.Proxy.Port = getEnvIntOrDefault("PROXY_PORT", config.Proxy.Port)
	config.Proxy.Verbose = getEnvBoolOrDefault("PROXY_VERBOSE", config.Proxy.Verbose)
	config.Logger.Level = getEnvOrDefault("LOG_LEVEL", config.Logger.Level)
	config.Logger.Format = getEnvOrDefault("LOG_FORMAT", config.Logger.Format)
	config.Logger.Output = getEnvOrDefault("LOG_OUTPUT", config.Logger.Output)
	config.Storage.Directory = getEnvOrDefault("STORAGE_DIR", config.Storage.Directory)
	config.Storage.Retention = getEnvIntOrDefault("STORAGE_RETENTION", config.Storage.Retention)

	return &config, nil
}

func setDefaults() {
	viper.SetDefault("proxy.listen_addr", defaultConfig.Proxy.ListenAddr)
	viper.SetDefault("proxy.port", defaultConfig.Proxy.Port)
	viper.SetDefault("proxy.verbose", defaultConfig.Proxy.Verbose)
	viper.SetDefault("logger.level", defaultConfig.Logger.Level)
	viper.SetDefault("logger.format", defaultConfig.Logger.Format)
	viper.SetDefault("logger.output", defaultConfig.Logger.Output)
	viper.SetDefault("parser.enabled", defaultConfig.Parser.Enabled)
	viper.SetDefault("parser.extract_api_keys", defaultConfig.Parser.ExtractAPIKeys)
	viper.SetDefault("parser.extract_endpoints", defaultConfig.Parser.ExtractEndpoints)
	viper.SetDefault("parser.extract_tokens", defaultConfig.Parser.ExtractTokens)
	viper.SetDefault("parser.sensitive_patterns", defaultConfig.Parser.SensitivePatterns)
	viper.SetDefault("storage.directory", defaultConfig.Storage.Directory)
	viper.SetDefault("storage.retention_days", defaultConfig.Storage.Retention)
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvIntOrDefault(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := fmt.Sscanf(value, "%d", new(int)); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvBoolOrDefault(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		return value == "true" || value == "1" || value == "yes"
	}
	return defaultValue
}

func SaveDefaultConfig() error {
	// Create config directory if it doesn't exist
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	configDir := filepath.Join(homeDir, ".codex-traffic-proxy")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	configPath := filepath.Join(configDir, "config.yaml")

	file, err := os.Create(configPath)
	if err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}
	defer file.Close()

	// Write YAML config
	configContent := `# Codex Traffic Proxy Configuration
proxy:
  listen_addr: "127.0.0.1"
  port: 8080
  verbose: false

logger:
  level: "info"  # debug, info, warn, error
  format: "json"  # json, text
  output: "stdout"  # stdout, stderr, file

parser:
  enabled: true
  extract_api_keys: true
  extract_endpoints: true
  extract_tokens: true
  sensitive_patterns:
    - "sk-[a-zA-Z0-9]{32,}"
    - "Bearer [a-zA-Z0-9\\-._~+/]+=*"
    - "authorization: Bearer [a-zA-Z0-9\\-._~+/]+=*"

storage:
  directory: "./logs"
  retention_days: 30
`

	if _, err := file.WriteString(configContent); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}