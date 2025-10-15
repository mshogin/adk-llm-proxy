package models_test

import (
	"testing"

	"github.com/mshogin/agents/internal/domain/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContextValidator_RegisterAgent(t *testing.T) {
	validator := models.NewContextValidator()

	validator.RegisterAgent("intent_detection", []string{"reasoning", "diagnostics"})
	validator.RegisterAgent("inference", []string{"reasoning", "llm"})

	// Validate write permissions
	err := validator.ValidateWrite("intent_detection", "reasoning", "intents")
	assert.NoError(t, err)

	err = validator.ValidateWrite("intent_detection", "llm", "provider")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not allowed to write")
}

func TestContextValidator_ValidateWrite_UnregisteredAgent(t *testing.T) {
	validator := models.NewContextValidator()

	err := validator.ValidateWrite("unknown_agent", "reasoning", "intents")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "agent not registered")
}

func TestContextValidator_ValidateWrite_AllowedNamespace(t *testing.T) {
	validator := models.NewContextValidator()
	validator.RegisterAgent("intent_detection", []string{"reasoning", "diagnostics"})

	// Should allow write to reasoning
	err := validator.ValidateWrite("intent_detection", "reasoning", "intents")
	assert.NoError(t, err)

	// Should allow write to diagnostics
	err = validator.ValidateWrite("intent_detection", "diagnostics", "errors")
	assert.NoError(t, err)

	// Should not allow write to llm
	err = validator.ValidateWrite("intent_detection", "llm", "provider")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not allowed to write to namespace 'llm'")
}

func TestContextValidator_ValidateWrite_WildcardPermission(t *testing.T) {
	validator := models.NewContextValidator()
	validator.RegisterAgent("orchestrator", []string{"*"})

	// Should allow write to any namespace
	err := validator.ValidateWrite("orchestrator", "reasoning", "intents")
	assert.NoError(t, err)

	err = validator.ValidateWrite("orchestrator", "llm", "provider")
	assert.NoError(t, err)

	err = validator.ValidateWrite("orchestrator", "diagnostics", "errors")
	assert.NoError(t, err)
}

func TestContextValidator_SafeSet_Reasoning(t *testing.T) {
	validator := models.NewContextValidator()
	validator.RegisterAgent("intent_detection", []string{"reasoning"})

	ctx := models.NewAgentContext("session-1", "trace-1")

	// Set intents
	intents := []models.Intent{
		{Type: "query", Confidence: 0.95},
	}
	err := validator.SafeSet(ctx, "intent_detection", "reasoning", "intents", intents)
	require.NoError(t, err)
	assert.Len(t, ctx.Reasoning.Intents, 1)

	// Set summary
	err = validator.SafeSet(ctx, "intent_detection", "reasoning", "summary", "test summary")
	require.NoError(t, err)
	assert.Equal(t, "test summary", ctx.Reasoning.Summary)
}

func TestContextValidator_SafeSet_UnauthorizedNamespace(t *testing.T) {
	validator := models.NewContextValidator()
	validator.RegisterAgent("intent_detection", []string{"reasoning"})

	ctx := models.NewAgentContext("session-1", "trace-1")

	// Try to set LLM data (not allowed)
	err := validator.SafeSet(ctx, "intent_detection", "llm", "provider", "openai")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not allowed to write to namespace 'llm'")
}

func TestContextValidator_SafeSet_InvalidType(t *testing.T) {
	validator := models.NewContextValidator()
	validator.RegisterAgent("agent1", []string{"reasoning"})

	ctx := models.NewAgentContext("session-1", "trace-1")

	// Try to set intents with wrong type
	err := validator.SafeSet(ctx, "agent1", "reasoning", "intents", "not an array")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must be []Intent")
}

func TestContextValidator_SafeSet_Metadata(t *testing.T) {
	validator := models.NewContextValidator()
	validator.RegisterAgent("agent1", []string{"metadata"})

	ctx := models.NewAgentContext("session-1", "trace-1")

	// Set locale (allowed)
	err := validator.SafeSet(ctx, "agent1", "metadata", "locale", "en-US")
	require.NoError(t, err)
	assert.Equal(t, "en-US", ctx.Metadata.Locale)

	// Try to set session_id (not allowed - read-only)
	err = validator.SafeSet(ctx, "agent1", "metadata", "session_id", "new-session")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "read-only")
}

func TestContextValidator_SafeSet_Enrichment(t *testing.T) {
	validator := models.NewContextValidator()
	validator.RegisterAgent("context_synthesizer", []string{"enrichment"})

	ctx := models.NewAgentContext("session-1", "trace-1")

	// Set facts
	facts := []models.Fact{
		{ID: "f1", Content: "test fact", Source: "test"},
	}
	err := validator.SafeSet(ctx, "context_synthesizer", "enrichment", "facts", facts)
	require.NoError(t, err)
	assert.Len(t, ctx.Enrichment.Facts, 1)
}

func TestContextValidator_SafeSet_LLM(t *testing.T) {
	validator := models.NewContextValidator()
	validator.RegisterAgent("inference", []string{"llm"})

	ctx := models.NewAgentContext("session-1", "trace-1")

	// Set provider
	err := validator.SafeSet(ctx, "inference", "llm", "provider", "openai")
	require.NoError(t, err)
	assert.Equal(t, "openai", ctx.LLM.Provider)

	// Set model
	err = validator.SafeSet(ctx, "inference", "llm", "model", "gpt-4o-mini")
	require.NoError(t, err)
	assert.Equal(t, "gpt-4o-mini", ctx.LLM.Model)
}

func TestContextValidator_SafeSet_Diagnostics(t *testing.T) {
	validator := models.NewContextValidator()
	validator.RegisterAgent("validation", []string{"diagnostics"})

	ctx := models.NewAgentContext("session-1", "trace-1")

	// Set validation reports
	reports := []models.ValidationReport{
		{AgentID: "validation", Passed: true},
	}
	err := validator.SafeSet(ctx, "validation", "diagnostics", "validation_reports", reports)
	require.NoError(t, err)
	assert.Len(t, ctx.Diagnostics.ValidationReports, 1)
}

func TestContextValidator_SafeSet_UnknownNamespace(t *testing.T) {
	validator := models.NewContextValidator()
	validator.RegisterAgent("agent1", []string{"unknown"})

	ctx := models.NewAgentContext("session-1", "trace-1")

	err := validator.SafeSet(ctx, "agent1", "unknown", "key", "value")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown namespace 'unknown'")
}

func TestDefaultAgentPermissions(t *testing.T) {
	permissions := models.DefaultAgentPermissions()

	// Verify intent detection has correct permissions
	intentPerms := permissions["intent_detection"]
	assert.Contains(t, intentPerms, "reasoning")
	assert.Contains(t, intentPerms, "diagnostics")
	assert.NotContains(t, intentPerms, "llm")

	// Verify inference has LLM access
	inferencePerms := permissions["inference"]
	assert.Contains(t, inferencePerms, "reasoning")
	assert.Contains(t, inferencePerms, "llm")

	// Verify orchestrator has wildcard
	orchestratorPerms := permissions["orchestrator"]
	assert.Contains(t, orchestratorPerms, "*")
}

func TestContextValidator_ValidateRead(t *testing.T) {
	validator := models.NewContextValidator()
	validator.RegisterAgent("agent1", []string{"reasoning"})

	// Read access should always be allowed
	err := validator.ValidateRead("agent1", "llm", "provider")
	assert.NoError(t, err)

	err = validator.ValidateRead("agent1", "diagnostics", "errors")
	assert.NoError(t, err)
}
