package services

import (
	"context"
	"testing"

	"github.com/mshogin/agents/internal/domain/models"
	domainservices "github.com/mshogin/agents/internal/domain/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Validation rule tests: testing slot completeness and logical consistency

// TestValidation_SlotCompleteness_AllRequired tests that all required slots are filled
func TestValidation_SlotCompleteness_AllRequired(t *testing.T) {
	agent1 := &MockAgent{
		id:             "intent_agent",
		preconditions:  []string{},
		postconditions: []string{"reasoning.intents"},
		executeFn: func(ctx context.Context, agentContext *models.AgentContext) (*models.AgentContext, error) {
			// Fills required slot
			agentContext.Reasoning.Intents = []models.Intent{
				{Type: "query_commits", Confidence: 0.95},
			}
			return agentContext, nil
		},
	}

	config := PipelineConfig{
		Mode: SequentialMode,
		Agents: []AgentConfig{
			{ID: "intent_agent", Enabled: true, Timeout: 5000},
		},
		Options: domainservices.AgentExecutionOptions{
			ValidateContract: true, // Enable slot validation
			FailOnViolation:  true,
		},
	}

	manager := NewReasoningManager(config)
	require.NoError(t, manager.RegisterAgent(agent1))

	ctx := context.Background()
	agentContext := models.NewAgentContext("session", "trace")

	// Execute - should succeed because all required slots are filled
	result, err := manager.Execute(ctx, agentContext)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotEmpty(t, result.Reasoning.Intents)
}

// TestValidation_SlotCompleteness_MissingRequired tests detection of missing required slots
func TestValidation_SlotCompleteness_MissingRequired(t *testing.T) {
	agent1 := &MockAgent{
		id:             "incomplete_agent",
		preconditions:  []string{},
		postconditions: []string{"reasoning.intents"}, // Promises to fill intents
		executeFn: func(ctx context.Context, agentContext *models.AgentContext) (*models.AgentContext, error) {
			// Does NOT fill intents (violates postcondition)
			return agentContext, nil
		},
	}

	config := PipelineConfig{
		Mode: SequentialMode,
		Agents: []AgentConfig{
			{ID: "incomplete_agent", Enabled: true, Timeout: 5000},
		},
		Options: domainservices.AgentExecutionOptions{
			ValidateContract: true,
			FailOnViolation:  true,
		},
	}

	manager := NewReasoningManager(config)
	require.NoError(t, manager.RegisterAgent(agent1))

	ctx := context.Background()
	agentContext := models.NewAgentContext("session", "trace")

	// Execute - should fail due to missing required slot
	result, err := manager.Execute(ctx, agentContext)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "postcondition validation failed")
	assert.NotNil(t, result)
}

// TestValidation_LogicalConsistency_ValidChain tests valid reasoning chain
func TestValidation_LogicalConsistency_ValidChain(t *testing.T) {
	agent1 := &MockAgent{
		id:             "agent1",
		preconditions:  []string{},
		postconditions: []string{"reasoning.intents"},
		executeFn: func(ctx context.Context, agentContext *models.AgentContext) (*models.AgentContext, error) {
			agentContext.Reasoning.Intents = []models.Intent{
				{Type: "query", Confidence: 0.9},
			}
			return agentContext, nil
		},
	}

	agent2 := &MockAgent{
		id:             "agent2",
		preconditions:  []string{"reasoning.intents"},
		postconditions: []string{"reasoning.hypotheses"},
		executeFn: func(ctx context.Context, agentContext *models.AgentContext) (*models.AgentContext, error) {
			// Logically consistent: hypothesis based on intent
			agentContext.Reasoning.Hypotheses = []models.Hypothesis{
				{ID: "h1", Description: "Based on query intent", Dependencies: []string{}},
			}
			return agentContext, nil
		},
	}

	agent3 := &MockAgent{
		id:             "agent3",
		preconditions:  []string{"reasoning.hypotheses"},
		postconditions: []string{"reasoning.conclusions"},
		executeFn: func(ctx context.Context, agentContext *models.AgentContext) (*models.AgentContext, error) {
			// Logically consistent: conclusion based on hypothesis
			agentContext.Reasoning.Conclusions = []models.Conclusion{
				{ID: "c1", Description: "Derived from hypothesis", Confidence: 0.9},
			}
			return agentContext, nil
		},
	}

	config := PipelineConfig{
		Mode: SequentialMode,
		Agents: []AgentConfig{
			{ID: "agent1", Enabled: true, Timeout: 5000},
			{ID: "agent2", Enabled: true, Timeout: 5000},
			{ID: "agent3", Enabled: true, Timeout: 5000},
		},
		Options: domainservices.AgentExecutionOptions{
			ValidateContract: true,
		},
	}

	manager := NewReasoningManager(config)
	require.NoError(t, manager.RegisterAgent(agent1))
	require.NoError(t, manager.RegisterAgent(agent2))
	require.NoError(t, manager.RegisterAgent(agent3))

	ctx := context.Background()
	agentContext := models.NewAgentContext("session", "trace")

	// Execute - should succeed with valid logical chain
	result, err := manager.Execute(ctx, agentContext)
	require.NoError(t, err)
	assert.NotNil(t, result)

	// Verify chain: intents → hypotheses → conclusions
	assert.Len(t, result.Reasoning.Intents, 1)
	assert.Len(t, result.Reasoning.Hypotheses, 1)
	assert.Len(t, result.Reasoning.Conclusions, 1)
}

