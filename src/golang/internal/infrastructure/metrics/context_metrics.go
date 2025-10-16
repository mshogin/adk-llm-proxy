package metrics

import (
	"encoding/json"
	"sync"

	"github.com/mshogin/agents/internal/domain/models"
)

// ContextMetricsCollector tracks context-related metrics including size,
// artifact count, and diff sizes.
//
// Design Principles:
// - Thread-safe metric collection
// - Track context growth over agent execution
// - Monitor diff sizes between agent runs
// - Detect context bloat and artifact proliferation
type ContextMetricsCollector struct {
	sessionID string
	traceID   string

	// Context snapshots by agent
	snapshots map[string]*ContextSnapshot // key: agent_id
	mu        sync.RWMutex

	// Session-level totals
	totalDiffSize       int
	totalArtifactCount  int
	maxContextSize      int
	snapshotCount       int
}

// ContextSnapshot captures context metrics at a specific point in time.
type ContextSnapshot struct {
	AgentID         string
	ContextSize     int // bytes
	ArtifactCount   int
	FactCount       int
	IntentCount     int
	ConclusionCount int
	ErrorCount      int
	WarningCount    int
	DiffSize        int // size of changes from previous snapshot
	Timestamp       int64
}

// ContextGrowth tracks how context grows over time.
type ContextGrowth struct {
	InitialSize int
	FinalSize   int
	GrowthBytes int
	GrowthRatio float64 // final / initial
}

// ArtifactGrowth tracks artifact accumulation.
type ArtifactGrowth struct {
	InitialCount int
	FinalCount   int
	AddedCount   int
	GrowthRatio  float64 // final / initial
}

// ContextSessionMetrics represents session-level context metrics.
type ContextSessionMetrics struct {
	SessionID           string
	TraceID             string
	MaxContextSize      int
	TotalDiffSize       int
	TotalArtifactCount  int
	SnapshotCount       int
	ContextGrowth       *ContextGrowth
	ArtifactGrowth      *ArtifactGrowth
	ByAgent             map[string]*ContextSnapshot
}

// NewContextMetricsCollector creates a new context metrics collector.
func NewContextMetricsCollector(sessionID, traceID string) *ContextMetricsCollector {
	return &ContextMetricsCollector{
		sessionID: sessionID,
		traceID:   traceID,
		snapshots: make(map[string]*ContextSnapshot),
	}
}

// RecordContextSnapshot captures context state after an agent execution.
func (c *ContextMetricsCollector) RecordContextSnapshot(
	agentID string,
	context *models.AgentContext,
	timestamp int64,
) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Calculate context size (serialize to JSON and measure bytes)
	contextBytes, _ := json.Marshal(context)
	contextSize := len(contextBytes)

	// Count artifacts
	artifactCount := 0
	if context.Reasoning != nil && context.Reasoning.Artifacts != nil {
		artifactCount = len(context.Reasoning.Artifacts)
	}

	// Count facts
	factCount := 0
	if context.Enrichment != nil && context.Enrichment.Facts != nil {
		factCount = len(context.Enrichment.Facts)
	}

	// Count intents
	intentCount := 0
	if context.Reasoning != nil && context.Reasoning.Intents != nil {
		intentCount = len(context.Reasoning.Intents)
	}

	// Count conclusions
	conclusionCount := 0
	if context.Reasoning != nil && context.Reasoning.Conclusions != nil {
		conclusionCount = len(context.Reasoning.Conclusions)
	}

	// Count errors and warnings
	errorCount := 0
	warningCount := 0
	if context.Diagnostics != nil {
		if context.Diagnostics.Errors != nil {
			errorCount = len(context.Diagnostics.Errors)
		}
		if context.Diagnostics.Warnings != nil {
			warningCount = len(context.Diagnostics.Warnings)
		}
	}

	// Calculate diff size (compare with previous snapshot for this agent)
	diffSize := 0
	if prevSnapshot, exists := c.snapshots[agentID]; exists {
		diffSize = contextSize - prevSnapshot.ContextSize
		if diffSize < 0 {
			diffSize = 0 // context shrunk, count as 0 diff
		}
	} else {
		diffSize = contextSize // first snapshot, entire context is "diff"
	}

	// Create snapshot
	snapshot := &ContextSnapshot{
		AgentID:         agentID,
		ContextSize:     contextSize,
		ArtifactCount:   artifactCount,
		FactCount:       factCount,
		IntentCount:     intentCount,
		ConclusionCount: conclusionCount,
		ErrorCount:      errorCount,
		WarningCount:    warningCount,
		DiffSize:        diffSize,
		Timestamp:       timestamp,
	}

	c.snapshots[agentID] = snapshot
	c.snapshotCount++
	c.totalDiffSize += diffSize
	c.totalArtifactCount = artifactCount // track latest total

	// Update max context size
	if contextSize > c.maxContextSize {
		c.maxContextSize = contextSize
	}
}

