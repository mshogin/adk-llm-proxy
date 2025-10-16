package metrics

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPerformanceProfiler(t *testing.T) {
	profiler := NewPerformanceProfiler("session123", "trace456")

	assert.Equal(t, "session123", profiler.sessionID)
	assert.Equal(t, "trace456", profiler.traceID)
	assert.NotNil(t, profiler.profiles)
	assert.NotZero(t, profiler.startTime)
}

func TestPerformanceProfiler_RecordProfile_SingleAgent(t *testing.T) {
	profiler := NewPerformanceProfiler("session123", "trace456")

	snapshot := ProfileSnapshot{
		Timestamp:        time.Now().Unix(),
		DurationMS:       150,
		MemoryAllocBytes: 1024,
		GoroutineCount:   2,
		Error:            nil,
	}

	profiler.RecordProfile("intent_detection", snapshot)

	profile := profiler.GetAgentProfile("intent_detection")
	require.NotNil(t, profile)

	assert.Equal(t, "intent_detection", profile.AgentID)
	assert.Equal(t, 1, profile.ExecutionCount)
	assert.Equal(t, int64(150), profile.TotalDurationMS)
	assert.Equal(t, int64(150), profile.AvgDurationMS)
	assert.Equal(t, int64(150), profile.MinDurationMS)
	assert.Equal(t, int64(150), profile.MaxDurationMS)
	assert.Equal(t, int64(1024), profile.TotalMemoryAllocBytes)
	assert.Equal(t, int64(1024), profile.AvgMemoryAllocBytes)
	assert.Equal(t, int64(1024), profile.MaxMemoryAllocBytes)
	assert.Equal(t, 2, profile.AvgGoroutineCount)
	assert.Equal(t, 2, profile.MaxGoroutineCount)
	assert.Equal(t, 0, profile.ErrorCount)
}

func TestPerformanceProfiler_RecordProfile_MultipleExecutions(t *testing.T) {
	profiler := NewPerformanceProfiler("session123", "trace456")

	profiler.RecordProfile("inference", ProfileSnapshot{
		DurationMS:       100,
		MemoryAllocBytes: 1000,
		GoroutineCount:   1,
	})

	profiler.RecordProfile("inference", ProfileSnapshot{
		DurationMS:       200,
		MemoryAllocBytes: 2000,
		GoroutineCount:   3,
	})

	profiler.RecordProfile("inference", ProfileSnapshot{
		DurationMS:       150,
		MemoryAllocBytes: 1500,
		GoroutineCount:   2,
	})

	profile := profiler.GetAgentProfile("inference")
	require.NotNil(t, profile)

	assert.Equal(t, 3, profile.ExecutionCount)
	assert.Equal(t, int64(450), profile.TotalDurationMS) // 100 + 200 + 150
	assert.Equal(t, int64(150), profile.AvgDurationMS)   // 450 / 3
	assert.Equal(t, int64(100), profile.MinDurationMS)
	assert.Equal(t, int64(200), profile.MaxDurationMS)
	assert.Equal(t, int64(4500), profile.TotalMemoryAllocBytes) // 1000 + 2000 + 1500
	assert.Equal(t, int64(1500), profile.AvgMemoryAllocBytes)   // 4500 / 3
	assert.Equal(t, int64(2000), profile.MaxMemoryAllocBytes)
	assert.Equal(t, 2, profile.AvgGoroutineCount) // (1 + 3 + 2) / 3 = 2
	assert.Equal(t, 3, profile.MaxGoroutineCount)
}

func TestPerformanceProfiler_RecordProfile_WithError(t *testing.T) {
	profiler := NewPerformanceProfiler("session123", "trace456")

	testErr := errors.New("test error")
	snapshot := ProfileSnapshot{
		Timestamp:  time.Now().Unix(),
		DurationMS: 150,
		Error:      testErr,
	}

	profiler.RecordProfile("validation", snapshot)

	profile := profiler.GetAgentProfile("validation")
	require.NotNil(t, profile)

	assert.Equal(t, 1, profile.ErrorCount)
	assert.Equal(t, "test error", profile.LastError)
	assert.NotZero(t, profile.LastErrorAt)
}

