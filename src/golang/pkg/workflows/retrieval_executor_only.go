package workflows

import (
	"context"
	"fmt"
	"time"

	"github.com/mshogin/agents/internal/domain/models"
	"github.com/mshogin/agents/internal/domain/services/agents"
	appservices "github.com/mshogin/agents/internal/application/services"
)

// RetrievalExecutorOnlyWorkflow executes only the RetrievalExecutorAgent for isolated testing.
// This workflow is designed for testing and debugging the RetrievalExecutorAgent in isolation
// without running the full 8-agent pipeline.
type RetrievalExecutorOnlyWorkflow struct {
	agent           *agents.RetrievalExecutorAgent
	llmOrchestrator *appservices.LLMOrchestrator
}

// NewRetrievalExecutorOnlyWorkflow creates a new workflow that executes only RetrievalExecutorAgent.
func NewRetrievalExecutorOnlyWorkflow(llmOrchestrator *appservices.LLMOrchestrator) *RetrievalExecutorOnlyWorkflow {
	return &RetrievalExecutorOnlyWorkflow{
		agent:           agents.NewRetrievalExecutorAgent(llmOrchestrator),
		llmOrchestrator: llmOrchestrator,
	}
}

// Name returns the workflow identifier.
func (w *RetrievalExecutorOnlyWorkflow) Name() string {
	return "retrieval_executor_only"
}

// Execute runs only the RetrievalExecutorAgent with the provided input.
// Returns a detailed result showing INPUT, OUTPUT, LLM interactions, and system prompt.
func (w *RetrievalExecutorOnlyWorkflow) Execute(ctx context.Context, input *models.ReasoningInput) (*models.ReasoningResult, error) {
	startTime := time.Now()

	// 1. Create agent context with mock preconditions (retrieval plans + queries)
	agentContext := w.createContextFromInput(input)

	// Store "before" state for comparison
	beforeContext := agentContext

	// 2. Execute only RetrievalExecutorAgent
	resultContext, err := w.agent.Execute(ctx, agentContext)
	if err != nil {
		return nil, fmt.Errorf("retrieval executor agent failed: %w", err)
	}

	// 3. Build detailed result showing INPUT/OUTPUT/SYSTEM_PROMPT
	result := w.buildDetailedResult(beforeContext, resultContext, time.Since(startTime))

	return result, nil
}

