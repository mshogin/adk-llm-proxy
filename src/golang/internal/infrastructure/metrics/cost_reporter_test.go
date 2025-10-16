package metrics

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCostReporter(t *testing.T) {
	reporter := NewCostReporter("session123", "trace456", 10.0)

	assert.Equal(t, "session123", reporter.sessionID)
	assert.Equal(t, "trace456", reporter.traceID)
	assert.Equal(t, 10.0, reporter.budgetLimit)
	assert.NotNil(t, reporter.agentCosts)
	assert.Zero(t, reporter.totalCost)
	assert.NotZero(t, reporter.startTime)
	assert.Zero(t, reporter.endTime)
}

func TestCostReporter_RecordAgentCost_SingleAgent(t *testing.T) {
	reporter := NewCostReporter("session123", "trace456", 0.0)

	reporter.RecordAgentCost("intent_detection", 0.005, 100, 50)

	agentCost := reporter.GetAgentCost("intent_detection")
	require.NotNil(t, agentCost)

	assert.Equal(t, "intent_detection", agentCost.AgentID)
	assert.Equal(t, 0.005, agentCost.TotalCost)
	assert.Equal(t, 1, agentCost.LLMCalls)
	assert.Equal(t, 100, agentCost.PromptTokens)
	assert.Equal(t, 50, agentCost.CompletionTokens)
	assert.Equal(t, 150, agentCost.TotalTokens)
	assert.Equal(t, 0.005, agentCost.AvgCostPerCall)
}

func TestCostReporter_RecordAgentCost_MultipleCallsSameAgent(t *testing.T) {
	reporter := NewCostReporter("session123", "trace456", 0.0)

	reporter.RecordAgentCost("inference", 0.010, 200, 100)
	reporter.RecordAgentCost("inference", 0.015, 300, 150)
	reporter.RecordAgentCost("inference", 0.020, 400, 200)

	agentCost := reporter.GetAgentCost("inference")
	require.NotNil(t, agentCost)

	assert.Equal(t, "inference", agentCost.AgentID)
	assert.Equal(t, 0.045, agentCost.TotalCost) // 0.010 + 0.015 + 0.020
	assert.Equal(t, 3, agentCost.LLMCalls)
	assert.Equal(t, 900, agentCost.PromptTokens)    // 200 + 300 + 400
	assert.Equal(t, 450, agentCost.CompletionTokens) // 100 + 150 + 200
	assert.Equal(t, 1350, agentCost.TotalTokens)     // 900 + 450
	assert.InDelta(t, 0.015, agentCost.AvgCostPerCall, 0.0001) // 0.045 / 3
}

func TestCostReporter_RecordAgentCost_MultipleAgents(t *testing.T) {
	reporter := NewCostReporter("session123", "trace456", 0.0)

	reporter.RecordAgentCost("intent_detection", 0.005, 100, 50)
	reporter.RecordAgentCost("inference", 0.010, 200, 100)
	reporter.RecordAgentCost("validation", 0.003, 50, 25)

	assert.Equal(t, 0.018, reporter.totalCost) // 0.005 + 0.010 + 0.003

	allCosts := reporter.GetAllAgentCosts()
	assert.Len(t, allCosts, 3)
}

func TestCostReporter_GetAgentCost_NonExistent(t *testing.T) {
	reporter := NewCostReporter("session123", "trace456", 0.0)

	agentCost := reporter.GetAgentCost("nonexistent")
	assert.Nil(t, agentCost)
}

func TestCostReporter_GetAllAgentCosts(t *testing.T) {
	reporter := NewCostReporter("session123", "trace456", 0.0)

	reporter.RecordAgentCost("intent_detection", 0.005, 100, 50)
	reporter.RecordAgentCost("inference", 0.010, 200, 100)
	reporter.RecordAgentCost("validation", 0.003, 50, 25)

	allCosts := reporter.GetAllAgentCosts()
	assert.Len(t, allCosts, 3)

	// Verify all agents present
	agentIDs := make(map[string]bool)
	for _, cost := range allCosts {
		agentIDs[cost.AgentID] = true
	}

	assert.True(t, agentIDs["intent_detection"])
	assert.True(t, agentIDs["inference"])
	assert.True(t, agentIDs["validation"])
}

