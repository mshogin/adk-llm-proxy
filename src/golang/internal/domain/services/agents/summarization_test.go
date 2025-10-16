package agents

import (
	"context"
	"testing"
	"time"

	"github.com/mshogin/agents/internal/domain/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSummarization_AgentID(t *testing.T) {
	agent := NewSummarizationAgent()
	assert.Equal(t, "summarization", agent.AgentID())
}

func TestSummarization_Preconditions(t *testing.T) {
	agent := NewSummarizationAgent()
	preconditions := agent.Preconditions()

	assert.Contains(t, preconditions, "reasoning.intents")
	assert.Contains(t, preconditions, "reasoning.conclusions")
}

func TestSummarization_Postconditions(t *testing.T) {
	agent := NewSummarizationAgent()
	postconditions := agent.Postconditions()

	assert.Contains(t, postconditions, "reasoning.summary")
	assert.Contains(t, postconditions, "reasoning.artifacts")
}

func TestSummarization_GetMetadata(t *testing.T) {
	agent := NewSummarizationAgent()
	metadata := agent.GetMetadata()

	assert.Equal(t, "summarization", metadata.ID)
	assert.Equal(t, "Summarization Agent", metadata.Name)
	assert.Equal(t, "1.0.0", metadata.Version)
	assert.Contains(t, metadata.Tags, "summarization")
	assert.Contains(t, metadata.Dependencies, "inference")
	assert.Contains(t, metadata.Dependencies, "validation")
}

func TestSummarization_GetCapabilities(t *testing.T) {
	agent := NewSummarizationAgent()
	capabilities := agent.GetCapabilities()

	assert.False(t, capabilities.SupportsParallelExecution)
	assert.True(t, capabilities.SupportsRetry)
	assert.False(t, capabilities.RequiresLLM)
	assert.True(t, capabilities.IsDeterministic)
	assert.Equal(t, int64(50), int64(capabilities.EstimatedDuration))
}

func TestSummarization_Execute_ValidInputs(t *testing.T) {
	agent := NewSummarizationAgent()

	ctx := &models.AgentContext{
		Reasoning: &models.ReasoningContext{
			Intents: []models.Intent{
				{Type: "query_commits", Confidence: 0.9},
			},
			Conclusions: []models.Conclusion{
				{
					ID:          "c0",
					Description: "Found 3 commit(s) from GitLab",
					Confidence:  0.95,
					Intent:      "query_commits",
				},
			},
		},
		Enrichment: &models.EnrichmentContext{
			Facts: []models.Fact{
				{ID: "f1", Content: "Commit 1", Source: "gitlab", Confidence: 0.95},
				{ID: "f2", Content: "Commit 2", Source: "gitlab", Confidence: 0.95},
				{ID: "f3", Content: "Commit 3", Source: "gitlab", Confidence: 0.95},
			},
		},
	}

	result, err := agent.Execute(context.Background(), ctx)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Reasoning)

	// Check summary
	assert.NotEmpty(t, result.Reasoning.Summary)
	assert.Contains(t, result.Reasoning.Summary, "query_commits")
	assert.Contains(t, result.Reasoning.Summary, "3 fact(s)")
	assert.Contains(t, result.Reasoning.Summary, "gitlab")

	// Check artifacts
	assert.NotEmpty(t, result.Reasoning.Artifacts)

	// Check audit
	require.NotNil(t, result.Audit)
	assert.NotEmpty(t, result.Audit.AgentRuns)

	// Find summarization run
	var summRun *models.AgentRun
	for _, run := range result.Audit.AgentRuns {
		if run.AgentID == "summarization" {
			summRun = &run
			break
		}
	}

	require.NotNil(t, summRun)
	assert.Equal(t, "success", summRun.Status)
	assert.Contains(t, summRun.KeysWritten, "reasoning.summary")
	assert.Contains(t, summRun.KeysWritten, "reasoning.artifacts")
}

