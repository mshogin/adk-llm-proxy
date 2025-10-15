package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/mshogin/agents/internal/application/services"
)

// PipelineYAMLConfig represents the YAML structure for pipeline configuration.
type PipelineYAMLConfig struct {
	Pipeline PipelineSection `yaml:"pipeline"`
}

// PipelineSection defines the pipeline configuration section.
type PipelineSection struct {
	Mode   string        `yaml:"mode"`   // sequential, parallel, conditional
	Agents []AgentConfig `yaml:"agents"` // List of agent configurations
}

// AgentConfig defines configuration for a single agent.
type AgentConfig struct {
	ID         string        `yaml:"id"`
	Enabled    bool          `yaml:"enabled"`
	DependsOn  []string      `yaml:"depends_on,omitempty"`
	Timeout    string        `yaml:"timeout"`    // e.g., "5s", "30s"
	Retry      int           `yaml:"retry"`      // Number of retries
	Conditions []string      `yaml:"conditions,omitempty"` // For conditional mode
}

// LoadPipelineConfig loads pipeline configuration from a YAML file.
func LoadPipelineConfig(path string) (*services.PipelineConfig, error) {
	// Read file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read pipeline config: %w", err)
	}

	// Parse YAML
	var yamlConfig PipelineYAMLConfig
	if err := yaml.Unmarshal(data, &yamlConfig); err != nil {
		return nil, fmt.Errorf("failed to parse pipeline config: %w", err)
	}

	// Convert to PipelineConfig
	config, err := convertToPipelineConfig(&yamlConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to convert pipeline config: %w", err)
	}

	// Validate
	if err := validatePipelineConfig(config); err != nil {
		return nil, fmt.Errorf("invalid pipeline config: %w", err)
	}

	return config, nil
}

// convertToPipelineConfig converts YAML config to PipelineConfig.
func convertToPipelineConfig(yamlConfig *PipelineYAMLConfig) (*services.PipelineConfig, error) {
	// Parse execution mode
	mode, err := parseExecutionMode(yamlConfig.Pipeline.Mode)
	if err != nil {
		return nil, err
	}

	// Convert agent configs
	agentConfigs := make([]services.AgentConfig, 0, len(yamlConfig.Pipeline.Agents))
	for _, agentYAML := range yamlConfig.Pipeline.Agents {
		agentConfig, err := convertToAgentConfig(&agentYAML)
		if err != nil {
			return nil, fmt.Errorf("failed to convert agent config for %s: %w", agentYAML.ID, err)
		}
		agentConfigs = append(agentConfigs, *agentConfig)
	}

	return &services.PipelineConfig{
		Mode:    mode,
		Agents:  agentConfigs,
		Options: services.DefaultExecutionOptions(),
	}, nil
}

// parseExecutionMode parses the execution mode string.
func parseExecutionMode(mode string) (services.ExecutionMode, error) {
	switch mode {
	case "sequential":
		return services.SequentialMode, nil
	case "parallel":
		return services.ParallelMode, nil
	case "conditional":
		return services.ConditionalMode, nil
	default:
		return "", fmt.Errorf("unknown execution mode: %s (valid: sequential, parallel, conditional)", mode)
	}
}

// convertToAgentConfig converts YAML agent config to AgentConfig.
func convertToAgentConfig(agentYAML *AgentConfig) (*services.AgentConfig, error) {
	// Parse timeout
	timeout, err := parseTimeout(agentYAML.Timeout)
	if err != nil {
		return nil, fmt.Errorf("invalid timeout for agent %s: %w", agentYAML.ID, err)
	}

	return &services.AgentConfig{
		ID:         agentYAML.ID,
		Enabled:    agentYAML.Enabled,
		DependsOn:  agentYAML.DependsOn,
		Timeout:    int(timeout.Milliseconds()),
		Retry:      agentYAML.Retry,
		Conditions: agentYAML.Conditions,
	}, nil
}

// parseTimeout parses a timeout string (e.g., "5s", "30s", "1m").
func parseTimeout(timeoutStr string) (time.Duration, error) {
	if timeoutStr == "" {
		return 30 * time.Second, nil // Default 30 seconds
	}

	duration, err := time.ParseDuration(timeoutStr)
	if err != nil {
		return 0, fmt.Errorf("invalid timeout format: %s (examples: 5s, 30s, 1m)", timeoutStr)
	}

	if duration <= 0 {
		return 0, fmt.Errorf("timeout must be positive: %s", timeoutStr)
	}

	return duration, nil
}

