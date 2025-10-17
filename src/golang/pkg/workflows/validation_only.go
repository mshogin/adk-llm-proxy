package workflows

import (
	"context"
	"fmt"
	"time"

	"github.com/mshogin/agents/internal/domain/models"
	"github.com/mshogin/agents/internal/domain/services/agents"
	appservices "github.com/mshogin/agents/internal/application/services"
)

// ValidationOnlyWorkflow executes only the ValidationAgent for isolated testing.
// This workflow is designed for testing and debugging the ValidationAgent in isolation
// without running the full 8-agent pipeline.
type ValidationOnlyWorkflow struct {
	agent           *agents.ValidationAgent
	llmOrchestrator *appservices.LLMOrchestrator
}

// NewValidationOnlyWorkflow creates a new workflow that executes only ValidationAgent.
func NewValidationOnlyWorkflow(llmOrchestrator *appservices.LLMOrchestrator) *ValidationOnlyWorkflow {
	return &ValidationOnlyWorkflow{
		agent:           agents.NewValidationAgent(llmOrchestrator),
		llmOrchestrator: llmOrchestrator,
	}
}

// Name returns the workflow identifier.
func (w *ValidationOnlyWorkflow) Name() string {
	return "validation_only"
}

// Execute runs only the ValidationAgent with the provided input.
// Returns a detailed result showing INPUT, OUTPUT, LLM interactions, and system prompt.
func (w *ValidationOnlyWorkflow) Execute(ctx context.Context, input *models.ReasoningInput) (*models.ReasoningResult, error) {
	startTime := time.Now()

	// 1. Create agent context with mock preconditions (complete reasoning context)
	agentContext := w.createContextFromInput(input)

	// Store "before" state for comparison
	beforeContext := agentContext

	// 2. Execute only ValidationAgent
	resultContext, err := w.agent.Execute(ctx, agentContext)
	if err != nil {
		return nil, fmt.Errorf("validation agent failed: %w", err)
	}

	// 3. Build detailed result showing INPUT/OUTPUT/SYSTEM_PROMPT
	result := w.buildDetailedResult(beforeContext, resultContext, time.Since(startTime))

	return result, nil
}

// createContextFromInput creates an AgentContext from ReasoningInput with mock preconditions.
// ValidationAgent requires: complete reasoning context to validate.
func (w *ValidationOnlyWorkflow) createContextFromInput(input *models.ReasoningInput) *models.AgentContext {
	// Generate session and trace IDs
	now := time.Now()
	sessionID := fmt.Sprintf("test-session-%d", now.Unix())
	traceID := fmt.Sprintf("test-trace-%d", now.UnixNano())

	ctx := models.NewAgentContext(sessionID, traceID)

	// Extract user input from messages
	userInput := ""
	if len(input.Messages) > 0 {
		lastMessage := input.Messages[len(input.Messages)-1]
		userInput = lastMessage.Content
	}

	// Mock preconditions: Create complete reasoning context for validation
	ctx.Reasoning.Intents = []models.Intent{
		{
			Type:       "query_commits",
			Confidence: 0.99,
			Slots: map[string]interface{}{
				"query_type": "commits",
				"project":    "gitlab-project",
			},
		},
	}

	ctx.Reasoning.Hypotheses = []models.Hypothesis{
		{
			Description: "User wants to see recent commit history from GitLab",
			Confidence:  0.95,
			Type:        "query",
			Evidence:    []string{"intent: query_commits"},
		},
	}

	ctx.Retrieval.Plans = []models.RetrievalPlan{
		{
			Source:      "gitlab",
			Description: "Fetch recent commits",
			Priority:    1,
		},
	}

	ctx.Retrieval.Artifacts = []models.Artifact{
		{
			ID:      "commit-123",
			Type:    "commit",
			Source:  "gitlab",
			Title:   "feat: add new feature",
			Content: "Implementation details...",
		},
	}

	ctx.Enrichment.Facts = []models.Fact{
		{
			Statement:  "New authentication feature was implemented",
			Source:     "gitlab-commit-123",
			Confidence: 0.95,
		},
	}

	ctx.Reasoning.Conclusions = []models.Conclusion{
		{
			Statement:  "Recent commits show authentication module updates",
			Confidence: 0.94,
			Type:       "finding",
			Evidence:   []string{"fact-1"},
		},
	}

	ctx.Reasoning.Summary = "The authentication module has been recently updated with new features."

	// Store original user input in LLM cache
	if ctx.LLM == nil {
		ctx.LLM = &models.LLMContext{
			Cache: make(map[string]interface{}),
		}
	}
	if ctx.LLM.Cache == nil {
		ctx.LLM.Cache = make(map[string]interface{})
	}
	ctx.LLM.Cache["original_user_input"] = userInput
	ctx.LLM.Cache["all_messages"] = input.Messages

	return ctx
}

