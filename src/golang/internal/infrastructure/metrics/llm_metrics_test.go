package metrics

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLLMMetricsCollector(t *testing.T) {
	collector := NewLLMMetricsCollector("session123", "trace456")

	assert.Equal(t, "session123", collector.sessionID)
	assert.Equal(t, "trace456", collector.traceID)
	assert.NotNil(t, collector.llmCalls)
	assert.NotNil(t, collector.decisions)
	assert.Equal(t, 0, collector.totalCalls)
}

func TestLLMMetricsCollector_RecordLLMCall_SingleCall(t *testing.T) {
	collector := NewLLMMetricsCollector("session123", "trace456")

	collector.RecordLLMCall(
		"openai", "gpt-4o-mini",
		100, 50, // prompt, completion tokens
		0.001,   // cost
		150,     // latency ms
		nil,     // no error
	)

	metrics := collector.GetModelMetrics("openai", "gpt-4o-mini")
	require.NotNil(t, metrics)

	assert.Equal(t, "openai", metrics.Provider)
	assert.Equal(t, "gpt-4o-mini", metrics.Model)
	assert.Equal(t, 1, metrics.CallCount)
	assert.Equal(t, 100, metrics.PromptTokens)
	assert.Equal(t, 50, metrics.CompletionTokens)
	assert.Equal(t, 150, metrics.TotalTokens)
	assert.InDelta(t, 0.001, metrics.TotalCost, 0.0001)
	assert.Equal(t, int64(150), metrics.AvgLatencyMS)
	assert.Equal(t, 0, metrics.FailureCount)
	assert.Empty(t, metrics.LastError)
}

func TestLLMMetricsCollector_RecordLLMCall_MultipleCalls(t *testing.T) {
	collector := NewLLMMetricsCollector("session123", "trace456")

	// First call
	collector.RecordLLMCall("openai", "gpt-4o-mini", 100, 50, 0.001, 150, nil)

	// Second call
	collector.RecordLLMCall("openai", "gpt-4o-mini", 200, 100, 0.002, 250, nil)

	metrics := collector.GetModelMetrics("openai", "gpt-4o-mini")
	require.NotNil(t, metrics)

	assert.Equal(t, 2, metrics.CallCount)
	assert.Equal(t, 300, metrics.PromptTokens)      // 100 + 200
	assert.Equal(t, 150, metrics.CompletionTokens)  // 50 + 100
	assert.Equal(t, 450, metrics.TotalTokens)       // 150 + 300
	assert.InDelta(t, 0.003, metrics.TotalCost, 0.0001) // 0.001 + 0.002
	assert.Equal(t, int64(200), metrics.AvgLatencyMS)   // (150 + 250) / 2
}

func TestLLMMetricsCollector_RecordLLMCall_WithError(t *testing.T) {
	collector := NewLLMMetricsCollector("session123", "trace456")

	err := errors.New("rate limit exceeded")
	collector.RecordLLMCall(
		"openai", "gpt-4o-mini",
		100, 0,
		0.0,
		0,
		err,
	)

	metrics := collector.GetModelMetrics("openai", "gpt-4o-mini")
	require.NotNil(t, metrics)

	assert.Equal(t, 1, metrics.FailureCount)
	assert.Equal(t, "rate limit exceeded", metrics.LastError)
}

func TestLLMMetricsCollector_RecordLLMCall_MultipleProviders(t *testing.T) {
	collector := NewLLMMetricsCollector("session123", "trace456")

	collector.RecordLLMCall("openai", "gpt-4o-mini", 100, 50, 0.001, 150, nil)
	collector.RecordLLMCall("anthropic", "claude-3-5-sonnet", 200, 100, 0.003, 300, nil)
	collector.RecordLLMCall("deepseek", "deepseek-chat", 50, 25, 0.0001, 100, nil)

	allMetrics := collector.GetAllModelMetrics()

	assert.Len(t, allMetrics, 3)
	assert.Contains(t, allMetrics, "openai/gpt-4o-mini")
	assert.Contains(t, allMetrics, "anthropic/claude-3-5-sonnet")
	assert.Contains(t, allMetrics, "deepseek/deepseek-chat")

	assert.Equal(t, 100, allMetrics["openai/gpt-4o-mini"].PromptTokens)
	assert.Equal(t, 200, allMetrics["anthropic/claude-3-5-sonnet"].PromptTokens)
	assert.Equal(t, 50, allMetrics["deepseek/deepseek-chat"].PromptTokens)
}

