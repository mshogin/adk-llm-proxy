package agents

import (
	"context"
	"testing"

	"github.com/mshogin/agents/internal/domain/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test agent initialization
func TestNewContextSynthesizerAgent(t *testing.T) {
	agent := NewContextSynthesizerAgent()
	assert.NotNil(t, agent)
	assert.Equal(t, "context_synthesizer", agent.AgentID())
}

// Test AgentID method
func TestContextSynthesizer_AgentID(t *testing.T) {
	agent := NewContextSynthesizerAgent()
	assert.Equal(t, "context_synthesizer", agent.AgentID())
}

// Test Preconditions method
func TestContextSynthesizer_Preconditions(t *testing.T) {
	agent := NewContextSynthesizerAgent()
	preconditions := agent.Preconditions()

	assert.Len(t, preconditions, 1)
	assert.Contains(t, preconditions, "retrieval.artifacts")
}

// Test Postconditions method
func TestContextSynthesizer_Postconditions(t *testing.T) {
	agent := NewContextSynthesizerAgent()
	postconditions := agent.Postconditions()

	assert.Len(t, postconditions, 3)
	assert.Contains(t, postconditions, "enrichment.facts")
	assert.Contains(t, postconditions, "enrichment.derived_knowledge")
	assert.Contains(t, postconditions, "enrichment.relationships")
}

// Test GetMetadata method
func TestContextSynthesizer_GetMetadata(t *testing.T) {
	agent := NewContextSynthesizerAgent()
	metadata := agent.GetMetadata()

	assert.Equal(t, "context_synthesizer", metadata.ID)
	assert.Equal(t, "Context Synthesizer Agent", metadata.Name)
	assert.NotEmpty(t, metadata.Description)
	assert.Equal(t, "1.0.0", metadata.Version)
	assert.NotEmpty(t, metadata.Tags)
	assert.Contains(t, metadata.Dependencies, "retrieval_planner")
}

// Test GetCapabilities method
func TestContextSynthesizer_GetCapabilities(t *testing.T) {
	agent := NewContextSynthesizerAgent()
	caps := agent.GetCapabilities()

	assert.False(t, caps.SupportsParallelExecution)
	assert.True(t, caps.SupportsRetry)
	assert.False(t, caps.RequiresLLM)
	assert.True(t, caps.IsDeterministic)
	assert.Greater(t, caps.EstimatedDuration, 0)
}

// Test Execute with valid artifacts
func TestContextSynthesizer_Execute_ValidArtifacts(t *testing.T) {
	agent := NewContextSynthesizerAgent()

	ctx := &models.AgentContext{
		Retrieval: &models.RetrievalContext{
			Artifacts: []models.Artifact{
				{
					ID:      "artifact1",
					Type:    "commit",
					Source:  "gitlab",
					Content: "First commit message",
				},
				{
					ID:      "artifact2",
					Type:    "issue",
					Source:  "youtrack",
					Content: "Bug report",
				},
			},
		},
		Enrichment: &models.EnrichmentContext{},
	}

	result, err := agent.Execute(context.Background(), ctx)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Enrichment)

	// Verify facts were created
	assert.Len(t, result.Enrichment.Facts, 2)

	// Verify fact structure
	for _, fact := range result.Enrichment.Facts {
		assert.NotEmpty(t, fact.ID)
		assert.NotEmpty(t, fact.Content)
		assert.NotEmpty(t, fact.Source)
		assert.Greater(t, fact.Confidence, 0.0)
		assert.NotNil(t, fact.Provenance)
	}

	// Verify derived knowledge
	assert.NotEmpty(t, result.Enrichment.DerivedKnowledge)

	// Verify relationships
	assert.NotEmpty(t, result.Enrichment.Relationships)
}

