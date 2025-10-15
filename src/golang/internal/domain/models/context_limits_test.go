package models_test

import (
	"testing"

	"github.com/mshogin/agents/internal/domain/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultContextLimits(t *testing.T) {
	limits := models.DefaultContextLimits()

	assert.Equal(t, 10*1024*1024, limits.MaxTotalSize)
	assert.Equal(t, 2*1024*1024, limits.MaxNamespaceSize)
	assert.Equal(t, 1000, limits.MaxArrayItems)
	assert.Equal(t, 100*1024, limits.ArtifactExternalizationThreshold)
	assert.Equal(t, 50*1024, limits.MaxInlineArtifactSize)
}

func TestContextSizeChecker_Check_ValidContext(t *testing.T) {
	ctx := models.NewAgentContext("session-1", "trace-1")
	ctx.Reasoning.Intents = []models.Intent{
		{Type: "query", Confidence: 0.95},
	}

	checker := models.NewContextSizeChecker(nil)
	err := checker.Check(ctx)
	assert.NoError(t, err)
}

func TestContextSizeChecker_Check_ArrayLimitExceeded(t *testing.T) {
	limits := &models.ContextLimits{
		MaxTotalSize:     10 * 1024 * 1024,
		MaxNamespaceSize: 2 * 1024 * 1024,
		MaxArrayItems:    5, // Small limit for testing
	}

	ctx := models.NewAgentContext("session-1", "trace-1")

	// Add more intents than allowed
	for i := 0; i < 10; i++ {
		ctx.Reasoning.Intents = append(ctx.Reasoning.Intents, models.Intent{
			Type:       "query",
			Confidence: 0.95,
		})
	}

	checker := models.NewContextSizeChecker(limits)
	err := checker.Check(ctx)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "reasoning.intents")
	assert.Contains(t, err.Error(), "context size limit exceeded")
}

func TestContextSizeChecker_ShouldExternalizeArtifacts(t *testing.T) {
	limits := &models.ContextLimits{
		ArtifactExternalizationThreshold: 1000, // 1KB threshold
	}

	ctx := models.NewAgentContext("session-1", "trace-1")

	checker := models.NewContextSizeChecker(limits)

	// Small retrieval context - no externalization needed
	ctx.Retrieval.Artifacts = []models.Artifact{
		{ID: "a1", Type: "text", Content: "small"},
	}
	assert.False(t, checker.ShouldExternalizeArtifacts(ctx))

	// Large retrieval context - externalization needed
	largeContent := make([]byte, 2000)
	for i := range largeContent {
		largeContent[i] = 'x'
	}
	ctx.Retrieval.Artifacts = []models.Artifact{
		{ID: "a1", Type: "text", Content: string(largeContent)},
	}
	assert.True(t, checker.ShouldExternalizeArtifacts(ctx))
}

func TestMemoryArtifactExternalizer_StoreRetrieve(t *testing.T) {
	externalizer := models.NewMemoryArtifactExternalizer()

	artifact := &models.Artifact{
		ID:      "a1",
		Type:    "document",
		Content: map[string]string{"text": "test content"},
		Source:  "test",
	}

	// Store
	ref, err := externalizer.Store(artifact)
	require.NoError(t, err)
	assert.Equal(t, "a1", ref.ID)
	assert.Equal(t, "document", ref.Type)
	assert.Contains(t, ref.Location, "memory://")

	// Retrieve
	retrieved, err := externalizer.Retrieve(ref)
	require.NoError(t, err)
	assert.Equal(t, artifact.ID, retrieved.ID)
	assert.Equal(t, artifact.Type, retrieved.Type)

	// Delete
	err = externalizer.Delete(ref)
	assert.NoError(t, err)

	// Verify deleted
	_, err = externalizer.Retrieve(ref)
	assert.Error(t, err)
}

func TestExternalizeArtifacts(t *testing.T) {
	ctx := models.NewAgentContext("session-1", "trace-1")

	// Create large artifact that should be externalized
	largeContent := make(map[string]interface{})
	largeContent["data"] = make([]byte, 60*1024) // 60 KB

	ctx.Retrieval.Artifacts = []models.Artifact{
		{
			ID:      "small1",
			Type:    "text",
			Content: "small content",
			Source:  "test",
		},
		{
			ID:      "large1",
			Type:    "document",
			Content: largeContent,
			Source:  "test",
		},
	}

	externalizer := models.NewMemoryArtifactExternalizer()
	limits := models.DefaultContextLimits()

	err := models.ExternalizeArtifacts(ctx, externalizer, limits)
	require.NoError(t, err)

	// Verify small artifact is unchanged
	assert.Equal(t, "small content", ctx.Retrieval.Artifacts[0].Content)

	// Verify large artifact is externalized
	largeArtifact := ctx.Retrieval.Artifacts[1]
	contentMap, ok := largeArtifact.Content.(map[string]interface{})
	require.True(t, ok)

	externalized, ok := contentMap["externalized"].(bool)
	require.True(t, ok)
	assert.True(t, externalized)

	// Verify warning was added
	assert.NotEmpty(t, ctx.Diagnostics.Warnings)
	assert.Contains(t, ctx.Diagnostics.Warnings[0].Message, "Externalized 1 large artifacts")
}

