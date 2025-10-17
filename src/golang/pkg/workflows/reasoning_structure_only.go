package workflows

import (
	"context"
	"fmt"
	"time"

	"github.com/mshogin/agents/internal/domain/models"
	"github.com/mshogin/agents/internal/domain/services/agents"
	appservices "github.com/mshogin/agents/internal/application/services"
)

// ReasoningStructureOnlyWorkflow executes only the ReasoningStructureAgent for isolated testing.
// This workflow is designed for testing and debugging the ReasoningStructureAgent in isolation
// without running the full 8-agent pipeline.
type ReasoningStructureOnlyWorkflow struct {
	agent           *agents.ReasoningStructureAgent
	llmOrchestrator *appservices.LLMOrchestrator
}

// NewReasoningStructureOnlyWorkflow creates a new workflow that executes only ReasoningStructureAgent.
func NewReasoningStructureOnlyWorkflow(llmOrchestrator *appservices.LLMOrchestrator) *ReasoningStructureOnlyWorkflow {
	return &ReasoningStructureOnlyWorkflow{
		agent:           agents.NewReasoningStructureAgent(llmOrchestrator),
		llmOrchestrator: llmOrchestrator,
	}
}

// Name returns the workflow identifier.
func (w *ReasoningStructureOnlyWorkflow) Name() string {
	return "reasoning_structure_only"
}

// Execute runs only the ReasoningStructureAgent with the provided input.
// Returns a detailed result showing INPUT, OUTPUT, LLM interactions, and system prompt.
func (w *ReasoningStructureOnlyWorkflow) Execute(ctx context.Context, input *models.ReasoningInput) (*models.ReasoningResult, error) {
	startTime := time.Now()

	// 1. Create agent context with mock preconditions (intents + entities)
	agentContext := w.createContextFromInput(input)

	// Store "before" state for comparison
	beforeContext := agentContext

	// 2. Execute only ReasoningStructureAgent
	resultContext, err := w.agent.Execute(ctx, agentContext)
	if err != nil {
		return nil, fmt.Errorf("reasoning structure agent failed: %w", err)
	}

	// 3. Build detailed result showing INPUT/OUTPUT/SYSTEM_PROMPT
	result := w.buildDetailedResult(beforeContext, resultContext, time.Since(startTime))

	return result, nil
}

// createContextFromInput creates an AgentContext from ReasoningInput with mock preconditions.
// ReasoningStructureAgent requires: intents and entities (from IntentDetectionAgent).
func (w *ReasoningStructureOnlyWorkflow) createContextFromInput(input *models.ReasoningInput) *models.AgentContext {
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

	// Mock preconditions: Create sample intents and entities
	// In a real scenario, these would come from IntentDetectionAgent
	ctx.Reasoning.Intents = []models.Intent{
		{
			Type:       "query_commits",
			Confidence: 0.99,
			Slots: map[string]interface{}{
				"query_type": "commits",
			},
		},
		{
			Type:       "request_help",
			Confidence: 0.64,
			Slots:      map[string]interface{}{},
		},
	}

	ctx.Reasoning.Entities = map[string][]string{
		"time_range": {"recent"},
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
func (w *ReasoningStructureOnlyWorkflow) buildDetailedResult(before, after *models.AgentContext, duration time.Duration) *models.ReasoningResult {
	result := models.NewReasoningResult("reasoning_structure_only", "Single agent test completed")

	message := "=== REASONING STRUCTURE AGENT TEST ===\n\n"

	// Show INPUT (preconditions from IntentDetectionAgent)
	message += "📥 INPUT (Preconditions from IntentDetectionAgent):\n"
	if len(before.Reasoning.Intents) > 0 {
		message += "  Intents:\n"
		for _, intent := range before.Reasoning.Intents {
			message += fmt.Sprintf("    • %s (confidence: %.2f)\n", intent.Type, intent.Confidence)
		}
	} else {
		message += "  • No intents\n"
	}

	if len(before.Reasoning.Entities) > 0 {
		message += "  Entities:\n"
		for key, values := range before.Reasoning.Entities {
			message += fmt.Sprintf("    • %s: %v\n", key, values)
		}
	}

	if before.LLM != nil && before.LLM.Cache != nil {
		if input, ok := before.LLM.Cache["original_user_input"].(string); ok && input != "" {
			message += fmt.Sprintf("  User message: \"%s\"\n", input)
		}
	}
	message += "\n"

	// Show OUTPUT (hypotheses and reasoning structure)
	message += "📤 OUTPUT:\n"
	if len(after.Reasoning.Hypotheses) > 0 {
		message += fmt.Sprintf("  Generated %d hypotheses:\n", len(after.Reasoning.Hypotheses))
		for i, h := range after.Reasoning.Hypotheses {
			if i < 5 { // Show first 5
				message += fmt.Sprintf("    %d. %s\n", i+1, h.Description)
				if h.Confidence > 0 {
					message += fmt.Sprintf("       Confidence: %.2f\n", h.Confidence)
				}
				if len(h.Dependencies) > 0 {
					message += fmt.Sprintf("       Dependencies: %v\n", h.Dependencies)
				}
			}
		}
		if len(after.Reasoning.Hypotheses) > 5 {
			message += fmt.Sprintf("    ... and %d more\n", len(after.Reasoning.Hypotheses)-5)
		}
	} else {
		message += "  • No hypotheses generated\n"
	}
	message += "\n"

	// Show LLM interaction details if agent trace is available
	if after.LLM != nil && after.LLM.Cache != nil {
		if traces, ok := after.LLM.Cache["agent_traces"].([]interface{}); ok {
			for _, trace := range traces {
				if traceMap, ok := trace.(map[string]interface{}); ok {
					if agentID, ok := traceMap["agent_id"].(string); ok && agentID == "reasoning_structure" {
						if triggered, ok := traceMap["llm_fallback_triggered"].(bool); ok && triggered {
							message += "💬 LLM INTERACTION:\n"
							if reason, ok := traceMap["llm_trigger_reason"].(string); ok {
								message += fmt.Sprintf("  Reason: %s\n", reason)
							}
							if llmCalls, ok := traceMap["llm_calls_made"].(int); ok {
								message += fmt.Sprintf("  Calls: %d\n", llmCalls)
							}
							if llmResponse, ok := traceMap["llm_response"].(string); ok && llmResponse != "" {
								// Show truncated response
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
		if metrics, ok := after.Diagnostics.Performance.AgentMetrics["reasoning_structure"]; ok {
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
func (w *ReasoningStructureOnlyWorkflow) buildSystemPrompt(ctx *models.AgentContext) string {
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

	// Add generated hypotheses
	if len(ctx.Reasoning.Hypotheses) > 0 {
		prompt += "**Reasoning Hypotheses:**\n"
		for i, h := range ctx.Reasoning.Hypotheses {
			if i < 5 {
				prompt += fmt.Sprintf("%d. %s (confidence: %.2f)\n", i+1, h.Description, h.Confidence)
			}
		}
		if len(ctx.Reasoning.Hypotheses) > 5 {
			prompt += fmt.Sprintf("... and %d more hypotheses\n", len(ctx.Reasoning.Hypotheses)-5)
		}
		prompt += "\n"
	}

	prompt += "Please use this context to provide an informed and accurate response to the user's question."

	return prompt
}
