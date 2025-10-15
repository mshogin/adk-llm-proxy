package services

import (
	"context"
	"testing"
	"time"

	"github.com/mshogin/agents/internal/domain/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewLLMThrottler tests throttler initialization
func TestNewLLMThrottler(t *testing.T) {
	profiles := map[string]*models.ModelProfile{
		"openai/gpt-4o-mini": {
			Provider:             "openai",
			Model:                "gpt-4o-mini",
			MaxRequestsPerSecond: 50,
			RequestTimeoutMS:     15000,
		},
		"deepseek/deepseek-chat": {
			Provider:             "deepseek",
			Model:                "deepseek-chat",
			MaxRequestsPerSecond: 20,
			RequestTimeoutMS:     15000,
		},
		"ollama/mistral": {
			Provider:             "ollama",
			Model:                "mistral",
			MaxRequestsPerSecond: 0, // Unlimited
			RequestTimeoutMS:     10000,
		},
	}

	throttler := NewLLMThrottler(profiles)

	assert.NotNil(t, throttler)
	assert.NotNil(t, throttler.limiters)

	// Should have limiters for models with MaxRequestsPerSecond > 0
	assert.Len(t, throttler.limiters, 2)
	assert.Contains(t, throttler.limiters, "openai/gpt-4o-mini")
	assert.Contains(t, throttler.limiters, "deepseek/deepseek-chat")
	assert.NotContains(t, throttler.limiters, "ollama/mistral") // No limit
}

// TestWaitForToken tests successful token acquisition
func TestWaitForToken(t *testing.T) {
	profiles := map[string]*models.ModelProfile{
		"openai/gpt-4o-mini": {
			Provider:             "openai",
			Model:                "gpt-4o-mini",
			MaxRequestsPerSecond: 100, // Very high to avoid waiting
			RequestTimeoutMS:     15000,
		},
	}

	throttler := NewLLMThrottler(profiles)
	ctx := context.Background()

	// Should succeed immediately
	err := throttler.WaitForToken(ctx, "openai", "gpt-4o-mini")
	assert.NoError(t, err)
}

// TestWaitForToken_Unlimited tests unlimited rate (no throttling)
func TestWaitForToken_Unlimited(t *testing.T) {
	profiles := map[string]*models.ModelProfile{
		"ollama/mistral": {
			Provider:             "ollama",
			Model:                "mistral",
			MaxRequestsPerSecond: 0, // Unlimited
			RequestTimeoutMS:     10000,
		},
	}

	throttler := NewLLMThrottler(profiles)
	ctx := context.Background()

	// Should succeed immediately with no limit
	err := throttler.WaitForToken(ctx, "ollama", "mistral")
	assert.NoError(t, err)
}

// TestWaitForToken_RateLimit tests rate limiting behavior
func TestWaitForToken_RateLimit(t *testing.T) {
	profiles := map[string]*models.ModelProfile{
		"test/model": {
			Provider:             "test",
			Model:                "model",
			MaxRequestsPerSecond: 10, // 10 requests per second
			RequestTimeoutMS:     5000,
		},
	}

	throttler := NewLLMThrottler(profiles)
	ctx := context.Background()

	// Make multiple requests quickly
	start := time.Now()
	for i := 0; i < 15; i++ {
		err := throttler.WaitForToken(ctx, "test", "model")
		require.NoError(t, err)
	}
	duration := time.Since(start)

	// Should take at least 400ms for 15 requests at 10 RPS
	// (initial bucket has 10 tokens, need to wait for 5 more)
	assert.GreaterOrEqual(t, duration.Milliseconds(), int64(400))
}

// TestWaitForToken_ContextCancellation tests context cancellation
func TestWaitForToken_ContextCancellation(t *testing.T) {
	profiles := map[string]*models.ModelProfile{
		"test/slow": {
			Provider:             "test",
			Model:                "slow",
			MaxRequestsPerSecond: 1, // Very slow
			RequestTimeoutMS:     5000,
		},
	}

	throttler := NewLLMThrottler(profiles)

	// Exhaust the token bucket
	ctx := context.Background()
	for i := 0; i < 5; i++ {
		_ = throttler.WaitForToken(ctx, "test", "slow")
	}

	// Create context with short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	// This should fail due to context cancellation
	err := throttler.WaitForToken(ctx, "test", "slow")
	assert.Error(t, err)
	assert.Equal(t, context.DeadlineExceeded, err)
}

// TestGetTimeout tests timeout retrieval
func TestGetTimeout(t *testing.T) {
	profiles := map[string]*models.ModelProfile{
		"openai/gpt-4o-mini": {
			Provider:             "openai",
			Model:                "gpt-4o-mini",
			MaxRequestsPerSecond: 50,
			RequestTimeoutMS:     15000,
		},
		"openai/o1": {
			Provider:             "openai",
			Model:                "o1",
			MaxRequestsPerSecond: 5,
			RequestTimeoutMS:     120000,
		},
	}

	throttler := NewLLMThrottler(profiles)

	// Test configured timeouts
	timeout1 := throttler.GetTimeout("openai", "gpt-4o-mini")
	assert.Equal(t, 15*time.Second, timeout1)

	timeout2 := throttler.GetTimeout("openai", "o1")
	assert.Equal(t, 120*time.Second, timeout2)

	// Test default timeout for unconfigured model
	timeoutDefault := throttler.GetTimeout("unknown", "model")
	assert.Equal(t, 30*time.Second, timeoutDefault)
}

// TestUpdateRateLimit tests dynamic rate limit updates
func TestUpdateRateLimit(t *testing.T) {
	throttler := NewLLMThrottler(make(map[string]*models.ModelProfile))

	// Add new rate limit
	throttler.UpdateRateLimit("test", "model", 20, 10000)

	stats := throttler.GetStats()
	require.Contains(t, stats, "test/model")
	assert.Equal(t, 20, stats["test/model"].MaxRequests)
	assert.Equal(t, 10000, stats["test/model"].TimeoutMS)

	// Update existing rate limit
	throttler.UpdateRateLimit("test", "model", 50, 15000)

	stats = throttler.GetStats()
	require.Contains(t, stats, "test/model")
	assert.Equal(t, 50, stats["test/model"].MaxRequests)
	assert.Equal(t, 15000, stats["test/model"].TimeoutMS)

	// Remove rate limit (set to 0)
	throttler.UpdateRateLimit("test", "model", 0, 0)

	stats = throttler.GetStats()
	assert.NotContains(t, stats, "test/model")
}

// TestGetStats tests statistics retrieval
func TestGetStats(t *testing.T) {
	profiles := map[string]*models.ModelProfile{
		"openai/gpt-4o-mini": {
			Provider:             "openai",
			Model:                "gpt-4o-mini",
			MaxRequestsPerSecond: 50,
			RequestTimeoutMS:     15000,
		},
		"deepseek/deepseek-chat": {
			Provider:             "deepseek",
			Model:                "deepseek-chat",
			MaxRequestsPerSecond: 20,
			RequestTimeoutMS:     15000,
		},
	}

	throttler := NewLLMThrottler(profiles)
	stats := throttler.GetStats()

	assert.Len(t, stats, 2)

	// Check OpenAI stats
	require.Contains(t, stats, "openai/gpt-4o-mini")
	assert.Equal(t, 50, stats["openai/gpt-4o-mini"].MaxRequests)
	assert.Equal(t, 50, stats["openai/gpt-4o-mini"].AvailableTokens) // Initial full bucket
	assert.Equal(t, 15000, stats["openai/gpt-4o-mini"].TimeoutMS)

	// Check DeepSeek stats
	require.Contains(t, stats, "deepseek/deepseek-chat")
	assert.Equal(t, 20, stats["deepseek/deepseek-chat"].MaxRequests)
	assert.Equal(t, 20, stats["deepseek/deepseek-chat"].AvailableTokens)
	assert.Equal(t, 15000, stats["deepseek/deepseek-chat"].TimeoutMS)
}

// TestTokenBucketRefill tests token bucket refill mechanism
func TestTokenBucketRefill(t *testing.T) {
	profiles := map[string]*models.ModelProfile{
		"test/model": {
			Provider:             "test",
			Model:                "model",
			MaxRequestsPerSecond: 10, // 10 tokens per second
			RequestTimeoutMS:     5000,
		},
	}

	throttler := NewLLMThrottler(profiles)
	ctx := context.Background()

	// Exhaust all tokens
	for i := 0; i < 10; i++ {
		err := throttler.WaitForToken(ctx, "test", "model")
		require.NoError(t, err)
	}

	// Check stats - should have 0 available tokens
	stats := throttler.GetStats()
	assert.Equal(t, 0, stats["test/model"].AvailableTokens)

	// Wait 1 second for refill
	time.Sleep(1 * time.Second)

	// Make one more request - should succeed after refill
	err := throttler.WaitForToken(ctx, "test", "model")
	assert.NoError(t, err)

	// Check stats - should have tokens refilled (minus the one we just used)
	stats = throttler.GetStats()
	assert.Greater(t, stats["test/model"].AvailableTokens, 5)
}

// TestThrottler_ConcurrentAccess tests thread-safe concurrent access
func TestThrottler_ConcurrentAccess(t *testing.T) {
	profiles := map[string]*models.ModelProfile{
		"test/concurrent": {
			Provider:             "test",
			Model:                "concurrent",
			MaxRequestsPerSecond: 100, // High enough for test
			RequestTimeoutMS:     5000,
		},
	}

	throttler := NewLLMThrottler(profiles)
	ctx := context.Background()

	// Run concurrent requests
	done := make(chan bool, 20)

	for i := 0; i < 20; i++ {
		go func() {
			defer func() { done <- true }()

			err := throttler.WaitForToken(ctx, "test", "concurrent")
			assert.NoError(t, err)

			// Also test stats concurrently
			_ = throttler.GetStats()
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 20; i++ {
		<-done
	}

	// Verify final state is consistent
	stats := throttler.GetStats()
	require.Contains(t, stats, "test/concurrent")
	assert.GreaterOrEqual(t, stats["test/concurrent"].MaxRequests, stats["test/concurrent"].AvailableTokens)
}

// TestOrchestrator_WithThrottling tests integration with LLM orchestrator
func TestOrchestrator_WithThrottling(t *testing.T) {
	orchestrator := NewLLMOrchestrator()

	// Get throttle stats
	stats := orchestrator.GetThrottleStats()
	assert.NotEmpty(t, stats)

	// Check some expected providers
	assert.Contains(t, stats, "openai/gpt-4o-mini")
	assert.Contains(t, stats, "deepseek/deepseek-chat")

	// Test wait for throttle
	ctx := context.Background()
	err := orchestrator.WaitForThrottle(ctx, "openai", "gpt-4o-mini")
	assert.NoError(t, err)

	// Test get timeout
	timeout := orchestrator.GetRequestTimeout("openai", "gpt-4o-mini")
	assert.Equal(t, 15*time.Second, timeout)

	// Test update throttle
	orchestrator.UpdateProviderThrottle("custom", "model", 100, 20000)
	timeout = orchestrator.GetRequestTimeout("custom", "model")
	assert.Equal(t, 20*time.Second, timeout)
}