// Test Execute with missing artifacts (precondition failure)
func TestContextSynthesizer_Execute_MissingArtifacts(t *testing.T) {
	agent := NewContextSynthesizerAgent()

	ctx := &models.AgentContext{
		Retrieval: &models.RetrievalContext{
			Artifacts: []models.Artifact{}, // Empty
		},
		Enrichment: &models.EnrichmentContext{},
	}

	result, err := agent.Execute(context.Background(), ctx)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "no artifacts found")
}

// Test Execute with nil retrieval context
func TestContextSynthesizer_Execute_NilRetrievalContext(t *testing.T) {
	agent := NewContextSynthesizerAgent()

	ctx := &models.AgentContext{
		Retrieval:  nil, // Nil retrieval
		Enrichment: &models.EnrichmentContext{},
	}

	result, err := agent.Execute(context.Background(), ctx)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "retrieval context is nil")
}

// Test fact normalization with string content
func TestContextSynthesizer_NormalizeFacts_StringContent(t *testing.T) {
	agent := NewContextSynthesizerAgent()

	artifacts := []models.Artifact{
		{
			ID:      "artifact1",
			Type:    "commit",
			Source:  "gitlab",
			Content: "Commit message",
		},
	}

	facts := agent.normalizeFacts(artifacts)

	require.Len(t, facts, 1)
	assert.Equal(t, "fact-artifact1", facts[0].ID)
	assert.Equal(t, "Commit message", facts[0].Content)
	assert.Equal(t, "gitlab", facts[0].Source)
	assert.Equal(t, 0.95, facts[0].Confidence) // GitLab has 0.95 confidence
	assert.NotNil(t, facts[0].Provenance)
}

// Test fact normalization with map content
func TestContextSynthesizer_NormalizeFacts_MapContent(t *testing.T) {
	agent := NewContextSynthesizerAgent()

	artifacts := []models.Artifact{
		{
			ID:     "artifact1",
			Type:   "issue",
			Source: "youtrack",
			Content: map[string]interface{}{
				"title":       "Bug report",
				"description": "Detailed description",
			},
		},
	}

	facts := agent.normalizeFacts(artifacts)

	require.Len(t, facts, 1)
	assert.Equal(t, "Bug report", facts[0].Content) // Title takes precedence
}

// Test deduplication with duplicate content
func TestContextSynthesizer_DeduplicateFacts_WithDuplicates(t *testing.T) {
	agent := NewContextSynthesizerAgent()

	facts := []models.Fact{
		{
			ID:      "fact1",
			Content: "Same content",
			Source:  "gitlab",
		},
		{
			ID:      "fact2",
			Content: "Same content", // Duplicate
			Source:  "gitlab",
		},
		{
			ID:      "fact3",
			Content: "Different content",
			Source:  "youtrack",
		},
	}

	deduplicated := agent.deduplicateFacts(facts)

	// Should keep 2 facts (duplicate removed)
	assert.Len(t, deduplicated, 2)

	// Verify first occurrence kept
	assert.Equal(t, "fact1", deduplicated[0].ID)
	assert.Equal(t, "fact3", deduplicated[1].ID)
}

// Test deduplication with case-insensitive matching
func TestContextSynthesizer_DeduplicateFacts_CaseInsensitive(t *testing.T) {
	agent := NewContextSynthesizerAgent()

	facts := []models.Fact{
		{
			ID:      "fact1",
			Content: "Content with CAPS",
			Source:  "gitlab",
		},
		{
			ID:      "fact2",
			Content: "content with caps", // Different case, same content
			Source:  "gitlab",
		},
	}

	deduplicated := agent.deduplicateFacts(facts)

	// Should keep only 1 (case-insensitive deduplication)
	assert.Len(t, deduplicated, 1)
	assert.Equal(t, "fact1", deduplicated[0].ID)
}

// Test deduplication with unique content
func TestContextSynthesizer_DeduplicateFacts_UniqueContent(t *testing.T) {
	agent := NewContextSynthesizerAgent()

	facts := []models.Fact{
		{
			ID:      "fact1",
			Content: "First fact",
			Source:  "gitlab",
		},
		{
			ID:      "fact2",
			Content: "Second fact",
			Source:  "youtrack",
		},
		{
			ID:      "fact3",
			Content: "Third fact",
			Source:  "analytics",
		},
	}

	deduplicated := agent.deduplicateFacts(facts)

	// All unique, should keep all
	assert.Len(t, deduplicated, 3)
}