// validatePipelineConfig validates the pipeline configuration.
func validatePipelineConfig(config *services.PipelineConfig) error {
	// Validate mode
	if config.Mode != services.SequentialMode &&
		config.Mode != services.ParallelMode &&
		config.Mode != services.ConditionalMode {
		return fmt.Errorf("invalid execution mode: %s", config.Mode)
	}

	// Validate agents
	if len(config.Agents) == 0 {
		return fmt.Errorf("no agents configured")
	}

	// Check for duplicate agent IDs
	seen := make(map[string]bool)
	for _, agent := range config.Agents {
		if agent.ID == "" {
			return fmt.Errorf("agent ID cannot be empty")
		}
		if seen[agent.ID] {
			return fmt.Errorf("duplicate agent ID: %s", agent.ID)
		}
		seen[agent.ID] = true
	}

	// Validate dependencies
	for _, agent := range config.Agents {
		for _, dep := range agent.DependsOn {
			if !seen[dep] {
				return fmt.Errorf("agent %s depends on unknown agent: %s", agent.ID, dep)
			}
			if dep == agent.ID {
				return fmt.Errorf("agent %s cannot depend on itself", agent.ID)
			}
		}
	}

	// Check for circular dependencies
	if err := checkCircularDependencies(config.Agents); err != nil {
		return err
	}

	return nil
}

// checkCircularDependencies detects circular dependencies in the agent graph.
func checkCircularDependencies(agents []services.AgentConfig) error {
	// Build dependency graph
	deps := make(map[string][]string)
	for _, agent := range agents {
		deps[agent.ID] = agent.DependsOn
	}

	// DFS to detect cycles
	visited := make(map[string]bool)
	recursionStack := make(map[string]bool)

	var hasCycle func(string) bool
	hasCycle = func(agentID string) bool {
		visited[agentID] = true
		recursionStack[agentID] = true

		for _, dep := range deps[agentID] {
			if !visited[dep] {
				if hasCycle(dep) {
					return true
				}
			} else if recursionStack[dep] {
				return true // Cycle detected
			}
		}

		recursionStack[agentID] = false
		return false
	}

	for _, agent := range agents {
		if !visited[agent.ID] {
			if hasCycle(agent.ID) {
				return fmt.Errorf("circular dependency detected involving agent: %s", agent.ID)
			}
		}
	}

	return nil
}

// SavePipelineConfig saves pipeline configuration to a YAML file.
func SavePipelineConfig(config *services.PipelineConfig, path string) error {
	// Convert to YAML config
	yamlConfig := convertFromPipelineConfig(config)

	// Marshal to YAML
	data, err := yaml.Marshal(yamlConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal pipeline config: %w", err)
	}

	// Write to file
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write pipeline config: %w", err)
	}

	return nil
}

// convertFromPipelineConfig converts PipelineConfig to YAML config.
func convertFromPipelineConfig(config *services.PipelineConfig) *PipelineYAMLConfig {
	agentYAMLs := make([]AgentConfig, 0, len(config.Agents))
	for _, agent := range config.Agents {
		agentYAMLs = append(agentYAMLs, AgentConfig{
			ID:         agent.ID,
			Enabled:    agent.Enabled,
			DependsOn:  agent.DependsOn,
			Timeout:    fmt.Sprintf("%dms", agent.Timeout),
			Retry:      agent.Retry,
			Conditions: agent.Conditions,
		})
	}

	return &PipelineYAMLConfig{
		Pipeline: PipelineSection{
			Mode:   string(config.Mode),
			Agents: agentYAMLs,
		},
	}
}

// CreateDefaultPipelineConfig creates a default pipeline configuration.
func CreateDefaultPipelineConfig() *services.PipelineConfig {
	return &services.PipelineConfig{
		Mode: services.SequentialMode,
		Agents: []services.AgentConfig{
			{
				ID:      "intent_detection",
				Enabled: true,
				Timeout: 5000, // 5 seconds
				Retry:   2,
			},
			{
				ID:        "reasoning_structure",
				Enabled:   true,
				DependsOn: []string{"intent_detection"},
				Timeout:   10000, // 10 seconds
				Retry:     2,
			},
			{
				ID:        "retrieval_planner",
				Enabled:   true,
				DependsOn: []string{"reasoning_structure"},
				Timeout:   5000,
				Retry:     2,
			},
			{
				ID:        "context_synthesizer",
				Enabled:   true,
				DependsOn: []string{"retrieval_planner"},
				Timeout:   15000, // 15 seconds
				Retry:     2,
			},
			{
				ID:        "inference",
				Enabled:   true,
				DependsOn: []string{"context_synthesizer", "reasoning_structure"},
				Timeout:   20000, // 20 seconds
				Retry:     1,
			},
			{
				ID:        "validation",
				Enabled:   true,
				DependsOn: []string{"inference"},
				Timeout:   5000,
				Retry:     1,
			},
			{
				ID:        "summarization",
				Enabled:   true,
				DependsOn: []string{"validation"},
				Timeout:   10000,
				Retry:     1,
			},
		},
		Options: services.DefaultExecutionOptions(),
	}
}
