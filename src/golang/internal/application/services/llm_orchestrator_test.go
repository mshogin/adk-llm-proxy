package services

import (
	"context"
	"testing"

	"github.com/mshogin/agents/internal/domain/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewLLMOrchestrator tests orchestrator creation
func TestNewLLMOrchestrator(t *testing.T) {
	orchestrator := NewLLMOrchestrator()

	assert.NotNil(t, orchestrator)
	assert.NotNil(t, orchestrator.profiles)
	assert.NotNil(t, orchestrator.strategies)
	assert.NotNil(t, orchestrator.cache)
	assert.Equal(t, 0.0, orchestrator.sessionBudgetUsed)
	assert.NotEmpty(t, orchestrator.profiles, "Should have default profiles loaded")
	assert.NotEmpty(t, orchestrator.strategies, "Should have default strategies loaded")
}

// TestSelectModel_IntentClassification tests model selection for simple task
func TestSelectModel_IntentClassification(t *testing.T) {
	orchestrator := NewLLMOrchestrator()
	ctx := context.Background()

	req := &LLMRequest{
		Prompt:      "What is the user asking for?",
		TaskType:    models.TaskTypeIntentClassification,
		AgentID:     "intent_agent",
		MaxTokens:   100,
		Temperature: 0.0,
		ContextSize: 400,
		UseCach:     false,
	}

	model, provider, err := orchestrator.SelectModel(ctx, req)

	require.NoError(t, err)
	assert.NotEmpty(t, model)
	assert.NotEmpty(t, provider)
	// Should select cheapest model for simple task
	assert.Contains(t, []string{"deepseek-chat", "gpt-4o-mini", "mistral"}, model)
}

// TestSelectModel_AdvancedReasoning tests model selection for complex task
func TestSelectModel_AdvancedReasoning(t *testing.T) {
	orchestrator := NewLLMOrchestrator()
	ctx := context.Background()

	req := &LLMRequest{
		Prompt:      "Perform deep multi-hop reasoning across these facts...",
		TaskType:    models.TaskTypeDeepReasoning,
		AgentID:     "inference_agent",
		MaxTokens:   2000,
		Temperature: 0.7,
		ContextSize: 50000,
		UseCach:     false,
	}

	model, provider, err := orchestrator.SelectModel(ctx, req)

	require.NoError(t, err)
	assert.NotEmpty(t, model)
	assert.NotEmpty(t, provider)
	// Should select advanced model for complex reasoning
	assert.Contains(t, []string{"o1-mini", "claude-3-5-sonnet", "gpt-4o"}, model)
}

// TestSelectModel_ContextSizeLimit tests that models are filtered by context size
func TestSelectModel_ContextSizeLimit(t *testing.T) {
	orchestrator := NewLLMOrchestrator()
	ctx := context.Background()

	// Request with very large context that exceeds many models
	req := &LLMRequest{
		Prompt:      "Analyze this large document...",
		TaskType:    models.TaskTypeLongContextAnalysis,
		AgentID:     "analysis_agent",
		MaxTokens:   1000,
		Temperature: 0.5,
		ContextSize: 150000, // Exceeds most models
		UseCach:     false,
	}

	model, provider, err := orchestrator.SelectModel(ctx, req)

	// Should either find a model with large context or fail
	if err == nil {
		// If successful, verify it's a model with large context window
		modelKey := provider + "/" + model
		profile := orchestrator.profiles[modelKey]
		assert.NotNil(t, profile)
		assert.GreaterOrEqual(t, profile.ContextLimit, req.ContextSize)
	} else {
		// Expected failure when no model can handle the context size
		assert.Contains(t, err.Error(), "no suitable model")
	}
}

// TestSelectModel_Fallback tests fallback chain
func TestSelectModel_Fallback(t *testing.T) {
	orchestrator := NewLLMOrchestrator()

	// Remove default model to force fallback
	delete(orchestrator.profiles, "deepseek/deepseek-chat")

	ctx := context.Background()
	req := &LLMRequest{
		Prompt:      "Simple task",
		TaskType:    models.TaskTypeIntentClassification,
		AgentID:     "test_agent",
		MaxTokens:   50,
		Temperature: 0.0,
		ContextSize: 200,
		UseCach:     false,
	}

	model, _, err := orchestrator.SelectModel(ctx, req)

	require.NoError(t, err)
	assert.NotEmpty(t, model)
	// Should fall back to gpt-4o-mini or mistral
	assert.Contains(t, []string{"gpt-4o-mini", "mistral"}, model)
}

// TestCalculateCost tests cost calculation
func TestCalculateCost(t *testing.T) {
	orchestrator := NewLLMOrchestrator()

	testCases := []struct {
		name     string
		provider string
		model    string
		tokens   int
		expected float64
	}{
		{
			name:     "DeepSeek cheap model",
			provider: "deepseek",
			model:    "deepseek-chat",
			tokens:   1000,
			expected: 0.0001, // $0.0001 per 1K tokens
		},
		{
			name:     "OpenAI mini model",
			provider: "openai",
			model:    "gpt-4o-mini",
			tokens:   1000,
			expected: 0.00015,
		},
		{
			name:     "OpenAI o1 expensive model",
			provider: "openai",
			model:    "o1",
			tokens:   1000,
			expected: 0.06,
		},
		{
			name:     "Half tokens",
			provider: "openai",
			model:    "gpt-4o-mini",
			tokens:   500,
			expected: 0.000075, // Half of 0.00015
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cost := orchestrator.CalculateCost(tc.provider, tc.model, tc.tokens)
			assert.InDelta(t, tc.expected, cost, 0.0000001)
		})
	}
}

