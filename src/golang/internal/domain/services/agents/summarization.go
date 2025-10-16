package agents

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/mshogin/agents/internal/domain/models"
	"github.com/mshogin/agents/internal/domain/services"
)

// SummarizationAgent generates executive summaries and structured output artifacts.
//
// Design Principles:
// - Generate concise executive summary of reasoning process
// - Create structured output artifacts (reports, command lists)
// - Provide context diff overview
// - Format output for downstream systems
// - Support multiple output formats
//
// Input Requirements:
// - reasoning.intents: Detected intents
// - reasoning.hypotheses: Reasoning structure
// - reasoning.conclusions: Generated conclusions
// - enrichment.facts: Normalized facts
//
// Output:
// - reasoning.summary: Executive summary of reasoning process
// - reasoning.artifacts: Structured output artifacts (reports, commands)
type SummarizationAgent struct {
	id string
}

// NewSummarizationAgent creates a new summarization agent.
func NewSummarizationAgent() *SummarizationAgent {
	return &SummarizationAgent{
		id: "summarization",
	}
}

// AgentID returns the unique identifier for this agent.
func (a *SummarizationAgent) AgentID() string {
	return a.id
}

// Preconditions returns the list of context keys required before execution.
func (a *SummarizationAgent) Preconditions() []string {
	return []string{
		"reasoning.intents",
		"reasoning.conclusions",
	}
}

// Postconditions returns the list of context keys guaranteed after execution.
func (a *SummarizationAgent) Postconditions() []string {
	return []string{
		"reasoning.summary",
		"reasoning.artifacts",
	}
}

// Execute generates executive summary and structured output artifacts.
func (a *SummarizationAgent) Execute(ctx context.Context, agentContext *models.AgentContext) (*models.AgentContext, error) {
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

	// Generate executive summary
	summary := a.generateExecutiveSummary(newContext)

	// Generate structured artifacts
	artifacts := a.generateArtifacts(newContext)

	// Write results
	if newContext.Reasoning == nil {
		newContext.Reasoning = &models.ReasoningContext{}
	}
	newContext.Reasoning.Summary = summary
	newContext.Reasoning.Artifacts = artifacts

	// Track execution
	duration := time.Since(startTime)
	a.recordAgentRun(newContext, duration, "success", nil)

	return newContext, nil
}

// validatePreconditions checks required context keys.
func (a *SummarizationAgent) validatePreconditions(ctx *models.AgentContext) error {
	if ctx.Reasoning == nil {
		return fmt.Errorf("reasoning context is nil")
	}

	if len(ctx.Reasoning.Intents) == 0 {
		return fmt.Errorf("no intents found (required: reasoning.intents)")
	}

	if len(ctx.Reasoning.Conclusions) == 0 {
		return fmt.Errorf("no conclusions found (required: reasoning.conclusions)")
	}

	return nil
}

// generateExecutiveSummary creates high-level summary of reasoning process.
func (a *SummarizationAgent) generateExecutiveSummary(ctx *models.AgentContext) string {
	var parts []string

	// Intent summary
	intentSummary := a.summarizeIntents(ctx.Reasoning.Intents)
	if intentSummary != "" {
		parts = append(parts, intentSummary)
	}

	// Data summary
	dataSummary := a.summarizeData(ctx)
	if dataSummary != "" {
		parts = append(parts, dataSummary)
	}

	// Conclusion summary
	conclusionSummary := a.summarizeConclusions(ctx.Reasoning.Conclusions)
	if conclusionSummary != "" {
		parts = append(parts, conclusionSummary)
	}

	// Validation summary
	validationSummary := a.summarizeValidation(ctx)
	if validationSummary != "" {
		parts = append(parts, validationSummary)
	}

	return strings.Join(parts, " ")
}

// summarizeIntents creates intent summary.
func (a *SummarizationAgent) summarizeIntents(intents []models.Intent) string {
	if len(intents) == 0 {
		return ""
	}

	if len(intents) == 1 {
		return fmt.Sprintf("Detected intent: %s.", intents[0].Type)
	}

	intentTypes := make([]string, len(intents))
	for i, intent := range intents {
		intentTypes[i] = intent.Type
	}

	return fmt.Sprintf("Detected %d intents: %s.", len(intents), strings.Join(intentTypes, ", "))
}