func TestSummarization_Execute_MissingIntents(t *testing.T) {
	agent := NewSummarizationAgent()

	ctx := &models.AgentContext{
		Reasoning: &models.ReasoningContext{
			Intents:     []models.Intent{}, // Empty
			Conclusions: []models.Conclusion{{ID: "c0"}},
		},
	}

	result, err := agent.Execute(context.Background(), ctx)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "no intents found")
}

func TestSummarization_Execute_MissingConclusions(t *testing.T) {
	agent := NewSummarizationAgent()

	ctx := &models.AgentContext{
		Reasoning: &models.ReasoningContext{
			Intents:     []models.Intent{{Type: "query_commits"}},
			Conclusions: []models.Conclusion{}, // Empty
		},
	}

	result, err := agent.Execute(context.Background(), ctx)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "no conclusions found")
}

func TestSummarization_Execute_NilReasoningContext(t *testing.T) {
	agent := NewSummarizationAgent()

	ctx := &models.AgentContext{
		Reasoning: nil,
	}

	result, err := agent.Execute(context.Background(), ctx)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "reasoning context is nil")
}

func TestSummarization_SummarizeIntents_SingleIntent(t *testing.T) {
	agent := NewSummarizationAgent()

	intents := []models.Intent{
		{Type: "query_commits", Confidence: 0.9},
	}

	summary := agent.summarizeIntents(intents)

	assert.Contains(t, summary, "query_commits")
	assert.Contains(t, summary, "Detected intent")
}

func TestSummarization_SummarizeIntents_MultipleIntents(t *testing.T) {
	agent := NewSummarizationAgent()

	intents := []models.Intent{
		{Type: "query_commits", Confidence: 0.9},
		{Type: "query_issues", Confidence: 0.8},
	}

	summary := agent.summarizeIntents(intents)

	assert.Contains(t, summary, "2 intents")
	assert.Contains(t, summary, "query_commits")
	assert.Contains(t, summary, "query_issues")
}

func TestSummarization_SummarizeData_SingleSource(t *testing.T) {
	agent := NewSummarizationAgent()

	ctx := &models.AgentContext{
		Enrichment: &models.EnrichmentContext{
			Facts: []models.Fact{
				{ID: "f1", Source: "gitlab"},
				{ID: "f2", Source: "gitlab"},
			},
		},
	}

	summary := agent.summarizeData(ctx)

	assert.Contains(t, summary, "2 fact(s)")
	assert.Contains(t, summary, "gitlab")
}

func TestSummarization_SummarizeData_MultipleSources(t *testing.T) {
	agent := NewSummarizationAgent()

	ctx := &models.AgentContext{
		Enrichment: &models.EnrichmentContext{
			Facts: []models.Fact{
				{ID: "f1", Source: "gitlab"},
				{ID: "f2", Source: "gitlab"},
				{ID: "f3", Source: "youtrack"},
			},
		},
	}

	summary := agent.summarizeData(ctx)

	assert.Contains(t, summary, "3 fact(s)")
	assert.Contains(t, summary, "gitlab")
	assert.Contains(t, summary, "youtrack")
}

func TestSummarization_SummarizeData_NoFacts(t *testing.T) {
	agent := NewSummarizationAgent()

	ctx := &models.AgentContext{
		Enrichment: &models.EnrichmentContext{
			Facts: []models.Fact{},
		},
	}

	summary := agent.summarizeData(ctx)

	assert.Contains(t, summary, "No data retrieved")
}

func TestSummarization_SummarizeConclusions_HighConfidence(t *testing.T) {
	agent := NewSummarizationAgent()

	conclusions := []models.Conclusion{
		{ID: "c0", Confidence: 0.95},
		{ID: "c1", Confidence: 0.92},
	}

	summary := agent.summarizeConclusions(conclusions)

	assert.Contains(t, summary, "2 conclusions")
	assert.Contains(t, summary, "high")
}

