package models

import (
	"fmt"
	"strings"
)

// ContextViolationError is raised when an agent tries to violate namespace isolation.
type ContextViolationError struct {
	AgentID   string
	Namespace string
	Key       string
	Message   string
}

func (e *ContextViolationError) Error() string {
	return fmt.Sprintf("context violation by agent %s: %s (namespace: %s, key: %s)",
		e.AgentID, e.Message, e.Namespace, e.Key)
}

// ContextValidator validates namespace isolation and access patterns.
type ContextValidator struct {
	// Map of agent ID to allowed namespaces
	agentPermissions map[string][]string
}

// NewContextValidator creates a new context validator.
func NewContextValidator() *ContextValidator {
	return &ContextValidator{
		agentPermissions: make(map[string][]string),
	}
}

// RegisterAgent registers an agent with its allowed namespaces.
func (v *ContextValidator) RegisterAgent(agentID string, allowedNamespaces []string) {
	v.agentPermissions[agentID] = allowedNamespaces
}

// ValidateWrite validates that an agent can write to a specific namespace.
func (v *ContextValidator) ValidateWrite(agentID, namespace, key string) error {
	// Get allowed namespaces for this agent
	allowed, exists := v.agentPermissions[agentID]
	if !exists {
		return &ContextViolationError{
			AgentID:   agentID,
			Namespace: namespace,
			Key:       key,
			Message:   "agent not registered",
		}
	}

	// Check if namespace is allowed
	if !contains(allowed, namespace) && !contains(allowed, "*") {
		return &ContextViolationError{
			AgentID:   agentID,
			Namespace: namespace,
			Key:       key,
			Message:   fmt.Sprintf("agent not allowed to write to namespace '%s'", namespace),
		}
	}

	return nil
}

// ValidateRead validates that an agent can read from a specific namespace.
// By default, all agents can read from all namespaces.
func (v *ContextValidator) ValidateRead(agentID, namespace, key string) error {
	// Read access is generally unrestricted for visibility
	return nil
}

// SafeSet safely sets a value in the context after validation.
func (v *ContextValidator) SafeSet(ctx *AgentContext, agentID, namespace, key string, value interface{}) error {
	// Validate write permission
	if err := v.ValidateWrite(agentID, namespace, key); err != nil {
		return err
	}

	// Set value in appropriate namespace
	switch strings.ToLower(namespace) {
	case "metadata":
		return v.setMetadata(ctx, key, value)
	case "reasoning":
		return v.setReasoning(ctx, key, value)
	case "enrichment":
		return v.setEnrichment(ctx, key, value)
	case "retrieval":
		return v.setRetrieval(ctx, key, value)
	case "llm":
		return v.setLLM(ctx, key, value)
	case "diagnostics":
		return v.setDiagnostics(ctx, key, value)
	case "audit":
		return v.setAudit(ctx, key, value)
	default:
		return &ContextViolationError{
			AgentID:   agentID,
			Namespace: namespace,
			Key:       key,
			Message:   fmt.Sprintf("unknown namespace '%s'", namespace),
		}
	}
}

// setMetadata sets a value in the metadata namespace.
func (v *ContextValidator) setMetadata(ctx *AgentContext, key string, value interface{}) error {
	// Metadata is mostly read-only except locale
	switch key {
	case "locale":
		if str, ok := value.(string); ok {
			ctx.Metadata.Locale = str
			return nil
		}
		return fmt.Errorf("locale must be a string")
	default:
		return fmt.Errorf("metadata field '%s' is read-only", key)
	}
}

// setReasoning sets a value in the reasoning namespace.
func (v *ContextValidator) setReasoning(ctx *AgentContext, key string, value interface{}) error {
	switch key {
	case "intents":
		if intents, ok := value.([]Intent); ok {
			ctx.Reasoning.Intents = intents
			return nil
		}
		return fmt.Errorf("intents must be []Intent")
	case "entities":
		if entities, ok := value.(map[string]interface{}); ok {
			ctx.Reasoning.Entities = entities
			return nil
		}
		return fmt.Errorf("entities must be map[string]interface{}")
	case "hypotheses":
		if hypotheses, ok := value.([]Hypothesis); ok {
			ctx.Reasoning.Hypotheses = hypotheses
			return nil
		}
		return fmt.Errorf("hypotheses must be []Hypothesis")
	case "conclusions":
		if conclusions, ok := value.([]Conclusion); ok {
			ctx.Reasoning.Conclusions = conclusions
			return nil
		}
		return fmt.Errorf("conclusions must be []Conclusion")
	case "summary":
		if str, ok := value.(string); ok {
			ctx.Reasoning.Summary = str
			return nil
		}
		return fmt.Errorf("summary must be a string")
	default:
		return fmt.Errorf("unknown reasoning field '%s'", key)
	}
}