// summarizeData creates data summary from facts.
func (a *SummarizationAgent) summarizeData(ctx *models.AgentContext) string {
	if ctx.Enrichment == nil || len(ctx.Enrichment.Facts) == 0 {
		return "No data retrieved."
	}

	// Count facts by source
	sourceCount := make(map[string]int)
	for _, fact := range ctx.Enrichment.Facts {
		sourceCount[fact.Source]++
	}

	if len(sourceCount) == 1 {
		for source, count := range sourceCount {
			return fmt.Sprintf("Retrieved %d fact(s) from %s.", count, source)
		}
	}

	// Multiple sources
	var sourceParts []string
	for source, count := range sourceCount {
		sourceParts = append(sourceParts, fmt.Sprintf("%d from %s", count, source))
	}

	return fmt.Sprintf("Retrieved %d fact(s) (%s).", len(ctx.Enrichment.Facts), strings.Join(sourceParts, ", "))
}

// summarizeConclusions creates conclusion summary.
func (a *SummarizationAgent) summarizeConclusions(conclusions []models.Conclusion) string {
	if len(conclusions) == 0 {
		return "No conclusions generated."
	}

	// Calculate average confidence
	totalConfidence := 0.0
	for _, c := range conclusions {
		totalConfidence += c.Confidence
	}
	avgConfidence := totalConfidence / float64(len(conclusions))

	confidenceLevel := "low"
	if avgConfidence >= 0.9 {
		confidenceLevel = "high"
	} else if avgConfidence >= 0.7 {
		confidenceLevel = "good"
	} else if avgConfidence >= 0.5 {
		confidenceLevel = "moderate"
	}

	if len(conclusions) == 1 {
		return fmt.Sprintf("Generated 1 conclusion with %s confidence (%.2f).", confidenceLevel, avgConfidence)
	}

	return fmt.Sprintf("Generated %d conclusions with %s average confidence (%.2f).", len(conclusions), confidenceLevel, avgConfidence)
}

// summarizeValidation creates validation summary.
func (a *SummarizationAgent) summarizeValidation(ctx *models.AgentContext) string {
	if ctx.Diagnostics == nil {
		return ""
	}

	errorCount := len(ctx.Diagnostics.Errors)
	warningCount := len(ctx.Diagnostics.Warnings)

	if errorCount == 0 && warningCount == 0 {
		return "Validation passed with no issues."
	}

	var parts []string
	if errorCount > 0 {
		parts = append(parts, fmt.Sprintf("%d error(s)", errorCount))
	}
	if warningCount > 0 {
		parts = append(parts, fmt.Sprintf("%d warning(s)", warningCount))
	}

	return fmt.Sprintf("Validation detected %s.", strings.Join(parts, " and "))
}

// generateArtifacts creates structured output artifacts.
func (a *SummarizationAgent) generateArtifacts(ctx *models.AgentContext) []models.Artifact {
	artifacts := []models.Artifact{}

	// Report artifact
	report := a.generateReport(ctx)
	artifacts = append(artifacts, models.Artifact{
		ID:      "report",
		Type:    "report",
		Source:  "summarization",
		Content: report,
	})

	// Command list artifact (if applicable)
	if commands := a.generateCommandList(ctx); commands != "" {
		artifacts = append(artifacts, models.Artifact{
			ID:      "commands",
			Type:    "command_list",
			Source:  "summarization",
			Content: commands,
		})
	}

	// Context diff artifact
	if diff := a.generateContextDiff(ctx); diff != "" {
		artifacts = append(artifacts, models.Artifact{
			ID:      "context_diff",
			Type:    "diff",
			Source:  "summarization",
			Content: diff,
		})
	}

	return artifacts
}

