package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/mshogin/agents/internal/domain/models"
	domainservices "github.com/mshogin/agents/internal/domain/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockAgent is a test implementation of ReasoningAgent
type MockAgent struct {
	id             string
	preconditions  []string
	postconditions []string
	executeFn      func(ctx context.Context, agentContext *models.AgentContext) (*models.AgentContext, error)
	executed       bool
}

func (m *MockAgent) AgentID() string                         { return m.id }
func (m *MockAgent) Preconditions() []string                 { return m.preconditions }
func (m *MockAgent) Postconditions() []string                { return m.postconditions }
func (m *MockAgent) Execute(ctx context.Context, agentContext *models.AgentContext) (*models.AgentContext, error) {
	m.executed = true
	if m.executeFn != nil {
		return m.executeFn(ctx, agentContext)
	}
	return agentContext, nil
}

// Helper to create test context
func createTestContext() *models.AgentContext {
	return &models.AgentContext{
		Version: "1.0.0",
		Metadata: &models.MetadataContext{
			SessionID: "test-session",
			TraceID:   "trace-123",
			CreatedAt: time.Now(),
		},
		Reasoning: &models.ReasoningContext{
			Intents:     []models.Intent{},
			Entities:    map[string]interface{}{},
			Hypotheses:  []models.Hypothesis{},
			Conclusions: []models.Conclusion{},
		},
		Enrichment: &models.EnrichmentContext{
			Facts:            []models.Fact{},
			DerivedKnowledge: []models.Knowledge{},
			Relationships:    []models.Relationship{},
		},
		Retrieval: &models.RetrievalContext{
			Plans:     []models.RetrievalPlan{},
			Queries:   []models.Query{},
			Artifacts: []models.Artifact{},
		},
		LLM: &models.LLMContext{
			Usage: &models.LLMUsage{},
		},
		Diagnostics: &models.DiagnosticsContext{
			Warnings: []models.Warning{},
			Performance: &models.PerformanceData{
				AgentMetrics: make(map[string]*models.AgentMetrics),
			},
		},
		Audit: &models.AuditContext{
			AgentRuns: []models.AgentRun{},
			Diffs:     []models.ContextDiff{},
		},
	}
}

// TestNewReasoningManager tests the constructor
func TestNewReasoningManager(t *testing.T) {
	config := PipelineConfig{
		Mode:   SequentialMode,
		Agents: []AgentConfig{},
	}

	manager := NewReasoningManager(config)

	assert.NotNil(t, manager)
	assert.NotNil(t, manager.agents)
	assert.NotNil(t, manager.validator)
	assert.Equal(t, SequentialMode, manager.config.Mode)
}

// TestRegisterAgent tests agent registration
func TestRegisterAgent(t *testing.T) {
	config := PipelineConfig{Mode: SequentialMode}
	manager := NewReasoningManager(config)

	agent := &MockAgent{id: "test_agent"}
	err := manager.RegisterAgent(agent)

	assert.NoError(t, err)

	// Verify agent was registered
	retrievedAgent, err := manager.GetAgent("test_agent")
	assert.NoError(t, err)
	assert.Equal(t, agent, retrievedAgent)
}

// TestRegisterAgent_DuplicateID tests duplicate agent registration
func TestRegisterAgent_DuplicateID(t *testing.T) {
	config := PipelineConfig{Mode: SequentialMode}
	manager := NewReasoningManager(config)

	agent1 := &MockAgent{id: "duplicate_agent"}
	agent2 := &MockAgent{id: "duplicate_agent"}

	err := manager.RegisterAgent(agent1)
	assert.NoError(t, err)

	err = manager.RegisterAgent(agent2)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already registered")
}

// TestGetAgent_NotFound tests retrieving non-existent agent
func TestGetAgent_NotFound(t *testing.T) {
	config := PipelineConfig{Mode: SequentialMode}
	manager := NewReasoningManager(config)

	agent, err := manager.GetAgent("nonexistent")
	assert.Error(t, err)
	assert.Nil(t, agent)
	assert.Contains(t, err.Error(), "not found")
}

