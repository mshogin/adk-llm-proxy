package metrics

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAlertManager(t *testing.T) {
	costReporter := NewCostReporter("s1", "t1", 10.0)
	profiler := NewPerformanceProfiler("s1", "t1")

	config := AlertConfig{
		BudgetWarningThreshold: 75.0,
		BudgetCriticalThreshold: 95.0,
	}

	am := NewAlertManager("s1", "t1", config, costReporter, profiler)

	assert.Equal(t, "s1", am.sessionID)
	assert.Equal(t, "t1", am.traceID)
	assert.Equal(t, 75.0, am.config.BudgetWarningThreshold)
	assert.Equal(t, 95.0, am.config.BudgetCriticalThreshold)
}

func TestNewDefaultAlertManager(t *testing.T) {
	costReporter := NewCostReporter("s1", "t1", 10.0)
	profiler := NewPerformanceProfiler("s1", "t1")

	am := NewDefaultAlertManager("s1", "t1", costReporter, profiler)

	assert.NotNil(t, am)
	assert.Equal(t, 50.0, am.config.BudgetInfoThreshold)
	assert.Equal(t, 80.0, am.config.BudgetWarningThreshold)
	assert.Equal(t, 100.0, am.config.BudgetCriticalThreshold)
}

func TestAlertManager_CheckBudgetAlerts_Info(t *testing.T) {
	costReporter := NewCostReporter("s1", "t1", 10.0)
	profiler := NewPerformanceProfiler("s1", "t1")

	costReporter.RecordAgentCost("agent1", 5.5, 1000, 500) // 55% of budget

	am := NewDefaultAlertManager("s1", "t1", costReporter, profiler)
	alerts := am.CheckAlerts()

	require.Len(t, alerts, 1)
	assert.Equal(t, AlertInfo, alerts[0].Severity)
	assert.Equal(t, AlertTypeBudgetOverrun, alerts[0].Type)
	assert.Contains(t, alerts[0].Message, "55.0%")
}

func TestAlertManager_CheckBudgetAlerts_Warning(t *testing.T) {
	costReporter := NewCostReporter("s1", "t1", 10.0)
	profiler := NewPerformanceProfiler("s1", "t1")

	costReporter.RecordAgentCost("agent1", 8.5, 1000, 500) // 85% of budget

	am := NewDefaultAlertManager("s1", "t1", costReporter, profiler)
	alerts := am.CheckAlerts()

	require.Len(t, alerts, 1)
	assert.Equal(t, AlertWarning, alerts[0].Severity)
	assert.Equal(t, AlertTypeBudgetOverrun, alerts[0].Type)
	assert.Contains(t, alerts[0].Message, "85.0%")
}

func TestAlertManager_CheckBudgetAlerts_Critical(t *testing.T) {
	costReporter := NewCostReporter("s1", "t1", 10.0)
	profiler := NewPerformanceProfiler("s1", "t1")

	costReporter.RecordAgentCost("agent1", 11.0, 1000, 500) // 110% of budget

	am := NewDefaultAlertManager("s1", "t1", costReporter, profiler)
	alerts := am.CheckAlerts()

	require.Len(t, alerts, 1)
	assert.Equal(t, AlertCritical, alerts[0].Severity)
	assert.Equal(t, AlertTypeBudgetOverrun, alerts[0].Type)
	assert.Contains(t, alerts[0].Message, "110.0%")
}

func TestAlertManager_CheckBudgetAlerts_NoBudgetLimit(t *testing.T) {
	costReporter := NewCostReporter("s1", "t1", 0.0) // No budget limit
	profiler := NewPerformanceProfiler("s1", "t1")

	costReporter.RecordAgentCost("agent1", 100.0, 1000, 500)

	am := NewDefaultAlertManager("s1", "t1", costReporter, profiler)
	alerts := am.CheckAlerts()

	assert.Empty(t, alerts) // No alerts when no budget limit
}

func TestAlertManager_CheckSLAViolations_SessionLevel_Warning(t *testing.T) {
	costReporter := NewCostReporter("s1", "t1", 10.0)
	profiler := NewPerformanceProfiler("s1", "t1")

	// Multiple short executions to exceed session threshold but not agent threshold
	for i := 0; i < 15; i++ {
		profiler.RecordProfile("agent1", ProfileSnapshot{DurationMS: 800}) // 15 * 800ms = 12 seconds total
	}

	am := NewDefaultAlertManager("s1", "t1", costReporter, profiler)
	alerts := am.CheckAlerts()

	// Find session-level SLA alert
	var sessionAlert *Alert
	for _, alert := range alerts {
		if alert.Type == AlertTypeSLAViolation && alert.AgentID == "" {
			sessionAlert = alert
			break
		}
	}

	require.NotNil(t, sessionAlert)
	assert.Equal(t, AlertWarning, sessionAlert.Severity)
	assert.Contains(t, sessionAlert.Message, "Session duration")
	assert.Contains(t, sessionAlert.Message, "12000ms")
}

