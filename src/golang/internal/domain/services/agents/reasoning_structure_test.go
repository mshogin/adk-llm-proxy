package agents

import (
	"context"
	"testing"

	"github.com/mshogin/agents/internal/domain/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test agent initialization and metadata
func TestNewReasoningStructureAgent(t *testing.T) {
	agent := NewReasoningStructureAgent()
	assert.NotNil(t, agent)
	assert.Equal(t, "reasoning_structure", agent.AgentID())
}

func TestReasoningStructureAgent_GetMetadata(t *testing.T) {
	agent := NewReasoningStructureAgent()
	metadata := agent.GetMetadata()

	assert.Equal(t, "reasoning_structure", metadata.ID)
	assert.Equal(t, "Reasoning Structure Agent", metadata.Name)
	assert.NotEmpty(t, metadata.Description)
	assert.Equal(t, "1.0.0", metadata.Version)
	assert.Contains(t, metadata.Tags, "reasoning")
	assert.Contains(t, metadata.Tags, "hypotheses")
	assert.Contains(t, metadata.Dependencies, "intent_detection")
}

func TestReasoningStructureAgent_GetCapabilities(t *testing.T) {
	agent := NewReasoningStructureAgent()
	caps := agent.GetCapabilities()

	assert.False(t, caps.SupportsParallelExecution)
	assert.True(t, caps.SupportsRetry)
	assert.False(t, caps.RequiresLLM)
	assert.True(t, caps.IsDeterministic)
	assert.Greater(t, caps.EstimatedDuration, 0) // EstimatedDuration is int, not int64
}

func TestReasoningStructureAgent_Preconditions(t *testing.T) {
	agent := NewReasoningStructureAgent()
	preconditions := agent.Preconditions()

	assert.Contains(t, preconditions, "reasoning.intents")
}

func TestReasoningStructureAgent_Postconditions(t *testing.T) {
	agent := NewReasoningStructureAgent()
	postconditions := agent.Postconditions()

	assert.Contains(t, postconditions, "reasoning.hypotheses")
	assert.Contains(t, postconditions, "reasoning.dependency_map")
}

// Test precondition validation
func TestValidatePreconditions_MissingIntents(t *testing.T) {
	agent := NewReasoningStructureAgent()
	ctx := models.NewAgentContext("test-session", "test-trace")

	// Clear intents
	ctx.Reasoning.Intents = []models.Intent{}

	_, err := agent.Execute(context.Background(), ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no intents found")
}

func TestValidatePreconditions_NilReasoningContext(t *testing.T) {
	agent := NewReasoningStructureAgent()
	ctx := models.NewAgentContext("test-session", "test-trace")
	ctx.Reasoning = nil

	_, err := agent.Execute(context.Background(), ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "reasoning context is nil")
}

// Test hypothesis generation for query_commits intent
func TestExecute_QueryCommitsIntent(t *testing.T) {
	agent := NewReasoningStructureAgent()
	ctx := models.NewAgentContext("test-session", "test-trace")

	// Add query_commits intent
	ctx.Reasoning.Intents = []models.Intent{
		{Type: "query_commits", Confidence: 0.9},
	}

	result, err := agent.Execute(context.Background(), ctx)
	require.NoError(t, err)

	// Verify hypotheses were generated
	assert.NotEmpty(t, result.Reasoning.Hypotheses)
	assert.GreaterOrEqual(t, len(result.Reasoning.Hypotheses), 3)

	// Check for expected hypothesis descriptions
	descriptions := []string{}
	for _, h := range result.Reasoning.Hypotheses {
		descriptions = append(descriptions, h.Description)
	}

	assert.Contains(t, descriptions, "Retrieve commit data from GitLab")
	assert.Contains(t, descriptions, "Filter and rank commits by relevance")
	assert.Contains(t, descriptions, "Format commit summary for user")

	// Verify dependency graph
	assert.NotNil(t, result.Reasoning.DependencyMap)

	// Verify audit trail
	assert.NotEmpty(t, result.Audit.AgentRuns)
	assert.Equal(t, "reasoning_structure", result.Audit.AgentRuns[0].AgentID)
	assert.Equal(t, "success", result.Audit.AgentRuns[0].Status)
}

// Test hypothesis generation for query_issues intent
func TestExecute_QueryIssuesIntent(t *testing.T) {
	agent := NewReasoningStructureAgent()
	ctx := models.NewAgentContext("test-session", "test-trace")

	ctx.Reasoning.Intents = []models.Intent{
		{Type: "query_issues", Confidence: 0.85},
	}

	result, err := agent.Execute(context.Background(), ctx)
	require.NoError(t, err)

	assert.NotEmpty(t, result.Reasoning.Hypotheses)
	assert.GreaterOrEqual(t, len(result.Reasoning.Hypotheses), 3)

	descriptions := []string{}
	for _, h := range result.Reasoning.Hypotheses {
		descriptions = append(descriptions, h.Description)
	}

	assert.Contains(t, descriptions, "Retrieve issue data from YouTrack")
	assert.Contains(t, descriptions, "Apply filters based on entities (status, date, project)")
	assert.Contains(t, descriptions, "Format issue summary for user")
}

// Test hypothesis generation for query_analytics intent
func TestExecute_QueryAnalyticsIntent(t *testing.T) {
	agent := NewReasoningStructureAgent()
	ctx := models.NewAgentContext("test-session", "test-trace")

	ctx.Reasoning.Intents = []models.Intent{
		{Type: "query_analytics", Confidence: 0.8},
	}

	result, err := agent.Execute(context.Background(), ctx)
	require.NoError(t, err)

	assert.NotEmpty(t, result.Reasoning.Hypotheses)
	assert.GreaterOrEqual(t, len(result.Reasoning.Hypotheses), 4)

	descriptions := []string{}
	for _, h := range result.Reasoning.Hypotheses {
		descriptions = append(descriptions, h.Description)
	}

	assert.Contains(t, descriptions, "Identify relevant data sources (commits, issues, metrics)")
	assert.Contains(t, descriptions, "Aggregate data from multiple sources")
	assert.Contains(t, descriptions, "Calculate statistics and trends")
	assert.Contains(t, descriptions, "Generate analytics report with visualizations")
}

// Test hypothesis generation for query_status intent
func TestExecute_QueryStatusIntent(t *testing.T) {
	agent := NewReasoningStructureAgent()
	ctx := models.NewAgentContext("test-session", "test-trace")

	ctx.Reasoning.Intents = []models.Intent{
		{Type: "query_status", Confidence: 0.75},
	}

	result, err := agent.Execute(context.Background(), ctx)
	require.NoError(t, err)

	assert.NotEmpty(t, result.Reasoning.Hypotheses)
	assert.GreaterOrEqual(t, len(result.Reasoning.Hypotheses), 3)

	descriptions := []string{}
	for _, h := range result.Reasoning.Hypotheses {
		descriptions = append(descriptions, h.Description)
	}

	assert.Contains(t, descriptions, "Check system health and status")
	assert.Contains(t, descriptions, "Gather recent events and changes")
	assert.Contains(t, descriptions, "Synthesize overall status report")
}

// Test hypothesis generation for command_action intent
func TestExecute_CommandActionIntent(t *testing.T) {
	agent := NewReasoningStructureAgent()
	ctx := models.NewAgentContext("test-session", "test-trace")

	ctx.Reasoning.Intents = []models.Intent{
		{Type: "command_action", Confidence: 0.9},
	}

	result, err := agent.Execute(context.Background(), ctx)
	require.NoError(t, err)

	assert.NotEmpty(t, result.Reasoning.Hypotheses)
	assert.GreaterOrEqual(t, len(result.Reasoning.Hypotheses), 3)

	descriptions := []string{}
	for _, h := range result.Reasoning.Hypotheses {
		descriptions = append(descriptions, h.Description)
	}

	assert.Contains(t, descriptions, "Validate command preconditions and permissions")
	assert.Contains(t, descriptions, "Execute command action")
	assert.Contains(t, descriptions, "Verify command execution result")
}

// Test hypothesis generation for request_help intent
func TestExecute_RequestHelpIntent(t *testing.T) {
	agent := NewReasoningStructureAgent()
	ctx := models.NewAgentContext("test-session", "test-trace")

	ctx.Reasoning.Intents = []models.Intent{
		{Type: "request_help", Confidence: 0.7},
	}

	result, err := agent.Execute(context.Background(), ctx)
	require.NoError(t, err)

	assert.NotEmpty(t, result.Reasoning.Hypotheses)
	assert.GreaterOrEqual(t, len(result.Reasoning.Hypotheses), 3)

	descriptions := []string{}
	for _, h := range result.Reasoning.Hypotheses {
		descriptions = append(descriptions, h.Description)
	}

	assert.Contains(t, descriptions, "Identify help topic and user context")
	assert.Contains(t, descriptions, "Retrieve relevant documentation and examples")
	assert.Contains(t, descriptions, "Format helpful response with examples")
}

// Test hypothesis generation for conversation intent
func TestExecute_ConversationIntent(t *testing.T) {
	agent := NewReasoningStructureAgent()
	ctx := models.NewAgentContext("test-session", "test-trace")

	ctx.Reasoning.Intents = []models.Intent{
		{Type: "conversation", Confidence: 0.6},
	}

	result, err := agent.Execute(context.Background(), ctx)
	require.NoError(t, err)

	assert.NotEmpty(t, result.Reasoning.Hypotheses)
	assert.GreaterOrEqual(t, len(result.Reasoning.Hypotheses), 1)

	descriptions := []string{}
	for _, h := range result.Reasoning.Hypotheses {
		descriptions = append(descriptions, h.Description)
	}

	assert.Contains(t, descriptions, "Generate appropriate conversational response")
}

// Test hypothesis generation for unknown intent type
func TestExecute_UnknownIntent(t *testing.T) {
	agent := NewReasoningStructureAgent()
	ctx := models.NewAgentContext("test-session", "test-trace")

	ctx.Reasoning.Intents = []models.Intent{
		{Type: "unknown_type", Confidence: 0.5},
	}

	result, err := agent.Execute(context.Background(), ctx)
	require.NoError(t, err)

	assert.NotEmpty(t, result.Reasoning.Hypotheses)
	assert.Contains(t, result.Reasoning.Hypotheses[0].Description, "Process unknown_type intent")
}

// Test multiple intents
func TestReasoningStructure_Execute_MultipleIntents(t *testing.T) {
	agent := NewReasoningStructureAgent()
	ctx := models.NewAgentContext("test-session", "test-trace")

	ctx.Reasoning.Intents = []models.Intent{
		{Type: "query_commits", Confidence: 0.9},
		{Type: "query_issues", Confidence: 0.7},
	}

	result, err := agent.Execute(context.Background(), ctx)
	require.NoError(t, err)

	// Should have hypotheses from both intents
	assert.NotEmpty(t, result.Reasoning.Hypotheses)
	assert.GreaterOrEqual(t, len(result.Reasoning.Hypotheses), 6) // 3 from commits + 3 from issues

	descriptions := []string{}
	for _, h := range result.Reasoning.Hypotheses {
		descriptions = append(descriptions, h.Description)
	}

	// Check for commit hypotheses
	assert.Contains(t, descriptions, "Retrieve commit data from GitLab")

	// Check for issue hypotheses
	assert.Contains(t, descriptions, "Retrieve issue data from YouTrack")
}

// Test low-confidence intents are filtered
func TestExecute_LowConfidenceIntentsFiltered(t *testing.T) {
	agent := NewReasoningStructureAgent()
	ctx := models.NewAgentContext("test-session", "test-trace")

	ctx.Reasoning.Intents = []models.Intent{
		{Type: "query_commits", Confidence: 0.9},
		{Type: "query_issues", Confidence: 0.2}, // Low confidence - should be filtered
	}

	result, err := agent.Execute(context.Background(), ctx)
	require.NoError(t, err)

	// Should only have hypotheses from high-confidence intent
	assert.NotEmpty(t, result.Reasoning.Hypotheses)

	descriptions := []string{}
	for _, h := range result.Reasoning.Hypotheses {
		descriptions = append(descriptions, h.Description)
	}

	// Check for commit hypotheses
	assert.Contains(t, descriptions, "Retrieve commit data from GitLab")

	// Low-confidence issue intent should not generate hypotheses
	for _, desc := range descriptions {
		assert.NotContains(t, desc, "YouTrack")
	}
}

// Test dependency graph structure
func TestExecute_DependencyGraphStructure(t *testing.T) {
	agent := NewReasoningStructureAgent()
	ctx := models.NewAgentContext("test-session", "test-trace")

	ctx.Reasoning.Intents = []models.Intent{
		{Type: "query_commits", Confidence: 0.9},
	}

	result, err := agent.Execute(context.Background(), ctx)
	require.NoError(t, err)

	// Verify dependency graph structure
	graph, ok := result.Reasoning.DependencyMap.(*DependencyGraph)
	require.True(t, ok)

	assert.NotEmpty(t, graph.Nodes)
	assert.NotNil(t, graph.Edges)

	// Check that all hypotheses are in the graph
	assert.Equal(t, len(result.Reasoning.Hypotheses), len(graph.Nodes))

	// Verify edges match hypothesis dependencies
	for _, h := range result.Reasoning.Hypotheses {
		if len(h.Dependencies) > 0 {
			// Each dependency should have an edge to this hypothesis
			for _, dep := range h.Dependencies {
				assert.Contains(t, graph.Edges[dep], h.ID)
			}
		}
	}
}

// Test hypothesis IDs are unique
func TestExecute_HypothesisIDsUnique(t *testing.T) {
	agent := NewReasoningStructureAgent()
	ctx := models.NewAgentContext("test-session", "test-trace")

	ctx.Reasoning.Intents = []models.Intent{
		{Type: "query_commits", Confidence: 0.9},
		{Type: "query_issues", Confidence: 0.85},
	}

	result, err := agent.Execute(context.Background(), ctx)
	require.NoError(t, err)

	// Check all hypothesis IDs are unique
	idMap := make(map[string]bool)
	for _, h := range result.Reasoning.Hypotheses {
		assert.False(t, idMap[h.ID], "Duplicate hypothesis ID: %s", h.ID)
		idMap[h.ID] = true
	}
}

// Test dependency chain is valid
func TestExecute_DependencyChainValid(t *testing.T) {
	agent := NewReasoningStructureAgent()
	ctx := models.NewAgentContext("test-session", "test-trace")

	ctx.Reasoning.Intents = []models.Intent{
		{Type: "query_commits", Confidence: 0.9},
	}

	result, err := agent.Execute(context.Background(), ctx)
	require.NoError(t, err)

	// Build ID map for quick lookup
	idMap := make(map[string]*models.Hypothesis)
	for i := range result.Reasoning.Hypotheses {
		idMap[result.Reasoning.Hypotheses[i].ID] = &result.Reasoning.Hypotheses[i]
	}

	// Verify all dependencies reference valid hypotheses
	for _, h := range result.Reasoning.Hypotheses {
		for _, dep := range h.Dependencies {
			_, exists := idMap[dep]
			assert.True(t, exists, "Hypothesis %s has invalid dependency: %s", h.ID, dep)
		}
	}
}

// Test cycle detection (manual cycle creation for testing)
func TestDetectCycles_NoCycle(t *testing.T) {
	agent := NewReasoningStructureAgent()

	graph := &DependencyGraph{
		Nodes: []string{"h0", "h1", "h2"},
		Edges: map[string][]string{
			"h0": {"h1"},
			"h1": {"h2"},
		},
	}

	cycles := agent.detectCycles(graph)
	assert.Empty(t, cycles)
}

func TestDetectCycles_WithCycle(t *testing.T) {
	agent := NewReasoningStructureAgent()

	// Create graph with cycle: h0 -> h1 -> h2 -> h0
	graph := &DependencyGraph{
		Nodes: []string{"h0", "h1", "h2"},
		Edges: map[string][]string{
			"h0": {"h1"},
			"h1": {"h2"},
			"h2": {"h0"},
		},
	}

	cycles := agent.detectCycles(graph)
	assert.NotEmpty(t, cycles)
}

// Test break cycles
func TestBreakCycles(t *testing.T) {
	agent := NewReasoningStructureAgent()

	// Create graph with cycle
	graph := &DependencyGraph{
		Nodes: []string{"h0", "h1", "h2"},
		Edges: map[string][]string{
			"h0": {"h1"},
			"h1": {"h2"},
			"h2": {"h0"},
		},
	}

	hypotheses := []models.Hypothesis{
		{ID: "h0", Description: "Step 0", Dependencies: []string{}},
		{ID: "h1", Description: "Step 1", Dependencies: []string{"h0"}},
		{ID: "h2", Description: "Step 2", Dependencies: []string{"h1"}},
	}

	cycles := agent.detectCycles(graph)
	require.NotEmpty(t, cycles)

	brokenGraph := agent.breakCycles(graph, hypotheses, cycles)

	// Verify cycle is broken
	newCycles := agent.detectCycles(brokenGraph)
	assert.Empty(t, newCycles)
}

// Test context isolation (original context not modified)
func TestReasoningStructure_Execute_ContextIsolation(t *testing.T) {
	agent := NewReasoningStructureAgent()
	ctx := models.NewAgentContext("test-session", "test-trace")

	ctx.Reasoning.Intents = []models.Intent{
		{Type: "query_commits", Confidence: 0.9},
	}

	// Store original state
	originalHypothesesCount := len(ctx.Reasoning.Hypotheses)

	result, err := agent.Execute(context.Background(), ctx)
	require.NoError(t, err)

	// Original context should not be modified
	assert.Equal(t, originalHypothesesCount, len(ctx.Reasoning.Hypotheses))

	// Result should have new data
	assert.NotEmpty(t, result.Reasoning.Hypotheses)
}

// Test audit trail
func TestReasoningStructure_Execute_AuditTrail(t *testing.T) {
	agent := NewReasoningStructureAgent()
	ctx := models.NewAgentContext("test-session", "test-trace")

	ctx.Reasoning.Intents = []models.Intent{
		{Type: "query_commits", Confidence: 0.9},
	}

	result, err := agent.Execute(context.Background(), ctx)
	require.NoError(t, err)

	// Verify audit trail
	require.NotEmpty(t, result.Audit.AgentRuns)
	run := result.Audit.AgentRuns[0]

	assert.Equal(t, "reasoning_structure", run.AgentID)
	assert.Equal(t, "success", run.Status)
	assert.GreaterOrEqual(t, run.DurationMS, int64(0))
	assert.Contains(t, run.KeysWritten, "reasoning.hypotheses")
	assert.Contains(t, run.KeysWritten, "reasoning.dependency_map")

	// Verify performance metrics
	assert.NotNil(t, result.Diagnostics.Performance)
	assert.NotEmpty(t, result.Diagnostics.Performance.AgentMetrics)
	metrics := result.Diagnostics.Performance.AgentMetrics["reasoning_structure"]
	assert.NotNil(t, metrics)
	assert.Equal(t, "success", metrics.Status)
	assert.Equal(t, 0, metrics.LLMCalls) // No LLM calls
}

// Test idempotency (same input produces same output)
func TestReasoningStructure_Execute_Idempotency(t *testing.T) {
	agent := NewReasoningStructureAgent()

	// Create identical contexts
	ctx1 := models.NewAgentContext("test-session", "test-trace")
	ctx1.Reasoning.Intents = []models.Intent{
		{Type: "query_commits", Confidence: 0.9},
	}

	ctx2 := models.NewAgentContext("test-session", "test-trace")
	ctx2.Reasoning.Intents = []models.Intent{
		{Type: "query_commits", Confidence: 0.9},
	}

	// Execute agent twice
	result1, err1 := agent.Execute(context.Background(), ctx1)
	require.NoError(t, err1)

	result2, err2 := agent.Execute(context.Background(), ctx2)
	require.NoError(t, err2)

	// Results should be identical (except timestamps)
	assert.Equal(t, len(result1.Reasoning.Hypotheses), len(result2.Reasoning.Hypotheses))

	for i := range result1.Reasoning.Hypotheses {
		assert.Equal(t, result1.Reasoning.Hypotheses[i].ID, result2.Reasoning.Hypotheses[i].ID)
		assert.Equal(t, result1.Reasoning.Hypotheses[i].Description, result2.Reasoning.Hypotheses[i].Description)
		assert.Equal(t, result1.Reasoning.Hypotheses[i].Dependencies, result2.Reasoning.Hypotheses[i].Dependencies)
	}
}

// Test with entities
func TestExecute_WithEntities(t *testing.T) {
	agent := NewReasoningStructureAgent()
	ctx := models.NewAgentContext("test-session", "test-trace")

	ctx.Reasoning.Intents = []models.Intent{
		{Type: "query_commits", Confidence: 0.9},
	}
	ctx.Reasoning.Entities = map[string]interface{}{
		"projects": []string{"gitlab-repo"},
		"dates":    []string{"last week"},
	}

	result, err := agent.Execute(context.Background(), ctx)
	require.NoError(t, err)

	// Entities should be available but not necessarily used in hypothesis generation
	// (that's implementation-specific, just verify no errors)
	assert.NotEmpty(t, result.Reasoning.Hypotheses)
}

// Benchmark hypothesis generation
func BenchmarkExecute_SingleIntent(b *testing.B) {
	agent := NewReasoningStructureAgent()
	ctx := models.NewAgentContext("test-session", "test-trace")
	ctx.Reasoning.Intents = []models.Intent{
		{Type: "query_commits", Confidence: 0.9},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := agent.Execute(context.Background(), ctx)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkExecute_MultipleIntents(b *testing.B) {
	agent := NewReasoningStructureAgent()
	ctx := models.NewAgentContext("test-session", "test-trace")
	ctx.Reasoning.Intents = []models.Intent{
		{Type: "query_commits", Confidence: 0.9},
		{Type: "query_issues", Confidence: 0.85},
		{Type: "query_analytics", Confidence: 0.8},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := agent.Execute(context.Background(), ctx)
		if err != nil {
			b.Fatal(err)
		}
	}
}