// createContextFromInput creates an AgentContext from ReasoningInput with mock preconditions.
// RetrievalExecutorAgent requires: retrieval plans and queries.
func (w *RetrievalExecutorOnlyWorkflow) createContextFromInput(input *models.ReasoningInput) *models.AgentContext {
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

	// Mock preconditions: Create sample retrieval plans and queries
	ctx.Retrieval.Plans = []models.RetrievalPlan{
		{
			Source:      "gitlab",
			Description: "Fetch recent commits from GitLab",
			Priority:    1,
			MaxResults:  10,
			Filters: map[string]interface{}{
				"project": "gitlab-project",
				"since":   "2024-01-01",
			},
		},
		{
			Source:      "youtrack",
			Description: "Fetch related issues from YouTrack",
			Priority:    2,
			MaxResults:  5,
			Filters: map[string]interface{}{
				"project": "ADK",
				"state":   "open",
			},
		},
	}

	ctx.Retrieval.Queries = []models.Query{
		{
			QueryString: "recent commits in gitlab-project",
			Source:      "gitlab",
			Filters: map[string]interface{}{
				"project": "gitlab-project",
				"limit":   10,
			},
		},
		{
			QueryString: "open issues in ADK project",
			Source:      "youtrack",
			Filters: map[string]interface{}{
				"project": "ADK",
				"state":   "open",
			},
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
func (w *RetrievalExecutorOnlyWorkflow) buildDetailedResult(before, after *models.AgentContext, duration time.Duration) *models.ReasoningResult {
	result := models.NewReasoningResult("retrieval_executor_only", "Single agent test completed")

	message := "=== RETRIEVAL EXECUTOR AGENT TEST ===\n\n"

	// Show INPUT (preconditions from RetrievalPlannerAgent)
	message += "📥 INPUT (Preconditions from RetrievalPlanner):\n"
	if len(before.Retrieval.Plans) > 0 {
		message += fmt.Sprintf("  Retrieval Plans: %d\n", len(before.Retrieval.Plans))
		for i, plan := range before.Retrieval.Plans {
			message += fmt.Sprintf("    %d. Source: %s, Priority: %d\n", i+1, plan.Source, plan.Priority)
			if plan.Description != "" {
				message += fmt.Sprintf("       %s\n", plan.Description)
			}
		}
	}

	if len(before.Retrieval.Queries) > 0 {
		message += fmt.Sprintf("  Queries: %d\n", len(before.Retrieval.Queries))
		for i, query := range before.Retrieval.Queries {
			if i < 3 {
				message += fmt.Sprintf("    %d. %s (Source: %s)\n", i+1, query.QueryString, query.Source)
			}
		}
		if len(before.Retrieval.Queries) > 3 {
			message += fmt.Sprintf("    ... and %d more\n", len(before.Retrieval.Queries)-3)
		}
	}

	if before.LLM != nil && before.LLM.Cache != nil {
		if input, ok := before.LLM.Cache["original_user_input"].(string); ok && input != "" {
			message += fmt.Sprintf("  User message: \"%s\"\n", input)
		}
	}
	message += "\n"

	// Show OUTPUT (artifacts retrieved from external sources)
	message += "📤 OUTPUT:\n"
	if len(after.Retrieval.Artifacts) > 0 {
		message += fmt.Sprintf("  Retrieved %d artifacts:\n", len(after.Retrieval.Artifacts))

		// Group artifacts by source
		artifactsBySource := make(map[string][]models.Artifact)
		for _, artifact := range after.Retrieval.Artifacts {
			artifactsBySource[artifact.Source] = append(artifactsBySource[artifact.Source], artifact)
		}

		for source, artifacts := range artifactsBySource {
			message += fmt.Sprintf("    From %s: %d artifacts\n", source, len(artifacts))
			for i, artifact := range artifacts {
				if i < 3 {
					message += fmt.Sprintf("      • Type: %s", artifact.Type)
					if artifact.Title != "" {
						message += fmt.Sprintf(", Title: %s", artifact.Title)
					}
					message += "\n"
					if artifact.Content != "" {
						contentPreview := artifact.Content
						if len(contentPreview) > 80 {
							contentPreview = contentPreview[:80] + "..."
						}
						message += fmt.Sprintf("        %s\n", contentPreview)
					}
				}
			}
			if len(artifacts) > 3 {
				message += fmt.Sprintf("      ... and %d more\n", len(artifacts)-3)
			}
		}
	} else {
		message += "  • No artifacts retrieved (Note: MCP servers may not be available in test mode)\n"
	}
	message += "\n"

	// Show LLM interaction details if agent trace is available
	if after.LLM != nil && after.LLM.Cache != nil {
		if traces, ok := after.LLM.Cache["agent_traces"].([]interface{}); ok {
			for _, trace := range traces {
				if traceMap, ok := trace.(map[string]interface{}); ok {
					if agentID, ok := traceMap["agent_id"].(string); ok && agentID == "retrieval_executor" {
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
		if metrics, ok := after.Diagnostics.Performance.AgentMetrics["retrieval_executor"]; ok {
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
func (w *RetrievalExecutorOnlyWorkflow) buildSystemPrompt(ctx *models.AgentContext) string {
	prompt := "You are an AI assistant with access to the following retrieved information:\n\n"

	// Add retrieval plans
	if len(ctx.Retrieval.Plans) > 0 {
		prompt += "**Retrieval Plans Executed:**\n"
		for i, plan := range ctx.Retrieval.Plans {
			prompt += fmt.Sprintf("%d. Source: %s (Priority: %d)\n", i+1, plan.Source, plan.Priority)
			if plan.Description != "" {
				prompt += fmt.Sprintf("   %s\n", plan.Description)
			}
		}
		prompt += "\n"
	}

	// Add retrieved artifacts
	if len(ctx.Retrieval.Artifacts) > 0 {
		prompt += fmt.Sprintf("**Retrieved Artifacts: %d total**\n\n", len(ctx.Retrieval.Artifacts))

		// Group by source
		artifactsBySource := make(map[string][]models.Artifact)
		for _, artifact := range ctx.Retrieval.Artifacts {
			artifactsBySource[artifact.Source] = append(artifactsBySource[artifact.Source], artifact)
		}

		for source, artifacts := range artifactsBySource {
			prompt += fmt.Sprintf("From %s (%d artifacts):\n", source, len(artifacts))
			for i, artifact := range artifacts {
				if i < 3 {
					prompt += fmt.Sprintf("- Type: %s", artifact.Type)
					if artifact.Title != "" {
						prompt += fmt.Sprintf(", Title: %s", artifact.Title)
					}
					prompt += "\n"
					if artifact.Content != "" {
						contentPreview := artifact.Content
						if len(contentPreview) > 150 {
							contentPreview = contentPreview[:150] + "..."
						}
						prompt += fmt.Sprintf("  %s\n", contentPreview)
					}
				}
			}
			if len(artifacts) > 3 {
				prompt += fmt.Sprintf("... and %d more artifacts\n", len(artifacts)-3)
			}
			prompt += "\n"
		}
	}

	prompt += "Please use this retrieved information to provide an informed and accurate response to the user's question."

	return prompt
}
