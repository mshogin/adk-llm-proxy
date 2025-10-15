package agents

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/mshogin/agents/internal/domain/models"
	"github.com/mshogin/agents/internal/domain/services"
)

// InferenceAgent makes conclusions based on facts, hypotheses, and goals.
//
// Design Principles:
// - Make inferences from enriched facts and knowledge
// - Assess conclusion confidence based on supporting evidence
// - Generate alternative interpretations for ambiguous cases
// - Use deterministic rules for simple inferences
// - Flag complex cases requiring LLM synthesis
//
// Input Requirements:
// - reasoning.intents: Original intents with goals
// - reasoning.hypotheses: Reasoning structure with dependencies
// - enrichment.facts: Normalized facts with provenance
// - enrichment.derived_knowledge: Higher-level knowledge items
//
// Output:
// - reasoning.conclusions[]: Inferences with confidence scores
// - reasoning.alternatives[]: Alternative interpretations
// - reasoning.inference_chain[]: Step-by-step reasoning trace
type InferenceAgent struct {
	id string
}

// NewInferenceAgent creates a new inference agent.
func NewInferenceAgent() *InferenceAgent {
	return &InferenceAgent{
		id: "inference",
	}
}

// AgentID returns the unique identifier for this agent.
func (a *InferenceAgent) AgentID() string {
	return a.id
}

// Preconditions returns the list of context keys required before execution.
func (a *InferenceAgent) Preconditions() []string {
	return []string{
		"reasoning.intents",
		"reasoning.hypotheses",
		"enrichment.facts",
	}
}

// Postconditions returns the list of context keys guaranteed after execution.
func (a *InferenceAgent) Postconditions() []string {
	return []string{
		"reasoning.conclusions",
		"reasoning.alternatives",
		"reasoning.inference_chain",
	}
}

// Execute makes inferences based on facts, hypotheses, and goals.
func (a *InferenceAgent) Execute(ctx context.Context, agentContext *models.AgentContext) (*models.AgentContext, error) {
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

	// Extract inputs
	intents := newContext.Reasoning.Intents
	hypotheses := newContext.Reasoning.Hypotheses
	facts := newContext.Enrichment.Facts
	knowledge := newContext.Enrichment.DerivedKnowledge

	// Generate inference chain
	inferenceChain := a.buildInferenceChain(hypotheses, facts, knowledge)

	// Make conclusions per intent
	conclusions := a.makeConclusions(intents, hypotheses, facts, knowledge, inferenceChain)

	// Generate alternative interpretations
	alternatives := a.generateAlternatives(conclusions, facts)

	// Write results
	newContext.Reasoning.Conclusions = conclusions
	newContext.Reasoning.Alternatives = alternatives
	newContext.Reasoning.InferenceChain = inferenceChain

	// Track execution
	duration := time.Since(startTime)
	a.recordAgentRun(newContext, duration, "success", nil)

	return newContext, nil
}

// validatePreconditions checks required context keys.
func (a *InferenceAgent) validatePreconditions(ctx *models.AgentContext) error {
	if ctx.Reasoning == nil {
		return fmt.Errorf("reasoning context is nil")
	}

	if len(ctx.Reasoning.Intents) == 0 {
		return fmt.Errorf("no intents found (required: reasoning.intents)")
	}

	if len(ctx.Reasoning.Hypotheses) == 0 {
		return fmt.Errorf("no hypotheses found (required: reasoning.hypotheses)")
	}

	if ctx.Enrichment == nil || len(ctx.Enrichment.Facts) == 0 {
		return fmt.Errorf("no facts found (required: enrichment.facts)")
	}

	return nil
}

// buildInferenceChain constructs step-by-step reasoning trace.
func (a *InferenceAgent) buildInferenceChain(hypotheses []models.Hypothesis, facts []models.Fact, knowledge []models.Knowledge) []models.InferenceStep {
	chain := []models.InferenceStep{}
	stepID := 0

	// Build chain from hypothesis dependencies
	for _, h := range hypotheses {
		step := models.InferenceStep{
			ID:          fmt.Sprintf("step%d", stepID),
			Description: fmt.Sprintf("Verify: %s", h.Description),
			Hypothesis:  h.ID,
			Evidence:    a.findSupportingEvidence(h, facts, knowledge),
			Confidence:  a.calculateEvidenceConfidence(h, facts, knowledge),
		}
		chain = append(chain, step)
		stepID++
	}

	return chain
}

