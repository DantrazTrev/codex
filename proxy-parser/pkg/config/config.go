package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config holds the proxy configuration
type Config struct {
	Port       int               `yaml:"port" json:"port"`
	Host       string            `yaml:"host" json:"host"`
	OutputFile string            `yaml:"output_file" json:"output_file"`
	Verbose    bool              `yaml:"verbose" json:"verbose"`
	Analyze    bool              `yaml:"analyze" json:"analyze"`
	GenAIOnly  bool              `yaml:"genai_only" json:"genai_only"`
	Proxy      ProxyConfig       `yaml:"proxy" json:"proxy"`
	Logging    LoggingConfig     `yaml:"logging" json:"logging"`
	Filtering  FilteringConfig   `yaml:"filtering" json:"filtering"`
	Analysis   AnalysisConfig    `yaml:"analysis" json:"analysis"`
}

// ProxyConfig holds proxy-specific settings
type ProxyConfig struct {
	Port int    `yaml:"port" json:"port"`
	Host string `yaml:"host" json:"host"`
}

// LoggingConfig holds logging settings
type LoggingConfig struct {
	Level  string `yaml:"level" json:"level"`
	Output string `yaml:"output" json:"output"`
	Format string `yaml:"format" json:"format"`
}

// FilteringConfig holds filtering settings
type FilteringConfig struct {
	IncludeEndpoints []string `yaml:"include_endpoints" json:"include_endpoints"`
	ExcludeEndpoints []string `yaml:"exclude_endpoints" json:"exclude_endpoints"`
	ExcludeHeaders   []string `yaml:"exclude_headers" json:"exclude_headers"`
}

// AnalysisConfig holds analysis settings
type AnalysisConfig struct {
	GenAIPatterns      []string `yaml:"genai_patterns" json:"genai_patterns"`
	HighlightKeywords  []string `yaml:"highlight_keywords" json:"highlight_keywords"`
}

// LoadFromFile loads configuration from a file
func (c *Config) LoadFromFile(filepath string) error {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	// Determine file format based on extension
	if strings.HasSuffix(filepath, ".json") {
		if err := json.Unmarshal(data, c); err != nil {
			return fmt.Errorf("failed to parse JSON config: %w", err)
		}
	} else if strings.HasSuffix(filepath, ".yaml") || strings.HasSuffix(filepath, ".yml") {
		if err := yaml.Unmarshal(data, c); err != nil {
			return fmt.Errorf("failed to parse YAML config: %w", err)
		}
	} else {
		// Try YAML first, then JSON
		if err := yaml.Unmarshal(data, c); err != nil {
			if err := json.Unmarshal(data, c); err != nil {
				return fmt.Errorf("failed to parse config (tried YAML and JSON): %w", err)
			}
		}
	}

	// Apply proxy config if present
	if c.Proxy.Port > 0 {
		c.Port = c.Proxy.Port
	}
	if c.Proxy.Host != "" {
		c.Host = c.Proxy.Host
	}

	// Apply logging config if present
	if c.Logging.Output != "" && c.OutputFile == "" {
		c.OutputFile = c.Logging.Output
	}

	return nil
}

// DefaultConfig returns default configuration
func DefaultConfig() *Config {
	return &Config{
		Port:       8080,
		Host:       "0.0.0.0",
		OutputFile: "",
		Verbose:    false,
		Analyze:    false,
		GenAIOnly:  false,
		Proxy: ProxyConfig{
			Port: 8080,
			Host: "0.0.0.0",
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "json",
		},
		Filtering: FilteringConfig{
			ExcludeHeaders: []string{"Cookie", "Set-Cookie"},
		},
		Analysis: AnalysisConfig{
			GenAIPatterns: []string{
				"gpt-4",
				"gpt-3.5",
				"claude",
				"completion",
				"chat/conversation",
			},
		},
	}
}