// generateReport creates detailed reasoning report.
func (a *SummarizationAgent) generateReport(ctx *models.AgentContext) string {
	var sections []string

	// Header
	sections = append(sections, "# Reasoning Report")
	sections = append(sections, "")

	// Intents section
	if len(ctx.Reasoning.Intents) > 0 {
		sections = append(sections, "## Detected Intents")
		for _, intent := range ctx.Reasoning.Intents {
			sections = append(sections, fmt.Sprintf("- **%s** (confidence: %.2f)", intent.Type, intent.Confidence))
		}
		sections = append(sections, "")
	}

	// Data section
	if ctx.Enrichment != nil && len(ctx.Enrichment.Facts) > 0 {
		sections = append(sections, "## Retrieved Data")

		// Group by source
		sourceCount := make(map[string]int)
		for _, fact := range ctx.Enrichment.Facts {
			sourceCount[fact.Source]++
		}

		for source, count := range sourceCount {
			sections = append(sections, fmt.Sprintf("- **%s**: %d fact(s)", source, count))
		}
		sections = append(sections, "")
	}

	// Conclusions section
	if len(ctx.Reasoning.Conclusions) > 0 {
		sections = append(sections, "## Conclusions")
		for _, conclusion := range ctx.Reasoning.Conclusions {
			sections = append(sections, fmt.Sprintf("- %s (confidence: %.2f)", conclusion.Description, conclusion.Confidence))
		}
		sections = append(sections, "")
	}

	// Validation section
	if ctx.Diagnostics != nil && (len(ctx.Diagnostics.Errors) > 0 || len(ctx.Diagnostics.Warnings) > 0) {
		sections = append(sections, "## Validation Results")

		if len(ctx.Diagnostics.Errors) > 0 {
			sections = append(sections, "### Errors")
			for _, err := range ctx.Diagnostics.Errors {
				sections = append(sections, fmt.Sprintf("- %s", err.Message))
			}
		}

		if len(ctx.Diagnostics.Warnings) > 0 {
			sections = append(sections, "### Warnings")
			for _, warning := range ctx.Diagnostics.Warnings {
				sections = append(sections, fmt.Sprintf("- %s", warning.Message))
			}
		}
		sections = append(sections, "")
	}

	return strings.Join(sections, "\n")
}

// generateCommandList creates list of executable commands (if applicable).
func (a *SummarizationAgent) generateCommandList(ctx *models.AgentContext) string {
	// Extract commands from conclusions that suggest actions
	commands := []string{}

	for _, conclusion := range ctx.Reasoning.Conclusions {
		// Check if conclusion suggests a command
		if strings.Contains(strings.ToLower(conclusion.Description), "commit") {
			commands = append(commands, "git log")
		}
		if strings.Contains(strings.ToLower(conclusion.Description), "issue") {
			commands = append(commands, "youtrack issues list")
		}
	}

	if len(commands) == 0 {
		return ""
	}

	return strings.Join(commands, "\n")
}

// generateContextDiff creates summary of context changes.
func (a *SummarizationAgent) generateContextDiff(ctx *models.AgentContext) string {
	if ctx.Audit == nil || len(ctx.Audit.AgentRuns) == 0 {
		return ""
	}

	var lines []string
	lines = append(lines, "Context Changes:")

	for _, run := range ctx.Audit.AgentRuns {
		if len(run.KeysWritten) > 0 {
			lines = append(lines, fmt.Sprintf("- %s: %s", run.AgentID, strings.Join(run.KeysWritten, ", ")))
		}
	}

	return strings.Join(lines, "\n")
}

// recordAgentRun records execution in audit trail.
func (a *SummarizationAgent) recordAgentRun(ctx *models.AgentContext, duration time.Duration, status string, err error) {
	run := models.AgentRun{
		Timestamp:  time.Now(),
		AgentID:    a.id,
		Status:     status,
		DurationMS: duration.Milliseconds(),
		KeysWritten: []string{
			"reasoning.summary",
			"reasoning.artifacts",
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
		LLMCalls:   0, // No LLM calls for rule-based summarization
		Status:     status,
	}
}

// GetMetadata returns agent metadata.
func (a *SummarizationAgent) GetMetadata() services.AgentMetadata {
	return services.AgentMetadata{
		ID:          a.id,
		Name:        "Summarization Agent",
		Description: "Generates executive summaries and structured output artifacts from reasoning process",
		Version:     "1.0.0",
		Author:      "ADK LLM Proxy",
		Tags:        []string{"summarization", "reporting", "output-formatting", "artifacts"},
		Dependencies: []string{"inference", "validation"},
	}
}

// GetCapabilities returns agent capabilities.
func (a *SummarizationAgent) GetCapabilities() services.AgentCapabilities {
	return services.AgentCapabilities{
		SupportsParallelExecution: false,
		SupportsRetry:             true,
		RequiresLLM:               false, // Rule-based summarization
		IsDeterministic:           true,
		EstimatedDuration:         50, // ~50ms for summarization
	}
}