func TestSummarization_SummarizeConclusions_GoodConfidence(t *testing.T) {
	agent := NewSummarizationAgent()

	conclusions := []models.Conclusion{
		{ID: "c0", Confidence: 0.75},
		{ID: "c1", Confidence: 0.80},
	}

	summary := agent.summarizeConclusions(conclusions)

	assert.Contains(t, summary, "2 conclusions")
	assert.Contains(t, summary, "good")
}

func TestSummarization_SummarizeConclusions_ModerateConfidence(t *testing.T) {
	agent := NewSummarizationAgent()

	conclusions := []models.Conclusion{
		{ID: "c0", Confidence: 0.55},
		{ID: "c1", Confidence: 0.65},
	}

	summary := agent.summarizeConclusions(conclusions)

	assert.Contains(t, summary, "2 conclusions")
	assert.Contains(t, summary, "moderate")
}

func TestSummarization_SummarizeConclusions_LowConfidence(t *testing.T) {
	agent := NewSummarizationAgent()

	conclusions := []models.Conclusion{
		{ID: "c0", Confidence: 0.35},
		{ID: "c1", Confidence: 0.45},
	}

	summary := agent.summarizeConclusions(conclusions)

	assert.Contains(t, summary, "2 conclusions")
	assert.Contains(t, summary, "low")
}

func TestSummarization_SummarizeValidation_NoIssues(t *testing.T) {
	agent := NewSummarizationAgent()

	ctx := &models.AgentContext{
		Diagnostics: &models.DiagnosticsContext{
			Errors:   []models.ErrorReport{},
			Warnings: []models.Warning{},
		},
	}

	summary := agent.summarizeValidation(ctx)

	assert.Contains(t, summary, "Validation passed")
}

func TestSummarization_SummarizeValidation_WithErrors(t *testing.T) {
	agent := NewSummarizationAgent()

	ctx := &models.AgentContext{
		Diagnostics: &models.DiagnosticsContext{
			Errors: []models.ErrorReport{
				{Message: "Error 1"},
			},
		},
	}

	summary := agent.summarizeValidation(ctx)

	assert.Contains(t, summary, "1 error(s)")
}

func TestSummarization_SummarizeValidation_WithWarnings(t *testing.T) {
	agent := NewSummarizationAgent()

	ctx := &models.AgentContext{
		Diagnostics: &models.DiagnosticsContext{
			Warnings: []models.Warning{
				{Message: "Warning 1"},
				{Message: "Warning 2"},
			},
		},
	}

	summary := agent.summarizeValidation(ctx)

	assert.Contains(t, summary, "2 warning(s)")
}

func TestSummarization_SummarizeValidation_WithErrorsAndWarnings(t *testing.T) {
	agent := NewSummarizationAgent()

	ctx := &models.AgentContext{
		Diagnostics: &models.DiagnosticsContext{
			Errors: []models.ErrorReport{
				{Message: "Error 1"},
			},
			Warnings: []models.Warning{
				{Message: "Warning 1"},
			},
		},
	}

	summary := agent.summarizeValidation(ctx)

	assert.Contains(t, summary, "1 error(s)")
	assert.Contains(t, summary, "1 warning(s)")
}

func TestSummarization_GenerateReport(t *testing.T) {
	agent := NewSummarizationAgent()

	ctx := &models.AgentContext{
		Reasoning: &models.ReasoningContext{
			Intents: []models.Intent{
				{Type: "query_commits", Confidence: 0.9},
			},
			Conclusions: []models.Conclusion{
				{
					ID:          "c0",
					Description: "Found 3 commit(s) from GitLab",
					Confidence:  0.95,
				},
			},
		},
		Enrichment: &models.EnrichmentContext{
			Facts: []models.Fact{
				{ID: "f1", Source: "gitlab", Content: "Commit 1"},
				{ID: "f2", Source: "gitlab", Content: "Commit 2"},
			},
		},
	}

	report := agent.generateReport(ctx)

	assert.Contains(t, report, "# Reasoning Report")
	assert.Contains(t, report, "## Detected Intents")
	assert.Contains(t, report, "query_commits")
	assert.Contains(t, report, "## Retrieved Data")
	assert.Contains(t, report, "gitlab")
	assert.Contains(t, report, "## Conclusions")
	assert.Contains(t, report, "Found 3 commit(s) from GitLab")
}

