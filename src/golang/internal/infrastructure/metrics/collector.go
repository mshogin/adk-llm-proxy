package metrics

import (
	"sync"
	"time"

	"github.com/mshogin/agents/internal/domain/models"
)

// Collector aggregates metrics from agent executions.
//
// Design Principles:
// - Thread-safe metric collection
// - Per-agent metric tracking
// - Session-level aggregation
// - Export to multiple formats (Prometheus, JSON logs, context)
//
// Tracked Metrics:
// - Duration per agent execution
// - LLM call counts and token usage
// - Success/failure rates
// - Context size and artifact counts
// - Cost tracking per agent and session
type Collector struct {
	sessionID string
	traceID   string
	startTime time.Time

	// Metrics by agent ID
	agentMetrics map[string]*AgentMetrics
	mu           sync.RWMutex

	// Session-level totals
	totalDuration time.Duration
	totalCost     float64
	totalTokens   int
}

// AgentMetrics tracks metrics for a single agent execution.
type AgentMetrics struct {
	AgentID      string
	ExecutionCount int
	TotalDurationMS int64
	AvgDurationMS   int64
	LLMCalls        int
	TotalTokens     int
	TotalCost       float64
	SuccessCount    int
	FailureCount    int
	LastStatus      string
	LastError       string
}

// NewCollector creates a new metrics collector for a session.
func NewCollector(sessionID, traceID string) *Collector {
	return &Collector{
		sessionID:    sessionID,
		traceID:      traceID,
		startTime:    time.Now(),
		agentMetrics: make(map[string]*AgentMetrics),
	}
}

// RecordAgentExecution records metrics from a completed agent execution.
func (c *Collector) RecordAgentExecution(run models.AgentRun) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Get or create agent metrics
	metrics, exists := c.agentMetrics[run.AgentID]
	if !exists {
		metrics = &AgentMetrics{
			AgentID: run.AgentID,
		}
		c.agentMetrics[run.AgentID] = metrics
	}

	// Update execution count
	metrics.ExecutionCount++

	// Update duration
	metrics.TotalDurationMS += run.DurationMS
	metrics.AvgDurationMS = metrics.TotalDurationMS / int64(metrics.ExecutionCount)

	// Update status
	metrics.LastStatus = run.Status
	if run.Error != "" {
		metrics.LastError = run.Error
		metrics.FailureCount++
	} else {
		metrics.SuccessCount++
	}

	// Update session totals
	c.totalDuration += time.Duration(run.DurationMS) * time.Millisecond
}

// RecordLLMUsage records LLM usage metrics.
func (c *Collector) RecordLLMUsage(agentID string, tokens int, cost float64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Get or create agent metrics
	metrics, exists := c.agentMetrics[agentID]
	if !exists {
		metrics = &AgentMetrics{
			AgentID: agentID,
		}
		c.agentMetrics[agentID] = metrics
	}

	// Update LLM metrics
	metrics.LLMCalls++
	metrics.TotalTokens += tokens
	metrics.TotalCost += cost

	// Update session totals
	c.totalTokens += tokens
	c.totalCost += cost
}

// GetAgentMetrics returns metrics for a specific agent.
func (c *Collector) GetAgentMetrics(agentID string) *AgentMetrics {
	c.mu.RLock()
	defer c.mu.RUnlock()

	metrics, exists := c.agentMetrics[agentID]
	if !exists {
		return nil
	}

	// Return a copy to avoid race conditions
	copy := *metrics
	return &copy
}

// GetAllAgentMetrics returns metrics for all agents.
func (c *Collector) GetAllAgentMetrics() map[string]*AgentMetrics {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Return deep copy to avoid race conditions
	result := make(map[string]*AgentMetrics, len(c.agentMetrics))
	for id, metrics := range c.agentMetrics {
		copy := *metrics
		result[id] = &copy
	}

	return result
}

// GetSessionMetrics returns session-level aggregate metrics.
func (c *Collector) GetSessionMetrics() SessionMetrics {
	c.mu.RLock()
	defer c.mu.RUnlock()

	totalExecutions := 0
	totalSuccess := 0
	totalFailures := 0

	for _, metrics := range c.agentMetrics {
		totalExecutions += metrics.ExecutionCount
		totalSuccess += metrics.SuccessCount
		totalFailures += metrics.FailureCount
	}

	return SessionMetrics{
		SessionID:       c.sessionID,
		TraceID:         c.traceID,
		StartTime:       c.startTime,
		TotalDurationMS: c.totalDuration.Milliseconds(),
		TotalExecutions: totalExecutions,
		SuccessCount:    totalSuccess,
		FailureCount:    totalFailures,
		TotalTokens:     c.totalTokens,
		TotalCostUSD:    c.totalCost,
		AgentCount:      len(c.agentMetrics),
	}
}

// ExportToContext exports metrics to AgentContext diagnostics.
func (c *Collector) ExportToContext(ctx *models.AgentContext) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if ctx.Diagnostics == nil {
		ctx.Diagnostics = &models.DiagnosticsContext{}
	}

	if ctx.Diagnostics.Performance == nil {
		ctx.Diagnostics.Performance = &models.PerformanceData{}
	}

	// Export total duration
	ctx.Diagnostics.Performance.TotalDurationMS = c.totalDuration.Milliseconds()

	// Export per-agent metrics
	if ctx.Diagnostics.Performance.AgentMetrics == nil {
		ctx.Diagnostics.Performance.AgentMetrics = make(map[string]*models.AgentMetrics)
	}

	for id, metrics := range c.agentMetrics {
		ctx.Diagnostics.Performance.AgentMetrics[id] = &models.AgentMetrics{
			DurationMS: metrics.AvgDurationMS,
			LLMCalls:   metrics.LLMCalls,
			Status:     metrics.LastStatus,
			Tokens:     metrics.TotalTokens,
			Cost:       metrics.TotalCost,
		}
	}
}

// Reset resets all metrics (useful for testing or session restart).
func (c *Collector) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.agentMetrics = make(map[string]*AgentMetrics)
	c.totalDuration = 0
	c.totalCost = 0
	c.totalTokens = 0
	c.startTime = time.Now()
}

// SessionMetrics represents session-level aggregate metrics.
type SessionMetrics struct {
	SessionID       string
	TraceID         string
	StartTime       time.Time
	TotalDurationMS int64
	TotalExecutions int
	SuccessCount    int
	FailureCount    int
	TotalTokens     int
	TotalCostUSD    float64
	AgentCount      int
}

// SuccessRate returns the success rate as a percentage (0-100).
func (m *SessionMetrics) SuccessRate() float64 {
	if m.TotalExecutions == 0 {
		return 0.0
	}
	return float64(m.SuccessCount) / float64(m.TotalExecutions) * 100.0
}

// AvgCostPerExecution returns the average cost per agent execution.
func (m *SessionMetrics) AvgCostPerExecution() float64 {
	if m.TotalExecutions == 0 {
		return 0.0
	}
	return m.TotalCostUSD / float64(m.TotalExecutions)
}

// AvgTokensPerExecution returns the average tokens used per execution.
func (m *SessionMetrics) AvgTokensPerExecution() float64 {
	if m.TotalExecutions == 0 {
		return 0.0
	}
	return float64(m.TotalTokens) / float64(m.TotalExecutions)
}
