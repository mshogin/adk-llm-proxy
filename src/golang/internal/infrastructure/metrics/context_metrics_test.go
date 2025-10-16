package metrics

import (
	"testing"
	"time"

	"github.com/mshogin/agents/internal/domain/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewContextMetricsCollector(t *testing.T) {
	collector := NewContextMetricsCollector("session123", "trace456")

	assert.Equal(t, "session123", collector.sessionID)
	assert.Equal(t, "trace456", collector.traceID)
	assert.NotNil(t, collector.snapshots)
	assert.Equal(t, 0, collector.snapshotCount)
}

func TestContextMetricsCollector_RecordContextSnapshot_FirstSnapshot(t *testing.T) {
	collector := NewContextMetricsCollector("session123", "trace456")

	ctx := &models.AgentContext{
		Reasoning: &models.ReasoningContext{
			Intents: []models.Intent{
				{Type: "query_commits", Confidence: 0.9},
			},
			Conclusions: []models.Conclusion{
				{ID: "c1", Description: "Found commits", Confidence: 0.95},
			},
			Artifacts: []models.Artifact{
				{ID: "a1", Type: "report"},
			},
		},
		Enrichment: &models.EnrichmentContext{
			Facts: []models.Fact{
				{ID: "f1", Source: "gitlab"},
				{ID: "f2", Source: "gitlab"},
			},
		},
	}

	timestamp := time.Now().Unix()
	collector.RecordContextSnapshot("intent_detection", ctx, timestamp)

	snapshot := collector.GetContextSnapshot("intent_detection")
	require.NotNil(t, snapshot)

	assert.Equal(t, "intent_detection", snapshot.AgentID)
	assert.Greater(t, snapshot.ContextSize, 0)
	assert.Equal(t, 1, snapshot.ArtifactCount)
	assert.Equal(t, 2, snapshot.FactCount)
	assert.Equal(t, 1, snapshot.IntentCount)
	assert.Equal(t, 1, snapshot.ConclusionCount)
	assert.Equal(t, snapshot.ContextSize, snapshot.DiffSize) // First snapshot, diff = full size
	assert.Equal(t, timestamp, snapshot.Timestamp)
}

func TestContextMetricsCollector_RecordContextSnapshot_MultipleSnapshots(t *testing.T) {
	collector := NewContextMetricsCollector("session123", "trace456")

	// First snapshot (small context)
	ctx1 := &models.AgentContext{
		Reasoning: &models.ReasoningContext{
			Intents: []models.Intent{
				{Type: "query_commits", Confidence: 0.9},
			},
		},
	}

	timestamp1 := time.Now().Unix()
	collector.RecordContextSnapshot("agent1", ctx1, timestamp1)

	snapshot1 := collector.GetContextSnapshot("agent1")
	initialSize := snapshot1.ContextSize

	// Second snapshot (larger context)
	ctx2 := &models.AgentContext{
		Reasoning: &models.ReasoningContext{
			Intents: []models.Intent{
				{Type: "query_commits", Confidence: 0.9},
			},
			Conclusions: []models.Conclusion{
				{ID: "c1", Description: "Found commits", Confidence: 0.95},
				{ID: "c2", Description: "Analyzed commits", Confidence: 0.90},
			},
		},
		Enrichment: &models.EnrichmentContext{
			Facts: []models.Fact{
				{ID: "f1", Source: "gitlab"},
				{ID: "f2", Source: "gitlab"},
				{ID: "f3", Source: "gitlab"},
			},
		},
	}

	timestamp2 := timestamp1 + 10
	collector.RecordContextSnapshot("agent1", ctx2, timestamp2)

	snapshot2 := collector.GetContextSnapshot("agent1")
	assert.Greater(t, snapshot2.ContextSize, initialSize)
	assert.Equal(t, snapshot2.ContextSize-initialSize, snapshot2.DiffSize)
	assert.Equal(t, 3, snapshot2.FactCount)
	assert.Equal(t, 2, snapshot2.ConclusionCount)
}

func TestContextMetricsCollector_RecordContextSnapshot_WithDiagnostics(t *testing.T) {
	collector := NewContextMetricsCollector("session123", "trace456")

	ctx := &models.AgentContext{
		Diagnostics: &models.DiagnosticsContext{
			Errors: []models.ErrorReport{
				{Message: "Error 1"},
				{Message: "Error 2"},
			},
			Warnings: []models.Warning{
				{Message: "Warning 1"},
			},
		},
	}

	timestamp := time.Now().Unix()
	collector.RecordContextSnapshot("validation", ctx, timestamp)

	snapshot := collector.GetContextSnapshot("validation")
	require.NotNil(t, snapshot)

	assert.Equal(t, 2, snapshot.ErrorCount)
	assert.Equal(t, 1, snapshot.WarningCount)
}

