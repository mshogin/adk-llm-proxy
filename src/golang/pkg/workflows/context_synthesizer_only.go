package workflows

import (
	"context"
	"fmt"
	"time"

	"github.com/mshogin/agents/internal/domain/models"
	"github.com/mshogin/agents/internal/domain/services/agents"
	appservices "github.com/mshogin/agents/internal/application/services"
)

// ContextSynthesizerOnlyWorkflow executes only the ContextSynthesizerAgent for isolated testing.
// This workflow is designed for testing and debugging the ContextSynthesizerAgent in isolation
// without running the full 8-agent pipeline.
type ContextSynthesizerOnlyWorkflow struct {
	agent           *agents.ContextSynthesizerAgent
	llmOrchestrator *appservices.LLMOrchestrator
}

// NewContextSynthesizerOnlyWorkflow creates a new workflow that executes only ContextSynthesizerAgent.
func NewContextSynthesizerOnlyWorkflow(llmOrchestrator *appservices.LLMOrchestrator) *ContextSynthesizerOnlyWorkflow {
	return &ContextSynthesizerOnlyWorkflow{
		agent:           agents.NewContextSynthesizerAgent(llmOrchestrator),
		llmOrchestrator: llmOrchestrator,
	}
}

// Name returns the workflow identifier.
func (w *ContextSynthesizerOnlyWorkflow) Name() string {
	return "context_synthesizer_only"
}

// Execute runs only the ContextSynthesizerAgent with the provided input.
// Returns a detailed result showing INPUT, OUTPUT, LLM interactions, and system prompt.
func (w *ContextSynthesizerOnlyWorkflow) Execute(ctx context.Context, input *models.ReasoningInput) (*models.ReasoningResult, error) {
	startTime := time.Now()

	// 1. Create agent context with mock preconditions (retrieved artifacts)
	agentContext := w.createContextFromInput(input)

	// Store "before" state for comparison
	beforeContext := agentContext

	// 2. Execute only ContextSynthesizerAgent
	resultContext, err := w.agent.Execute(ctx, agentContext)
	if err != nil {
		return nil, fmt.Errorf("context synthesizer agent failed: %w", err)
	}

	// 3. Build detailed result showing INPUT/OUTPUT/SYSTEM_PROMPT
	result := w.buildDetailedResult(beforeContext, resultContext, time.Since(startTime))

	return result, nil
}

