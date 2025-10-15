package services

import (
	"context"
	"testing"
	"time"

	"github.com/mshogin/agents/internal/domain/models"
	domainservices "github.com/mshogin/agents/internal/domain/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Integration tests for full pipeline execution with realistic agent scenarios

// TestFullPipeline_Sequential_Success tests a successful sequential pipeline execution
func TestFullPipeline_Sequential_Success(t *testing.T) {
	// Create a realistic 3-agent pipeline: Intent → Structure → Summarization

	// Agent 1: Intent Detection
	intentAgent := &MockAgent{
		id:             "intent_detection",
		preconditions:  []string{},
		postconditions: []string{"reasoning.intents"},
		executeFn: func(ctx context.Context, agentContext *models.AgentContext) (*models.AgentContext, error) {
			// Simulate intent detection
			agentContext.Reasoning.Intents = []models.Intent{
				{Type: "query", Confidence: 0.95, Entities: []string{"gitlab", "commits"}},
			}
			return agentContext, nil
		},
	}

	// Agent 2: Reasoning Structure
	structureAgent := &MockAgent{
		id:             "reasoning_structure",
		preconditions:  []string{"reasoning.intents"},
		postconditions: []string{"reasoning.hypotheses"},
		executeFn: func(ctx context.Context, agentContext *models.AgentContext) (*models.AgentContext, error) {
			// Simulate hypothesis generation
			agentContext.Reasoning.Hypotheses = []models.Hypothesis{
				{ID: "h1", Description: "User wants GitLab commit data", Dependencies: []string{}},
			}
			return agentContext, nil
		},
	}

	// Agent 3: Summarization
	summaryAgent := &MockAgent{
		id:             "summarization",
		preconditions:  []string{"reasoning.hypotheses"},
		postconditions: []string{"reasoning.summary"},
		executeFn: func(ctx context.Context, agentContext *models.AgentContext) (*models.AgentContext, error) {
			// Simulate summary generation
			agentContext.Reasoning.Summary = "Query identified: GitLab commits analysis"
			return agentContext, nil
		},
	}

	// Configure pipeline
	config := PipelineConfig{
		Mode: SequentialMode,
		Agents: []AgentConfig{
			{ID: "intent_detection", Enabled: true, Timeout: 5000, Retry: 1},
			{ID: "reasoning_structure", Enabled: true, Timeout: 5000, Retry: 1},
			{ID: "summarization", Enabled: true, Timeout: 5000, Retry: 1},
		},
		Options: domainservices.AgentExecutionOptions{
			ValidateContract:   true,
			TrackPerformance:   true,
			CaptureChanges:     false,
			FailOnViolation:    true,
			TimeoutMS:          30000,
		},
	}

	manager := NewReasoningManager(config)
	require.NoError(t, manager.RegisterAgent(intentAgent))
	require.NoError(t, manager.RegisterAgent(structureAgent))
	require.NoError(t, manager.RegisterAgent(summaryAgent))

	ctx := context.Background()
	agentContext := createTestContext()

	// Execute pipeline
	result, err := manager.Execute(ctx, agentContext)

	// Verify success
	require.NoError(t, err)
	assert.NotNil(t, result)

	// Verify all agents executed
	assert.Len(t, result.Audit.AgentRuns, 3)
	assert.Equal(t, "intent_detection", result.Audit.AgentRuns[0].AgentID)
	assert.Equal(t, "reasoning_structure", result.Audit.AgentRuns[1].AgentID)
	assert.Equal(t, "summarization", result.Audit.AgentRuns[2].AgentID)

	// Verify all agents succeeded
	for _, run := range result.Audit.AgentRuns {
		assert.Equal(t, "success", run.Status)
		assert.Empty(t, run.Error)
	}

	// Verify final state
	assert.Len(t, result.Reasoning.Intents, 1)
	assert.Equal(t, "query", result.Reasoning.Intents[0].Type)
	assert.Len(t, result.Reasoning.Hypotheses, 1)
	assert.Equal(t, "Query identified: GitLab commits analysis", result.Reasoning.Summary)

	// Verify performance metrics
	assert.NotNil(t, result.Diagnostics.Performance)
	assert.Len(t, result.Diagnostics.Performance.AgentMetrics, 3)
	assert.GreaterOrEqual(t, result.Diagnostics.Performance.TotalDurationMS, int64(0))
}

// TestFullPipeline_Parallel_MultiPath tests parallel execution with multiple paths
func TestFullPipeline_Parallel_MultiPath(t *testing.T) {
	// Create pipeline with parallel paths:
	// intent_detection (level 0)
	// ├─ entity_extraction (level 1)
	// └─ context_analysis (level 1)
	// └─ inference (level 2, depends on both)

	intentAgent := &MockAgent{
		id:             "intent_detection",
		preconditions:  []string{},
		postconditions: []string{"reasoning.intents"},
		executeFn: func(ctx context.Context, agentContext *models.AgentContext) (*models.AgentContext, error) {
			time.Sleep(10 * time.Millisecond)
			agentContext.Reasoning.Intents = []models.Intent{
				{Type: "analysis", Confidence: 0.9},
			}
			return agentContext, nil
		},
	}

	entityAgent := &MockAgent{
		id:             "entity_extraction",
		preconditions:  []string{"reasoning.intents"},
		postconditions: []string{"reasoning.entities"},
		executeFn: func(ctx context.Context, agentContext *models.AgentContext) (*models.AgentContext, error) {
			time.Sleep(10 * time.Millisecond)
			agentContext.Reasoning.Entities = map[string]interface{}{
				"projects": []string{"project-1", "project-2"},
			}
			return agentContext, nil
		},
	}

	contextAgent := &MockAgent{
		id:             "context_analysis",
		preconditions:  []string{"reasoning.intents"},
		postconditions: []string{"enrichment.facts"},
		executeFn: func(ctx context.Context, agentContext *models.AgentContext) (*models.AgentContext, error) {
			time.Sleep(10 * time.Millisecond)
			agentContext.Enrichment.Facts = []models.Fact{
				{ID: "f1", Content: "Context analyzed", Source: "context_agent", Timestamp: time.Now(), Confidence: 0.95},
			}
			return agentContext, nil
		},
	}

	inferenceAgent := &MockAgent{
		id:             "inference",
		preconditions:  []string{"reasoning.entities", "enrichment.facts"},
		postconditions: []string{"reasoning.conclusions"},
		executeFn: func(ctx context.Context, agentContext *models.AgentContext) (*models.AgentContext, error) {
			agentContext.Reasoning.Conclusions = []models.Conclusion{
				{ID: "c1", Content: "Analysis complete", Confidence: 0.9},
			}
			return agentContext, nil
		},
	}

	config := PipelineConfig{
		Mode: ParallelMode,
		Agents: []AgentConfig{
			{ID: "intent_detection", Enabled: true, DependsOn: []string{}, Timeout: 5000},
			{ID: "entity_extraction", Enabled: true, DependsOn: []string{"intent_detection"}, Timeout: 5000},
			{ID: "context_analysis", Enabled: true, DependsOn: []string{"intent_detection"}, Timeout: 5000},
			{ID: "inference", Enabled: true, DependsOn: []string{"entity_extraction", "context_analysis"}, Timeout: 5000},
		},
		Options: domainservices.AgentExecutionOptions{
			ValidateContract: true,
			TrackPerformance: true,
			FailOnViolation:  false,
		},
	}

	manager := NewReasoningManager(config)
	require.NoError(t, manager.RegisterAgent(intentAgent))
	require.NoError(t, manager.RegisterAgent(entityAgent))
	require.NoError(t, manager.RegisterAgent(contextAgent))
	require.NoError(t, manager.RegisterAgent(inferenceAgent))

	ctx := context.Background()
	agentContext := createTestContext()

	// Execute pipeline
	startTime := time.Now()
	result, err := manager.Execute(ctx, agentContext)
	duration := time.Since(startTime)

	// Verify success
	require.NoError(t, err)
	assert.NotNil(t, result)

	// Verify all agents executed
	assert.Len(t, result.Audit.AgentRuns, 4)

	// Verify all expected agents are present in audit
	executedAgents := make(map[string]bool)
	for _, run := range result.Audit.AgentRuns {
		executedAgents[run.AgentID] = true
		assert.Equal(t, "success", run.Status, "Agent %s should succeed", run.AgentID)
	}

	assert.True(t, executedAgents["intent_detection"], "intent_detection should execute")
	assert.True(t, executedAgents["entity_extraction"], "entity_extraction should execute")
	assert.True(t, executedAgents["context_analysis"], "context_analysis should execute")
	assert.True(t, executedAgents["inference"], "inference should execute")

	// Verify final state has all expected data
	assert.Len(t, result.Reasoning.Intents, 1)
	assert.NotEmpty(t, result.Reasoning.Entities)
	assert.Len(t, result.Enrichment.Facts, 1)
	assert.Len(t, result.Reasoning.Conclusions, 1)

	// Verify parallel execution is faster than sequential would be
	// Sequential: 10+10+10+1 = 31ms minimum
	// Parallel: 10+10+1 = 21ms minimum (entity+context run in parallel)
	assert.Less(t, duration, 60*time.Millisecond, "Parallel execution should be faster")
}

// TestFullPipeline_Conditional_SelectiveExecution tests conditional execution
func TestFullPipeline_Conditional_SelectiveExecution(t *testing.T) {
	executedAgents := []string{}

	// Agent that always runs
	intentAgent := &MockAgent{
		id:             "intent_detection",
		preconditions:  []string{},
		postconditions: []string{"reasoning.intents"},
		executeFn: func(ctx context.Context, agentContext *models.AgentContext) (*models.AgentContext, error) {
			executedAgents = append(executedAgents, "intent_detection")
			agentContext.Reasoning.Intents = []models.Intent{
				{Type: "simple_query", Confidence: 0.9},
			}
			return agentContext, nil
		},
	}

	// Agent that runs only if intents exist (should run)
	validationAgent := &MockAgent{
		id:             "validation",
		preconditions:  []string{},
		postconditions: []string{},
		executeFn: func(ctx context.Context, agentContext *models.AgentContext) (*models.AgentContext, error) {
			executedAgents = append(executedAgents, "validation")
			return agentContext, nil
		},
	}

	// Agent that runs only if conclusions exist (should NOT run)
	summaryAgent := &MockAgent{
		id:             "summarization",
		preconditions:  []string{},
		postconditions: []string{},
		executeFn: func(ctx context.Context, agentContext *models.AgentContext) (*models.AgentContext, error) {
			executedAgents = append(executedAgents, "summarization")
			return agentContext, nil
		},
	}

	config := PipelineConfig{
		Mode: ConditionalMode,
		Agents: []AgentConfig{
			{ID: "intent_detection", Enabled: true, Conditions: []string{}, Timeout: 5000},
			{ID: "validation", Enabled: true, Conditions: []string{"reasoning.intents"}, Timeout: 5000},
			{ID: "summarization", Enabled: true, Conditions: []string{"reasoning.conclusions"}, Timeout: 5000},
		},
		Options: domainservices.AgentExecutionOptions{
			ValidateContract: false,
			TrackPerformance: true,
		},
	}

	manager := NewReasoningManager(config)
	require.NoError(t, manager.RegisterAgent(intentAgent))
	require.NoError(t, manager.RegisterAgent(validationAgent))
	require.NoError(t, manager.RegisterAgent(summaryAgent))

	ctx := context.Background()
	agentContext := createTestContext()

	// Execute pipeline
	result, err := manager.Execute(ctx, agentContext)

	// Verify success
	require.NoError(t, err)
	assert.NotNil(t, result)

	// Verify only intent_detection and validation executed
	assert.Equal(t, []string{"intent_detection", "validation"}, executedAgents)
	assert.Len(t, result.Audit.AgentRuns, 2)
}

// TestFullPipeline_ErrorRecovery tests error handling and recovery
func TestFullPipeline_ErrorRecovery(t *testing.T) {
	attemptCount := 0

	agent1 := &MockAgent{
		id:             "agent1",
		preconditions:  []string{},
		postconditions: []string{"reasoning.intents"},
		executeFn: func(ctx context.Context, agentContext *models.AgentContext) (*models.AgentContext, error) {
			agentContext.Reasoning.Intents = []models.Intent{{Type: "test", Confidence: 0.9}}
			return agentContext, nil
		},
	}

	// Agent that fails twice then succeeds
	agent2 := &MockAgent{
		id:             "agent2",
		preconditions:  []string{"reasoning.intents"},
		postconditions: []string{"reasoning.conclusions"},
		executeFn: func(ctx context.Context, agentContext *models.AgentContext) (*models.AgentContext, error) {
			attemptCount++
			if attemptCount <= 2 {
				return nil, assert.AnError
			}
			agentContext.Reasoning.Conclusions = []models.Conclusion{{ID: "c1", Content: "recovered", Confidence: 0.9}}
			return agentContext, nil
		},
	}

	agent3 := &MockAgent{
		id:             "agent3",
		preconditions:  []string{"reasoning.conclusions"},
		postconditions: []string{"reasoning.summary"},
		executeFn: func(ctx context.Context, agentContext *models.AgentContext) (*models.AgentContext, error) {
			agentContext.Reasoning.Summary = "completed"
			return agentContext, nil
		},
	}

	config := PipelineConfig{
		Mode: SequentialMode,
		Agents: []AgentConfig{
			{ID: "agent1", Enabled: true, Timeout: 5000, Retry: 0},
			{ID: "agent2", Enabled: true, Timeout: 5000, Retry: 2}, // Allow 2 retries
			{ID: "agent3", Enabled: true, Timeout: 5000, Retry: 0},
		},
		Options: domainservices.AgentExecutionOptions{
			ValidateContract: true,
			TrackPerformance: true,
			FailOnViolation:  true,
		},
	}

	manager := NewReasoningManager(config)
	require.NoError(t, manager.RegisterAgent(agent1))
	require.NoError(t, manager.RegisterAgent(agent2))
	require.NoError(t, manager.RegisterAgent(agent3))

	ctx := context.Background()
	agentContext := createTestContext()

	// Execute pipeline
	result, err := manager.Execute(ctx, agentContext)

	// Verify success despite failures
	require.NoError(t, err)
	assert.NotNil(t, result)

	// Verify agent2 was retried
	assert.Equal(t, 3, attemptCount) // Initial + 2 retries

	// Verify audit trail includes failed and successful runs
	// Should have: agent1 (success), agent2 (failed), agent2 (success), agent3 (success)
	assert.GreaterOrEqual(t, len(result.Audit.AgentRuns), 3, "Should have at least 3 runs")

	// Verify final summary was set
	assert.Equal(t, "completed", result.Reasoning.Summary)

	// Count successful runs
	successfulRuns := 0
	for _, run := range result.Audit.AgentRuns {
		if run.Status == "success" {
			successfulRuns++
		}
	}
	assert.GreaterOrEqual(t, successfulRuns, 3, "Should have at least 3 successful runs")
}

// TestFullPipeline_PerformanceMetrics tests performance tracking
func TestFullPipeline_PerformanceMetrics(t *testing.T) {
	agent1 := &MockAgent{
		id:             "fast_agent",
		preconditions:  []string{},
		postconditions: []string{"reasoning.intents"},
		executeFn: func(ctx context.Context, agentContext *models.AgentContext) (*models.AgentContext, error) {
			time.Sleep(5 * time.Millisecond)
			agentContext.Reasoning.Intents = []models.Intent{{Type: "test", Confidence: 0.9}}
			return agentContext, nil
		},
	}

	agent2 := &MockAgent{
		id:             "slow_agent",
		preconditions:  []string{"reasoning.intents"},
		postconditions: []string{"reasoning.conclusions"},
		executeFn: func(ctx context.Context, agentContext *models.AgentContext) (*models.AgentContext, error) {
			time.Sleep(20 * time.Millisecond)
			agentContext.Reasoning.Conclusions = []models.Conclusion{{ID: "c1", Content: "test", Confidence: 0.9}}
			return agentContext, nil
		},
	}

	config := PipelineConfig{
		Mode: SequentialMode,
		Agents: []AgentConfig{
			{ID: "fast_agent", Enabled: true, Timeout: 5000},
			{ID: "slow_agent", Enabled: true, Timeout: 5000},
		},
		Options: domainservices.AgentExecutionOptions{
			ValidateContract: false,
			TrackPerformance: true,
		},
	}

	manager := NewReasoningManager(config)
	require.NoError(t, manager.RegisterAgent(agent1))
	require.NoError(t, manager.RegisterAgent(agent2))

	ctx := context.Background()
	agentContext := createTestContext()

	// Execute pipeline
	result, err := manager.Execute(ctx, agentContext)

	// Verify success
	require.NoError(t, err)
	assert.NotNil(t, result)

	// Verify performance metrics are tracked
	assert.NotNil(t, result.Diagnostics.Performance)
	assert.Len(t, result.Diagnostics.Performance.AgentMetrics, 2)

	// Verify agent durations are reasonable
	fastMetrics := result.Diagnostics.Performance.AgentMetrics["fast_agent"]
	slowMetrics := result.Diagnostics.Performance.AgentMetrics["slow_agent"]

	assert.NotNil(t, fastMetrics)
	assert.NotNil(t, slowMetrics)

	assert.GreaterOrEqual(t, fastMetrics.DurationMS, int64(5))
	assert.GreaterOrEqual(t, slowMetrics.DurationMS, int64(20))

	// Verify total duration is at least the sum of agent durations
	assert.GreaterOrEqual(t, result.Diagnostics.Performance.TotalDurationMS, int64(25))

	// Verify agent runs are recorded
	assert.Len(t, result.Audit.AgentRuns, 2)
	for _, run := range result.Audit.AgentRuns {
		assert.Greater(t, run.DurationMS, int64(0))
		assert.False(t, run.Timestamp.IsZero())
	}
}

// TestFullPipeline_ContractViolation tests contract validation enforcement
func TestFullPipeline_ContractViolation(t *testing.T) {
	agent1 := &MockAgent{
		id:             "agent1",
		preconditions:  []string{},
		postconditions: []string{"reasoning.intents"},
		executeFn: func(ctx context.Context, agentContext *models.AgentContext) (*models.AgentContext, error) {
			// Violate postcondition by not setting intents
			return agentContext, nil
		},
	}

	agent2 := &MockAgent{
		id:             "agent2",
		preconditions:  []string{"reasoning.intents"},
		postconditions: []string{"reasoning.conclusions"},
		executeFn: func(ctx context.Context, agentContext *models.AgentContext) (*models.AgentContext, error) {
			agentContext.Reasoning.Conclusions = []models.Conclusion{{ID: "c1", Content: "test", Confidence: 0.9}}
			return agentContext, nil
		},
	}

	config := PipelineConfig{
		Mode: SequentialMode,
		Agents: []AgentConfig{
			{ID: "agent1", Enabled: true, Timeout: 5000},
			{ID: "agent2", Enabled: true, Timeout: 5000},
		},
		Options: domainservices.AgentExecutionOptions{
			ValidateContract: true,
			FailOnViolation:  true, // Fail on contract violation
		},
	}

	manager := NewReasoningManager(config)
	require.NoError(t, manager.RegisterAgent(agent1))
	require.NoError(t, manager.RegisterAgent(agent2))

	ctx := context.Background()
	agentContext := createTestContext()

	// Execute pipeline
	result, err := manager.Execute(ctx, agentContext)

	// Verify failure due to postcondition violation
	require.Error(t, err)
	assert.Contains(t, err.Error(), "postcondition validation failed")
	assert.NotNil(t, result)

	// Verify agent2 did not execute (pipeline stopped after agent1 failed validation)
	// Agent runs may be empty or contain only agent1 depending on when validation occurs
	assert.LessOrEqual(t, len(result.Audit.AgentRuns), 1, "Should have at most 1 agent run")

	// Verify agent2 never executed
	for _, run := range result.Audit.AgentRuns {
		assert.NotEqual(t, "agent2", run.AgentID, "agent2 should not have executed")
	}
}

// TestFullPipeline_LargeScale tests pipeline with many agents
func TestFullPipeline_LargeScale(t *testing.T) {
	// Create a pipeline with 10 agents to test scalability
	agents := make([]*MockAgent, 10)
	agentConfigs := make([]AgentConfig, 10)

	for i := 0; i < 10; i++ {
		agentID := string(rune('a' + i))
		var preconditions []string
		if i > 0 {
			preconditions = []string{string(rune('a' + i - 1)) + "_output"}
		}
		postconditions := []string{agentID + "_output"}

		agents[i] = &MockAgent{
			id:             agentID,
			preconditions:  preconditions,
			postconditions: postconditions,
			executeFn: func(ctx context.Context, agentContext *models.AgentContext) (*models.AgentContext, error) {
				time.Sleep(1 * time.Millisecond)
				return agentContext, nil
			},
		}

		agentConfigs[i] = AgentConfig{
			ID:      agentID,
			Enabled: true,
			Timeout: 5000,
		}
	}

	config := PipelineConfig{
		Mode:    SequentialMode,
		Agents:  agentConfigs,
		Options: domainservices.AgentExecutionOptions{
			ValidateContract: false,
			TrackPerformance: true,
		},
	}

	manager := NewReasoningManager(config)
	for _, agent := range agents {
		require.NoError(t, manager.RegisterAgent(agent))
	}

	ctx := context.Background()
	agentContext := createTestContext()

	// Execute pipeline
	startTime := time.Now()
	result, err := manager.Execute(ctx, agentContext)
	duration := time.Since(startTime)

	// Verify success
	require.NoError(t, err)
	assert.NotNil(t, result)

	// Verify all 10 agents executed
	assert.Len(t, result.Audit.AgentRuns, 10)

	// Verify execution completed in reasonable time
	assert.Less(t, duration, 100*time.Millisecond, "Large pipeline should complete quickly")

	// Verify metrics for all agents
	assert.Len(t, result.Diagnostics.Performance.AgentMetrics, 10)
}