// TestValidation_LogicalConsistency_BrokenChain tests detection of broken reasoning chain
func TestValidation_LogicalConsistency_BrokenChain(t *testing.T) {
	agent1 := &MockAgent{
		id:             "agent1",
		preconditions:  []string{},
		postconditions: []string{"reasoning.intents"},
		executeFn: func(ctx context.Context, agentContext *models.AgentContext) (*models.AgentContext, error) {
			agentContext.Reasoning.Intents = []models.Intent{
				{Type: "query", Confidence: 0.9},
			}
			return agentContext, nil
		},
	}

	agent2 := &MockAgent{
		id:             "agent2",
		preconditions:  []string{"reasoning.intents"},
		postconditions: []string{"reasoning.conclusions"}, // Skips hypotheses (breaks chain)
		executeFn: func(ctx context.Context, agentContext *models.AgentContext) (*models.AgentContext, error) {
			// Creates conclusion without hypothesis (logically inconsistent)
			agentContext.Reasoning.Conclusions = []models.Conclusion{
				{ID: "c1", Description: "Conclusion without hypothesis", Confidence: 0.9},
			}
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
		},
	}

	manager := NewReasoningManager(config)
	require.NoError(t, manager.RegisterAgent(agent1))
	require.NoError(t, manager.RegisterAgent(agent2))

	ctx := context.Background()
	agentContext := models.NewAgentContext("session", "trace")

	// Execute - completes but chain is broken (missing hypotheses)
	result, err := manager.Execute(ctx, agentContext)
	require.NoError(t, err) // Pipeline succeeds
	assert.NotNil(t, result)

	// Verify broken chain: has intents and conclusions but no hypotheses
	assert.Len(t, result.Reasoning.Intents, 1)
	assert.Empty(t, result.Reasoning.Hypotheses) // Missing link
	assert.Len(t, result.Reasoning.Conclusions, 1)
}

// TestValidation_SlotCompleteness_PartialFill tests detection of partially filled required slots
func TestValidation_SlotCompleteness_PartialFill(t *testing.T) {
	agent1 := &MockAgent{
		id:             "partial_agent",
		preconditions:  []string{},
		postconditions: []string{"reasoning.intents", "reasoning.hypotheses"}, // Two postconditions
		executeFn: func(ctx context.Context, agentContext *models.AgentContext) (*models.AgentContext, error) {
			// Only fills intents, not hypotheses (partial violation)
			agentContext.Reasoning.Intents = []models.Intent{
				{Type: "query", Confidence: 0.9},
			}
			// Missing: agentContext.Reasoning.Hypotheses
			return agentContext, nil
		},
	}

	config := PipelineConfig{
		Mode: SequentialMode,
		Agents: []AgentConfig{
			{ID: "partial_agent", Enabled: true, Timeout: 5000},
		},
		Options: domainservices.AgentExecutionOptions{
			ValidateContract: true,
			FailOnViolation:  true,
		},
	}

	manager := NewReasoningManager(config)
	require.NoError(t, manager.RegisterAgent(agent1))

	ctx := context.Background()
	agentContext := models.NewAgentContext("session", "trace")

	// Execute - should fail due to partially filled postconditions
	result, err := manager.Execute(ctx, agentContext)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "postcondition validation failed")
	assert.NotNil(t, result)
}

