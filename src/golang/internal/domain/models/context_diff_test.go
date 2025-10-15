package models_test

import (
	"testing"

	"github.com/mshogin/agents/internal/domain/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDiffTracker(t *testing.T) {
	ctx := models.NewAgentContext("session-1", "trace-1")

	tracker, err := models.NewDiffTracker(ctx)
	require.NoError(t, err)
	assert.NotNil(t, tracker)
}

func TestDiffTracker_Capture_NoChanges(t *testing.T) {
	ctx := models.NewAgentContext("session-1", "trace-1")

	tracker, err := models.NewDiffTracker(ctx)
	require.NoError(t, err)

	// Capture with no changes
	diff, err := tracker.Capture("agent1", ctx)
	require.NoError(t, err)
	assert.NotNil(t, diff)
	assert.Equal(t, "agent1", diff.AgentID)
	assert.Empty(t, diff.Changes)
}

func TestDiffTracker_Capture_ReasoningChanges(t *testing.T) {
	ctx := models.NewAgentContext("session-1", "trace-1")

	tracker, err := models.NewDiffTracker(ctx)
	require.NoError(t, err)

	// Add intents
	ctx.Reasoning.Intents = []models.Intent{
		{Type: "query", Confidence: 0.95},
		{Type: "analysis", Confidence: 0.88},
	}

	// Add summary
	ctx.Reasoning.Summary = "test summary"

	// Capture changes
	diff, err := tracker.Capture("intent_detection", ctx)
	require.NoError(t, err)

	assert.Equal(t, "intent_detection", diff.AgentID)
	assert.NotEmpty(t, diff.Changes)

	// Check reasoning changes
	reasoningChanges, ok := diff.Changes["reasoning"]
	require.True(t, ok)

	changes := reasoningChanges.(map[string]interface{})
	assert.Equal(t, 2, changes["intents_added"])
	assert.Equal(t, true, changes["summary_updated"])
}

func TestDiffTracker_Capture_EnrichmentChanges(t *testing.T) {
	ctx := models.NewAgentContext("session-1", "trace-1")

	tracker, err := models.NewDiffTracker(ctx)
	require.NoError(t, err)

	// Add facts
	ctx.Enrichment.Facts = []models.Fact{
		{ID: "f1", Content: "fact 1", Source: "test"},
		{ID: "f2", Content: "fact 2", Source: "test"},
	}

	// Add relationships
	ctx.Enrichment.Relationships = []models.Relationship{
		{From: "a", To: "b", Type: "links_to"},
	}

	// Capture changes
	diff, err := tracker.Capture("context_synthesizer", ctx)
	require.NoError(t, err)

	enrichmentChanges, ok := diff.Changes["enrichment"]
	require.True(t, ok)

	changes := enrichmentChanges.(map[string]interface{})
	assert.Equal(t, 2, changes["facts_added"])
	assert.Equal(t, 1, changes["relationships_added"])
}

func TestDiffTracker_Capture_LLMChanges(t *testing.T) {
	ctx := models.NewAgentContext("session-1", "trace-1")
	ctx.LLM.Provider = "openai"
	ctx.LLM.Model = "gpt-4o-mini"

	tracker, err := models.NewDiffTracker(ctx)
	require.NoError(t, err)

	// Change provider and model
	ctx.LLM.Provider = "anthropic"
	ctx.LLM.Model = "claude-sonnet"

	// Add usage
	ctx.LLM.Usage.TotalTokens = 1250
	ctx.LLM.Usage.CostUSD = 0.0234

	// Capture changes
	diff, err := tracker.Capture("inference", ctx)
	require.NoError(t, err)

	llmChanges, ok := diff.Changes["llm"]
	require.True(t, ok)

	changes := llmChanges.(map[string]interface{})

	providerChange := changes["provider_changed"].(map[string]string)
	assert.Equal(t, "openai", providerChange["from"])
	assert.Equal(t, "anthropic", providerChange["to"])

	modelChange := changes["model_changed"].(map[string]string)
	assert.Equal(t, "gpt-4o-mini", modelChange["from"])
	assert.Equal(t, "claude-sonnet", modelChange["to"])

	assert.Equal(t, 1250, changes["tokens_added"])
	assert.Equal(t, 0.0234, changes["cost_added"])
}

