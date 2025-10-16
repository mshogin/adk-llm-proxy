package agents

import (
	"context"
	"testing"

	"github.com/mshogin/agents/internal/domain/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test agent initialization
func TestNewValidationAgent(t *testing.T) {
	agent := NewValidationAgent()
	assert.NotNil(t, agent)
	assert.Equal(t, "validation", agent.AgentID())
}

// Test AgentID method
func TestValidation_AgentID(t *testing.T) {
	agent := NewValidationAgent()
	assert.Equal(t, "validation", agent.AgentID())
}

// Test Preconditions method
func TestValidation_Preconditions(t *testing.T) {
	agent := NewValidationAgent()
	preconditions := agent.Preconditions()

	assert.Len(t, preconditions, 3)
	assert.Contains(t, preconditions, "reasoning.intents")
	assert.Contains(t, preconditions, "reasoning.hypotheses")
	assert.Contains(t, preconditions, "reasoning.conclusions")
}

// Test Postconditions method
func TestValidation_Postconditions(t *testing.T) {
	agent := NewValidationAgent()
	postconditions := agent.Postconditions()

	assert.Len(t, postconditions, 3)
	assert.Contains(t, postconditions, "diagnostics.validation_reports")
	assert.Contains(t, postconditions, "diagnostics.errors")
	assert.Contains(t, postconditions, "diagnostics.warnings")
}

// Test GetMetadata method
func TestValidation_GetMetadata(t *testing.T) {
	agent := NewValidationAgent()
	metadata := agent.GetMetadata()

	assert.Equal(t, "validation", metadata.ID)
	assert.Equal(t, "Validation Agent", metadata.Name)
	assert.NotEmpty(t, metadata.Description)
	assert.Equal(t, "1.0.0", metadata.Version)
	assert.NotEmpty(t, metadata.Tags)
	assert.Contains(t, metadata.Dependencies, "inference")
}

// Test GetCapabilities method
func TestValidation_GetCapabilities(t *testing.T) {
	agent := NewValidationAgent()
	caps := agent.GetCapabilities()

	assert.False(t, caps.SupportsParallelExecution)
	assert.True(t, caps.SupportsRetry)
	assert.False(t, caps.RequiresLLM)
	assert.True(t, caps.IsDeterministic)
	assert.Greater(t, caps.EstimatedDuration, 0)
}

// Test Execute with valid inputs (all checks pass)
func TestValidation_Execute_AllCheckPass(t *testing.T) {
	agent := NewValidationAgent()

	ctx := &models.AgentContext{
		Reasoning: &models.ReasoningContext{
			Intents: []models.Intent{
				{Type: "query_commits", Confidence: 0.9},
			},
			Hypotheses: []models.Hypothesis{
				{ID: "h0", Description: "Retrieve commit data"},
			},
			Conclusions: []models.Conclusion{
				{
					ID:          "c0",
					Description: "Found 2 commits",
					Confidence:  0.95,
					Evidence:    []string{"fact:f1"},
					Intent:      "query_commits",
				},
			},
		},
		Enrichment: &models.EnrichmentContext{
			Facts: []models.Fact{
				{
					ID:         "f1",
					Content:    "Commit message",
					Source:     "gitlab",
					Confidence: 0.95,
					Provenance: map[string]interface{}{
						"artifact_id": "a1",
						"source":      "gitlab",
					},
				},
			},
		},
	}

	result, err := agent.Execute(context.Background(), ctx)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Diagnostics)

	// Verify validation reports created
	assert.Len(t, result.Diagnostics.ValidationReports, 5) // 5 validation checks

	// All checks should pass
	for _, report := range result.Diagnostics.ValidationReports {
		assert.True(t, report.Passed, "Report should pass: %v", report.Issues)
	}

	// No errors or warnings
	assert.Empty(t, result.Diagnostics.Errors)
	assert.Empty(t, result.Diagnostics.Warnings)
}

