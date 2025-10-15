package agents

import (
	"context"
	"fmt"
	"testing"

	"github.com/mshogin/agents/internal/domain/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test agent initialization
func TestNewInferenceAgent(t *testing.T) {
	agent := NewInferenceAgent()
	assert.NotNil(t, agent)
	assert.Equal(t, "inference", agent.AgentID())
}

// Test AgentID method
func TestInference_AgentID(t *testing.T) {
	agent := NewInferenceAgent()
	assert.Equal(t, "inference", agent.AgentID())
}

// Test Preconditions method
func TestInference_Preconditions(t *testing.T) {
	agent := NewInferenceAgent()
	preconditions := agent.Preconditions()

	assert.Len(t, preconditions, 3)
	assert.Contains(t, preconditions, "reasoning.intents")
	assert.Contains(t, preconditions, "reasoning.hypotheses")
	assert.Contains(t, preconditions, "enrichment.facts")
}

// Test Postconditions method
func TestInference_Postconditions(t *testing.T) {
	agent := NewInferenceAgent()
	postconditions := agent.Postconditions()

	assert.Len(t, postconditions, 3)
	assert.Contains(t, postconditions, "reasoning.conclusions")
	assert.Contains(t, postconditions, "reasoning.alternatives")
	assert.Contains(t, postconditions, "reasoning.inference_chain")
}

// Test GetMetadata method
func TestInference_GetMetadata(t *testing.T) {
	agent := NewInferenceAgent()
	metadata := agent.GetMetadata()

	assert.Equal(t, "inference", metadata.ID)
	assert.Equal(t, "Inference Agent", metadata.Name)
	assert.NotEmpty(t, metadata.Description)
	assert.Equal(t, "1.0.0", metadata.Version)
	assert.NotEmpty(t, metadata.Tags)
	assert.Contains(t, metadata.Dependencies, "context_synthesizer")
}

// Test GetCapabilities method
func TestInference_GetCapabilities(t *testing.T) {
	agent := NewInferenceAgent()
	caps := agent.GetCapabilities()

	assert.False(t, caps.SupportsParallelExecution)
	assert.True(t, caps.SupportsRetry)
	assert.False(t, caps.RequiresLLM)
	assert.True(t, caps.IsDeterministic)
	assert.Greater(t, caps.EstimatedDuration, 0)
}

// Test Execute with valid inputs
func TestInference_Execute_ValidInputs(t *testing.T) {
	agent := NewInferenceAgent()

	ctx := &models.AgentContext{
		Reasoning: &models.ReasoningContext{
			Intents: []models.Intent{
				{Type: "query_commits", Confidence: 0.9},
			},
			Hypotheses: []models.Hypothesis{
				{ID: "h0", Description: "Retrieve commit data"},
				{ID: "h1", Description: "Filter by date range"},
			},
		},
		Enrichment: &models.EnrichmentContext{
			Facts: []models.Fact{
				{ID: "f1", Content: "Commit message 1", Source: "gitlab", Confidence: 0.95},
				{ID: "f2", Content: "Commit message 2", Source: "gitlab", Confidence: 0.95},
			},
			DerivedKnowledge: []models.Knowledge{
				{ID: "k0", Content: "Aggregated 2 facts from gitlab"},
			},
		},
	}

	result, err := agent.Execute(context.Background(), ctx)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Reasoning)

	// Verify conclusions created
	assert.NotEmpty(t, result.Reasoning.Conclusions)

	// Verify inference chain created
	assert.NotEmpty(t, result.Reasoning.InferenceChain)

	// Verify alternatives created
	assert.NotNil(t, result.Reasoning.Alternatives)
}

// Test Execute with missing intents (precondition failure)
func TestInference_Execute_MissingIntents(t *testing.T) {
	agent := NewInferenceAgent()

	ctx := &models.AgentContext{
		Reasoning: &models.ReasoningContext{
			Intents: []models.Intent{}, // Empty
			Hypotheses: []models.Hypothesis{
				{ID: "h0", Description: "Test hypothesis"},
			},
		},
		Enrichment: &models.EnrichmentContext{
			Facts: []models.Fact{
				{ID: "f1", Content: "Test fact", Source: "gitlab"},
			},
		},
	}

	result, err := agent.Execute(context.Background(), ctx)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "no intents found")
}