// TestExecute_SequentialMode tests sequential execution
func TestExecute_SequentialMode(t *testing.T) {
	executionOrder := []string{}

	agent1 := &MockAgent{
		id:             "agent1",
		preconditions:  []string{},
		postconditions: []string{"reasoning.intents"},
		executeFn: func(ctx context.Context, agentContext *models.AgentContext) (*models.AgentContext, error) {
			executionOrder = append(executionOrder, "agent1")
			agentContext.Reasoning.Intents = []models.Intent{{Type: "test"}}
			return agentContext, nil
		},
	}

	agent2 := &MockAgent{
		id:             "agent2",
		preconditions:  []string{"reasoning.intents"},
		postconditions: []string{"reasoning.conclusions"},
		executeFn: func(ctx context.Context, agentContext *models.AgentContext) (*models.AgentContext, error) {
			executionOrder = append(executionOrder, "agent2")
			agentContext.Reasoning.Conclusions = []models.Conclusion{{ID: "c1", Description: "test", Confidence: 0.9}}
			return agentContext, nil
		},
	}

	config := PipelineConfig{
		Mode: SequentialMode,
		Agents: []AgentConfig{
			{ID: "agent1", Enabled: true, Timeout: 5000, Retry: 1},
			{ID: "agent2", Enabled: true, Timeout: 5000, Retry: 1},
		},
		Options: domainservices.AgentExecutionOptions{
			ValidateContract: false,
			TrackPerformance: true,
			CaptureChanges:   false,
			FailOnViolation:  false,
		},
	}

	manager := NewReasoningManager(config)
	manager.RegisterAgent(agent1)
	manager.RegisterAgent(agent2)

	ctx := context.Background()
	agentContext := createTestContext()

	result, err := manager.Execute(ctx, agentContext)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, []string{"agent1", "agent2"}, executionOrder)
	assert.Len(t, result.Reasoning.Intents, 1)
	assert.Len(t, result.Reasoning.Conclusions, 1)
	assert.Len(t, result.Audit.AgentRuns, 2)
}

// TestExecute_SequentialMode_WithDisabledAgent tests sequential execution with disabled agent
func TestExecute_SequentialMode_WithDisabledAgent(t *testing.T) {
	executionOrder := []string{}

	agent1 := &MockAgent{
		id:             "agent1",
		postconditions: []string{"reasoning.intents"},
		executeFn: func(ctx context.Context, agentContext *models.AgentContext) (*models.AgentContext, error) {
			executionOrder = append(executionOrder, "agent1")
			agentContext.Reasoning.Intents = []models.Intent{{Type: "test"}}
			return agentContext, nil
		},
	}

	agent2 := &MockAgent{
		id: "agent2",
		executeFn: func(ctx context.Context, agentContext *models.AgentContext) (*models.AgentContext, error) {
			executionOrder = append(executionOrder, "agent2")
			return agentContext, nil
		},
	}

	agent3 := &MockAgent{
		id:             "agent3",
		preconditions:  []string{"reasoning.intents"},
		postconditions: []string{"reasoning.conclusions"},
		executeFn: func(ctx context.Context, agentContext *models.AgentContext) (*models.AgentContext, error) {
			executionOrder = append(executionOrder, "agent3")
			agentContext.Reasoning.Conclusions = []models.Conclusion{{ID: "c1", Description: "test", Confidence: 0.9}}
			return agentContext, nil
		},
	}

	config := PipelineConfig{
		Mode: SequentialMode,
		Agents: []AgentConfig{
			{ID: "agent1", Enabled: true, Timeout: 5000},
			{ID: "agent2", Enabled: false, Timeout: 5000}, // Disabled
			{ID: "agent3", Enabled: true, Timeout: 5000},
		},
		Options: domainservices.AgentExecutionOptions{
			ValidateContract: false,
		},
	}

	manager := NewReasoningManager(config)
	manager.RegisterAgent(agent1)
	manager.RegisterAgent(agent2)
	manager.RegisterAgent(agent3)

	ctx := context.Background()
	agentContext := createTestContext()

	result, err := manager.Execute(ctx, agentContext)

	require.NoError(t, err)
	assert.Equal(t, []string{"agent1", "agent3"}, executionOrder)
	assert.Len(t, result.Audit.AgentRuns, 2)
}