func TestAlertManager_CheckSLAViolations_SessionLevel_Critical(t *testing.T) {
	costReporter := NewCostReporter("s1", "t1", 10.0)
	profiler := NewPerformanceProfiler("s1", "t1")

	// Multiple short executions to exceed session critical threshold but not agent threshold
	for i := 0; i < 40; i++ {
		profiler.RecordProfile("agent1", ProfileSnapshot{DurationMS: 900}) // 40 * 900ms = 36 seconds total
	}

	am := NewDefaultAlertManager("s1", "t1", costReporter, profiler)
	alerts := am.CheckAlerts()

	// Find session-level SLA alert
	var sessionAlert *Alert
	for _, alert := range alerts {
		if alert.Type == AlertTypeSLAViolation && alert.AgentID == "" {
			sessionAlert = alert
			break
		}
	}

	require.NotNil(t, sessionAlert)
	assert.Equal(t, AlertCritical, sessionAlert.Severity)
	assert.Contains(t, sessionAlert.Message, "36000ms")
}

func TestAlertManager_CheckSLAViolations_AgentLevel_Warning(t *testing.T) {
	costReporter := NewCostReporter("s1", "t1", 10.0)
	profiler := NewPerformanceProfiler("s1", "t1")

	profiler.RecordProfile("slow_agent", ProfileSnapshot{DurationMS: 1500}) // 1.5 seconds

	am := NewDefaultAlertManager("s1", "t1", costReporter, profiler)
	alerts := am.CheckAlerts()

	require.Len(t, alerts, 1)
	assert.Equal(t, AlertWarning, alerts[0].Severity)
	assert.Equal(t, AlertTypeSLAViolation, alerts[0].Type)
	assert.Equal(t, "slow_agent", alerts[0].AgentID)
	assert.Contains(t, alerts[0].Message, "slow_agent")
	assert.Contains(t, alerts[0].Message, "1500ms")
}

func TestAlertManager_CheckSLAViolations_AgentLevel_Critical(t *testing.T) {
	costReporter := NewCostReporter("s1", "t1", 10.0)
	profiler := NewPerformanceProfiler("s1", "t1")

	profiler.RecordProfile("slow_agent", ProfileSnapshot{DurationMS: 6000}) // 6 seconds

	am := NewDefaultAlertManager("s1", "t1", costReporter, profiler)
	alerts := am.CheckAlerts()

	require.Len(t, alerts, 1)
	assert.Equal(t, AlertCritical, alerts[0].Severity)
	assert.Equal(t, AlertTypeSLAViolation, alerts[0].Type)
	assert.Equal(t, "slow_agent", alerts[0].AgentID)
}

func TestAlertManager_CheckErrorRateAlerts_Warning(t *testing.T) {
	costReporter := NewCostReporter("s1", "t1", 10.0)
	profiler := NewPerformanceProfiler("s1", "t1")

	// 15% error rate (15 errors out of 100 executions) with short duration
	for i := 0; i < 85; i++ {
		profiler.RecordProfile("agent1", ProfileSnapshot{DurationMS: 10})
	}
	for i := 0; i < 15; i++ {
		profiler.RecordProfile("agent1", ProfileSnapshot{
			DurationMS: 10,
			Error:      assert.AnError,
		})
	}

	am := NewDefaultAlertManager("s1", "t1", costReporter, profiler)
	alerts := am.CheckAlerts()

	// Find error rate alert
	var errorRateAlert *Alert
	for _, alert := range alerts {
		if alert.Type == AlertTypeErrorRate {
			errorRateAlert = alert
			break
		}
	}

	require.NotNil(t, errorRateAlert)
	assert.Equal(t, AlertWarning, errorRateAlert.Severity)
	assert.Equal(t, "agent1", errorRateAlert.AgentID)
	assert.Contains(t, errorRateAlert.Message, "15.0%")
}

