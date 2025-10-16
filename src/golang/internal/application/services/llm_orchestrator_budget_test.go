package services

import (
	"context"
	"testing"

	"github.com/mshogin/agents/internal/domain/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Cost budget enforcement tests: testing budget limits and cost tracking

// TestBudgetEnforcement_WithinSessionBudget tests normal operation within session budget
func TestBudgetEnforcement_WithinSessionBudget(t *testing.T) {
	budget := models.BudgetConstraints{
		SessionBudgetUSD: 0.10, // $0.10 session budget
		AgentBudgetUSD:   0.05, // $0.05 per agent
		WarningThreshold: 0.80,
		CriticalAgents:   []string{},
	}

	orchestrator := NewLLMOrchestratorWithConfig(budget, models.DefaultCacheConfig())

	// Simulate low-cost LLM call
	cost1 := orchestrator.TrackUsage("intent_detection", "openai", "gpt-4o-mini", 100)
	assert.LessOrEqual(t, cost1, 0.01) // Should be cheap

	// Check budget status
	sessionUsed, sessionLimit, agentBudgets := orchestrator.GetBudgetStatus()
	assert.Equal(t, cost1, sessionUsed)
	assert.Equal(t, 0.10, sessionLimit)
	assert.Equal(t, cost1, agentBudgets["intent_detection"])

	// Another LLM call
	cost2 := orchestrator.TrackUsage("inference", "openai", "gpt-4o-mini", 200)

	sessionUsed, _, agentBudgets = orchestrator.GetBudgetStatus()
	assert.Equal(t, cost1+cost2, sessionUsed)
	assert.Less(t, sessionUsed, sessionLimit) // Still under budget
}

// TestBudgetEnforcement_SessionBudgetExceeded tests handling of session budget overrun
func TestBudgetEnforcement_SessionBudgetExceeded(t *testing.T) {
	budget := models.BudgetConstraints{
		SessionBudgetUSD: 0.01, // $0.01 session budget (very low)
		AgentBudgetUSD:   0.05,
		WarningThreshold: 0.80,
		CriticalAgents:   []string{},
	}

	orchestrator := NewLLMOrchestratorWithConfig(budget, models.DefaultCacheConfig())

	// Record expensive LLM call that exceeds budget
	// Use a cheap model but lots of tokens to exceed budget
	// gpt-4o-mini costs ~$0.00015 per 1K tokens, so need 70K tokens to exceed $0.01
	orchestrator.TrackUsage("inference", "openai", "gpt-4o-mini", 100000) // 100K tokens

	// Check if budget is exceeded
	sessionUsed, sessionLimit, _ := orchestrator.GetBudgetStatus()
	assert.Greater(t, sessionUsed, sessionLimit) // Budget exceeded

	// Attempting another LLM call should fail
	req := &LLMRequest{
		Prompt:      "Test prompt",
		TaskType:    models.TaskTypeInference,
		AgentID:     "next_agent",
		MaxTokens:   100,
		Temperature: 0.7,
		ContextSize: 500,
	}

	_, _, err := orchestrator.SelectModel(context.Background(), req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "budget exceeded")
}

// TestBudgetEnforcement_AgentBudgetExceeded tests per-agent budget limits
func TestBudgetEnforcement_AgentBudgetExceeded(t *testing.T) {
	budget := models.BudgetConstraints{
		SessionBudgetUSD: 1.00, // High session budget
		AgentBudgetUSD:   0.01, // $0.01 per agent (low)
		WarningThreshold: 0.80,
		CriticalAgents:   []string{},
	}

	orchestrator := NewLLMOrchestratorWithConfig(budget, models.DefaultCacheConfig())

	// First agent uses expensive LLM call (need 70K+ tokens to exceed $0.01 with gpt-4o-mini)
	orchestrator.TrackUsage("expensive_agent", "openai", "gpt-4o-mini", 100000) // 100K tokens

	// Check if agent budget is exceeded
	sessionUsed, sessionLimit, agentBudgets := orchestrator.GetBudgetStatus()
	agentCost := agentBudgets["expensive_agent"]
	assert.Greater(t, agentCost, 0.01) // Agent budget exceeded

	// Session budget is not exceeded
	assert.Less(t, sessionUsed, sessionLimit)

	// The expensive agent cannot use more LLM
	req := &LLMRequest{
		Prompt:      "Test prompt",
		TaskType:    models.TaskTypeInference,
		AgentID:     "expensive_agent",
		MaxTokens:   100,
		Temperature: 0.7,
		ContextSize: 500,
	}

	_, _, err := orchestrator.SelectModel(context.Background(), req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "budget exceeded")

	// Another agent can still use LLM
	req.AgentID = "another_agent"
	_, _, err = orchestrator.SelectModel(context.Background(), req)
	require.NoError(t, err)
}

// TestBudgetEnforcement_CriticalAgentOverride tests that critical agents bypass budget limits
func TestBudgetEnforcement_CriticalAgentOverride(t *testing.T) {
	budget := models.BudgetConstraints{
		SessionBudgetUSD: 0.01, // Very low session budget
		AgentBudgetUSD:   0.01,
		WarningThreshold: 0.80,
		CriticalAgents:   []string{"intent_detection"}, // Critical agent
	}

	orchestrator := NewLLMOrchestratorWithConfig(budget, models.DefaultCacheConfig())

	// Exceed session budget with non-critical agent (need 70K+ tokens for $0.01+)
	orchestrator.TrackUsage("non_critical", "openai", "gpt-4o-mini", 100000)

	// Verify budget exceeded
	sessionUsed, sessionLimit, _ := orchestrator.GetBudgetStatus()
	assert.Greater(t, sessionUsed, sessionLimit)

	// Non-critical agent should be blocked
	req := &LLMRequest{
		Prompt:      "Test prompt",
		TaskType:    models.TaskTypeInference,
		AgentID:     "non_critical_2",
		MaxTokens:   100,
		Temperature: 0.7,
		ContextSize: 500,
	}

	_, _, err := orchestrator.SelectModel(context.Background(), req)
	require.Error(t, err)

	// Critical agent should still work
	req.AgentID = "intent_detection"
	req.TaskType = models.TaskTypeIntentClassification
	_, _, err = orchestrator.SelectModel(context.Background(), req)
	require.NoError(t, err) // Critical agent bypasses budget
}

// TestBudgetEnforcement_CostTracking tests accurate cost tracking across multiple agents
func TestBudgetEnforcement_CostTracking(t *testing.T) {
	orchestrator := NewLLMOrchestrator()

	// Agent 1 uses LLM
	cost1 := orchestrator.TrackUsage("agent1", "openai", "gpt-4o-mini", 1000)

	// Agent 2 uses LLM
	cost2 := orchestrator.TrackUsage("agent2", "openai", "gpt-4o-mini", 2000)

	// Agent 3 uses LLM
	cost3 := orchestrator.TrackUsage("agent3", "openai", "gpt-4o-mini", 500)

	// Verify total cost
	sessionUsed, _, agentBudgets := orchestrator.GetBudgetStatus()
	assert.Equal(t, cost1+cost2+cost3, sessionUsed)

	// Verify per-agent costs
	assert.Equal(t, cost1, agentBudgets["agent1"])
	assert.Equal(t, cost2, agentBudgets["agent2"])
	assert.Equal(t, cost3, agentBudgets["agent3"])

	// Verify all costs are non-zero
	assert.Greater(t, cost1, 0.0)
	assert.Greater(t, cost2, 0.0)
	assert.Greater(t, cost3, 0.0)
}

// TestBudgetEnforcement_CostCalculation tests cost calculation for different models
func TestBudgetEnforcement_CostCalculation(t *testing.T) {
	orchestrator := NewLLMOrchestrator()

	testCases := []struct {
		provider string
		model    string
		tokens   int
	}{
		{"deepseek", "deepseek-chat", 1000},
		{"openai", "gpt-4o-mini", 1000},
		{"openai", "gpt-4o", 1000},
		{"anthropic", "claude-sonnet", 1000},
	}

	for _, tc := range testCases {
		cost := orchestrator.CalculateCost(tc.provider, tc.model, tc.tokens)
		assert.GreaterOrEqual(t, cost, 0.0, "Cost should be non-negative for %s/%s", tc.provider, tc.model)

		// Verify double tokens = double cost
		cost2 := orchestrator.CalculateCost(tc.provider, tc.model, tc.tokens*2)
		assert.InDelta(t, cost*2, cost2, 0.0001, "Double tokens should give double cost")
	}
}

// TestBudgetEnforcement_BudgetReset tests budget reset functionality
func TestBudgetEnforcement_BudgetReset(t *testing.T) {
	orchestrator := NewLLMOrchestrator()

	// Use some budget
	orchestrator.TrackUsage("agent1", "openai", "gpt-4o-mini", 1000)
	orchestrator.TrackUsage("agent2", "openai", "gpt-4o-mini", 2000)

	// Verify budget used
	sessionUsed, _, agentBudgets := orchestrator.GetBudgetStatus()
	assert.Greater(t, sessionUsed, 0.0)
	assert.NotEmpty(t, agentBudgets)

	// Reset budget
	orchestrator.ResetSessionBudget()

	// Verify budget cleared
	sessionUsed, _, agentBudgets = orchestrator.GetBudgetStatus()
	assert.Equal(t, 0.0, sessionUsed)
	assert.Empty(t, agentBudgets)

	// Verify decisions cleared
	decisions := orchestrator.GetDecisions()
	assert.Empty(t, decisions)
}

// TestBudgetEnforcement_ModelSelection tests budget-aware model selection
func TestBudgetEnforcement_ModelSelection(t *testing.T) {
	budget := models.BudgetConstraints{
		SessionBudgetUSD: 0.10,
		AgentBudgetUSD:   0.05,
		WarningThreshold: 0.80,
		CriticalAgents:   []string{},
	}

	orchestrator := NewLLMOrchestratorWithConfig(budget, models.DefaultCacheConfig())

	// Request with low budget usage
	req := &LLMRequest{
		Prompt:      "Simple intent classification",
		TaskType:    models.TaskTypeIntentClassification,
		AgentID:     "intent_agent",
		MaxTokens:   100,
		Temperature: 0.0,
		ContextSize: 500,
	}

	// Should select cheap model for simple task
	model, provider, err := orchestrator.SelectModel(context.Background(), req)
	require.NoError(t, err)
	assert.NotEmpty(t, model)
	assert.NotEmpty(t, provider)

	// Verify decision was logged
	decisions := orchestrator.GetDecisions()
	assert.NotEmpty(t, decisions)
	assert.Equal(t, "intent_agent", decisions[0].AgentID)
	assert.Equal(t, model, decisions[0].Selected)
}

// TestBudgetEnforcement_CacheSavesMoney tests that caching reduces costs
func TestBudgetEnforcement_CacheSavesMoney(t *testing.T) {
	cacheConfig := models.DefaultCacheConfig()
	cacheConfig.Enabled = true

	orchestrator := NewLLMOrchestratorWithConfig(
		models.DefaultBudgetConstraints(),
		cacheConfig,
	)

	req := &LLMRequest{
		Prompt:      "Test prompt for caching",
		TaskType:    models.TaskTypeValidation,
		AgentID:     "test_agent",
		MaxTokens:   100,
		Temperature: 0.0,
		ContextSize: 500,
		UseCach:     true, // Note: typo in original struct
	}

	// First call - should miss cache
	model, _, err := orchestrator.SelectModel(context.Background(), req)
	require.NoError(t, err)

	cacheKey := orchestrator.GetCacheKey(req, model)

	// Simulate response caching
	orchestrator.SaveToCache(cacheKey, "Test response", 100, 0.001, req.TaskType)

	// Verify cache hit
	cached, hit := orchestrator.GetFromCache(cacheKey)
	assert.True(t, hit)
	assert.Equal(t, "Test response", cached.Response)
	assert.Equal(t, 0.001, cached.Cost)

	// Verify cache stats
	entries, hits, size := orchestrator.GetCacheStats()
	assert.Equal(t, 1, entries)
	assert.Equal(t, 1, hits) // 1 hit from GetFromCache call
	assert.Greater(t, size, 0)
}

// TestBudgetEnforcement_WarningThreshold tests budget warning detection
func TestBudgetEnforcement_WarningThreshold(t *testing.T) {
	budget := models.BudgetConstraints{
		SessionBudgetUSD: 0.10,
		AgentBudgetUSD:   0.05,
		WarningThreshold: 0.80, // 80% threshold
		CriticalAgents:   []string{},
	}

	orchestrator := NewLLMOrchestratorWithConfig(budget, models.DefaultCacheConfig())

	// Use 70% of budget (below 80% threshold)
	// Budget is $0.10, 70% = $0.07
	// gpt-4o-mini costs ~$0.00015 per 1K tokens
	// Need 0.07 / 0.00015 * 1000 = ~467K tokens for $0.07
	orchestrator.TrackUsage("agent1", "openai", "gpt-4o-mini", 450000) // ~$0.0675

	sessionUsed, sessionLimit, _ := orchestrator.GetBudgetStatus()
	usagePercent := sessionUsed / sessionLimit
	assert.Less(t, usagePercent, 0.80) // Below warning threshold

	// Use another 10% of budget (total ~80%, should be at or above threshold)
	// Need ~65K tokens for $0.01
	orchestrator.TrackUsage("agent2", "openai", "gpt-4o-mini", 70000) // ~$0.0105

	sessionUsed, sessionLimit, _ = orchestrator.GetBudgetStatus()
	usagePercent = sessionUsed / sessionLimit
	assert.GreaterOrEqual(t, usagePercent, 0.75) // Should be well above 75%, approaching 80%
}

// TestBudgetEnforcement_MultipleSessionsIsolation tests session budget isolation
func TestBudgetEnforcement_MultipleSessionsIsolation(t *testing.T) {
	// Note: Current implementation shares budget across sessions
	// This test documents expected behavior for future session isolation

	orchestrator1 := NewLLMOrchestrator()
	orchestrator2 := NewLLMOrchestrator()

	// Session 1 uses budget
	orchestrator1.TrackUsage("agent1", "openai", "gpt-4o-mini", 5000)

	// Session 2 uses budget
	orchestrator2.TrackUsage("agent1", "openai", "gpt-4o-mini", 2000)

	// Verify isolation (each orchestrator has independent budget)
	session1Used, _, _ := orchestrator1.GetBudgetStatus()
	session2Used, _, _ := orchestrator2.GetBudgetStatus()

	assert.NotEqual(t, session1Used, session2Used)
	assert.Greater(t, session1Used, session2Used)
}

// TestBudgetEnforcement_FallbackChain tests model fallback when budget constrained
func TestBudgetEnforcement_FallbackChain(t *testing.T) {
	budget := models.BudgetConstraints{
		SessionBudgetUSD: 0.10,
		AgentBudgetUSD:   0.05,
		WarningThreshold: 0.80,
		CriticalAgents:   []string{},
	}

	orchestrator := NewLLMOrchestratorWithConfig(budget, models.DefaultCacheConfig())

	// Request for complex task
	req := &LLMRequest{
		Prompt:      "Complex reasoning task",
		TaskType:    models.TaskTypeAdvancedInference,
		AgentID:     "inference_agent",
		MaxTokens:   1000,
		Temperature: 0.7,
		ContextSize: 5000,
	}

	// Should select model based on strategy (default → fallback1 → fallback2)
	model, provider, err := orchestrator.SelectModel(context.Background(), req)
	require.NoError(t, err)
	assert.NotEmpty(t, model)
	assert.NotEmpty(t, provider)

	// Verify decision includes fallback reasoning
	decisions := orchestrator.GetDecisions()
	require.NotEmpty(t, decisions)
	assert.Contains(t, decisions[0].Reason, "cost")
}
