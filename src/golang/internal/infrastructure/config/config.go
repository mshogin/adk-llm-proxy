package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the application configuration.
type Config struct {
	Server      ServerConfig              `yaml:"server"`
	Providers   map[string]ProviderConfig `yaml:"providers"`
	Workflows   WorkflowsConfig           `yaml:"workflows"`
	Advanced    AdvancedConfig            `yaml:"advanced"`
	Performance PerformanceConfig         `yaml:"performance"`
	Logging     LoggingConfig             `yaml:"logging"`
	Security    SecurityConfig            `yaml:"security"`
}

// ServerConfig contains HTTP server settings.
type ServerConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

// ProviderConfig contains LLM provider settings.
type ProviderConfig struct {
	APIKey     string        `yaml:"api_key"`
	BaseURL    string        `yaml:"base_url"`
	Enabled    bool          `yaml:"enabled"`
	Timeout    time.Duration `yaml:"timeout"`
	MaxRetries int           `yaml:"max_retries"`
}

// WorkflowsConfig contains workflow settings.
type WorkflowsConfig struct {
	Default string   `yaml:"default"`
	Enabled []string `yaml:"enabled"`
}

// AdvancedConfig contains advanced workflow settings.
type AdvancedConfig struct {
	ADKAgentPath      string        `yaml:"adk_agent_path"`
	OpenAIAPIKey      string        `yaml:"openai_api_key"`
	OpenAIModel       string        `yaml:"openai_model"`
	ADKTimeout        time.Duration `yaml:"adk_timeout"`
	OpenAITimeout     time.Duration `yaml:"openai_timeout"`
	ParallelExecution bool          `yaml:"parallel_execution"`
}

// PerformanceConfig contains performance tuning settings.
type PerformanceConfig struct {
	ReadTimeout       time.Duration `yaml:"read_timeout"`
	WriteTimeout      time.Duration `yaml:"write_timeout"`
	IdleTimeout       time.Duration `yaml:"idle_timeout"`
	MaxIdleConns      int           `yaml:"max_idle_conns"`
	MaxConnsPerHost   int           `yaml:"max_conns_per_host"`
	IdleConnTimeout   time.Duration `yaml:"idle_conn_timeout"`
	StreamBufferSize  int           `yaml:"stream_buffer_size"`
	FlushInterval     time.Duration `yaml:"flush_interval"`
}

// LoggingConfig contains logging settings.
type LoggingConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
	Output string `yaml:"output"`
}

// SecurityConfig contains security settings.
type SecurityConfig struct {
	RequireAuth        bool     `yaml:"require_auth"`
	APIKeys            []string `yaml:"api_keys"`
	CORSEnabled        bool     `yaml:"cors_enabled"`
	CORSOrigins        []string `yaml:"cors_origins"`
	RateLimitEnabled   bool     `yaml:"rate_limit_enabled"`
	RateLimitRequests  int      `yaml:"rate_limit_requests"`
	RateLimitWindow    string   `yaml:"rate_limit_window"`
}

// Load reads and parses the configuration file.
func Load(path string) (*Config, error) {
	// Read file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Expand environment variables
	expanded := expandEnvVars(string(data))

	// Parse YAML
	var cfg Config
	if err := yaml.Unmarshal([]byte(expanded), &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Set defaults
	cfg.setDefaults()

	return &cfg, nil
}

// Validate checks if the configuration is valid.
func (c *Config) Validate() error {
	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", c.Server.Port)
	}

	if c.Workflows.Default == "" {
		return fmt.Errorf("default workflow must be specified")
	}

	// Check that at least one provider is enabled
	hasEnabledProvider := false
	for _, provider := range c.Providers {
		if provider.Enabled {
			hasEnabledProvider = true
			break
		}
	}
	if !hasEnabledProvider {
		return fmt.Errorf("at least one provider must be enabled")
	}

	return nil
}

// setDefaults sets default values for optional fields.
func (c *Config) setDefaults() {
	// Server defaults
	if c.Server.Host == "" {
		c.Server.Host = "0.0.0.0"
	}
	if c.Server.Port == 0 {
		c.Server.Port = 8001
	}

	// Provider defaults
	for name, provider := range c.Providers {
		if provider.Timeout == 0 {
			provider.Timeout = 30 * time.Second
		}
		if provider.MaxRetries == 0 {
			provider.MaxRetries = 3
		}
		c.Providers[name] = provider
	}

	// Workflow defaults
	if c.Workflows.Default == "" {
		c.Workflows.Default = "basic"
	}
	if len(c.Workflows.Enabled) == 0 {
		c.Workflows.Enabled = []string{"default", "basic", "advanced"}
	}

	// Advanced defaults
	if c.Advanced.ADKAgentPath == "" {
		c.Advanced.ADKAgentPath = "workflows/python/adk_agent.py"
	}
	if c.Advanced.OpenAIModel == "" {
		c.Advanced.OpenAIModel = "gpt-4o-mini"
	}
	if c.Advanced.ADKTimeout == 0 {
		c.Advanced.ADKTimeout = 10 * time.Second
	}
	if c.Advanced.OpenAITimeout == 0 {
		c.Advanced.OpenAITimeout = 10 * time.Second
	}

	// Performance defaults
	if c.Performance.ReadTimeout == 0 {
		c.Performance.ReadTimeout = 30 * time.Second
	}
	if c.Performance.WriteTimeout == 0 {
		c.Performance.WriteTimeout = 30 * time.Second
	}
	if c.Performance.IdleTimeout == 0 {
		c.Performance.IdleTimeout = 60 * time.Second
	}
	if c.Performance.MaxIdleConns == 0 {
		c.Performance.MaxIdleConns = 100
	}
	if c.Performance.MaxConnsPerHost == 0 {
		c.Performance.MaxConnsPerHost = 10
	}
	if c.Performance.IdleConnTimeout == 0 {
		c.Performance.IdleConnTimeout = 90 * time.Second
	}
	if c.Performance.StreamBufferSize == 0 {
		c.Performance.StreamBufferSize = 10
	}
	if c.Performance.FlushInterval == 0 {
		c.Performance.FlushInterval = 100 * time.Millisecond
	}

	// Logging defaults
	if c.Logging.Level == "" {
		c.Logging.Level = "info"
	}
	if c.Logging.Format == "" {
		c.Logging.Format = "json"
	}
	if c.Logging.Output == "" {
		c.Logging.Output = "stdout"
	}
}

// expandEnvVars replaces ${VAR} and $VAR with environment variable values.
func expandEnvVars(s string) string {
	return os.Expand(s, func(key string) string {
		return os.Getenv(key)
	})
}

// GetProviderByModel returns the provider name for a given model.
// This maps model names to providers (e.g., "gpt-4o-mini" -> "openai").
func (c *Config) GetProviderByModel(model string) string {
	model = strings.ToLower(model)

	// OpenAI models
	if strings.HasPrefix(model, "gpt-") || strings.HasPrefix(model, "o1-") {
		return "openai"
	}

	// Anthropic models
	if strings.HasPrefix(model, "claude-") {
		return "anthropic"
	}

	// DeepSeek models
	if strings.HasPrefix(model, "deepseek-") {
		return "deepseek"
	}

	// Ollama models (default for unknown models)
	return "ollama"
}
