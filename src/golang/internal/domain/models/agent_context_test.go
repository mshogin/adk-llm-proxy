package models_test

import (
	"testing"
	"time"

	"github.com/mshogin/agents/internal/domain/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAgentContext(t *testing.T) {
	sessionID := "session-123"
	traceID := "trace-456"

	ctx := models.NewAgentContext(sessionID, traceID)

	assert.Equal(t, "1.0.0", ctx.Version)
	assert.Equal(t, sessionID, ctx.Metadata.SessionID)
	assert.Equal(t, traceID, ctx.Metadata.TraceID)
	assert.NotNil(t, ctx.Reasoning)
	assert.NotNil(t, ctx.Enrichment)
	assert.NotNil(t, ctx.Retrieval)
	assert.NotNil(t, ctx.LLM)
	assert.NotNil(t, ctx.Diagnostics)
	assert.NotNil(t, ctx.Audit)
}

func TestAgentContext_Clone(t *testing.T) {
	original := models.NewAgentContext("session-1", "trace-1")
	original.Reasoning.Summary = "test summary"
	original.LLM.Provider = "openai"

	clone, err := original.Clone()
	require.NoError(t, err)

	// Verify clone has same values
	assert.Equal(t, original.Metadata.SessionID, clone.Metadata.SessionID)
	assert.Equal(t, original.Reasoning.Summary, clone.Reasoning.Summary)
	assert.Equal(t, original.LLM.Provider, clone.LLM.Provider)

	// Verify modifying clone doesn't affect original
	clone.Reasoning.Summary = "modified"
	assert.NotEqual(t, original.Reasoning.Summary, clone.Reasoning.Summary)
}

func TestAgentContext_Serialize_Deserialize(t *testing.T) {
	original := models.NewAgentContext("session-1", "trace-1")
	original.Reasoning.Summary = "test summary"
	original.LLM.Provider = "openai"
	original.LLM.Model = "gpt-4o-mini"

	// Serialize
	data, err := original.Serialize()
	require.NoError(t, err)
	assert.NotEmpty(t, data)

	// Deserialize
	restored, err := models.Deserialize(data)
	require.NoError(t, err)

	assert.Equal(t, original.Metadata.SessionID, restored.Metadata.SessionID)
	assert.Equal(t, original.Reasoning.Summary, restored.Reasoning.Summary)
	assert.Equal(t, original.LLM.Provider, restored.LLM.Provider)
	assert.Equal(t, original.LLM.Model, restored.LLM.Model)
}

func TestAgentContext_ReasoningNamespace(t *testing.T) {
	ctx := models.NewAgentContext("session-1", "trace-1")

	// Add intents
	ctx.Reasoning.Intents = []models.Intent{
		{Type: "query", Confidence: 0.95},
		{Type: "analysis", Confidence: 0.88},
	}

	// Add hypotheses
	ctx.Reasoning.Hypotheses = []models.Hypothesis{
		{ID: "h1", Description: "User wants to analyze commits"},
	}

	// Add conclusions
	ctx.Reasoning.Conclusions = []models.Conclusion{
		{ID: "c1", Content: "Analysis complete", Confidence: 0.92},
	}

	assert.Len(t, ctx.Reasoning.Intents, 2)
	assert.Len(t, ctx.Reasoning.Hypotheses, 1)
	assert.Len(t, ctx.Reasoning.Conclusions, 1)
}

func TestAgentContext_EnrichmentNamespace(t *testing.T) {
	ctx := models.NewAgentContext("session-1", "trace-1")

	// Add facts
	ctx.Enrichment.Facts = []models.Fact{
		{
			ID:         "f1",
			Content:    "Commit abc123 by user1",
			Source:     "gitlab",
			Timestamp:  time.Now(),
			Confidence: 1.0,
		},
	}

	// Add derived knowledge
	ctx.Enrichment.DerivedKnowledge = []models.Knowledge{
		{ID: "k1", Content: "User1 is active contributor"},
	}

	// Add relationships
	ctx.Enrichment.Relationships = []models.Relationship{
		{From: "user1", To: "project-x", Type: "contributes_to"},
	}

	assert.Len(t, ctx.Enrichment.Facts, 1)
	assert.Len(t, ctx.Enrichment.DerivedKnowledge, 1)
	assert.Len(t, ctx.Enrichment.Relationships, 1)
}