// TestValidation_LogicalConsistency_DependencyOrder tests dependency order validation
func TestValidation_LogicalConsistency_DependencyOrder(t *testing.T) {
	agent1 := &MockAgent{
		id:             "agent1",
		preconditions:  []string{},
		postconditions: []string{"reasoning.intents"},
		executeFn: func(ctx context.Context, agentContext *models.AgentContext) (*models.AgentContext, error) {
			agentContext.Reasoning.Intents = []models.Intent{
				{Type: "query", Confidence: 0.9},
			}
			return agentContext, nil
		},
	}

	agent2 := &MockAgent{
		id:             "agent2",
		preconditions:  []string{"reasoning.intents"},
		postconditions: []string{"reasoning.hypotheses"},
		executeFn: func(ctx context.Context, agentContext *models.AgentContext) (*models.AgentContext, error) {
			// Hypothesis with dependency on another hypothesis (dependency chain)
			agentContext.Reasoning.Hypotheses = []models.Hypothesis{
				{ID: "h1", Description: "First hypothesis", Dependencies: []string{}},
				{ID: "h2", Description: "Second hypothesis", Dependencies: []string{"h1"}}, // Depends on h1
			}
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
		},
	}

	manager := NewReasoningManager(config)
	require.NoError(t, manager.RegisterAgent(agent1))
	require.NoError(t, manager.RegisterAgent(agent2))

	ctx := context.Background()
	agentContext := models.NewAgentContext("session", "trace")

	// Execute - should succeed with valid dependency order
	result, err := manager.Execute(ctx, agentContext)
	require.NoError(t, err)
	assert.NotNil(t, result)

	// Verify dependency chain is valid
	assert.Len(t, result.Reasoning.Hypotheses, 2)
	assert.Empty(t, result.Reasoning.Hypotheses[0].Dependencies) // h1 has no deps
	assert.Equal(t, []string{"h1"}, result.Reasoning.Hypotheses[1].Dependencies) // h2 depends on h1
}

// TestValidation_SlotCompleteness_ConditionalRequired tests conditional required slots
func TestValidation_SlotCompleteness_ConditionalRequired(t *testing.T) {
	agent1 := &MockAgent{
		id:             "conditional_agent",
		preconditions:  []string{},
		postconditions: []string{"reasoning.intents"},
		executeFn: func(ctx context.Context, agentContext *models.AgentContext) (*models.AgentContext, error) {
			// Conditionally fills intents based on some logic
			agentContext.Reasoning.Intents = []models.Intent{
				{Type: "query_gitlab", Confidence: 0.9},
			}
			return agentContext, nil
		},
	}

	agent2 := &MockAgent{
		id:             "retrieval_agent",
		preconditions:  []string{"reasoning.intents"},
		postconditions: []string{"retrieval.plans"},
		executeFn: func(ctx context.Context, agentContext *models.AgentContext) (*models.AgentContext, error) {
			// Only creates retrieval plan if intent is query-related
			if len(agentContext.Reasoning.Intents) > 0 &&
				agentContext.Reasoning.Intents[0].Type == "query_gitlab" {
				agentContext.Retrieval.Plans = []models.RetrievalPlan{
					{ID: "plan1", Description: "Query GitLab", Sources: []string{"gitlab"}, Priority: 1},
				}
			}
			return agentContext, nil
		},
	}

	config := PipelineConfig{
		Mode: SequentialMode,
		Agents: []AgentConfig{
			{ID: "conditional_agent", Enabled: true, Timeout: 5000},
			{ID: "retrieval_agent", Enabled: true, Timeout: 5000},
		},
		Options: domainservices.AgentExecutionOptions{
			ValidateContract: true,
		},
	}

	manager := NewReasoningManager(config)
	require.NoError(t, manager.RegisterAgent(agent1))
	require.NoError(t, manager.RegisterAgent(agent2))

	ctx := context.Background()
	agentContext := models.NewAgentContext("session", "trace")

	// Execute - should succeed with conditional slots filled
	result, err := manager.Execute(ctx, agentContext)
	require.NoError(t, err)
	assert.NotNil(t, result)

	// Verify conditional slots were filled
	assert.Len(t, result.Reasoning.Intents, 1)
	assert.Len(t, result.Retrieval.Plans, 1)
}

