package models

import (
	"encoding/json"
	"fmt"
	"reflect"
	"time"
)

// DiffTracker tracks changes to the AgentContext.
type DiffTracker struct {
	// Previous state snapshot
	previousState *AgentContext
}

// NewDiffTracker creates a new diff tracker.
func NewDiffTracker(initialState *AgentContext) (*DiffTracker, error) {
	snapshot, err := initialState.Clone()
	if err != nil {
		return nil, fmt.Errorf("failed to create initial snapshot: %w", err)
	}

	return &DiffTracker{
		previousState: snapshot,
	}, nil
}

// Capture captures changes between the previous state and the current state.
func (d *DiffTracker) Capture(agentID string, currentState *AgentContext) (*ContextDiff, error) {
	changes, err := d.computeDiff(d.previousState, currentState)
	if err != nil {
		return nil, fmt.Errorf("failed to compute diff: %w", err)
	}

	diff := &ContextDiff{
		Timestamp: time.Now(),
		AgentID:   agentID,
		Changes:   changes,
	}

	// Update previous state
	snapshot, err := currentState.Clone()
	if err != nil {
		return nil, fmt.Errorf("failed to update snapshot: %w", err)
	}
	d.previousState = snapshot

	return diff, nil
}

// computeDiff computes the differences between two contexts.
func (d *DiffTracker) computeDiff(previous, current *AgentContext) (map[string]interface{}, error) {
	changes := make(map[string]interface{})

	// Compare each namespace
	if !reflect.DeepEqual(previous.Reasoning, current.Reasoning) {
		reasoningChanges := d.compareReasoning(previous.Reasoning, current.Reasoning)
		if len(reasoningChanges) > 0 {
			changes["reasoning"] = reasoningChanges
		}
	}

	if !reflect.DeepEqual(previous.Enrichment, current.Enrichment) {
		enrichmentChanges := d.compareEnrichment(previous.Enrichment, current.Enrichment)
		if len(enrichmentChanges) > 0 {
			changes["enrichment"] = enrichmentChanges
		}
	}

	if !reflect.DeepEqual(previous.Retrieval, current.Retrieval) {
		retrievalChanges := d.compareRetrieval(previous.Retrieval, current.Retrieval)
		if len(retrievalChanges) > 0 {
			changes["retrieval"] = retrievalChanges
		}
	}

	if !reflect.DeepEqual(previous.LLM, current.LLM) {
		llmChanges := d.compareLLM(previous.LLM, current.LLM)
		if len(llmChanges) > 0 {
			changes["llm"] = llmChanges
		}
	}

	if !reflect.DeepEqual(previous.Diagnostics, current.Diagnostics) {
		diagnosticsChanges := d.compareDiagnostics(previous.Diagnostics, current.Diagnostics)
		if len(diagnosticsChanges) > 0 {
			changes["diagnostics"] = diagnosticsChanges
		}
	}

	return changes, nil
}

// compareReasoning compares reasoning contexts.
func (d *DiffTracker) compareReasoning(previous, current *ReasoningContext) map[string]interface{} {
	changes := make(map[string]interface{})

	if len(current.Intents) != len(previous.Intents) {
		changes["intents_added"] = len(current.Intents) - len(previous.Intents)
	}

	if len(current.Hypotheses) != len(previous.Hypotheses) {
		changes["hypotheses_added"] = len(current.Hypotheses) - len(previous.Hypotheses)
	}

	if len(current.Conclusions) != len(previous.Conclusions) {
		changes["conclusions_added"] = len(current.Conclusions) - len(previous.Conclusions)
	}

	if current.Summary != previous.Summary && current.Summary != "" {
		changes["summary_updated"] = true
	}

	return changes
}

// compareEnrichment compares enrichment contexts.
func (d *DiffTracker) compareEnrichment(previous, current *EnrichmentContext) map[string]interface{} {
	changes := make(map[string]interface{})

	if len(current.Facts) != len(previous.Facts) {
		changes["facts_added"] = len(current.Facts) - len(previous.Facts)
	}

	if len(current.DerivedKnowledge) != len(previous.DerivedKnowledge) {
		changes["knowledge_added"] = len(current.DerivedKnowledge) - len(previous.DerivedKnowledge)
	}

	if len(current.Relationships) != len(previous.Relationships) {
		changes["relationships_added"] = len(current.Relationships) - len(previous.Relationships)
	}

	return changes
}