func TestDiffTracker_Capture_DiagnosticsChanges(t *testing.T) {
	ctx := models.NewAgentContext("session-1", "trace-1")

	tracker, err := models.NewDiffTracker(ctx)
	require.NoError(t, err)

	// Add errors
	ctx.Diagnostics.Errors = []models.ErrorReport{
		{AgentID: "agent1", Message: "error 1", Severity: "high"},
	}

	// Add warnings
	ctx.Diagnostics.Warnings = []models.Warning{
		{AgentID: "agent2", Message: "warning 1"},
		{AgentID: "agent3", Message: "warning 2"},
	}

	// Capture changes
	diff, err := tracker.Capture("validation", ctx)
	require.NoError(t, err)

	diagnosticsChanges, ok := diff.Changes["diagnostics"]
	require.True(t, ok)

	changes := diagnosticsChanges.(map[string]interface{})
	assert.Equal(t, 1, changes["errors_added"])
	assert.Equal(t, 2, changes["warnings_added"])
}

func TestDiffTracker_Capture_MultipleAgents(t *testing.T) {
	ctx := models.NewAgentContext("session-1", "trace-1")

	tracker, err := models.NewDiffTracker(ctx)
	require.NoError(t, err)

	// Agent 1: Add intents
	ctx.Reasoning.Intents = []models.Intent{
		{Type: "query", Confidence: 0.95},
	}

	diff1, err := tracker.Capture("intent_detection", ctx)
	require.NoError(t, err)
	assert.NotEmpty(t, diff1.Changes["reasoning"])

	// Agent 2: Add facts
	ctx.Enrichment.Facts = []models.Fact{
		{ID: "f1", Content: "fact 1", Source: "test"},
	}

	diff2, err := tracker.Capture("context_synthesizer", ctx)
	require.NoError(t, err)
	assert.NotEmpty(t, diff2.Changes["enrichment"])

	// Verify each diff only contains changes from that agent
	_, hasReasoning := diff2.Changes["reasoning"]
	assert.False(t, hasReasoning) // diff2 shouldn't have reasoning changes
}

func TestContextDiff_Summary(t *testing.T) {
	diff := &models.ContextDiff{
		AgentID: "intent_detection",
		Changes: map[string]interface{}{
			"reasoning": map[string]interface{}{
				"intents_added": 2,
				"summary_updated": true,
			},
		},
	}

	summary := diff.Summary()
	assert.Contains(t, summary, "[intent_detection]")
	assert.Contains(t, summary, "Changes:")
	assert.Contains(t, summary, "reasoning")
}

func TestContextDiff_Summary_NoChanges(t *testing.T) {
	diff := &models.ContextDiff{
		AgentID: "agent1",
		Changes: map[string]interface{}{},
	}

	summary := diff.Summary()
	assert.Contains(t, summary, "[agent1]")
	assert.Contains(t, summary, "No changes")
}

func TestContextDiff_ToJSON(t *testing.T) {
	diff := &models.ContextDiff{
		AgentID: "agent1",
		Changes: map[string]interface{}{
			"reasoning": map[string]interface{}{
				"intents_added": 2,
			},
		},
	}

	jsonData, err := diff.ToJSON()
	require.NoError(t, err)
	assert.NotEmpty(t, jsonData)
	assert.Contains(t, string(jsonData), "agent1")
	assert.Contains(t, string(jsonData), "reasoning")
}

func TestMergeDiffs(t *testing.T) {
	diff1 := &models.ContextDiff{
		AgentID: "agent1",
		Changes: map[string]interface{}{
			"reasoning": map[string]interface{}{
				"intents_added": 2,
			},
		},
	}

	diff2 := &models.ContextDiff{
		AgentID: "agent2",
		Changes: map[string]interface{}{
			"enrichment": map[string]interface{}{
				"facts_added": 3,
			},
		},
	}

	merged := models.MergeDiffs([]*models.ContextDiff{diff1, diff2})
	require.NotNil(t, merged)

	assert.Equal(t, "merged", merged.AgentID)
	assert.Len(t, merged.Changes, 2)
	assert.NotEmpty(t, merged.Changes["reasoning"])
	assert.NotEmpty(t, merged.Changes["enrichment"])
}

func TestMergeDiffs_Empty(t *testing.T) {
	merged := models.MergeDiffs([]*models.ContextDiff{})
	assert.Nil(t, merged)
}

func TestMergeDiffs_SameNamespace(t *testing.T) {
	diff1 := &models.ContextDiff{
		AgentID: "agent1",
		Changes: map[string]interface{}{
			"reasoning": map[string]interface{}{
				"intents_added": 2,
			},
		},
	}

	diff2 := &models.ContextDiff{
		AgentID: "agent2",
		Changes: map[string]interface{}{
			"reasoning": map[string]interface{}{
				"hypotheses_added": 3,
			},
		},
	}

	merged := models.MergeDiffs([]*models.ContextDiff{diff1, diff2})
	require.NotNil(t, merged)

	reasoningChanges := merged.Changes["reasoning"].(map[string]interface{})
	assert.Equal(t, 2, reasoningChanges["intents_added"])
	assert.Equal(t, 3, reasoningChanges["hypotheses_added"])
}