// Test Execute with missing hypotheses (precondition failure)
func TestInference_Execute_MissingHypotheses(t *testing.T) {
	agent := NewInferenceAgent()

	ctx := &models.AgentContext{
		Reasoning: &models.ReasoningContext{
			Intents: []models.Intent{
				{Type: "query_commits", Confidence: 0.9},
			},
			Hypotheses: []models.Hypothesis{}, // Empty
		},
		Enrichment: &models.EnrichmentContext{
			Facts: []models.Fact{
				{ID: "f1", Content: "Test fact", Source: "gitlab"},
			},
		},
	}

	result, err := agent.Execute(context.Background(), ctx)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "no hypotheses found")
}

// Test Execute with missing facts (precondition failure)
func TestInference_Execute_MissingFacts(t *testing.T) {
	agent := NewInferenceAgent()

	ctx := &models.AgentContext{
		Reasoning: &models.ReasoningContext{
			Intents: []models.Intent{
				{Type: "query_commits", Confidence: 0.9},
			},
			Hypotheses: []models.Hypothesis{
				{ID: "h0", Description: "Test hypothesis"},
			},
		},
		Enrichment: &models.EnrichmentContext{
			Facts: []models.Fact{}, // Empty
		},
	}

	result, err := agent.Execute(context.Background(), ctx)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "no facts found")
}

// Test Execute with nil reasoning context
func TestInference_Execute_NilReasoningContext(t *testing.T) {
	agent := NewInferenceAgent()

	ctx := &models.AgentContext{
		Reasoning:  nil,
		Enrichment: &models.EnrichmentContext{},
	}

	result, err := agent.Execute(context.Background(), ctx)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "reasoning context is nil")
}

// Test makeCommitConclusions
func TestInference_MakeCommitConclusions(t *testing.T) {
	agent := NewInferenceAgent()

	intent := models.Intent{Type: "query_commits", Confidence: 0.9}
	facts := []models.Fact{
		{ID: "f1", Content: "Commit 1", Source: "gitlab"},
		{ID: "f2", Content: "Commit 2", Source: "gitlab"},
	}
	knowledge := []models.Knowledge{}
	chain := []models.InferenceStep{}
	conclusionID := 0

	conclusions := agent.makeCommitConclusions(intent, facts, knowledge, chain, &conclusionID)

	assert.Len(t, conclusions, 1)
	assert.Contains(t, conclusions[0].Description, "Found 2 commit(s)")
	assert.Equal(t, 0.95, conclusions[0].Confidence)
	assert.Equal(t, "query_commits", conclusions[0].Intent)
}

// Test makeIssueConclusions
func TestInference_MakeIssueConclusions(t *testing.T) {
	agent := NewInferenceAgent()

	intent := models.Intent{Type: "query_issues", Confidence: 0.85}
	facts := []models.Fact{
		{ID: "f1", Content: "Issue 1", Source: "youtrack"},
	}
	knowledge := []models.Knowledge{}
	chain := []models.InferenceStep{}
	conclusionID := 0

	conclusions := agent.makeIssueConclusions(intent, facts, knowledge, chain, &conclusionID)

	assert.Len(t, conclusions, 1)
	assert.Contains(t, conclusions[0].Description, "Found 1 issue(s)")
	assert.Equal(t, 0.95, conclusions[0].Confidence)
	assert.Equal(t, "query_issues", conclusions[0].Intent)
}