func TestCostReporter_GetCostReport_NoBudget(t *testing.T) {
	reporter := NewCostReporter("session123", "trace456", 0.0)

	reporter.RecordAgentCost("intent_detection", 0.005, 100, 50)
	reporter.RecordAgentCost("inference", 0.010, 200, 100)

	time.Sleep(10 * time.Millisecond)
	reporter.Finalize()

	report := reporter.GetCostReport()

	assert.Equal(t, "session123", report.SessionID)
	assert.Equal(t, "trace456", report.TraceID)
	assert.Equal(t, 0.015, report.TotalCost)
	assert.Equal(t, 0.0, report.BudgetLimit)
	assert.Equal(t, 0.0, report.BudgetUsage)
	assert.Equal(t, 0.0, report.BudgetRemaining)
	assert.False(t, report.IsOverBudget)
	assert.Equal(t, 2, report.AgentCount)
	assert.Len(t, report.AgentCosts, 2)
	assert.Greater(t, report.Duration, time.Duration(0))
	assert.GreaterOrEqual(t, report.Duration.Milliseconds(), int64(10))
}

func TestCostReporter_GetCostReport_WithBudget_UnderLimit(t *testing.T) {
	reporter := NewCostReporter("session123", "trace456", 1.0)

	reporter.RecordAgentCost("intent_detection", 0.30, 1000, 500)
	reporter.RecordAgentCost("inference", 0.20, 800, 400)

	report := reporter.GetCostReport()

	assert.Equal(t, 0.50, report.TotalCost)
	assert.Equal(t, 1.0, report.BudgetLimit)
	assert.Equal(t, 50.0, report.BudgetUsage) // 0.50 / 1.0 * 100
	assert.Equal(t, 0.50, report.BudgetRemaining)
	assert.False(t, report.IsOverBudget)
}

func TestCostReporter_GetCostReport_WithBudget_OverLimit(t *testing.T) {
	reporter := NewCostReporter("session123", "trace456", 1.0)

	reporter.RecordAgentCost("intent_detection", 0.60, 2000, 1000)
	reporter.RecordAgentCost("inference", 0.50, 1500, 750)

	report := reporter.GetCostReport()

	assert.InDelta(t, 1.10, report.TotalCost, 0.0001)
	assert.Equal(t, 1.0, report.BudgetLimit)
	assert.InDelta(t, 110.0, report.BudgetUsage, 0.01) // 1.10 / 1.0 * 100
	assert.InDelta(t, -0.10, report.BudgetRemaining, 0.0001)
	assert.True(t, report.IsOverBudget)
}

func TestCostReporter_GetCostReport_TopCostAgents(t *testing.T) {
	reporter := NewCostReporter("session123", "trace456", 0.0)

	// Add 7 agents with different costs
	reporter.RecordAgentCost("agent1", 0.100, 1000, 500)
	reporter.RecordAgentCost("agent2", 0.050, 500, 250)
	reporter.RecordAgentCost("agent3", 0.200, 2000, 1000)
	reporter.RecordAgentCost("agent4", 0.030, 300, 150)
	reporter.RecordAgentCost("agent5", 0.150, 1500, 750)
	reporter.RecordAgentCost("agent6", 0.080, 800, 400)
	reporter.RecordAgentCost("agent7", 0.010, 100, 50)

	report := reporter.GetCostReport()

	assert.Len(t, report.TopCostAgents, 5) // Top 5

	// Verify ordering (descending by cost)
	assert.Equal(t, "agent3", report.TopCostAgents[0].AgentID) // 0.200
	assert.Equal(t, "agent5", report.TopCostAgents[1].AgentID) // 0.150
	assert.Equal(t, "agent1", report.TopCostAgents[2].AgentID) // 0.100
	assert.Equal(t, "agent6", report.TopCostAgents[3].AgentID) // 0.080
	assert.Equal(t, "agent2", report.TopCostAgents[4].AgentID) // 0.050
}

