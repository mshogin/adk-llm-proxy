package workflows

import (
	"context"
	"fmt"
	"time"

	"github.com/mshogin/agents/internal/domain/models"
	"github.com/mshogin/agents/internal/domain/services/agents"
	appservices "github.com/mshogin/agents/internal/application/services"
)

// RetrievalPlannerOnlyWorkflow executes only the RetrievalPlannerAgent for isolated testing.
// This workflow is designed for testing and debugging the RetrievalPlannerAgent in isolation
// without running the full 8-agent pipeline.
type RetrievalPlannerOnlyWorkflow struct {
	agent           *agents.RetrievalPlannerAgent
	llmOrchestrator *appservices.LLMOrchestrator
}

// NewRetrievalPlannerOnlyWorkflow creates a new workflow that executes only RetrievalPlannerAgent.
func NewRetrievalPlannerOnlyWorkflow(llmOrchestrator *appservices.LLMOrchestrator) *RetrievalPlannerOnlyWorkflow {
	return &RetrievalPlannerOnlyWorkflow{
		agent:           agents.NewRetrievalPlannerAgent(llmOrchestrator),
		llmOrchestrator: llmOrchestrator,
	}
}

// Name returns the workflow identifier.
func (w *RetrievalPlannerOnlyWorkflow) Name() string {
	return "retrieval_planner_only"
}

// Execute runs only the RetrievalPlannerAgent with the provided input.
// Returns a detailed result showing INPUT, OUTPUT, LLM interactions, and system prompt.
func (w *RetrievalPlannerOnlyWorkflow) Execute(ctx context.Context, input *models.ReasoningInput) (*models.ReasoningResult, error) {
	startTime := time.Now()

	// 1. Create agent context with mock preconditions (intents + entities + hypotheses)
	agentContext := w.createContextFromInput(input)

	// Store "before" state for comparison
	beforeContext := agentContext

	// 2. Execute only RetrievalPlannerAgent
	resultContext, err := w.agent.Execute(ctx, agentContext)
	if err != nil {
		return nil, fmt.Errorf("retrieval planner agent failed: %w", err)
	}

	// 3. Build detailed result showing INPUT/OUTPUT/SYSTEM_PROMPT
	result := w.buildDetailedResult(beforeContext, resultContext, time.Since(startTime))

	return result, nil
}