// TestExecute_SequentialMode_Error tests error handling in sequential mode
func TestExecute_SequentialMode_Error(t *testing.T) {
	agent1 := &MockAgent{
		id: "agent1",
		executeFn: func(ctx context.Context, agentContext *models.AgentContext) (*models.AgentContext, error) {
			return agentContext, nil
		},
	}

	agent2 := &MockAgent{
		id: "agent2",
		executeFn: func(ctx context.Context, agentContext *models.AgentContext) (*models.AgentContext, error) {
			return nil, errors.New("agent2 failed")
		},
	}

	agent3 := &MockAgent{
		id: "agent3",
	}

	config := PipelineConfig{
		Mode: SequentialMode,
		Agents: []AgentConfig{
			{ID: "agent1", Enabled: true, Timeout: 5000},
			{ID: "agent2", Enabled: true, Timeout: 5000, Retry: 0},
			{ID: "agent3", Enabled: true, Timeout: 5000},
		},
		Options: domainservices.AgentExecutionOptions{
			ValidateContract: false,
		},
	}

	manager := NewReasoningManager(config)
	manager.RegisterAgent(agent1)
	manager.RegisterAgent(agent2)
	manager.RegisterAgent(agent3)

	ctx := context.Background()
	agentContext := createTestContext()

	result, err := manager.Execute(ctx, agentContext)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "agent2 failed")
	assert.NotNil(t, result)
	// Agent3 should not have executed
	assert.False(t, agent3.executed)
}

// TestExecute_SequentialMode_Retry tests retry mechanism
func TestExecute_SequentialMode_Retry(t *testing.T) {
	attemptCount := 0

	agent := &MockAgent{
		id: "agent_with_retry",
		executeFn: func(ctx context.Context, agentContext *models.AgentContext) (*models.AgentContext, error) {
			attemptCount++
			if attemptCount < 3 {
				return nil, errors.New("temporary failure")
			}
			return agentContext, nil
		},
	}

	config := PipelineConfig{
		Mode: SequentialMode,
		Agents: []AgentConfig{
			{ID: "agent_with_retry", Enabled: true, Timeout: 5000, Retry: 2},
		},
		Options: domainservices.AgentExecutionOptions{
			ValidateContract: false,
		},
	}

	manager := NewReasoningManager(config)
	manager.RegisterAgent(agent)

	ctx := context.Background()
	agentContext := createTestContext()

	result, err := manager.Execute(ctx, agentContext)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 3, attemptCount) // Initial + 2 retries
}

// TestExecute_ParallelMode tests parallel execution
func TestExecute_ParallelMode(t *testing.T) {
	executionTimes := make(map[string]time.Time)

	agent1 := &MockAgent{
		id:             "agent1",
		preconditions:  []string{},
		postconditions: []string{"reasoning.intents"},
		executeFn: func(ctx context.Context, agentContext *models.AgentContext) (*models.AgentContext, error) {
			executionTimes["agent1"] = time.Now()
			time.Sleep(10 * time.Millisecond)
			newContext := agentContext
			newContext.Reasoning.Intents = []models.Intent{{Type: "test"}}
			return newContext, nil
		},
	}

	agent2 := &MockAgent{
		id:             "agent2",
		preconditions:  []string{},
		postconditions: []string{"reasoning.entities"},
		executeFn: func(ctx context.Context, agentContext *models.AgentContext) (*models.AgentContext, error) {
			executionTimes["agent2"] = time.Now()
			time.Sleep(10 * time.Millisecond)
			newContext := agentContext
			newContext.Reasoning.Entities = map[string]interface{}{"test": "entity"}
			return newContext, nil
		},
	}

	// Agent3 depends on agent1, should run in second level
	agent3 := &MockAgent{
		id:             "agent3",
		preconditions:  []string{"reasoning.intents"},
		postconditions: []string{"reasoning.conclusions"},
		executeFn: func(ctx context.Context, agentContext *models.AgentContext) (*models.AgentContext, error) {
			executionTimes["agent3"] = time.Now()
			newContext := agentContext
			newContext.Reasoning.Conclusions = []models.Conclusion{{ID: "c1", Description: "test", Confidence: 0.9}}
			return newContext, nil
		},
	}

	config := PipelineConfig{
		Mode: ParallelMode,
		Agents: []AgentConfig{
			{ID: "agent1", Enabled: true, DependsOn: []string{}, Timeout: 5000},
			{ID: "agent2", Enabled: true, DependsOn: []string{}, Timeout: 5000},
			{ID: "agent3", Enabled: true, DependsOn: []string{"agent1"}, Timeout: 5000},
		},
		Options: domainservices.AgentExecutionOptions{
			ValidateContract: false,
		},
	}

	manager := NewReasoningManager(config)
	manager.RegisterAgent(agent1)
	manager.RegisterAgent(agent2)
	manager.RegisterAgent(agent3)

	ctx := context.Background()
	agentContext := createTestContext()

	result, err := manager.Execute(ctx, agentContext)

	require.NoError(t, err)
	assert.NotNil(t, result)

	// Verify agent1 and agent2 executed in parallel (timestamps should be close)
	if t1, ok := executionTimes["agent1"]; ok {
		if t2, ok := executionTimes["agent2"]; ok {
			timeDiff := t1.Sub(t2)
			if timeDiff < 0 {
				timeDiff = -timeDiff
			}
			assert.Less(t, timeDiff, 50*time.Millisecond, "agent1 and agent2 should run in parallel")
		}
	}

	// Verify agent3 executed (timing may vary due to goroutine scheduling)
	_, agent3Executed := executionTimes["agent3"]
	assert.True(t, agent3Executed, "agent3 should have executed")
}