// findSupportingEvidence finds facts/knowledge supporting a hypothesis.
func (a *InferenceAgent) findSupportingEvidence(h models.Hypothesis, facts []models.Fact, knowledge []models.Knowledge) []string {
	evidence := []string{}

	// Match facts by keyword overlap
	keywords := a.extractKeywords(h.Description)
	for _, fact := range facts {
		if a.hasKeywordOverlap(keywords, fact.Content) {
			evidence = append(evidence, fmt.Sprintf("fact:%s", fact.ID))
		}
	}

	// Match knowledge by keyword overlap
	for _, k := range knowledge {
		if a.hasKeywordOverlap(keywords, k.Content) {
			evidence = append(evidence, fmt.Sprintf("knowledge:%s", k.ID))
		}
	}

	return evidence
}

// extractKeywords extracts important words from text.
func (a *InferenceAgent) extractKeywords(text string) []string {
	// Simple keyword extraction (lowercase, split by spaces)
	words := strings.Fields(strings.ToLower(text))

	// Filter stop words
	stopWords := map[string]bool{
		"the": true, "a": true, "an": true, "and": true, "or": true,
		"but": true, "is": true, "are": true, "was": true, "were": true,
		"in": true, "on": true, "at": true, "to": true, "for": true,
		"of": true, "with": true, "by": true, "from": true, "as": true,
	}

	keywords := []string{}
	for _, word := range words {
		if !stopWords[word] && len(word) > 2 {
			keywords = append(keywords, word)
		}
	}

	return keywords
}

// hasKeywordOverlap checks if text contains any keywords.
func (a *InferenceAgent) hasKeywordOverlap(keywords []string, text string) bool {
	textLower := strings.ToLower(text)
	for _, keyword := range keywords {
		if strings.Contains(textLower, keyword) {
			return true
		}
	}
	return false
}

// calculateEvidenceConfidence calculates confidence based on supporting evidence.
func (a *InferenceAgent) calculateEvidenceConfidence(h models.Hypothesis, facts []models.Fact, knowledge []models.Knowledge) float64 {
	evidence := a.findSupportingEvidence(h, facts, knowledge)

	// Confidence based on evidence count and quality
	if len(evidence) == 0 {
		return 0.20 // Minimal confidence (no evidence)
	} else if len(evidence) == 1 {
		return 0.50 // Moderate confidence (single evidence)
	} else if len(evidence) == 2 {
		return 0.70 // Good confidence (multiple evidence)
	} else {
		return 0.90 // High confidence (strong evidence)
	}
}

// makeConclusions generates conclusions for each intent.
func (a *InferenceAgent) makeConclusions(intents []models.Intent, hypotheses []models.Hypothesis, facts []models.Fact, knowledge []models.Knowledge, chain []models.InferenceStep) []models.Conclusion {
	conclusions := []models.Conclusion{}
	conclusionID := 0

	for _, intent := range intents {
		// Generate conclusions based on intent type
		intentConclusions := a.makeConclusionsForIntent(intent, hypotheses, facts, knowledge, chain, &conclusionID)
		conclusions = append(conclusions, intentConclusions...)
	}

	return conclusions
}

// makeConclusionsForIntent generates conclusions for a specific intent.
func (a *InferenceAgent) makeConclusionsForIntent(intent models.Intent, hypotheses []models.Hypothesis, facts []models.Fact, knowledge []models.Knowledge, chain []models.InferenceStep, conclusionID *int) []models.Conclusion {
	conclusions := []models.Conclusion{}

	switch intent.Type {
	case "query_commits":
		conclusions = append(conclusions, a.makeCommitConclusions(intent, facts, knowledge, chain, conclusionID)...)
	case "query_issues":
		conclusions = append(conclusions, a.makeIssueConclusions(intent, facts, knowledge, chain, conclusionID)...)
	case "query_analytics":
		conclusions = append(conclusions, a.makeAnalyticsConclusions(intent, facts, knowledge, chain, conclusionID)...)
	case "query_status":
		conclusions = append(conclusions, a.makeStatusConclusions(intent, facts, knowledge, chain, conclusionID)...)
	default:
		// Generic conclusion
		conclusions = append(conclusions, models.Conclusion{
			ID:          fmt.Sprintf("c%d", *conclusionID),
			Description: fmt.Sprintf("Data retrieved for %s intent", intent.Type),
			Confidence:  0.70,
			Evidence:    a.collectAllEvidence(facts, knowledge),
			Intent:      intent.Type,
		})
		*conclusionID++
	}

	return conclusions
}

