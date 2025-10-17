package services

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/mshogin/agents/internal/domain/models"
	"github.com/mshogin/agents/internal/domain/services"
)

// LLMOrchestrator manages dynamic LLM model selection with cost tracking and caching.
type LLMOrchestrator struct {
	// LLM providers for making actual calls
	providers map[string]services.LLMProvider

	// Model profiles indexed by "provider/model"
	profiles map[string]*models.ModelProfile

	// Selection strategies indexed by task type
	strategies map[models.TaskType]*models.ModelSelectionStrategy

	// Budget tracking
	budgetConstraints models.BudgetConstraints
	sessionBudgetUsed float64
	agentBudgetUsed   map[string]float64 // per-agent budget tracking

	// Response cache
	cache         map[string]*CachedResponse
	cacheConfig   models.CacheConfig
	cacheMu       sync.RWMutex

	// Rate limiting and throttling
	throttler *LLMThrottler

	// Metrics and logging
	decisions []models.LLMDecision
	mu        sync.RWMutex
}

// CachedResponse represents a cached LLM response.
type CachedResponse struct {
	Response   string
	Tokens     int
	Cost       float64
	CachedAt   time.Time
	ExpiresAt  time.Time
	HitCount   int
}

// LLMRequest represents a request to the LLM orchestrator.
type LLMRequest struct {
	Prompt       string
	TaskType     models.TaskType
	AgentID      string
	MaxTokens    int
	Temperature  float64
	ContextSize  int
	UseCach bool
}

// LLMResponse represents a response from the LLM orchestrator.
type LLMResponse struct {
	Response       string
	Model          string
	Provider       string
	Tokens         int
	Cost           float64
	FromCache      bool
	SelectionReason string
}

// NewLLMOrchestrator creates a new LLM orchestrator.
func NewLLMOrchestrator() *LLMOrchestrator {
	profiles := make(map[string]*models.ModelProfile)
	for _, profile := range models.DefaultModelProfiles() {
		key := fmt.Sprintf("%s/%s", profile.Provider, profile.Model)
		p := profile // Create copy to avoid pointer issues
		profiles[key] = &p
	}

	strategies := make(map[models.TaskType]*models.ModelSelectionStrategy)
	for _, strategy := range models.DefaultSelectionStrategies() {
		s := strategy // Create copy
		strategies[s.TaskType] = &s
	}

	return &LLMOrchestrator{
		providers:         make(map[string]services.LLMProvider),
		profiles:          profiles,
		strategies:        strategies,
		budgetConstraints: models.DefaultBudgetConstraints(),
		sessionBudgetUsed: 0.0,
		agentBudgetUsed:   make(map[string]float64),
		cache:             make(map[string]*CachedResponse),
		cacheConfig:       models.DefaultCacheConfig(),
		throttler:         NewLLMThrottler(profiles),
		decisions:         []models.LLMDecision{},
	}
}

// RegisterProviders registers LLM providers for making actual calls.
func (o *LLMOrchestrator) RegisterProviders(providers map[string]services.LLMProvider) {
	o.mu.Lock()
	defer o.mu.Unlock()

	for name, provider := range providers {
		o.providers[name] = provider
	}
}

// NewLLMOrchestratorWithConfig creates a new LLM orchestrator with custom configuration.
func NewLLMOrchestratorWithConfig(
	budget models.BudgetConstraints,
	cacheConfig models.CacheConfig,
) *LLMOrchestrator {
	orchestrator := NewLLMOrchestrator()
	orchestrator.budgetConstraints = budget
	orchestrator.cacheConfig = cacheConfig
	return orchestrator
}