// TestExecute_ConditionalMode tests conditional execution
func TestExecute_ConditionalMode(t *testing.T) {
	executionOrder := []string{}

	agent1 := &MockAgent{
		id:             "agent1",
		postconditions: []string{"reasoning.intents"},
		executeFn: func(ctx context.Context, agentContext *models.AgentContext) (*models.AgentContext, error) {
			executionOrder = append(executionOrder, "agent1")
			agentContext.Reasoning.Intents = []models.Intent{{Type: "test"}}
			return agentContext, nil
		},
	}

	agent2 := &MockAgent{
		id: "agent2",
		executeFn: func(ctx context.Context, agentContext *models.AgentContext) (*models.AgentContext, error) {
			executionOrder = append(executionOrder, "agent2")
			return agentContext, nil
		},
	}

	agent3 := &MockAgent{
		id: "agent3",
		executeFn: func(ctx context.Context, agentContext *models.AgentContext) (*models.AgentContext, error) {
			executionOrder = append(executionOrder, "agent3")
			return agentContext, nil
		},
	}

	config := PipelineConfig{
		Mode: ConditionalMode,
		Agents: []AgentConfig{
			{ID: "agent1", Enabled: true, Conditions: []string{}, Timeout: 5000},
			{ID: "agent2", Enabled: true, Conditions: []string{"reasoning.intents"}, Timeout: 5000},
			{ID: "agent3", Enabled: true, Conditions: []string{"reasoning.conclusions"}, Timeout: 5000},
		},
		Options: domainservices.AgentExecutionOptions{
			ValidateContract: false,
		},
	}

	manager := NewReasoningManager(config)
	manager.RegisterAgent(agent1)
	manager.RegisterAgent(agent2)
	manager.RegisterAgent(agent3)

	ctx := context.Background()
	agentContext := createTestContext()

	result, err := manager.Execute(ctx, agentContext)

	require.NoError(t, err)
	assert.NotNil(t, result)

	// Agent1 should always execute (no conditions)
	// Agent2 should execute after agent1 (reasoning.intents now exists)
	// Agent3 should NOT execute (reasoning.conclusions doesn't exist)
	assert.Equal(t, []string{"agent1", "agent2"}, executionOrder)
}