func TestPerformanceProfiler_StartProfile(t *testing.T) {
	profiler := NewPerformanceProfiler("session123", "trace456")

	done := profiler.StartProfile("intent_detection")
	time.Sleep(10 * time.Millisecond)
	done(nil)

	profile := profiler.GetAgentProfile("intent_detection")
	require.NotNil(t, profile)

	assert.Equal(t, 1, profile.ExecutionCount)
	assert.GreaterOrEqual(t, profile.TotalDurationMS, int64(10))
	assert.Equal(t, 0, profile.ErrorCount)
}

func TestPerformanceProfiler_StartProfile_WithError(t *testing.T) {
	profiler := NewPerformanceProfiler("session123", "trace456")

	done := profiler.StartProfile("inference")
	time.Sleep(5 * time.Millisecond)
	testErr := errors.New("inference failed")
	done(testErr)

	profile := profiler.GetAgentProfile("inference")
	require.NotNil(t, profile)

	assert.Equal(t, 1, profile.ExecutionCount)
	assert.GreaterOrEqual(t, profile.TotalDurationMS, int64(5))
	assert.Equal(t, 1, profile.ErrorCount)
	assert.Equal(t, "inference failed", profile.LastError)
}

func TestPerformanceProfiler_RecordOperation(t *testing.T) {
	profiler := NewPerformanceProfiler("session123", "trace456")

	profiler.RecordOperation("intent_detection", "regex_matching", 10)
	profiler.RecordOperation("intent_detection", "entity_extraction", 20)
	profiler.RecordOperation("intent_detection", "regex_matching", 15)

	profile := profiler.GetAgentProfile("intent_detection")
	require.NotNil(t, profile)

	assert.Len(t, profile.Operations, 2)

	regexOp := profile.Operations["regex_matching"]
	require.NotNil(t, regexOp)
	assert.Equal(t, "regex_matching", regexOp.OperationName)
	assert.Equal(t, 2, regexOp.ExecutionCount)
	assert.Equal(t, int64(25), regexOp.TotalDurationMS) // 10 + 15
	assert.Equal(t, int64(12), regexOp.AvgDurationMS)   // 25 / 2 = 12.5 truncated
	assert.Equal(t, int64(10), regexOp.MinDurationMS)
	assert.Equal(t, int64(15), regexOp.MaxDurationMS)

	entityOp := profile.Operations["entity_extraction"]
	require.NotNil(t, entityOp)
	assert.Equal(t, "entity_extraction", entityOp.OperationName)
	assert.Equal(t, 1, entityOp.ExecutionCount)
	assert.Equal(t, int64(20), entityOp.TotalDurationMS)
}

func TestPerformanceProfiler_GetAgentProfile_NonExistent(t *testing.T) {
	profiler := NewPerformanceProfiler("session123", "trace456")

	profile := profiler.GetAgentProfile("nonexistent")
	assert.Nil(t, profile)
}

func TestPerformanceProfiler_GetAllProfiles(t *testing.T) {
	profiler := NewPerformanceProfiler("session123", "trace456")

	profiler.RecordProfile("intent_detection", ProfileSnapshot{DurationMS: 100})
	profiler.RecordProfile("inference", ProfileSnapshot{DurationMS: 200})
	profiler.RecordProfile("validation", ProfileSnapshot{DurationMS: 50})

	profiles := profiler.GetAllProfiles()
	assert.Len(t, profiles, 3)

	// Verify all agents present
	agentIDs := make(map[string]bool)
	for _, profile := range profiles {
		agentIDs[profile.AgentID] = true
	}

	assert.True(t, agentIDs["intent_detection"])
	assert.True(t, agentIDs["inference"])
	assert.True(t, agentIDs["validation"])
}

func TestPerformanceProfiler_GetPerformanceReport(t *testing.T) {
	profiler := NewPerformanceProfiler("session123", "trace456")

	profiler.RecordProfile("intent_detection", ProfileSnapshot{DurationMS: 100})
	profiler.RecordProfile("inference", ProfileSnapshot{DurationMS: 200})
	profiler.RecordProfile("validation", ProfileSnapshot{DurationMS: 50})

	time.Sleep(10 * time.Millisecond)

	report := profiler.GetPerformanceReport()

	assert.Equal(t, "session123", report.SessionID)
	assert.Equal(t, "trace456", report.TraceID)
	assert.Equal(t, int64(350), report.TotalDurationMS) // 100 + 200 + 50
	assert.Len(t, report.AgentProfiles, 3)
	assert.Greater(t, report.Duration, time.Duration(0))
	assert.GreaterOrEqual(t, report.Duration.Milliseconds(), int64(10))
}

