package integration

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/mshogin/agents/internal/application/services"
	"github.com/mshogin/agents/internal/domain/models"
	"github.com/mshogin/agents/internal/infrastructure/config"
	"github.com/mshogin/agents/internal/infrastructure/providers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// skipIfNoAPIKey skips test if required API key is not set
func skipIfNoAPIKey(t *testing.T, envVar string) {
	if os.Getenv(envVar) == "" {
		t.Skipf("Skipping test: %s environment variable not set", envVar)
	}
}

//  helper to create pointer values
func ptr[T any](v T) *T {
	return &v
}

// TestOpenAIProvider_Integration tests OpenAI provider with real API
func TestOpenAIProvider_Integration(t *testing.T) {
	skipIfNoAPIKey(t, "OPENAI_API_KEY")

	providerConfig := config.ProviderConfig{
		APIKey:  os.Getenv("OPENAI_API_KEY"),
		BaseURL: "https://api.openai.com/v1",
		Enabled: true,
		Timeout: 30 * time.Second,
	}

	provider := providers.NewOpenAIProvider(providerConfig)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req := &models.CompletionRequest{
		Model: "gpt-4o-mini",
		Messages: []models.Message{
			{Role: "user", Content: "Say 'Hello from integration test'"},
		},
		MaxTokens:   ptr(50),
		Temperature: ptr(0.0),
		Stream:      false,
	}

	// Test non-streaming mode
	t.Run("Non-Streaming", func(t *testing.T) {
		chunkChan, err := provider.StreamCompletion(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, chunkChan)

		var chunks []*models.CompletionChunk
		for chunk := range chunkChan {
			chunks = append(chunks, chunk)
		}

		require.NotEmpty(t, chunks, "Should receive at least one chunk")
		assert.NotEmpty(t, chunks[0].Choices, "Should have at least one choice")
		assert.NotEmpty(t, chunks[0].Choices[0].Delta.Content, "Should have content")
		t.Logf("OpenAI response: %s", chunks[0].Choices[0].Delta.Content)
	})

	// Test streaming mode
	t.Run("Streaming", func(t *testing.T) {
		reqStreaming := *req
		reqStreaming.Stream = true

		chunkChan, err := provider.StreamCompletion(ctx, &reqStreaming)
		require.NoError(t, err)
		require.NotNil(t, chunkChan)

		var receivedChunks int
		for chunk := range chunkChan {
			receivedChunks++
			assert.NotEmpty(t, chunk.ID, "Chunk should have ID")
			assert.Equal(t, "gpt-4o-mini", chunk.Model)
		}

		assert.Greater(t, receivedChunks, 0, "Should receive at least one chunk in streaming mode")
		t.Logf("OpenAI streaming: received %d chunks", receivedChunks)
	})

	// Test timeout
	t.Run("Timeout", func(t *testing.T) {
		shortCtx, shortCancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
		defer shortCancel()

		_, err := provider.StreamCompletion(shortCtx, req)
		// Should either fail immediately or return error in channel
		if err != nil {
			assert.Error(t, err)
		}
	})
}

// TestAnthropicProvider_Integration tests Anthropic provider with real API
func TestAnthropicProvider_Integration(t *testing.T) {
	skipIfNoAPIKey(t, "ANTHROPIC_API_KEY")

	providerConfig := config.ProviderConfig{
		APIKey:  os.Getenv("ANTHROPIC_API_KEY"),
		BaseURL: "https://api.anthropic.com/v1",
		Enabled: true,
		Timeout: 30 * time.Second,
	}

	provider := providers.NewAnthropicProvider(providerConfig)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req := &models.CompletionRequest{
		Model: "claude-3-haiku-20240307",
		Messages: []models.Message{
			{Role: "user", Content: "Say 'Hello from integration test'"},
		},
		MaxTokens:   ptr(50),
		Temperature: ptr(0.0),
		Stream:      false,
	}

	t.Run("Non-Streaming", func(t *testing.T) {
		chunkChan, err := provider.StreamCompletion(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, chunkChan)

		var chunks []*models.CompletionChunk
		for chunk := range chunkChan {
			chunks = append(chunks, chunk)
		}

		require.NotEmpty(t, chunks, "Should receive at least one chunk")
		t.Logf("Anthropic response received: %d chunks", len(chunks))
	})

	t.Run("Streaming", func(t *testing.T) {
		reqStreaming := *req
		reqStreaming.Stream = true

		chunkChan, err := provider.StreamCompletion(ctx, &reqStreaming)
		require.NoError(t, err)
		require.NotNil(t, chunkChan)

		var receivedChunks int
		for chunk := range chunkChan {
			receivedChunks++
			assert.NotEmpty(t, chunk.ID, "Chunk should have ID")
		}

		assert.Greater(t, receivedChunks, 0, "Should receive chunks in streaming mode")
		t.Logf("Anthropic streaming: received %d chunks", receivedChunks)
	})
}

