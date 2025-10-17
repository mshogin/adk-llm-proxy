package agents

import (
	"context"
	"crypto/md5"
	"fmt"
	"strings"
	"time"

	"github.com/mshogin/agents/internal/domain/models"
	"github.com/mshogin/agents/internal/domain/services"
)

// ContextSynthesizerAgent normalizes and merges facts from multiple sources.
//
// Design Principles:
// - Normalize facts from different data sources
// - Implement deduplication logic
// - Track fact provenance (source, timestamp, confidence)
// - Derive knowledge from normalized facts
//
// Input Requirements:
// - retrieval.artifacts: Retrieved data from various sources
//
// Output:
// - enrichment.facts[]: Normalized facts with provenance
// - enrichment.derived_knowledge[]: Knowledge derived from facts
// - enrichment.relationships[]: Relationships between entities
type ContextSynthesizerAgent struct {
	id string
}

// NewContextSynthesizerAgent creates a new context synthesizer agent.
func NewContextSynthesizerAgent() *ContextSynthesizerAgent {
	return &ContextSynthesizerAgent{
		id: "context_synthesizer",
	}
}

// AgentID returns the unique identifier for this agent.
func (a *ContextSynthesizerAgent) AgentID() string {
	return a.id
}

// Preconditions returns the list of context keys required before execution.
func (a *ContextSynthesizerAgent) Preconditions() []string {
	return []string{
		"retrieval.artifacts",
	}
}

// Postconditions returns the list of context keys guaranteed after execution.
func (a *ContextSynthesizerAgent) Postconditions() []string {
	return []string{
		"enrichment.facts",
		"enrichment.derived_knowledge",
		"enrichment.relationships",
	}
}

// Execute normalizes and synthesizes facts from retrieval artifacts.
func (a *ContextSynthesizerAgent) Execute(ctx context.Context, agentContext *models.AgentContext) (*models.AgentContext, error) {
	startTime := time.Now()

	// Clone context
	newContext, err := agentContext.Clone()
	if err != nil {
		return nil, fmt.Errorf("failed to clone context: %w", err)
	}

	// Validate preconditions
	if err := a.validatePreconditions(newContext); err != nil {
		return nil, fmt.Errorf("precondition validation failed: %w", err)
	}

	// Extract artifacts
	artifacts := newContext.Retrieval.Artifacts

	// Store detailed agent trace
	agentTrace := map[string]interface{}{
		"agent_id":        a.id,
		"input_artifacts": artifacts,
		"artifacts_count": len(artifacts),
	}

	// Normalize facts from artifacts
	facts := a.normalizeFacts(artifacts)
	agentTrace["normalized_facts_count"] = len(facts)

	// Deduplicate facts
	beforeDedup := len(facts)
	facts = a.deduplicateFacts(facts)
	agentTrace["deduplicated_facts_count"] = len(facts)
	agentTrace["duplicates_removed"] = beforeDedup - len(facts)

	// Derive knowledge from facts
	knowledge := a.deriveKnowledge(facts)
	agentTrace["derived_knowledge_count"] = len(knowledge)

	// Extract relationships
	relationships := a.extractRelationships(facts)
	agentTrace["relationships_count"] = len(relationships)

	// Write results
	newContext.Enrichment.Facts = facts
	newContext.Enrichment.DerivedKnowledge = knowledge
	newContext.Enrichment.Relationships = relationships

	// Store final output in trace
	agentTrace["output_facts"] = facts
	agentTrace["output_knowledge"] = knowledge
	agentTrace["output_relationships"] = relationships

	// Store agent trace in LLM cache
	if newContext.LLM == nil {
		newContext.LLM = &models.LLMContext{
			Cache: make(map[string]interface{}),
		}
	}
	if newContext.LLM.Cache == nil {
		newContext.LLM.Cache = make(map[string]interface{})
	}
	if traces, ok := newContext.LLM.Cache["agent_traces"].([]interface{}); ok {
		newContext.LLM.Cache["agent_traces"] = append(traces, agentTrace)
	} else {
		newContext.LLM.Cache["agent_traces"] = []interface{}{agentTrace}
	}

	// Track execution
	duration := time.Since(startTime)
	a.recordAgentRun(newContext, duration, "success", nil)

	return newContext, nil
}

// validatePreconditions checks required context keys.
func (a *ContextSynthesizerAgent) validatePreconditions(ctx *models.AgentContext) error {
	if ctx.Retrieval == nil {
		return fmt.Errorf("retrieval context is nil")
	}

	if len(ctx.Retrieval.Artifacts) == 0 {
		return fmt.Errorf("no artifacts found (required: retrieval.artifacts)")
	}

	return nil
}

// normalizeFacts converts raw artifacts to normalized facts.
func (a *ContextSynthesizerAgent) normalizeFacts(artifacts []models.Artifact) []models.Fact {
	facts := []models.Fact{}

	for _, artifact := range artifacts {
		fact := models.Fact{
			ID:         fmt.Sprintf("fact-%s", artifact.ID),
			Content:    a.extractContent(artifact),
			Source:     artifact.Source,
			Timestamp:  time.Now(),
			Confidence: a.calculateConfidence(artifact),
			Provenance: map[string]interface{}{
				"artifact_id":   artifact.ID,
				"artifact_type": artifact.Type,
				"source":        artifact.Source,
			},
		}
		facts = append(facts, fact)
	}

	return facts
}

