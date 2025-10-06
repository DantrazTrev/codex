package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

// Config represents the application configuration
type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Proxy    ProxyConfig    `yaml:"proxy"`
	Parser   ParserConfig   `yaml:"parser"`
	Logging  LoggingConfig  `yaml:"logging"`
	Output   OutputConfig   `yaml:"output"`
}

// ServerConfig holds server configuration
type ServerConfig struct {
	Port         int    `yaml:"port"`
	Host         string `yaml:"host"`
	ReadTimeout  int    `yaml:"read_timeout"`
	WriteTimeout int    `yaml:"write_timeout"`
}

// ProxyConfig holds proxy configuration
type ProxyConfig struct {
	TargetURL     string            `yaml:"target_url"`
	Timeout       int               `yaml:"timeout"`
	Headers       map[string]string `yaml:"headers"`
	SkipTLSVerify bool              `yaml:"skip_tls_verify"`
}

// ParserConfig holds parser configuration
type ParserConfig struct {
	Enabled        bool     `yaml:"enabled"`
	ParseGenAI     bool     `yaml:"parse_genai"`
	ParseVibeCoding bool    `yaml:"parse_vibe_coding"`
	Keywords       []string `yaml:"keywords"`
	MaxBodySize    int64    `yaml:"max_body_size"`
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
	File   string `yaml:"file"`
}

// OutputConfig holds output configuration
type OutputConfig struct {
	Format     string `yaml:"format"` // json, yaml, csv
	File       string `yaml:"file"`
	Console    bool   `yaml:"console"`
	MaxEntries int    `yaml:"max_entries"`
}

var v *viper.Viper

func init() {
	v = viper.New()
	v.SetConfigType("yaml")
}

func SetConfigFile(file string) {
	v.SetConfigFile(file)
}

func SetConfigName(name string) {
	v.SetConfigName(name)
}

func AddConfigPath(path string) {
	v.AddConfigPath(path)
}

func ReadInConfig() error {
	return v.ReadInConfig()
}

func Load() (*Config, error) {
	// Set defaults
	setDefaults()

	// Unmarshal into config struct
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Validate configuration
	if err := validateConfig(&cfg); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &cfg, nil
}

func setDefaults() {
	// Server defaults
	v.SetDefault("server.port", 8080)
	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.read_timeout", 30)
	v.SetDefault("server.write_timeout", 30)

	// Proxy defaults
	v.SetDefault("proxy.target_url", "https://api.openai.com")
	v.SetDefault("proxy.timeout", 30)
	v.SetDefault("proxy.skip_tls_verify", false)

	// Parser defaults
	v.SetDefault("parser.enabled", true)
	v.SetDefault("parser.parse_genai", true)
	v.SetDefault("parser.parse_vibe_coding", true)
	v.SetDefault("parser.max_body_size", 10485760) // 10MB

	// Logging defaults
	v.SetDefault("logging.level", "info")
	v.SetDefault("logging.format", "json")

	// Output defaults
	v.SetDefault("output.format", "json")
	v.SetDefault("output.console", true)
	v.SetDefault("output.max_entries", 1000)
}

func validateConfig(cfg *Config) error {
	if cfg.Server.Port <= 0 || cfg.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", cfg.Server.Port)
	}

	if cfg.Proxy.TargetURL == "" {
		return fmt.Errorf("proxy target URL is required")
	}

	if cfg.Parser.MaxBodySize <= 0 {
		return fmt.Errorf("parser max body size must be positive")
	}

	return nil
}

// SaveConfig saves the current configuration to a file
func SaveConfig(cfg *Config, filename string) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Ensure directory exists
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write file
	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// ConfigFileNotFoundError represents a configuration file not found error
type ConfigFileNotFoundError struct {
	Err error
}

func (e ConfigFileNotFoundError) Error() string {
	return e.Err.Error()
}