func TestContextMetricsCollector_GetContextSnapshot_NonExistent(t *testing.T) {
	collector := NewContextMetricsCollector("session123", "trace456")

	snapshot := collector.GetContextSnapshot("nonexistent")

	assert.Nil(t, snapshot)
}

func TestContextMetricsCollector_GetAllContextSnapshots(t *testing.T) {
	collector := NewContextMetricsCollector("session123", "trace456")

	ctx1 := &models.AgentContext{
		Reasoning: &models.ReasoningContext{
			Intents: []models.Intent{{Type: "query_commits"}},
		},
	}

	ctx2 := &models.AgentContext{
		Reasoning: &models.ReasoningContext{
			Conclusions: []models.Conclusion{{ID: "c1"}},
		},
	}

	timestamp := time.Now().Unix()
	collector.RecordContextSnapshot("agent1", ctx1, timestamp)
	collector.RecordContextSnapshot("agent2", ctx2, timestamp+10)

	snapshots := collector.GetAllContextSnapshots()

	assert.Len(t, snapshots, 2)
	assert.Contains(t, snapshots, "agent1")
	assert.Contains(t, snapshots, "agent2")
	assert.Equal(t, 1, snapshots["agent1"].IntentCount)
	assert.Equal(t, 1, snapshots["agent2"].ConclusionCount)
}

func TestContextMetricsCollector_GetContextGrowth_NoSnapshots(t *testing.T) {
	collector := NewContextMetricsCollector("session123", "trace456")

	growth := collector.GetContextGrowth()

	assert.Equal(t, 0, growth.InitialSize)
	assert.Equal(t, 0, growth.FinalSize)
	assert.Equal(t, 0, growth.GrowthBytes)
	assert.Equal(t, 0.0, growth.GrowthRatio)
}

func TestContextMetricsCollector_GetContextGrowth_WithGrowth(t *testing.T) {
	collector := NewContextMetricsCollector("session123", "trace456")

	// Small initial context
	ctx1 := &models.AgentContext{
		Reasoning: &models.ReasoningContext{
			Intents: []models.Intent{{Type: "query_commits"}},
		},
	}

	// Large final context
	ctx2 := &models.AgentContext{
		Reasoning: &models.ReasoningContext{
			Intents: []models.Intent{
				{Type: "query_commits"},
			},
			Conclusions: []models.Conclusion{
				{ID: "c1", Description: "Found commits", Confidence: 0.95},
				{ID: "c2", Description: "Analyzed commits", Confidence: 0.90},
				{ID: "c3", Description: "Validated commits", Confidence: 0.85},
			},
		},
		Enrichment: &models.EnrichmentContext{
			Facts: []models.Fact{
				{ID: "f1", Source: "gitlab"},
				{ID: "f2", Source: "gitlab"},
				{ID: "f3", Source: "gitlab"},
				{ID: "f4", Source: "gitlab"},
			},
		},
	}

	timestamp1 := time.Now().Unix()
	collector.RecordContextSnapshot("agent1", ctx1, timestamp1)
	collector.RecordContextSnapshot("agent2", ctx2, timestamp1+100)

	growth := collector.GetContextGrowth()

	assert.Greater(t, growth.FinalSize, growth.InitialSize)
	assert.Equal(t, growth.FinalSize-growth.InitialSize, growth.GrowthBytes)
	assert.Greater(t, growth.GrowthRatio, 1.0)
}

func TestContextMetricsCollector_GetArtifactGrowth_NoSnapshots(t *testing.T) {
	collector := NewContextMetricsCollector("session123", "trace456")

	growth := collector.GetArtifactGrowth()

	assert.Equal(t, 0, growth.InitialCount)
	assert.Equal(t, 0, growth.FinalCount)
	assert.Equal(t, 0, growth.AddedCount)
	assert.Equal(t, 0.0, growth.GrowthRatio)
}

func TestContextMetricsCollector_GetArtifactGrowth_WithGrowth(t *testing.T) {
	collector := NewContextMetricsCollector("session123", "trace456")

	// Initial context with 1 artifact
	ctx1 := &models.AgentContext{
		Reasoning: &models.ReasoningContext{
			Artifacts: []models.Artifact{
				{ID: "a1", Type: "report"},
			},
		},
	}

	// Final context with 3 artifacts
	ctx2 := &models.AgentContext{
		Reasoning: &models.ReasoningContext{
			Artifacts: []models.Artifact{
				{ID: "a1", Type: "report"},
				{ID: "a2", Type: "command_list"},
				{ID: "a3", Type: "diff"},
			},
		},
	}

	timestamp1 := time.Now().Unix()
	collector.RecordContextSnapshot("agent1", ctx1, timestamp1)
	collector.RecordContextSnapshot("agent2", ctx2, timestamp1+100)

	growth := collector.GetArtifactGrowth()

	assert.Equal(t, 1, growth.InitialCount)
	assert.Equal(t, 3, growth.FinalCount)
	assert.Equal(t, 2, growth.AddedCount)
	assert.InDelta(t, 3.0, growth.GrowthRatio, 0.01) // 3/1 = 3.0
}