// createContextFromInput creates an AgentContext from ReasoningInput with mock preconditions.
// RetrievalPlannerAgent requires: intents, entities, and hypotheses.
func (w *RetrievalPlannerOnlyWorkflow) createContextFromInput(input *models.ReasoningInput) *models.AgentContext {
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

	// Mock preconditions: Create sample intents, entities, and hypotheses
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

	ctx.Reasoning.Entities = map[string][]string{
		"time_range": {"recent"},
		"project":    {"gitlab-project"},
	}

	ctx.Reasoning.Hypotheses = []models.Hypothesis{
		{
			Description: "User wants to see recent commit history from GitLab",
			Confidence:  0.95,
			Type:        "query",
			Evidence:    []string{"intent: query_commits", "entity: project=gitlab-project"},
		},
		{
			Description: "User may need help understanding commits",
			Confidence:  0.60,
			Type:        "support",
			Evidence:    []string{"intent: request_help"},
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
func (w *RetrievalPlannerOnlyWorkflow) buildDetailedResult(before, after *models.AgentContext, duration time.Duration) *models.ReasoningResult {
	result := models.NewReasoningResult("retrieval_planner_only", "Single agent test completed")

	message := "=== RETRIEVAL PLANNER AGENT TEST ===\n\n"

	// Show INPUT (preconditions from previous agents)
	message += "📥 INPUT (Preconditions from IntentDetection + ReasoningStructure):\n"
	if len(before.Reasoning.Intents) > 0 {
		message += "  Intents:\n"
		for _, intent := range before.Reasoning.Intents {
			message += fmt.Sprintf("    • %s (confidence: %.2f)\n", intent.Type, intent.Confidence)
		}
	}

	if len(before.Reasoning.Entities) > 0 {
		message += "  Entities:\n"
		for key, values := range before.Reasoning.Entities {
			message += fmt.Sprintf("    • %s: %v\n", key, values)
		}
	}

	if len(before.Reasoning.Hypotheses) > 0 {
		message += fmt.Sprintf("  Hypotheses: %d generated\n", len(before.Reasoning.Hypotheses))
		for i, h := range before.Reasoning.Hypotheses {
			if i < 3 {
				message += fmt.Sprintf("    %d. %s (%.2f)\n", i+1, h.Description, h.Confidence)
			}
		}
		if len(before.Reasoning.Hypotheses) > 3 {
			message += fmt.Sprintf("    ... and %d more\n", len(before.Reasoning.Hypotheses)-3)
		}
	}

	if before.LLM != nil && before.LLM.Cache != nil {
		if input, ok := before.LLM.Cache["original_user_input"].(string); ok && input != "" {
			message += fmt.Sprintf("  User message: \"%s\"\n", input)
		}
	}
	message += "\n"

	// Show OUTPUT (retrieval plans and queries)
	message += "📤 OUTPUT:\n"
	if len(after.Retrieval.Plans) > 0 {
		message += fmt.Sprintf("  Generated %d retrieval plans:\n", len(after.Retrieval.Plans))
		for i, plan := range after.Retrieval.Plans {
			message += fmt.Sprintf("    %d. Source: %s, Priority: %d\n", i+1, plan.Source, plan.Priority)
			if plan.Description != "" {
				message += fmt.Sprintf("       Description: %s\n", plan.Description)
			}
			if plan.MaxResults > 0 {
				message += fmt.Sprintf("       Max Results: %d\n", plan.MaxResults)
			}
		}
	} else {
		message += "  • No retrieval plans generated\n"
	}

	if len(after.Retrieval.Queries) > 0 {
		message += fmt.Sprintf("\n  Generated %d queries:\n", len(after.Retrieval.Queries))
		for i, query := range after.Retrieval.Queries {
			if i < 5 {
				message += fmt.Sprintf("    %d. %s\n", i+1, query.QueryString)
				if query.Source != "" {
					message += fmt.Sprintf("       Source: %s\n", query.Source)
				}
			}
		}
		if len(after.Retrieval.Queries) > 5 {
			message += fmt.Sprintf("    ... and %d more\n", len(after.Retrieval.Queries)-5)
		}
	}
	message += "\n"

	// Show LLM interaction details if agent trace is available
	if after.LLM != nil && after.LLM.Cache != nil {
		if traces, ok := after.LLM.Cache["agent_traces"].([]interface{}); ok {
			for _, trace := range traces {
				if traceMap, ok := trace.(map[string]interface{}); ok {
					if agentID, ok := traceMap["agent_id"].(string); ok && agentID == "retrieval_planner" {
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
		if metrics, ok := after.Diagnostics.Performance.AgentMetrics["retrieval_planner"]; ok {
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
func (w *RetrievalPlannerOnlyWorkflow) buildSystemPrompt(ctx *models.AgentContext) string {
	prompt := "You are an AI assistant with access to the following reasoning context:\n\n"

	// Add detected intents
	if len(ctx.Reasoning.Intents) > 0 {
		prompt += "**Detected Intents:**\n"
		for _, intent := range ctx.Reasoning.Intents {
			prompt += fmt.Sprintf("- %s (confidence: %.2f)\n", intent.Type, intent.Confidence)
		}
		prompt += "\n"
	}

	// Add detected entities
	if len(ctx.Reasoning.Entities) > 0 {
		prompt += "**Extracted Entities:**\n"
		for key, values := range ctx.Reasoning.Entities {
			prompt += fmt.Sprintf("- %s: %v\n", key, values)
		}
		prompt += "\n"
	}

	// Add hypotheses
	if len(ctx.Reasoning.Hypotheses) > 0 {
		prompt += "**Reasoning Hypotheses:**\n"
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

	// Add retrieval plans
	if len(ctx.Retrieval.Plans) > 0 {
		prompt += "**Retrieval Plans:**\n"
		for i, plan := range ctx.Retrieval.Plans {
			prompt += fmt.Sprintf("%d. Source: %s (Priority: %d)\n", i+1, plan.Source, plan.Priority)
			if plan.Description != "" {
				prompt += fmt.Sprintf("   %s\n", plan.Description)
			}
		}
		prompt += "\n"
	}

	// Add queries
	if len(ctx.Retrieval.Queries) > 0 {
		prompt += "**Retrieval Queries:**\n"
		for i, query := range ctx.Retrieval.Queries {
			if i < 5 {
				prompt += fmt.Sprintf("%d. %s", i+1, query.QueryString)
				if query.Source != "" {
					prompt += fmt.Sprintf(" (Source: %s)", query.Source)
				}
				prompt += "\n"
			}
		}
		if len(ctx.Retrieval.Queries) > 5 {
			prompt += fmt.Sprintf("... and %d more queries\n", len(ctx.Retrieval.Queries)-5)
		}
		prompt += "\n"
	}

	prompt += "Please use this context to provide an informed and accurate response to the user's question."

	return prompt
}