// Test Execute with missing intents (precondition failure)
func TestValidation_Execute_MissingIntents(t *testing.T) {
	agent := NewValidationAgent()

	ctx := &models.AgentContext{
		Reasoning: &models.ReasoningContext{
			Intents:     []models.Intent{}, // Empty
			Hypotheses:  []models.Hypothesis{{ID: "h0"}},
			Conclusions: []models.Conclusion{{ID: "c0"}},
		},
	}

	result, err := agent.Execute(context.Background(), ctx)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "no intents found")
}

// Test Execute with missing hypotheses (precondition failure)
func TestValidation_Execute_MissingHypotheses(t *testing.T) {
	agent := NewValidationAgent()

	ctx := &models.AgentContext{
		Reasoning: &models.ReasoningContext{
			Intents:     []models.Intent{{Type: "query_commits"}},
			Hypotheses:  []models.Hypothesis{}, // Empty
			Conclusions: []models.Conclusion{{ID: "c0"}},
		},
	}

	result, err := agent.Execute(context.Background(), ctx)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "no hypotheses found")
}

// Test Execute with missing conclusions (precondition failure)
func TestValidation_Execute_MissingConclusions(t *testing.T) {
	agent := NewValidationAgent()

	ctx := &models.AgentContext{
		Reasoning: &models.ReasoningContext{
			Intents:     []models.Intent{{Type: "query_commits"}},
			Hypotheses:  []models.Hypothesis{{ID: "h0"}},
			Conclusions: []models.Conclusion{}, // Empty
		},
	}

	result, err := agent.Execute(context.Background(), ctx)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "no conclusions found")
}

// Test validateIntentCompleteness with missing conclusion
func TestValidation_IntentCompleteness_MissingConclusion(t *testing.T) {
	agent := NewValidationAgent()

	ctx := &models.AgentContext{
		Reasoning: &models.ReasoningContext{
			Intents: []models.Intent{
				{Type: "query_commits", Confidence: 0.9},
				{Type: "query_issues", Confidence: 0.85},
			},
			Hypotheses: []models.Hypothesis{
				{ID: "h0", Description: "Test"},
			},
			Conclusions: []models.Conclusion{
				// Only has conclusion for query_commits, missing query_issues
				{ID: "c0", Intent: "query_commits"},
			},
		},
	}

	result, err := agent.Execute(context.Background(), ctx)

	require.NoError(t, err)
	require.NotNil(t, result)

	// Find intent completeness report (first report)
	var intentReport *models.ValidationReport
	if len(result.Diagnostics.ValidationReports) > 0 {
		intentReport = &result.Diagnostics.ValidationReports[0]
	}

	require.NotNil(t, intentReport)
	assert.False(t, intentReport.Passed)
	assert.Len(t, intentReport.Issues, 1)
	assert.Contains(t, intentReport.Issues[0], "query_issues")
	assert.NotEmpty(t, intentReport.AutoFixes)

	// Should have warning
	assert.NotEmpty(t, result.Diagnostics.Warnings)
}

// Test validateHypothesisConsistency with missing dependency
func TestValidation_HypothesisConsistency_MissingDependency(t *testing.T) {
	agent := NewValidationAgent()

	ctx := &models.AgentContext{
		Reasoning: &models.ReasoningContext{
			Intents: []models.Intent{{Type: "query_commits"}},
			Hypotheses: []models.Hypothesis{
				{ID: "h0", Description: "First", Dependencies: []string{"h999"}}, // h999 doesn't exist
			},
			Conclusions: []models.Conclusion{
				{ID: "c0", Intent: "query_commits"},
			},
		},
	}

	result, err := agent.Execute(context.Background(), ctx)

	require.NoError(t, err)
	require.NotNil(t, result)

	// Find hypothesis consistency report (second report)
	var hypothesisReport *models.ValidationReport
	if len(result.Diagnostics.ValidationReports) > 1 {
		hypothesisReport = &result.Diagnostics.ValidationReports[1]
	}

	require.NotNil(t, hypothesisReport)
	assert.False(t, hypothesisReport.Passed)
	assert.Len(t, hypothesisReport.Issues, 1)
	assert.Contains(t, hypothesisReport.Issues[0], "h999")
	assert.NotEmpty(t, hypothesisReport.AutoFixes)

	// Should have error
	assert.NotEmpty(t, result.Diagnostics.Errors)
}