func TestSummarization_GenerateReport_WithValidation(t *testing.T) {
	agent := NewSummarizationAgent()

	ctx := &models.AgentContext{
		Reasoning: &models.ReasoningContext{
			Intents: []models.Intent{
				{Type: "query_commits", Confidence: 0.9},
			},
			Conclusions: []models.Conclusion{
				{ID: "c0", Description: "Found commits", Confidence: 0.95},
			},
		},
		Diagnostics: &models.DiagnosticsContext{
			Errors: []models.ErrorReport{
				{Message: "Error 1"},
			},
			Warnings: []models.Warning{
				{Message: "Warning 1"},
			},
		},
	}

	report := agent.generateReport(ctx)

	assert.Contains(t, report, "## Validation Results")
	assert.Contains(t, report, "### Errors")
	assert.Contains(t, report, "Error 1")
	assert.Contains(t, report, "### Warnings")
	assert.Contains(t, report, "Warning 1")
}

func TestSummarization_GenerateCommandList_CommitIntent(t *testing.T) {
	agent := NewSummarizationAgent()

	ctx := &models.AgentContext{
		Reasoning: &models.ReasoningContext{
			Conclusions: []models.Conclusion{
				{Description: "Found 3 commits from GitLab"},
			},
		},
	}

	commands := agent.generateCommandList(ctx)

	assert.Contains(t, commands, "git log")
}

func TestSummarization_GenerateCommandList_IssueIntent(t *testing.T) {
	agent := NewSummarizationAgent()

	ctx := &models.AgentContext{
		Reasoning: &models.ReasoningContext{
			Conclusions: []models.Conclusion{
				{Description: "Found 5 issues from YouTrack"},
			},
		},
	}

	commands := agent.generateCommandList(ctx)

	assert.Contains(t, commands, "youtrack issues list")
}

func TestSummarization_GenerateCommandList_NoCommands(t *testing.T) {
	agent := NewSummarizationAgent()

	ctx := &models.AgentContext{
		Reasoning: &models.ReasoningContext{
			Conclusions: []models.Conclusion{
				{Description: "System is active"},
			},
		},
	}

	commands := agent.generateCommandList(ctx)

	assert.Empty(t, commands)
}

func TestSummarization_GenerateContextDiff(t *testing.T) {
	agent := NewSummarizationAgent()

	ctx := &models.AgentContext{
		Audit: &models.AuditContext{
			AgentRuns: []models.AgentRun{
				{
					AgentID:     "intent_detection",
					KeysWritten: []string{"reasoning.intents"},
				},
				{
					AgentID:     "inference",
					KeysWritten: []string{"reasoning.conclusions"},
				},
			},
		},
	}

	diff := agent.generateContextDiff(ctx)

	assert.Contains(t, diff, "Context Changes")
	assert.Contains(t, diff, "intent_detection: reasoning.intents")
	assert.Contains(t, diff, "inference: reasoning.conclusions")
}

func TestSummarization_GenerateContextDiff_NoAudit(t *testing.T) {
	agent := NewSummarizationAgent()

	ctx := &models.AgentContext{
		Audit: nil,
	}

	diff := agent.generateContextDiff(ctx)

	assert.Empty(t, diff)
}

