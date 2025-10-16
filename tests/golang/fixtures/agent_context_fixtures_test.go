package fixtures

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEmptyContext(t *testing.T) {
	ctx := EmptyContext()

	assert.Equal(t, "1.0.0", ctx.Version)
	assert.Equal(t, "session-empty", ctx.Metadata.SessionID)
	assert.Equal(t, "trace-empty", ctx.Metadata.TraceID)
	assert.Equal(t, FixedTime, ctx.Metadata.CreatedAt)

	// All namespaces should be initialized but empty
	assert.NotNil(t, ctx.Reasoning)
	assert.NotNil(t, ctx.Enrichment)
	assert.NotNil(t, ctx.Retrieval)
	assert.NotNil(t, ctx.LLM)
	assert.NotNil(t, ctx.Diagnostics)
	assert.NotNil(t, ctx.Audit)

	// Should have no data
	assert.Empty(t, ctx.Reasoning.Intents)
	assert.Empty(t, ctx.Enrichment.Facts)
	assert.Empty(t, ctx.Audit.AgentRuns)
}

func TestMinimalContext(t *testing.T) {
	ctx := MinimalContext()

	assert.Equal(t, "1.0.0", ctx.Version)
	assert.Equal(t, "session-minimal", ctx.Metadata.SessionID)
	assert.Equal(t, "en-US", ctx.Metadata.Locale)
}

func TestContextWithIntent(t *testing.T) {
	ctx := ContextWithIntent()

	require.Len(t, ctx.Reasoning.Intents, 1)
	intent := ctx.Reasoning.Intents[0]

	assert.Equal(t, "query", intent.Type)
	assert.Equal(t, 0.95, intent.Confidence)
	assert.Contains(t, intent.Entities, "user_id")
	assert.Contains(t, intent.Entities, "timeframe")

	assert.Contains(t, ctx.Reasoning.Entities, "user_id")
	assert.Equal(t, "user123", ctx.Reasoning.Entities["user_id"])
}

func TestContextWithMultipleIntents(t *testing.T) {
	ctx := ContextWithMultipleIntents()

	require.Len(t, ctx.Reasoning.Intents, 3)

	// Verify intents are sorted by confidence (highest first)
	assert.Equal(t, "query", ctx.Reasoning.Intents[0].Type)
	assert.Equal(t, 0.85, ctx.Reasoning.Intents[0].Confidence)

	assert.Equal(t, "command", ctx.Reasoning.Intents[1].Type)
	assert.Equal(t, 0.75, ctx.Reasoning.Intents[1].Confidence)

	assert.Equal(t, "clarification", ctx.Reasoning.Intents[2].Type)
	assert.Equal(t, 0.40, ctx.Reasoning.Intents[2].Confidence)
}

func TestContextWithReasoningChain(t *testing.T) {
	ctx := ContextWithReasoningChain()

	// Check hypotheses
	require.Len(t, ctx.Reasoning.Hypotheses, 2)
	assert.Equal(t, "hyp-1", ctx.Reasoning.Hypotheses[0].ID)
	assert.Empty(t, ctx.Reasoning.Hypotheses[0].Dependencies)
	assert.Equal(t, "hyp-2", ctx.Reasoning.Hypotheses[1].ID)
	assert.Contains(t, ctx.Reasoning.Hypotheses[1].Dependencies, "hyp-1")

	// Check inference chain
	require.Len(t, ctx.Reasoning.InferenceChain, 2)
	assert.Equal(t, "inf-1", ctx.Reasoning.InferenceChain[0].ID)
	assert.Equal(t, "hyp-1", ctx.Reasoning.InferenceChain[0].Hypothesis)
	assert.Equal(t, 0.90, ctx.Reasoning.InferenceChain[0].Confidence)

	// Check conclusions
	require.Len(t, ctx.Reasoning.Conclusions, 1)
	assert.Equal(t, "conc-1", ctx.Reasoning.Conclusions[0].ID)
	assert.Equal(t, 0.90, ctx.Reasoning.Conclusions[0].Confidence)
	assert.Contains(t, ctx.Reasoning.Conclusions[0].Evidence, "inf-1")
	assert.Contains(t, ctx.Reasoning.Conclusions[0].Evidence, "inf-2")

	// Check confidence scores
	assert.Equal(t, 0.95, ctx.Reasoning.ConfidenceScores["intent_detection"])
	assert.Equal(t, 0.88, ctx.Reasoning.ConfidenceScores["reasoning"])

	// Check summary
	assert.Contains(t, ctx.Reasoning.Summary, "user123")
	assert.Contains(t, ctx.Reasoning.Summary, "last week")
}