// TestTrackUsage tests usage tracking
func TestTrackUsage(t *testing.T) {
	orchestrator := NewLLMOrchestrator()

	cost1 := orchestrator.TrackUsage("agent1", "openai", "gpt-4o-mini", 1000)
	assert.Greater(t, cost1, 0.0)
	assert.Equal(t, cost1, orchestrator.sessionBudgetUsed)
	assert.Equal(t, cost1, orchestrator.agentBudgetUsed["agent1"])

	cost2 := orchestrator.TrackUsage("agent1", "openai", "gpt-4o", 2000)
	assert.Greater(t, cost2, cost1)
	assert.Equal(t, cost1+cost2, orchestrator.sessionBudgetUsed)
	assert.Equal(t, cost1+cost2, orchestrator.agentBudgetUsed["agent1"])

	cost3 := orchestrator.TrackUsage("agent2", "deepseek", "deepseek-chat", 500)
	assert.Greater(t, cost3, 0.0)
	assert.Equal(t, cost1+cost2+cost3, orchestrator.sessionBudgetUsed)
	assert.Equal(t, cost3, orchestrator.agentBudgetUsed["agent2"])
}

// TestBudgetConstraints tests budget enforcement
func TestBudgetConstraints(t *testing.T) {
	budget := models.BudgetConstraints{
		SessionBudgetUSD:            0.01,
		AgentBudgetUSD:              0.005,
		WarningThreshold:            0.8,
		EmergencyDegradationEnabled: true,
		CriticalAgents:              []string{"critical_agent"},
	}

	orchestrator := NewLLMOrchestratorWithConfig(budget, models.DefaultCacheConfig())
	ctx := context.Background()

	// Use up most of the budget
	orchestrator.TrackUsage("agent1", "openai", "gpt-4o", 4000) // ~$0.01

	req := &LLMRequest{
		Prompt:      "Test",
		TaskType:    models.TaskTypeIntentClassification,
		AgentID:     "agent2",
		ContextSize: 100,
	}

	// Should fail due to budget exceeded
	_, _, err := orchestrator.SelectModel(ctx, req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "budget exceeded")
}

// TestBudgetConstraints_CriticalAgent tests critical agent exception
func TestBudgetConstraints_CriticalAgent(t *testing.T) {
	budget := models.BudgetConstraints{
		SessionBudgetUSD:            0.01,
		AgentBudgetUSD:              0.005,
		WarningThreshold:            0.8,
		EmergencyDegradationEnabled: true,
		CriticalAgents:              []string{"critical_agent"},
	}

	orchestrator := NewLLMOrchestratorWithConfig(budget, models.DefaultCacheConfig())
	ctx := context.Background()

	// Use up all of the budget
	orchestrator.TrackUsage("agent1", "openai", "gpt-4o", 5000) // ~$0.0125 (exceeds budget)

	req := &LLMRequest{
		Prompt:      "Critical task",
		TaskType:    models.TaskTypeIntentClassification,
		AgentID:     "critical_agent",
		ContextSize: 100,
	}

	// Should succeed for critical agent despite budget exceeded
	model, provider, err := orchestrator.SelectModel(ctx, req)
	require.NoError(t, err)
	assert.NotEmpty(t, model)
	assert.NotEmpty(t, provider)
}

