package models

import (
	"encoding/json"
	"fmt"
)

// ContextLimits defines size constraints for AgentContext.
type ContextLimits struct {
	// Maximum total context size in bytes
	MaxTotalSize int `json:"max_total_size"`

	// Maximum size for individual namespace in bytes
	MaxNamespaceSize int `json:"max_namespace_size"`

	// Maximum number of items in arrays
	MaxArrayItems int `json:"max_array_items"`

	// Threshold for externalizing artifacts (in bytes)
	ArtifactExternalizationThreshold int `json:"artifact_externalization_threshold"`

	// Maximum individual artifact size before externalization
	MaxInlineArtifactSize int `json:"max_inline_artifact_size"`
}

// DefaultContextLimits returns default context size limits.
func DefaultContextLimits() *ContextLimits {
	return &ContextLimits{
		MaxTotalSize:                     10 * 1024 * 1024, // 10 MB
		MaxNamespaceSize:                 2 * 1024 * 1024,  // 2 MB
		MaxArrayItems:                    1000,
		ArtifactExternalizationThreshold: 100 * 1024, // 100 KB
		MaxInlineArtifactSize:            50 * 1024,  // 50 KB
	}
}

// ContextSizeError is raised when context size limits are exceeded.
type ContextSizeError struct {
	Limit   string
	Current int
	Maximum int
}

func (e *ContextSizeError) Error() string {
	return fmt.Sprintf("context size limit exceeded: %s (current: %d bytes, max: %d bytes)",
		e.Limit, e.Current, e.Maximum)
}

// ContextSizeChecker validates context against size limits.
type ContextSizeChecker struct {
	limits *ContextLimits
}

// NewContextSizeChecker creates a new size checker with given limits.
func NewContextSizeChecker(limits *ContextLimits) *ContextSizeChecker {
	if limits == nil {
		limits = DefaultContextLimits()
	}
	return &ContextSizeChecker{limits: limits}
}

// Check validates the context against size limits.
func (c *ContextSizeChecker) Check(ctx *AgentContext) error {
	// Check total size
	totalSize, err := c.getTotalSize(ctx)
	if err != nil {
		return fmt.Errorf("failed to calculate context size: %w", err)
	}

	if totalSize > c.limits.MaxTotalSize {
		return &ContextSizeError{
			Limit:   "total_context",
			Current: totalSize,
			Maximum: c.limits.MaxTotalSize,
		}
	}

	// Check individual namespace sizes
	if err := c.checkNamespaceSize("reasoning", ctx.Reasoning); err != nil {
		return err
	}
	if err := c.checkNamespaceSize("enrichment", ctx.Enrichment); err != nil {
		return err
	}
	if err := c.checkNamespaceSize("retrieval", ctx.Retrieval); err != nil {
		return err
	}
	if err := c.checkNamespaceSize("llm", ctx.LLM); err != nil {
		return err
	}
	if err := c.checkNamespaceSize("diagnostics", ctx.Diagnostics); err != nil {
		return err
	}
	if err := c.checkNamespaceSize("audit", ctx.Audit); err != nil {
		return err
	}

	// Check array sizes
	if err := c.checkArrayLimits(ctx); err != nil {
		return err
	}

	return nil
}

// getTotalSize calculates the total serialized size of the context.
func (c *ContextSizeChecker) getTotalSize(ctx *AgentContext) (int, error) {
	data, err := json.Marshal(ctx)
	if err != nil {
		return 0, err
	}
	return len(data), nil
}

// checkNamespaceSize checks the size of a single namespace.
func (c *ContextSizeChecker) checkNamespaceSize(name string, namespace interface{}) error {
	data, err := json.Marshal(namespace)
	if err != nil {
		return fmt.Errorf("failed to marshal namespace %s: %w", name, err)
	}

	size := len(data)
	if size > c.limits.MaxNamespaceSize {
		return &ContextSizeError{
			Limit:   fmt.Sprintf("namespace_%s", name),
			Current: size,
			Maximum: c.limits.MaxNamespaceSize,
		}
	}

	return nil
}