// Test confidence scoring for structured sources
func TestContextSynthesizer_CalculateConfidence_StructuredSources(t *testing.T) {
	agent := NewContextSynthesizerAgent()

	tests := []struct {
		name       string
		source     string
		confidence float64
	}{
		{"GitLab", "gitlab", 0.95},
		{"YouTrack", "youtrack", 0.95},
		{"Analytics", "analytics", 0.85},
		{"Unknown", "unknown", 0.70},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			artifact := models.Artifact{Source: tt.source}
			confidence := agent.calculateConfidence(artifact)
			assert.Equal(t, tt.confidence, confidence)
		})
	}
}

// Test knowledge derivation from facts
func TestContextSynthesizer_DeriveKnowledge(t *testing.T) {
	agent := NewContextSynthesizerAgent()

	facts := []models.Fact{
		{
			ID:      "fact1",
			Content: "Commit 1",
			Source:  "gitlab",
		},
		{
			ID:      "fact2",
			Content: "Commit 2",
			Source:  "gitlab",
		},
		{
			ID:      "fact3",
			Content: "Issue 1",
			Source:  "youtrack",
		},
	}

	knowledge := agent.deriveKnowledge(facts)

	// Should create 2 knowledge items (1 per source)
	assert.Len(t, knowledge, 2)

	// Verify knowledge structure
	for _, k := range knowledge {
		assert.NotEmpty(t, k.ID)
		assert.NotEmpty(t, k.Content)
		assert.NotEmpty(t, k.DerivedFrom)
	}

	// Verify aggregation
	gitlabKnowledge := findKnowledgeBySource(knowledge, "gitlab")
	assert.NotNil(t, gitlabKnowledge)
	assert.Contains(t, gitlabKnowledge.Content, "2 facts")
	assert.Len(t, gitlabKnowledge.DerivedFrom, 2)
}

// Test relationship extraction
func TestContextSynthesizer_ExtractRelationships(t *testing.T) {
	agent := NewContextSynthesizerAgent()

	facts := []models.Fact{
		{
			ID:      "fact1",
			Content: "Commit 1",
			Source:  "gitlab",
		},
		{
			ID:      "fact2",
			Content: "Issue 1",
			Source:  "youtrack",
		},
		{
			ID:      "fact3",
			Content: "Metric 1",
			Source:  "analytics",
		},
	}

	relationships := agent.extractRelationships(facts)

	// With 3 sources, should create 3 relationships (3 choose 2 = 3)
	assert.Len(t, relationships, 3)

	// Verify relationship structure
	for _, rel := range relationships {
		assert.NotEmpty(t, rel.From)
		assert.NotEmpty(t, rel.To)
		assert.Equal(t, "related_source", rel.Type)
	}
}

// Test context isolation (input not modified)
func TestContextSynthesizer_Execute_ContextIsolation(t *testing.T) {
	agent := NewContextSynthesizerAgent()

	originalCtx := &models.AgentContext{
		Retrieval: &models.RetrievalContext{
			Artifacts: []models.Artifact{
				{
					ID:      "artifact1",
					Type:    "commit",
					Source:  "gitlab",
					Content: "Commit message",
				},
			},
		},
		Enrichment: &models.EnrichmentContext{},
	}

	// Execute agent
	result, err := agent.Execute(context.Background(), originalCtx)

	require.NoError(t, err)
	require.NotNil(t, result)

	// Original context should not be modified
	assert.Nil(t, originalCtx.Enrichment.Facts)
	assert.Nil(t, originalCtx.Enrichment.DerivedKnowledge)
	assert.Nil(t, originalCtx.Enrichment.Relationships)

	// Result should have new data
	assert.NotNil(t, result.Enrichment.Facts)
	assert.NotNil(t, result.Enrichment.DerivedKnowledge)
	assert.NotNil(t, result.Enrichment.Relationships)
}