// SelectModel selects the best model for a given task based on complexity, budget, and availability.
func (o *LLMOrchestrator) SelectModel(ctx context.Context, req *LLMRequest) (string, string, error) {
	o.mu.Lock()
	defer o.mu.Unlock()

	// Get selection strategy for task type
	strategy, exists := o.strategies[req.TaskType]
	if !exists {
		// Default to simple task strategy
		strategy = o.strategies[models.TaskTypeValidation]
	}

	// Check budget constraints
	budgetExceeded := o.checkBudgetConstraints(req.AgentID)
	if budgetExceeded {
		// Check if this is a critical agent
		if !o.isCriticalAgent(req.AgentID) {
			return "", "", fmt.Errorf("budget exceeded for agent %s", req.AgentID)
		}
		// Critical agents continue even if budget is exceeded
	}

	// Try default model first
	model, provider, reason := o.trySelectModel(strategy.DefaultModel, req, "default_for_task_type")
	if model != "" {
		o.logDecision(req.AgentID, req.TaskType, model, reason)
		return model, provider, nil
	}

	// Try fallback 1
	if strategy.Fallback1 != "" {
		model, provider, reason = o.trySelectModel(strategy.Fallback1, req, "fallback_1_default_unavailable")
		if model != "" {
			o.logDecision(req.AgentID, req.TaskType, model, reason)
			return model, provider, nil
		}
	}

	// Try fallback 2
	if strategy.Fallback2 != "" {
		model, provider, reason = o.trySelectModel(strategy.Fallback2, req, "fallback_2_all_others_unavailable")
		if model != "" {
			o.logDecision(req.AgentID, req.TaskType, model, reason)
			return model, provider, nil
		}
	}

	// No suitable model found
	return "", "", fmt.Errorf("no suitable model found for task type %s", req.TaskType)
}

// trySelectModel attempts to select a specific model.
func (o *LLMOrchestrator) trySelectModel(modelKey string, req *LLMRequest, reason string) (string, string, string) {
	if modelKey == "" {
		return "", "", ""
	}

	parts := strings.Split(modelKey, "/")
	if len(parts) != 2 {
		return "", "", ""
	}

	provider := parts[0]
	model := parts[1]

	profile, exists := o.profiles[modelKey]
	if !exists {
		return "", "", ""
	}

	// Check context size limit
	if req.ContextSize > profile.ContextLimit {
		return "", "", ""
	}

	// Model is suitable
	fullReason := fmt.Sprintf("%s (cost: $%.6f/1K, quality: %s, speed: %s)",
		reason, profile.CostPer1KTokens, profile.Quality, profile.Speed)
	return model, provider, fullReason
}

// checkBudgetConstraints checks if budget limits are exceeded.
func (o *LLMOrchestrator) checkBudgetConstraints(agentID string) bool {
	// Check session budget
	if o.sessionBudgetUsed >= o.budgetConstraints.SessionBudgetUSD {
		return true
	}

	// Check agent budget
	if agentBudget, exists := o.agentBudgetUsed[agentID]; exists {
		if agentBudget >= o.budgetConstraints.AgentBudgetUSD {
			return true
		}
	}

	// Check warning threshold
	sessionThreshold := o.budgetConstraints.SessionBudgetUSD * o.budgetConstraints.WarningThreshold
	if o.sessionBudgetUsed >= sessionThreshold {
		// Budget warning - could log or emit event
	}

	return false
}

// isCriticalAgent checks if an agent is marked as critical.
func (o *LLMOrchestrator) isCriticalAgent(agentID string) bool {
	for _, critical := range o.budgetConstraints.CriticalAgents {
		if critical == agentID {
			return true
		}
	}
	return false
}

// CalculateCost calculates the cost for a given model and token count.
func (o *LLMOrchestrator) CalculateCost(provider, model string, tokens int) float64 {
	modelKey := fmt.Sprintf("%s/%s", provider, model)
	profile, exists := o.profiles[modelKey]
	if !exists {
		return 0.0
	}

	// Cost per 1K tokens * (tokens / 1000)
	return profile.CostPer1KTokens * (float64(tokens) / 1000.0)
}

// TrackUsage tracks token usage and cost for an agent.
func (o *LLMOrchestrator) TrackUsage(agentID, provider, model string, tokens int) float64 {
	o.mu.Lock()
	defer o.mu.Unlock()

	cost := o.CalculateCost(provider, model, tokens)

	// Update session budget
	o.sessionBudgetUsed += cost

	// Update agent budget
	if _, exists := o.agentBudgetUsed[agentID]; !exists {
		o.agentBudgetUsed[agentID] = 0.0
	}
	o.agentBudgetUsed[agentID] += cost

	return cost
}

// GetCacheKey generates a cache key for a request.
func (o *LLMOrchestrator) GetCacheKey(req *LLMRequest, model string) string {
	// Hash the prompt and parameters
	data := fmt.Sprintf("%s|%s|%d|%.2f|%s",
		req.Prompt, model, req.MaxTokens, req.Temperature, req.TaskType)

	hash := sha256.Sum256([]byte(data))
	return fmt.Sprintf("%x", hash)
}

