package metrics

import (
	"runtime"
	"sync"
	"time"
)

// PerformanceProfiler tracks detailed performance metrics per agent.
//
// Design Principles:
// - Track CPU time, memory usage, and latency per agent
// - Support nested profiling (agent → sub-operations)
// - Aggregate statistics across multiple executions
// - Identify performance bottlenecks
// - Thread-safe profiling
type PerformanceProfiler struct {
	sessionID string
	traceID   string

	// Agent profiles
	profiles map[string]*AgentProfile
	mu       sync.RWMutex

	// Session-level stats
	startTime      time.Time
	totalDurationMS int64
}

// AgentProfile tracks performance metrics for a single agent.
type AgentProfile struct {
	AgentID string

	// Execution statistics
	ExecutionCount int
	TotalDurationMS int64
	MinDurationMS   int64
	MaxDurationMS   int64
	AvgDurationMS   int64

	// Memory statistics
	TotalMemoryAllocBytes int64
	AvgMemoryAllocBytes   int64
	MaxMemoryAllocBytes   int64

	// CPU statistics (goroutine count as proxy)
	AvgGoroutineCount int
	MaxGoroutineCount int

	// Operation breakdown
	Operations map[string]*OperationProfile

	// Error tracking
	ErrorCount  int
	LastError   string
	LastErrorAt int64
}

// OperationProfile tracks performance of a specific operation within an agent.
type OperationProfile struct {
	OperationName string
	ExecutionCount int
	TotalDurationMS int64
	AvgDurationMS   int64
	MinDurationMS   int64
	MaxDurationMS   int64
}

// ProfileSnapshot represents a point-in-time performance measurement.
type ProfileSnapshot struct {
	Timestamp       int64
	DurationMS      int64
	MemoryAllocBytes int64
	GoroutineCount  int
	Error           error
}

// PerformanceReport represents a complete performance profile.
type PerformanceReport struct {
	SessionID       string
	TraceID         string
	TotalDurationMS int64
	AgentProfiles   []*AgentProfile
	TopBottlenecks  []*AgentProfile // Top 5 slowest agents
	Duration        time.Duration
}

// NewPerformanceProfiler creates a new performance profiler.
func NewPerformanceProfiler(sessionID, traceID string) *PerformanceProfiler {
	return &PerformanceProfiler{
		sessionID: sessionID,
		traceID:   traceID,
		profiles:  make(map[string]*AgentProfile),
		startTime: time.Now(),
	}
}

// StartProfile begins profiling an agent execution.
// Returns a function to be called when the operation completes.
func (p *PerformanceProfiler) StartProfile(agentID string) func(error) {
	startTime := time.Now()
	var memBefore runtime.MemStats
	runtime.ReadMemStats(&memBefore)
	goroutinesBefore := runtime.NumGoroutine()

	return func(err error) {
		endTime := time.Now()
		var memAfter runtime.MemStats
		runtime.ReadMemStats(&memAfter)
		goroutinesAfter := runtime.NumGoroutine()

		snapshot := ProfileSnapshot{
			Timestamp:        endTime.Unix(),
			DurationMS:       endTime.Sub(startTime).Milliseconds(),
			MemoryAllocBytes: int64(memAfter.Alloc - memBefore.Alloc),
			GoroutineCount:   goroutinesAfter - goroutinesBefore,
			Error:            err,
		}

		p.RecordProfile(agentID, snapshot)
	}
}

// RecordProfile records a performance snapshot for an agent.
func (p *PerformanceProfiler) RecordProfile(agentID string, snapshot ProfileSnapshot) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Get or create profile
	profile, exists := p.profiles[agentID]
	if !exists {
		profile = &AgentProfile{
			AgentID:    agentID,
			Operations: make(map[string]*OperationProfile),
			MinDurationMS: snapshot.DurationMS,
			MaxDurationMS: snapshot.DurationMS,
		}
		p.profiles[agentID] = profile
	}

	// Update execution count
	profile.ExecutionCount++

	// Update duration statistics
	profile.TotalDurationMS += snapshot.DurationMS
	if snapshot.DurationMS < profile.MinDurationMS {
		profile.MinDurationMS = snapshot.DurationMS
	}
	if snapshot.DurationMS > profile.MaxDurationMS {
		profile.MaxDurationMS = snapshot.DurationMS
	}
	profile.AvgDurationMS = profile.TotalDurationMS / int64(profile.ExecutionCount)

	// Update memory statistics
	if snapshot.MemoryAllocBytes > 0 {
		profile.TotalMemoryAllocBytes += snapshot.MemoryAllocBytes
		profile.AvgMemoryAllocBytes = profile.TotalMemoryAllocBytes / int64(profile.ExecutionCount)
		if snapshot.MemoryAllocBytes > profile.MaxMemoryAllocBytes {
			profile.MaxMemoryAllocBytes = snapshot.MemoryAllocBytes
		}
	}

	// Update goroutine statistics
	if snapshot.GoroutineCount > 0 {
		// Running average
		profile.AvgGoroutineCount = (profile.AvgGoroutineCount*(profile.ExecutionCount-1) + snapshot.GoroutineCount) / profile.ExecutionCount
		if snapshot.GoroutineCount > profile.MaxGoroutineCount {
			profile.MaxGoroutineCount = snapshot.GoroutineCount
		}
	}

	// Update error tracking
	if snapshot.Error != nil {
		profile.ErrorCount++
		profile.LastError = snapshot.Error.Error()
		profile.LastErrorAt = snapshot.Timestamp
	}

	// Update session total
	p.totalDurationMS += snapshot.DurationMS
}

