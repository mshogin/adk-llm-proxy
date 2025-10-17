package agents

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/mshogin/agents/internal/domain/models"
	"github.com/mshogin/agents/internal/domain/services"
)

// ValidationAgent validates completeness and consistency of reasoning pipeline.
//
// Design Principles:
// - Check completeness of required slots per intent
// - Validate logical consistency of reasoning chain
// - Detect dependency cycles and missing artifacts
// - Generate auto-fix hints for common issues
// - Flag potential issues without blocking execution
//
// Input Requirements:
// - reasoning.intents: List of detected intents
// - reasoning.hypotheses: Reasoning structure with dependencies
// - reasoning.conclusions: Generated conclusions
// - enrichment.facts: Normalized facts
//
// Output:
// - diagnostics.validation_reports[]: Validation results per check
// - diagnostics.errors[]: Critical validation errors
// - diagnostics.warnings[]: Non-critical validation warnings
type ValidationAgent struct {
	id string
}

// NewValidationAgent creates a new validation agent.
func NewValidationAgent() *ValidationAgent {
	return &ValidationAgent{
		id: "validation",
	}
}

// AgentID returns the unique identifier for this agent.
func (a *ValidationAgent) AgentID() string {
	return a.id
}

// Preconditions returns the list of context keys required before execution.
func (a *ValidationAgent) Preconditions() []string {
	return []string{
		"reasoning.intents",
		"reasoning.hypotheses",
		"reasoning.conclusions",
	}
}

// Postconditions returns the list of context keys guaranteed after execution.
func (a *ValidationAgent) Postconditions() []string {
	return []string{
		"diagnostics.validation_reports",
		"diagnostics.errors",
		"diagnostics.warnings",
	}
}