// GetFromCache retrieves a cached response if available and not expired.
func (o *LLMOrchestrator) GetFromCache(cacheKey string) (*CachedResponse, bool) {
	if !o.cacheConfig.Enabled {
		return nil, false
	}

	o.cacheMu.RLock()
	defer o.cacheMu.RUnlock()

	cached, exists := o.cache[cacheKey]
	if !exists {
		return nil, false
	}

	// Check expiration
	if time.Now().After(cached.ExpiresAt) {
		return nil, false
	}

	// Increment hit count
	cached.HitCount++

	return cached, true
}

// SaveToCache saves a response to the cache.
func (o *LLMOrchestrator) SaveToCache(cacheKey string, response string, tokens int, cost float64, taskType models.TaskType) {
	if !o.cacheConfig.Enabled {
		return
	}

	// Determine TTL based on task type
	var ttl int
	switch taskType {
	case models.TaskTypeIntentClassification, models.TaskTypeEntityExtraction, models.TaskTypeValidation:
		ttl = o.cacheConfig.ClassificationTTL
	case models.TaskTypeTextSynthesis, models.TaskTypeMediumSynthesis:
		ttl = o.cacheConfig.SynthesisTTL
	case models.TaskTypeInference, models.TaskTypeAdvancedInference, models.TaskTypeDeepReasoning:
		ttl = o.cacheConfig.InferenceTTL
	default:
		ttl = o.cacheConfig.SynthesisTTL
	}

	o.cacheMu.Lock()
	defer o.cacheMu.Unlock()

	now := time.Now()
	cached := &CachedResponse{
		Response:  response,
		Tokens:    tokens,
		Cost:      cost,
		CachedAt:  now,
		ExpiresAt: now.Add(time.Duration(ttl) * time.Second),
		HitCount:  0,
	}

	o.cache[cacheKey] = cached

	// TODO: Implement cache size limit and eviction policy
}

// ClearExpiredCache removes expired entries from the cache.
func (o *LLMOrchestrator) ClearExpiredCache() {
	o.cacheMu.Lock()
	defer o.cacheMu.Unlock()

	now := time.Now()
	for key, cached := range o.cache {
		if now.After(cached.ExpiresAt) {
			delete(o.cache, key)
		}
	}
}

// GetCacheStats returns cache statistics.
func (o *LLMOrchestrator) GetCacheStats() (totalEntries, hits, size int) {
	o.cacheMu.RLock()
	defer o.cacheMu.RUnlock()

	totalEntries = len(o.cache)
	for _, cached := range o.cache {
		hits += cached.HitCount
		size += len(cached.Response)
	}

	return totalEntries, hits, size
}

// logDecision logs an LLM selection decision.
func (o *LLMOrchestrator) logDecision(agentID string, taskType models.TaskType, model, reason string) {
	decision := models.LLMDecision{
		Timestamp: time.Now(),
		AgentID:   agentID,
		TaskType:  string(taskType),
		Selected:  model,
		Reason:    reason,
	}

	o.decisions = append(o.decisions, decision)
}

// GetDecisions returns all LLM selection decisions.
func (o *LLMOrchestrator) GetDecisions() []models.LLMDecision {
	o.mu.RLock()
	defer o.mu.RUnlock()

	// Return a copy
	decisions := make([]models.LLMDecision, len(o.decisions))
	copy(decisions, o.decisions)
	return decisions
}

// GetBudgetStatus returns current budget usage.
func (o *LLMOrchestrator) GetBudgetStatus() (sessionUsed, sessionLimit float64, agentBudgets map[string]float64) {
	o.mu.RLock()
	defer o.mu.RUnlock()

	// Copy agent budgets
	agentBudgets = make(map[string]float64)
	for agent, budget := range o.agentBudgetUsed {
		agentBudgets[agent] = budget
	}

	return o.sessionBudgetUsed, o.budgetConstraints.SessionBudgetUSD, agentBudgets
}

// ResetSessionBudget resets the session budget tracking.
func (o *LLMOrchestrator) ResetSessionBudget() {
	o.mu.Lock()
	defer o.mu.Unlock()

	o.sessionBudgetUsed = 0.0
	o.agentBudgetUsed = make(map[string]float64)
	o.decisions = []models.LLMDecision{}
}

// AddModelProfile adds or updates a model profile.
func (o *LLMOrchestrator) AddModelProfile(profile models.ModelProfile) {
	o.mu.Lock()
	defer o.mu.Unlock()

	key := fmt.Sprintf("%s/%s", profile.Provider, profile.Model)
	o.profiles[key] = &profile
}