// GetContextSnapshot returns metrics snapshot for a specific agent.
func (c *ContextMetricsCollector) GetContextSnapshot(agentID string) *ContextSnapshot {
	c.mu.RLock()
	defer c.mu.RUnlock()

	snapshot, exists := c.snapshots[agentID]
	if !exists {
		return nil
	}

	// Return a copy to avoid race conditions
	copy := *snapshot
	return &copy
}

// GetAllContextSnapshots returns all context snapshots.
func (c *ContextMetricsCollector) GetAllContextSnapshots() map[string]*ContextSnapshot {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Return deep copy to avoid race conditions
	result := make(map[string]*ContextSnapshot, len(c.snapshots))
	for agentID, snapshot := range c.snapshots {
		copy := *snapshot
		result[agentID] = &copy
	}

	return result
}

// GetContextGrowth calculates context size growth from first to last snapshot.
func (c *ContextMetricsCollector) GetContextGrowth() *ContextGrowth {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if len(c.snapshots) == 0 {
		return &ContextGrowth{}
	}

	// Find first and last snapshots by timestamp
	var firstSnapshot, lastSnapshot *ContextSnapshot
	var minTimestamp, maxTimestamp int64 = -1, -1

	for _, snapshot := range c.snapshots {
		if minTimestamp == -1 || snapshot.Timestamp < minTimestamp {
			minTimestamp = snapshot.Timestamp
			firstSnapshot = snapshot
		}
		if maxTimestamp == -1 || snapshot.Timestamp > maxTimestamp {
			maxTimestamp = snapshot.Timestamp
			lastSnapshot = snapshot
		}
	}

	if firstSnapshot == nil || lastSnapshot == nil {
		return &ContextGrowth{}
	}

	initialSize := firstSnapshot.ContextSize
	finalSize := lastSnapshot.ContextSize
	growthBytes := finalSize - initialSize

	growthRatio := 0.0
	if initialSize > 0 {
		growthRatio = float64(finalSize) / float64(initialSize)
	}

	return &ContextGrowth{
		InitialSize: initialSize,
		FinalSize:   finalSize,
		GrowthBytes: growthBytes,
		GrowthRatio: growthRatio,
	}
}

// GetArtifactGrowth calculates artifact count growth from first to last snapshot.
func (c *ContextMetricsCollector) GetArtifactGrowth() *ArtifactGrowth {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if len(c.snapshots) == 0 {
		return &ArtifactGrowth{}
	}

	// Find first and last snapshots by timestamp
	var firstSnapshot, lastSnapshot *ContextSnapshot
	var minTimestamp, maxTimestamp int64 = -1, -1

	for _, snapshot := range c.snapshots {
		if minTimestamp == -1 || snapshot.Timestamp < minTimestamp {
			minTimestamp = snapshot.Timestamp
			firstSnapshot = snapshot
		}
		if maxTimestamp == -1 || snapshot.Timestamp > maxTimestamp {
			maxTimestamp = snapshot.Timestamp
			lastSnapshot = snapshot
		}
	}

	if firstSnapshot == nil || lastSnapshot == nil {
		return &ArtifactGrowth{}
	}

	initialCount := firstSnapshot.ArtifactCount
	finalCount := lastSnapshot.ArtifactCount
	addedCount := finalCount - initialCount

	growthRatio := 0.0
	if initialCount > 0 {
		growthRatio = float64(finalCount) / float64(initialCount)
	} else if finalCount > 0 {
		growthRatio = float64(finalCount) // started from 0
	}

	return &ArtifactGrowth{
		InitialCount: initialCount,
		FinalCount:   finalCount,
		AddedCount:   addedCount,
		GrowthRatio:  growthRatio,
	}
}

