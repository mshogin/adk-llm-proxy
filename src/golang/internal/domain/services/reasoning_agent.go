package services

import (
	"context"

	"github.com/mshogin/agents/internal/domain/models"
)

// ReasoningAgent defines the interface for all reasoning agents.
// Each agent must implement this interface to participate in the reasoning pipeline.
//
// Design Principles:
// - Single Responsibility: Each agent does ONE thing well
// - Contract-based: Preconditions and postconditions define agent behavior
// - Context-driven: All data flows through AgentContext
// - Idempotent: Re-running with same context produces same result
type ReasoningAgent interface {
	// AgentID returns the unique identifier for this agent
	AgentID() string

	// Preconditions returns the list of context keys that must exist
	// before this agent can execute successfully.
	//
	// Examples:
	// - Intent Detection: [] (no preconditions, first agent)
	// - Reasoning Structure: ["reasoning.intents"] (needs intents first)
	// - Inference: ["reasoning.hypotheses", "enrichment.facts"]
	Preconditions() []string

	// Postconditions returns the list of context keys that this agent
	// guarantees to populate after successful execution.
	//
	// Examples:
	// - Intent Detection: ["reasoning.intents", "reasoning.entities"]
	// - Inference: ["reasoning.conclusions"]
	Postconditions() []string

	// Execute runs the agent's logic with the given context.
	// The agent reads from context based on its preconditions,
	// performs its reasoning, and writes to context based on postconditions.
	//
	// The agent MUST NOT modify the input context directly.
	// It should return a new/modified context with its changes.
	//
	// Returns:
	// - Updated AgentContext with agent's contributions
	// - Error if execution fails (context unchanged on error)
	Execute(ctx context.Context, agentContext *models.AgentContext) (*models.AgentContext, error)
}

// AgentMetadata provides additional information about an agent.
type AgentMetadata struct {
	ID           string   // Unique agent identifier
	Name         string   // Human-readable name
	Description  string   // Agent purpose
	Version      string   // Agent version
	Author       string   // Agent author
	Tags         []string // Categorization tags
	Dependencies []string // Other agents this depends on
}

// MetadataProvider is an optional interface for agents that provide metadata.
type MetadataProvider interface {
	GetMetadata() AgentMetadata
}

// AgentCapabilities defines optional capabilities an agent may support.
type AgentCapabilities struct {
	SupportsParallelExecution bool // Can run in parallel with other agents
	SupportsRetry             bool // Can be retried on failure
	RequiresLLM               bool // Requires LLM access
	IsDeterministic           bool // Same input always produces same output
	EstimatedDuration         int  // Estimated execution time in milliseconds
}

// CapabilitiesProvider is an optional interface for agents that declare capabilities.
type CapabilitiesProvider interface {
	GetCapabilities() AgentCapabilities
}

// AgentConfig provides configuration for agent execution.
type AgentConfig struct {
	Enabled        bool              // Whether agent is enabled
	Timeout        int               // Execution timeout in milliseconds
	RetryCount     int               // Number of retries on failure
	RetryDelay     int               // Delay between retries in milliseconds
	CustomSettings map[string]string // Agent-specific settings
}

// ConfigurableAgent is an optional interface for agents that can be configured.
type ConfigurableAgent interface {
	Configure(config AgentConfig) error
	GetConfig() AgentConfig
}

// AgentHealth represents agent health status.
type AgentHealth struct {
	Healthy      bool   // Overall health status
	Message      string // Health status message
	LastExecuted int64  // Last execution timestamp (Unix milliseconds)
	SuccessRate  float64 // Success rate (0.0-1.0)
}

// HealthCheckProvider is an optional interface for agents that support health checks.
type HealthCheckProvider interface {
	CheckHealth(ctx context.Context) AgentHealth
}

// ValidationResult represents the result of validating agent preconditions/postconditions.
type ValidationResult struct {
	Valid           bool     // Whether validation passed
	MissingKeys     []string // Required keys that are missing
	InvalidKeys     []string // Keys with invalid values
	ValidationError string   // Validation error message
}