// setEnrichment sets a value in the enrichment namespace.
func (v *ContextValidator) setEnrichment(ctx *AgentContext, key string, value interface{}) error {
	switch key {
	case "facts":
		if facts, ok := value.([]Fact); ok {
			ctx.Enrichment.Facts = facts
			return nil
		}
		return fmt.Errorf("facts must be []Fact")
	case "derived_knowledge":
		if knowledge, ok := value.([]Knowledge); ok {
			ctx.Enrichment.DerivedKnowledge = knowledge
			return nil
		}
		return fmt.Errorf("derived_knowledge must be []Knowledge")
	case "relationships":
		if rels, ok := value.([]Relationship); ok {
			ctx.Enrichment.Relationships = rels
			return nil
		}
		return fmt.Errorf("relationships must be []Relationship")
	default:
		return fmt.Errorf("unknown enrichment field '%s'", key)
	}
}

// setRetrieval sets a value in the retrieval namespace.
func (v *ContextValidator) setRetrieval(ctx *AgentContext, key string, value interface{}) error {
	switch key {
	case "plans":
		if plans, ok := value.([]RetrievalPlan); ok {
			ctx.Retrieval.Plans = plans
			return nil
		}
		return fmt.Errorf("plans must be []RetrievalPlan")
	case "queries":
		if queries, ok := value.([]Query); ok {
			ctx.Retrieval.Queries = queries
			return nil
		}
		return fmt.Errorf("queries must be []Query")
	case "artifacts":
		if artifacts, ok := value.([]Artifact); ok {
			ctx.Retrieval.Artifacts = artifacts
			return nil
		}
		return fmt.Errorf("artifacts must be []Artifact")
	default:
		return fmt.Errorf("unknown retrieval field '%s'", key)
	}
}

// setLLM sets a value in the llm namespace.
func (v *ContextValidator) setLLM(ctx *AgentContext, key string, value interface{}) error {
	switch key {
	case "provider":
		if str, ok := value.(string); ok {
			ctx.LLM.Provider = str
			return nil
		}
		return fmt.Errorf("provider must be a string")
	case "model":
		if str, ok := value.(string); ok {
			ctx.LLM.Model = str
			return nil
		}
		return fmt.Errorf("model must be a string")
	case "usage":
		if usage, ok := value.(*LLMUsage); ok {
			ctx.LLM.Usage = usage
			return nil
		}
		return fmt.Errorf("usage must be *LLMUsage")
	case "decisions":
		if decisions, ok := value.([]LLMDecision); ok {
			ctx.LLM.Decisions = decisions
			return nil
		}
		return fmt.Errorf("decisions must be []LLMDecision")
	default:
		return fmt.Errorf("unknown llm field '%s'", key)
	}
}

// setDiagnostics sets a value in the diagnostics namespace.
func (v *ContextValidator) setDiagnostics(ctx *AgentContext, key string, value interface{}) error {
	switch key {
	case "errors":
		if errors, ok := value.([]ErrorReport); ok {
			ctx.Diagnostics.Errors = errors
			return nil
		}
		return fmt.Errorf("errors must be []ErrorReport")
	case "warnings":
		if warnings, ok := value.([]Warning); ok {
			ctx.Diagnostics.Warnings = warnings
			return nil
		}
		return fmt.Errorf("warnings must be []Warning")
	case "performance":
		if perf, ok := value.(*PerformanceData); ok {
			ctx.Diagnostics.Performance = perf
			return nil
		}
		return fmt.Errorf("performance must be *PerformanceData")
	case "validation_reports":
		if reports, ok := value.([]ValidationReport); ok {
			ctx.Diagnostics.ValidationReports = reports
			return nil
		}
		return fmt.Errorf("validation_reports must be []ValidationReport")
	default:
		return fmt.Errorf("unknown diagnostics field '%s'", key)
	}
}

// setAudit sets a value in the audit namespace.
func (v *ContextValidator) setAudit(ctx *AgentContext, key string, value interface{}) error {
	switch key {
	case "agent_runs":
		if runs, ok := value.([]AgentRun); ok {
			ctx.Audit.AgentRuns = runs
			return nil
		}
		return fmt.Errorf("agent_runs must be []AgentRun")
	case "diffs":
		if diffs, ok := value.([]ContextDiff); ok {
			ctx.Audit.Diffs = diffs
			return nil
		}
		return fmt.Errorf("diffs must be []ContextDiff")
	default:
		return fmt.Errorf("unknown audit field '%s'", key)
	}
}

// Helper function to check if a slice contains a string.
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// DefaultAgentPermissions returns standard permissions for common agent types.
func DefaultAgentPermissions() map[string][]string {
	return map[string][]string{
		"intent_detection":    {"reasoning", "diagnostics", "audit"},
		"reasoning_structure": {"reasoning", "diagnostics", "audit"},
		"retrieval_planner":   {"retrieval", "diagnostics", "audit"},
		"retrieval_executor":  {"retrieval", "diagnostics", "audit"},
		"context_synthesizer": {"enrichment", "diagnostics", "audit"},
		"inference":           {"reasoning", "enrichment", "llm", "diagnostics", "audit"},
		"validation":          {"diagnostics", "audit"},
		"summarization":       {"reasoning", "diagnostics", "audit"},
		"orchestrator":        {"*"}, // Orchestrator can write to all namespaces
	}
}