// Test makeAnalyticsConclusions
func TestInference_MakeAnalyticsConclusions(t *testing.T) {
	agent := NewInferenceAgent()

	intent := models.Intent{Type: "query_analytics", Confidence: 0.8}
	facts := []models.Fact{
		{ID: "f1", Content: "Metric 1", Source: "gitlab"},
		{ID: "f2", Content: "Metric 2", Source: "youtrack"},
	}
	knowledge := []models.Knowledge{}
	chain := []models.InferenceStep{}
	conclusionID := 0

	conclusions := agent.makeAnalyticsConclusions(intent, facts, knowledge, chain, &conclusionID)

	assert.Len(t, conclusions, 2) // One per source
	assert.Contains(t, conclusions[0].Description, "Aggregated")
	assert.Contains(t, conclusions[1].Description, "Aggregated")
}

// Test makeStatusConclusions
func TestInference_MakeStatusConclusions(t *testing.T) {
	agent := NewInferenceAgent()

	intent := models.Intent{Type: "query_status", Confidence: 0.75}
	facts := []models.Fact{
		{ID: "f1", Content: "Data 1", Source: "gitlab"},
		{ID: "f2", Content: "Data 2", Source: "youtrack"},
	}
	knowledge := []models.Knowledge{}
	chain := []models.InferenceStep{}
	conclusionID := 0

	conclusions := agent.makeStatusConclusions(intent, facts, knowledge, chain, &conclusionID)

	assert.Len(t, conclusions, 1)
	assert.Contains(t, conclusions[0].Description, "limited activity")
	assert.Equal(t, 0.75, conclusions[0].Confidence)
}

// Test makeStatusConclusions with high activity
func TestInference_MakeStatusConclusions_HighActivity(t *testing.T) {
	agent := NewInferenceAgent()

	intent := models.Intent{Type: "query_status", Confidence: 0.75}

	// Create 15 facts for high activity
	facts := []models.Fact{}
	for i := 0; i < 15; i++ {
		facts = append(facts, models.Fact{
			ID:      fmt.Sprintf("f%d", i),
			Content: fmt.Sprintf("Data %d", i),
			Source:  "gitlab",
		})
	}

	knowledge := []models.Knowledge{}
	chain := []models.InferenceStep{}
	conclusionID := 0

	conclusions := agent.makeStatusConclusions(intent, facts, knowledge, chain, &conclusionID)

	assert.Len(t, conclusions, 1)
	assert.Contains(t, conclusions[0].Description, "active with significant data")
	assert.Equal(t, 0.90, conclusions[0].Confidence)
}

// Test buildInferenceChain
func TestInference_BuildInferenceChain(t *testing.T) {
	agent := NewInferenceAgent()

	hypotheses := []models.Hypothesis{
		{ID: "h0", Description: "Retrieve commit data from GitLab"},
		{ID: "h1", Description: "Filter commits by date range"},
	}
	facts := []models.Fact{
		{ID: "f1", Content: "commit data from gitlab", Source: "gitlab"},
	}
	knowledge := []models.Knowledge{}

	chain := agent.buildInferenceChain(hypotheses, facts, knowledge)

	assert.Len(t, chain, 2) // One step per hypothesis
	assert.Equal(t, "h0", chain[0].Hypothesis)
	assert.Equal(t, "h1", chain[1].Hypothesis)
}

// Test extractKeywords
func TestInference_ExtractKeywords(t *testing.T) {
	agent := NewInferenceAgent()

	tests := []struct {
		name     string
		text     string
		expected int
	}{
		{
			name:     "Simple text",
			text:     "Retrieve commit data",
			expected: 3, // retrieve, commit, data
		},
		{
			name:     "With stop words",
			text:     "The commit was made in the repository",
			expected: 3, // commit, made, repository
		},
		{
			name:     "Short words filtered",
			text:     "A commit by me",
			expected: 1, // commit (other words too short or stop words)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keywords := agent.extractKeywords(tt.text)
			assert.Len(t, keywords, tt.expected)
		})
	}
}

