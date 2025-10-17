package workflows

import (
	"context"
	"fmt"
	"time"

	"github.com/mshogin/agents/internal/domain/models"
	"github.com/mshogin/agents/internal/domain/services/agents"
	appservices "github.com/mshogin/agents/internal/application/services"
)

// IntentDetectionOnlyWorkflow executes only the IntentDetectionAgent for isolated testing.
// This workflow is designed for testing and debugging the IntentDetectionAgent in isolation
// without running the full 8-agent pipeline.
type IntentDetectionOnlyWorkflow struct {
	agent           *agents.IntentDetectionAgent
	llmOrchestrator *appservices.LLMOrchestrator
}

// NewIntentDetectionOnlyWorkflow creates a new workflow that executes only IntentDetectionAgent.
func NewIntentDetectionOnlyWorkflow(llmOrchestrator *appservices.LLMOrchestrator) *IntentDetectionOnlyWorkflow {
	return &IntentDetectionOnlyWorkflow{
		agent:           agents.NewIntentDetectionAgent(llmOrchestrator),
		llmOrchestrator: llmOrchestrator,
	}
}

// Name returns the workflow identifier.
func (w *IntentDetectionOnlyWorkflow) Name() string {
	return "intent_detection_only"
}

// Execute runs only the IntentDetectionAgent with the provided input.
// Returns a detailed result showing INPUT, OUTPUT, LLM interactions, and system prompt.
func (w *IntentDetectionOnlyWorkflow) Execute(ctx context.Context, input *models.ReasoningInput) (*models.ReasoningResult, error) {
	startTime := time.Now()

	// 1. Create agent context from user input
	agentContext := w.createContextFromInput(input)

	// Store "before" state for comparison
	beforeContext := agentContext

	// 2. Execute only IntentDetectionAgent
	resultContext, err := w.agent.Execute(ctx, agentContext)
	if err != nil {
		return nil, fmt.Errorf("intent detection agent failed: %w", err)
	}

	// 3. Build detailed result showing INPUT/OUTPUT/SYSTEM_PROMPT
	result := w.buildDetailedResult(beforeContext, resultContext, time.Since(startTime))

	return result, nil
}

// createContextFromInput creates an AgentContext from ReasoningInput.
// This is the minimal context needed for IntentDetectionAgent (which has no preconditions).
func (w *IntentDetectionOnlyWorkflow) createContextFromInput(input *models.ReasoningInput) *models.AgentContext {
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

		// Store input text in retrieval context for intent detection agent
		ctx.Retrieval.Queries = []models.Query{
			{
				QueryString: userInput,
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
	}

	return ctx
}

// buildDetailedResult creates a comprehensive result showing all agent details.
func (w *IntentDetectionOnlyWorkflow) buildDetailedResult(before, after *models.AgentContext, duration time.Duration) *models.ReasoningResult {
	result := models.NewReasoningResult("intent_detection_only", "Single agent test completed")

	message := "=== INTENT DETECTION AGENT TEST ===\n\n"

	// Show INPUT
	message += "📥 INPUT:\n"
	if before.LLM != nil && before.LLM.Cache != nil {
		if input, ok := before.LLM.Cache["original_user_input"].(string); ok {
			message += fmt.Sprintf("  User message: \"%s\"\n", input)
		}
	}
	message += "\n"

	// Show OUTPUT
	message += "📤 OUTPUT:\n"
	if len(after.Reasoning.Intents) > 0 {
		for _, intent := range after.Reasoning.Intents {
			message += fmt.Sprintf("  • %s (confidence: %.2f)\n", intent.Type, intent.Confidence)
		}
	} else {
		message += "  • No intents detected\n"
	}

	// Show entities if any
	if len(after.Reasoning.Entities) > 0 {
		message += "\n  Entities:\n"
		for key, values := range after.Reasoning.Entities {
			message += fmt.Sprintf("    • %s: %v\n", key, values)
		}
	}

	// Show clarification questions if any
	if len(after.Reasoning.ClarificationQuestions) > 0 {
		message += "\n  Clarification needed:\n"
		for _, q := range after.Reasoning.ClarificationQuestions {
			message += fmt.Sprintf("    ? %s\n", q.Question)
			if len(q.Options) > 0 {
				for _, opt := range q.Options {
					message += fmt.Sprintf("      - %s\n", opt)
				}
			}
		}
	}
	message += "\n"

	// Show LLM interaction details if agent trace is available
	if after.LLM != nil && after.LLM.Cache != nil {
		if traces, ok := after.LLM.Cache["agent_traces"].([]interface{}); ok {
			for _, trace := range traces {
				if traceMap, ok := trace.(map[string]interface{}); ok {
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

	// Show system prompt that would go to LLM
	message += "=== 📤 SYSTEM PROMPT FOR LLM ===\n\n"
	systemPrompt := w.buildSystemPrompt(after)
	message += systemPrompt
	message += "\n\n"

	// Show metrics
	message += "⏱️  METRICS:\n"
	message += fmt.Sprintf("  Duration: %dms\n", duration.Milliseconds())

	if after.Diagnostics != nil && after.Diagnostics.Performance != nil {
		if metrics, ok := after.Diagnostics.Performance.AgentMetrics["intent_detection"]; ok {
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
func (w *IntentDetectionOnlyWorkflow) buildSystemPrompt(ctx *models.AgentContext) string {
	prompt := "You are an AI assistant with access to the following reasoning context:\n\n"

	// Add detected intents
	if len(ctx.Reasoning.Intents) > 0 {
		prompt += "**Detected Intent:**\n"
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

	// Add clarification questions if any
	if len(ctx.Reasoning.ClarificationQuestions) > 0 {
		prompt += "**Clarification Needed:**\n"
		for _, q := range ctx.Reasoning.ClarificationQuestions {
			prompt += fmt.Sprintf("- %s\n", q.Question)
			if q.Reason != "" {
				prompt += fmt.Sprintf("  Reason: %s\n", q.Reason)
			}
		}
		prompt += "\n"
	}

	prompt += "Please use this context to provide an informed and accurate response to the user's question."

	return prompt
}
