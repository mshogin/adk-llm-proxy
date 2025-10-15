package agents

import (
	"context"
	"testing"

	"github.com/mshogin/agents/internal/domain/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test agent initialization
func TestNewRetrievalPlannerAgent(t *testing.T) {
	agent := NewRetrievalPlannerAgent()
	assert.NotNil(t, agent)
	assert.Equal(t, "retrieval_planner", agent.AgentID())
}

func TestRetrievalPlannerAgent_GetMetadata(t *testing.T) {
	agent := NewRetrievalPlannerAgent()
	metadata := agent.GetMetadata()

	assert.Equal(t, "retrieval_planner", metadata.ID)
	assert.Equal(t, "Retrieval Planner Agent", metadata.Name)
	assert.NotEmpty(t, metadata.Description)
	assert.Contains(t, metadata.Tags, "retrieval")
	assert.Contains(t, metadata.Tags, "planning")
	assert.Contains(t, metadata.Dependencies, "intent_detection")
	assert.Contains(t, metadata.Dependencies, "reasoning_structure")
}

func TestRetrievalPlannerAgent_GetCapabilities(t *testing.T) {
	agent := NewRetrievalPlannerAgent()
	caps := agent.GetCapabilities()

	assert.False(t, caps.SupportsParallelExecution)
	assert.True(t, caps.SupportsRetry)
	assert.False(t, caps.RequiresLLM)
	assert.True(t, caps.IsDeterministic)
	assert.Greater(t, caps.EstimatedDuration, 0)
}

func TestRetrievalPlannerAgent_Preconditions(t *testing.T) {
	agent := NewRetrievalPlannerAgent()
	preconditions := agent.Preconditions()

	assert.Contains(t, preconditions, "reasoning.intents")
	assert.Contains(t, preconditions, "reasoning.hypotheses")
}

func TestRetrievalPlannerAgent_Postconditions(t *testing.T) {
	agent := NewRetrievalPlannerAgent()
	postconditions := agent.Postconditions()

	assert.Contains(t, postconditions, "retrieval.plans")
	assert.Contains(t, postconditions, "retrieval.queries")
}

// Test precondition validation
func TestRetrievalPlanner_Execute_MissingIntents(t *testing.T) {
	agent := NewRetrievalPlannerAgent()
	ctx := models.NewAgentContext("test-session", "test-trace")

	// Add hypotheses but no intents
	ctx.Reasoning.Hypotheses = []models.Hypothesis{
		{ID: "h0", Description: "Test hypothesis"},
	}

	_, err := agent.Execute(context.Background(), ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no intents found")
}

func TestRetrievalPlanner_Execute_MissingHypotheses(t *testing.T) {
	agent := NewRetrievalPlannerAgent()
	ctx := models.NewAgentContext("test-session", "test-trace")

	// Add intents but no hypotheses
	ctx.Reasoning.Intents = []models.Intent{
		{Type: "query_commits", Confidence: 0.9},
	}

	_, err := agent.Execute(context.Background(), ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no hypotheses found")
}

// Test retrieval plan generation for query_commits intent
func TestRetrievalPlanner_Execute_QueryCommits(t *testing.T) {
	agent := NewRetrievalPlannerAgent()
	ctx := models.NewAgentContext("test-session", "test-trace")

	ctx.Reasoning.Intents = []models.Intent{
		{Type: "query_commits", Confidence: 0.9},
	}
	ctx.Reasoning.Hypotheses = []models.Hypothesis{
		{ID: "h0", Description: "Retrieve commit data"},
	}
	ctx.Reasoning.Entities = map[string]interface{}{
		"projects": []string{"gitlab-repo"},
		"dates":    []string{"last week"},
	}

	result, err := agent.Execute(context.Background(), ctx)
	require.NoError(t, err)

	// Verify plans were generated
	assert.NotEmpty(t, result.Retrieval.Plans)
	assert.Equal(t, 1, len(result.Retrieval.Plans))

	plan := result.Retrieval.Plans[0]
	assert.NotEmpty(t, plan.ID)
	assert.Contains(t, plan.Description, "GitLab")
	assert.Contains(t, plan.Sources, "gitlab")
	assert.Equal(t, 10, plan.Priority) // High priority for structured source

	// Verify filters include entities
	assert.NotNil(t, plan.Filters)
	assert.Contains(t, plan.Filters, "projects")
	assert.Contains(t, plan.Filters, "dates")

	// Verify queries were generated
	assert.NotEmpty(t, result.Retrieval.Queries)
	assert.Equal(t, 1, len(result.Retrieval.Queries))

	query := result.Retrieval.Queries[0]
	assert.NotEmpty(t, query.ID)
	assert.Contains(t, query.QueryString, "GitLab")
	assert.Equal(t, "gitlab", query.Source)
}

// Test retrieval plan generation for query_issues intent
func TestRetrievalPlanner_Execute_QueryIssues(t *testing.T) {
	agent := NewRetrievalPlannerAgent()
	ctx := models.NewAgentContext("test-session", "test-trace")

	ctx.Reasoning.Intents = []models.Intent{
		{Type: "query_issues", Confidence: 0.85},
	}
	ctx.Reasoning.Hypotheses = []models.Hypothesis{
		{ID: "h0", Description: "Retrieve issue data"},
	}
	ctx.Reasoning.Entities = map[string]interface{}{
		"projects": []string{"youtrack-project"},
		"statuses": []string{"open", "in-progress"},
	}

	result, err := agent.Execute(context.Background(), ctx)
	require.NoError(t, err)

	assert.NotEmpty(t, result.Retrieval.Plans)
	plan := result.Retrieval.Plans[0]
	assert.Contains(t, plan.Description, "YouTrack")
	assert.Contains(t, plan.Sources, "youtrack")
	assert.Equal(t, 10, plan.Priority)

	// Verify status filters are included
	assert.Contains(t, plan.Filters, "statuses")
}

// Test retrieval plan generation for query_analytics intent
func TestRetrievalPlanner_Execute_QueryAnalytics(t *testing.T) {
	agent := NewRetrievalPlannerAgent()
	ctx := models.NewAgentContext("test-session", "test-trace")

	ctx.Reasoning.Intents = []models.Intent{
		{Type: "query_analytics", Confidence: 0.8},
	}
	ctx.Reasoning.Hypotheses = []models.Hypothesis{
		{ID: "h0", Description: "Aggregate data"},
	}

	result, err := agent.Execute(context.Background(), ctx)
	require.NoError(t, err)

	// Analytics should generate multiple plans (GitLab + YouTrack)
	assert.NotEmpty(t, result.Retrieval.Plans)
	assert.GreaterOrEqual(t, len(result.Retrieval.Plans), 2)

	// Check for both GitLab and YouTrack sources
	sources := []string{}
	for _, plan := range result.Retrieval.Plans {
		if len(plan.Sources) > 0 {
			sources = append(sources, plan.Sources[0])
		}
	}
	assert.Contains(t, sources, "gitlab")
	assert.Contains(t, sources, "youtrack")
}

// Test retrieval plan generation for query_status intent
func TestRetrievalPlanner_Execute_QueryStatus(t *testing.T) {
	agent := NewRetrievalPlannerAgent()
	ctx := models.NewAgentContext("test-session", "test-trace")

	ctx.Reasoning.Intents = []models.Intent{
		{Type: "query_status", Confidence: 0.75},
	}
	ctx.Reasoning.Hypotheses = []models.Hypothesis{
		{ID: "h0", Description: "Check status"},
	}

	result, err := agent.Execute(context.Background(), ctx)
	require.NoError(t, err)

	// Status queries should generate multiple plans
	assert.NotEmpty(t, result.Retrieval.Plans)
	assert.GreaterOrEqual(t, len(result.Retrieval.Plans), 2)

	// Verify priority values
	for _, plan := range result.Retrieval.Plans {
		assert.Greater(t, plan.Priority, 0)
	}
}

// Test multiple intents
func TestRetrievalPlanner_Execute_MultipleIntents(t *testing.T) {
	agent := NewRetrievalPlannerAgent()
	ctx := models.NewAgentContext("test-session", "test-trace")

	ctx.Reasoning.Intents = []models.Intent{
		{Type: "query_commits", Confidence: 0.9},
		{Type: "query_issues", Confidence: 0.85},
	}
	ctx.Reasoning.Hypotheses = []models.Hypothesis{
		{ID: "h0", Description: "Retrieve data"},
	}

	result, err := agent.Execute(context.Background(), ctx)
	require.NoError(t, err)

	// Should have plans from both intents
	assert.NotEmpty(t, result.Retrieval.Plans)
	assert.GreaterOrEqual(t, len(result.Retrieval.Plans), 2)

	// Verify both GitLab and YouTrack sources are present
	sources := []string{}
	for _, plan := range result.Retrieval.Plans {
		if len(plan.Sources) > 0 {
			sources = append(sources, plan.Sources[0])
		}
	}
	assert.Contains(t, sources, "gitlab")
	assert.Contains(t, sources, "youtrack")
}

// Test low-confidence intent filtering
func TestRetrievalPlanner_Execute_LowConfidenceFiltered(t *testing.T) {
	agent := NewRetrievalPlannerAgent()
	ctx := models.NewAgentContext("test-session", "test-trace")

	ctx.Reasoning.Intents = []models.Intent{
		{Type: "query_commits", Confidence: 0.9},
		{Type: "query_issues", Confidence: 0.2}, // Too low
	}
	ctx.Reasoning.Hypotheses = []models.Hypothesis{
		{ID: "h0", Description: "Retrieve data"},
	}

	result, err := agent.Execute(context.Background(), ctx)
	require.NoError(t, err)

	// Should only have plans from high-confidence intent
	assert.NotEmpty(t, result.Retrieval.Plans)

	// Verify only GitLab source (from commits, not YouTrack from issues)
	for _, plan := range result.Retrieval.Plans {
		if len(plan.Sources) > 0 {
			assert.Equal(t, "gitlab", plan.Sources[0])
		}
	}
}

// Test priority sorting
func TestRetrievalPlanner_Execute_PrioritySorting(t *testing.T) {
	agent := NewRetrievalPlannerAgent()
	ctx := models.NewAgentContext("test-session", "test-trace")

	// Mix high and low priority intents
	ctx.Reasoning.Intents = []models.Intent{
		{Type: "query_status", Confidence: 0.75},   // Priority 8
		{Type: "query_commits", Confidence: 0.9},   // Priority 10
	}
	ctx.Reasoning.Hypotheses = []models.Hypothesis{
		{ID: "h0", Description: "Retrieve data"},
	}

	result, err := agent.Execute(context.Background(), ctx)
	require.NoError(t, err)

	// Plans should be sorted by priority (highest first)
	assert.NotEmpty(t, result.Retrieval.Plans)
	for i := 0; i < len(result.Retrieval.Plans)-1; i++ {
		assert.GreaterOrEqual(t, result.Retrieval.Plans[i].Priority, result.Retrieval.Plans[i+1].Priority)
	}
}

// Test query normalization
func TestRetrievalPlanner_QueryNormalization(t *testing.T) {
	agent := NewRetrievalPlannerAgent()
	ctx := models.NewAgentContext("test-session", "test-trace")

	ctx.Reasoning.Intents = []models.Intent{
		{Type: "query_commits", Confidence: 0.9},
	}
	ctx.Reasoning.Hypotheses = []models.Hypothesis{
		{ID: "h0", Description: "Retrieve data"},
	}
	ctx.Reasoning.Entities = map[string]interface{}{
		"projects": []string{"project1", "project2"},
		"dates":    []string{"2024-01-15"},
	}

	result, err := agent.Execute(context.Background(), ctx)
	require.NoError(t, err)

	assert.NotEmpty(t, result.Retrieval.Queries)
	query := result.Retrieval.Queries[0]

	// Query string should include filter details
	assert.NotEmpty(t, query.QueryString)
	assert.Contains(t, query.QueryString, "projects:")
	assert.Contains(t, query.QueryString, "dates:")

	// Query should have filters
	assert.NotNil(t, query.Filters)
	assert.Contains(t, query.Filters, "projects")
	assert.Contains(t, query.Filters, "dates")
}

// Test context isolation
func TestRetrievalPlanner_Execute_ContextIsolation(t *testing.T) {
	agent := NewRetrievalPlannerAgent()
	ctx := models.NewAgentContext("test-session", "test-trace")

	ctx.Reasoning.Intents = []models.Intent{
		{Type: "query_commits", Confidence: 0.9},
	}
	ctx.Reasoning.Hypotheses = []models.Hypothesis{
		{ID: "h0", Description: "Retrieve data"},
	}

	// Store original state
	originalPlansCount := len(ctx.Retrieval.Plans)

	result, err := agent.Execute(context.Background(), ctx)
	require.NoError(t, err)

	// Original context should not be modified
	assert.Equal(t, originalPlansCount, len(ctx.Retrieval.Plans))

	// Result should have new data
	assert.NotEmpty(t, result.Retrieval.Plans)
}

// Test audit trail
func TestRetrievalPlanner_Execute_AuditTrail(t *testing.T) {
	agent := NewRetrievalPlannerAgent()
	ctx := models.NewAgentContext("test-session", "test-trace")

	ctx.Reasoning.Intents = []models.Intent{
		{Type: "query_commits", Confidence: 0.9},
	}
	ctx.Reasoning.Hypotheses = []models.Hypothesis{
		{ID: "h0", Description: "Retrieve data"},
	}

	result, err := agent.Execute(context.Background(), ctx)
	require.NoError(t, err)

	// Verify audit trail
	require.NotEmpty(t, result.Audit.AgentRuns)
	run := result.Audit.AgentRuns[len(result.Audit.AgentRuns)-1]

	assert.Equal(t, "retrieval_planner", run.AgentID)
	assert.Equal(t, "success", run.Status)
	assert.GreaterOrEqual(t, run.DurationMS, int64(0))
	assert.Contains(t, run.KeysWritten, "retrieval.plans")
	assert.Contains(t, run.KeysWritten, "retrieval.queries")

	// Verify performance metrics
	assert.NotNil(t, result.Diagnostics.Performance)
	metrics := result.Diagnostics.Performance.AgentMetrics["retrieval_planner"]
	assert.NotNil(t, metrics)
	assert.Equal(t, "success", metrics.Status)
	assert.Equal(t, 0, metrics.LLMCalls)
}

// Test idempotency
func TestRetrievalPlanner_Execute_Idempotency(t *testing.T) {
	agent := NewRetrievalPlannerAgent()

	// Create identical contexts
	ctx1 := models.NewAgentContext("test-session", "test-trace")
	ctx1.Reasoning.Intents = []models.Intent{
		{Type: "query_commits", Confidence: 0.9},
	}
	ctx1.Reasoning.Hypotheses = []models.Hypothesis{
		{ID: "h0", Description: "Retrieve data"},
	}

	ctx2 := models.NewAgentContext("test-session", "test-trace")
	ctx2.Reasoning.Intents = []models.Intent{
		{Type: "query_commits", Confidence: 0.9},
	}
	ctx2.Reasoning.Hypotheses = []models.Hypothesis{
		{ID: "h0", Description: "Retrieve data"},
	}

	// Execute agent twice
	result1, err1 := agent.Execute(context.Background(), ctx1)
	require.NoError(t, err1)

	result2, err2 := agent.Execute(context.Background(), ctx2)
	require.NoError(t, err2)

	// Results should be identical
	assert.Equal(t, len(result1.Retrieval.Plans), len(result2.Retrieval.Plans))
	assert.Equal(t, len(result1.Retrieval.Queries), len(result2.Retrieval.Queries))

	for i := range result1.Retrieval.Plans {
		assert.Equal(t, result1.Retrieval.Plans[i].Description, result2.Retrieval.Plans[i].Description)
		assert.Equal(t, result1.Retrieval.Plans[i].Sources, result2.Retrieval.Plans[i].Sources)
		assert.Equal(t, result1.Retrieval.Plans[i].Priority, result2.Retrieval.Plans[i].Priority)
	}
}

// Benchmark plan generation
func BenchmarkRetrievalPlanner_Execute(b *testing.B) {
	agent := NewRetrievalPlannerAgent()
	ctx := models.NewAgentContext("test-session", "test-trace")
	ctx.Reasoning.Intents = []models.Intent{
		{Type: "query_commits", Confidence: 0.9},
		{Type: "query_issues", Confidence: 0.85},
	}
	ctx.Reasoning.Hypotheses = []models.Hypothesis{
		{ID: "h0", Description: "Retrieve data"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := agent.Execute(context.Background(), ctx)
		if err != nil {
			b.Fatal(err)
		}
	}
}