// Test audit trail tracking
func TestContextSynthesizer_Execute_AuditTrail(t *testing.T) {
	agent := NewContextSynthesizerAgent()

	ctx := &models.AgentContext{
		Retrieval: &models.RetrievalContext{
			Artifacts: []models.Artifact{
				{
					ID:      "artifact1",
					Type:    "commit",
					Source:  "gitlab",
					Content: "Commit message",
				},
			},
		},
		Enrichment: &models.EnrichmentContext{},
	}

	result, err := agent.Execute(context.Background(), ctx)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Audit)

	// Verify audit trail
	assert.Len(t, result.Audit.AgentRuns, 1)

	run := result.Audit.AgentRuns[0]
	assert.Equal(t, "context_synthesizer", run.AgentID)
	assert.Equal(t, "success", run.Status)
	assert.GreaterOrEqual(t, run.DurationMS, int64(0))
	assert.Contains(t, run.KeysWritten, "enrichment.facts")
	assert.Contains(t, run.KeysWritten, "enrichment.derived_knowledge")
	assert.Contains(t, run.KeysWritten, "enrichment.relationships")

	// Verify diagnostics
	require.NotNil(t, result.Diagnostics)
	require.NotNil(t, result.Diagnostics.Performance)
	require.NotNil(t, result.Diagnostics.Performance.AgentMetrics)

	metrics := result.Diagnostics.Performance.AgentMetrics["context_synthesizer"]
	require.NotNil(t, metrics)
	assert.GreaterOrEqual(t, metrics.DurationMS, int64(0))
	assert.Equal(t, 0, metrics.LLMCalls) // No LLM calls
	assert.Equal(t, "success", metrics.Status)
}

// Test idempotency (same input produces same output)
func TestContextSynthesizer_Execute_Idempotency(t *testing.T) {
	agent := NewContextSynthesizerAgent()

	ctx := &models.AgentContext{
		Retrieval: &models.RetrievalContext{
			Artifacts: []models.Artifact{
				{
					ID:      "artifact1",
					Type:    "commit",
					Source:  "gitlab",
					Content: "Commit message",
				},
			},
		},
		Enrichment: &models.EnrichmentContext{},
	}

	// Execute twice
	result1, err1 := agent.Execute(context.Background(), ctx)
	require.NoError(t, err1)

	result2, err2 := agent.Execute(context.Background(), ctx)
	require.NoError(t, err2)

	// Results should be the same (ignoring timestamps and audit trail)
	assert.Equal(t, len(result1.Enrichment.Facts), len(result2.Enrichment.Facts))
	assert.Equal(t, len(result1.Enrichment.DerivedKnowledge), len(result2.Enrichment.DerivedKnowledge))
	assert.Equal(t, len(result1.Enrichment.Relationships), len(result2.Enrichment.Relationships))

	// Fact content should be identical
	for i := range result1.Enrichment.Facts {
		assert.Equal(t, result1.Enrichment.Facts[i].Content, result2.Enrichment.Facts[i].Content)
		assert.Equal(t, result1.Enrichment.Facts[i].Source, result2.Enrichment.Facts[i].Source)
		assert.Equal(t, result1.Enrichment.Facts[i].Confidence, result2.Enrichment.Facts[i].Confidence)
	}
}

