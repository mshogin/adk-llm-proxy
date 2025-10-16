package metrics

import (
	"testing"
	"time"

	"github.com/mshogin/agents/internal/domain/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCollector(t *testing.T) {
	collector := NewCollector("session123", "trace456")

	assert.Equal(t, "session123", collector.sessionID)
	assert.Equal(t, "trace456", collector.traceID)
	assert.NotNil(t, collector.agentMetrics)
	assert.NotZero(t, collector.startTime)
}

func TestCollector_RecordAgentExecution_SingleAgent(t *testing.T) {
	collector := NewCollector("session123", "trace456")

	run := models.AgentRun{
		AgentID:    "intent_detection",
		Status:     "success",
		DurationMS: 100,
	}

	collector.RecordAgentExecution(run)

	metrics := collector.GetAgentMetrics("intent_detection")
	require.NotNil(t, metrics)

	assert.Equal(t, "intent_detection", metrics.AgentID)
	assert.Equal(t, 1, metrics.ExecutionCount)
	assert.Equal(t, int64(100), metrics.TotalDurationMS)
	assert.Equal(t, int64(100), metrics.AvgDurationMS)
	assert.Equal(t, 1, metrics.SuccessCount)
	assert.Equal(t, 0, metrics.FailureCount)
	assert.Equal(t, "success", metrics.LastStatus)
}

func TestCollector_RecordAgentExecution_MultipleExecutions(t *testing.T) {
	collector := NewCollector("session123", "trace456")

	// Record first execution
	collector.RecordAgentExecution(models.AgentRun{
		AgentID:    "intent_detection",
		Status:     "success",
		DurationMS: 100,
	})

	// Record second execution
	collector.RecordAgentExecution(models.AgentRun{
		AgentID:    "intent_detection",
		Status:     "success",
		DurationMS: 200,
	})

	metrics := collector.GetAgentMetrics("intent_detection")
	require.NotNil(t, metrics)

	assert.Equal(t, 2, metrics.ExecutionCount)
	assert.Equal(t, int64(300), metrics.TotalDurationMS)
	assert.Equal(t, int64(150), metrics.AvgDurationMS) // (100+200)/2
	assert.Equal(t, 2, metrics.SuccessCount)
}

func TestCollector_RecordAgentExecution_WithError(t *testing.T) {
	collector := NewCollector("session123", "trace456")

	run := models.AgentRun{
		AgentID:    "intent_detection",
		Status:     "error",
		DurationMS: 50,
		Error:      "failed to parse input",
	}

	collector.RecordAgentExecution(run)

	metrics := collector.GetAgentMetrics("intent_detection")
	require.NotNil(t, metrics)

	assert.Equal(t, 0, metrics.SuccessCount)
	assert.Equal(t, 1, metrics.FailureCount)
	assert.Equal(t, "error", metrics.LastStatus)
	assert.Equal(t, "failed to parse input", metrics.LastError)
}

func TestCollector_RecordAgentExecution_MultipleAgents(t *testing.T) {
	collector := NewCollector("session123", "trace456")

	collector.RecordAgentExecution(models.AgentRun{
		AgentID:    "intent_detection",
		Status:     "success",
		DurationMS: 100,
	})

	collector.RecordAgentExecution(models.AgentRun{
		AgentID:    "inference",
		Status:     "success",
		DurationMS: 150,
	})

	allMetrics := collector.GetAllAgentMetrics()

	assert.Len(t, allMetrics, 2)
	assert.Contains(t, allMetrics, "intent_detection")
	assert.Contains(t, allMetrics, "inference")

	assert.Equal(t, int64(100), allMetrics["intent_detection"].TotalDurationMS)
	assert.Equal(t, int64(150), allMetrics["inference"].TotalDurationMS)
}

func TestCollector_RecordLLMUsage(t *testing.T) {
	collector := NewCollector("session123", "trace456")

	collector.RecordLLMUsage("intent_detection", 500, 0.001)

	metrics := collector.GetAgentMetrics("intent_detection")
	require.NotNil(t, metrics)

	assert.Equal(t, 1, metrics.LLMCalls)
	assert.Equal(t, 500, metrics.TotalTokens)
	assert.Equal(t, 0.001, metrics.TotalCost)
}

func TestCollector_RecordLLMUsage_MultipleCalls(t *testing.T) {
	collector := NewCollector("session123", "trace456")

	collector.RecordLLMUsage("intent_detection", 500, 0.001)
	collector.RecordLLMUsage("intent_detection", 300, 0.0006)

	metrics := collector.GetAgentMetrics("intent_detection")
	require.NotNil(t, metrics)

	assert.Equal(t, 2, metrics.LLMCalls)
	assert.Equal(t, 800, metrics.TotalTokens)
	assert.InDelta(t, 0.0016, metrics.TotalCost, 0.0001)
}

func TestCollector_GetAgentMetrics_NonExistent(t *testing.T) {
	collector := NewCollector("session123", "trace456")

	metrics := collector.GetAgentMetrics("nonexistent")

	assert.Nil(t, metrics)
}