func TestLLMMetricsCollector_CacheMetrics_NoActivity(t *testing.T) {
	collector := NewLLMMetricsCollector("session123", "trace456")

	cacheMetrics := collector.GetCacheMetrics()

	assert.Equal(t, 0, cacheMetrics.Hits)
	assert.Equal(t, 0, cacheMetrics.Misses)
	assert.Equal(t, 0.0, cacheMetrics.HitRate)
}

func TestLLMMetricsCollector_CacheMetrics_AllHits(t *testing.T) {
	collector := NewLLMMetricsCollector("session123", "trace456")

	collector.RecordCacheHit()
	collector.RecordCacheHit()
	collector.RecordCacheHit()

	cacheMetrics := collector.GetCacheMetrics()

	assert.Equal(t, 3, cacheMetrics.Hits)
	assert.Equal(t, 0, cacheMetrics.Misses)
	assert.InDelta(t, 100.0, cacheMetrics.HitRate, 0.01)
}

func TestLLMMetricsCollector_CacheMetrics_Mixed(t *testing.T) {
	collector := NewLLMMetricsCollector("session123", "trace456")

	collector.RecordCacheHit()
	collector.RecordCacheHit()
	collector.RecordCacheHit()
	collector.RecordCacheMiss()
	collector.RecordCacheMiss()

	cacheMetrics := collector.GetCacheMetrics()

	assert.Equal(t, 3, cacheMetrics.Hits)
	assert.Equal(t, 2, cacheMetrics.Misses)
	assert.InDelta(t, 60.0, cacheMetrics.HitRate, 0.01) // 3/5 = 60%
}

func TestLLMMetricsCollector_ModelSelection_Single(t *testing.T) {
	collector := NewLLMMetricsCollector("session123", "trace456")

	decision := ModelSelectionDecision{
		AgentID:           "intent_detection",
		TaskType:          "intent_classification",
		ContextSize:       500,
		SelectedModel:     "deepseek/deepseek-chat",
		Reason:            "optimal_for_task",
		AlternativeModels: []string{"openai/gpt-4o-mini", "ollama/mistral"},
		EstimatedCost:     0.0001,
	}

	collector.RecordModelSelection(decision)

	decisions := collector.GetModelSelectionDecisions()
	require.Len(t, decisions, 1)

	assert.Equal(t, "intent_detection", decisions[0].AgentID)
	assert.Equal(t, "intent_classification", decisions[0].TaskType)
	assert.Equal(t, 500, decisions[0].ContextSize)
	assert.Equal(t, "deepseek/deepseek-chat", decisions[0].SelectedModel)
	assert.Equal(t, "optimal_for_task", decisions[0].Reason)
	assert.Len(t, decisions[0].AlternativeModels, 2)
	assert.InDelta(t, 0.0001, decisions[0].EstimatedCost, 0.00001)
	assert.NotZero(t, decisions[0].Timestamp)
}

func TestLLMMetricsCollector_ModelSelection_Multiple(t *testing.T) {
	collector := NewLLMMetricsCollector("session123", "trace456")

	decision1 := ModelSelectionDecision{
		AgentID:       "intent_detection",
		TaskType:      "intent_classification",
		SelectedModel: "deepseek/deepseek-chat",
		Reason:        "optimal_for_task",
	}

	decision2 := ModelSelectionDecision{
		AgentID:       "inference",
		TaskType:      "advanced_inference",
		SelectedModel: "openai/gpt-4o",
		Reason:        "complex_reasoning_required",
	}

	decision3 := ModelSelectionDecision{
		AgentID:       "validation",
		TaskType:      "simple_validation",
		SelectedModel: "openai/gpt-4o-mini",
		Reason:        "budget_constraint",
	}

	collector.RecordModelSelection(decision1)
	collector.RecordModelSelection(decision2)
	collector.RecordModelSelection(decision3)

	decisions := collector.GetModelSelectionDecisions()
	require.Len(t, decisions, 3)

	assert.Equal(t, "intent_detection", decisions[0].AgentID)
	assert.Equal(t, "inference", decisions[1].AgentID)
	assert.Equal(t, "validation", decisions[2].AgentID)
}

func TestLLMMetricsCollector_GetModelMetrics_NonExistent(t *testing.T) {
	collector := NewLLMMetricsCollector("session123", "trace456")

	metrics := collector.GetModelMetrics("nonexistent", "model")

	assert.Nil(t, metrics)
}