func TestContextWithAlternatives(t *testing.T) {
	ctx := ContextWithAlternatives()

	require.Len(t, ctx.Reasoning.Alternatives, 2)

	alt1 := ctx.Reasoning.Alternatives[0]
	assert.Equal(t, "alt-1", alt1.ID)
	assert.Equal(t, "conc-1", alt1.Conclusion)
	assert.Equal(t, 0.65, alt1.Confidence)
	assert.Contains(t, alt1.Description, "current week")

	alt2 := ctx.Reasoning.Alternatives[1]
	assert.Equal(t, 0.55, alt2.Confidence)
	assert.Contains(t, alt2.Description, "real-time")
}

func TestContextWithEnrichment(t *testing.T) {
	ctx := ContextWithEnrichment()

	// Check facts
	require.Len(t, ctx.Enrichment.Facts, 2)
	fact1 := ctx.Enrichment.Facts[0]
	assert.Equal(t, "fact-1", fact1.ID)
	assert.Equal(t, "user_database", fact1.Source)
	assert.Equal(t, 1.0, fact1.Confidence)
	assert.Contains(t, fact1.Provenance, "table")

	// Check derived knowledge
	require.Len(t, ctx.Enrichment.DerivedKnowledge, 2)
	know1 := ctx.Enrichment.DerivedKnowledge[0]
	assert.Equal(t, "know-1", know1.ID)
	assert.Contains(t, know1.DerivedFrom, "fact-1")

	// Check relationships
	require.Len(t, ctx.Enrichment.Relationships, 2)
	rel1 := ctx.Enrichment.Relationships[0]
	assert.Equal(t, "user123", rel1.From)
	assert.Equal(t, "premium_tier", rel1.To)
	assert.Equal(t, "member_of", rel1.Type)

	// Check context links
	assert.Contains(t, ctx.Enrichment.ContextLinks, "session:session-reasoning")
	assert.Contains(t, ctx.Enrichment.ContextLinks, "user:user123")
}

func TestContextWithRetrieval(t *testing.T) {
	ctx := ContextWithRetrieval()

	// Check retrieval plans
	require.Len(t, ctx.Retrieval.Plans, 2)
	plan1 := ctx.Retrieval.Plans[0]
	assert.Equal(t, "plan-1", plan1.ID)
	assert.Equal(t, 1, plan1.Priority)
	assert.Contains(t, plan1.Sources, "activity_database")
	assert.Contains(t, plan1.Filters, "user_id")
	assert.Equal(t, "user123", plan1.Filters["user_id"])

	// Check queries
	require.Len(t, ctx.Retrieval.Queries, 1)
	query1 := ctx.Retrieval.Queries[0]
	assert.Equal(t, "query-1", query1.ID)
	assert.Equal(t, "activity_database", query1.Source)
	assert.Contains(t, query1.QueryString, "SELECT")
	assert.NotNil(t, query1.Results)

	// Check artifacts
	require.Len(t, ctx.Retrieval.Artifacts, 1)
	artifact1 := ctx.Retrieval.Artifacts[0]
	assert.Equal(t, "artifact-1", artifact1.ID)
	assert.Equal(t, "query_results", artifact1.Type)
}

