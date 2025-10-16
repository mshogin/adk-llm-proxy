package metrics

import (
	"sync"
	"time"
)

// LLMMetricsCollector tracks LLM-specific metrics including cache hits,
// model selection decisions, and detailed token usage.
//
// Design Principles:
// - Thread-safe metric collection
// - Per-provider and per-model tracking
// - Cache hit/miss tracking
// - Model selection decision logging
// - Token breakdown (prompt vs completion)
type LLMMetricsCollector struct {
	sessionID string
	traceID   string

	// LLM call metrics by provider and model
	llmCalls   map[string]*LLMCallMetrics // key: provider/model
	mu         sync.RWMutex

	// Cache metrics
	cacheHits   int
	cacheMisses int

	// Model selection decisions
	decisions []ModelSelectionDecision

	// Session-level totals
	totalCalls       int
	totalPromptTokens     int
	totalCompletionTokens int
	totalCost        float64
}

// LLMCallMetrics tracks metrics for a specific provider/model combination.
type LLMCallMetrics struct {
	Provider          string
	Model             string
	CallCount         int
	PromptTokens      int
	CompletionTokens  int
	TotalTokens       int
	TotalCost         float64
	AvgLatencyMS      int64
	TotalLatencyMS    int64
	FailureCount      int
	LastError         string
	LastUsedAt        time.Time
}

// ModelSelectionDecision records why a particular model was selected.
type ModelSelectionDecision struct {
	Timestamp       time.Time
	AgentID         string
	TaskType        string        // e.g., "intent_classification", "inference"
	ContextSize     int           // tokens
	SelectedModel   string        // e.g., "openai/gpt-4o-mini"
	Reason          string        // e.g., "optimal_for_task", "budget_constraint", "fallback"
	AlternativeModels []string    // models that were considered but not selected
	EstimatedCost   float64
}

// CacheMetrics returns cache hit/miss statistics.
type CacheMetrics struct {
	Hits     int
	Misses   int
	HitRate  float64 // percentage (0-100)
}

// LLMSessionMetrics represents session-level LLM aggregate metrics.
type LLMSessionMetrics struct {
	SessionID             string
	TraceID               string
	TotalCalls            int
	TotalPromptTokens     int
	TotalCompletionTokens int
	TotalTokens           int
	TotalCostUSD          float64
	CacheHitRate          float64
	UniqueModels          int
	DecisionCount         int
	ByProvider            map[string]*ProviderMetrics
}

// ProviderMetrics tracks metrics for a specific LLM provider.
type ProviderMetrics struct {
	Provider          string
	CallCount         int
	TotalTokens       int
	TotalCost         float64
	AvgLatencyMS      int64
	FailureCount      int
	FailureRate       float64 // percentage (0-100)
}

// NewLLMMetricsCollector creates a new LLM metrics collector.
func NewLLMMetricsCollector(sessionID, traceID string) *LLMMetricsCollector {
	return &LLMMetricsCollector{
		sessionID: sessionID,
		traceID:   traceID,
		llmCalls:  make(map[string]*LLMCallMetrics),
		decisions: []ModelSelectionDecision{},
	}
}

// RecordLLMCall records metrics from an LLM API call.
func (c *LLMMetricsCollector) RecordLLMCall(
	provider, model string,
	promptTokens, completionTokens int,
	cost float64,
	latencyMS int64,
	err error,
) {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := provider + "/" + model

	// Get or create metrics for this provider/model
	metrics, exists := c.llmCalls[key]
	if !exists {
		metrics = &LLMCallMetrics{
			Provider: provider,
			Model:    model,
		}
		c.llmCalls[key] = metrics
	}

	// Update call count
	metrics.CallCount++
	c.totalCalls++

	// Update token counts
	metrics.PromptTokens += promptTokens
	metrics.CompletionTokens += completionTokens
	metrics.TotalTokens += promptTokens + completionTokens

	c.totalPromptTokens += promptTokens
	c.totalCompletionTokens += completionTokens

	// Update cost
	metrics.TotalCost += cost
	c.totalCost += cost

	// Update latency
	metrics.TotalLatencyMS += latencyMS
	metrics.AvgLatencyMS = metrics.TotalLatencyMS / int64(metrics.CallCount)

	// Update error tracking
	if err != nil {
		metrics.FailureCount++
		metrics.LastError = err.Error()
	}

	// Update timestamp
	metrics.LastUsedAt = time.Now()
}

// RecordCacheHit records a cache hit event.
func (c *LLMMetricsCollector) RecordCacheHit() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cacheHits++
}

// RecordCacheMiss records a cache miss event.
func (c *LLMMetricsCollector) RecordCacheMiss() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cacheMisses++
}