// TestCacheKey tests cache key generation
func TestCacheKey(t *testing.T) {
	orchestrator := NewLLMOrchestrator()

	req1 := &LLMRequest{
		Prompt:      "Hello world",
		TaskType:    models.TaskTypeIntentClassification,
		MaxTokens:   100,
		Temperature: 0.0,
	}

	req2 := &LLMRequest{
		Prompt:      "Hello world",
		TaskType:    models.TaskTypeIntentClassification,
		MaxTokens:   100,
		Temperature: 0.0,
	}

	req3 := &LLMRequest{
		Prompt:      "Hello world",
		TaskType:    models.TaskTypeIntentClassification,
		MaxTokens:   200, // Different max tokens
		Temperature: 0.0,
	}

	key1 := orchestrator.GetCacheKey(req1, "gpt-4o-mini")
	key2 := orchestrator.GetCacheKey(req2, "gpt-4o-mini")
	key3 := orchestrator.GetCacheKey(req3, "gpt-4o-mini")

	// Same requests should have same key
	assert.Equal(t, key1, key2)

	// Different parameters should have different key
	assert.NotEqual(t, key1, key3)
}

// TestCache tests caching functionality
func TestCache(t *testing.T) {
	cacheConfig := models.CacheConfig{
		Enabled:           true,
		ClassificationTTL: 60,
		SynthesisTTL:      30,
		InferenceTTL:      15,
		MaxSizeMB:         100,
		TargetHitRate:     0.4,
	}

	orchestrator := NewLLMOrchestratorWithConfig(
		models.DefaultBudgetConstraints(),
		cacheConfig,
	)

	cacheKey := "test_key"
	response := "This is a cached response"
	tokens := 50
	cost := 0.001

	// Save to cache
	orchestrator.SaveToCache(cacheKey, response, tokens, cost, models.TaskTypeIntentClassification)

	// Retrieve from cache
	cached, found := orchestrator.GetFromCache(cacheKey)
	require.True(t, found)
	assert.Equal(t, response, cached.Response)
	assert.Equal(t, tokens, cached.Tokens)
	assert.Equal(t, cost, cached.Cost)
	assert.Equal(t, 1, cached.HitCount)

	// Retrieve again to increment hit count
	cached2, found2 := orchestrator.GetFromCache(cacheKey)
	require.True(t, found2)
	assert.Equal(t, 2, cached2.HitCount)
}

// TestCache_NotFound tests cache miss
func TestCache_NotFound(t *testing.T) {
	orchestrator := NewLLMOrchestrator()

	cached, found := orchestrator.GetFromCache("nonexistent_key")
	assert.False(t, found)
	assert.Nil(t, cached)
}

// TestGetCacheStats tests cache statistics
func TestGetCacheStats(t *testing.T) {
	orchestrator := NewLLMOrchestrator()

	// Add some cache entries
	orchestrator.SaveToCache("key1", "response1", 100, 0.001, models.TaskTypeIntentClassification)
	orchestrator.SaveToCache("key2", "response2", 200, 0.002, models.TaskTypeTextSynthesis)

	// Get from cache to increment hit count
	orchestrator.GetFromCache("key1")
	orchestrator.GetFromCache("key1")
	orchestrator.GetFromCache("key2")

	entries, hits, size := orchestrator.GetCacheStats()

	assert.Equal(t, 2, entries)
	assert.Equal(t, 3, hits) // 2 hits for key1, 1 hit for key2
	assert.Greater(t, size, 0)
}

// TestDecisionLogging tests LLM decision logging
func TestDecisionLogging(t *testing.T) {
	orchestrator := NewLLMOrchestrator()
	ctx := context.Background()

	req := &LLMRequest{
		Prompt:      "Test prompt",
		TaskType:    models.TaskTypeIntentClassification,
		AgentID:     "test_agent",
		ContextSize: 100,
	}

	// Select a model (should log decision)
	model, _, err := orchestrator.SelectModel(ctx, req)
	require.NoError(t, err)

	decisions := orchestrator.GetDecisions()
	require.Len(t, decisions, 1)

	decision := decisions[0]
	assert.Equal(t, "test_agent", decision.AgentID)
	assert.Equal(t, string(models.TaskTypeIntentClassification), decision.TaskType)
	assert.Equal(t, model, decision.Selected)
	assert.NotEmpty(t, decision.Reason)
	assert.False(t, decision.Timestamp.IsZero())
}