// Test detectDependencyCycles with actual cycle
func TestValidation_DependencyCycles_CycleDetected(t *testing.T) {
	agent := NewValidationAgent()

	ctx := &models.AgentContext{
		Reasoning: &models.ReasoningContext{
			Intents: []models.Intent{{Type: "query_commits"}},
			Hypotheses: []models.Hypothesis{
				{ID: "h0", Description: "First", Dependencies: []string{"h1"}},
				{ID: "h1", Description: "Second", Dependencies: []string{"h2"}},
				{ID: "h2", Description: "Third", Dependencies: []string{"h0"}}, // Cycle: h0 → h1 → h2 → h0
			},
			Conclusions: []models.Conclusion{
				{ID: "c0", Intent: "query_commits"},
			},
		},
	}

	result, err := agent.Execute(context.Background(), ctx)

	require.NoError(t, err)
	require.NotNil(t, result)

	// Find dependency cycle report (third report)
	var cycleReport *models.ValidationReport
	if len(result.Diagnostics.ValidationReports) > 2 {
		cycleReport = &result.Diagnostics.ValidationReports[2]
	}

	require.NotNil(t, cycleReport)
	assert.False(t, cycleReport.Passed)
	assert.Len(t, cycleReport.Issues, 1)
	assert.Contains(t, cycleReport.Issues[0], "cycle")
	assert.NotEmpty(t, cycleReport.AutoFixes)

	// Should have error
	assert.NotEmpty(t, result.Diagnostics.Errors)
}

// Test detectDependencyCycles with no cycle
func TestValidation_DependencyCycles_NoCycle(t *testing.T) {
	agent := NewValidationAgent()

	ctx := &models.AgentContext{
		Reasoning: &models.ReasoningContext{
			Intents: []models.Intent{{Type: "query_commits"}},
			Hypotheses: []models.Hypothesis{
				{ID: "h0", Description: "First", Dependencies: []string{}},
				{ID: "h1", Description: "Second", Dependencies: []string{"h0"}},
				{ID: "h2", Description: "Third", Dependencies: []string{"h1"}},
			},
			Conclusions: []models.Conclusion{
				{ID: "c0", Intent: "query_commits"},
			},
		},
	}

	result, err := agent.Execute(context.Background(), ctx)

	require.NoError(t, err)
	require.NotNil(t, result)

	// Find dependency cycle report (third report)
	var cycleReport *models.ValidationReport
	if len(result.Diagnostics.ValidationReports) > 2 {
		cycleReport = &result.Diagnostics.ValidationReports[2]
	}

	require.NotNil(t, cycleReport)
	assert.True(t, cycleReport.Passed) // No cycle
	assert.Empty(t, cycleReport.Issues)
}

// Test validateConclusionEvidence with missing evidence
func TestValidation_ConclusionEvidence_MissingEvidence(t *testing.T) {
	agent := NewValidationAgent()

	ctx := &models.AgentContext{
		Reasoning: &models.ReasoningContext{
			Intents: []models.Intent{{Type: "query_commits"}},
			Hypotheses: []models.Hypothesis{
				{ID: "h0", Description: "Test"},
			},
			Conclusions: []models.Conclusion{
				{ID: "c0", Intent: "query_commits", Evidence: []string{}}, // No evidence
			},
		},
		Enrichment: &models.EnrichmentContext{
			Facts: []models.Fact{
				{ID: "f1", Content: "Test fact"},
			},
		},
	}

	result, err := agent.Execute(context.Background(), ctx)

	require.NoError(t, err)
	require.NotNil(t, result)

	// Find conclusion evidence report (fourth report)
	var evidenceReport *models.ValidationReport
	if len(result.Diagnostics.ValidationReports) > 3 {
		evidenceReport = &result.Diagnostics.ValidationReports[3]
	}

	require.NotNil(t, evidenceReport)
	assert.False(t, evidenceReport.Passed)
	assert.Len(t, evidenceReport.Issues, 1)
	assert.Contains(t, evidenceReport.Issues[0], "no supporting evidence")
	assert.NotEmpty(t, evidenceReport.AutoFixes)

	// Should have warning
	assert.NotEmpty(t, result.Diagnostics.Warnings)
}