func TestContextMetricsCollector_GetArtifactGrowth_FromZero(t *testing.T) {
	collector := NewContextMetricsCollector("session123", "trace456")

	// Initial context with no artifacts
	ctx1 := &models.AgentContext{
		Reasoning: &models.ReasoningContext{
			Intents: []models.Intent{{Type: "query_commits"}},
		},
	}

	// Final context with 2 artifacts
	ctx2 := &models.AgentContext{
		Reasoning: &models.ReasoningContext{
			Artifacts: []models.Artifact{
				{ID: "a1", Type: "report"},
				{ID: "a2", Type: "command_list"},
			},
		},
	}

	timestamp1 := time.Now().Unix()
	collector.RecordContextSnapshot("agent1", ctx1, timestamp1)
	collector.RecordContextSnapshot("agent2", ctx2, timestamp1+100)

	growth := collector.GetArtifactGrowth()

	assert.Equal(t, 0, growth.InitialCount)
	assert.Equal(t, 2, growth.FinalCount)
	assert.Equal(t, 2, growth.AddedCount)
	assert.Equal(t, 2.0, growth.GrowthRatio) // Special case: started from 0
}

func TestContextMetricsCollector_GetContextSessionMetrics_Empty(t *testing.T) {
	collector := NewContextMetricsCollector("session123", "trace456")

	sessionMetrics := collector.GetContextSessionMetrics()

	assert.Equal(t, "session123", sessionMetrics.SessionID)
	assert.Equal(t, "trace456", sessionMetrics.TraceID)
	assert.Equal(t, 0, sessionMetrics.MaxContextSize)
	assert.Equal(t, 0, sessionMetrics.TotalDiffSize)
	assert.Equal(t, 0, sessionMetrics.TotalArtifactCount)
	assert.Equal(t, 0, sessionMetrics.SnapshotCount)
}

func TestContextMetricsCollector_GetContextSessionMetrics_WithData(t *testing.T) {
	collector := NewContextMetricsCollector("session123", "trace456")

	ctx1 := &models.AgentContext{
		Reasoning: &models.ReasoningContext{
			Intents: []models.Intent{{Type: "query_commits"}},
			Artifacts: []models.Artifact{
				{ID: "a1", Type: "report"},
			},
		},
	}

	ctx2 := &models.AgentContext{
		Reasoning: &models.ReasoningContext{
			Intents: []models.Intent{{Type: "query_commits"}},
			Conclusions: []models.Conclusion{
				{ID: "c1", Description: "Found commits"},
				{ID: "c2", Description: "Analyzed commits"},
			},
			Artifacts: []models.Artifact{
				{ID: "a1", Type: "report"},
				{ID: "a2", Type: "command_list"},
			},
		},
		Enrichment: &models.EnrichmentContext{
			Facts: []models.Fact{
				{ID: "f1", Source: "gitlab"},
			},
		},
	}

	timestamp1 := time.Now().Unix()
	collector.RecordContextSnapshot("agent1", ctx1, timestamp1)
	collector.RecordContextSnapshot("agent2", ctx2, timestamp1+10)

	sessionMetrics := collector.GetContextSessionMetrics()

	assert.Equal(t, "session123", sessionMetrics.SessionID)
	assert.Equal(t, "trace456", sessionMetrics.TraceID)
	assert.Greater(t, sessionMetrics.MaxContextSize, 0)
	assert.Greater(t, sessionMetrics.TotalDiffSize, 0)
	assert.Equal(t, 2, sessionMetrics.TotalArtifactCount) // Latest count
	assert.Equal(t, 2, sessionMetrics.SnapshotCount)
	assert.NotNil(t, sessionMetrics.ContextGrowth)
	assert.NotNil(t, sessionMetrics.ArtifactGrowth)
	assert.Len(t, sessionMetrics.ByAgent, 2)
}