func TestCostReporter_GetCostReport_TopCostAgents_LessThan5(t *testing.T) {
	reporter := NewCostReporter("session123", "trace456", 0.0)

	reporter.RecordAgentCost("agent1", 0.100, 1000, 500)
	reporter.RecordAgentCost("agent2", 0.050, 500, 250)

	report := reporter.GetCostReport()

	assert.Len(t, report.TopCostAgents, 2)
}

func TestCostReporter_CheckBudgetAlerts_NoBudget(t *testing.T) {
	reporter := NewCostReporter("session123", "trace456", 0.0)

	reporter.RecordAgentCost("intent_detection", 0.50, 1000, 500)

	alerts := reporter.CheckBudgetAlerts()

	assert.Empty(t, alerts)
}

func TestCostReporter_CheckBudgetAlerts_InfoLevel(t *testing.T) {
	reporter := NewCostReporter("session123", "trace456", 1.0)

	reporter.RecordAgentCost("intent_detection", 0.55, 1000, 500)

	alerts := reporter.CheckBudgetAlerts()

	require.Len(t, alerts, 1)
	assert.Equal(t, AlertInfo, alerts[0].Severity)
	assert.Equal(t, "session123", alerts[0].SessionID)
	assert.Equal(t, 0.55, alerts[0].CurrentCost)
	assert.Equal(t, 1.0, alerts[0].BudgetLimit)
	assert.Contains(t, alerts[0].Message, "55.0%")
}

func TestCostReporter_CheckBudgetAlerts_WarningLevel(t *testing.T) {
	reporter := NewCostReporter("session123", "trace456", 1.0)

	reporter.RecordAgentCost("intent_detection", 0.85, 1000, 500)

	alerts := reporter.CheckBudgetAlerts()

	require.Len(t, alerts, 1)
	assert.Equal(t, AlertWarning, alerts[0].Severity)
	assert.Contains(t, alerts[0].Message, "85.0%")
}

func TestCostReporter_CheckBudgetAlerts_CriticalLevel(t *testing.T) {
	reporter := NewCostReporter("session123", "trace456", 1.0)

	reporter.RecordAgentCost("intent_detection", 1.10, 1000, 500)

	alerts := reporter.CheckBudgetAlerts()

	require.Len(t, alerts, 1)
	assert.Equal(t, AlertCritical, alerts[0].Severity)
	assert.Contains(t, alerts[0].Message, "Budget exceeded")
	assert.Contains(t, alerts[0].Message, "110.0%")
}

func TestCostReporter_BudgetUsagePerAgent(t *testing.T) {
	reporter := NewCostReporter("session123", "trace456", 1.0)

	reporter.RecordAgentCost("intent_detection", 0.30, 1000, 500)
	reporter.RecordAgentCost("inference", 0.50, 1500, 750)

	intentCost := reporter.GetAgentCost("intent_detection")
	inferenceCost := reporter.GetAgentCost("inference")

	assert.Equal(t, 30.0, intentCost.BudgetUsage)   // 0.30 / 1.0 * 100
	assert.Equal(t, 50.0, inferenceCost.BudgetUsage) // 0.50 / 1.0 * 100
}

func TestCostReporter_Finalize(t *testing.T) {
	reporter := NewCostReporter("session123", "trace456", 0.0)

	assert.Zero(t, reporter.endTime)

	reporter.Finalize()

	assert.NotZero(t, reporter.endTime)
}

func TestCostReporter_Reset(t *testing.T) {
	reporter := NewCostReporter("session123", "trace456", 1.0)

	reporter.RecordAgentCost("intent_detection", 0.30, 1000, 500)
	reporter.RecordAgentCost("inference", 0.50, 1500, 750)
	reporter.Finalize()

	assert.Equal(t, 0.80, reporter.totalCost)
	assert.Len(t, reporter.agentCosts, 2)
	assert.NotZero(t, reporter.endTime)

	reporter.Reset()

	assert.Zero(t, reporter.totalCost)
	assert.Empty(t, reporter.agentCosts)
	assert.Zero(t, reporter.endTime)
	assert.NotZero(t, reporter.startTime)
}