// TestGetBudgetStatus tests budget status retrieval
func TestGetBudgetStatus(t *testing.T) {
	orchestrator := NewLLMOrchestrator()

	// Track some usage
	orchestrator.TrackUsage("agent1", "openai", "gpt-4o-mini", 1000)
	orchestrator.TrackUsage("agent2", "deepseek", "deepseek-chat", 500)

	sessionUsed, sessionLimit, agentBudgets := orchestrator.GetBudgetStatus()
	_ = sessionUsed
	_ = sessionLimit

	assert.Greater(t, sessionUsed, 0.0)
	assert.Greater(t, sessionLimit, 0.0)
	assert.Len(t, agentBudgets, 2)
	assert.Greater(t, agentBudgets["agent1"], 0.0)
	assert.Greater(t, agentBudgets["agent2"], 0.0)
}

// TestResetSessionBudget tests budget reset
func TestResetSessionBudget(t *testing.T) {
	orchestrator := NewLLMOrchestrator()
	ctx := context.Background()

	// Track some usage and make a decision
	orchestrator.TrackUsage("agent1", "openai", "gpt-4o-mini", 1000)

	// Make a model selection to log a decision
	req := &LLMRequest{
		Prompt:      "Test",
		TaskType:    models.TaskTypeIntentClassification,
		AgentID:     "agent1",
		ContextSize: 100,
	}
	_, _, _ = orchestrator.SelectModel(ctx, req)

	assert.Greater(t, orchestrator.sessionBudgetUsed, 0.0)
	assert.Len(t, orchestrator.agentBudgetUsed, 1)
	assert.NotEmpty(t, orchestrator.decisions)

	// Reset
	orchestrator.ResetSessionBudget()

	assert.Equal(t, 0.0, orchestrator.sessionBudgetUsed)
	assert.Empty(t, orchestrator.agentBudgetUsed)
	assert.Empty(t, orchestrator.decisions)
}

// TestAddModelProfile tests custom model profile addition
func TestAddModelProfile(t *testing.T) {
	orchestrator := NewLLMOrchestrator()

	customProfile := models.ModelProfile{
		Provider:        "custom",
		Model:           "custom-model",
		Quality:         "premium",
		Speed:           "fast",
		CostPer1KTokens: 0.005,
		ContextLimit:    50000,
	}

	orchestrator.AddModelProfile(customProfile)

	profile, exists := orchestrator.profiles["custom/custom-model"]
	require.True(t, exists)
	assert.Equal(t, "custom", profile.Provider)
	assert.Equal(t, "custom-model", profile.Model)
	assert.Equal(t, 0.005, profile.CostPer1KTokens)
}

// TestAddSelectionStrategy tests custom strategy addition
func TestAddSelectionStrategy(t *testing.T) {
	orchestrator := NewLLMOrchestrator()

	customStrategy := models.ModelSelectionStrategy{
		TaskType:       models.TaskType("custom_task"),
		Complexity:     models.TaskComplexityMedium,
		DefaultModel:   "openai/gpt-4o",
		Fallback1:      "anthropic/claude-3-5-sonnet",
		Fallback2:      "",
		MaxContextSize: 10000,
	}

	orchestrator.AddSelectionStrategy(customStrategy)

	strategy, exists := orchestrator.strategies[models.TaskType("custom_task")]
	require.True(t, exists)
	assert.Equal(t, "openai/gpt-4o", strategy.DefaultModel)
	assert.Equal(t, models.TaskComplexityMedium, strategy.Complexity)
}

// TestSerialize tests state serialization
func TestSerialize(t *testing.T) {
	orchestrator := NewLLMOrchestrator()

	// Add some state
	orchestrator.TrackUsage("agent1", "openai", "gpt-4o-mini", 1000)
	orchestrator.SaveToCache("key1", "response", 100, 0.001, models.TaskTypeIntentClassification)

	data, err := orchestrator.Serialize()
	require.NoError(t, err)
	assert.NotEmpty(t, data)

	// Verify it's valid JSON
	assert.Contains(t, string(data), "session_budget_used")
	assert.Contains(t, string(data), "agent_budgets")
	assert.Contains(t, string(data), "cache_entries")
}

// TestConcurrentAccess tests thread-safety
func TestConcurrentAccess(t *testing.T) {
	orchestrator := NewLLMOrchestrator()
	ctx := context.Background()

	// Run concurrent operations
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(id int) {
			defer func() { done <- true }()

			req := &LLMRequest{
				Prompt:      "Test",
				TaskType:    models.TaskTypeIntentClassification,
				AgentID:     "agent",
				ContextSize: 100,
			}

			_, _, err := orchestrator.SelectModel(ctx, req)
			assert.NoError(t, err)

			orchestrator.TrackUsage("agent", "openai", "gpt-4o-mini", 100)

			_, _, _ = orchestrator.GetBudgetStatus()
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify state is consistent
	assert.Greater(t, orchestrator.sessionBudgetUsed, 0.0)
}