func TestContextMetricsCollector_MaxContextSize(t *testing.T) {
	collector := NewContextMetricsCollector("session123", "trace456")

	// Small context
	ctx1 := &models.AgentContext{
		Reasoning: &models.ReasoningContext{
			Intents: []models.Intent{{Type: "query_commits"}},
		},
	}

	// Large context
	ctx2 := &models.AgentContext{
		Reasoning: &models.ReasoningContext{
			Intents: []models.Intent{{Type: "query_commits"}},
			Conclusions: []models.Conclusion{
				{ID: "c1", Description: "Very long description " + string(make([]byte, 1000))},
			},
		},
	}

	// Medium context
	ctx3 := &models.AgentContext{
		Reasoning: &models.ReasoningContext{
			Intents: []models.Intent{
				{Type: "query_commits"},
				{Type: "query_issues"},
			},
		},
	}

	timestamp := time.Now().Unix()
	collector.RecordContextSnapshot("agent1", ctx1, timestamp)
	collector.RecordContextSnapshot("agent2", ctx2, timestamp+10)
	collector.RecordContextSnapshot("agent3", ctx3, timestamp+20)

	sessionMetrics := collector.GetContextSessionMetrics()

	// Max should be the size of ctx2 (largest context)
	snapshot2 := collector.GetContextSnapshot("agent2")
	assert.Equal(t, snapshot2.ContextSize, sessionMetrics.MaxContextSize)
}

func TestContextMetricsCollector_Reset(t *testing.T) {
	collector := NewContextMetricsCollector("session123", "trace456")

	// Add some data
	ctx := &models.AgentContext{
		Reasoning: &models.ReasoningContext{
			Intents: []models.Intent{{Type: "query_commits"}},
			Artifacts: []models.Artifact{{ID: "a1", Type: "report"}},
		},
	}

	timestamp := time.Now().Unix()
	collector.RecordContextSnapshot("agent1", ctx, timestamp)

	// Verify data exists
	assert.NotNil(t, collector.GetContextSnapshot("agent1"))
	assert.Equal(t, 1, collector.snapshotCount)
	assert.Greater(t, collector.maxContextSize, 0)

	// Reset
	collector.Reset()

	// Verify data cleared
	assert.Nil(t, collector.GetContextSnapshot("agent1"))
	assert.Equal(t, 0, collector.snapshotCount)
	assert.Equal(t, 0, collector.maxContextSize)
	assert.Equal(t, 0, collector.totalDiffSize)
	assert.Equal(t, 0, collector.totalArtifactCount)
}

func TestContextMetricsCollector_ThreadSafety(t *testing.T) {
	collector := NewContextMetricsCollector("session123", "trace456")

	// Simulate concurrent access
	done := make(chan bool)
	numGoroutines := 10
	snapshotsPerGoroutine := 10

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			for j := 0; j < snapshotsPerGoroutine; j++ {
				ctx := &models.AgentContext{
					Reasoning: &models.ReasoningContext{
						Intents: []models.Intent{{Type: "query_commits"}},
						Artifacts: []models.Artifact{{ID: "a1", Type: "report"}},
					},
				}

				timestamp := time.Now().Unix()
				collector.RecordContextSnapshot("agent1", ctx, timestamp)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Verify correct snapshot count
	sessionMetrics := collector.GetContextSessionMetrics()
	assert.Equal(t, numGoroutines*snapshotsPerGoroutine, sessionMetrics.SnapshotCount)
}

func TestContextMetricsCollector_GetAllContextSnapshots_Copy(t *testing.T) {
	collector := NewContextMetricsCollector("session123", "trace456")

	ctx := &models.AgentContext{
		Reasoning: &models.ReasoningContext{
			Intents: []models.Intent{{Type: "query_commits"}},
		},
	}

	timestamp := time.Now().Unix()
	collector.RecordContextSnapshot("agent1", ctx, timestamp)

	// Get snapshots
	snapshots1 := collector.GetAllContextSnapshots()

	// Modify the returned map (should not affect internal state)
	snapshots1["agent1"].ContextSize = 999999

	// Get snapshots again
	snapshots2 := collector.GetAllContextSnapshots()

	// Should still have original value
	assert.NotEqual(t, 999999, snapshots2["agent1"].ContextSize)
}

func TestContextMetricsCollector_GetContextSnapshot_Copy(t *testing.T) {
	collector := NewContextMetricsCollector("session123", "trace456")

	ctx := &models.AgentContext{
		Reasoning: &models.ReasoningContext{
			Intents: []models.Intent{{Type: "query_commits"}},
		},
	}

	timestamp := time.Now().Unix()
	collector.RecordContextSnapshot("agent1", ctx, timestamp)

	// Get snapshot
	snapshot1 := collector.GetContextSnapshot("agent1")

	// Modify the returned struct (should not affect internal state)
	snapshot1.ContextSize = 999999

	// Get snapshot again
	snapshot2 := collector.GetContextSnapshot("agent1")

	// Should still have original value
	assert.NotEqual(t, 999999, snapshot2.ContextSize)
}