func TestCostReporter_ThreadSafety(t *testing.T) {
	reporter := NewCostReporter("session123", "trace456", 10.0)

	done := make(chan bool)
	numGoroutines := 10
	recordsPerGoroutine := 100

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			for j := 0; j < recordsPerGoroutine; j++ {
				reporter.RecordAgentCost("agent1", 0.001, 10, 5)
				reporter.RecordAgentCost("agent2", 0.002, 20, 10)
				reporter.GetAgentCost("agent1")
				reporter.GetAllAgentCosts()
				reporter.CheckBudgetAlerts()
			}
			done <- true
		}(i)
	}

	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	report := reporter.GetCostReport()
	assert.Equal(t, 2, report.AgentCount)

	agent1Cost := reporter.GetAgentCost("agent1")
	agent2Cost := reporter.GetAgentCost("agent2")

	assert.Equal(t, numGoroutines*recordsPerGoroutine, agent1Cost.LLMCalls)
	assert.Equal(t, numGoroutines*recordsPerGoroutine, agent2Cost.LLMCalls)
	assert.InDelta(t, 1.0, agent1Cost.TotalCost, 0.01) // 0.001 * 1000
	assert.InDelta(t, 2.0, agent2Cost.TotalCost, 0.01) // 0.002 * 1000
}

func TestCostReporter_GetAgentCost_Copy(t *testing.T) {
	reporter := NewCostReporter("session123", "trace456", 0.0)

	reporter.RecordAgentCost("intent_detection", 0.005, 100, 50)

	cost1 := reporter.GetAgentCost("intent_detection")
	cost1.TotalCost = 999.99 // Modify copy

	cost2 := reporter.GetAgentCost("intent_detection")
	assert.Equal(t, 0.005, cost2.TotalCost) // Original unchanged
}

func TestCostReporter_GetAllAgentCosts_Copy(t *testing.T) {
	reporter := NewCostReporter("session123", "trace456", 0.0)

	reporter.RecordAgentCost("intent_detection", 0.005, 100, 50)

	costs1 := reporter.GetAllAgentCosts()
	costs1[0].TotalCost = 999.99 // Modify copy

	costs2 := reporter.GetAllAgentCosts()
	assert.Equal(t, 0.005, costs2[0].TotalCost) // Original unchanged
}

func TestFormatCostReport(t *testing.T) {
	reporter := NewCostReporter("session123", "trace456", 10.0)

	reporter.RecordAgentCost("intent_detection", 5.50, 10000, 5000)
	reporter.RecordAgentCost("inference", 3.00, 6000, 3000)
	reporter.RecordAgentCost("validation", 0.50, 1000, 500)

	time.Sleep(10 * time.Millisecond)
	reporter.Finalize()

	report := reporter.GetCostReport()
	formatted := FormatCostReport(report)

	assert.Contains(t, formatted, "=== Cost Report ===")
	assert.Contains(t, formatted, "Session ID: session123")
	assert.Contains(t, formatted, "Trace ID: trace456")
	assert.Contains(t, formatted, "Total Cost: $9.0000")
	assert.Contains(t, formatted, "Budget Limit: $10.0000")
	assert.Contains(t, formatted, "Budget Usage: 90.0%")
	assert.Contains(t, formatted, "Budget Remaining: $1.0000")
	assert.Contains(t, formatted, "Agent Count: 3")
	assert.Contains(t, formatted, "Top Cost Agents:")
	assert.Contains(t, formatted, "1. intent_detection: $5.5000")
	assert.Contains(t, formatted, "2. inference: $3.0000")
	assert.Contains(t, formatted, "3. validation: $0.5000")
	assert.NotContains(t, formatted, "OVER BUDGET") // Under budget
}

func TestFormatCostReport_OverBudget(t *testing.T) {
	reporter := NewCostReporter("session123", "trace456", 5.0)

	reporter.RecordAgentCost("intent_detection", 6.00, 10000, 5000)

	report := reporter.GetCostReport()
	formatted := FormatCostReport(report)

	assert.Contains(t, formatted, "Total Cost: $6.0000")
	assert.Contains(t, formatted, "Budget Limit: $5.0000")
	assert.Contains(t, formatted, "⚠️  OVER BUDGET")
}
