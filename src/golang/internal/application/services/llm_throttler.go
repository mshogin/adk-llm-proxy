package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/mshogin/agents/internal/domain/models"
)

// LLMThrottler manages rate limiting and timeouts for LLM providers.
type LLMThrottler struct {
	// Per-provider rate limiters
	limiters map[string]*rateLimiter
	mu       sync.RWMutex
}

// rateLimiter implements token bucket algorithm for rate limiting.
type rateLimiter struct {
	maxRequests    int           // Maximum requests per second
	tokens         int           // Current available tokens
	lastRefill     time.Time     // Last time tokens were refilled
	requestTimeout time.Duration // Request timeout
	mu             sync.Mutex
}

// NewLLMThrottler creates a new throttler with profiles.
func NewLLMThrottler(profiles map[string]*models.ModelProfile) *LLMThrottler {
	throttler := &LLMThrottler{
		limiters: make(map[string]*rateLimiter),
	}

	// Create rate limiters for each provider
	for key, profile := range profiles {
		if profile.MaxRequestsPerSecond > 0 {
			throttler.limiters[key] = &rateLimiter{
				maxRequests:    profile.MaxRequestsPerSecond,
				tokens:         profile.MaxRequestsPerSecond, // Start with full bucket
				lastRefill:     time.Now(),
				requestTimeout: time.Duration(profile.RequestTimeoutMS) * time.Millisecond,
			}
		}
	}

	return throttler
}

// WaitForToken waits for permission to make a request to the specified provider/model.
// Returns error if rate limit is reached or context is cancelled.
func (t *LLMThrottler) WaitForToken(ctx context.Context, provider, model string) error {
	modelKey := fmt.Sprintf("%s/%s", provider, model)

	t.mu.RLock()
	limiter, exists := t.limiters[modelKey]
	t.mu.RUnlock()

	if !exists {
		// No rate limit configured - allow immediately
		return nil
	}

	// Try to acquire token
	for {
		if limiter.tryAcquire() {
			return nil
		}

		// Wait a bit before retrying
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(10 * time.Millisecond):
			// Continue loop
		}
	}
}

// GetTimeout returns the configured timeout for a provider/model.
func (t *LLMThrottler) GetTimeout(provider, model string) time.Duration {
	modelKey := fmt.Sprintf("%s/%s", provider, model)

	t.mu.RLock()
	limiter, exists := t.limiters[modelKey]
	t.mu.RUnlock()

	if !exists {
		// Default timeout if not configured
		return 30 * time.Second
	}

	return limiter.requestTimeout
}

// tryAcquire attempts to acquire a token from the bucket.
// Returns true if token acquired, false otherwise.
func (r *rateLimiter) tryAcquire() bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Refill tokens based on elapsed time (token bucket algorithm)
	now := time.Now()
	elapsed := now.Sub(r.lastRefill)

	// Refill tokens: one token per (1 second / maxRequests)
	tokensToAdd := int(elapsed.Seconds() * float64(r.maxRequests))
	if tokensToAdd > 0 {
		r.tokens += tokensToAdd
		if r.tokens > r.maxRequests {
			r.tokens = r.maxRequests // Cap at max
		}
		r.lastRefill = now
	}

	// Try to acquire token
	if r.tokens > 0 {
		r.tokens--
		return true
	}

	return false
}

// UpdateRateLimit updates the rate limit for a specific provider/model.
func (t *LLMThrottler) UpdateRateLimit(provider, model string, maxRPS int, timeoutMS int) {
	modelKey := fmt.Sprintf("%s/%s", provider, model)

	t.mu.Lock()
	defer t.mu.Unlock()

	if maxRPS > 0 {
		t.limiters[modelKey] = &rateLimiter{
			maxRequests:    maxRPS,
			tokens:         maxRPS,
			lastRefill:     time.Now(),
			requestTimeout: time.Duration(timeoutMS) * time.Millisecond,
		}
	} else {
		// Remove rate limit if maxRPS is 0
		delete(t.limiters, modelKey)
	}
}

// GetStats returns current throttling statistics.
func (t *LLMThrottler) GetStats() map[string]ThrottleStats {
	t.mu.RLock()
	defer t.mu.RUnlock()

	stats := make(map[string]ThrottleStats)
	for key, limiter := range t.limiters {
		limiter.mu.Lock()
		stats[key] = ThrottleStats{
			MaxRequests:      limiter.maxRequests,
			AvailableTokens:  limiter.tokens,
			TimeoutMS:        int(limiter.requestTimeout.Milliseconds()),
		}
		limiter.mu.Unlock()
	}

	return stats
}

// ThrottleStats contains throttling statistics for a provider/model.
type ThrottleStats struct {
	MaxRequests     int `json:"max_requests"`
	AvailableTokens int `json:"available_tokens"`
	TimeoutMS       int `json:"timeout_ms"`
}
