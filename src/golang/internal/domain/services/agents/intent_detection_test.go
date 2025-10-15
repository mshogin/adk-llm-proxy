package agents

import (
	"context"
	"testing"

	"github.com/mshogin/agents/internal/domain/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewIntentDetectionAgent tests agent initialization
func TestNewIntentDetectionAgent(t *testing.T) {
	agent := NewIntentDetectionAgent()

	assert.NotNil(t, agent)
	assert.Equal(t, "intent_detection", agent.AgentID())
}

// TestAgentID tests AgentID method
func TestAgentID(t *testing.T) {
	agent := NewIntentDetectionAgent()
	assert.Equal(t, "intent_detection", agent.AgentID())
}

// TestPreconditions tests Preconditions method
func TestPreconditions(t *testing.T) {
	agent := NewIntentDetectionAgent()
	preconditions := agent.Preconditions()

	// Intent detection is first agent - no preconditions
	assert.Empty(t, preconditions)
}

// TestPostconditions tests Postconditions method
func TestPostconditions(t *testing.T) {
	agent := NewIntentDetectionAgent()
	postconditions := agent.Postconditions()

	expected := []string{
		"reasoning.intents",
		"reasoning.entities",
		"reasoning.confidence_scores",
	}

	assert.Equal(t, expected, postconditions)
}

// TestGetMetadata tests GetMetadata method
func TestGetMetadata(t *testing.T) {
	agent := NewIntentDetectionAgent()
	metadata := agent.GetMetadata()

	assert.Equal(t, "intent_detection", metadata.ID)
	assert.Equal(t, "Intent Detection Agent", metadata.Name)
	assert.NotEmpty(t, metadata.Description)
	assert.Equal(t, "1.0.0", metadata.Version)
	assert.Contains(t, metadata.Tags, "intent")
	assert.Empty(t, metadata.Dependencies)
}

// TestGetCapabilities tests GetCapabilities method
func TestGetCapabilities(t *testing.T) {
	agent := NewIntentDetectionAgent()
	capabilities := agent.GetCapabilities()

	assert.False(t, capabilities.SupportsParallelExecution) // First agent - must run first
	assert.True(t, capabilities.SupportsRetry)
	assert.False(t, capabilities.RequiresLLM) // Rule-based
	assert.True(t, capabilities.IsDeterministic)
	assert.Greater(t, capabilities.EstimatedDuration, 0)
}

// TestExecute_QueryCommits tests intent detection for commit queries
func TestExecute_QueryCommits(t *testing.T) {
	agent := NewIntentDetectionAgent()

	tests := []struct {
		name     string
		query    string
		expected string
	}{
		{
			name:     "latest commits",
			query:    "show me the latest commits",
			expected: "query_commits",
		},
		{
			name:     "recent changes",
			query:    "what changed recently?",
			expected: "query_commits",
		},
		{
			name:     "git history",
			query:    "list commits from last week",
			expected: "query_commits",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := models.NewAgentContext("test-session", "test-trace")
			ctx.Retrieval.Queries = []models.Query{
				{QueryString: tt.query},
			}

			result, err := agent.Execute(context.Background(), ctx)

			require.NoError(t, err)
			require.NotNil(t, result)
			require.NotEmpty(t, result.Reasoning.Intents)

			// Check primary intent type
			assert.Equal(t, tt.expected, result.Reasoning.Intents[0].Type)
			assert.Greater(t, result.Reasoning.Intents[0].Confidence, 0.0)

			// Check confidence scores exist
			assert.NotEmpty(t, result.Reasoning.ConfidenceScores)
			assert.Contains(t, result.Reasoning.ConfidenceScores, "primary_intent")
			assert.Contains(t, result.Reasoning.ConfidenceScores, "overall")
		})
	}
}

// TestExecute_QueryIssues tests intent detection for issue queries
func TestExecute_QueryIssues(t *testing.T) {
	agent := NewIntentDetectionAgent()

	tests := []struct {
		name     string
		query    string
		expected string
	}{
		{
			name:     "open issues",
			query:    "show me open issues",
			expected: "query_issues",
		},
		{
			name:     "my tasks",
			query:    "what are my tasks?",
			expected: "query_issues",
		},
		{
			name:     "bugs",
			query:    "list all bugs in the project",
			expected: "query_issues",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := models.NewAgentContext("test-session", "test-trace")
			ctx.Retrieval.Queries = []models.Query{
				{QueryString: tt.query},
			}

			result, err := agent.Execute(context.Background(), ctx)

			require.NoError(t, err)
			require.NotNil(t, result)
			require.NotEmpty(t, result.Reasoning.Intents)

			assert.Equal(t, tt.expected, result.Reasoning.Intents[0].Type)
			assert.Greater(t, result.Reasoning.Intents[0].Confidence, 0.0)
		})
	}
}

