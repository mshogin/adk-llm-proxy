package metrics

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// CostReporter generates cost reports for sessions and agents.
//
// Design Principles:
// - Aggregate cost data from multiple collectors
// - Per-agent and per-session breakdowns
// - Budget tracking and alerting
// - Cost trend analysis
type CostReporter struct {
	sessionID string
	traceID   string

	// Cost data by agent
	agentCosts map[string]*AgentCostData
	mu         sync.RWMutex

	// Session-level totals
	totalCost   float64
	budgetLimit float64 // 0 = no limit

	// Timestamps
	startTime time.Time
	endTime   time.Time
}

// AgentCostData tracks cost information for a single agent.
type AgentCostData struct {
	AgentID         string
	TotalCost       float64
	LLMCalls        int
	TotalTokens     int
	PromptTokens    int
	CompletionTokens int
	AvgCostPerCall  float64
	BudgetUsage     float64 // percentage of session budget (0-100)
}

// CostReport represents a complete cost report.
type CostReport struct {
	SessionID      string
	TraceID        string
	TotalCost      float64
	BudgetLimit    float64
	BudgetUsage    float64    // percentage (0-100)
	BudgetRemaining float64
	IsOverBudget   bool
	AgentCosts     []*AgentCostData
	AgentCount     int
	TopCostAgents  []*AgentCostData // Top 5 most expensive agents
	StartTime      time.Time
	EndTime        time.Time
	Duration       time.Duration
}

// CostAlert represents a cost-related alert.
type CostAlert struct {
	Timestamp   time.Time
	Severity    AlertSeverity
	Message     string
	SessionID   string
	AgentID     string // Empty for session-level alerts
	CurrentCost float64
	BudgetLimit float64
}

// AlertSeverity represents the severity level of an alert.
type AlertSeverity string

const (
	AlertInfo     AlertSeverity = "info"
	AlertWarning  AlertSeverity = "warning"
	AlertCritical AlertSeverity = "critical"
)

// NewCostReporter creates a new cost reporter.
func NewCostReporter(sessionID, traceID string, budgetLimit float64) *CostReporter {
	return &CostReporter{
		sessionID:   sessionID,
		traceID:     traceID,
		agentCosts:  make(map[string]*AgentCostData),
		budgetLimit: budgetLimit,
		startTime:   time.Now(),
	}
}

// RecordAgentCost records cost data for an agent.
func (r *CostReporter) RecordAgentCost(
	agentID string,
	cost float64,
	promptTokens, completionTokens int,
) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Get or create agent cost data
	agentCost, exists := r.agentCosts[agentID]
	if !exists {
		agentCost = &AgentCostData{
			AgentID: agentID,
		}
		r.agentCosts[agentID] = agentCost
	}

	// Update agent costs
	agentCost.TotalCost += cost
	agentCost.LLMCalls++
	agentCost.PromptTokens += promptTokens
	agentCost.CompletionTokens += completionTokens
	agentCost.TotalTokens = agentCost.PromptTokens + agentCost.CompletionTokens

	// Calculate average cost per call
	if agentCost.LLMCalls > 0 {
		agentCost.AvgCostPerCall = agentCost.TotalCost / float64(agentCost.LLMCalls)
	}

	// Update session total
	r.totalCost += cost

	// Calculate budget usage for agent
	if r.budgetLimit > 0 {
		agentCost.BudgetUsage = (agentCost.TotalCost / r.budgetLimit) * 100.0
	}
}

// GetAgentCost returns cost data for a specific agent.
func (r *CostReporter) GetAgentCost(agentID string) *AgentCostData {
	r.mu.RLock()
	defer r.mu.RUnlock()

	agentCost, exists := r.agentCosts[agentID]
	if !exists {
		return nil
	}

	// Return a copy
	copy := *agentCost
	return &copy
}

// GetAllAgentCosts returns cost data for all agents.
func (r *CostReporter) GetAllAgentCosts() []*AgentCostData {
	r.mu.RLock()
	defer r.mu.RUnlock()

	costs := make([]*AgentCostData, 0, len(r.agentCosts))
	for _, agentCost := range r.agentCosts {
		copy := *agentCost
		costs = append(costs, &copy)
	}

	return costs
}

// GetCostReport generates a complete cost report.
func (r *CostReporter) GetCostReport() *CostReport {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Calculate budget metrics
	budgetUsage := 0.0
	budgetRemaining := r.budgetLimit
	isOverBudget := false

	if r.budgetLimit > 0 {
		budgetUsage = (r.totalCost / r.budgetLimit) * 100.0
		budgetRemaining = r.budgetLimit - r.totalCost
		isOverBudget = r.totalCost > r.budgetLimit
	}

	// Collect agent costs
	agentCosts := make([]*AgentCostData, 0, len(r.agentCosts))
	for _, agentCost := range r.agentCosts {
		copy := *agentCost
		agentCosts = append(agentCosts, &copy)
	}

	// Find top cost agents (top 5)
	topCostAgents := r.getTopCostAgentsUnsafe(5)

	// Calculate duration
	endTime := r.endTime
	if endTime.IsZero() {
		endTime = time.Now()
	}

	duration := endTime.Sub(r.startTime)

	return &CostReport{
		SessionID:       r.sessionID,
		TraceID:         r.traceID,
		TotalCost:       r.totalCost,
		BudgetLimit:     r.budgetLimit,
		BudgetUsage:     budgetUsage,
		BudgetRemaining: budgetRemaining,
		IsOverBudget:    isOverBudget,
		AgentCosts:      agentCosts,
		AgentCount:      len(agentCosts),
		TopCostAgents:   topCostAgents,
		StartTime:       r.startTime,
		EndTime:         endTime,
		Duration:        duration,
	}
}