// checkArrayLimits checks array item counts.
func (c *ContextSizeChecker) checkArrayLimits(ctx *AgentContext) error {
	// Reasoning arrays
	if len(ctx.Reasoning.Intents) > c.limits.MaxArrayItems {
		return &ContextSizeError{
			Limit:   "reasoning.intents",
			Current: len(ctx.Reasoning.Intents),
			Maximum: c.limits.MaxArrayItems,
		}
	}
	if len(ctx.Reasoning.Hypotheses) > c.limits.MaxArrayItems {
		return &ContextSizeError{
			Limit:   "reasoning.hypotheses",
			Current: len(ctx.Reasoning.Hypotheses),
			Maximum: c.limits.MaxArrayItems,
		}
	}
	if len(ctx.Reasoning.Conclusions) > c.limits.MaxArrayItems {
		return &ContextSizeError{
			Limit:   "reasoning.conclusions",
			Current: len(ctx.Reasoning.Conclusions),
			Maximum: c.limits.MaxArrayItems,
		}
	}

	// Enrichment arrays
	if len(ctx.Enrichment.Facts) > c.limits.MaxArrayItems {
		return &ContextSizeError{
			Limit:   "enrichment.facts",
			Current: len(ctx.Enrichment.Facts),
			Maximum: c.limits.MaxArrayItems,
		}
	}
	if len(ctx.Enrichment.DerivedKnowledge) > c.limits.MaxArrayItems {
		return &ContextSizeError{
			Limit:   "enrichment.derived_knowledge",
			Current: len(ctx.Enrichment.DerivedKnowledge),
			Maximum: c.limits.MaxArrayItems,
		}
	}

	// Retrieval arrays
	if len(ctx.Retrieval.Plans) > c.limits.MaxArrayItems {
		return &ContextSizeError{
			Limit:   "retrieval.plans",
			Current: len(ctx.Retrieval.Plans),
			Maximum: c.limits.MaxArrayItems,
		}
	}
	if len(ctx.Retrieval.Queries) > c.limits.MaxArrayItems {
		return &ContextSizeError{
			Limit:   "retrieval.queries",
			Current: len(ctx.Retrieval.Queries),
			Maximum: c.limits.MaxArrayItems,
		}
	}
	if len(ctx.Retrieval.Artifacts) > c.limits.MaxArrayItems {
		return &ContextSizeError{
			Limit:   "retrieval.artifacts",
			Current: len(ctx.Retrieval.Artifacts),
			Maximum: c.limits.MaxArrayItems,
		}
	}

	// Audit arrays
	if len(ctx.Audit.AgentRuns) > c.limits.MaxArrayItems {
		return &ContextSizeError{
			Limit:   "audit.agent_runs",
			Current: len(ctx.Audit.AgentRuns),
			Maximum: c.limits.MaxArrayItems,
		}
	}
	if len(ctx.Audit.Diffs) > c.limits.MaxArrayItems {
		return &ContextSizeError{
			Limit:   "audit.diffs",
			Current: len(ctx.Audit.Diffs),
			Maximum: c.limits.MaxArrayItems,
		}
	}

	return nil
}

// ShouldExternalizeArtifacts checks if artifacts should be externalized based on size.
func (c *ContextSizeChecker) ShouldExternalizeArtifacts(ctx *AgentContext) bool {
	retrievalSize, err := c.getNamespaceSize(ctx.Retrieval)
	if err != nil {
		return false
	}
	return retrievalSize > c.limits.ArtifactExternalizationThreshold
}

// getNamespaceSize calculates the size of a namespace.
func (c *ContextSizeChecker) getNamespaceSize(namespace interface{}) (int, error) {
	data, err := json.Marshal(namespace)
	if err != nil {
		return 0, err
	}
	return len(data), nil
}

// ExternalArtifactReference represents a reference to an externalized artifact.
type ExternalArtifactReference struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Size     int    `json:"size"`
	Location string `json:"location"` // Storage location (e.g., "s3://bucket/key", "file:///path")
	Checksum string `json:"checksum,omitempty"`
}

// ArtifactExternalizer handles externalization of large artifacts.
type ArtifactExternalizer interface {
	// Store externalizes an artifact and returns a reference
	Store(artifact *Artifact) (*ExternalArtifactReference, error)

	// Retrieve loads an artifact from external storage
	Retrieve(ref *ExternalArtifactReference) (*Artifact, error)

	// Delete removes an externalized artifact
	Delete(ref *ExternalArtifactReference) error
}

// MemoryArtifactExternalizer is a simple in-memory implementation for testing.
type MemoryArtifactExternalizer struct {
	storage map[string]*Artifact
}

// NewMemoryArtifactExternalizer creates a new in-memory externalizer.
func NewMemoryArtifactExternalizer() *MemoryArtifactExternalizer {
	return &MemoryArtifactExternalizer{
		storage: make(map[string]*Artifact),
	}
}

// Store stores an artifact in memory.
func (e *MemoryArtifactExternalizer) Store(artifact *Artifact) (*ExternalArtifactReference, error) {
	data, err := json.Marshal(artifact.Content)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal artifact: %w", err)
	}

	ref := &ExternalArtifactReference{
		ID:       artifact.ID,
		Type:     artifact.Type,
		Size:     len(data),
		Location: fmt.Sprintf("memory://%s", artifact.ID),
	}

	e.storage[artifact.ID] = artifact
	return ref, nil
}