// ContractValidator validates agent contracts (preconditions/postconditions).
type ContractValidator struct {
	agent ReasoningAgent
}

// NewContractValidator creates a new contract validator for an agent.
func NewContractValidator(agent ReasoningAgent) *ContractValidator {
	return &ContractValidator{agent: agent}
}

// ValidatePreconditions checks if all preconditions are satisfied in the context.
func (v *ContractValidator) ValidatePreconditions(ctx *models.AgentContext) ValidationResult {
	result := ValidationResult{
		Valid:       true,
		MissingKeys: []string{},
	}

	for _, key := range v.agent.Preconditions() {
		if !v.keyExists(ctx, key) {
			result.Valid = false
			result.MissingKeys = append(result.MissingKeys, key)
		}
	}

	if !result.Valid {
		result.ValidationError = "Missing required preconditions"
	}

	return result
}

// ValidatePostconditions checks if all postconditions were satisfied after execution.
func (v *ContractValidator) ValidatePostconditions(ctx *models.AgentContext) ValidationResult {
	result := ValidationResult{
		Valid:       true,
		MissingKeys: []string{},
	}

	for _, key := range v.agent.Postconditions() {
		if !v.keyExists(ctx, key) {
			result.Valid = false
			result.MissingKeys = append(result.MissingKeys, key)
		}
	}

	if !result.Valid {
		result.ValidationError = "Agent failed to satisfy postconditions"
	}

	return result
}

// keyExists checks if a context key exists and has a non-nil value.
func (v *ContractValidator) keyExists(ctx *models.AgentContext, key string) bool {
	// Parse key format: "namespace.field"
	// Examples: "reasoning.intents", "enrichment.facts", "llm.usage"

	// This is a simplified check - full implementation would parse the key
	// and check the specific field in the namespace.
	// For now, we check if the namespace itself is populated.

	// TODO: Implement full key path checking
	// For example: "reasoning.intents" should check len(ctx.Reasoning.Intents) > 0

	switch key {
	case "reasoning.intents":
		return len(ctx.Reasoning.Intents) > 0
	case "reasoning.entities":
		return len(ctx.Reasoning.Entities) > 0
	case "reasoning.hypotheses":
		return len(ctx.Reasoning.Hypotheses) > 0
	case "reasoning.conclusions":
		return len(ctx.Reasoning.Conclusions) > 0
	case "reasoning.summary":
		return ctx.Reasoning.Summary != ""
	case "enrichment.facts":
		return len(ctx.Enrichment.Facts) > 0
	case "enrichment.derived_knowledge":
		return len(ctx.Enrichment.DerivedKnowledge) > 0
	case "enrichment.relationships":
		return len(ctx.Enrichment.Relationships) > 0
	case "retrieval.plans":
		return len(ctx.Retrieval.Plans) > 0
	case "retrieval.queries":
		return len(ctx.Retrieval.Queries) > 0
	case "retrieval.artifacts":
		return len(ctx.Retrieval.Artifacts) > 0
	case "llm.usage":
		return ctx.LLM.Usage != nil && ctx.LLM.Usage.TotalTokens > 0
	case "llm.decisions":
		return len(ctx.LLM.Decisions) > 0
	default:
		// Unknown key - assume it doesn't exist
		return false
	}
}

// AgentExecutionOptions provides options for agent execution.
type AgentExecutionOptions struct {
	ValidateContract   bool // Validate pre/postconditions
	TrackPerformance   bool // Track execution time and metrics
	CaptureChanges     bool // Capture context diffs
	FailOnViolation    bool // Fail if contract is violated
	TimeoutMS          int  // Execution timeout in milliseconds
}

// DefaultExecutionOptions returns default execution options.
func DefaultExecutionOptions() AgentExecutionOptions {
	return AgentExecutionOptions{
		ValidateContract: true,
		TrackPerformance: true,
		CaptureChanges:   true,
		FailOnViolation:  true,
		TimeoutMS:        30000, // 30 seconds
	}
}