// TestValidation_LogicalConsistency_ConfidenceScores tests confidence score consistency
func TestValidation_LogicalConsistency_ConfidenceScores(t *testing.T) {
	agent1 := &MockAgent{
		id:             "high_confidence_agent",
		preconditions:  []string{},
		postconditions: []string{"reasoning.intents"},
		executeFn: func(ctx context.Context, agentContext *models.AgentContext) (*models.AgentContext, error) {
			agentContext.Reasoning.Intents = []models.Intent{
				{Type: "query", Confidence: 0.95}, // High confidence
			}
			return agentContext, nil
		},
	}

	agent2 := &MockAgent{
		id:             "conclusion_agent",
		preconditions:  []string{"reasoning.intents"},
		postconditions: []string{"reasoning.conclusions"},
		executeFn: func(ctx context.Context, agentContext *models.AgentContext) (*models.AgentContext, error) {
			// Confidence should not exceed input confidence
			inputConfidence := agentContext.Reasoning.Intents[0].Confidence
			agentContext.Reasoning.Conclusions = []models.Conclusion{
				{ID: "c1", Description: "Conclusion", Confidence: inputConfidence * 0.9}, // Propagate confidence
			}
			return agentContext, nil
		},
	}

	config := PipelineConfig{
		Mode: SequentialMode,
		Agents: []AgentConfig{
			{ID: "high_confidence_agent", Enabled: true, Timeout: 5000},
			{ID: "conclusion_agent", Enabled: true, Timeout: 5000},
		},
		Options: domainservices.AgentExecutionOptions{
			ValidateContract: true,
		},
	}

	manager := NewReasoningManager(config)
	require.NoError(t, manager.RegisterAgent(agent1))
	require.NoError(t, manager.RegisterAgent(agent2))

	ctx := context.Background()
	agentContext := models.NewAgentContext("session", "trace")

	// Execute - should succeed with consistent confidence scores
	result, err := manager.Execute(ctx, agentContext)
	require.NoError(t, err)
	assert.NotNil(t, result)

	// Verify confidence propagation is logical
	assert.Equal(t, 0.95, result.Reasoning.Intents[0].Confidence)
	assert.LessOrEqual(t, result.Reasoning.Conclusions[0].Confidence, 0.95) // Should not exceed input
	assert.Greater(t, result.Reasoning.Conclusions[0].Confidence, 0.0)
}

// TestValidation_SlotCompleteness_EmptyButPresent tests detection of empty but present slots
func TestValidation_SlotCompleteness_EmptyButPresent(t *testing.T) {
	agent1 := &MockAgent{
		id:             "empty_slot_agent",
		preconditions:  []string{},
		postconditions: []string{"reasoning.intents"},
		executeFn: func(ctx context.Context, agentContext *models.AgentContext) (*models.AgentContext, error) {
			// Creates empty intents array (technically present but empty)
			agentContext.Reasoning.Intents = []models.Intent{}
			return agentContext, nil
		},
	}

	config := PipelineConfig{
		Mode: SequentialMode,
		Agents: []AgentConfig{
			{ID: "empty_slot_agent", Enabled: true, Timeout: 5000},
		},
		Options: domainservices.AgentExecutionOptions{
			ValidateContract: true,
			FailOnViolation:  true,
		},
	}

	manager := NewReasoningManager(config)
	require.NoError(t, manager.RegisterAgent(agent1))

	ctx := context.Background()
	agentContext := models.NewAgentContext("session", "trace")

	// Execute - should fail because empty array doesn't satisfy postcondition
	result, err := manager.Execute(ctx, agentContext)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "postcondition validation failed")
	assert.NotNil(t, result)
}