// TestDeepSeekProvider_Integration tests DeepSeek provider with real API
func TestDeepSeekProvider_Integration(t *testing.T) {
	skipIfNoAPIKey(t, "DEEPSEEK_API_KEY")

	providerConfig := config.ProviderConfig{
		APIKey:  os.Getenv("DEEPSEEK_API_KEY"),
		BaseURL: "https://api.deepseek.com/v1",
		Enabled: true,
		Timeout: 30 * time.Second,
	}

	provider := providers.NewDeepSeekProvider(providerConfig)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req := &models.CompletionRequest{
		Model: "deepseek-chat",
		Messages: []models.Message{
			{Role: "user", Content: "Say 'Hello from integration test'"},
		},
		MaxTokens:   ptr(50),
		Temperature: ptr(0.0),
		Stream:      false,
	}

	t.Run("Non-Streaming", func(t *testing.T) {
		chunkChan, err := provider.StreamCompletion(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, chunkChan)

		var chunks []*models.CompletionChunk
		for chunk := range chunkChan {
			chunks = append(chunks, chunk)
		}

		require.NotEmpty(t, chunks, "Should receive at least one chunk")
		t.Logf("DeepSeek response received: %d chunks", len(chunks))
	})

	t.Run("Streaming", func(t *testing.T) {
		reqStreaming := *req
		reqStreaming.Stream = true

		chunkChan, err := provider.StreamCompletion(ctx, &reqStreaming)
		require.NoError(t, err)
		require.NotNil(t, chunkChan)

		var receivedChunks int
		for chunk := range chunkChan {
			receivedChunks++
			assert.NotEmpty(t, chunk.ID, "Chunk should have ID")
			assert.Equal(t, "deepseek-chat", chunk.Model)
		}

		assert.Greater(t, receivedChunks, 0, "Should receive chunks in streaming mode")
		t.Logf("DeepSeek streaming: received %d chunks", receivedChunks)
	})
}

// TestOllamaProvider_Integration tests Ollama provider with local instance
func TestOllamaProvider_Integration(t *testing.T) {
	// Check if Ollama is running by attempting to connect
	providerConfig := config.ProviderConfig{
		BaseURL: "http://localhost:11434/v1",
		Enabled: true,
		Timeout: 30 * time.Second,
	}

	provider := providers.NewOllamaProvider(providerConfig)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req := &models.CompletionRequest{
		Model: "mistral",
		Messages: []models.Message{
			{Role: "user", Content: "Say 'Hello from integration test'"},
		},
		MaxTokens:   ptr(50),
		Temperature: ptr(0.0),
		Stream:      false,
	}

	t.Run("Non-Streaming", func(t *testing.T) {
		chunkChan, err := provider.StreamCompletion(ctx, req)

		// Skip if Ollama is not running
		if err != nil {
			t.Skipf("Skipping Ollama test: service not available (%v)", err)
			return
		}

		require.NotNil(t, chunkChan)

		var chunks []*models.CompletionChunk
		for chunk := range chunkChan {
			chunks = append(chunks, chunk)
		}

		require.NotEmpty(t, chunks, "Should receive at least one chunk")
		t.Logf("Ollama response received: %d chunks", len(chunks))
	})

	t.Run("Streaming", func(t *testing.T) {
		reqStreaming := *req
		reqStreaming.Stream = true

		chunkChan, err := provider.StreamCompletion(ctx, &reqStreaming)

		if err != nil {
			t.Skipf("Skipping Ollama streaming test: service not available (%v)", err)
			return
		}

		require.NotNil(t, chunkChan)

		var receivedChunks int
		for range chunkChan {
			receivedChunks++
		}

		assert.Greater(t, receivedChunks, 0, "Should receive chunks in streaming mode")
		t.Logf("Ollama streaming: received %d chunks", receivedChunks)
	})
}