func TestAgentContext_LLMNamespace(t *testing.T) {
	ctx := models.NewAgentContext("session-1", "trace-1")

	// Set LLM info
	ctx.LLM.Provider = "openai"
	ctx.LLM.Model = "gpt-4o-mini"

	// Add usage
	ctx.LLM.Usage = &models.LLMUsage{
		TotalTokens:      1250,
		PromptTokens:     800,
		CompletionTokens: 450,
		CostUSD:          0.0234,
		ByAgent: map[string]float64{
			"intent_detection": 0.0050,
			"inference":        0.0184,
		},
	}

	// Add decision
	ctx.LLM.Decisions = []models.LLMDecision{
		{
			Timestamp:  time.Now(),
			AgentID:    "inference",
			TaskType:   "medium_synthesis",
			Selected:   "gpt-4o",
			Reason:     "Task requires medium synthesis",
			Complexity: "medium",
		},
	}

	assert.Equal(t, "openai", ctx.LLM.Provider)
	assert.Equal(t, 1250, ctx.LLM.Usage.TotalTokens)
	assert.Len(t, ctx.LLM.Decisions, 1)
}

func TestAgentContext_DiagnosticsNamespace(t *testing.T) {
	ctx := models.NewAgentContext("session-1", "trace-1")

	// Add error
	ctx.Diagnostics.Errors = []models.ErrorReport{
		{
			Timestamp: time.Now(),
			AgentID:   "retrieval",
			Message:   "Failed to fetch data",
			Severity:  "high",
		},
	}

	// Add warning
	ctx.Diagnostics.Warnings = []models.Warning{
		{
			Timestamp: time.Now(),
			AgentID:   "inference",
			Message:   "Low confidence result",
		},
	}

	// Add performance data
	ctx.Diagnostics.Performance.TotalDurationMS = 2340
	ctx.Diagnostics.Performance.AgentMetrics = map[string]*models.AgentMetrics{
		"intent_detection": {
			DurationMS: 120,
			LLMCalls:   0,
			Status:     "success",
		},
		"inference": {
			DurationMS: 850,
			LLMCalls:   2,
			Status:     "success",
			Tokens:     1200,
			Cost:       0.024,
		},
	}

	assert.Len(t, ctx.Diagnostics.Errors, 1)
	assert.Len(t, ctx.Diagnostics.Warnings, 1)
	assert.Equal(t, int64(2340), ctx.Diagnostics.Performance.TotalDurationMS)
	assert.Len(t, ctx.Diagnostics.Performance.AgentMetrics, 2)
}

func TestAgentContext_AuditNamespace(t *testing.T) {
	ctx := models.NewAgentContext("session-1", "trace-1")

	// Add agent runs
	ctx.Audit.AgentRuns = []models.AgentRun{
		{
			Timestamp:  time.Now(),
			AgentID:    "intent_detection",
			Status:     "success",
			DurationMS: 120,
			KeysWritten: []string{"reasoning.intents", "reasoning.entities"},
		},
		{
			Timestamp:  time.Now(),
			AgentID:    "inference",
			Status:     "success",
			DurationMS: 850,
			KeysWritten: []string{"reasoning.conclusions"},
		},
	}

	// Add diffs
	ctx.Audit.Diffs = []models.ContextDiff{
		{
			Timestamp: time.Now(),
			AgentID:   "intent_detection",
			Changes: map[string]interface{}{
				"reasoning": map[string]interface{}{
					"intents_added": 2,
				},
			},
		},
	}

	assert.Len(t, ctx.Audit.AgentRuns, 2)
	assert.Len(t, ctx.Audit.Diffs, 1)
}