// RecordOperation records a performance snapshot for a specific operation within an agent.
func (p *PerformanceProfiler) RecordOperation(agentID, operationName string, durationMS int64) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Get or create agent profile
	profile, exists := p.profiles[agentID]
	if !exists {
		profile = &AgentProfile{
			AgentID:    agentID,
			Operations: make(map[string]*OperationProfile),
		}
		p.profiles[agentID] = profile
	}

	// Get or create operation profile
	op, exists := profile.Operations[operationName]
	if !exists {
		op = &OperationProfile{
			OperationName: operationName,
			MinDurationMS: durationMS,
			MaxDurationMS: durationMS,
		}
		profile.Operations[operationName] = op
	}

	// Update operation statistics
	op.ExecutionCount++
	op.TotalDurationMS += durationMS
	if durationMS < op.MinDurationMS {
		op.MinDurationMS = durationMS
	}
	if durationMS > op.MaxDurationMS {
		op.MaxDurationMS = durationMS
	}
	op.AvgDurationMS = op.TotalDurationMS / int64(op.ExecutionCount)
}

// GetAgentProfile returns the performance profile for a specific agent.
func (p *PerformanceProfiler) GetAgentProfile(agentID string) *AgentProfile {
	p.mu.RLock()
	defer p.mu.RUnlock()

	profile, exists := p.profiles[agentID]
	if !exists {
		return nil
	}

	// Return a deep copy
	copy := *profile
	copy.Operations = make(map[string]*OperationProfile)
	for k, v := range profile.Operations {
		opCopy := *v
		copy.Operations[k] = &opCopy
	}

	return &copy
}

// GetAllProfiles returns all agent profiles.
func (p *PerformanceProfiler) GetAllProfiles() []*AgentProfile {
	p.mu.RLock()
	defer p.mu.RUnlock()

	profiles := make([]*AgentProfile, 0, len(p.profiles))
	for _, profile := range p.profiles {
		// Deep copy
		copy := *profile
		copy.Operations = make(map[string]*OperationProfile)
		for k, v := range profile.Operations {
			opCopy := *v
			copy.Operations[k] = &opCopy
		}
		profiles = append(profiles, &copy)
	}

	return profiles
}

// GetPerformanceReport generates a complete performance report.
func (p *PerformanceProfiler) GetPerformanceReport() *PerformanceReport {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// Collect all profiles
	profiles := make([]*AgentProfile, 0, len(p.profiles))
	for _, profile := range p.profiles {
		// Deep copy
		copy := *profile
		copy.Operations = make(map[string]*OperationProfile)
		for k, v := range profile.Operations {
			opCopy := *v
			copy.Operations[k] = &opCopy
		}
		profiles = append(profiles, &copy)
	}

	// Find top bottlenecks (top 5 by avg duration)
	topBottlenecks := p.getTopBottlenecksUnsafe(5)

	// Calculate duration
	duration := time.Since(p.startTime)

	return &PerformanceReport{
		SessionID:       p.sessionID,
		TraceID:         p.traceID,
		TotalDurationMS: p.totalDurationMS,
		AgentProfiles:   profiles,
		TopBottlenecks:  topBottlenecks,
		Duration:        duration,
	}
}

// getTopBottlenecksUnsafe returns top N agents by average duration without locking.
func (p *PerformanceProfiler) getTopBottlenecksUnsafe(n int) []*AgentProfile {
	// Collect all profiles
	profiles := make([]*AgentProfile, 0, len(p.profiles))
	for _, profile := range p.profiles {
		// Deep copy
		copy := *profile
		copy.Operations = make(map[string]*OperationProfile)
		for k, v := range profile.Operations {
			opCopy := *v
			copy.Operations[k] = &opCopy
		}
		profiles = append(profiles, &copy)
	}

	// Sort by average duration (descending) using simple bubble sort
	for i := 0; i < len(profiles); i++ {
		for j := i + 1; j < len(profiles); j++ {
			if profiles[j].AvgDurationMS > profiles[i].AvgDurationMS {
				profiles[i], profiles[j] = profiles[j], profiles[i]
			}
		}
	}

	// Return top N
	if len(profiles) > n {
		return profiles[:n]
	}

	return profiles
}

// GetBottlenecks returns agents with average execution time above threshold (in milliseconds).
func (p *PerformanceProfiler) GetBottlenecks(thresholdMS int64) []*AgentProfile {
	p.mu.RLock()
	defer p.mu.RUnlock()

	bottlenecks := []*AgentProfile{}

	for _, profile := range p.profiles {
		if profile.AvgDurationMS >= thresholdMS {
			// Deep copy
			copy := *profile
			copy.Operations = make(map[string]*OperationProfile)
			for k, v := range profile.Operations {
				opCopy := *v
				copy.Operations[k] = &opCopy
			}
			bottlenecks = append(bottlenecks, &copy)
		}
	}

	return bottlenecks
}

// GetMemoryHogs returns agents with average memory allocation above threshold (in bytes).
func (p *PerformanceProfiler) GetMemoryHogs(thresholdBytes int64) []*AgentProfile {
	p.mu.RLock()
	defer p.mu.RUnlock()

	memoryHogs := []*AgentProfile{}

	for _, profile := range p.profiles {
		if profile.AvgMemoryAllocBytes >= thresholdBytes {
			// Deep copy
			copy := *profile
			copy.Operations = make(map[string]*OperationProfile)
			for k, v := range profile.Operations {
				opCopy := *v
				copy.Operations[k] = &opCopy
			}
			memoryHogs = append(memoryHogs, &copy)
		}
	}

	return memoryHogs
}

// Reset resets the profiler (useful for testing).
func (p *PerformanceProfiler) Reset() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.profiles = make(map[string]*AgentProfile)
	p.totalDurationMS = 0
	p.startTime = time.Now()
}