func TestPerformanceProfiler_GetPerformanceReport_TopBottlenecks(t *testing.T) {
	profiler := NewPerformanceProfiler("session123", "trace456")

	// Add 7 agents with different average durations
	profiler.RecordProfile("agent1", ProfileSnapshot{DurationMS: 100})
	profiler.RecordProfile("agent2", ProfileSnapshot{DurationMS: 50})
	profiler.RecordProfile("agent3", ProfileSnapshot{DurationMS: 200})
	profiler.RecordProfile("agent4", ProfileSnapshot{DurationMS: 30})
	profiler.RecordProfile("agent5", ProfileSnapshot{DurationMS: 150})
	profiler.RecordProfile("agent6", ProfileSnapshot{DurationMS: 80})
	profiler.RecordProfile("agent7", ProfileSnapshot{DurationMS: 10})

	report := profiler.GetPerformanceReport()

	assert.Len(t, report.TopBottlenecks, 5) // Top 5

	// Verify ordering (descending by average duration)
	assert.Equal(t, "agent3", report.TopBottlenecks[0].AgentID) // 200
	assert.Equal(t, "agent5", report.TopBottlenecks[1].AgentID) // 150
	assert.Equal(t, "agent1", report.TopBottlenecks[2].AgentID) // 100
	assert.Equal(t, "agent6", report.TopBottlenecks[3].AgentID) // 80
	assert.Equal(t, "agent2", report.TopBottlenecks[4].AgentID) // 50
}

func TestPerformanceProfiler_GetPerformanceReport_TopBottlenecks_LessThan5(t *testing.T) {
	profiler := NewPerformanceProfiler("session123", "trace456")

	profiler.RecordProfile("agent1", ProfileSnapshot{DurationMS: 100})
	profiler.RecordProfile("agent2", ProfileSnapshot{DurationMS: 50})

	report := profiler.GetPerformanceReport()

	assert.Len(t, report.TopBottlenecks, 2)
}

func TestPerformanceProfiler_GetBottlenecks(t *testing.T) {
	profiler := NewPerformanceProfiler("session123", "trace456")

	profiler.RecordProfile("fast_agent", ProfileSnapshot{DurationMS: 50})
	profiler.RecordProfile("slow_agent1", ProfileSnapshot{DurationMS: 500})
	profiler.RecordProfile("slow_agent2", ProfileSnapshot{DurationMS: 800})

	bottlenecks := profiler.GetBottlenecks(100) // Threshold: 100ms

	assert.Len(t, bottlenecks, 2)

	// Verify only slow agents included
	agentIDs := make(map[string]bool)
	for _, profile := range bottlenecks {
		agentIDs[profile.AgentID] = true
	}

	assert.False(t, agentIDs["fast_agent"])
	assert.True(t, agentIDs["slow_agent1"])
	assert.True(t, agentIDs["slow_agent2"])
}

func TestPerformanceProfiler_GetMemoryHogs(t *testing.T) {
	profiler := NewPerformanceProfiler("session123", "trace456")

	profiler.RecordProfile("low_memory", ProfileSnapshot{
		DurationMS:       100,
		MemoryAllocBytes: 512,
	})

	profiler.RecordProfile("high_memory1", ProfileSnapshot{
		DurationMS:       100,
		MemoryAllocBytes: 10240,
	})

	profiler.RecordProfile("high_memory2", ProfileSnapshot{
		DurationMS:       100,
		MemoryAllocBytes: 20480,
	})

	memoryHogs := profiler.GetMemoryHogs(1024) // Threshold: 1KB

	assert.Len(t, memoryHogs, 2)

	// Verify only high-memory agents included
	agentIDs := make(map[string]bool)
	for _, profile := range memoryHogs {
		agentIDs[profile.AgentID] = true
	}

	assert.False(t, agentIDs["low_memory"])
	assert.True(t, agentIDs["high_memory1"])
	assert.True(t, agentIDs["high_memory2"])
}

func TestPerformanceProfiler_Reset(t *testing.T) {
	profiler := NewPerformanceProfiler("session123", "trace456")

	profiler.RecordProfile("intent_detection", ProfileSnapshot{DurationMS: 100})
	profiler.RecordProfile("inference", ProfileSnapshot{DurationMS: 200})

	assert.Equal(t, int64(300), profiler.totalDurationMS)
	assert.Len(t, profiler.profiles, 2)

	profiler.Reset()

	assert.Zero(t, profiler.totalDurationMS)
	assert.Empty(t, profiler.profiles)
	assert.NotZero(t, profiler.startTime)
}