// makeCommitConclusions generates conclusions for commit queries.
func (a *InferenceAgent) makeCommitConclusions(intent models.Intent, facts []models.Fact, knowledge []models.Knowledge, chain []models.InferenceStep, conclusionID *int) []models.Conclusion {
	conclusions := []models.Conclusion{}

	// Count commit-related facts
	commitFacts := a.countFactsBySource(facts, "gitlab")

	if commitFacts > 0 {
		conclusions = append(conclusions, models.Conclusion{
			ID:          fmt.Sprintf("c%d", *conclusionID),
			Description: fmt.Sprintf("Found %d commit(s) from GitLab", commitFacts),
			Confidence:  0.95,
			Evidence:    a.collectEvidenceBySource(facts, knowledge, "gitlab"),
			Intent:      intent.Type,
		})
		*conclusionID++
	} else {
		conclusions = append(conclusions, models.Conclusion{
			ID:          fmt.Sprintf("c%d", *conclusionID),
			Description: "No commits found matching criteria",
			Confidence:  0.80,
			Evidence:    []string{},
			Intent:      intent.Type,
		})
		*conclusionID++
	}

	return conclusions
}

// makeIssueConclusions generates conclusions for issue queries.
func (a *InferenceAgent) makeIssueConclusions(intent models.Intent, facts []models.Fact, knowledge []models.Knowledge, chain []models.InferenceStep, conclusionID *int) []models.Conclusion {
	conclusions := []models.Conclusion{}

	// Count issue-related facts
	issueFacts := a.countFactsBySource(facts, "youtrack")

	if issueFacts > 0 {
		conclusions = append(conclusions, models.Conclusion{
			ID:          fmt.Sprintf("c%d", *conclusionID),
			Description: fmt.Sprintf("Found %d issue(s) from YouTrack", issueFacts),
			Confidence:  0.95,
			Evidence:    a.collectEvidenceBySource(facts, knowledge, "youtrack"),
			Intent:      intent.Type,
		})
		*conclusionID++
	} else {
		conclusions = append(conclusions, models.Conclusion{
			ID:          fmt.Sprintf("c%d", *conclusionID),
			Description: "No issues found matching criteria",
			Confidence:  0.80,
			Evidence:    []string{},
			Intent:      intent.Type,
		})
		*conclusionID++
	}

	return conclusions
}

// makeAnalyticsConclusions generates conclusions for analytics queries.
func (a *InferenceAgent) makeAnalyticsConclusions(intent models.Intent, facts []models.Fact, knowledge []models.Knowledge, chain []models.InferenceStep, conclusionID *int) []models.Conclusion {
	conclusions := []models.Conclusion{}

	// Aggregate analytics from multiple sources
	sources := a.getUniqueSources(facts)

	for _, source := range sources {
		factCount := a.countFactsBySource(facts, source)
		if factCount > 0 {
			conclusions = append(conclusions, models.Conclusion{
				ID:          fmt.Sprintf("c%d", *conclusionID),
				Description: fmt.Sprintf("Aggregated %d metrics from %s", factCount, source),
				Confidence:  0.85,
				Evidence:    a.collectEvidenceBySource(facts, knowledge, source),
				Intent:      intent.Type,
			})
			*conclusionID++
		}
	}

	return conclusions
}

// makeStatusConclusions generates conclusions for status queries.
func (a *InferenceAgent) makeStatusConclusions(intent models.Intent, facts []models.Fact, knowledge []models.Knowledge, chain []models.InferenceStep, conclusionID *int) []models.Conclusion {
	conclusions := []models.Conclusion{}

	// Determine system health from facts
	totalFacts := len(facts)

	if totalFacts > 10 {
		conclusions = append(conclusions, models.Conclusion{
			ID:          fmt.Sprintf("c%d", *conclusionID),
			Description: "System is active with significant data",
			Confidence:  0.90,
			Evidence:    a.collectAllEvidence(facts, knowledge),
			Intent:      intent.Type,
		})
	} else if totalFacts > 0 {
		conclusions = append(conclusions, models.Conclusion{
			ID:          fmt.Sprintf("c%d", *conclusionID),
			Description: "System has limited activity",
			Confidence:  0.75,
			Evidence:    a.collectAllEvidence(facts, knowledge),
			Intent:      intent.Type,
		})
	} else {
		conclusions = append(conclusions, models.Conclusion{
			ID:          fmt.Sprintf("c%d", *conclusionID),
			Description: "No recent activity detected",
			Confidence:  0.80,
			Evidence:    []string{},
			Intent:      intent.Type,
		})
	}
	*conclusionID++

	return conclusions
}