// createContextFromInput creates an AgentContext from ReasoningInput with mock preconditions.
// ContextSynthesizerAgent requires: artifacts from retrieval executor.
func (w *ContextSynthesizerOnlyWorkflow) createContextFromInput(input *models.ReasoningInput) *models.AgentContext {
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

	// Mock preconditions: Create sample artifacts from different sources
	ctx.Retrieval.Artifacts = []models.Artifact{
		{
			ID:      "commit-123",
			Type:    "commit",
			Source:  "gitlab",
			Title:   "feat: add new feature",
			Content: "Added new authentication feature with JWT support. Updated documentation.",
			Metadata: map[string]interface{}{
				"author":    "john.doe",
				"timestamp": "2024-01-15T10:30:00Z",
				"project":   "backend-api",
			},
			Confidence: 0.95,
		},
		{
			ID:      "commit-124",
			Type:    "commit",
			Source:  "gitlab",
			Title:   "fix: resolve authentication bug",
			Content: "Fixed JWT token validation issue causing intermittent login failures.",
			Metadata: map[string]interface{}{
				"author":    "jane.smith",
				"timestamp": "2024-01-16T14:20:00Z",
				"project":   "backend-api",
			},
			Confidence: 0.93,
		},
		{
			ID:      "issue-456",
			Type:    "issue",
			Source:  "youtrack",
			Title:   "Authentication fails intermittently",
			Content: "Users report intermittent login failures. JWT token validation may be the issue.",
			Metadata: map[string]interface{}{
				"status":     "resolved",
				"priority":   "high",
				"created_at": "2024-01-14T09:00:00Z",
				"resolved":   "2024-01-16T15:00:00Z",
			},
			Confidence: 0.90,
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
func (w *ContextSynthesizerOnlyWorkflow) buildDetailedResult(before, after *models.AgentContext, duration time.Duration) *models.ReasoningResult {
	result := models.NewReasoningResult("context_synthesizer_only", "Single agent test completed")

	message := "=== CONTEXT SYNTHESIZER AGENT TEST ===\n\n"

	// Show INPUT (preconditions from RetrievalExecutorAgent)
	message += "📥 INPUT (Preconditions from RetrievalExecutor):\n"
	if len(before.Retrieval.Artifacts) > 0 {
		message += fmt.Sprintf("  Retrieved Artifacts: %d\n", len(before.Retrieval.Artifacts))

		// Group by source
		artifactsBySource := make(map[string]int)
		for _, artifact := range before.Retrieval.Artifacts {
			artifactsBySource[artifact.Source]++
		}

		for source, count := range artifactsBySource {
			message += fmt.Sprintf("    • %s: %d artifacts\n", source, count)
		}

		// Show sample artifacts
		message += "\n  Sample artifacts:\n"
		for i, artifact := range before.Retrieval.Artifacts {
			if i < 3 {
				message += fmt.Sprintf("    %d. [%s] %s\n", i+1, artifact.Type, artifact.Title)
				if artifact.Content != "" {
					contentPreview := artifact.Content
					if len(contentPreview) > 60 {
						contentPreview = contentPreview[:60] + "..."
					}
					message += fmt.Sprintf("       %s\n", contentPreview)
				}
			}
		}
		if len(before.Retrieval.Artifacts) > 3 {
			message += fmt.Sprintf("    ... and %d more\n", len(before.Retrieval.Artifacts)-3)
		}
	} else {
		message += "  • No artifacts\n"
	}

	if before.LLM != nil && before.LLM.Cache != nil {
		if input, ok := before.LLM.Cache["original_user_input"].(string); ok && input != "" {
			message += fmt.Sprintf("\n  User message: \"%s\"\n", input)
		}
	}
	message += "\n"

	// Show OUTPUT (synthesized facts and knowledge)
	message += "📤 OUTPUT:\n"
	if len(after.Enrichment.Facts) > 0 {
		message += fmt.Sprintf("  Synthesized Facts: %d\n", len(after.Enrichment.Facts))
		for i, fact := range after.Enrichment.Facts {
			if i < 5 {
				message += fmt.Sprintf("    %d. %s\n", i+1, fact.Statement)
				if fact.Confidence > 0 {
					message += fmt.Sprintf("       Confidence: %.2f, Source: %s\n", fact.Confidence, fact.Source)
				}
			}
		}
		if len(after.Enrichment.Facts) > 5 {
			message += fmt.Sprintf("    ... and %d more\n", len(after.Enrichment.Facts)-5)
		}
	} else {
		message += "  • No facts synthesized\n"
	}

	if len(after.Enrichment.DerivedKnowledge) > 0 {
		message += fmt.Sprintf("\n  Derived Knowledge: %d items\n", len(after.Enrichment.DerivedKnowledge))
		for i, knowledge := range after.Enrichment.DerivedKnowledge {
			if i < 3 {
				message += fmt.Sprintf("    %d. %s\n", i+1, knowledge.Insight)
				if len(knowledge.Sources) > 0 {
					message += fmt.Sprintf("       Sources: %v\n", knowledge.Sources)
				}
			}
		}
		if len(after.Enrichment.DerivedKnowledge) > 3 {
			message += fmt.Sprintf("    ... and %d more\n", len(after.Enrichment.DerivedKnowledge)-3)
		}
	}

	if len(after.Enrichment.Relationships) > 0 {
		message += fmt.Sprintf("\n  Relationships: %d found\n", len(after.Enrichment.Relationships))
		for i, rel := range after.Enrichment.Relationships {
			if i < 3 {
				message += fmt.Sprintf("    %d. %s → [%s] → %s\n", i+1, rel.From, rel.Type, rel.To)
			}
		}
		if len(after.Enrichment.Relationships) > 3 {
			message += fmt.Sprintf("    ... and %d more\n", len(after.Enrichment.Relationships)-3)
		}
	}
	message += "\n"

	// Show LLM interaction details if agent trace is available
	if after.LLM != nil && after.LLM.Cache != nil {
		if traces, ok := after.LLM.Cache["agent_traces"].([]interface{}); ok {
			for _, trace := range traces {
				if traceMap, ok := trace.(map[string]interface{}); ok {
					if agentID, ok := traceMap["agent_id"].(string); ok && agentID == "context_synthesizer" {
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
		if metrics, ok := after.Diagnostics.Performance.AgentMetrics["context_synthesizer"]; ok {
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
func (w *ContextSynthesizerOnlyWorkflow) buildSystemPrompt(ctx *models.AgentContext) string {
	prompt := "You are an AI assistant with access to the following synthesized context:\n\n"

	// Add synthesized facts
	if len(ctx.Enrichment.Facts) > 0 {
		prompt += "**Synthesized Facts:**\n"
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
				prompt += fmt.Sprintf("%d. %s\n", i+1, knowledge.Insight)
				if knowledge.Confidence > 0 {
					prompt += fmt.Sprintf("   Confidence: %.2f\n", knowledge.Confidence)
				}
			}
		}
		if len(ctx.Enrichment.DerivedKnowledge) > 3 {
			prompt += fmt.Sprintf("... and %d more insights\n", len(ctx.Enrichment.DerivedKnowledge)-3)
		}
		prompt += "\n"
	}

	// Add relationships
	if len(ctx.Enrichment.Relationships) > 0 {
		prompt += "**Relationships:**\n"
		for i, rel := range ctx.Enrichment.Relationships {
			if i < 5 {
				prompt += fmt.Sprintf("- %s → [%s] → %s\n", rel.From, rel.Type, rel.To)
			}
		}
		if len(ctx.Enrichment.Relationships) > 5 {
			prompt += fmt.Sprintf("... and %d more relationships\n", len(ctx.Enrichment.Relationships)-5)
		}
		prompt += "\n"
	}

	prompt += "Please use this synthesized context to provide an informed and accurate response to the user's question."

	return prompt
}