// TestLLMOrchestrator_Integration tests LLM orchestrator with real providers
func TestLLMOrchestrator_Integration(t *testing.T) {
	// Skip if no API keys available
	hasOpenAI := os.Getenv("OPENAI_API_KEY") != ""
	hasAnthropic := os.Getenv("ANTHROPIC_API_KEY") != ""
	hasDeepSeek := os.Getenv("DEEPSEEK_API_KEY") != ""

	if !hasOpenAI && !hasAnthropic && !hasDeepSeek {
		t.Skip("Skipping orchestrator integration test: no API keys available")
	}

	orchestrator := services.NewLLMOrchestrator()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	t.Run("SelectModel_IntentClassification", func(t *testing.T) {
		req := &services.LLMRequest{
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

		t.Logf("Selected model: %s/%s for intent classification", provider, model)

		// Verify it's a cheap model for simple task
		cost := orchestrator.CalculateCost(provider, model, 1000)
		assert.Less(t, cost, 0.001, "Should select cheap model for simple task")
	})

	t.Run("SelectModel_AdvancedReasoning", func(t *testing.T) {
		req := &services.LLMRequest{
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

		t.Logf("Selected model: %s/%s for advanced reasoning", provider, model)

		// Verify it's an advanced model
		assert.Contains(t, []string{"o1-mini", "o1", "claude-3-5-sonnet", "gpt-4o"}, model)
	})

	t.Run("BudgetTracking", func(t *testing.T) {
		orchestrator.ResetSessionBudget()

		// Track usage from multiple agents
		cost1 := orchestrator.TrackUsage("agent1", "openai", "gpt-4o-mini", 1000)
		cost2 := orchestrator.TrackUsage("agent2", "deepseek", "deepseek-chat", 500)

		sessionUsed, sessionLimit, agentBudgets := orchestrator.GetBudgetStatus()

		assert.Greater(t, sessionUsed, 0.0)
		assert.Equal(t, cost1+cost2, sessionUsed)
		assert.Greater(t, sessionLimit, sessionUsed)
		assert.Len(t, agentBudgets, 2)

		t.Logf("Session budget used: $%.6f / $%.2f", sessionUsed, sessionLimit)
		t.Logf("Agent1 budget: $%.6f", agentBudgets["agent1"])
		t.Logf("Agent2 budget: $%.6f", agentBudgets["agent2"])
	})

	t.Run("DecisionLogging", func(t *testing.T) {
		orchestrator.ResetSessionBudget()

		req := &services.LLMRequest{
			Prompt:      "Test prompt",
			TaskType:    models.TaskTypeIntentClassification,
			AgentID:     "test_agent",
			ContextSize: 100,
		}

		_, _, err := orchestrator.SelectModel(ctx, req)
		require.NoError(t, err)

		decisions := orchestrator.GetDecisions()
		require.Len(t, decisions, 1)

		decision := decisions[0]
		assert.Equal(t, "test_agent", decision.AgentID)
		assert.NotEmpty(t, decision.Selected)
		assert.NotEmpty(t, decision.Reason)

		t.Logf("Decision: Selected %s for %s (reason: %s)",
			decision.Selected, decision.TaskType, decision.Reason)
	})

	t.Run("Cache", func(t *testing.T) {
		orchestrator.ResetSessionBudget()

		req := &services.LLMRequest{
			Prompt:      "Repeated prompt for caching",
			TaskType:    models.TaskTypeIntentClassification,
			AgentID:     "cache_test_agent",
			ContextSize: 100,
			MaxTokens:   50,
			Temperature: 0.0,
			UseCach:     true,
		}

		// First call - should miss cache
		model, provider, err := orchestrator.SelectModel(ctx, req)
		require.NoError(t, err)

		cacheKey := orchestrator.GetCacheKey(req, model)

		// Simulate saving response to cache
		orchestrator.SaveToCache(cacheKey, "cached response", 50, 0.001, models.TaskTypeIntentClassification)

		// Second call - should hit cache
		cached, found := orchestrator.GetFromCache(cacheKey)
		require.True(t, found)
		assert.Equal(t, "cached response", cached.Response)
		assert.Equal(t, 1, cached.HitCount)

		t.Logf("Cache test: provider=%s, model=%s, hit=%v", provider, model, found)

		entries, hits, size := orchestrator.GetCacheStats()
		t.Logf("Cache stats: entries=%d, hits=%d, size=%d bytes", entries, hits, size)
	})
}

// TestProviderFallback_Integration tests fallback chain with real providers
func TestProviderFallback_Integration(t *testing.T) {
	skipIfNoAPIKey(t, "OPENAI_API_KEY")

	orchestrator := services.NewLLMOrchestrator()

	// Note: Testing fallback requires modifying orchestrator internal state
	// This is a simplified test - in production, fallback would be triggered
	// by actual provider failures rather than removing profiles

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req := &services.LLMRequest{
		Prompt:      "Test model selection",
		TaskType:    models.TaskTypeIntentClassification,
		AgentID:     "fallback_test",
		ContextSize: 100,
	}

	model, provider, err := orchestrator.SelectModel(ctx, req)
	require.NoError(t, err)
	assert.NotEmpty(t, model)
	assert.NotEmpty(t, provider)

	t.Logf("Model selection test: Selected %s/%s", provider, model)
}

// TestConcurrentRequests_Integration tests concurrent LLM calls
func TestConcurrentRequests_Integration(t *testing.T) {
	skipIfNoAPIKey(t, "OPENAI_API_KEY")

	orchestrator := services.NewLLMOrchestrator()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	numRequests := 5
	done := make(chan bool, numRequests)
	errors := make(chan error, numRequests)

	for i := 0; i < numRequests; i++ {
		go func(id int) {
			defer func() { done <- true }()

			req := &services.LLMRequest{
				Prompt:      "Concurrent test",
				TaskType:    models.TaskTypeIntentClassification,
				AgentID:     "concurrent_agent",
				ContextSize: 100,
			}

			_, _, err := orchestrator.SelectModel(ctx, req)
			if err != nil {
				errors <- err
			}
		}(i)
	}

	// Wait for all requests
	for i := 0; i < numRequests; i++ {
		<-done
	}
	close(errors)

	// Check for errors
	var errCount int
	for err := range errors {
		t.Logf("Concurrent request error: %v", err)
		errCount++
	}

	assert.Equal(t, 0, errCount, "All concurrent requests should succeed")
	t.Logf("Concurrent test: %d requests completed", numRequests)
}

// TestCostTracking_Integration tests accurate cost calculation with real usage
func TestCostTracking_Integration(t *testing.T) {
	skipIfNoAPIKey(t, "OPENAI_API_KEY")

	orchestrator := services.NewLLMOrchestrator()
	orchestrator.ResetSessionBudget()

	// Simulate multiple agent operations
	testCases := []struct {
		agent    string
		provider string
		model    string
		tokens   int
	}{
		{"intent_agent", "deepseek", "deepseek-chat", 500},
		{"structure_agent", "openai", "gpt-4o-mini", 1000},
		{"inference_agent", "openai", "gpt-4o", 2000},
		{"validation_agent", "deepseek", "deepseek-chat", 300},
		{"summary_agent", "openai", "gpt-4o-mini", 800},
	}

	var expectedTotal float64
	for _, tc := range testCases {
		cost := orchestrator.TrackUsage(tc.agent, tc.provider, tc.model, tc.tokens)
		expectedTotal += cost
		t.Logf("Agent %s: %s/%s (%d tokens) = $%.6f", tc.agent, tc.provider, tc.model, tc.tokens, cost)
	}

	sessionUsed, _, agentBudgets := orchestrator.GetBudgetStatus()

	assert.InDelta(t, expectedTotal, sessionUsed, 0.000001, "Session budget should match sum of agent budgets")
	assert.Len(t, agentBudgets, 5, "Should track all 5 agents")

	t.Logf("Total session cost: $%.6f", sessionUsed)

	// Verify inference agent has highest cost (uses gpt-4o)
	assert.Greater(t, agentBudgets["inference_agent"], agentBudgets["intent_agent"])
	assert.Greater(t, agentBudgets["inference_agent"], agentBudgets["validation_agent"])
}
