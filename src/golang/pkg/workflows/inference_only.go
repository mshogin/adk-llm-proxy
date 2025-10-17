package workflows

import (
	"context"
	"fmt"
	"time"

	"github.com/mshogin/agents/internal/domain/models"
	"github.com/mshogin/agents/internal/domain/services/agents"
	appservices "github.com/mshogin/agents/internal/application/services"
)

// InferenceOnlyWorkflow executes only the InferenceAgent for isolated testing.
// This workflow is designed for testing and debugging the InferenceAgent in isolation
// without running the full 8-agent pipeline.
type InferenceOnlyWorkflow struct {
	agent           *agents.InferenceAgent
	llmOrchestrator *appservices.LLMOrchestrator
}

// NewInferenceOnlyWorkflow creates a new workflow that executes only InferenceAgent.
func NewInferenceOnlyWorkflow(llmOrchestrator *appservices.LLMOrchestrator) *InferenceOnlyWorkflow {
	return &InferenceOnlyWorkflow{
		agent:           agents.NewInferenceAgent(llmOrchestrator),
		llmOrchestrator: llmOrchestrator,
	}
}

// Name returns the workflow identifier.
func (w *InferenceOnlyWorkflow) Name() string {
	return "inference_only"
}

// Execute runs only the InferenceAgent with the provided input.
// Returns a detailed result showing INPUT, OUTPUT, LLM interactions, and system prompt.
func (w *InferenceOnlyWorkflow) Execute(ctx context.Context, input *models.ReasoningInput) (*models.ReasoningResult, error) {
	startTime := time.Now()

	// 1. Create agent context with mock preconditions (hypotheses + enriched facts)
	agentContext := w.createContextFromInput(input)

	// Store "before" state for comparison
	beforeContext := agentContext

	// 2. Execute only InferenceAgent
	resultContext, err := w.agent.Execute(ctx, agentContext)
	if err != nil {
		return nil, fmt.Errorf("inference agent failed: %w", err)
	}

	// 3. Build detailed result showing INPUT/OUTPUT/SYSTEM_PROMPT
	result := w.buildDetailedResult(beforeContext, resultContext, time.Since(startTime))

	return result, nil
}