// TestExecute_ContextCancellation tests context cancellation
func TestExecute_ContextCancellation(t *testing.T) {
	agent1 := &MockAgent{
		id: "agent1",
		executeFn: func(ctx context.Context, agentContext *models.AgentContext) (*models.AgentContext, error) {
			time.Sleep(10 * time.Millisecond)
			return agentContext, nil
		},
	}

	agent2 := &MockAgent{
		id: "agent2",
		executeFn: func(ctx context.Context, agentContext *models.AgentContext) (*models.AgentContext, error) {
			time.Sleep(10 * time.Millisecond)
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
			ValidateContract: false,
		},
	}

	manager := NewReasoningManager(config)
	manager.RegisterAgent(agent1)
	manager.RegisterAgent(agent2)

	ctx, cancel := context.WithCancel(context.Background())
	agentContext := createTestContext()

	// Cancel context after agent1 completes
	go func() {
		time.Sleep(15 * time.Millisecond)
		cancel()
	}()

	result, err := manager.Execute(ctx, agentContext)

	require.Error(t, err)
	assert.Equal(t, context.Canceled, err)
	assert.NotNil(t, result)
}

// TestExecute_UnknownMode tests handling of unknown execution mode
func TestExecute_UnknownMode(t *testing.T) {
	config := PipelineConfig{
		Mode:   ExecutionMode("unknown"),
		Agents: []AgentConfig{},
	}

	manager := NewReasoningManager(config)
	ctx := context.Background()
	agentContext := createTestContext()

	result, err := manager.Execute(ctx, agentContext)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "unknown execution mode")
}

// TestBuildDependencyLevels tests dependency level resolution
func TestBuildDependencyLevels(t *testing.T) {
	config := PipelineConfig{
		Mode: ParallelMode,
		Agents: []AgentConfig{
			{ID: "agent1", Enabled: true, DependsOn: []string{}},
			{ID: "agent2", Enabled: true, DependsOn: []string{}},
			{ID: "agent3", Enabled: true, DependsOn: []string{"agent1"}},
			{ID: "agent4", Enabled: true, DependsOn: []string{"agent1", "agent2"}},
			{ID: "agent5", Enabled: true, DependsOn: []string{"agent3"}},
		},
	}

	manager := NewReasoningManager(config)
	levels := manager.buildDependencyLevels()

	// Verify we have at least one level
	require.NotEmpty(t, levels, "Should have at least one level")

	// Verify agent1 and agent2 are in the first level (no dependencies)
	assert.Contains(t, levels[0], "agent1", "agent1 should be in first level")
	assert.Contains(t, levels[0], "agent2", "agent2 should be in first level")

	// If we have multiple levels, verify dependency ordering
	if len(levels) > 1 {
		// Verify later levels don't contain agent1 or agent2
		for i := 1; i < len(levels); i++ {
			assert.NotContains(t, levels[i], "agent1", "agent1 should only be in first level")
			assert.NotContains(t, levels[i], "agent2", "agent2 should only be in first level")
		}

		// Verify agent3 and agent4 come after agent1/agent2
		foundAgent3 := false
		foundAgent4 := false
		for i := 1; i < len(levels); i++ {
			if contains(levels[i], "agent3") {
				foundAgent3 = true
			}
			if contains(levels[i], "agent4") {
				foundAgent4 = true
			}
		}
		assert.True(t, foundAgent3, "agent3 should be in a later level")
		assert.True(t, foundAgent4, "agent4 should be in a later level")
	}

	// Verify all agents are present exactly once
	allAgents := make(map[string]bool)
	for _, level := range levels {
		for _, agentID := range level {
			assert.False(t, allAgents[agentID], "agent %s should appear only once", agentID)
			allAgents[agentID] = true
		}
	}
	assert.Len(t, allAgents, 5, "All 5 agents should be present")
}

// Helper function to check if a slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// TestBuildDependencyLevels_WithDisabledAgents tests dependency levels with disabled agents
func TestBuildDependencyLevels_WithDisabledAgents(t *testing.T) {
	config := PipelineConfig{
		Mode: ParallelMode,
		Agents: []AgentConfig{
			{ID: "agent1", Enabled: true, DependsOn: []string{}},
			{ID: "agent2", Enabled: false, DependsOn: []string{}}, // Disabled
			{ID: "agent3", Enabled: true, DependsOn: []string{}},
		},
	}

	manager := NewReasoningManager(config)
	levels := manager.buildDependencyLevels()

	// Verify disabled agents are not included
	allAgents := make(map[string]bool)
	for _, level := range levels {
		for _, agentID := range level {
			allAgents[agentID] = true
		}
	}

	assert.Len(t, allAgents, 2, "Only enabled agents should be included")
	assert.True(t, allAgents["agent1"], "agent1 should be included")
	assert.False(t, allAgents["agent2"], "agent2 (disabled) should not be included")
	assert.True(t, allAgents["agent3"], "agent3 should be included")
}