// AddSelectionStrategy adds or updates a selection strategy.
func (o *LLMOrchestrator) AddSelectionStrategy(strategy models.ModelSelectionStrategy) {
	o.mu.Lock()
	defer o.mu.Unlock()

	o.strategies[strategy.TaskType] = &strategy
}

// Serialize returns JSON representation of orchestrator state.
func (o *LLMOrchestrator) Serialize() ([]byte, error) {
	o.mu.RLock()
	defer o.mu.RUnlock()

	state := map[string]interface{}{
		"session_budget_used": o.sessionBudgetUsed,
		"agent_budgets":       o.agentBudgetUsed,
		"decisions_count":     len(o.decisions),
		"cache_entries":       len(o.cache),
	}

	return json.Marshal(state)
}

// WaitForThrottle waits for rate limit permission to make a request.
// Returns error if rate limit is reached or context is cancelled.
func (o *LLMOrchestrator) WaitForThrottle(ctx context.Context, provider, model string) error {
	return o.throttler.WaitForToken(ctx, provider, model)
}

// GetRequestTimeout returns the configured timeout for a provider/model.
func (o *LLMOrchestrator) GetRequestTimeout(provider, model string) time.Duration {
	return o.throttler.GetTimeout(provider, model)
}

// GetThrottleStats returns current throttling statistics.
func (o *LLMOrchestrator) GetThrottleStats() map[string]ThrottleStats {
	return o.throttler.GetStats()
}

// UpdateProviderThrottle updates throttling settings for a specific provider/model.
func (o *LLMOrchestrator) UpdateProviderThrottle(provider, model string, maxRPS int, timeoutMS int) {
	o.throttler.UpdateRateLimit(provider, model, maxRPS, timeoutMS)
}

// Call makes an LLM request with automatic model selection, caching, and cost tracking.
// This is the main entry point for agents to make LLM calls.
func (o *LLMOrchestrator) Call(ctx context.Context, req *LLMRequest) (*LLMResponse, error) {
	// Check cache first
	cacheKey := ""
	if req.UseCach {
		model, _, err := o.SelectModel(ctx, req)
		if err == nil {
			cacheKey = o.GetCacheKey(req, model)
			if cached, found := o.GetFromCache(cacheKey); found {
				// Return cached response
				return &LLMResponse{
					Response:        cached.Response,
					Model:           model,
					Provider:        "",
					Tokens:          cached.Tokens,
					Cost:            cached.Cost,
					FromCache:       true,
					SelectionReason: "cached_response",
				}, nil
			}
		}
	}

	// Select best model for this task
	model, provider, err := o.SelectModel(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to select model: %w", err)
	}

	// Get provider
	llmProvider, exists := o.providers[provider]
	if !exists {
		return nil, fmt.Errorf("provider %s not available", provider)
	}

	// Wait for rate limit
	if err := o.WaitForThrottle(ctx, provider, model); err != nil {
		return nil, fmt.Errorf("rate limit error: %w", err)
	}

	// Build completion request
	maxTokens := req.MaxTokens
	temperature := req.Temperature
	completionReq := &models.CompletionRequest{
		Model: model,
		Messages: []models.Message{
			{
				Role:    "user",
				Content: req.Prompt,
			},
		},
		MaxTokens:   &maxTokens,
		Temperature: &temperature,
		Stream:      false,
	}

	// Call provider (non-streaming for simplicity)
	responseChan, err := llmProvider.StreamCompletion(ctx, completionReq)
	if err != nil {
		return nil, fmt.Errorf("provider call failed: %w", err)
	}

	// Collect streaming response
	fullResponse := ""
	for chunk := range responseChan {
		if len(chunk.Choices) > 0 && chunk.Choices[0].Delta.Content != "" {
			fullResponse += chunk.Choices[0].Delta.Content
		}
	}

	// Estimate tokens (rough approximation: 1 token ~= 4 characters)
	totalTokens := len(req.Prompt)/4 + len(fullResponse)/4

	// Calculate cost and track usage
	cost := o.TrackUsage(req.AgentID, provider, model, totalTokens)

	// Save to cache if enabled
	if req.UseCach && cacheKey != "" {
		o.SaveToCache(cacheKey, fullResponse, totalTokens, cost, req.TaskType)
	}

	return &LLMResponse{
		Response:        fullResponse,
		Model:           model,
		Provider:        provider,
		Tokens:          totalTokens,
		Cost:            cost,
		FromCache:       false,
		SelectionReason: fmt.Sprintf("selected %s/%s for %s task", provider, model, req.TaskType),
	}, nil
}