// createContextFromInput creates an AgentContext from ReasoningInput with mock preconditions.
// InferenceAgent requires: hypotheses and enriched facts/knowledge.
func (w *InferenceOnlyWorkflow) createContextFromInput(input *models.ReasoningInput) *models.AgentContext {
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

	// Mock preconditions: Create sample hypotheses and enriched facts
	ctx.Reasoning.Hypotheses = []models.Hypothesis{
		{
			Description: "User wants to see recent commit history from GitLab",
			Confidence:  0.95,
			Type:        "query",
			Evidence:    []string{"intent: query_commits", "entity: project=gitlab-project"},
		},
		{
			Description: "Authentication feature was recently added and fixed",
			Confidence:  0.88,
			Type:        "observation",
			Evidence:    []string{"commit: feat: add auth", "commit: fix: resolve auth bug"},
		},
	}

	ctx.Enrichment.Facts = []models.Fact{
		{
			Statement:  "New JWT authentication feature was implemented on 2024-01-15",
			Source:     "gitlab-commit-123",
			Confidence: 0.95,
			Timestamp:  "2024-01-15T10:30:00Z",
		},
		{
			Statement:  "Authentication bug was fixed on 2024-01-16",
			Source:     "gitlab-commit-124",
			Confidence: 0.93,
			Timestamp:  "2024-01-16T14:20:00Z",
		},
		{
			Statement:  "Issue #456 reported intermittent login failures before fix",
			Source:     "youtrack-issue-456",
			Confidence: 0.90,
			Timestamp:  "2024-01-14T09:00:00Z",
		},
	}

	ctx.Enrichment.DerivedKnowledge = []models.DerivedKnowledge{
		{
			Insight:    "The authentication bug was likely caused by JWT token validation issues",
			Confidence: 0.85,
			Sources:    []string{"gitlab-commit-124", "youtrack-issue-456"},
		},
		{
			Insight:    "Recent commits show active development in authentication module",
			Confidence: 0.92,
			Sources:    []string{"gitlab-commit-123", "gitlab-commit-124"},
		},
	}

	ctx.Enrichment.Relationships = []models.Relationship{
		{
			From: "commit-123",
			To:   "issue-456",
			Type: "fixes",
		},
		{
			From: "commit-124",
			To:   "commit-123",
			Type: "follows",
		},
	}

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
func (w *InferenceOnlyWorkflow) buildDetailedResult(before, after *models.AgentContext, duration time.Duration) *models.ReasoningResult {
	result := models.NewReasoningResult("inference_only", "Single agent test completed")

	message := "=== INFERENCE AGENT TEST ===\n\n"

	// Show INPUT (preconditions from previous agents)
	message += "📥 INPUT (Preconditions from ReasoningStructure + ContextSynthesizer):\n"
	if len(before.Reasoning.Hypotheses) > 0 {
		message += fmt.Sprintf("  Hypotheses: %d\n", len(before.Reasoning.Hypotheses))
		for i, h := range before.Reasoning.Hypotheses {
			if i < 3 {
				message += fmt.Sprintf("    %d. %s (%.2f)\n", i+1, h.Description, h.Confidence)
			}
		}
		if len(before.Reasoning.Hypotheses) > 3 {
			message += fmt.Sprintf("    ... and %d more\n", len(before.Reasoning.Hypotheses)-3)
		}
	}

	if len(before.Enrichment.Facts) > 0 {
		message += fmt.Sprintf("\n  Facts: %d\n", len(before.Enrichment.Facts))
		for i, fact := range before.Enrichment.Facts {
			if i < 3 {
				message += fmt.Sprintf("    %d. %s (%.2f)\n", i+1, fact.Statement, fact.Confidence)
			}
		}
		if len(before.Enrichment.Facts) > 3 {
			message += fmt.Sprintf("    ... and %d more\n", len(before.Enrichment.Facts)-3)
		}
	}

	if len(before.Enrichment.DerivedKnowledge) > 0 {
		message += fmt.Sprintf("\n  Derived Knowledge: %d items\n", len(before.Enrichment.DerivedKnowledge))
		for i, knowledge := range before.Enrichment.DerivedKnowledge {
			if i < 2 {
				message += fmt.Sprintf("    %d. %s\n", i+1, knowledge.Insight)
			}
		}
		if len(before.Enrichment.DerivedKnowledge) > 2 {
			message += fmt.Sprintf("    ... and %d more\n", len(before.Enrichment.DerivedKnowledge)-2)
		}
	}

	if before.LLM != nil && before.LLM.Cache != nil {
		if input, ok := before.LLM.Cache["original_user_input"].(string); ok && input != "" {
			message += fmt.Sprintf("\n  User message: \"%s\"\n", input)
		}
	}
	message += "\n"

	// Show OUTPUT (conclusions and updated confidence scores)
	message += "📤 OUTPUT:\n"
	if len(after.Reasoning.Conclusions) > 0 {
		message += fmt.Sprintf("  Conclusions: %d\n", len(after.Reasoning.Conclusions))
		for i, conclusion := range after.Reasoning.Conclusions {
			message += fmt.Sprintf("    %d. %s\n", i+1, conclusion.Statement)
			if conclusion.Confidence > 0 {
				message += fmt.Sprintf("       Confidence: %.2f\n", conclusion.Confidence)
			}
			if len(conclusion.Evidence) > 0 {
				message += fmt.Sprintf("       Evidence: %v\n", conclusion.Evidence)
			}
		}
	} else {
		message += "  • No conclusions drawn\n"
	}

	// Show confidence scores if updated
	if len(after.Reasoning.ConfidenceScores) > 0 {
		message += fmt.Sprintf("\n  Updated Confidence Scores: %d items\n", len(after.Reasoning.ConfidenceScores))
		for key, score := range after.Reasoning.ConfidenceScores {
			message += fmt.Sprintf("    • %s: %.2f\n", key, score)
		}
	}
	message += "\n"

	// Show LLM interaction details if agent trace is available
	if after.LLM != nil && after.LLM.Cache != nil {
		if traces, ok := after.LLM.Cache["agent_traces"].([]interface{}); ok {
			for _, trace := range traces {
				if traceMap, ok := trace.(map[string]interface{}); ok {
					if agentID, ok := traceMap["agent_id"].(string); ok && agentID == "inference" {
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
		if metrics, ok := after.Diagnostics.Performance.AgentMetrics["inference"]; ok {
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
func (w *InferenceOnlyWorkflow) buildSystemPrompt(ctx *models.AgentContext) string {
	prompt := "You are an AI assistant with access to the following reasoning context:\n\n"

	// Add hypotheses
	if len(ctx.Reasoning.Hypotheses) > 0 {
		prompt += "**Hypotheses:**\n"
		for i, h := range ctx.Reasoning.Hypotheses {
			if i < 3 {
				prompt += fmt.Sprintf("%d. %s (confidence: %.2f)\n", i+1, h.Description, h.Confidence)
			}
		}
		if len(ctx.Reasoning.Hypotheses) > 3 {
			prompt += fmt.Sprintf("... and %d more hypotheses\n", len(ctx.Reasoning.Hypotheses)-3)
		}
		prompt += "\n"
	}

	// Add facts
	if len(ctx.Enrichment.Facts) > 0 {
		prompt += "**Facts:**\n"
		for i, fact := range ctx.Enrichment.Facts {
			if i < 5 {
				prompt += fmt.Sprintf("%d. %s (confidence: %.2f)\n", i+1, fact.Statement, fact.Confidence)
			}
		}
		if len(ctx.Enrichment.Facts) > 5 {
			prompt += fmt.Sprintf("... and %d more facts\n", len(ctx.Enrichment.Facts)-5)
		}
		prompt += "\n"
	}

	// Add derived knowledge
	if len(ctx.Enrichment.DerivedKnowledge) > 0 {
		prompt += "**Derived Knowledge:**\n"
		for i, knowledge := range ctx.Enrichment.DerivedKnowledge {
			if i < 3 {
				prompt += fmt.Sprintf("%d. %s (confidence: %.2f)\n", i+1, knowledge.Insight, knowledge.Confidence)
			}
		}
		if len(ctx.Enrichment.DerivedKnowledge) > 3 {
			prompt += fmt.Sprintf("... and %d more insights\n", len(ctx.Enrichment.DerivedKnowledge)-3)
		}
		prompt += "\n"
	}

	// Add conclusions
	if len(ctx.Reasoning.Conclusions) > 0 {
		prompt += "**Conclusions:**\n"
		for i, conclusion := range ctx.Reasoning.Conclusions {
			prompt += fmt.Sprintf("%d. %s (confidence: %.2f)\n", i+1, conclusion.Statement, conclusion.Confidence)
		}
		prompt += "\n"
	}

	prompt += "Please use this reasoning context to provide an informed and accurate response to the user's question."

	return prompt
}