func TestExternalizeArtifacts_NoLargeArtifacts(t *testing.T) {
	ctx := models.NewAgentContext("session-1", "trace-1")

	ctx.Retrieval.Artifacts = []models.Artifact{
		{ID: "a1", Type: "text", Content: "small 1", Source: "test"},
		{ID: "a2", Type: "text", Content: "small 2", Source: "test"},
	}

	externalizer := models.NewMemoryArtifactExternalizer()

	err := models.ExternalizeArtifacts(ctx, externalizer, nil)
	require.NoError(t, err)

	// Verify no externalization occurred
	assert.Equal(t, "small 1", ctx.Retrieval.Artifacts[0].Content)
	assert.Equal(t, "small 2", ctx.Retrieval.Artifacts[1].Content)
	assert.Empty(t, ctx.Diagnostics.Warnings)
}

func TestGetStats(t *testing.T) {
	ctx := models.NewAgentContext("session-1", "trace-1")

	ctx.Reasoning.Intents = []models.Intent{
		{Type: "query", Confidence: 0.95},
		{Type: "analysis", Confidence: 0.88},
	}
	ctx.Reasoning.Hypotheses = []models.Hypothesis{
		{ID: "h1", Description: "test hypothesis"},
	}
	ctx.Enrichment.Facts = []models.Fact{
		{ID: "f1", Content: "fact 1", Source: "test"},
		{ID: "f2", Content: "fact 2", Source: "test"},
		{ID: "f3", Content: "fact 3", Source: "test"},
	}

	stats, err := models.GetStats(ctx)
	require.NoError(t, err)

	// Check total size
	assert.Greater(t, stats.TotalSize, 0)

	// Check namespace sizes
	assert.Greater(t, stats.NamespaceSizes["reasoning"], 0)
	assert.Greater(t, stats.NamespaceSizes["enrichment"], 0)

	// Check array counts
	assert.Equal(t, 2, stats.ArrayCounts["intents"])
	assert.Equal(t, 1, stats.ArrayCounts["hypotheses"])
	assert.Equal(t, 3, stats.ArrayCounts["facts"])
	assert.Equal(t, 0, stats.ArrayCounts["conclusions"])
}

func TestGetStats_WithExternalizedArtifacts(t *testing.T) {
	ctx := models.NewAgentContext("session-1", "trace-1")

	// Add normal artifact
	ctx.Retrieval.Artifacts = []models.Artifact{
		{ID: "a1", Type: "text", Content: "normal content"},
	}

	// Add externalized artifact
	ctx.Retrieval.Artifacts = append(ctx.Retrieval.Artifacts, models.Artifact{
		ID:   "a2",
		Type: "document",
		Content: map[string]interface{}{
			"externalized": true,
			"reference": &models.ExternalArtifactReference{
				ID:       "a2",
				Location: "memory://a2",
			},
		},
	})

	stats, err := models.GetStats(ctx)
	require.NoError(t, err)

	assert.Equal(t, 2, stats.ArrayCounts["artifacts"])
	assert.Equal(t, 1, stats.ExternalizedCount)
}

func TestContextSizeError_Error(t *testing.T) {
	err := &models.ContextSizeError{
		Limit:   "test_limit",
		Current: 5000,
		Maximum: 1000,
	}

	assert.Contains(t, err.Error(), "context size limit exceeded")
	assert.Contains(t, err.Error(), "test_limit")
	assert.Contains(t, err.Error(), "5000")
	assert.Contains(t, err.Error(), "1000")
}

func TestContextSizeChecker_Check_NamespaceSizeExceeded(t *testing.T) {
	limits := &models.ContextLimits{
		MaxTotalSize:     10 * 1024 * 1024,
		MaxNamespaceSize: 1000, // 1KB limit for testing
		MaxArrayItems:    1000,
	}

	ctx := models.NewAgentContext("session-1", "trace-1")

	// Add many facts to exceed namespace size
	for i := 0; i < 100; i++ {
		largeContent := make([]byte, 100)
		for j := range largeContent {
			largeContent[j] = 'x'
		}
		ctx.Enrichment.Facts = append(ctx.Enrichment.Facts, models.Fact{
			ID:      string(rune(i)),
			Content: string(largeContent),
			Source:  "test",
		})
	}

	checker := models.NewContextSizeChecker(limits)
	err := checker.Check(ctx)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "namespace_enrichment")
}