// Test multiple artifacts from different sources
func TestContextSynthesizer_Execute_MultipleSources(t *testing.T) {
	agent := NewContextSynthesizerAgent()

	ctx := &models.AgentContext{
		Retrieval: &models.RetrievalContext{
			Artifacts: []models.Artifact{
				{
					ID:      "artifact1",
					Type:    "commit",
					Source:  "gitlab",
					Content: "Commit 1",
				},
				{
					ID:      "artifact2",
					Type:    "commit",
					Source:  "gitlab",
					Content: "Commit 2",
				},
				{
					ID:      "artifact3",
					Type:    "issue",
					Source:  "youtrack",
					Content: "Issue 1",
				},
				{
					ID:      "artifact4",
					Type:    "metric",
					Source:  "analytics",
					Content: "Metric 1",
				},
			},
		},
		Enrichment: &models.EnrichmentContext{},
	}

	result, err := agent.Execute(context.Background(), ctx)

	require.NoError(t, err)
	require.NotNil(t, result)

	// Verify facts created for all artifacts
	assert.Len(t, result.Enrichment.Facts, 4)

	// Verify knowledge derived per source (3 sources = 3 knowledge items)
	assert.Len(t, result.Enrichment.DerivedKnowledge, 3)

	// Verify relationships between sources
	assert.NotEmpty(t, result.Enrichment.Relationships)
}

// Test extraction of content from different artifact types
func TestContextSynthesizer_ExtractContent_VariousTypes(t *testing.T) {
	agent := NewContextSynthesizerAgent()

	tests := []struct {
		name     string
		artifact models.Artifact
		expected string
	}{
		{
			name: "String content",
			artifact: models.Artifact{
				Content: "Simple string",
			},
			expected: "Simple string",
		},
		{
			name: "Map with title",
			artifact: models.Artifact{
				Content: map[string]interface{}{
					"title": "Title text",
				},
			},
			expected: "Title text",
		},
		{
			name: "Map with description",
			artifact: models.Artifact{
				Content: map[string]interface{}{
					"description": "Description text",
				},
			},
			expected: "Description text",
		},
		{
			name: "Map with message",
			artifact: models.Artifact{
				Content: map[string]interface{}{
					"message": "Message text",
				},
			},
			expected: "Message text",
		},
		{
			name: "Map with multiple fields (title priority)",
			artifact: models.Artifact{
				Content: map[string]interface{}{
					"title":       "Title text",
					"description": "Description text",
					"message":     "Message text",
				},
			},
			expected: "Title text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content := agent.extractContent(tt.artifact)
			assert.Equal(t, tt.expected, content)
		})
	}
}

// Test content hash generation
func TestContextSynthesizer_ContentHash(t *testing.T) {
	agent := NewContextSynthesizerAgent()

	// Same content should produce same hash
	hash1 := agent.contentHash("test content")
	hash2 := agent.contentHash("test content")
	assert.Equal(t, hash1, hash2)

	// Different content should produce different hash
	hash3 := agent.contentHash("different content")
	assert.NotEqual(t, hash1, hash3)

	// Case-insensitive (normalized)
	hash4 := agent.contentHash("TEST CONTENT")
	assert.Equal(t, hash1, hash4)

	// Whitespace-insensitive (normalized)
	hash5 := agent.contentHash("  test content  ")
	assert.Equal(t, hash1, hash5)
}

// Test provenance tracking
func TestContextSynthesizer_Provenance(t *testing.T) {
	agent := NewContextSynthesizerAgent()

	artifacts := []models.Artifact{
		{
			ID:      "artifact123",
			Type:    "commit",
			Source:  "gitlab",
			Content: "Commit message",
		},
	}

	facts := agent.normalizeFacts(artifacts)

	require.Len(t, facts, 1)

	// Verify provenance structure
	provenance := facts[0].Provenance
	require.NotNil(t, provenance)

	assert.Equal(t, "artifact123", provenance["artifact_id"])
	assert.Equal(t, "commit", provenance["artifact_type"])
	assert.Equal(t, "gitlab", provenance["source"])
}

// Helper function to find knowledge by source
func findKnowledgeBySource(knowledge []models.Knowledge, source string) *models.Knowledge {
	for _, k := range knowledge {
		if contains(k.Content, source) {
			return &k
		}
	}
	return nil
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[len(s)-len(substr):] == substr ||
		   s[:len(substr)] == substr ||
		   findSubstring(s, substr)
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