// extractContent extracts textual content from artifact.
func (a *ContextSynthesizerAgent) extractContent(artifact models.Artifact) string {
	// Handle different artifact types
	if str, ok := artifact.Content.(string); ok {
		return str
	}

	if m, ok := artifact.Content.(map[string]interface{}); ok {
		// Extract key fields
		if title, ok := m["title"].(string); ok {
			return title
		}
		if desc, ok := m["description"].(string); ok {
			return desc
		}
		if msg, ok := m["message"].(string); ok {
			return msg
		}
	}

	return fmt.Sprintf("%v", artifact.Content)
}

// calculateConfidence assigns confidence score to fact.
func (a *ContextSynthesizerAgent) calculateConfidence(artifact models.Artifact) float64 {
	// Higher confidence for structured sources
	switch artifact.Source {
	case "gitlab", "youtrack":
		return 0.95 // High confidence (structured data)
	case "analytics":
		return 0.85
	default:
		return 0.70 // Lower confidence (unstructured)
	}
}

// deduplicateFacts removes duplicate facts using content hash.
func (a *ContextSynthesizerAgent) deduplicateFacts(facts []models.Fact) []models.Fact {
	seen := make(map[string]bool)
	deduplicated := []models.Fact{}

	for _, fact := range facts {
		hash := a.contentHash(fact.Content)
		if !seen[hash] {
			seen[hash] = true
			deduplicated = append(deduplicated, fact)
		}
	}

	return deduplicated
}

// contentHash generates hash for deduplication.
func (a *ContextSynthesizerAgent) contentHash(content string) string {
	normalized := strings.ToLower(strings.TrimSpace(content))
	hash := md5.Sum([]byte(normalized))
	return fmt.Sprintf("%x", hash)
}

// deriveKnowledge creates derived knowledge from facts.
func (a *ContextSynthesizerAgent) deriveKnowledge(facts []models.Fact) []models.Knowledge {
	knowledge := []models.Knowledge{}

	// Group facts by source
	sourceGroups := make(map[string][]models.Fact)
	for _, fact := range facts {
		sourceGroups[fact.Source] = append(sourceGroups[fact.Source], fact)
	}

	// Derive knowledge per source
	knowledgeID := 0
	for source, sourceFacts := range sourceGroups {
		if len(sourceFacts) > 0 {
			k := models.Knowledge{
				ID:      fmt.Sprintf("k%d", knowledgeID),
				Content: fmt.Sprintf("Aggregated %d facts from %s", len(sourceFacts), source),
				DerivedFrom: func() []string {
					ids := []string{}
					for _, f := range sourceFacts {
						ids = append(ids, f.ID)
					}
					return ids
				}(),
			}
			knowledge = append(knowledge, k)
			knowledgeID++
		}
	}

	return knowledge
}

// extractRelationships finds relationships between entities in facts.
func (a *ContextSynthesizerAgent) extractRelationships(facts []models.Fact) []models.Relationship {
	relationships := []models.Relationship{}

	// Simple relationship extraction based on source
	sources := make(map[string]bool)
	for _, fact := range facts {
		sources[fact.Source] = true
	}

	// Create relationships between facts from same source
	sourceList := []string{}
	for source := range sources {
		sourceList = append(sourceList, source)
	}

	for i := 0; i < len(sourceList); i++ {
		for j := i + 1; j < len(sourceList); j++ {
			relationships = append(relationships, models.Relationship{
				From: sourceList[i],
				To:   sourceList[j],
				Type: "related_source",
			})
		}
	}

	return relationships
}

// recordAgentRun records execution in audit trail.
func (a *ContextSynthesizerAgent) recordAgentRun(ctx *models.AgentContext, duration time.Duration, status string, err error) {
	run := models.AgentRun{
		Timestamp:  time.Now(),
		AgentID:    a.id,
		Status:     status,
		DurationMS: duration.Milliseconds(),
		KeysWritten: []string{
			"enrichment.facts",
			"enrichment.derived_knowledge",
			"enrichment.relationships",
		},
	}

	if err != nil {
		run.Error = err.Error()
	}

	if ctx.Audit == nil {
		ctx.Audit = &models.AuditContext{}
	}

	ctx.Audit.AgentRuns = append(ctx.Audit.AgentRuns, run)

	if ctx.Diagnostics == nil {
		ctx.Diagnostics = &models.DiagnosticsContext{
			Performance: &models.PerformanceData{},
		}
	}

	if ctx.Diagnostics.Performance.AgentMetrics == nil {
		ctx.Diagnostics.Performance.AgentMetrics = make(map[string]*models.AgentMetrics)
	}

	ctx.Diagnostics.Performance.AgentMetrics[a.id] = &models.AgentMetrics{
		DurationMS: duration.Milliseconds(),
		LLMCalls:   0,
		Status:     status,
	}
}

// GetMetadata returns agent metadata.
func (a *ContextSynthesizerAgent) GetMetadata() services.AgentMetadata {
	return services.AgentMetadata{
		ID:          a.id,
		Name:        "Context Synthesizer Agent",
		Description: "Normalizes and merges facts from multiple sources with deduplication and provenance tracking",
		Version:     "1.0.0",
		Author:      "ADK LLM Proxy",
		Tags:        []string{"synthesis", "normalization", "deduplication", "provenance", "enrichment"},
		Dependencies: []string{"retrieval_planner"},
	}
}

// GetCapabilities returns agent capabilities.
func (a *ContextSynthesizerAgent) GetCapabilities() services.AgentCapabilities {
	return services.AgentCapabilities{
		SupportsParallelExecution: false,
		SupportsRetry:             true,
		RequiresLLM:               false,
		IsDeterministic:           true,
		EstimatedDuration:         100,
	}
}