func TestContextWithLLMUsage(t *testing.T) {
	ctx := ContextWithLLMUsage()

	assert.Equal(t, "openai", ctx.LLM.Provider)
	assert.Equal(t, "gpt-4", ctx.LLM.Model)

	// Check usage
	require.NotNil(t, ctx.LLM.Usage)
	assert.Equal(t, 5430, ctx.LLM.Usage.TotalTokens)
	assert.Equal(t, 3200, ctx.LLM.Usage.PromptTokens)
	assert.Equal(t, 2230, ctx.LLM.Usage.CompletionTokens)
	assert.InDelta(t, 0.1629, ctx.LLM.Usage.CostUSD, 0.0001)

	// Check per-agent costs
	assert.Contains(t, ctx.LLM.Usage.ByAgent, "intent_detection")
	assert.InDelta(t, 0.0320, ctx.LLM.Usage.ByAgent["intent_detection"], 0.0001)

	// Check LLM decisions
	require.Len(t, ctx.LLM.Decisions, 2)
	decision1 := ctx.LLM.Decisions[0]
	assert.Equal(t, "intent_detection", decision1.AgentID)
	assert.Equal(t, "gpt-3.5-turbo", decision1.Selected)
	assert.Equal(t, "low", decision1.Complexity)
	assert.Contains(t, decision1.Reason, "cost optimization")

	// Check cache
	assert.Contains(t, ctx.LLM.Cache, "intent:query:user_metrics")
}

func TestContextWithErrors(t *testing.T) {
	ctx := ContextWithErrors()

	// Check errors
	require.Len(t, ctx.Diagnostics.Errors, 2)
	err1 := ctx.Diagnostics.Errors[0]
	assert.Equal(t, "intent_detection", err1.AgentID)
	assert.Equal(t, "high", err1.Severity)
	assert.Contains(t, err1.Message, "Failed to parse")

	err2 := ctx.Diagnostics.Errors[1]
	assert.Equal(t, "critical", err2.Severity)
	assert.Contains(t, err2.Message, "Missing required entities")

	// Check warnings
	require.Len(t, ctx.Diagnostics.Warnings, 2)
	warn1 := ctx.Diagnostics.Warnings[0]
	assert.Equal(t, "intent_detection", warn1.AgentID)
	assert.Contains(t, warn1.Message, "Low confidence")
}

func TestContextWithValidationIssues(t *testing.T) {
	ctx := ContextWithValidationIssues()

	require.Len(t, ctx.Diagnostics.ValidationReports, 2)

	// Check failed validation
	report1 := ctx.Diagnostics.ValidationReports[0]
	assert.False(t, report1.Passed)
	assert.Len(t, report1.Issues, 2)
	assert.Contains(t, report1.Issues[0], "Confidence score below threshold")
	assert.Len(t, report1.AutoFixes, 1)
	assert.Contains(t, report1.AutoFixes[0], "clarification")

	// Check passed validation
	report2 := ctx.Diagnostics.ValidationReports[1]
	assert.True(t, report2.Passed)
	assert.Empty(t, report2.Issues)
}

func TestContextWithPerformanceMetrics(t *testing.T) {
	ctx := ContextWithPerformanceMetrics()

	require.NotNil(t, ctx.Diagnostics.Performance)
	assert.Equal(t, int64(3420), ctx.Diagnostics.Performance.TotalDurationMS)

	// Check agent metrics
	require.Len(t, ctx.Diagnostics.Performance.AgentMetrics, 3)

	intentMetrics := ctx.Diagnostics.Performance.AgentMetrics["intent_detection"]
	require.NotNil(t, intentMetrics)
	assert.Equal(t, int64(245), intentMetrics.DurationMS)
	assert.Equal(t, 1, intentMetrics.LLMCalls)
	assert.Equal(t, "completed", intentMetrics.Status)
	assert.Equal(t, 856, intentMetrics.Tokens)
	assert.InDelta(t, 0.0086, intentMetrics.Cost, 0.0001)

	reasoningMetrics := ctx.Diagnostics.Performance.AgentMetrics["reasoning"]
	assert.Equal(t, int64(1850), reasoningMetrics.DurationMS)
	assert.Equal(t, 3, reasoningMetrics.LLMCalls)
}

func TestContextWithAuditTrail(t *testing.T) {
	ctx := ContextWithAuditTrail()

	// Check agent runs
	require.Len(t, ctx.Audit.AgentRuns, 3)
	run1 := ctx.Audit.AgentRuns[0]
	assert.Equal(t, "intent_detection", run1.AgentID)
	assert.Equal(t, "completed", run1.Status)
	assert.Equal(t, int64(245), run1.DurationMS)
	assert.Contains(t, run1.KeysWritten, "reasoning.intents")
	assert.Empty(t, run1.Error)

	// Check diffs
	require.Len(t, ctx.Audit.Diffs, 2)
	diff1 := ctx.Audit.Diffs[0]
	assert.Equal(t, "intent_detection", diff1.AgentID)
	assert.Contains(t, diff1.Changes, "reasoning.intents")
}