// Test hasKeywordOverlap
func TestInference_HasKeywordOverlap(t *testing.T) {
	agent := NewInferenceAgent()

	keywords := []string{"commit", "gitlab", "data"}

	tests := []struct {
		name     string
		text     string
		expected bool
	}{
		{"Match found", "This is commit data", true},
		{"No match", "This is issue tracking", false},
		{"Case insensitive", "COMMIT information", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := agent.hasKeywordOverlap(keywords, tt.text)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Test calculateEvidenceConfidence
func TestInference_CalculateEvidenceConfidence(t *testing.T) {
	agent := NewInferenceAgent()

	tests := []struct {
		name             string
		evidenceCount    int
		expectedMin      float64
		expectedMax      float64
	}{
		{"No evidence", 0, 0.15, 0.25},
		{"Single evidence", 1, 0.45, 0.55},
		{"Two evidence", 2, 0.65, 0.75},
		{"Many evidence", 3, 0.85, 0.95},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create hypothesis with matching keywords
			h := models.Hypothesis{
				ID:          "h0",
				Description: "test hypothesis",
			}

			// Create facts that may or may not match
			facts := []models.Fact{}
			for i := 0; i < tt.evidenceCount; i++ {
				facts = append(facts, models.Fact{
					ID:      fmt.Sprintf("f%d", i),
					Content: "test hypothesis evidence", // Matches keywords
					Source:  "gitlab",
				})
			}

			knowledge := []models.Knowledge{}

			confidence := agent.calculateEvidenceConfidence(h, facts, knowledge)
			assert.GreaterOrEqual(t, confidence, tt.expectedMin)
			assert.LessOrEqual(t, confidence, tt.expectedMax)
		})
	}
}

// Test generateAlternatives
func TestInference_GenerateAlternatives(t *testing.T) {
	agent := NewInferenceAgent()

	conclusions := []models.Conclusion{
		{ID: "c0", Description: "Low confidence conclusion", Confidence: 0.50},
		{ID: "c1", Description: "High confidence conclusion", Confidence: 0.95},
		{ID: "c2", Description: "Medium confidence conclusion", Confidence: 0.65},
	}
	facts := []models.Fact{}

	alternatives := agent.generateAlternatives(conclusions, facts)

	// Only low-confidence conclusions generate alternatives (< 0.70)
	assert.Len(t, alternatives, 2)
	assert.Contains(t, alternatives[0].Description, "requires more data")
	assert.Less(t, alternatives[0].Confidence, conclusions[0].Confidence)
}

// Test context isolation (input not modified)
func TestInference_Execute_ContextIsolation(t *testing.T) {
	agent := NewInferenceAgent()

	originalCtx := &models.AgentContext{
		Reasoning: &models.ReasoningContext{
			Intents: []models.Intent{
				{Type: "query_commits", Confidence: 0.9},
			},
			Hypotheses: []models.Hypothesis{
				{ID: "h0", Description: "Test hypothesis"},
			},
		},
		Enrichment: &models.EnrichmentContext{
			Facts: []models.Fact{
				{ID: "f1", Content: "Test fact", Source: "gitlab"},
			},
		},
	}

	// Execute agent
	result, err := agent.Execute(context.Background(), originalCtx)

	require.NoError(t, err)
	require.NotNil(t, result)

	// Original context should not be modified
	assert.Nil(t, originalCtx.Reasoning.Conclusions)
	assert.Nil(t, originalCtx.Reasoning.InferenceChain)
	assert.Nil(t, originalCtx.Reasoning.Alternatives)

	// Result should have new data
	assert.NotNil(t, result.Reasoning.Conclusions)
	assert.NotNil(t, result.Reasoning.InferenceChain)
	assert.NotNil(t, result.Reasoning.Alternatives)
}

// Test audit trail tracking
func TestInference_Execute_AuditTrail(t *testing.T) {
	agent := NewInferenceAgent()

	ctx := &models.AgentContext{
		Reasoning: &models.ReasoningContext{
			Intents: []models.Intent{
				{Type: "query_commits", Confidence: 0.9},
			},
			Hypotheses: []models.Hypothesis{
				{ID: "h0", Description: "Test hypothesis"},
			},
		},
		Enrichment: &models.EnrichmentContext{
			Facts: []models.Fact{
				{ID: "f1", Content: "Test fact", Source: "gitlab"},
			},
		},
	}

	result, err := agent.Execute(context.Background(), ctx)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Audit)

	// Verify audit trail
	assert.Len(t, result.Audit.AgentRuns, 1)

	run := result.Audit.AgentRuns[0]
	assert.Equal(t, "inference", run.AgentID)
	assert.Equal(t, "success", run.Status)
	assert.GreaterOrEqual(t, run.DurationMS, int64(0))
	assert.Contains(t, run.KeysWritten, "reasoning.conclusions")
	assert.Contains(t, run.KeysWritten, "reasoning.alternatives")
	assert.Contains(t, run.KeysWritten, "reasoning.inference_chain")

	// Verify diagnostics
	require.NotNil(t, result.Diagnostics)
	require.NotNil(t, result.Diagnostics.Performance)
	require.NotNil(t, result.Diagnostics.Performance.AgentMetrics)

	metrics := result.Diagnostics.Performance.AgentMetrics["inference"]
	require.NotNil(t, metrics)
	assert.GreaterOrEqual(t, metrics.DurationMS, int64(0))
	assert.Equal(t, 0, metrics.LLMCalls) // No LLM calls
	assert.Equal(t, "success", metrics.Status)
}