// TestExecute_QueryAnalytics tests intent detection for analytics queries
func TestExecute_QueryAnalytics(t *testing.T) {
	agent := NewIntentDetectionAgent()

	tests := []struct {
		name     string
		query    string
		expected string
	}{
		{
			name:     "statistics",
			query:    "show me statistics for this project",
			expected: "query_analytics",
		},
		{
			name:     "count",
			query:    "how many items were processed last week?",
			expected: "query_analytics",
		},
		{
			name:     "trends",
			query:    "what are the trends over time?",
			expected: "query_analytics",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := models.NewAgentContext("test-session", "test-trace")
			ctx.Retrieval.Queries = []models.Query{
				{QueryString: tt.query},
			}

			result, err := agent.Execute(context.Background(), ctx)

			require.NoError(t, err)
			require.NotNil(t, result)
			require.NotEmpty(t, result.Reasoning.Intents)

			assert.Equal(t, tt.expected, result.Reasoning.Intents[0].Type)
			assert.Greater(t, result.Reasoning.Intents[0].Confidence, 0.0)
		})
	}
}

// TestExecute_QueryStatus tests intent detection for status queries
func TestExecute_QueryStatus(t *testing.T) {
	agent := NewIntentDetectionAgent()

	tests := []struct {
		name     string
		query    string
		expected string
	}{
		{
			name:     "system status",
			query:    "what's the status of the system?",
			expected: "query_status",
		},
		{
			name:     "health check",
			query:    "is everything running ok?",
			expected: "query_status",
		},
		{
			name:     "check status",
			query:    "check status of the deployment",
			expected: "query_status",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := models.NewAgentContext("test-session", "test-trace")
			ctx.Retrieval.Queries = []models.Query{
				{QueryString: tt.query},
			}

			result, err := agent.Execute(context.Background(), ctx)

			require.NoError(t, err)
			require.NotNil(t, result)
			require.NotEmpty(t, result.Reasoning.Intents)

			assert.Equal(t, tt.expected, result.Reasoning.Intents[0].Type)
			assert.Greater(t, result.Reasoning.Intents[0].Confidence, 0.0)
		})
	}
}

// TestExecute_CommandAction tests intent detection for command actions
func TestExecute_CommandAction(t *testing.T) {
	agent := NewIntentDetectionAgent()

	tests := []struct {
		name     string
		query    string
		expected string
	}{
		{
			name:     "deploy",
			query:    "please deploy the latest version",
			expected: "command_action",
		},
		{
			name:     "restart",
			query:    "restart the server",
			expected: "command_action",
		},
		{
			name:     "create",
			query:    "create a new issue for this bug",
			expected: "command_action",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := models.NewAgentContext("test-session", "test-trace")
			ctx.Retrieval.Queries = []models.Query{
				{QueryString: tt.query},
			}

			result, err := agent.Execute(context.Background(), ctx)

			require.NoError(t, err)
			require.NotNil(t, result)
			require.NotEmpty(t, result.Reasoning.Intents)

			assert.Equal(t, tt.expected, result.Reasoning.Intents[0].Type)
			assert.Greater(t, result.Reasoning.Intents[0].Confidence, 0.0)
		})
	}
}

// TestExecute_RequestHelp tests intent detection for help requests
func TestExecute_RequestHelp(t *testing.T) {
	agent := NewIntentDetectionAgent()

	tests := []struct {
		name     string
		query    string
		expected string
	}{
		{
			name:     "how to",
			query:    "how do I configure the application?",
			expected: "request_help",
		},
		{
			name:     "what is",
			query:    "what is MCP?",
			expected: "request_help",
		},
		{
			name:     "help",
			query:    "can you help me with this error?",
			expected: "request_help",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := models.NewAgentContext("test-session", "test-trace")
			ctx.Retrieval.Queries = []models.Query{
				{QueryString: tt.query},
			}

			result, err := agent.Execute(context.Background(), ctx)

			require.NoError(t, err)
			require.NotNil(t, result)
			require.NotEmpty(t, result.Reasoning.Intents)

			assert.Equal(t, tt.expected, result.Reasoning.Intents[0].Type)
			assert.Greater(t, result.Reasoning.Intents[0].Confidence, 0.0)
		})
	}
}