// Test validateConclusionEvidence with invalid evidence reference
func TestValidation_ConclusionEvidence_InvalidReference(t *testing.T) {
	agent := NewValidationAgent()

	ctx := &models.AgentContext{
		Reasoning: &models.ReasoningContext{
			Intents: []models.Intent{{Type: "query_commits"}},
			Hypotheses: []models.Hypothesis{
				{ID: "h0", Description: "Test"},
			},
			Conclusions: []models.Conclusion{
				{ID: "c0", Intent: "query_commits", Evidence: []string{"fact:f999"}}, // f999 doesn't exist
			},
		},
		Enrichment: &models.EnrichmentContext{
			Facts: []models.Fact{
				{ID: "f1", Content: "Test fact"},
			},
		},
	}

	result, err := agent.Execute(context.Background(), ctx)

	require.NoError(t, err)
	require.NotNil(t, result)

	// Find conclusion evidence report (fourth report)
	var evidenceReport *models.ValidationReport
	if len(result.Diagnostics.ValidationReports) > 3 {
		evidenceReport = &result.Diagnostics.ValidationReports[3]
	}

	require.NotNil(t, evidenceReport)
	assert.False(t, evidenceReport.Passed)
	assert.Len(t, evidenceReport.Issues, 1)
	assert.Contains(t, evidenceReport.Issues[0], "non-existent evidence")
	assert.NotEmpty(t, evidenceReport.AutoFixes)

	// Should have error
	assert.NotEmpty(t, result.Diagnostics.Errors)
}

// Test validateFactProvenance with missing provenance
func TestValidation_FactProvenance_MissingProvenance(t *testing.T) {
	agent := NewValidationAgent()

	ctx := &models.AgentContext{
		Reasoning: &models.ReasoningContext{
			Intents: []models.Intent{{Type: "query_commits"}},
			Hypotheses: []models.Hypothesis{
				{ID: "h0", Description: "Test"},
			},
			Conclusions: []models.Conclusion{
				{ID: "c0", Intent: "query_commits", Evidence: []string{"fact:f1"}},
			},
		},
		Enrichment: &models.EnrichmentContext{
			Facts: []models.Fact{
				{ID: "f1", Content: "Test fact", Provenance: nil}, // Missing provenance
			},
		},
	}

	result, err := agent.Execute(context.Background(), ctx)

	require.NoError(t, err)
	require.NotNil(t, result)

	// Find fact provenance report (fifth report)
	var provenanceReport *models.ValidationReport
	if len(result.Diagnostics.ValidationReports) > 4 {
		provenanceReport = &result.Diagnostics.ValidationReports[4]
	}

	require.NotNil(t, provenanceReport)
	assert.False(t, provenanceReport.Passed)
	assert.Len(t, provenanceReport.Issues, 1)
	assert.Contains(t, provenanceReport.Issues[0], "missing provenance")
	assert.NotEmpty(t, provenanceReport.AutoFixes)

	// Should have warning
	assert.NotEmpty(t, result.Diagnostics.Warnings)
}

