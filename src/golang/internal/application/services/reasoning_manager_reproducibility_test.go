package services

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/mshogin/agents/internal/domain/models"
	domainservices "github.com/mshogin/agents/internal/domain/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Reproducibility test cases: testing deterministic behavior with fixed configuration

// TestReproducibility_IdenticalInput tests that identical inputs produce identical outputs
func TestReproducibility_IdenticalInput(t *testing.T) {
	agent1 := &MockAgent{
		id:             "deterministic_agent",
		preconditions:  []string{},
		postconditions: []string{"reasoning.intents"},
		executeFn: func(ctx context.Context, agentContext *models.AgentContext) (*models.AgentContext, error) {
			// Deterministic agent: always produces same output for same input
			agentContext.Reasoning.Intents = []models.Intent{
				{Type: "query", Confidence: 0.95},
				{Type: "action", Confidence: 0.85},
			}
			return agentContext, nil
		},
	}

	config := PipelineConfig{
		Mode: SequentialMode,
		Agents: []AgentConfig{
			{ID: "deterministic_agent", Enabled: true, Timeout: 5000},
		},
		Options: domainservices.AgentExecutionOptions{
			ValidateContract: true,
			TrackPerformance: false, // Disable performance tracking for deterministic results
		},
	}

	manager := NewReasoningManager(config)
	require.NoError(t, manager.RegisterAgent(agent1))

	ctx := context.Background()

	// Run 1
	agentContext1 := models.NewAgentContext("repro-session-1", "repro-trace-1")
	result1, err1 := manager.Execute(ctx, agentContext1)
	require.NoError(t, err1)

	// Run 2 with identical input
	agentContext2 := models.NewAgentContext("repro-session-1", "repro-trace-1")
	result2, err2 := manager.Execute(ctx, agentContext2)
	require.NoError(t, err2)

	// Results should be identical (excluding performance metrics and timestamps)
	assert.Equal(t, len(result1.Reasoning.Intents), len(result2.Reasoning.Intents))
	assert.Equal(t, result1.Reasoning.Intents[0].Type, result2.Reasoning.Intents[0].Type)
	assert.Equal(t, result1.Reasoning.Intents[0].Confidence, result2.Reasoning.Intents[0].Confidence)
	assert.Equal(t, result1.Reasoning.Intents[1].Type, result2.Reasoning.Intents[1].Type)
	assert.Equal(t, result1.Reasoning.Intents[1].Confidence, result2.Reasoning.Intents[1].Confidence)
}