// TestExecute_Conversation tests intent detection for general conversation
func TestExecute_Conversation(t *testing.T) {
	agent := NewIntentDetectionAgent()

	tests := []struct {
		name     string
		query    string
		expected string
	}{
		{
			name:     "greeting",
			query:    "hello there",
			expected: "conversation",
		},
		{
			name:     "thanks",
			query:    "thanks!", // Simple thanks without "help" keyword
			expected: "conversation",
		},
		{
			name:     "goodbye",
			query:    "bye, see you later",
			expected: "conversation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := models.NewAgentContext("test-session", "test-trace")
			ctx.Retrieval.Queries = []models.Query{
				{QueryString: tt.query},
			}

			result, err := agent.Execute(context.Background(), ctx)

			require.NoError(t, err)
			require.NotNil(t, result)
			require.NotEmpty(t, result.Reasoning.Intents)

			assert.Equal(t, tt.expected, result.Reasoning.Intents[0].Type)
		})
	}
}

// TestExecute_EntityExtraction tests entity extraction
func TestExecute_EntityExtraction(t *testing.T) {
	agent := NewIntentDetectionAgent()

	tests := []struct {
		name           string
		query          string
		expectProjects bool
		expectDates    bool
		expectProviders bool
		expectStatuses bool
	}{
		{
			name:           "with project and date",
			query:          "show commits from project gitlab-mcp from last week",
			expectProjects: true,
			expectDates:    true,
		},
		{
			name:            "with provider and status",
			query:           "check open issues in gitlab",
			expectProviders: true,
			expectStatuses:  true,
		},
		{
			name:        "with explicit date",
			query:       "commits from 2024-01-15",
			expectDates: true,
		},
		{
			name:            "with multiple providers",
			query:           "compare gitlab and youtrack",
			expectProviders: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := models.NewAgentContext("test-session", "test-trace")
			ctx.Retrieval.Queries = []models.Query{
				{QueryString: tt.query},
			}

			result, err := agent.Execute(context.Background(), ctx)

			require.NoError(t, err)
			require.NotNil(t, result)
			require.NotNil(t, result.Reasoning.Entities)

			entities := result.Reasoning.Entities

			if tt.expectProjects {
				assert.Contains(t, entities, "projects")
			}

			if tt.expectDates {
				assert.Contains(t, entities, "dates")
			}

			if tt.expectProviders {
				assert.Contains(t, entities, "providers")
			}

			if tt.expectStatuses {
				assert.Contains(t, entities, "statuses")
			}
		})
	}
}

// TestExecute_MultipleIntents tests detection of multiple intents
func TestExecute_MultipleIntents(t *testing.T) {
	agent := NewIntentDetectionAgent()

	ctx := models.NewAgentContext("test-session", "test-trace")
	ctx.Retrieval.Queries = []models.Query{
		{QueryString: "show me recent commits and their statistics"},
	}

	result, err := agent.Execute(context.Background(), ctx)

	require.NoError(t, err)
	require.NotNil(t, result)

	// Should detect both query_commits and query_analytics
	intents := result.Reasoning.Intents
	assert.GreaterOrEqual(t, len(intents), 2)

	intentTypes := make([]string, 0, len(intents))
	for _, intent := range intents {
		intentTypes = append(intentTypes, intent.Type)
	}

	// Check that both intents are present
	assert.Contains(t, intentTypes, "query_commits")
	assert.Contains(t, intentTypes, "query_analytics")
}

// TestExecute_ConfidenceScoring tests confidence score calculation
func TestExecute_ConfidenceScoring(t *testing.T) {
	agent := NewIntentDetectionAgent()

	tests := []struct {
		name               string
		query              string
		minPrimaryConfidence float64
	}{
		{
			name:               "high confidence - explicit",
			query:              "show me the latest commits from the repository",
			minPrimaryConfidence: 0.7,
		},
		{
			name:               "medium confidence - partial match",
			query:              "what changed?",
			minPrimaryConfidence: 0.3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := models.NewAgentContext("test-session", "test-trace")
			ctx.Retrieval.Queries = []models.Query{
				{QueryString: tt.query},
			}

			result, err := agent.Execute(context.Background(), ctx)

			require.NoError(t, err)
			require.NotNil(t, result)

			// Check confidence scores
			scores := result.Reasoning.ConfidenceScores
			require.Contains(t, scores, "primary_intent")
			assert.GreaterOrEqual(t, scores["primary_intent"], tt.minPrimaryConfidence)
		})
	}
}