func TestCollector_GetSessionMetrics_Empty(t *testing.T) {
	collector := NewCollector("session123", "trace456")

	sessionMetrics := collector.GetSessionMetrics()

	assert.Equal(t, "session123", sessionMetrics.SessionID)
	assert.Equal(t, "trace456", sessionMetrics.TraceID)
	assert.Equal(t, 0, sessionMetrics.TotalExecutions)
	assert.Equal(t, 0, sessionMetrics.AgentCount)
	assert.Equal(t, int64(0), sessionMetrics.TotalDurationMS)
	assert.Equal(t, 0.0, sessionMetrics.TotalCostUSD)
}

func TestCollector_GetSessionMetrics_WithData(t *testing.T) {
	collector := NewCollector("session123", "trace456")

	// Record some executions
	collector.RecordAgentExecution(models.AgentRun{
		AgentID:    "intent_detection",
		Status:     "success",
		DurationMS: 100,
	})

	collector.RecordAgentExecution(models.AgentRun{
		AgentID:    "inference",
		Status:     "success",
		DurationMS: 150,
	})

	collector.RecordAgentExecution(models.AgentRun{
		AgentID:    "validation",
		Status:     "error",
		DurationMS: 50,
		Error:      "validation failed",
	})

	// Record LLM usage
	collector.RecordLLMUsage("intent_detection", 500, 0.001)

	sessionMetrics := collector.GetSessionMetrics()

	assert.Equal(t, "session123", sessionMetrics.SessionID)
	assert.Equal(t, "trace456", sessionMetrics.TraceID)
	assert.Equal(t, 3, sessionMetrics.TotalExecutions)
	assert.Equal(t, 3, sessionMetrics.AgentCount)
	assert.Equal(t, int64(300), sessionMetrics.TotalDurationMS) // 100+150+50
	assert.Equal(t, 2, sessionMetrics.SuccessCount)
	assert.Equal(t, 1, sessionMetrics.FailureCount)
	assert.Equal(t, 500, sessionMetrics.TotalTokens)
	assert.InDelta(t, 0.001, sessionMetrics.TotalCostUSD, 0.0001)
}

func TestSessionMetrics_SuccessRate(t *testing.T) {
	tests := []struct {
		name           string
		totalExec      int
		successCount   int
		expectedRate   float64
	}{
		{
			name:         "100% success",
			totalExec:    10,
			successCount: 10,
			expectedRate: 100.0,
		},
		{
			name:         "50% success",
			totalExec:    10,
			successCount: 5,
			expectedRate: 50.0,
		},
		{
			name:         "0% success",
			totalExec:    10,
			successCount: 0,
			expectedRate: 0.0,
		},
		{
			name:         "no executions",
			totalExec:    0,
			successCount: 0,
			expectedRate: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metrics := SessionMetrics{
				TotalExecutions: tt.totalExec,
				SuccessCount:    tt.successCount,
			}

			rate := metrics.SuccessRate()

			assert.InDelta(t, tt.expectedRate, rate, 0.01)
		})
	}
}

func TestSessionMetrics_AvgCostPerExecution(t *testing.T) {
	tests := []struct {
		name        string
		totalExec   int
		totalCost   float64
		expectedAvg float64
	}{
		{
			name:        "with executions",
			totalExec:   10,
			totalCost:   0.1,
			expectedAvg: 0.01,
		},
		{
			name:        "no executions",
			totalExec:   0,
			totalCost:   0.0,
			expectedAvg: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metrics := SessionMetrics{
				TotalExecutions: tt.totalExec,
				TotalCostUSD:    tt.totalCost,
			}

			avg := metrics.AvgCostPerExecution()

			assert.InDelta(t, tt.expectedAvg, avg, 0.0001)
		})
	}
}

func TestSessionMetrics_AvgTokensPerExecution(t *testing.T) {
	tests := []struct {
		name        string
		totalExec   int
		totalTokens int
		expectedAvg float64
	}{
		{
			name:        "with executions",
			totalExec:   10,
			totalTokens: 5000,
			expectedAvg: 500.0,
		},
		{
			name:        "no executions",
			totalExec:   0,
			totalTokens: 0,
			expectedAvg: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metrics := SessionMetrics{
				TotalExecutions: tt.totalExec,
				TotalTokens:     tt.totalTokens,
			}

			avg := metrics.AvgTokensPerExecution()

			assert.InDelta(t, tt.expectedAvg, avg, 0.01)
		})
	}
}