// compareRetrieval compares retrieval contexts.
func (d *DiffTracker) compareRetrieval(previous, current *RetrievalContext) map[string]interface{} {
	changes := make(map[string]interface{})

	if len(current.Plans) != len(previous.Plans) {
		changes["plans_added"] = len(current.Plans) - len(previous.Plans)
	}

	if len(current.Queries) != len(previous.Queries) {
		changes["queries_added"] = len(current.Queries) - len(previous.Queries)
	}

	if len(current.Artifacts) != len(previous.Artifacts) {
		changes["artifacts_added"] = len(current.Artifacts) - len(previous.Artifacts)
	}

	return changes
}

// compareLLM compares LLM contexts.
func (d *DiffTracker) compareLLM(previous, current *LLMContext) map[string]interface{} {
	changes := make(map[string]interface{})

	if current.Provider != previous.Provider {
		changes["provider_changed"] = map[string]string{
			"from": previous.Provider,
			"to":   current.Provider,
		}
	}

	if current.Model != previous.Model {
		changes["model_changed"] = map[string]string{
			"from": previous.Model,
			"to":   current.Model,
		}
	}

	if current.Usage != nil && previous.Usage != nil {
		if current.Usage.TotalTokens != previous.Usage.TotalTokens {
			changes["tokens_added"] = current.Usage.TotalTokens - previous.Usage.TotalTokens
		}
		if current.Usage.CostUSD != previous.Usage.CostUSD {
			changes["cost_added"] = current.Usage.CostUSD - previous.Usage.CostUSD
		}
	}

	if len(current.Decisions) != len(previous.Decisions) {
		changes["decisions_added"] = len(current.Decisions) - len(previous.Decisions)
	}

	return changes
}

// compareDiagnostics compares diagnostics contexts.
func (d *DiffTracker) compareDiagnostics(previous, current *DiagnosticsContext) map[string]interface{} {
	changes := make(map[string]interface{})

	if len(current.Errors) != len(previous.Errors) {
		changes["errors_added"] = len(current.Errors) - len(previous.Errors)
	}

	if len(current.Warnings) != len(previous.Warnings) {
		changes["warnings_added"] = len(current.Warnings) - len(previous.Warnings)
	}

	if len(current.ValidationReports) != len(previous.ValidationReports) {
		changes["validation_reports_added"] = len(current.ValidationReports) - len(previous.ValidationReports)
	}

	return changes
}

// Summary returns a human-readable summary of the diff.
func (diff *ContextDiff) Summary() string {
	if len(diff.Changes) == 0 {
		return fmt.Sprintf("[%s] No changes", diff.AgentID)
	}

	summary := fmt.Sprintf("[%s] Changes:", diff.AgentID)
	for namespace, changes := range diff.Changes {
		changeMap, ok := changes.(map[string]interface{})
		if !ok {
			continue
		}
		for key, value := range changeMap {
			summary += fmt.Sprintf(" %s.%s=%v", namespace, key, value)
		}
	}

	return summary
}

// ToJSON converts the diff to JSON bytes.
func (diff *ContextDiff) ToJSON() ([]byte, error) {
	return json.Marshal(diff)
}

// ApplyDiff applies a diff to a context (for potential undo/redo functionality).
// This is a simplified version - real implementation would need more sophisticated merge logic.
func ApplyDiff(ctx *AgentContext, diff *ContextDiff) error {
	// This is a placeholder for future undo/redo functionality
	// Full implementation would require reversible operations
	return fmt.Errorf("apply diff not yet implemented")
}

// MergeDiffs merges multiple diffs into a single diff.
func MergeDiffs(diffs []*ContextDiff) *ContextDiff {
	if len(diffs) == 0 {
		return nil
	}

	merged := &ContextDiff{
		Timestamp: time.Now(),
		AgentID:   "merged",
		Changes:   make(map[string]interface{}),
	}

	// Merge changes from all diffs
	for _, diff := range diffs {
		for namespace, changes := range diff.Changes {
			if _, exists := merged.Changes[namespace]; !exists {
				merged.Changes[namespace] = make(map[string]interface{})
			}

			changeMap := merged.Changes[namespace].(map[string]interface{})
			diffChanges := changes.(map[string]interface{})

			for key, value := range diffChanges {
				changeMap[key] = value
			}
		}
	}

	return merged
}