// buildDetailedResult creates a comprehensive result showing all agent details.
func (w *ValidationOnlyWorkflow) buildDetailedResult(before, after *models.AgentContext, duration time.Duration) *models.ReasoningResult {
	result := models.NewReasoningResult("validation_only", "Single agent test completed")

	message := "=== VALIDATION AGENT TEST ===\n\n"

	// Show INPUT (complete reasoning context to validate)
	message += "📥 INPUT (Complete Reasoning Context to Validate):\n"
	message += fmt.Sprintf("  Intents: %d\n", len(before.Reasoning.Intents))
	message += fmt.Sprintf("  Hypotheses: %d\n", len(before.Reasoning.Hypotheses))
	message += fmt.Sprintf("  Retrieval Plans: %d\n", len(before.Retrieval.Plans))
	message += fmt.Sprintf("  Artifacts: %d\n", len(before.Retrieval.Artifacts))
	message += fmt.Sprintf("  Facts: %d\n", len(before.Enrichment.Facts))
	message += fmt.Sprintf("  Conclusions: %d\n", len(before.Reasoning.Conclusions))
	if before.Reasoning.Summary != "" {
		message += "  Summary: Present\n"
	}

	if before.LLM != nil && before.LLM.Cache != nil {
		if input, ok := before.LLM.Cache["original_user_input"].(string); ok && input != "" {
			message += fmt.Sprintf("\n  User message: \"%s\"\n", input)
		}
	}
	message += "\n"

	// Show OUTPUT (validation results)
	message += "📤 OUTPUT:\n"

	// Validation reports
	if after.Diagnostics != nil && len(after.Diagnostics.ValidationReports) > 0 {
		message += fmt.Sprintf("  Validation Reports: %d\n", len(after.Diagnostics.ValidationReports))
		for _, report := range after.Diagnostics.ValidationReports {
			statusIcon := "✓"
			if report.Status == "failed" {
				statusIcon = "✗"
			} else if report.Status == "warning" {
				statusIcon = "⚠"
			}
			message += fmt.Sprintf("    %s %s: %s\n", statusIcon, report.CheckName, report.Status)
			if report.Message != "" {
				message += fmt.Sprintf("       %s\n", report.Message)
			}
		}
	} else {
		message += "  • No validation reports generated\n"
	}

	// Errors
	if after.Diagnostics != nil && len(after.Diagnostics.Errors) > 0 {
		message += fmt.Sprintf("\n  Errors: %d\n", len(after.Diagnostics.Errors))
		for i, err := range after.Diagnostics.Errors {
			if i < 5 {
				message += fmt.Sprintf("    • %s: %s\n", err.Code, err.Message)
			}
		}
		if len(after.Diagnostics.Errors) > 5 {
			message += fmt.Sprintf("    ... and %d more\n", len(after.Diagnostics.Errors)-5)
		}
	}

	// Warnings
	if after.Diagnostics != nil && len(after.Diagnostics.Warnings) > 0 {
		message += fmt.Sprintf("\n  Warnings: %d\n", len(after.Diagnostics.Warnings))
		for i, warn := range after.Diagnostics.Warnings {
			if i < 5 {
				message += fmt.Sprintf("    • %s: %s\n", warn.Code, warn.Message)
			}
		}
		if len(after.Diagnostics.Warnings) > 5 {
			message += fmt.Sprintf("    ... and %d more\n", len(after.Diagnostics.Warnings)-5)
		}
	}

	// Overall validation status
	if after.Diagnostics != nil {
		message += "\n  Overall Status: "
		if len(after.Diagnostics.Errors) > 0 {
			message += "❌ FAILED (validation errors found)\n"
		} else if len(after.Diagnostics.Warnings) > 0 {
			message += "⚠️  WARNING (potential issues found)\n"
		} else {
			message += "✅ PASSED (no issues found)\n"
		}
	}
	message += "\n"

	// Show LLM interaction details if agent trace is available
	if after.LLM != nil && after.LLM.Cache != nil {
		if traces, ok := after.LLM.Cache["agent_traces"].([]interface{}); ok {
			for _, trace := range traces {
				if traceMap, ok := trace.(map[string]interface{}); ok {
					if agentID, ok := traceMap["agent_id"].(string); ok && agentID == "validation" {
						if triggered, ok := traceMap["llm_fallback_triggered"].(bool); ok && triggered {
							message += "💬 LLM INTERACTION:\n"
							if reason, ok := traceMap["llm_trigger_reason"].(string); ok {
								message += fmt.Sprintf("  Reason: %s\n", reason)
							}
							if llmCalls, ok := traceMap["llm_calls_made"].(int); ok {
								message += fmt.Sprintf("  Calls: %d\n", llmCalls)
							}
							if llmResponse, ok := traceMap["llm_response"].(string); ok && llmResponse != "" {
								if len(llmResponse) > 200 {
									message += fmt.Sprintf("  Response: %s...\n", llmResponse[:200])
								} else {
									message += fmt.Sprintf("  Response: %s\n", llmResponse)
								}
							}
							message += "\n"
						}
					}
				}
			}
		}
	}

	// Show system prompt that would go to LLM
	message += "=== 📤 SYSTEM PROMPT FOR LLM ===\n\n"
	systemPrompt := w.buildSystemPrompt(after)
	message += systemPrompt
	message += "\n\n"

	// Show metrics
	message += "⏱️  METRICS:\n"
	message += fmt.Sprintf("  Duration: %dms\n", duration.Milliseconds())

	if after.Diagnostics != nil && after.Diagnostics.Performance != nil {
		if metrics, ok := after.Diagnostics.Performance.AgentMetrics["validation"]; ok {
			message += fmt.Sprintf("  LLM calls: %d\n", metrics.LLMCalls)
			if metrics.Cost > 0 {
				message += fmt.Sprintf("  Cost: $%.6f\n", metrics.Cost)
			}
		}
	}

	// Show agent execution summary
	if len(after.Audit.AgentRuns) > 0 {
		message += "\n📊 AGENT EXECUTION:\n"
		for _, run := range after.Audit.AgentRuns {
			status := "✓"
			if run.Status != "success" {
				status = "✗"
			}
			message += fmt.Sprintf("  %s %s: %dms\n", status, run.AgentID, run.DurationMS)
		}
	}

	result.Message = message

	// Add enriched messages with system prompt
	result.EnrichedMessages = []models.Message{
		{
			Role:    "system",
			Content: systemPrompt,
		},
	}

	return result
}