// RecordModelSelection records a model selection decision.
func (c *LLMMetricsCollector) RecordModelSelection(decision ModelSelectionDecision) {
	c.mu.Lock()
	defer c.mu.Unlock()

	decision.Timestamp = time.Now()
	c.decisions = append(c.decisions, decision)
}

// GetCacheMetrics returns cache hit/miss statistics.
func (c *LLMMetricsCollector) GetCacheMetrics() CacheMetrics {
	c.mu.RLock()
	defer c.mu.RUnlock()

	total := c.cacheHits + c.cacheMisses
	hitRate := 0.0
	if total > 0 {
		hitRate = float64(c.cacheHits) / float64(total) * 100.0
	}

	return CacheMetrics{
		Hits:    c.cacheHits,
		Misses:  c.cacheMisses,
		HitRate: hitRate,
	}
}

// GetModelMetrics returns metrics for a specific provider/model.
func (c *LLMMetricsCollector) GetModelMetrics(provider, model string) *LLMCallMetrics {
	c.mu.RLock()
	defer c.mu.RUnlock()

	key := provider + "/" + model
	metrics, exists := c.llmCalls[key]
	if !exists {
		return nil
	}

	// Return a copy to avoid race conditions
	copy := *metrics
	return &copy
}

// GetAllModelMetrics returns metrics for all provider/model combinations.
func (c *LLMMetricsCollector) GetAllModelMetrics() map[string]*LLMCallMetrics {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Return deep copy to avoid race conditions
	result := make(map[string]*LLMCallMetrics, len(c.llmCalls))
	for key, metrics := range c.llmCalls {
		copy := *metrics
		result[key] = &copy
	}

	return result
}

// GetModelSelectionDecisions returns all model selection decisions.
func (c *LLMMetricsCollector) GetModelSelectionDecisions() []ModelSelectionDecision {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Return a copy to avoid race conditions
	decisions := make([]ModelSelectionDecision, len(c.decisions))
	copy(decisions, c.decisions)
	return decisions
}

// GetLLMSessionMetrics returns session-level LLM aggregate metrics.
func (c *LLMMetricsCollector) GetLLMSessionMetrics() LLMSessionMetrics {
	c.mu.RLock()
	defer c.mu.RUnlock()

	cacheMetrics := c.getCacheMetricsUnsafe()

	// Aggregate by provider
	byProvider := make(map[string]*ProviderMetrics)
	for _, metrics := range c.llmCalls {
		provider := metrics.Provider

		providerMetrics, exists := byProvider[provider]
		if !exists {
			providerMetrics = &ProviderMetrics{
				Provider: provider,
			}
			byProvider[provider] = providerMetrics
		}

		providerMetrics.CallCount += metrics.CallCount
		providerMetrics.TotalTokens += metrics.TotalTokens
		providerMetrics.TotalCost += metrics.TotalCost
		providerMetrics.FailureCount += metrics.FailureCount

		// Update average latency (weighted by call count)
		providerMetrics.AvgLatencyMS =
			(providerMetrics.AvgLatencyMS*int64(providerMetrics.CallCount-metrics.CallCount) +
			 metrics.TotalLatencyMS) / int64(providerMetrics.CallCount)

		// Calculate failure rate
		if providerMetrics.CallCount > 0 {
			providerMetrics.FailureRate =
				float64(providerMetrics.FailureCount) / float64(providerMetrics.CallCount) * 100.0
		}
	}

	return LLMSessionMetrics{
		SessionID:             c.sessionID,
		TraceID:               c.traceID,
		TotalCalls:            c.totalCalls,
		TotalPromptTokens:     c.totalPromptTokens,
		TotalCompletionTokens: c.totalCompletionTokens,
		TotalTokens:           c.totalPromptTokens + c.totalCompletionTokens,
		TotalCostUSD:          c.totalCost,
		CacheHitRate:          cacheMetrics.HitRate,
		UniqueModels:          len(c.llmCalls),
		DecisionCount:         len(c.decisions),
		ByProvider:            byProvider,
	}
}

// getCacheMetricsUnsafe returns cache metrics without locking (internal use only).
func (c *LLMMetricsCollector) getCacheMetricsUnsafe() CacheMetrics {
	total := c.cacheHits + c.cacheMisses
	hitRate := 0.0
	if total > 0 {
		hitRate = float64(c.cacheHits) / float64(total) * 100.0
	}

	return CacheMetrics{
		Hits:    c.cacheHits,
		Misses:  c.cacheMisses,
		HitRate: hitRate,
	}
}

// Reset resets all LLM metrics (useful for testing or session restart).
func (c *LLMMetricsCollector) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.llmCalls = make(map[string]*LLMCallMetrics)
	c.decisions = []ModelSelectionDecision{}
	c.cacheHits = 0
	c.cacheMisses = 0
	c.totalCalls = 0
	c.totalPromptTokens = 0
	c.totalCompletionTokens = 0
	c.totalCost = 0.0
}