func TestContextWithFailedAgentRun(t *testing.T) {
	ctx := ContextWithFailedAgentRun()

	require.Len(t, ctx.Audit.AgentRuns, 2)

	// Check successful run
	run1 := ctx.Audit.AgentRuns[0]
	assert.Equal(t, "completed", run1.Status)
	assert.Empty(t, run1.Error)

	// Check failed run
	run2 := ctx.Audit.AgentRuns[1]
	assert.Equal(t, "failed", run2.Status)
	assert.NotEmpty(t, run2.Error)
	assert.Contains(t, run2.Error, "confidence threshold")
	assert.Empty(t, run2.KeysWritten)

	// Check error was recorded
	require.Len(t, ctx.Diagnostics.Errors, 1)
	assert.Equal(t, "reasoning", ctx.Diagnostics.Errors[0].AgentID)
	assert.Equal(t, "critical", ctx.Diagnostics.Errors[0].Severity)
}

func TestComplexContext(t *testing.T) {
	ctx := ComplexContext()

	// Verify all namespaces are populated
	assert.NotEmpty(t, ctx.Reasoning.Intents)
	assert.NotEmpty(t, ctx.Reasoning.Hypotheses)
	assert.NotEmpty(t, ctx.Reasoning.Conclusions)
	assert.NotEmpty(t, ctx.Enrichment.Facts)
	assert.NotEmpty(t, ctx.Enrichment.DerivedKnowledge)
	assert.NotEmpty(t, ctx.Retrieval.Plans)
	assert.NotEmpty(t, ctx.Retrieval.Queries)
	assert.Greater(t, ctx.LLM.Usage.TotalTokens, 0)
	assert.NotEmpty(t, ctx.LLM.Decisions)
	assert.Greater(t, ctx.Diagnostics.Performance.TotalDurationMS, int64(0))
	assert.NotEmpty(t, ctx.Audit.AgentRuns)
	assert.NotEmpty(t, ctx.Diagnostics.ValidationReports)

	// Verify it's a complete, valid context
	assert.Equal(t, "session-complex", ctx.Metadata.SessionID)
	assert.Equal(t, "en-US", ctx.Metadata.Locale)
}

func TestContextWithArtifacts(t *testing.T) {
	ctx := ContextWithArtifacts()

	// Check reasoning artifacts
	require.Len(t, ctx.Reasoning.Artifacts, 2)
	artifact1 := ctx.Reasoning.Artifacts[0]
	assert.Equal(t, "summary-1", artifact1.ID)
	assert.Equal(t, "text_summary", artifact1.Type)
	assert.Equal(t, "summarization_agent", artifact1.Source)

	// Check retrieval artifacts
	require.Len(t, ctx.Retrieval.Artifacts, 1)
	artifact2 := ctx.Retrieval.Artifacts[0]
	assert.Equal(t, "data-1", artifact2.ID)
	assert.Equal(t, "query_results", artifact2.Type)
}

func TestContextSerialization(t *testing.T) {
	// Test that fixtures can be serialized and deserialized
	original := ComplexContext()

	// Serialize
	data, err := original.Serialize()
	require.NoError(t, err)
	assert.NotEmpty(t, data)

	// Deserialize
	deserialized, err := original.Clone()
	require.NoError(t, err)
	require.NotNil(t, deserialized)

	// Verify key fields match
	assert.Equal(t, original.Version, deserialized.Version)
	assert.Equal(t, original.Metadata.SessionID, deserialized.Metadata.SessionID)
	assert.Equal(t, len(original.Reasoning.Intents), len(deserialized.Reasoning.Intents))
	assert.Equal(t, len(original.Audit.AgentRuns), len(deserialized.Audit.AgentRuns))
}

func TestFixtureImmutability(t *testing.T) {
	// Verify that calling the same fixture multiple times returns independent instances
	ctx1 := EmptyContext()
	ctx2 := EmptyContext()

	// Modify ctx1
	ctx1.Metadata.SessionID = "modified"

	// Verify ctx2 is unchanged
	assert.Equal(t, "session-empty", ctx2.Metadata.SessionID)
}
