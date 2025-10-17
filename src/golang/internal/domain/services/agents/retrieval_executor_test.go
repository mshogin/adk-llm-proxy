package agents

import (
	"context"
	"testing"
	"time"

	"github.com/mshogin/agents/internal/domain/models"
	"github.com/mshogin/agents/internal/infrastructure/datasources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test agent initialization
func TestNewRetrievalExecutorAgent(t *testing.T) {
	mockClient := datasources.NewMockMultiSourceClient()
	agent := NewRetrievalExecutorAgent(mockClient, 5, 30*time.Second)

	assert.NotNil(t, agent)
	assert.Equal(t, "retrieval_executor", agent.AgentID())
}

func TestRetrievalExecutorAgent_GetMetadata(t *testing.T) {
	mockClient := datasources.NewMockMultiSourceClient()
	agent := NewRetrievalExecutorAgent(mockClient, 5, 30*time.Second)
	metadata := agent.GetMetadata()

	assert.Equal(t, "retrieval_executor", metadata.ID)
	assert.Equal(t, "Retrieval Executor Agent", metadata.Name)
	assert.NotEmpty(t, metadata.Description)
	assert.Contains(t, metadata.Tags, "retrieval")
	assert.Contains(t, metadata.Tags, "execution")
	assert.Contains(t, metadata.Tags, "gitlab")
	assert.Contains(t, metadata.Tags, "youtrack")
	assert.Contains(t, metadata.Dependencies, "retrieval_planner")
}

func TestRetrievalExecutorAgent_GetCapabilities(t *testing.T) {
	mockClient := datasources.NewMockMultiSourceClient()
	agent := NewRetrievalExecutorAgent(mockClient, 5, 30*time.Second)
	caps := agent.GetCapabilities()

	assert.True(t, caps.SupportsParallelExecution)  // Supports parallel queries
	assert.True(t, caps.SupportsRetry)              // Can retry on failure
	assert.False(t, caps.RequiresLLM)               // No LLM needed
	assert.False(t, caps.IsDeterministic)           // Depends on external data
	assert.Greater(t, caps.EstimatedDuration, 0)
}

func TestRetrievalExecutorAgent_Preconditions(t *testing.T) {
	mockClient := datasources.NewMockMultiSourceClient()
	agent := NewRetrievalExecutorAgent(mockClient, 5, 30*time.Second)
	preconditions := agent.Preconditions()

	assert.Contains(t, preconditions, "retrieval.plans")
	assert.Contains(t, preconditions, "retrieval.queries")
}

func TestRetrievalExecutorAgent_Postconditions(t *testing.T) {
	mockClient := datasources.NewMockMultiSourceClient()
	agent := NewRetrievalExecutorAgent(mockClient, 5, 30*time.Second)
	postconditions := agent.Postconditions()

	assert.Contains(t, postconditions, "retrieval.artifacts")
}

// Test precondition validation
func TestRetrievalExecutor_Execute_MissingPlans(t *testing.T) {
	mockClient := datasources.NewMockMultiSourceClient()
	agent := NewRetrievalExecutorAgent(mockClient, 5, 30*time.Second)
	ctx := models.NewAgentContext("test-session", "test-trace")

	// Add queries but no plans
	ctx.Retrieval.Queries = []models.Query{
		{ID: "q1", QueryString: "test query", Source: "gitlab"},
	}

	_, err := agent.Execute(context.Background(), ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no retrieval plans found")
}

func TestRetrievalExecutor_Execute_MissingQueries(t *testing.T) {
	mockClient := datasources.NewMockMultiSourceClient()
	agent := NewRetrievalExecutorAgent(mockClient, 5, 30*time.Second)
	ctx := models.NewAgentContext("test-session", "test-trace")

	// Add plans but no queries
	ctx.Retrieval.Plans = []models.RetrievalPlan{
		{ID: "plan1", Description: "Test plan", Sources: []string{"gitlab"}},
	}

	_, err := agent.Execute(context.Background(), ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no queries found")
}

// Test successful query execution
func TestRetrievalExecutor_Execute_Success_GitLab(t *testing.T) {
	mockClient := datasources.NewMockMultiSourceClient()
	agent := NewRetrievalExecutorAgent(mockClient, 5, 30*time.Second)
	ctx := models.NewAgentContext("test-session", "test-trace")

	// Setup retrieval context
	ctx.Retrieval.Plans = []models.RetrievalPlan{
		{ID: "plan1", Description: "Fetch GitLab commits", Sources: []string{"gitlab"}, Priority: 10},
	}
	ctx.Retrieval.Queries = []models.Query{
		{ID: "plan1_query", QueryString: "Retrieve commit data from GitLab", Source: "gitlab"},
	}

	result, err := agent.Execute(context.Background(), ctx)
	require.NoError(t, err)

	// Verify artifacts were retrieved
	assert.NotEmpty(t, result.Retrieval.Artifacts)
	assert.GreaterOrEqual(t, len(result.Retrieval.Artifacts), 1)

	// Verify artifact properties
	for _, artifact := range result.Retrieval.Artifacts {
		assert.NotEmpty(t, artifact.ID)
		assert.Equal(t, "gitlab", artifact.Source)
		assert.Equal(t, "commit", artifact.Type)
		assert.NotNil(t, artifact.Content)
	}
}

func TestRetrievalExecutor_Execute_Success_YouTrack(t *testing.T) {
	mockClient := datasources.NewMockMultiSourceClient()
	agent := NewRetrievalExecutorAgent(mockClient, 5, 30*time.Second)
	ctx := models.NewAgentContext("test-session", "test-trace")

	// Setup retrieval context
	ctx.Retrieval.Plans = []models.RetrievalPlan{
		{ID: "plan1", Description: "Fetch YouTrack issues", Sources: []string{"youtrack"}, Priority: 10},
	}
	ctx.Retrieval.Queries = []models.Query{
		{ID: "plan1_query", QueryString: "Retrieve issue data from YouTrack", Source: "youtrack"},
	}

	result, err := agent.Execute(context.Background(), ctx)
	require.NoError(t, err)

	// Verify artifacts were retrieved
	assert.NotEmpty(t, result.Retrieval.Artifacts)

	// Verify artifact properties
	for _, artifact := range result.Retrieval.Artifacts {
		assert.NotEmpty(t, artifact.ID)
		assert.Equal(t, "youtrack", artifact.Source)
		assert.Equal(t, "issue", artifact.Type)
		assert.NotNil(t, artifact.Content)
	}
}

// Test multiple queries execution
func TestRetrievalExecutor_Execute_MultipleQueries(t *testing.T) {
	mockClient := datasources.NewMockMultiSourceClient()
	agent := NewRetrievalExecutorAgent(mockClient, 5, 30*time.Second)
	ctx := models.NewAgentContext("test-session", "test-trace")

	// Setup multiple retrieval queries
	ctx.Retrieval.Plans = []models.RetrievalPlan{
		{ID: "plan1", Description: "Fetch GitLab commits", Sources: []string{"gitlab"}, Priority: 10},
		{ID: "plan2", Description: "Fetch YouTrack issues", Sources: []string{"youtrack"}, Priority: 9},
	}
	ctx.Retrieval.Queries = []models.Query{
		{ID: "plan1_query", QueryString: "Retrieve commit data", Source: "gitlab"},
		{ID: "plan2_query", QueryString: "Retrieve issue data", Source: "youtrack"},
	}

	result, err := agent.Execute(context.Background(), ctx)
	require.NoError(t, err)

	// Verify artifacts from both sources
	assert.NotEmpty(t, result.Retrieval.Artifacts)
	assert.GreaterOrEqual(t, len(result.Retrieval.Artifacts), 2)

	// Verify both sources are present
	sources := make(map[string]bool)
	for _, artifact := range result.Retrieval.Artifacts {
		sources[artifact.Source] = true
	}
	assert.True(t, sources["gitlab"])
	assert.True(t, sources["youtrack"])
}

// Test error handling
func TestRetrievalExecutor_Execute_PartialFailure(t *testing.T) {
	// Create mock client with one failing source
	mockClient := datasources.NewMultiSourceClient()
	mockClient.RegisterClient(datasources.NewMockGitLabClient())
	failingClient := datasources.NewMockDataSourceClient("failing-source").WithFailure(true)
	mockClient.RegisterClient(failingClient)

	agent := NewRetrievalExecutorAgent(mockClient, 5, 30*time.Second)
	ctx := models.NewAgentContext("test-session", "test-trace")

	// Setup queries for both working and failing sources
	ctx.Retrieval.Plans = []models.RetrievalPlan{
		{ID: "plan1", Description: "Test", Sources: []string{"gitlab"}, Priority: 10},
		{ID: "plan2", Description: "Test", Sources: []string{"failing-source"}, Priority: 9},
	}
	ctx.Retrieval.Queries = []models.Query{
		{ID: "q1", QueryString: "query 1", Source: "gitlab"},
		{ID: "q2", QueryString: "query 2", Source: "failing-source"},
	}

	result, err := agent.Execute(context.Background(), ctx)
	require.NoError(t, err) // Agent should not fail completely

	// Verify we got artifacts from successful source
	assert.NotEmpty(t, result.Retrieval.Artifacts)

	// Verify error was logged
	assert.NotEmpty(t, result.Diagnostics.Errors)
	foundError := false
	for _, errReport := range result.Diagnostics.Errors {
		if errReport.AgentID == "retrieval_executor" {
			foundError = true
			assert.Contains(t, errReport.Message, "failed")
			assert.Equal(t, "warning", errReport.Severity)
		}
	}
	assert.True(t, foundError, "Expected error to be logged in diagnostics")
}

// Test unknown source handling (graceful degradation)
func TestRetrievalExecutor_Execute_UnknownSource(t *testing.T) {
	mockClient := datasources.NewMockMultiSourceClient()
	agent := NewRetrievalExecutorAgent(mockClient, 5, 30*time.Second)
	ctx := models.NewAgentContext("test-session", "test-trace")

	// Query for unknown source
	ctx.Retrieval.Plans = []models.RetrievalPlan{
		{ID: "plan1", Description: "Test", Sources: []string{"unknown"}, Priority: 10},
	}
	ctx.Retrieval.Queries = []models.Query{
		{ID: "q1", QueryString: "query", Source: "unknown-source"},
	}

	result, err := agent.Execute(context.Background(), ctx)
	require.NoError(t, err) // Should not fail

	// Artifacts may be empty but agent succeeds
	assert.NotNil(t, result.Retrieval.Artifacts)
}

// Test context isolation
func TestRetrievalExecutor_Execute_ContextIsolation(t *testing.T) {
	mockClient := datasources.NewMockMultiSourceClient()
	agent := NewRetrievalExecutorAgent(mockClient, 5, 30*time.Second)
	ctx := models.NewAgentContext("test-session", "test-trace")

	ctx.Retrieval.Plans = []models.RetrievalPlan{
		{ID: "plan1", Description: "Test", Sources: []string{"gitlab"}, Priority: 10},
	}
	ctx.Retrieval.Queries = []models.Query{
		{ID: "q1", QueryString: "query", Source: "gitlab"},
	}

	// Store original state
	originalArtifactsCount := len(ctx.Retrieval.Artifacts)

	result, err := agent.Execute(context.Background(), ctx)
	require.NoError(t, err)

	// Original context should not be modified
	assert.Equal(t, originalArtifactsCount, len(ctx.Retrieval.Artifacts))

	// Result should have new artifacts
	assert.NotEmpty(t, result.Retrieval.Artifacts)
}

// Test audit trail
func TestRetrievalExecutor_Execute_AuditTrail(t *testing.T) {
	mockClient := datasources.NewMockMultiSourceClient()
	agent := NewRetrievalExecutorAgent(mockClient, 5, 30*time.Second)
	ctx := models.NewAgentContext("test-session", "test-trace")

	ctx.Retrieval.Plans = []models.RetrievalPlan{
		{ID: "plan1", Description: "Test", Sources: []string{"gitlab"}, Priority: 10},
	}
	ctx.Retrieval.Queries = []models.Query{
		{ID: "q1", QueryString: "query", Source: "gitlab"},
	}

	result, err := agent.Execute(context.Background(), ctx)
	require.NoError(t, err)

	// Verify audit trail
	require.NotEmpty(t, result.Audit.AgentRuns)
	run := result.Audit.AgentRuns[len(result.Audit.AgentRuns)-1]

	assert.Equal(t, "retrieval_executor", run.AgentID)
	assert.Equal(t, "success", run.Status)
	assert.GreaterOrEqual(t, run.DurationMS, int64(0))
	assert.Contains(t, run.KeysWritten, "retrieval.artifacts")

	// Verify performance metrics
	assert.NotNil(t, result.Diagnostics.Performance)
	metrics := result.Diagnostics.Performance.AgentMetrics["retrieval_executor"]
	assert.NotNil(t, metrics)
	assert.Equal(t, "success", metrics.Status)
	assert.Equal(t, 0, metrics.LLMCalls) // No LLM calls
}

// Test parallel execution (multiple queries run concurrently)
func TestRetrievalExecutor_Execute_ParallelExecution(t *testing.T) {
	mockClient := datasources.NewMockMultiSourceClient()
	agent := NewRetrievalExecutorAgent(mockClient, 5, 30*time.Second)
	ctx := models.NewAgentContext("test-session", "test-trace")

	// Create many queries to test parallelism
	ctx.Retrieval.Plans = []models.RetrievalPlan{
		{ID: "plan1", Description: "Test", Sources: []string{"gitlab"}, Priority: 10},
	}

	numQueries := 10
	for i := 0; i < numQueries; i++ {
		ctx.Retrieval.Queries = append(ctx.Retrieval.Queries, models.Query{
			ID:          "q" + string(rune(i)),
			QueryString: "test query",
			Source:      "gitlab",
		})
	}

	startTime := time.Now()
	result, err := agent.Execute(context.Background(), ctx)
	duration := time.Since(startTime)

	require.NoError(t, err)
	assert.NotEmpty(t, result.Retrieval.Artifacts)

	// Parallel execution should be faster than sequential
	// (Each mock query is instant, but parallelism proves concurrency works)
	assert.Less(t, duration, 5*time.Second, "Parallel execution took too long")
}

// Test timeout handling
func TestRetrievalExecutor_Execute_TimeoutHandling(t *testing.T) {
	mockClient := datasources.NewMockMultiSourceClient()
	// Set very short timeout
	agent := NewRetrievalExecutorAgent(mockClient, 5, 1*time.Nanosecond)
	ctx := models.NewAgentContext("test-session", "test-trace")

	ctx.Retrieval.Plans = []models.RetrievalPlan{
		{ID: "plan1", Description: "Test", Sources: []string{"gitlab"}, Priority: 10},
	}
	ctx.Retrieval.Queries = []models.Query{
		{ID: "q1", QueryString: "query", Source: "gitlab"},
	}

	result, err := agent.Execute(context.Background(), ctx)
	require.NoError(t, err) // Agent should not fail on timeout

	// May have empty artifacts due to timeout, but no crash
	assert.NotNil(t, result.Retrieval.Artifacts)
}

// Test concurrency limit
func TestRetrievalExecutor_ConcurrencyLimit(t *testing.T) {
	mockClient := datasources.NewMockMultiSourceClient()
	// Set max concurrency to 2
	agent := NewRetrievalExecutorAgent(mockClient, 2, 30*time.Second)
	ctx := models.NewAgentContext("test-session", "test-trace")

	ctx.Retrieval.Plans = []models.RetrievalPlan{
		{ID: "plan1", Description: "Test", Sources: []string{"gitlab"}, Priority: 10},
	}

	// Create 5 queries (more than concurrency limit)
	for i := 0; i < 5; i++ {
		ctx.Retrieval.Queries = append(ctx.Retrieval.Queries, models.Query{
			ID:          "q" + string(rune(i)),
			QueryString: "test",
			Source:      "gitlab",
		})
	}

	result, err := agent.Execute(context.Background(), ctx)
	require.NoError(t, err)

	// All queries should complete despite concurrency limit
	assert.NotEmpty(t, result.Retrieval.Artifacts)
}

// Benchmark query execution
func BenchmarkRetrievalExecutor_Execute(b *testing.B) {
	mockClient := datasources.NewMockMultiSourceClient()
	agent := NewRetrievalExecutorAgent(mockClient, 5, 30*time.Second)

	ctx := models.NewAgentContext("test-session", "test-trace")
	ctx.Retrieval.Plans = []models.RetrievalPlan{
		{ID: "plan1", Description: "Test", Sources: []string{"gitlab"}, Priority: 10},
		{ID: "plan2", Description: "Test", Sources: []string{"youtrack"}, Priority: 9},
	}
	ctx.Retrieval.Queries = []models.Query{
		{ID: "q1", QueryString: "query 1", Source: "gitlab"},
		{ID: "q2", QueryString: "query 2", Source: "youtrack"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := agent.Execute(context.Background(), ctx)
		if err != nil {
			b.Fatal(err)
		}
	}
}