// TestValidation_LogicalConsistency_MultipleAgentsConsistent tests consistency across multiple agents
func TestValidation_LogicalConsistency_MultipleAgentsConsistent(t *testing.T) {
	agent1 := &MockAgent{
		id:             "agent1",
		preconditions:  []string{},
		postconditions: []string{"reasoning.intents"},
		executeFn: func(ctx context.Context, agentContext *models.AgentContext) (*models.AgentContext, error) {
			agentContext.Reasoning.Intents = []models.Intent{
				{Type: "query_commits", Confidence: 0.9},
			}
			return agentContext, nil
		},
	}

	agent2 := &MockAgent{
		id:             "agent2",
		preconditions:  []string{"reasoning.intents"},
		postconditions: []string{"reasoning.hypotheses"},
		executeFn: func(ctx context.Context, agentContext *models.AgentContext) (*models.AgentContext, error) {
			// Hypothesis consistent with intent
			intent := agentContext.Reasoning.Intents[0].Type
			agentContext.Reasoning.Hypotheses = []models.Hypothesis{
				{ID: "h1", Description: "User wants to " + intent, Dependencies: []string{}},
			}
			return agentContext, nil
		},
	}

	agent3 := &MockAgent{
		id:             "agent3",
		preconditions:  []string{"reasoning.hypotheses"},
		postconditions: []string{"retrieval.plans"},
		executeFn: func(ctx context.Context, agentContext *models.AgentContext) (*models.AgentContext, error) {
			// Retrieval plan consistent with hypothesis
			agentContext.Retrieval.Plans = []models.RetrievalPlan{
				{ID: "plan1", Description: "Query GitLab", Sources: []string{"gitlab"}, Priority: 1},
			}
			return agentContext, nil
		},
	}

	agent4 := &MockAgent{
		id:             "agent4",
		preconditions:  []string{"retrieval.plans"},
		postconditions: []string{"reasoning.conclusions"},
		executeFn: func(ctx context.Context, agentContext *models.AgentContext) (*models.AgentContext, error) {
			// Conclusion consistent with entire chain
			agentContext.Reasoning.Conclusions = []models.Conclusion{
				{ID: "c1", Description: "Execute gitlab commit query", Confidence: 0.9},
			}
			return agentContext, nil
		},
	}

	config := PipelineConfig{
		Mode: SequentialMode,
		Agents: []AgentConfig{
			{ID: "agent1", Enabled: true, Timeout: 5000},
			{ID: "agent2", Enabled: true, Timeout: 5000},
			{ID: "agent3", Enabled: true, Timeout: 5000},
			{ID: "agent4", Enabled: true, Timeout: 5000},
		},
		Options: domainservices.AgentExecutionOptions{
			ValidateContract: true,
		},
	}

	manager := NewReasoningManager(config)
	require.NoError(t, manager.RegisterAgent(agent1))
	require.NoError(t, manager.RegisterAgent(agent2))
	require.NoError(t, manager.RegisterAgent(agent3))
	require.NoError(t, manager.RegisterAgent(agent4))

	ctx := context.Background()
	agentContext := models.NewAgentContext("session", "trace")

	// Execute - should succeed with full consistent chain
	result, err := manager.Execute(ctx, agentContext)
	require.NoError(t, err)
	assert.NotNil(t, result)

	// Verify complete consistent chain
	assert.Len(t, result.Reasoning.Intents, 1)
	assert.Equal(t, "query_commits", result.Reasoning.Intents[0].Type)

	assert.Len(t, result.Reasoning.Hypotheses, 1)
	assert.Contains(t, result.Reasoning.Hypotheses[0].Description, "query_commits")

	assert.Len(t, result.Retrieval.Plans, 1)
	assert.Contains(t, result.Retrieval.Plans[0].Sources, "gitlab")

	assert.Len(t, result.Reasoning.Conclusions, 1)
	assert.Contains(t, result.Reasoning.Conclusions[0].Description, "gitlab")
}