// GetContextSessionMetrics returns session-level context metrics.
func (c *ContextMetricsCollector) GetContextSessionMetrics() ContextSessionMetrics {
	c.mu.RLock()
	defer c.mu.RUnlock()

	contextGrowth := c.getContextGrowthUnsafe()
	artifactGrowth := c.getArtifactGrowthUnsafe()

	// Create by-agent snapshots copy
	byAgent := make(map[string]*ContextSnapshot, len(c.snapshots))
	for agentID, snapshot := range c.snapshots {
		copy := *snapshot
		byAgent[agentID] = &copy
	}

	return ContextSessionMetrics{
		SessionID:          c.sessionID,
		TraceID:            c.traceID,
		MaxContextSize:     c.maxContextSize,
		TotalDiffSize:      c.totalDiffSize,
		TotalArtifactCount: c.totalArtifactCount,
		SnapshotCount:      c.snapshotCount,
		ContextGrowth:      contextGrowth,
		ArtifactGrowth:     artifactGrowth,
		ByAgent:            byAgent,
	}
}

// getContextGrowthUnsafe returns context growth without locking (internal use only).
func (c *ContextMetricsCollector) getContextGrowthUnsafe() *ContextGrowth {
	if len(c.snapshots) == 0 {
		return &ContextGrowth{}
	}

	// Find first and last snapshots by timestamp
	var firstSnapshot, lastSnapshot *ContextSnapshot
	var minTimestamp, maxTimestamp int64 = -1, -1

	for _, snapshot := range c.snapshots {
		if minTimestamp == -1 || snapshot.Timestamp < minTimestamp {
			minTimestamp = snapshot.Timestamp
			firstSnapshot = snapshot
		}
		if maxTimestamp == -1 || snapshot.Timestamp > maxTimestamp {
			maxTimestamp = snapshot.Timestamp
			lastSnapshot = snapshot
		}
	}

	if firstSnapshot == nil || lastSnapshot == nil {
		return &ContextGrowth{}
	}

	initialSize := firstSnapshot.ContextSize
	finalSize := lastSnapshot.ContextSize
	growthBytes := finalSize - initialSize

	growthRatio := 0.0
	if initialSize > 0 {
		growthRatio = float64(finalSize) / float64(initialSize)
	}

	return &ContextGrowth{
		InitialSize: initialSize,
		FinalSize:   finalSize,
		GrowthBytes: growthBytes,
		GrowthRatio: growthRatio,
	}
}

// getArtifactGrowthUnsafe returns artifact growth without locking (internal use only).
func (c *ContextMetricsCollector) getArtifactGrowthUnsafe() *ArtifactGrowth {
	if len(c.snapshots) == 0 {
		return &ArtifactGrowth{}
	}

	// Find first and last snapshots by timestamp
	var firstSnapshot, lastSnapshot *ContextSnapshot
	var minTimestamp, maxTimestamp int64 = -1, -1

	for _, snapshot := range c.snapshots {
		if minTimestamp == -1 || snapshot.Timestamp < minTimestamp {
			minTimestamp = snapshot.Timestamp
			firstSnapshot = snapshot
		}
		if maxTimestamp == -1 || snapshot.Timestamp > maxTimestamp {
			maxTimestamp = snapshot.Timestamp
			lastSnapshot = snapshot
		}
	}

	if firstSnapshot == nil || lastSnapshot == nil {
		return &ArtifactGrowth{}
	}

	initialCount := firstSnapshot.ArtifactCount
	finalCount := lastSnapshot.ArtifactCount
	addedCount := finalCount - initialCount

	growthRatio := 0.0
	if initialCount > 0 {
		growthRatio = float64(finalCount) / float64(initialCount)
	} else if finalCount > 0 {
		growthRatio = float64(finalCount) // started from 0
	}

	return &ArtifactGrowth{
		InitialCount: initialCount,
		FinalCount:   finalCount,
		AddedCount:   addedCount,
		GrowthRatio:  growthRatio,
	}
}

// Reset resets all context metrics (useful for testing or session restart).
func (c *ContextMetricsCollector) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.snapshots = make(map[string]*ContextSnapshot)
	c.totalDiffSize = 0
	c.totalArtifactCount = 0
	c.maxContextSize = 0
	c.snapshotCount = 0
}