func TestAlertManager_CheckErrorRateAlerts_Critical(t *testing.T) {
	costReporter := NewCostReporter("s1", "t1", 10.0)
	profiler := NewPerformanceProfiler("s1", "t1")

	// 30% error rate (30 errors out of 100 executions) with short duration
	for i := 0; i < 70; i++ {
		profiler.RecordProfile("agent1", ProfileSnapshot{DurationMS: 10})
	}
	for i := 0; i < 30; i++ {
		profiler.RecordProfile("agent1", ProfileSnapshot{
			DurationMS: 10,
			Error:      assert.AnError,
		})
	}

	am := NewDefaultAlertManager("s1", "t1", costReporter, profiler)
	alerts := am.CheckAlerts()

	// Find error rate alert
	var errorRateAlert *Alert
	for _, alert := range alerts {
		if alert.Type == AlertTypeErrorRate {
			errorRateAlert = alert
			break
		}
	}

	require.NotNil(t, errorRateAlert)
	assert.Equal(t, AlertCritical, errorRateAlert.Severity)
	assert.Equal(t, "agent1", errorRateAlert.AgentID)
	assert.Contains(t, errorRateAlert.Message, "30.0%")
}

func TestAlertManager_CheckMemoryAlerts_Warning(t *testing.T) {
	costReporter := NewCostReporter("s1", "t1", 10.0)
	profiler := NewPerformanceProfiler("s1", "t1")

	// 150MB average memory
	profiler.RecordProfile("memory_hog", ProfileSnapshot{
		DurationMS:       100,
		MemoryAllocBytes: 150 * 1024 * 1024,
	})

	am := NewDefaultAlertManager("s1", "t1", costReporter, profiler)
	alerts := am.CheckAlerts()

	require.Len(t, alerts, 1)
	assert.Equal(t, AlertWarning, alerts[0].Severity)
	assert.Equal(t, AlertTypeMemoryExhaustion, alerts[0].Type)
	assert.Equal(t, "memory_hog", alerts[0].AgentID)
	assert.Contains(t, alerts[0].Message, "150MB")
}

func TestAlertManager_CheckMemoryAlerts_Critical(t *testing.T) {
	costReporter := NewCostReporter("s1", "t1", 10.0)
	profiler := NewPerformanceProfiler("s1", "t1")

	// 600MB average memory
	profiler.RecordProfile("memory_hog", ProfileSnapshot{
		DurationMS:       100,
		MemoryAllocBytes: 600 * 1024 * 1024,
	})

	am := NewDefaultAlertManager("s1", "t1", costReporter, profiler)
	alerts := am.CheckAlerts()

	require.Len(t, alerts, 1)
	assert.Equal(t, AlertCritical, alerts[0].Severity)
	assert.Equal(t, AlertTypeMemoryExhaustion, alerts[0].Type)
	assert.Equal(t, "memory_hog", alerts[0].AgentID)
	assert.Contains(t, alerts[0].Message, "600MB")
}

func TestAlertManager_MultipleAlerts(t *testing.T) {
	costReporter := NewCostReporter("s1", "t1", 10.0)
	profiler := NewPerformanceProfiler("s1", "t1")

	// Trigger budget alert (85%)
	costReporter.RecordAgentCost("agent1", 8.5, 1000, 500)

	// Trigger SLA violation (6 seconds)
	profiler.RecordProfile("slow_agent", ProfileSnapshot{DurationMS: 6000})

	// Trigger error rate alert (30%)
	for i := 0; i < 70; i++ {
		profiler.RecordProfile("error_agent", ProfileSnapshot{DurationMS: 100})
	}
	for i := 0; i < 30; i++ {
		profiler.RecordProfile("error_agent", ProfileSnapshot{
			DurationMS: 100,
			Error:      assert.AnError,
		})
	}

	am := NewDefaultAlertManager("s1", "t1", costReporter, profiler)
	alerts := am.CheckAlerts()

	assert.GreaterOrEqual(t, len(alerts), 3) // At least 3 alerts

	// Verify we have alerts of different types
	alertTypes := make(map[AlertType]bool)
	for _, alert := range alerts {
		alertTypes[alert.Type] = true
	}

	assert.True(t, alertTypes[AlertTypeBudgetOverrun])
	assert.True(t, alertTypes[AlertTypeSLAViolation])
	assert.True(t, alertTypes[AlertTypeErrorRate])
}

func TestAlertManager_Deduplication(t *testing.T) {
	costReporter := NewCostReporter("s1", "t1", 10.0)
	profiler := NewPerformanceProfiler("s1", "t1")

	costReporter.RecordAgentCost("agent1", 8.5, 1000, 500) // 85% of budget

	config := AlertConfig{
		BudgetWarningThreshold:     80.0,
		DeduplicationWindowSeconds: 1, // 1 second for faster testing
	}

	am := NewAlertManager("s1", "t1", config, costReporter, profiler)

	// First check - should fire alert
	alerts1 := am.CheckAlerts()
	assert.Len(t, alerts1, 1)
	assert.Equal(t, AlertTypeBudgetOverrun, alerts1[0].Type)

	// Second check immediately - should not fire duplicate
	alerts2 := am.CheckAlerts()
	assert.Empty(t, alerts2) // No new alerts due to deduplication

	// Wait for deduplication window to expire
	time.Sleep(1100 * time.Millisecond)

	// Third check - should fire again after window expires
	alerts3 := am.CheckAlerts()

	// Find budget alert in alerts3
	var budgetAlert *Alert
	for _, alert := range alerts3 {
		if alert.Type == AlertTypeBudgetOverrun {
			budgetAlert = alert
			break
		}
	}

	assert.NotNil(t, budgetAlert) // Should have fired again
}