// TestExecute_AuditTrail tests audit trail recording
func TestExecute_AuditTrail(t *testing.T) {
	agent := NewIntentDetectionAgent()

	ctx := models.NewAgentContext("test-session", "test-trace")
	ctx.Retrieval.Queries = []models.Query{
		{QueryString: "show me recent commits"},
	}

	result, err := agent.Execute(context.Background(), ctx)

	require.NoError(t, err)
	require.NotNil(t, result)

	// Check audit trail
	require.NotNil(t, result.Audit)
	require.NotEmpty(t, result.Audit.AgentRuns)

	run := result.Audit.AgentRuns[0]
	assert.Equal(t, "intent_detection", run.AgentID)
	assert.Equal(t, "success", run.Status)
	assert.GreaterOrEqual(t, run.DurationMS, int64(0)) // May be 0 for very fast execution
	assert.NotEmpty(t, run.KeysWritten)
	assert.Contains(t, run.KeysWritten, "reasoning.intents")

	// Check performance metrics
	require.NotNil(t, result.Diagnostics)
	require.NotNil(t, result.Diagnostics.Performance)
	require.NotEmpty(t, result.Diagnostics.Performance.AgentMetrics)
	assert.Contains(t, result.Diagnostics.Performance.AgentMetrics, "intent_detection")

	metrics := result.Diagnostics.Performance.AgentMetrics["intent_detection"]
	assert.Equal(t, "success", metrics.Status)
	assert.Equal(t, 0, metrics.LLMCalls) // No LLM for rule-based
}

// TestExecute_EmptyInput tests handling of empty input
func TestExecute_EmptyInput(t *testing.T) {
	agent := NewIntentDetectionAgent()

	ctx := models.NewAgentContext("test-session", "test-trace")
	// No query provided

	_, err := agent.Execute(context.Background(), ctx)

	// Should return error for empty input
	require.Error(t, err)
	if err != nil {
		assert.Contains(t, err.Error(), "no input text")
	}
}

// TestExecute_Idempotency tests that agent is idempotent
func TestExecute_Idempotency(t *testing.T) {
	agent := NewIntentDetectionAgent()

	ctx := models.NewAgentContext("test-session", "test-trace")
	ctx.Retrieval.Queries = []models.Query{
		{QueryString: "show me recent commits"},
	}

	// Execute twice
	result1, err1 := agent.Execute(context.Background(), ctx)
	require.NoError(t, err1)

	result2, err2 := agent.Execute(context.Background(), ctx)
	require.NoError(t, err2)

	// Results should be the same (same intents, same confidence)
	require.Equal(t, len(result1.Reasoning.Intents), len(result2.Reasoning.Intents))

	if len(result1.Reasoning.Intents) > 0 {
		assert.Equal(t, result1.Reasoning.Intents[0].Type, result2.Reasoning.Intents[0].Type)
		assert.Equal(t, result1.Reasoning.Intents[0].Confidence, result2.Reasoning.Intents[0].Confidence)
	}
}

// TestExecute_ContextIsolation tests that agent doesn't modify input context
func TestExecute_ContextIsolation(t *testing.T) {
	agent := NewIntentDetectionAgent()

	ctx := models.NewAgentContext("test-session", "test-trace")
	ctx.Retrieval.Queries = []models.Query{
		{QueryString: "show me recent commits"},
	}

	// Store original values
	originalSessionID := ctx.Metadata.SessionID

	// Execute agent
	result, err := agent.Execute(context.Background(), ctx)
	require.NoError(t, err)

	// Original context should be unchanged
	assert.Equal(t, originalSessionID, ctx.Metadata.SessionID)
	assert.Empty(t, ctx.Reasoning.Intents) // Original should be empty

	// Result should have new data
	assert.NotEmpty(t, result.Reasoning.Intents)
}

// TestExecute_RealWorldScenarios tests real-world query patterns
func TestExecute_RealWorldScenarios(t *testing.T) {
	agent := NewIntentDetectionAgent()

	tests := []struct {
		name            string
		query           string
		expectedIntent  string
		expectedEntities []string // Entity types we expect
	}{
		{
			name:            "complex commit query",
			query:           "show me all commits from gitlab-mcp project last week",
			expectedIntent:  "query_commits",
			expectedEntities: []string{"projects", "dates"},
		},
		{
			name:            "multi-provider comparison",
			query:           "compare open issues between gitlab and youtrack",
			expectedIntent:  "query_issues",
			expectedEntities: []string{"providers", "statuses"},
		},
		{
			name:            "analytics with date range",
			query:           "show me statistics for this month",
			expectedIntent:  "query_analytics",
			expectedEntities: []string{"dates"},
		},
		{
			name:           "deployment command",
			query:          "deploy version 2.0 to production",
			expectedIntent: "command_action",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := models.NewAgentContext("test-session", "test-trace")
			ctx.Retrieval.Queries = []models.Query{
				{QueryString: tt.query},
			}

			result, err := agent.Execute(context.Background(), ctx)

			require.NoError(t, err)
			require.NotNil(t, result)
			require.NotEmpty(t, result.Reasoning.Intents)

			// Check primary intent
			assert.Equal(t, tt.expectedIntent, result.Reasoning.Intents[0].Type)

			// Check expected entities
			for _, entityType := range tt.expectedEntities {
				assert.Contains(t, result.Reasoning.Entities, entityType,
					"Expected entity type %s not found", entityType)
			}
		})
	}
}