func TestLLMMetricsCollector_GetLLMSessionMetrics_Empty(t *testing.T) {
	collector := NewLLMMetricsCollector("session123", "trace456")

	sessionMetrics := collector.GetLLMSessionMetrics()

	assert.Equal(t, "session123", sessionMetrics.SessionID)
	assert.Equal(t, "trace456", sessionMetrics.TraceID)
	assert.Equal(t, 0, sessionMetrics.TotalCalls)
	assert.Equal(t, 0, sessionMetrics.TotalPromptTokens)
	assert.Equal(t, 0, sessionMetrics.TotalCompletionTokens)
	assert.Equal(t, 0, sessionMetrics.TotalTokens)
	assert.Equal(t, 0.0, sessionMetrics.TotalCostUSD)
	assert.Equal(t, 0.0, sessionMetrics.CacheHitRate)
	assert.Equal(t, 0, sessionMetrics.UniqueModels)
	assert.Equal(t, 0, sessionMetrics.DecisionCount)
}

func TestLLMMetricsCollector_GetLLMSessionMetrics_WithData(t *testing.T) {
	collector := NewLLMMetricsCollector("session123", "trace456")

	// Record some LLM calls
	collector.RecordLLMCall("openai", "gpt-4o-mini", 100, 50, 0.001, 150, nil)
	collector.RecordLLMCall("openai", "gpt-4o", 500, 300, 0.005, 500, nil)
	collector.RecordLLMCall("anthropic", "claude-3-5-sonnet", 200, 100, 0.003, 300, nil)

	// Record cache activity
	collector.RecordCacheHit()
	collector.RecordCacheHit()
	collector.RecordCacheMiss()

	// Record model selection
	decision := ModelSelectionDecision{
		AgentID:       "inference",
		SelectedModel: "openai/gpt-4o",
	}
	collector.RecordModelSelection(decision)

	sessionMetrics := collector.GetLLMSessionMetrics()

	assert.Equal(t, "session123", sessionMetrics.SessionID)
	assert.Equal(t, "trace456", sessionMetrics.TraceID)
	assert.Equal(t, 3, sessionMetrics.TotalCalls)
	assert.Equal(t, 800, sessionMetrics.TotalPromptTokens)      // 100 + 500 + 200
	assert.Equal(t, 450, sessionMetrics.TotalCompletionTokens)  // 50 + 300 + 100
	assert.Equal(t, 1250, sessionMetrics.TotalTokens)           // 800 + 450
	assert.InDelta(t, 0.009, sessionMetrics.TotalCostUSD, 0.0001) // 0.001 + 0.005 + 0.003
	assert.InDelta(t, 66.67, sessionMetrics.CacheHitRate, 0.1)    // 2/3 = 66.67%
	assert.Equal(t, 3, sessionMetrics.UniqueModels)
	assert.Equal(t, 1, sessionMetrics.DecisionCount)
}

func TestLLMMetricsCollector_ProviderMetrics(t *testing.T) {
	collector := NewLLMMetricsCollector("session123", "trace456")

	// OpenAI calls (2 models)
	collector.RecordLLMCall("openai", "gpt-4o-mini", 100, 50, 0.001, 150, nil)
	collector.RecordLLMCall("openai", "gpt-4o", 500, 300, 0.005, 500, nil)
	collector.RecordLLMCall("openai", "gpt-4o-mini", 100, 50, 0.001, 150, errors.New("timeout"))

	// Anthropic calls (1 model)
	collector.RecordLLMCall("anthropic", "claude-3-5-sonnet", 200, 100, 0.003, 300, nil)

	sessionMetrics := collector.GetLLMSessionMetrics()

	require.Contains(t, sessionMetrics.ByProvider, "openai")
	require.Contains(t, sessionMetrics.ByProvider, "anthropic")

	openaiMetrics := sessionMetrics.ByProvider["openai"]
	assert.Equal(t, 3, openaiMetrics.CallCount)
	assert.Equal(t, 1100, openaiMetrics.TotalTokens) // (100+50) + (500+300) + (100+50) = 150 + 800 + 150
	assert.InDelta(t, 0.007, openaiMetrics.TotalCost, 0.0001)
	assert.Equal(t, 1, openaiMetrics.FailureCount)
	assert.InDelta(t, 33.33, openaiMetrics.FailureRate, 0.1) // 1/3 = 33.33%

	anthropicMetrics := sessionMetrics.ByProvider["anthropic"]
	assert.Equal(t, 1, anthropicMetrics.CallCount)
	assert.Equal(t, 300, anthropicMetrics.TotalTokens)
	assert.InDelta(t, 0.003, anthropicMetrics.TotalCost, 0.0001)
	assert.Equal(t, 0, anthropicMetrics.FailureCount)
	assert.Equal(t, 0.0, anthropicMetrics.FailureRate)
}