// Test validateFactProvenance with invalid confidence score
func TestValidation_FactProvenance_InvalidConfidence(t *testing.T) {
	agent := NewValidationAgent()

	ctx := &models.AgentContext{
		Reasoning: &models.ReasoningContext{
			Intents: []models.Intent{{Type: "query_commits"}},
			Hypotheses: []models.Hypothesis{
				{ID: "h0", Description: "Test"},
			},
			Conclusions: []models.Conclusion{
				{ID: "c0", Intent: "query_commits", Evidence: []string{"fact:f1"}},
			},
		},
		Enrichment: &models.EnrichmentContext{
			Facts: []models.Fact{
				{
					ID:         "f1",
					Content:    "Test fact",
					Confidence: 1.5, // Invalid (>1.0)
					Provenance: map[string]interface{}{"source": "test"},
				},
			},
		},
	}

	result, err := agent.Execute(context.Background(), ctx)

	require.NoError(t, err)
	require.NotNil(t, result)

	// Find fact provenance report (fifth report)
	var provenanceReport *models.ValidationReport
	if len(result.Diagnostics.ValidationReports) > 4 {
		provenanceReport = &result.Diagnostics.ValidationReports[4]
	}

	require.NotNil(t, provenanceReport)
	assert.False(t, provenanceReport.Passed)
	assert.Contains(t, provenanceReport.Issues[0], "invalid confidence")
	assert.NotEmpty(t, provenanceReport.AutoFixes)

	// Should have error
	assert.NotEmpty(t, result.Diagnostics.Errors)
}

// Test evidenceExists helper
func TestValidation_EvidenceExists(t *testing.T) {
	agent := NewValidationAgent()

	ctx := &models.AgentContext{
		Enrichment: &models.EnrichmentContext{
			Facts: []models.Fact{
				{ID: "f1", Content: "Test fact"},
			},
			DerivedKnowledge: []models.Knowledge{
				{ID: "k1", Content: "Test knowledge"},
			},
		},
	}

	tests := []struct {
		name         string
		evidenceRef  string
		shouldExist  bool
	}{
		{"Valid fact reference", "fact:f1", true},
		{"Valid knowledge reference", "knowledge:k1", true},
		{"Invalid fact reference", "fact:f999", false},
		{"Invalid knowledge reference", "knowledge:k999", false},
		{"Malformed reference", "invalid", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exists := agent.evidenceExists(tt.evidenceRef, ctx)
			assert.Equal(t, tt.shouldExist, exists)
		})
	}
}

// Test context isolation (input not modified)
func TestValidation_Execute_ContextIsolation(t *testing.T) {
	agent := NewValidationAgent()

	originalCtx := &models.AgentContext{
		Reasoning: &models.ReasoningContext{
			Intents: []models.Intent{{Type: "query_commits"}},
			Hypotheses: []models.Hypothesis{
				{ID: "h0", Description: "Test"},
			},
			Conclusions: []models.Conclusion{
				{ID: "c0", Intent: "query_commits", Evidence: []string{"fact:f1"}},
			},
		},
		Enrichment: &models.EnrichmentContext{
			Facts: []models.Fact{
				{ID: "f1", Content: "Test", Confidence: 0.95, Provenance: map[string]interface{}{"source": "test"}},
			},
		},
	}

	// Execute agent
	result, err := agent.Execute(context.Background(), originalCtx)

	require.NoError(t, err)
	require.NotNil(t, result)

	// Original context should not be modified
	assert.Nil(t, originalCtx.Diagnostics)

	// Result should have diagnostics
	assert.NotNil(t, result.Diagnostics)
	assert.NotEmpty(t, result.Diagnostics.ValidationReports)
}