// TestReproducibility_MultiplePipelineRuns tests reproducibility across multiple runs
func TestReproducibility_MultiplePipelineRuns(t *testing.T) {
	agent1 := &MockAgent{
		id:             "agent1",
		preconditions:  []string{},
		postconditions: []string{"reasoning.intents"},
		executeFn: func(ctx context.Context, agentContext *models.AgentContext) (*models.AgentContext, error) {
			agentContext.Reasoning.Intents = []models.Intent{{Type: "test", Confidence: 0.9}}
			return agentContext, nil
		},
	}

	agent2 := &MockAgent{
		id:             "agent2",
		preconditions:  []string{"reasoning.intents"},
		postconditions: []string{"reasoning.conclusions"},
		executeFn: func(ctx context.Context, agentContext *models.AgentContext) (*models.AgentContext, error) {
			// Deterministic processing based on intents
			intent := agentContext.Reasoning.Intents[0].Type
			agentContext.Reasoning.Conclusions = []models.Conclusion{
				{ID: "c1", Description: "Processed: " + intent, Confidence: 0.9},
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
			TrackPerformance: false,
		},
	}

	manager := NewReasoningManager(config)
	require.NoError(t, manager.RegisterAgent(agent1))
	require.NoError(t, manager.RegisterAgent(agent2))

	ctx := context.Background()

	var results []*models.AgentContext
	const numRuns = 5

	// Run pipeline 5 times
	for i := 0; i < numRuns; i++ {
		agentContext := models.NewAgentContext("repro-session", "repro-trace")
		result, err := manager.Execute(ctx, agentContext)
		require.NoError(t, err)
		results = append(results, result)
	}

	// All results should be identical
	for i := 1; i < numRuns; i++ {
		assert.Equal(t, results[0].Reasoning.Intents, results[i].Reasoning.Intents,
			"Run %d intents differ from run 0", i)
		assert.Equal(t, results[0].Reasoning.Conclusions, results[i].Reasoning.Conclusions,
			"Run %d conclusions differ from run 0", i)
	}
}

// TestReproducibility_ContextSerialization tests that context serialization is deterministic
func TestReproducibility_ContextSerialization(t *testing.T) {
	// Create a complex context
	ctx := models.NewAgentContext("session-1", "trace-1")
	ctx.Reasoning.Intents = []models.Intent{
		{Type: "query", Confidence: 0.95},
		{Type: "action", Confidence: 0.85},
	}
	ctx.Reasoning.Hypotheses = []models.Hypothesis{
		{ID: "h1", Description: "Hypothesis 1", Dependencies: []string{"dep1"}}, // Non-empty dependencies
		{ID: "h2", Description: "Hypothesis 2", Dependencies: []string{"h1"}},
	}
	ctx.Reasoning.Conclusions = []models.Conclusion{
		{ID: "c1", Description: "Conclusion 1", Confidence: 0.9},
	}
	ctx.Enrichment.Facts = []models.Fact{
		{ID: "f1", Content: "Fact 1", Source: "test", Confidence: 0.95},
	}

	// Serialize 3 times
	serialized1, err1 := ctx.Serialize()
	require.NoError(t, err1)

	serialized2, err2 := ctx.Serialize()
	require.NoError(t, err2)

	serialized3, err3 := ctx.Serialize()
	require.NoError(t, err3)

	// All serializations should be identical (byte-for-byte)
	assert.Equal(t, serialized1, serialized2, "Serialization 1 and 2 should be identical")
	assert.Equal(t, serialized2, serialized3, "Serialization 2 and 3 should be identical")

	// Verify deserialization produces equivalent context
	var ctx2 models.AgentContext
	err := json.Unmarshal(serialized1, &ctx2)
	require.NoError(t, err)

	// Compare key fields (not deep equality due to timestamp differences)
	assert.Equal(t, len(ctx.Reasoning.Intents), len(ctx2.Reasoning.Intents))
	assert.Equal(t, ctx.Reasoning.Intents[0].Type, ctx2.Reasoning.Intents[0].Type)
	assert.Equal(t, ctx.Reasoning.Intents[0].Confidence, ctx2.Reasoning.Intents[0].Confidence)

	assert.Equal(t, len(ctx.Reasoning.Hypotheses), len(ctx2.Reasoning.Hypotheses))
	assert.Equal(t, ctx.Reasoning.Hypotheses[0].ID, ctx2.Reasoning.Hypotheses[0].ID)
	assert.Equal(t, ctx.Reasoning.Hypotheses[0].Description, ctx2.Reasoning.Hypotheses[0].Description)

	assert.Equal(t, len(ctx.Reasoning.Conclusions), len(ctx2.Reasoning.Conclusions))
	assert.Equal(t, ctx.Reasoning.Conclusions[0].ID, ctx2.Reasoning.Conclusions[0].ID)

	assert.Equal(t, len(ctx.Enrichment.Facts), len(ctx2.Enrichment.Facts))
	assert.Equal(t, ctx.Enrichment.Facts[0].ID, ctx2.Enrichment.Facts[0].ID)
}

// TestReproducibility_ContextCloning tests that cloning produces identical copies
func TestReproducibility_ContextCloning(t *testing.T) {
	// Create original context
	original := models.NewAgentContext("session-1", "trace-1")
	original.Reasoning.Intents = []models.Intent{
		{Type: "query", Confidence: 0.95},
	}
	original.Reasoning.Hypotheses = []models.Hypothesis{
		{ID: "h1", Description: "Test hypothesis", Dependencies: []string{"dep1"}},
	}
	original.Enrichment.Facts = []models.Fact{
		{ID: "f1", Content: "Test fact", Source: "test", Confidence: 0.95},
	}

	// Clone 3 times
	clone1, err1 := original.Clone()
	require.NoError(t, err1)

	clone2, err2 := original.Clone()
	require.NoError(t, err2)

	clone3, err3 := original.Clone()
	require.NoError(t, err3)

	// All clones should have same content (compare key fields)
	// Use field-by-field comparison instead of DeepEqual to handle nil vs empty slice differences
	assert.Equal(t, len(original.Reasoning.Intents), len(clone1.Reasoning.Intents))
	assert.Equal(t, len(original.Reasoning.Intents), len(clone2.Reasoning.Intents))
	assert.Equal(t, len(original.Reasoning.Intents), len(clone3.Reasoning.Intents))

	assert.Equal(t, original.Reasoning.Intents[0].Type, clone1.Reasoning.Intents[0].Type)
	assert.Equal(t, original.Reasoning.Intents[0].Type, clone2.Reasoning.Intents[0].Type)
	assert.Equal(t, original.Reasoning.Intents[0].Type, clone3.Reasoning.Intents[0].Type)

	assert.Equal(t, len(original.Reasoning.Hypotheses), len(clone1.Reasoning.Hypotheses))
	assert.Equal(t, len(original.Reasoning.Hypotheses), len(clone2.Reasoning.Hypotheses))
	assert.Equal(t, len(original.Reasoning.Hypotheses), len(clone3.Reasoning.Hypotheses))

	assert.Equal(t, len(original.Enrichment.Facts), len(clone1.Enrichment.Facts))
	assert.Equal(t, len(original.Enrichment.Facts), len(clone2.Enrichment.Facts))
	assert.Equal(t, len(original.Enrichment.Facts), len(clone3.Enrichment.Facts))

	// Modify clone1, should not affect others
	clone1.Reasoning.Intents = append(clone1.Reasoning.Intents, models.Intent{Type: "new", Confidence: 0.8})

	// Original and other clones should remain unchanged
	assert.Len(t, original.Reasoning.Intents, 1)
	assert.Len(t, clone2.Reasoning.Intents, 1)
	assert.Len(t, clone3.Reasoning.Intents, 1)
	assert.Len(t, clone1.Reasoning.Intents, 2)
}

// TestReproducibility_AgentOrderMatters tests that agent execution order is deterministic
func TestReproducibility_AgentOrderMatters(t *testing.T) {
	agent1 := &MockAgent{
		id:             "agent1",
		preconditions:  []string{},
		postconditions: []string{"reasoning.intents"},
		executeFn: func(ctx context.Context, agentContext *models.AgentContext) (*models.AgentContext, error) {
			agentContext.Reasoning.Intents = []models.Intent{{Type: "intent1", Confidence: 0.9}}
			return agentContext, nil
		},
	}

	agent2 := &MockAgent{
		id:             "agent2",
		preconditions:  []string{"reasoning.intents"},
		postconditions: []string{"reasoning.hypotheses"},
		executeFn: func(ctx context.Context, agentContext *models.AgentContext) (*models.AgentContext, error) {
			// Depends on agent1's output
			intent := agentContext.Reasoning.Intents[0].Type
			agentContext.Reasoning.Hypotheses = []models.Hypothesis{
				{ID: "h1", Description: "Based on: " + intent, Dependencies: []string{}},
			}
			return agentContext, nil
		},
	}

	agent3 := &MockAgent{
		id:             "agent3",
		preconditions:  []string{"reasoning.hypotheses"},
		postconditions: []string{"reasoning.conclusions"},
		executeFn: func(ctx context.Context, agentContext *models.AgentContext) (*models.AgentContext, error) {
			// Depends on agent2's output
			hypothesis := agentContext.Reasoning.Hypotheses[0].Description
			agentContext.Reasoning.Conclusions = []models.Conclusion{
				{ID: "c1", Description: "Conclusion from: " + hypothesis, Confidence: 0.9},
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
			TrackPerformance: false,
		},
	}

	manager := NewReasoningManager(config)
	require.NoError(t, manager.RegisterAgent(agent1))
	require.NoError(t, manager.RegisterAgent(agent2))
	require.NoError(t, manager.RegisterAgent(agent3))

	ctx := context.Background()

	// Run 3 times
	var results []*models.AgentContext
	for i := 0; i < 3; i++ {
		agentContext := models.NewAgentContext("session", "trace")
		result, err := manager.Execute(ctx, agentContext)
		require.NoError(t, err)
		results = append(results, result)
	}

	// Verify same chain of reasoning in all runs
	expectedConclusion := "Conclusion from: Based on: intent1"
	for i, result := range results {
		assert.Equal(t, "intent1", result.Reasoning.Intents[0].Type, "Run %d: intent mismatch", i)
		assert.Equal(t, "Based on: intent1", result.Reasoning.Hypotheses[0].Description, "Run %d: hypothesis mismatch", i)
		assert.Equal(t, expectedConclusion, result.Reasoning.Conclusions[0].Description, "Run %d: conclusion mismatch", i)
	}
}

// TestReproducibility_EmptyPipeline tests empty pipeline reproducibility
func TestReproducibility_EmptyPipeline(t *testing.T) {
	config := PipelineConfig{
		Mode:    SequentialMode,
		Agents:  []AgentConfig{},
		Options: domainservices.AgentExecutionOptions{},
	}

	manager := NewReasoningManager(config)
	ctx := context.Background()

	// Run empty pipeline 5 times
	var results []*models.AgentContext
	for i := 0; i < 5; i++ {
		agentContext := models.NewAgentContext("session", "trace")
		result, err := manager.Execute(ctx, agentContext)
		require.NoError(t, err)
		results = append(results, result)
	}

	// All results should be empty contexts with same structure
	for i, result := range results {
		assert.Empty(t, result.Reasoning.Intents, "Run %d: should have no intents", i)
		assert.Empty(t, result.Reasoning.Conclusions, "Run %d: should have no conclusions", i)
		assert.Empty(t, result.Audit.AgentRuns, "Run %d: should have no agent runs", i)
	}
}

// TestReproducibility_ConfigChange tests that different configs produce different results
func TestReproducibility_ConfigChange(t *testing.T) {
	agent := &MockAgent{
		id:             "configurable_agent",
		preconditions:  []string{},
		postconditions: []string{"reasoning.intents"},
		executeFn: func(ctx context.Context, agentContext *models.AgentContext) (*models.AgentContext, error) {
			agentContext.Reasoning.Intents = []models.Intent{{Type: "test", Confidence: 0.9}}
			return agentContext, nil
		},
	}

	// Config 1: Agent enabled
	config1 := PipelineConfig{
		Mode: SequentialMode,
		Agents: []AgentConfig{
			{ID: "configurable_agent", Enabled: true, Timeout: 5000},
		},
		Options: domainservices.AgentExecutionOptions{
			ValidateContract: false,
			TrackPerformance: false,
		},
	}

	// Config 2: Agent disabled
	config2 := PipelineConfig{
		Mode: SequentialMode,
		Agents: []AgentConfig{
			{ID: "configurable_agent", Enabled: false, Timeout: 5000},
		},
		Options: domainservices.AgentExecutionOptions{
			ValidateContract: false,
			TrackPerformance: false,
		},
	}

	manager1 := NewReasoningManager(config1)
	require.NoError(t, manager1.RegisterAgent(agent))

	manager2 := NewReasoningManager(config2)
	require.NoError(t, manager2.RegisterAgent(agent))

	ctx := context.Background()

	// Run with config 1 (agent enabled)
	agentContext1 := models.NewAgentContext("session", "trace")
	result1, err1 := manager1.Execute(ctx, agentContext1)
	require.NoError(t, err1)

	// Run with config 2 (agent disabled)
	agentContext2 := models.NewAgentContext("session", "trace")
	result2, err2 := manager2.Execute(ctx, agentContext2)
	require.NoError(t, err2)

	// Results should differ: config1 has intents, config2 doesn't
	assert.NotEmpty(t, result1.Reasoning.Intents, "Config 1 should have intents")
	assert.Empty(t, result2.Reasoning.Intents, "Config 2 should not have intents")
}

// TestReproducibility_ConditionalExecution tests deterministic conditional behavior
func TestReproducibility_ConditionalExecution(t *testing.T) {
	agent1 := &MockAgent{
		id:             "agent1",
		preconditions:  []string{},
		postconditions: []string{"reasoning.intents"},
		executeFn: func(ctx context.Context, agentContext *models.AgentContext) (*models.AgentContext, error) {
			agentContext.Reasoning.Intents = []models.Intent{{Type: "conditional_trigger", Confidence: 0.9}}
			return agentContext, nil
		},
	}

	agent2 := &MockAgent{
		id:             "agent2",
		preconditions:  []string{"reasoning.intents"}, // Only runs if intents exist
		postconditions: []string{"reasoning.conclusions"},
		executeFn: func(ctx context.Context, agentContext *models.AgentContext) (*models.AgentContext, error) {
			if len(agentContext.Reasoning.Intents) > 0 {
				agentContext.Reasoning.Conclusions = []models.Conclusion{
					{ID: "c1", Description: "Conditional execution", Confidence: 0.9},
				}
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
			TrackPerformance: false,
		},
	}

	manager := NewReasoningManager(config)
	require.NoError(t, manager.RegisterAgent(agent1))
	require.NoError(t, manager.RegisterAgent(agent2))

	ctx := context.Background()

	// Run 3 times
	var results []*models.AgentContext
	for i := 0; i < 3; i++ {
		agentContext := models.NewAgentContext("session", "trace")
		result, err := manager.Execute(ctx, agentContext)
		require.NoError(t, err)
		results = append(results, result)
	}

	// All runs should execute agent2 conditionally in same way
	for i, result := range results {
		assert.Len(t, result.Reasoning.Intents, 1, "Run %d: should have intents", i)
		assert.Len(t, result.Reasoning.Conclusions, 1, "Run %d: should have conditional conclusions", i)
		assert.Equal(t, "Conditional execution", result.Reasoning.Conclusions[0].Description, "Run %d: conclusion mismatch", i)
	}
}