// buildSystemPrompt creates the system prompt from reasoning context.
func (w *ValidationOnlyWorkflow) buildSystemPrompt(ctx *models.AgentContext) string {
	prompt := "You are an AI assistant. The reasoning chain has been validated with the following results:\n\n"

	// Show validation status
	if ctx.Diagnostics != nil {
		if len(ctx.Diagnostics.ValidationReports) > 0 {
			prompt += "**Validation Results:**\n"
			for _, report := range ctx.Diagnostics.ValidationReports {
				prompt += fmt.Sprintf("- %s: %s\n", report.CheckName, report.Status)
				if report.Message != "" {
					prompt += fmt.Sprintf("  %s\n", report.Message)
				}
			}
			prompt += "\n"
		}

		// Show errors if any
		if len(ctx.Diagnostics.Errors) > 0 {
			prompt += "**Errors Found:**\n"
			for i, err := range ctx.Diagnostics.Errors {
				if i < 3 {
					prompt += fmt.Sprintf("- %s: %s\n", err.Code, err.Message)
				}
			}
			if len(ctx.Diagnostics.Errors) > 3 {
				prompt += fmt.Sprintf("... and %d more errors\n", len(ctx.Diagnostics.Errors)-3)
			}
			prompt += "\n"
		}

		// Show warnings if any
		if len(ctx.Diagnostics.Warnings) > 0 {
			prompt += "**Warnings:**\n"
			for i, warn := range ctx.Diagnostics.Warnings {
				if i < 3 {
					prompt += fmt.Sprintf("- %s: %s\n", warn.Code, warn.Message)
				}
			}
			if len(ctx.Diagnostics.Warnings) > 3 {
				prompt += fmt.Sprintf("... and %d more warnings\n", len(ctx.Diagnostics.Warnings)-3)
			}
			prompt += "\n"
		}

		// Overall status
		if len(ctx.Diagnostics.Errors) > 0 {
			prompt += "**Overall Status:** FAILED - Validation errors found\n"
		} else if len(ctx.Diagnostics.Warnings) > 0 {
			prompt += "**Overall Status:** WARNING - Potential issues detected\n"
		} else {
			prompt += "**Overall Status:** PASSED - No issues found\n"
		}
	}

	prompt += "\nPlease provide a response based on the validated reasoning chain."

	return prompt
}