func TestCollector_ExportToContext(t *testing.T) {
	collector := NewCollector("session123", "trace456")

	// Record some data
	collector.RecordAgentExecution(models.AgentRun{
		AgentID:    "intent_detection",
		Status:     "success",
		DurationMS: 100,
	})

	collector.RecordLLMUsage("intent_detection", 500, 0.001)

	// Export to context
	ctx := &models.AgentContext{}
	collector.ExportToContext(ctx)

	require.NotNil(t, ctx.Diagnostics)
	require.NotNil(t, ctx.Diagnostics.Performance)

	assert.Equal(t, int64(100), ctx.Diagnostics.Performance.TotalDurationMS)

	require.NotNil(t, ctx.Diagnostics.Performance.AgentMetrics)
	require.Contains(t, ctx.Diagnostics.Performance.AgentMetrics, "intent_detection")

	agentMetrics := ctx.Diagnostics.Performance.AgentMetrics["intent_detection"]
	assert.Equal(t, int64(100), agentMetrics.DurationMS)
	assert.Equal(t, 1, agentMetrics.LLMCalls)
	assert.Equal(t, 500, agentMetrics.Tokens)
	assert.InDelta(t, 0.001, agentMetrics.Cost, 0.0001)
	assert.Equal(t, "success", agentMetrics.Status)
}

func TestCollector_ExportToContext_NilDiagnostics(t *testing.T) {
	collector := NewCollector("session123", "trace456")

	collector.RecordAgentExecution(models.AgentRun{
		AgentID:    "intent_detection",
		Status:     "success",
		DurationMS: 100,
	})

	// Export to context with nil diagnostics
	ctx := &models.AgentContext{
		Diagnostics: nil,
	}

	collector.ExportToContext(ctx)

	// Should create diagnostics and performance
	require.NotNil(t, ctx.Diagnostics)
	require.NotNil(t, ctx.Diagnostics.Performance)
	assert.Equal(t, int64(100), ctx.Diagnostics.Performance.TotalDurationMS)
}

func TestCollector_Reset(t *testing.T) {
	collector := NewCollector("session123", "trace456")

	// Add some data
	collector.RecordAgentExecution(models.AgentRun{
		AgentID:    "intent_detection",
		Status:     "success",
		DurationMS: 100,
	})

	collector.RecordLLMUsage("intent_detection", 500, 0.001)

	// Verify data exists
	assert.NotNil(t, collector.GetAgentMetrics("intent_detection"))
	assert.Equal(t, 500, collector.totalTokens)

	// Reset
	startTimeBefore := collector.startTime
	time.Sleep(10 * time.Millisecond) // Small delay to ensure time changes
	collector.Reset()

	// Verify data cleared
	assert.Nil(t, collector.GetAgentMetrics("intent_detection"))
	assert.Equal(t, 0, collector.totalTokens)
	assert.Equal(t, 0.0, collector.totalCost)
	assert.Equal(t, time.Duration(0), collector.totalDuration)
	assert.NotEqual(t, startTimeBefore, collector.startTime) // Start time updated
}

func TestCollector_ThreadSafety(t *testing.T) {
	collector := NewCollector("session123", "trace456")

	// Simulate concurrent access
	done := make(chan bool)
	numGoroutines := 10
	executionsPerGoroutine := 100

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			for j := 0; j < executionsPerGoroutine; j++ {
				collector.RecordAgentExecution(models.AgentRun{
					AgentID:    "test_agent",
					Status:     "success",
					DurationMS: 10,
				})

				collector.RecordLLMUsage("test_agent", 100, 0.0001)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Verify correct totals
	metrics := collector.GetAgentMetrics("test_agent")
	require.NotNil(t, metrics)

	expectedExecutions := numGoroutines * executionsPerGoroutine
	assert.Equal(t, expectedExecutions, metrics.ExecutionCount)
	assert.Equal(t, expectedExecutions, metrics.LLMCalls)
	assert.Equal(t, expectedExecutions*100, metrics.TotalTokens)
}

func TestCollector_GetAllAgentMetrics_Copy(t *testing.T) {
	collector := NewCollector("session123", "trace456")

	collector.RecordAgentExecution(models.AgentRun{
		AgentID:    "intent_detection",
		Status:     "success",
		DurationMS: 100,
	})

	// Get metrics
	metrics1 := collector.GetAllAgentMetrics()

	// Modify the returned map (should not affect internal state)
	metrics1["intent_detection"].ExecutionCount = 999

	// Get metrics again
	metrics2 := collector.GetAllAgentMetrics()

	// Should still have original value
	assert.Equal(t, 1, metrics2["intent_detection"].ExecutionCount)
}

func TestCollector_GetAgentMetrics_Copy(t *testing.T) {
	collector := NewCollector("session123", "trace456")

	collector.RecordAgentExecution(models.AgentRun{
		AgentID:    "intent_detection",
		Status:     "success",
		DurationMS: 100,
	})

	// Get metrics
	metrics1 := collector.GetAgentMetrics("intent_detection")

	// Modify the returned struct (should not affect internal state)
	metrics1.ExecutionCount = 999

	// Get metrics again
	metrics2 := collector.GetAgentMetrics("intent_detection")

	// Should still have original value
	assert.Equal(t, 1, metrics2.ExecutionCount)
}

func TestCollector_SessionDuration(t *testing.T) {
	collector := NewCollector("session123", "trace456")

	startTime := collector.startTime

	// Wait a bit
	time.Sleep(50 * time.Millisecond)

	// Session duration should be positive
	elapsed := time.Since(startTime)
	assert.Greater(t, elapsed.Milliseconds(), int64(40)) // Allow some tolerance
}