// Execute validates the reasoning pipeline for completeness and consistency.
func (a *ValidationAgent) Execute(ctx context.Context, agentContext *models.AgentContext) (*models.AgentContext, error) {
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

	// Store detailed agent trace
	agentTrace := map[string]interface{}{
		"agent_id": a.id,
		"input_intents_count": 0,
		"input_hypotheses_count": 0,
		"input_conclusions_count": 0,
		"input_facts_count": 0,
	}

	if newContext.Reasoning != nil {
		agentTrace["input_intents_count"] = len(newContext.Reasoning.Intents)
		agentTrace["input_hypotheses_count"] = len(newContext.Reasoning.Hypotheses)
		agentTrace["input_conclusions_count"] = len(newContext.Reasoning.Conclusions)
	}
	if newContext.Enrichment != nil {
		agentTrace["input_facts_count"] = len(newContext.Enrichment.Facts)
	}

	// Run validation checks
	reports := []models.ValidationReport{}
	errors := []models.ErrorReport{}
	warnings := []models.Warning{}

	// Check 1: Intent completeness
	intentReport, intentErrors, intentWarnings := a.validateIntentCompleteness(newContext)
	reports = append(reports, intentReport)
	errors = append(errors, intentErrors...)
	warnings = append(warnings, intentWarnings...)
	agentTrace["check_1_intent_completeness"] = map[string]interface{}{
		"passed": intentReport.Passed,
		"issues_count": len(intentReport.Issues),
	}

	// Check 2: Hypothesis consistency
	hypothesisReport, hypothesisErrors, hypothesisWarnings := a.validateHypothesisConsistency(newContext)
	reports = append(reports, hypothesisReport)
	errors = append(errors, hypothesisErrors...)
	warnings = append(warnings, hypothesisWarnings...)
	agentTrace["check_2_hypothesis_consistency"] = map[string]interface{}{
		"passed": hypothesisReport.Passed,
		"issues_count": len(hypothesisReport.Issues),
	}

	// Check 3: Dependency cycles
	cycleReport, cycleErrors, cycleWarnings := a.detectDependencyCycles(newContext)
	reports = append(reports, cycleReport)
	errors = append(errors, cycleErrors...)
	warnings = append(warnings, cycleWarnings...)
	agentTrace["check_3_dependency_cycles"] = map[string]interface{}{
		"passed": cycleReport.Passed,
		"issues_count": len(cycleReport.Issues),
	}

	// Check 4: Conclusion evidence
	conclusionReport, conclusionErrors, conclusionWarnings := a.validateConclusionEvidence(newContext)
	reports = append(reports, conclusionReport)
	errors = append(errors, conclusionErrors...)
	warnings = append(warnings, conclusionWarnings...)
	agentTrace["check_4_conclusion_evidence"] = map[string]interface{}{
		"passed": conclusionReport.Passed,
		"issues_count": len(conclusionReport.Issues),
	}

	// Check 5: Fact provenance
	provenanceReport, provenanceErrors, provenanceWarnings := a.validateFactProvenance(newContext)
	reports = append(reports, provenanceReport)
	errors = append(errors, provenanceErrors...)
	warnings = append(warnings, provenanceWarnings...)
	agentTrace["check_5_fact_provenance"] = map[string]interface{}{
		"passed": provenanceReport.Passed,
		"issues_count": len(provenanceReport.Issues),
	}

	agentTrace["total_reports"] = len(reports)
	agentTrace["total_errors"] = len(errors)
	agentTrace["total_warnings"] = len(warnings)

	// Write results
	if newContext.Diagnostics == nil {
		newContext.Diagnostics = &models.DiagnosticsContext{}
	}
	newContext.Diagnostics.ValidationReports = reports

	// Ensure Errors and Warnings are initialized to empty slices, not nil
	if newContext.Diagnostics.Errors == nil {
		newContext.Diagnostics.Errors = []models.ErrorReport{}
	}
	newContext.Diagnostics.Errors = append(newContext.Diagnostics.Errors, errors...)

	if newContext.Diagnostics.Warnings == nil {
		newContext.Diagnostics.Warnings = []models.Warning{}
	}
	newContext.Diagnostics.Warnings = append(newContext.Diagnostics.Warnings, warnings...)

	// Store final output in trace
	agentTrace["output_reports"] = reports
	agentTrace["output_errors"] = errors
	agentTrace["output_warnings"] = warnings

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
func (a *ValidationAgent) validatePreconditions(ctx *models.AgentContext) error {
	if ctx.Reasoning == nil {
		return fmt.Errorf("reasoning context is nil")
	}

	if len(ctx.Reasoning.Intents) == 0 {
		return fmt.Errorf("no intents found (required: reasoning.intents)")
	}

	if len(ctx.Reasoning.Hypotheses) == 0 {
		return fmt.Errorf("no hypotheses found (required: reasoning.hypotheses)")
	}

	if len(ctx.Reasoning.Conclusions) == 0 {
		return fmt.Errorf("no conclusions found (required: reasoning.conclusions)")
	}

	return nil
}

// validateIntentCompleteness checks if all intents have corresponding conclusions.
func (a *ValidationAgent) validateIntentCompleteness(ctx *models.AgentContext) (models.ValidationReport, []models.ErrorReport, []models.Warning) {
	report := models.ValidationReport{
		Timestamp: time.Now(),
		AgentID:   a.id,
		Passed:    true,
		Issues:    []string{},
		AutoFixes: []string{},
	}

	errors := []models.ErrorReport{}
	warnings := []models.Warning{}

	// Map conclusions by intent
	conclusionsByIntent := make(map[string]int)
	for _, c := range ctx.Reasoning.Conclusions {
		conclusionsByIntent[c.Intent]++
	}

	// Check each intent has at least one conclusion
	for _, intent := range ctx.Reasoning.Intents {
		if conclusionsByIntent[intent.Type] == 0 {
			report.Passed = false
			issue := fmt.Sprintf("Intent '%s' has no conclusions", intent.Type)
			report.Issues = append(report.Issues, issue)

			warnings = append(warnings, models.Warning{
				Timestamp: time.Now(),
				AgentID:   a.id,
				Message:   issue,
			})

			// Auto-fix hint
			autoFix := fmt.Sprintf("Ensure Inference Agent generates conclusions for '%s' intent", intent.Type)
			report.AutoFixes = append(report.AutoFixes, autoFix)
		}
	}

	return report, errors, warnings
}

// validateHypothesisConsistency checks if hypotheses form a valid reasoning chain.
func (a *ValidationAgent) validateHypothesisConsistency(ctx *models.AgentContext) (models.ValidationReport, []models.ErrorReport, []models.Warning) {
	report := models.ValidationReport{
		Timestamp: time.Now(),
		AgentID:   a.id,
		Passed:    true,
		Issues:    []string{},
		AutoFixes: []string{},
	}

	errors := []models.ErrorReport{}
	warnings := []models.Warning{}

	// Build hypothesis ID map
	hypothesisMap := make(map[string]bool)
	for _, h := range ctx.Reasoning.Hypotheses {
		hypothesisMap[h.ID] = true
	}

	// Check dependencies
	for _, h := range ctx.Reasoning.Hypotheses {
		for _, depID := range h.Dependencies {
			if !hypothesisMap[depID] {
				report.Passed = false
				issue := fmt.Sprintf("Hypothesis '%s' depends on missing hypothesis '%s'", h.ID, depID)
				report.Issues = append(report.Issues, issue)

				errors = append(errors, models.ErrorReport{
					Timestamp: time.Now(),
					AgentID:   a.id,
					Message:   issue,
					Severity:  "error",
				})

				// Auto-fix hint
				autoFix := fmt.Sprintf("Add missing hypothesis '%s' or remove dependency from '%s'", depID, h.ID)
				report.AutoFixes = append(report.AutoFixes, autoFix)
			}
		}
	}

	return report, errors, warnings
}

// detectDependencyCycles detects circular dependencies in hypothesis graph.
func (a *ValidationAgent) detectDependencyCycles(ctx *models.AgentContext) (models.ValidationReport, []models.ErrorReport, []models.Warning) {
	report := models.ValidationReport{
		Timestamp: time.Now(),
		AgentID:   a.id,
		Passed:    true,
		Issues:    []string{},
		AutoFixes: []string{},
	}

	errors := []models.ErrorReport{}
	warnings := []models.Warning{}

	// Build adjacency list
	graph := make(map[string][]string)
	for _, h := range ctx.Reasoning.Hypotheses {
		graph[h.ID] = h.Dependencies
	}

	// Detect cycles using DFS
	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	for _, h := range ctx.Reasoning.Hypotheses {
		if !visited[h.ID] {
			if cycle := a.detectCycleDFS(h.ID, graph, visited, recStack, []string{}); len(cycle) > 0 {
				report.Passed = false
				issue := fmt.Sprintf("Dependency cycle detected: %s", strings.Join(cycle, " → "))
				report.Issues = append(report.Issues, issue)

				errors = append(errors, models.ErrorReport{
					Timestamp: time.Now(),
					AgentID:   a.id,
					Message:   issue,
					Severity:  "error",
				})

				// Auto-fix hint
				autoFix := fmt.Sprintf("Break cycle by removing dependency from '%s' to '%s'", cycle[len(cycle)-1], cycle[0])
				report.AutoFixes = append(report.AutoFixes, autoFix)
			}
		}
	}

	return report, errors, warnings
}

// detectCycleDFS performs DFS to detect cycles.
func (a *ValidationAgent) detectCycleDFS(node string, graph map[string][]string, visited, recStack map[string]bool, path []string) []string {
	visited[node] = true
	recStack[node] = true
	path = append(path, node)

	for _, neighbor := range graph[node] {
		if !visited[neighbor] {
			if cycle := a.detectCycleDFS(neighbor, graph, visited, recStack, path); len(cycle) > 0 {
				return cycle
			}
		} else if recStack[neighbor] {
			// Cycle detected, extract cycle path
			cycleStart := 0
			for i, n := range path {
				if n == neighbor {
					cycleStart = i
					break
				}
			}
			return path[cycleStart:]
		}
	}

	recStack[node] = false
	return []string{}
}

// validateConclusionEvidence checks if conclusions have supporting evidence.
func (a *ValidationAgent) validateConclusionEvidence(ctx *models.AgentContext) (models.ValidationReport, []models.ErrorReport, []models.Warning) {
	report := models.ValidationReport{
		Timestamp: time.Now(),
		AgentID:   a.id,
		Passed:    true,
		Issues:    []string{},
		AutoFixes: []string{},
	}

	errors := []models.ErrorReport{}
	warnings := []models.Warning{}

	// Check each conclusion has evidence
	for _, c := range ctx.Reasoning.Conclusions {
		if len(c.Evidence) == 0 {
			report.Passed = false
			issue := fmt.Sprintf("Conclusion '%s' has no supporting evidence", c.ID)
			report.Issues = append(report.Issues, issue)

			warnings = append(warnings, models.Warning{
				Timestamp: time.Now(),
				AgentID:   a.id,
				Message:   issue,
			})

			// Auto-fix hint
			autoFix := fmt.Sprintf("Link conclusion '%s' to relevant facts or knowledge items", c.ID)
			report.AutoFixes = append(report.AutoFixes, autoFix)
		}

		// Validate evidence references exist
		if ctx.Enrichment != nil {
			for _, evidenceRef := range c.Evidence {
				if !a.evidenceExists(evidenceRef, ctx) {
					report.Passed = false
					issue := fmt.Sprintf("Conclusion '%s' references non-existent evidence '%s'", c.ID, evidenceRef)
					report.Issues = append(report.Issues, issue)

					errors = append(errors, models.ErrorReport{
						Timestamp: time.Now(),
						AgentID:   a.id,
						Message:   issue,
						Severity:  "error",
					})

					// Auto-fix hint
					autoFix := fmt.Sprintf("Remove invalid evidence reference '%s' from conclusion '%s'", evidenceRef, c.ID)
					report.AutoFixes = append(report.AutoFixes, autoFix)
				}
			}
		}
	}

	return report, errors, warnings
}

// evidenceExists checks if evidence reference exists in context.
func (a *ValidationAgent) evidenceExists(evidenceRef string, ctx *models.AgentContext) bool {
	parts := strings.SplitN(evidenceRef, ":", 2)
	if len(parts) != 2 {
		return false
	}

	refType := parts[0]
	refID := parts[1]

	switch refType {
	case "fact":
		for _, fact := range ctx.Enrichment.Facts {
			if fact.ID == refID {
				return true
			}
		}
	case "knowledge":
		for _, k := range ctx.Enrichment.DerivedKnowledge {
			if k.ID == refID {
				return true
			}
		}
	}

	return false
}

// validateFactProvenance checks if facts have complete provenance information.
func (a *ValidationAgent) validateFactProvenance(ctx *models.AgentContext) (models.ValidationReport, []models.ErrorReport, []models.Warning) {
	report := models.ValidationReport{
		Timestamp: time.Now(),
		AgentID:   a.id,
		Passed:    true,
		Issues:    []string{},
		AutoFixes: []string{},
	}

	errors := []models.ErrorReport{}
	warnings := []models.Warning{}

	if ctx.Enrichment == nil {
		return report, errors, warnings
	}

	// Check each fact has provenance
	for _, fact := range ctx.Enrichment.Facts {
		if fact.Provenance == nil || len(fact.Provenance) == 0 {
			report.Passed = false
			issue := fmt.Sprintf("Fact '%s' missing provenance information", fact.ID)
			report.Issues = append(report.Issues, issue)

			warnings = append(warnings, models.Warning{
				Timestamp: time.Now(),
				AgentID:   a.id,
				Message:   issue,
			})

			// Auto-fix hint
			autoFix := fmt.Sprintf("Add provenance metadata (artifact_id, source) to fact '%s'", fact.ID)
			report.AutoFixes = append(report.AutoFixes, autoFix)
		}

		// Check confidence score is valid
		if fact.Confidence < 0.0 || fact.Confidence > 1.0 {
			report.Passed = false
			issue := fmt.Sprintf("Fact '%s' has invalid confidence score %.2f (must be 0.0-1.0)", fact.ID, fact.Confidence)
			report.Issues = append(report.Issues, issue)

			errors = append(errors, models.ErrorReport{
				Timestamp: time.Now(),
				AgentID:   a.id,
				Message:   issue,
				Severity:  "error",
			})

			// Auto-fix hint
			autoFix := fmt.Sprintf("Normalize confidence score for fact '%s' to valid range [0.0, 1.0]", fact.ID)
			report.AutoFixes = append(report.AutoFixes, autoFix)
		}
	}

	return report, errors, warnings
}

// recordAgentRun records execution in audit trail.
func (a *ValidationAgent) recordAgentRun(ctx *models.AgentContext, duration time.Duration, status string, err error) {
	run := models.AgentRun{
		Timestamp:  time.Now(),
		AgentID:    a.id,
		Status:     status,
		DurationMS: duration.Milliseconds(),
		KeysWritten: []string{
			"diagnostics.validation_reports",
			"diagnostics.errors",
			"diagnostics.warnings",
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

	if ctx.Diagnostics.Performance == nil {
		ctx.Diagnostics.Performance = &models.PerformanceData{}
	}

	if ctx.Diagnostics.Performance.AgentMetrics == nil {
		ctx.Diagnostics.Performance.AgentMetrics = make(map[string]*models.AgentMetrics)
	}

	ctx.Diagnostics.Performance.AgentMetrics[a.id] = &models.AgentMetrics{
		DurationMS: duration.Milliseconds(),
		LLMCalls:   0, // No LLM calls for rule-based validation
		Status:     status,
	}
}

// GetMetadata returns agent metadata.
func (a *ValidationAgent) GetMetadata() services.AgentMetadata {
	return services.AgentMetadata{
		ID:          a.id,
		Name:        "Validation Agent",
		Description: "Validates completeness and consistency of reasoning pipeline with auto-fix hints",
		Version:     "1.0.0",
		Author:      "ADK LLM Proxy",
		Tags:        []string{"validation", "consistency", "completeness", "quality-assurance", "debugging"},
		Dependencies: []string{"inference"},
	}
}

// GetCapabilities returns agent capabilities.
func (a *ValidationAgent) GetCapabilities() services.AgentCapabilities {
	return services.AgentCapabilities{
		SupportsParallelExecution: false,
		SupportsRetry:             true,
		RequiresLLM:               false, // Rule-based validation
		IsDeterministic:           true,
		EstimatedDuration:         100, // ~100ms for validation
	}
}