func TestSummarization_GenerateArtifacts(t *testing.T) {
	agent := NewSummarizationAgent()

	ctx := &models.AgentContext{
		Reasoning: &models.ReasoningContext{
			Intents: []models.Intent{
				{Type: "query_commits", Confidence: 0.9},
			},
			Conclusions: []models.Conclusion{
				{ID: "c0", Description: "Found commits", Confidence: 0.95},
			},
		},
	}

	artifacts := agent.generateArtifacts(ctx)

	assert.NotEmpty(t, artifacts)

	// Check for report artifact
	var reportFound bool
	for _, artifact := range artifacts {
		if artifact.Type == "report" {
			reportFound = true
			assert.Equal(t, "report", artifact.ID)
			assert.Equal(t, "summarization", artifact.Source)
			assert.NotEmpty(t, artifact.Content)
		}
	}
	assert.True(t, reportFound, "Report artifact not found")
}

func TestSummarization_InputContextNotModified(t *testing.T) {
	agent := NewSummarizationAgent()

	originalCtx := &models.AgentContext{
		Reasoning: &models.ReasoningContext{
			Intents: []models.Intent{
				{Type: "query_commits", Confidence: 0.9},
			},
			Conclusions: []models.Conclusion{
				{ID: "c0", Description: "Found commits", Confidence: 0.95},
			},
		},
	}

	result, err := agent.Execute(context.Background(), originalCtx)

	require.NoError(t, err)
	require.NotNil(t, result)

	// Original context should not have summary
	assert.Empty(t, originalCtx.Reasoning.Summary)

	// Result context should have summary
	assert.NotEmpty(t, result.Reasoning.Summary)
}

func TestSummarization_ExecutionMetrics(t *testing.T) {
	agent := NewSummarizationAgent()

	ctx := &models.AgentContext{
		Reasoning: &models.ReasoningContext{
			Intents: []models.Intent{
				{Type: "query_commits", Confidence: 0.9},
			},
			Conclusions: []models.Conclusion{
				{ID: "c0", Description: "Found commits", Confidence: 0.95},
			},
		},
	}

	result, err := agent.Execute(context.Background(), ctx)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Diagnostics)
	require.NotNil(t, result.Diagnostics.Performance)
	require.NotNil(t, result.Diagnostics.Performance.AgentMetrics)

	metrics, exists := result.Diagnostics.Performance.AgentMetrics["summarization"]
	assert.True(t, exists)
	assert.Equal(t, 0, metrics.LLMCalls) // No LLM calls
	assert.Equal(t, "success", metrics.Status)
	assert.GreaterOrEqual(t, metrics.DurationMS, int64(0))
}

func TestSummarization_Idempotency(t *testing.T) {
	agent := NewSummarizationAgent()

	ctx := &models.AgentContext{
		Reasoning: &models.ReasoningContext{
			Intents: []models.Intent{
				{Type: "query_commits", Confidence: 0.9},
			},
			Conclusions: []models.Conclusion{
				{ID: "c0", Description: "Found commits", Confidence: 0.95},
			},
		},
	}

	// Execute twice
	result1, err1 := agent.Execute(context.Background(), ctx)
	require.NoError(t, err1)

	ctx2 := &models.AgentContext{
		Reasoning: &models.ReasoningContext{
			Intents: []models.Intent{
				{Type: "query_commits", Confidence: 0.9},
			},
			Conclusions: []models.Conclusion{
				{ID: "c0", Description: "Found commits", Confidence: 0.95},
			},
		},
	}

	result2, err2 := agent.Execute(context.Background(), ctx2)
	require.NoError(t, err2)

	// Summaries should be identical
	assert.Equal(t, result1.Reasoning.Summary, result2.Reasoning.Summary)
}

func TestSummarization_PerformanceBenchmark(t *testing.T) {
	agent := NewSummarizationAgent()

	ctx := &models.AgentContext{
		Reasoning: &models.ReasoningContext{
			Intents: []models.Intent{
				{Type: "query_commits", Confidence: 0.9},
			},
			Hypotheses: []models.Hypothesis{
				{ID: "h0", Description: "Retrieve commits"},
			},
			Conclusions: []models.Conclusion{
				{ID: "c0", Description: "Found commits", Confidence: 0.95},
			},
		},
		Enrichment: &models.EnrichmentContext{
			Facts: make([]models.Fact, 50),
		},
	}

	start := time.Now()
	result, err := agent.Execute(context.Background(), ctx)
	duration := time.Since(start)

	require.NoError(t, err)
	require.NotNil(t, result)

	// Should complete in < 100ms
	assert.Less(t, duration.Milliseconds(), int64(100),
		"Summarization took %dms, expected < 100ms", duration.Milliseconds())
}