func TestPerformanceProfiler_ThreadSafety(t *testing.T) {
	profiler := NewPerformanceProfiler("session123", "trace456")

	done := make(chan bool)
	numGoroutines := 10
	snapshotsPerGoroutine := 100

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			for j := 0; j < snapshotsPerGoroutine; j++ {
				profiler.RecordProfile("agent1", ProfileSnapshot{
					DurationMS:       int64(10 + id*j),
					MemoryAllocBytes: int64(1024 + id*j),
					GoroutineCount:   1,
				})
				profiler.RecordProfile("agent2", ProfileSnapshot{
					DurationMS:       int64(20 + id*j),
					MemoryAllocBytes: int64(2048 + id*j),
					GoroutineCount:   2,
				})
				profiler.RecordOperation("agent1", "op1", int64(5+id*j))
			}
			done <- true
		}(i)
	}

	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	report := profiler.GetPerformanceReport()
	assert.Len(t, report.AgentProfiles, 2)

	agent1 := profiler.GetAgentProfile("agent1")
	agent2 := profiler.GetAgentProfile("agent2")

	assert.Equal(t, numGoroutines*snapshotsPerGoroutine, agent1.ExecutionCount)
	assert.Equal(t, numGoroutines*snapshotsPerGoroutine, agent2.ExecutionCount)
}

func TestPerformanceProfiler_GetAgentProfile_Copy(t *testing.T) {
	profiler := NewPerformanceProfiler("session123", "trace456")

	profiler.RecordProfile("intent_detection", ProfileSnapshot{DurationMS: 100})
	profiler.RecordOperation("intent_detection", "regex", 10)

	profile1 := profiler.GetAgentProfile("intent_detection")
	profile1.TotalDurationMS = 999999 // Modify copy
	profile1.Operations["regex"].TotalDurationMS = 888888 // Modify nested copy

	profile2 := profiler.GetAgentProfile("intent_detection")
	assert.Equal(t, int64(100), profile2.TotalDurationMS) // Original unchanged
	assert.Equal(t, int64(10), profile2.Operations["regex"].TotalDurationMS) // Original unchanged
}

func TestPerformanceProfiler_GetAllProfiles_Copy(t *testing.T) {
	profiler := NewPerformanceProfiler("session123", "trace456")

	profiler.RecordProfile("intent_detection", ProfileSnapshot{DurationMS: 100})

	profiles1 := profiler.GetAllProfiles()
	profiles1[0].TotalDurationMS = 999999 // Modify copy

	profiles2 := profiler.GetAllProfiles()
	assert.Equal(t, int64(100), profiles2[0].TotalDurationMS) // Original unchanged
}

func TestPerformanceProfiler_NegativeMemoryAlloc(t *testing.T) {
	profiler := NewPerformanceProfiler("session123", "trace456")

	// Negative memory allocation (e.g., due to GC) should be ignored
	profiler.RecordProfile("agent1", ProfileSnapshot{
		DurationMS:       100,
		MemoryAllocBytes: -1024,
	})

	profile := profiler.GetAgentProfile("agent1")
	require.NotNil(t, profile)

	assert.Equal(t, int64(0), profile.TotalMemoryAllocBytes)
	assert.Equal(t, int64(0), profile.AvgMemoryAllocBytes)
	assert.Equal(t, int64(0), profile.MaxMemoryAllocBytes)
}

func TestPerformanceProfiler_ZeroGoroutineCount(t *testing.T) {
	profiler := NewPerformanceProfiler("session123", "trace456")

	// Zero or negative goroutine count should be ignored
	profiler.RecordProfile("agent1", ProfileSnapshot{
		DurationMS:     100,
		GoroutineCount: 0,
	})

	profiler.RecordProfile("agent1", ProfileSnapshot{
		DurationMS:     100,
		GoroutineCount: -1,
	})

	profile := profiler.GetAgentProfile("agent1")
	require.NotNil(t, profile)

	assert.Equal(t, 0, profile.AvgGoroutineCount)
	assert.Equal(t, 0, profile.MaxGoroutineCount)
}