// countFactsBySource counts facts from a specific source.
func (a *InferenceAgent) countFactsBySource(facts []models.Fact, source string) int {
	count := 0
	for _, fact := range facts {
		if fact.Source == source {
			count++
		}
	}
	return count
}

// getUniqueSources returns list of unique sources.
func (a *InferenceAgent) getUniqueSources(facts []models.Fact) []string {
	sourceMap := make(map[string]bool)
	for _, fact := range facts {
		sourceMap[fact.Source] = true
	}

	sources := []string{}
	for source := range sourceMap {
		sources = append(sources, source)
	}
	return sources
}

// collectEvidenceBySource collects evidence references from a specific source.
func (a *InferenceAgent) collectEvidenceBySource(facts []models.Fact, knowledge []models.Knowledge, source string) []string {
	evidence := []string{}

	for _, fact := range facts {
		if fact.Source == source {
			evidence = append(evidence, fmt.Sprintf("fact:%s", fact.ID))
		}
	}

	for _, k := range knowledge {
		if strings.Contains(strings.ToLower(k.Content), strings.ToLower(source)) {
			evidence = append(evidence, fmt.Sprintf("knowledge:%s", k.ID))
		}
	}

	return evidence
}

// collectAllEvidence collects all evidence references.
func (a *InferenceAgent) collectAllEvidence(facts []models.Fact, knowledge []models.Knowledge) []string {
	evidence := []string{}

	for _, fact := range facts {
		evidence = append(evidence, fmt.Sprintf("fact:%s", fact.ID))
	}

	for _, k := range knowledge {
		evidence = append(evidence, fmt.Sprintf("knowledge:%s", k.ID))
	}

	return evidence
}

// generateAlternatives generates alternative interpretations for ambiguous conclusions.
func (a *InferenceAgent) generateAlternatives(conclusions []models.Conclusion, facts []models.Fact) []models.Alternative {
	alternatives := []models.Alternative{}
	altID := 0

	for _, c := range conclusions {
		// Generate alternatives for low-confidence conclusions
		if c.Confidence < 0.70 {
			alt := models.Alternative{
				ID:          fmt.Sprintf("alt%d", altID),
				Conclusion:  c.ID,
				Description: fmt.Sprintf("Alternative: %s (requires more data)", c.Description),
				Confidence:  c.Confidence * 0.8, // Lower confidence for alternatives
			}
			alternatives = append(alternatives, alt)
			altID++
		}
	}

	return alternatives
}

// recordAgentRun records execution in audit trail.
func (a *InferenceAgent) recordAgentRun(ctx *models.AgentContext, duration time.Duration, status string, err error) {
	run := models.AgentRun{
		Timestamp:  time.Now(),
		AgentID:    a.id,
		Status:     status,
		DurationMS: duration.Milliseconds(),
		KeysWritten: []string{
			"reasoning.conclusions",
			"reasoning.alternatives",
			"reasoning.inference_chain",
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
		LLMCalls:   0, // No LLM calls for rule-based inference
		Status:     status,
	}
}

// GetMetadata returns agent metadata.
func (a *InferenceAgent) GetMetadata() services.AgentMetadata {
	return services.AgentMetadata{
		ID:          a.id,
		Name:        "Inference Agent",
		Description: "Makes conclusions based on facts, hypotheses, and goals with confidence scoring and alternative interpretations",
		Version:     "1.0.0",
		Author:      "ADK LLM Proxy",
		Tags:        []string{"inference", "conclusions", "reasoning", "confidence", "alternatives"},
		Dependencies: []string{"context_synthesizer"},
	}
}

// GetCapabilities returns agent capabilities.
func (a *InferenceAgent) GetCapabilities() services.AgentCapabilities {
	return services.AgentCapabilities{
		SupportsParallelExecution: false,
		SupportsRetry:             true,
		RequiresLLM:               false, // Rule-based for now
		IsDeterministic:           true,
		EstimatedDuration:         150, // ~150ms for inference
	}
}