// Retrieve retrieves an artifact from memory.
func (e *MemoryArtifactExternalizer) Retrieve(ref *ExternalArtifactReference) (*Artifact, error) {
	artifact, exists := e.storage[ref.ID]
	if !exists {
		return nil, fmt.Errorf("artifact not found: %s", ref.ID)
	}
	return artifact, nil
}

// Delete removes an artifact from memory.
func (e *MemoryArtifactExternalizer) Delete(ref *ExternalArtifactReference) error {
	delete(e.storage, ref.ID)
	return nil
}

// ExternalizeArtifacts externalizes large artifacts in the context.
func ExternalizeArtifacts(ctx *AgentContext, externalizer ArtifactExternalizer, limits *ContextLimits) error {
	if limits == nil {
		limits = DefaultContextLimits()
	}

	var externalizedCount int
	var newArtifacts []Artifact

	for _, artifact := range ctx.Retrieval.Artifacts {
		// Calculate artifact size
		data, err := json.Marshal(artifact.Content)
		if err != nil {
			return fmt.Errorf("failed to marshal artifact %s: %w", artifact.ID, err)
		}

		size := len(data)

		// Externalize if too large
		if size > limits.MaxInlineArtifactSize {
			ref, err := externalizer.Store(&artifact)
			if err != nil {
				return fmt.Errorf("failed to externalize artifact %s: %w", artifact.ID, err)
			}

			// Replace content with reference
			artifact.Content = map[string]interface{}{
				"externalized": true,
				"reference":    ref,
			}
			externalizedCount++
		}

		newArtifacts = append(newArtifacts, artifact)
	}

	ctx.Retrieval.Artifacts = newArtifacts

	// Log externalization
	if externalizedCount > 0 {
		ctx.Diagnostics.Warnings = append(ctx.Diagnostics.Warnings, Warning{
			AgentID: "system",
			Message: fmt.Sprintf("Externalized %d large artifacts", externalizedCount),
		})
	}

	return nil
}

// GetContextStats returns statistics about context size.
type ContextStats struct {
	TotalSize         int            `json:"total_size"`
	NamespaceSizes    map[string]int `json:"namespace_sizes"`
	ArrayCounts       map[string]int `json:"array_counts"`
	ExternalizedCount int            `json:"externalized_count"`
}

// GetStats calculates statistics about the context.
func GetStats(ctx *AgentContext) (*ContextStats, error) {
	stats := &ContextStats{
		NamespaceSizes: make(map[string]int),
		ArrayCounts:    make(map[string]int),
	}

	// Total size
	data, err := json.Marshal(ctx)
	if err != nil {
		return nil, err
	}
	stats.TotalSize = len(data)

	// Namespace sizes
	checker := NewContextSizeChecker(nil)

	if size, err := checker.getNamespaceSize(ctx.Reasoning); err == nil {
		stats.NamespaceSizes["reasoning"] = size
	}
	if size, err := checker.getNamespaceSize(ctx.Enrichment); err == nil {
		stats.NamespaceSizes["enrichment"] = size
	}
	if size, err := checker.getNamespaceSize(ctx.Retrieval); err == nil {
		stats.NamespaceSizes["retrieval"] = size
	}
	if size, err := checker.getNamespaceSize(ctx.LLM); err == nil {
		stats.NamespaceSizes["llm"] = size
	}
	if size, err := checker.getNamespaceSize(ctx.Diagnostics); err == nil {
		stats.NamespaceSizes["diagnostics"] = size
	}
	if size, err := checker.getNamespaceSize(ctx.Audit); err == nil {
		stats.NamespaceSizes["audit"] = size
	}

	// Array counts
	stats.ArrayCounts["intents"] = len(ctx.Reasoning.Intents)
	stats.ArrayCounts["hypotheses"] = len(ctx.Reasoning.Hypotheses)
	stats.ArrayCounts["conclusions"] = len(ctx.Reasoning.Conclusions)
	stats.ArrayCounts["facts"] = len(ctx.Enrichment.Facts)
	stats.ArrayCounts["knowledge"] = len(ctx.Enrichment.DerivedKnowledge)
	stats.ArrayCounts["plans"] = len(ctx.Retrieval.Plans)
	stats.ArrayCounts["queries"] = len(ctx.Retrieval.Queries)
	stats.ArrayCounts["artifacts"] = len(ctx.Retrieval.Artifacts)
	stats.ArrayCounts["agent_runs"] = len(ctx.Audit.AgentRuns)
	stats.ArrayCounts["diffs"] = len(ctx.Audit.Diffs)

	// Count externalized artifacts
	for _, artifact := range ctx.Retrieval.Artifacts {
		if content, ok := artifact.Content.(map[string]interface{}); ok {
			if externalized, ok := content["externalized"].(bool); ok && externalized {
				stats.ExternalizedCount++
			}
		}
	}

	return stats, nil
}