func TestSummarization_FullPipelineOutput(t *testing.T) {
	agent := NewSummarizationAgent()

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
					Description: "Found 3 commit(s) from GitLab",
					Confidence:  0.95,
					Intent:      "query_commits",
					Evidence:    []string{"fact:f1", "fact:f2"},
				},
			},
		},
		Enrichment: &models.EnrichmentContext{
			Facts: []models.Fact{
				{ID: "f1", Content: "Commit 1", Source: "gitlab", Confidence: 0.95},
				{ID: "f2", Content: "Commit 2", Source: "gitlab", Confidence: 0.95},
			},
		},
		Diagnostics: &models.DiagnosticsContext{
			Errors:   []models.ErrorReport{},
			Warnings: []models.Warning{},
		},
		Audit: &models.AuditContext{
			AgentRuns: []models.AgentRun{
				{AgentID: "intent_detection", KeysWritten: []string{"reasoning.intents"}},
				{AgentID: "inference", KeysWritten: []string{"reasoning.conclusions"}},
			},
		},
	}

	result, err := agent.Execute(context.Background(), ctx)

	require.NoError(t, err)
	require.NotNil(t, result)

	// Validate summary
	summary := result.Reasoning.Summary
	assert.Contains(t, summary, "query_commits")
	assert.Contains(t, summary, "2 fact(s)")
	assert.Contains(t, summary, "gitlab")
	assert.Contains(t, summary, "1 conclusion")
	assert.Contains(t, summary, "high confidence")
	assert.Contains(t, summary, "Validation passed")

	// Validate artifacts
	assert.NotEmpty(t, result.Reasoning.Artifacts)

	// Find and validate report artifact
	var report string
	for _, artifact := range result.Reasoning.Artifacts {
		if artifact.Type == "report" {
			content, ok := artifact.Content.(string)
			require.True(t, ok, "Report content should be string")
			report = content
			break
		}
	}

	assert.NotEmpty(t, report)
	assert.Contains(t, report, "# Reasoning Report")
	assert.Contains(t, report, "query_commits")
	assert.Contains(t, report, "GitLab")

	// Find and validate context diff artifact
	var diff string
	for _, artifact := range result.Reasoning.Artifacts {
		if artifact.Type == "diff" {
			content, ok := artifact.Content.(string)
			require.True(t, ok, "Diff content should be string")
			diff = content
			break
		}
	}

	assert.NotEmpty(t, diff)
	assert.Contains(t, diff, "Context Changes")
	assert.Contains(t, diff, "intent_detection")
	assert.Contains(t, diff, "inference")
}

func TestSummarization_ArtifactTypes(t *testing.T) {
	agent := NewSummarizationAgent()

	ctx := &models.AgentContext{
		Reasoning: &models.ReasoningContext{
			Intents: []models.Intent{
				{Type: "query_commits", Confidence: 0.9},
			},
			Conclusions: []models.Conclusion{
				{ID: "c0", Description: "Found commits", Confidence: 0.95},
			},
		},
		Audit: &models.AuditContext{
			AgentRuns: []models.AgentRun{
				{AgentID: "test", KeysWritten: []string{"test.key"}},
			},
		},
	}

	artifacts := agent.generateArtifacts(ctx)

	artifactTypes := make(map[string]bool)
	for _, artifact := range artifacts {
		artifactTypes[artifact.Type] = true
	}

	// Should always have report
	assert.True(t, artifactTypes["report"], "Report artifact missing")

	// Should have diff when audit exists
	assert.True(t, artifactTypes["diff"], "Diff artifact missing")
}