// getTopCostAgentsUnsafe returns top N most expensive agents without locking.
func (r *CostReporter) getTopCostAgentsUnsafe(n int) []*AgentCostData {
	// Collect all agents
	agents := make([]*AgentCostData, 0, len(r.agentCosts))
	for _, agentCost := range r.agentCosts {
		copy := *agentCost
		agents = append(agents, &copy)
	}

	// Sort by total cost (descending) using simple bubble sort for small N
	for i := 0; i < len(agents); i++ {
		for j := i + 1; j < len(agents); j++ {
			if agents[j].TotalCost > agents[i].TotalCost {
				agents[i], agents[j] = agents[j], agents[i]
			}
		}
	}

	// Return top N
	if len(agents) > n {
		return agents[:n]
	}

	return agents
}

// CheckBudgetAlerts checks for budget-related alerts.
func (r *CostReporter) CheckBudgetAlerts() []*CostAlert {
	r.mu.RLock()
	defer r.mu.RUnlock()

	alerts := []*CostAlert{}

	if r.budgetLimit == 0 {
		return alerts // No budget limit set
	}

	// Check session-level budget
	budgetUsage := (r.totalCost / r.budgetLimit) * 100.0

	if budgetUsage >= 100.0 {
		alerts = append(alerts, &CostAlert{
			Timestamp:   time.Now(),
			Severity:    AlertCritical,
			Message:     fmt.Sprintf("Budget exceeded: $%.4f / $%.4f (%.1f%%)", r.totalCost, r.budgetLimit, budgetUsage),
			SessionID:   r.sessionID,
			CurrentCost: r.totalCost,
			BudgetLimit: r.budgetLimit,
		})
	} else if budgetUsage >= 80.0 {
		alerts = append(alerts, &CostAlert{
			Timestamp:   time.Now(),
			Severity:    AlertWarning,
			Message:     fmt.Sprintf("Budget warning: $%.4f / $%.4f (%.1f%%)", r.totalCost, r.budgetLimit, budgetUsage),
			SessionID:   r.sessionID,
			CurrentCost: r.totalCost,
			BudgetLimit: r.budgetLimit,
		})
	} else if budgetUsage >= 50.0 {
		alerts = append(alerts, &CostAlert{
			Timestamp:   time.Now(),
			Severity:    AlertInfo,
			Message:     fmt.Sprintf("Budget info: $%.4f / $%.4f (%.1f%%)", r.totalCost, r.budgetLimit, budgetUsage),
			SessionID:   r.sessionID,
			CurrentCost: r.totalCost,
			BudgetLimit: r.budgetLimit,
		})
	}

	return alerts
}

// Finalize marks the cost report as complete.
func (r *CostReporter) Finalize() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.endTime = time.Now()
}

// Reset resets the cost reporter (useful for testing).
func (r *CostReporter) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.agentCosts = make(map[string]*AgentCostData)
	r.totalCost = 0.0
	r.startTime = time.Now()
	r.endTime = time.Time{}
}

// FormatCostReport formats a cost report as a human-readable string.
func FormatCostReport(report *CostReport) string {
	var builder strings.Builder

	builder.WriteString(fmt.Sprintf("=== Cost Report ===\n"))
	builder.WriteString(fmt.Sprintf("Session ID: %s\n", report.SessionID))
	builder.WriteString(fmt.Sprintf("Trace ID: %s\n", report.TraceID))
	builder.WriteString(fmt.Sprintf("Duration: %v\n", report.Duration))
	builder.WriteString(fmt.Sprintf("\n"))

	builder.WriteString(fmt.Sprintf("Total Cost: $%.4f\n", report.TotalCost))
	if report.BudgetLimit > 0 {
		builder.WriteString(fmt.Sprintf("Budget Limit: $%.4f\n", report.BudgetLimit))
		builder.WriteString(fmt.Sprintf("Budget Usage: %.1f%%\n", report.BudgetUsage))
		builder.WriteString(fmt.Sprintf("Budget Remaining: $%.4f\n", report.BudgetRemaining))

		if report.IsOverBudget {
			builder.WriteString(fmt.Sprintf("⚠️  OVER BUDGET\n"))
		}
	}
	builder.WriteString(fmt.Sprintf("\n"))

	builder.WriteString(fmt.Sprintf("Agent Count: %d\n", report.AgentCount))
	builder.WriteString(fmt.Sprintf("\n"))

	if len(report.TopCostAgents) > 0 {
		builder.WriteString(fmt.Sprintf("Top Cost Agents:\n"))
		for i, agent := range report.TopCostAgents {
			builder.WriteString(fmt.Sprintf("%d. %s: $%.4f (%d calls, %d tokens)\n",
				i+1, agent.AgentID, agent.TotalCost, agent.LLMCalls, agent.TotalTokens))
		}
		builder.WriteString(fmt.Sprintf("\n"))
	}

	builder.WriteString(fmt.Sprintf("==================\n"))

	return builder.String()
}