func TestLLMMetricsCollector_Reset(t *testing.T) {
	collector := NewLLMMetricsCollector("session123", "trace456")

	// Add some data
	collector.RecordLLMCall("openai", "gpt-4o-mini", 100, 50, 0.001, 150, nil)
	collector.RecordCacheHit()
	collector.RecordModelSelection(ModelSelectionDecision{
		AgentID:       "test",
		SelectedModel: "openai/gpt-4o-mini",
	})

	// Verify data exists
	assert.NotNil(t, collector.GetModelMetrics("openai", "gpt-4o-mini"))
	assert.Equal(t, 1, collector.totalCalls)
	assert.Equal(t, 1, collector.cacheHits)
	assert.Len(t, collector.decisions, 1)

	// Reset
	collector.Reset()

	// Verify data cleared
	assert.Nil(t, collector.GetModelMetrics("openai", "gpt-4o-mini"))
	assert.Equal(t, 0, collector.totalCalls)
	assert.Equal(t, 0, collector.cacheHits)
	assert.Equal(t, 0, collector.cacheMisses)
	assert.Len(t, collector.decisions, 0)
	assert.Equal(t, 0, collector.totalPromptTokens)
	assert.Equal(t, 0, collector.totalCompletionTokens)
	assert.Equal(t, 0.0, collector.totalCost)
}

func TestLLMMetricsCollector_ThreadSafety(t *testing.T) {
	collector := NewLLMMetricsCollector("session123", "trace456")

	// Simulate concurrent access
	done := make(chan bool)
	numGoroutines := 10
	callsPerGoroutine := 100

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			for j := 0; j < callsPerGoroutine; j++ {
				collector.RecordLLMCall(
					"openai", "gpt-4o-mini",
					100, 50,
					0.001,
					150,
					nil,
				)

				if j%2 == 0 {
					collector.RecordCacheHit()
				} else {
					collector.RecordCacheMiss()
				}

				if j%10 == 0 {
					collector.RecordModelSelection(ModelSelectionDecision{
						AgentID:       "test_agent",
						SelectedModel: "openai/gpt-4o-mini",
					})
				}
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Verify correct totals
	metrics := collector.GetModelMetrics("openai", "gpt-4o-mini")
	require.NotNil(t, metrics)

	expectedCalls := numGoroutines * callsPerGoroutine
	assert.Equal(t, expectedCalls, metrics.CallCount)
	assert.Equal(t, expectedCalls*100, metrics.PromptTokens)
	assert.Equal(t, expectedCalls*50, metrics.CompletionTokens)

	cacheMetrics := collector.GetCacheMetrics()
	assert.Equal(t, expectedCalls, cacheMetrics.Hits+cacheMetrics.Misses)

	decisions := collector.GetModelSelectionDecisions()
	assert.Equal(t, numGoroutines*10, len(decisions))
}

func TestLLMMetricsCollector_GetAllModelMetrics_Copy(t *testing.T) {
	collector := NewLLMMetricsCollector("session123", "trace456")

	collector.RecordLLMCall("openai", "gpt-4o-mini", 100, 50, 0.001, 150, nil)

	// Get metrics
	metrics1 := collector.GetAllModelMetrics()

	// Modify the returned map (should not affect internal state)
	metrics1["openai/gpt-4o-mini"].CallCount = 999

	// Get metrics again
	metrics2 := collector.GetAllModelMetrics()

	// Should still have original value
	assert.Equal(t, 1, metrics2["openai/gpt-4o-mini"].CallCount)
}

func TestLLMMetricsCollector_GetModelMetrics_Copy(t *testing.T) {
	collector := NewLLMMetricsCollector("session123", "trace456")

	collector.RecordLLMCall("openai", "gpt-4o-mini", 100, 50, 0.001, 150, nil)

	// Get metrics
	metrics1 := collector.GetModelMetrics("openai", "gpt-4o-mini")

	// Modify the returned struct (should not affect internal state)
	metrics1.CallCount = 999

	// Get metrics again
	metrics2 := collector.GetModelMetrics("openai", "gpt-4o-mini")

	// Should still have original value
	assert.Equal(t, 1, metrics2.CallCount)
}

func TestLLMMetricsCollector_LastUsedAt(t *testing.T) {
	collector := NewLLMMetricsCollector("session123", "trace456")

	beforeCall := time.Now()
	collector.RecordLLMCall("openai", "gpt-4o-mini", 100, 50, 0.001, 150, nil)
	afterCall := time.Now()

	metrics := collector.GetModelMetrics("openai", "gpt-4o-mini")
	require.NotNil(t, metrics)

	assert.True(t, metrics.LastUsedAt.After(beforeCall) || metrics.LastUsedAt.Equal(beforeCall))
	assert.True(t, metrics.LastUsedAt.Before(afterCall) || metrics.LastUsedAt.Equal(afterCall))
}