func TestAlertManager_GetActiveAlerts(t *testing.T) {
	costReporter := NewCostReporter("s1", "t1", 10.0)
	profiler := NewPerformanceProfiler("s1", "t1")

	costReporter.RecordAgentCost("agent1", 8.5, 1000, 500)

	am := NewDefaultAlertManager("s1", "t1", costReporter, profiler)
	am.CheckAlerts()

	activeAlerts := am.GetActiveAlerts()
	assert.Len(t, activeAlerts, 1)
	assert.False(t, activeAlerts[0].Resolved)
}

func TestAlertManager_GetAlertHistory(t *testing.T) {
	costReporter := NewCostReporter("s1", "t1", 10.0)
	profiler := NewPerformanceProfiler("s1", "t1")

	costReporter.RecordAgentCost("agent1", 8.5, 1000, 500)

	am := NewDefaultAlertManager("s1", "t1", costReporter, profiler)
	am.CheckAlerts()

	history := am.GetAlertHistory()
	assert.Len(t, history, 1)
}

func TestAlertManager_ResolveAlert(t *testing.T) {
	costReporter := NewCostReporter("s1", "t1", 10.0)
	profiler := NewPerformanceProfiler("s1", "t1")

	costReporter.RecordAgentCost("agent1", 8.5, 1000, 500)

	am := NewDefaultAlertManager("s1", "t1", costReporter, profiler)
	alerts := am.CheckAlerts()

	require.Len(t, alerts, 1)
	alertID := alerts[0].ID

	am.ResolveAlert(alertID)

	activeAlerts := am.GetActiveAlerts()
	assert.Empty(t, activeAlerts) // Alert should be resolved
}

func TestAlertManager_ClearResolvedAlerts(t *testing.T) {
	costReporter := NewCostReporter("s1", "t1", 10.0)
	profiler := NewPerformanceProfiler("s1", "t1")

	costReporter.RecordAgentCost("agent1", 8.5, 1000, 500)

	am := NewDefaultAlertManager("s1", "t1", costReporter, profiler)
	alerts := am.CheckAlerts()

	require.Len(t, alerts, 1)
	alertID := alerts[0].ID

	am.ResolveAlert(alertID)
	am.ClearResolvedAlerts()

	// Alert should be removed from active list but remain in history
	activeAlerts := am.GetActiveAlerts()
	assert.Empty(t, activeAlerts)

	history := am.GetAlertHistory()
	assert.Len(t, history, 1)
}

func TestAlertManager_Reset(t *testing.T) {
	costReporter := NewCostReporter("s1", "t1", 10.0)
	profiler := NewPerformanceProfiler("s1", "t1")

	costReporter.RecordAgentCost("agent1", 8.5, 1000, 500)

	am := NewDefaultAlertManager("s1", "t1", costReporter, profiler)
	am.CheckAlerts()

	assert.NotEmpty(t, am.alerts)
	assert.NotEmpty(t, am.alertHistory)

	am.Reset()

	assert.Empty(t, am.alerts)
	assert.Empty(t, am.alertHistory)
	assert.Empty(t, am.firedAlerts)
}

func TestAlertManager_ThreadSafety(t *testing.T) {
	costReporter := NewCostReporter("s1", "t1", 10.0)
	profiler := NewPerformanceProfiler("s1", "t1")

	am := NewDefaultAlertManager("s1", "t1", costReporter, profiler)

	done := make(chan bool)
	numGoroutines := 10
	checksPerGoroutine := 100

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			for j := 0; j < checksPerGoroutine; j++ {
				// Simulate varying conditions
				costReporter.RecordAgentCost("agent1", 0.1, 100, 50)
				profiler.RecordProfile("agent1", ProfileSnapshot{DurationMS: 100})

				am.CheckAlerts()
				am.GetActiveAlerts()
				am.GetAlertHistory()
			}
			done <- true
		}(i)
	}

	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Should not panic and should have alerts
	assert.NotNil(t, am.GetActiveAlerts())
}

func TestAlertManager_NilReporters(t *testing.T) {
	// Test with nil cost reporter
	am := NewDefaultAlertManager("s1", "t1", nil, nil)
	alerts := am.CheckAlerts()
	assert.Empty(t, alerts) // Should not panic
}