// Test audit trail tracking
func TestValidation_Execute_AuditTrail(t *testing.T) {
	agent := NewValidationAgent()

	ctx := &models.AgentContext{
		Reasoning: &models.ReasoningContext{
			Intents: []models.Intent{{Type: "query_commits"}},
			Hypotheses: []models.Hypothesis{
				{ID: "h0", Description: "Test"},
			},
			Conclusions: []models.Conclusion{
				{ID: "c0", Intent: "query_commits", Evidence: []string{"fact:f1"}},
			},
		},
		Enrichment: &models.EnrichmentContext{
			Facts: []models.Fact{
				{ID: "f1", Content: "Test", Confidence: 0.95, Provenance: map[string]interface{}{"source": "test"}},
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
	assert.Equal(t, "validation", run.AgentID)
	assert.Equal(t, "success", run.Status)
	assert.GreaterOrEqual(t, run.DurationMS, int64(0))
	assert.Contains(t, run.KeysWritten, "diagnostics.validation_reports")
	assert.Contains(t, run.KeysWritten, "diagnostics.errors")
	assert.Contains(t, run.KeysWritten, "diagnostics.warnings")

	// Verify diagnostics
	require.NotNil(t, result.Diagnostics)
	require.NotNil(t, result.Diagnostics.Performance)
	require.NotNil(t, result.Diagnostics.Performance.AgentMetrics)

	metrics := result.Diagnostics.Performance.AgentMetrics["validation"]
	require.NotNil(t, metrics)
	assert.GreaterOrEqual(t, metrics.DurationMS, int64(0))
	assert.Equal(t, 0, metrics.LLMCalls) // No LLM calls
	assert.Equal(t, "success", metrics.Status)
}

// Test idempotency (same input produces same output)
func TestValidation_Execute_Idempotency(t *testing.T) {
	agent := NewValidationAgent()

	ctx := &models.AgentContext{
		Reasoning: &models.ReasoningContext{
			Intents: []models.Intent{{Type: "query_commits"}},
			Hypotheses: []models.Hypothesis{
				{ID: "h0", Description: "Test"},
			},
			Conclusions: []models.Conclusion{
				{ID: "c0", Intent: "query_commits", Evidence: []string{"fact:f1"}},
			},
		},
		Enrichment: &models.EnrichmentContext{
			Facts: []models.Fact{
				{ID: "f1", Content: "Test", Confidence: 0.95, Provenance: map[string]interface{}{"source": "test"}},
			},
		},
	}

	// Execute twice
	result1, err1 := agent.Execute(context.Background(), ctx)
	require.NoError(t, err1)

	result2, err2 := agent.Execute(context.Background(), ctx)
	require.NoError(t, err2)

	// Results should be the same (ignoring timestamps and audit trail)
	assert.Equal(t, len(result1.Diagnostics.ValidationReports), len(result2.Diagnostics.ValidationReports))
	assert.Equal(t, len(result1.Diagnostics.Errors), len(result2.Diagnostics.Errors))
	assert.Equal(t, len(result1.Diagnostics.Warnings), len(result2.Diagnostics.Warnings))

	// Report pass/fail status should be identical
	for i := range result1.Diagnostics.ValidationReports {
		assert.Equal(t, result1.Diagnostics.ValidationReports[i].Passed, result2.Diagnostics.ValidationReports[i].Passed)
		assert.Equal(t, len(result1.Diagnostics.ValidationReports[i].Issues), len(result2.Diagnostics.ValidationReports[i].Issues))
	}
}

// Test multiple validation failures
func TestValidation_Execute_MultipleFailures(t *testing.T) {
	agent := NewValidationAgent()

	ctx := &models.AgentContext{
		Reasoning: &models.ReasoningContext{
			Intents: []models.Intent{
				{Type: "query_commits"},
				{Type: "query_issues"}, // Missing conclusion
			},
			Hypotheses: []models.Hypothesis{
				{ID: "h0", Dependencies: []string{"h999"}}, // Missing dependency
			},
			Conclusions: []models.Conclusion{
				{ID: "c0", Intent: "query_commits", Evidence: []string{}}, // No evidence
			},
		},
		Enrichment: &models.EnrichmentContext{
			Facts: []models.Fact{
				{ID: "f1", Confidence: 1.5}, // Invalid confidence
			},
		},
	}

	result, err := agent.Execute(context.Background(), ctx)

	require.NoError(t, err)
	require.NotNil(t, result)

	// Should have multiple validation failures
	failedReports := 0
	for _, report := range result.Diagnostics.ValidationReports {
		if !report.Passed {
			failedReports++
		}
	}
	assert.Greater(t, failedReports, 1) // Multiple failures

	// Should have both errors and warnings
	assert.NotEmpty(t, result.Diagnostics.Errors)
	assert.NotEmpty(t, result.Diagnostics.Warnings)
}