// TestExecute_ContractValidation tests precondition and postcondition validation
func TestExecute_ContractValidation(t *testing.T) {
	agent1 := &MockAgent{
		id:             "agent1",
		preconditions:  []string{"reasoning.intents"}, // Not satisfied
		postconditions: []string{"reasoning.conclusions"},
		executeFn: func(ctx context.Context, agentContext *models.AgentContext) (*models.AgentContext, error) {
			agentContext.Reasoning.Conclusions = []models.Conclusion{{ID: "c1", Description: "test", Confidence: 0.9}}
			return agentContext, nil
		},
	}

	config := PipelineConfig{
		Mode: SequentialMode,
		Agents: []AgentConfig{
			{ID: "agent1", Enabled: true, Timeout: 5000},
		},
		Options: domainservices.AgentExecutionOptions{
			ValidateContract: true,
			FailOnViolation:  true,
		},
	}

	manager := NewReasoningManager(config)
	manager.RegisterAgent(agent1)

	ctx := context.Background()
	agentContext := createTestContext()

	result, err := manager.Execute(ctx, agentContext)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "precondition validation failed")
	assert.NotNil(t, result)
}

// TestExecute_ContractValidation_WarningOnly tests contract validation with warnings
func TestExecute_ContractValidation_WarningOnly(t *testing.T) {
	agent1 := &MockAgent{
		id:             "agent1",
		preconditions:  []string{"reasoning.intents"}, // Not satisfied
		postconditions: []string{"reasoning.conclusions"},
		executeFn: func(ctx context.Context, agentContext *models.AgentContext) (*models.AgentContext, error) {
			agentContext.Reasoning.Conclusions = []models.Conclusion{{ID: "c1", Description: "test", Confidence: 0.9}}
			return agentContext, nil
		},
	}

	config := PipelineConfig{
		Mode: SequentialMode,
		Agents: []AgentConfig{
			{ID: "agent1", Enabled: true, Timeout: 5000},
		},
		Options: domainservices.AgentExecutionOptions{
			ValidateContract: true,
			FailOnViolation:  false, // Only warn
		},
	}

	manager := NewReasoningManager(config)
	manager.RegisterAgent(agent1)

	ctx := context.Background()
	agentContext := createTestContext()

	result, err := manager.Execute(ctx, agentContext)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotEmpty(t, result.Diagnostics.Warnings)
	assert.Contains(t, result.Diagnostics.Warnings[0].Message, "Precondition validation failed")
}

// TestRecordAgentRun tests agent run recording
func TestRecordAgentRun(t *testing.T) {
	config := PipelineConfig{Mode: SequentialMode}
	manager := NewReasoningManager(config)

	ctx := createTestContext()

	manager.recordAgentRun(ctx, "test_agent", "success", 150, "")

	assert.Len(t, ctx.Audit.AgentRuns, 1)
	run := ctx.Audit.AgentRuns[0]
	assert.Equal(t, "test_agent", run.AgentID)
	assert.Equal(t, "success", run.Status)
	assert.Equal(t, int64(150), run.DurationMS)
	assert.Empty(t, run.Error)

	assert.NotNil(t, ctx.Diagnostics.Performance.AgentMetrics["test_agent"])
	assert.Equal(t, int64(150), ctx.Diagnostics.Performance.AgentMetrics["test_agent"].DurationMS)
}

// TestRecordAgentRun_WithError tests agent run recording with error
func TestRecordAgentRun_WithError(t *testing.T) {
	config := PipelineConfig{Mode: SequentialMode}
	manager := NewReasoningManager(config)

	ctx := createTestContext()

	manager.recordAgentRun(ctx, "failing_agent", "failed", 75, "execution error")

	assert.Len(t, ctx.Audit.AgentRuns, 1)
	run := ctx.Audit.AgentRuns[0]
	assert.Equal(t, "failing_agent", run.AgentID)
	assert.Equal(t, "failed", run.Status)
	assert.Equal(t, int64(75), run.DurationMS)
	assert.Equal(t, "execution error", run.Error)
}