// Test idempotency (same input produces same output)
func TestInference_Execute_Idempotency(t *testing.T) {
	agent := NewInferenceAgent()

	ctx := &models.AgentContext{
		Reasoning: &models.ReasoningContext{
			Intents: []models.Intent{
				{Type: "query_commits", Confidence: 0.9},
			},
			Hypotheses: []models.Hypothesis{
				{ID: "h0", Description: "Retrieve commit data"},
			},
		},
		Enrichment: &models.EnrichmentContext{
			Facts: []models.Fact{
				{ID: "f1", Content: "Commit message", Source: "gitlab"},
			},
		},
	}

	// Execute twice
	result1, err1 := agent.Execute(context.Background(), ctx)
	require.NoError(t, err1)

	result2, err2 := agent.Execute(context.Background(), ctx)
	require.NoError(t, err2)

	// Results should be the same (ignoring timestamps and audit trail)
	assert.Equal(t, len(result1.Reasoning.Conclusions), len(result2.Reasoning.Conclusions))
	assert.Equal(t, len(result1.Reasoning.InferenceChain), len(result2.Reasoning.InferenceChain))
	assert.Equal(t, len(result1.Reasoning.Alternatives), len(result2.Reasoning.Alternatives))

	// Conclusion content should be identical
	for i := range result1.Reasoning.Conclusions {
		assert.Equal(t, result1.Reasoning.Conclusions[i].Description, result2.Reasoning.Conclusions[i].Description)
		assert.Equal(t, result1.Reasoning.Conclusions[i].Confidence, result2.Reasoning.Conclusions[i].Confidence)
		assert.Equal(t, result1.Reasoning.Conclusions[i].Intent, result2.Reasoning.Conclusions[i].Intent)
	}
}

// Test multiple intents
func TestInference_Execute_MultipleIntents(t *testing.T) {
	agent := NewInferenceAgent()

	ctx := &models.AgentContext{
		Reasoning: &models.ReasoningContext{
			Intents: []models.Intent{
				{Type: "query_commits", Confidence: 0.9},
				{Type: "query_issues", Confidence: 0.85},
			},
			Hypotheses: []models.Hypothesis{
				{ID: "h0", Description: "Retrieve data"},
			},
		},
		Enrichment: &models.EnrichmentContext{
			Facts: []models.Fact{
				{ID: "f1", Content: "Commit 1", Source: "gitlab"},
				{ID: "f2", Content: "Issue 1", Source: "youtrack"},
			},
		},
	}

	result, err := agent.Execute(context.Background(), ctx)

	require.NoError(t, err)
	require.NotNil(t, result)

	// Should have conclusions for both intents
	assert.Len(t, result.Reasoning.Conclusions, 2)

	// Verify conclusions cover both intents
	intents := []string{}
	for _, c := range result.Reasoning.Conclusions {
		intents = append(intents, c.Intent)
	}
	assert.Contains(t, intents, "query_commits")
	assert.Contains(t, intents, "query_issues")
}
