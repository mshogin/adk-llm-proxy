package workflows

import (
	"context"
	"fmt"
	"time"

	"github.com/mshogin/agents/internal/domain/models"
	"github.com/mshogin/agents/internal/domain/services/agents"
	appservices "github.com/mshogin/agents/internal/application/services"
)

// SummarizationOnlyWorkflow executes only the SummarizationAgent for isolated testing.
// This workflow is designed for testing and debugging the SummarizationAgent in isolation
// without running the full 8-agent pipeline.
type SummarizationOnlyWorkflow struct {
	agent           *agents.SummarizationAgent
	llmOrchestrator *appservices.LLMOrchestrator
}

// NewSummarizationOnlyWorkflow creates a new workflow that executes only SummarizationAgent.
func NewSummarizationOnlyWorkflow(llmOrchestrator *appservices.LLMOrchestrator) *SummarizationOnlyWorkflow {
	return &SummarizationOnlyWorkflow{
		agent:           agents.NewSummarizationAgent(llmOrchestrator),
		llmOrchestrator: llmOrchestrator,
	}
}

// Name returns the workflow identifier.
func (w *SummarizationOnlyWorkflow) Name() string {
	return "summarization_only"
}

// Execute runs only the SummarizationAgent with the provided input.
// Returns a detailed result showing INPUT, OUTPUT, LLM interactions, and system prompt.
func (w *SummarizationOnlyWorkflow) Execute(ctx context.Context, input *models.ReasoningInput) (*models.ReasoningResult, error) {
	startTime := time.Now()

	// 1. Create agent context with mock preconditions (complete reasoning context)
	agentContext := w.createContextFromInput(input)

	// Store "before" state for comparison
	beforeContext := agentContext

	// 2. Execute only SummarizationAgent
	resultContext, err := w.agent.Execute(ctx, agentContext)
	if err != nil {
		return nil, fmt.Errorf("summarization agent failed: %w", err)
	}

	// 3. Build detailed result showing INPUT/OUTPUT/SYSTEM_PROMPT
	result := w.buildDetailedResult(beforeContext, resultContext, time.Since(startTime))

	return result, nil
}

// createContextFromInput creates an AgentContext from ReasoningInput with mock preconditions.
// SummarizationAgent requires: complete reasoning context (conclusions, facts, etc.).
func (w *SummarizationOnlyWorkflow) createContextFromInput(input *models.ReasoningInput) *models.AgentContext {
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

	// Mock preconditions: Create complete reasoning context
	ctx.Reasoning.Intents = []models.Intent{
		{Type: "query_commits", Confidence: 0.99},
	}

	ctx.Reasoning.Hypotheses = []models.Hypothesis{
		{
			Description: "User wants to see recent commit history from GitLab",
			Confidence:  0.95,
			Type:        "query",
		},
	}

	ctx.Enrichment.Facts = []models.Fact{
		{
			Statement:  "New JWT authentication feature was implemented on 2024-01-15",
			Source:     "gitlab-commit-123",
			Confidence: 0.95,
		},
		{
			Statement:  "Authentication bug was fixed on 2024-01-16",
			Source:     "gitlab-commit-124",
			Confidence: 0.93,
		},
	}

	ctx.Enrichment.DerivedKnowledge = []models.DerivedKnowledge{
		{
			Insight:    "Recent commits show active development in authentication module",
			Confidence: 0.92,
			Sources:    []string{"gitlab-commit-123", "gitlab-commit-124"},
		},
	}

	ctx.Reasoning.Conclusions = []models.Conclusion{
		{
			Statement:  "The authentication module has been recently updated with new features and bug fixes",
			Confidence: 0.94,
			Type:       "finding",
			Evidence:   []string{"fact-1", "fact-2", "knowledge-1"},
		},
		{
			Statement:  "User likely wants information about authentication-related commits",
			Confidence: 0.91,
			Type:       "intent_fulfillment",
			Evidence:   []string{"intent: query_commits", "hypothesis-1"},
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
func (w *SummarizationOnlyWorkflow) buildDetailedResult(before, after *models.AgentContext, duration time.Duration) *models.ReasoningResult {
	result := models.NewReasoningResult("summarization_only", "Single agent test completed")

	message := "=== SUMMARIZATION AGENT TEST ===\n\n"

	// Show INPUT (preconditions from all previous agents)
	message += "📥 INPUT (Complete Reasoning Context):\n"
	if len(before.Reasoning.Intents) > 0 {
		message += fmt.Sprintf("  Intents: %d\n", len(before.Reasoning.Intents))
	}
	if len(before.Reasoning.Hypotheses) > 0 {
		message += fmt.Sprintf("  Hypotheses: %d\n", len(before.Reasoning.Hypotheses))
	}
	if len(before.Enrichment.Facts) > 0 {
		message += fmt.Sprintf("  Facts: %d\n", len(before.Enrichment.Facts))
	}
	if len(before.Enrichment.DerivedKnowledge) > 0 {
		message += fmt.Sprintf("  Derived Knowledge: %d items\n", len(before.Enrichment.DerivedKnowledge))
	}
	if len(before.Reasoning.Conclusions) > 0 {
		message += fmt.Sprintf("  Conclusions: %d\n", len(before.Reasoning.Conclusions))
		for i, conclusion := range before.Reasoning.Conclusions {
			if i < 2 {
				message += fmt.Sprintf("    %d. %s (%.2f)\n", i+1, conclusion.Statement, conclusion.Confidence)
			}
		}
		if len(before.Reasoning.Conclusions) > 2 {
			message += fmt.Sprintf("    ... and %d more\n", len(before.Reasoning.Conclusions)-2)
		}
	}

	if before.LLM != nil && before.LLM.Cache != nil {
		if input, ok := before.LLM.Cache["original_user_input"].(string); ok && input != "" {
			message += fmt.Sprintf("\n  User message: \"%s\"\n", input)
		}
	}
	message += "\n"

	// Show OUTPUT (summary and final response)
	message += "📤 OUTPUT:\n"
	if after.Reasoning.Summary != "" {
		message += "  Executive Summary:\n"
		summaryLines := splitIntoLines(after.Reasoning.Summary, 80)
		for _, line := range summaryLines {
			if len(summaryLines) <= 10 || len(line) > 0 {
				message += fmt.Sprintf("    %s\n", line)
			}
		}
	} else {
		message += "  • No summary generated\n"
	}

	// Show final response if available
	if after.Reasoning.FinalResponse != "" {
		message += "\n  Final Response:\n"
		responseLines := splitIntoLines(after.Reasoning.FinalResponse, 80)
		for i, line := range responseLines {
			if i < 15 {
				message += fmt.Sprintf("    %s\n", line)
			}
		}
		if len(responseLines) > 15 {
			message += fmt.Sprintf("    ... (%d more lines)\n", len(responseLines)-15)
		}
	}
	message += "\n"

	// Show LLM interaction details if agent trace is available
	if after.LLM != nil && after.LLM.Cache != nil {
		if traces, ok := after.LLM.Cache["agent_traces"].([]interface{}); ok {
			for _, trace := range traces {
				if traceMap, ok := trace.(map[string]interface{}); ok {
					if agentID, ok := traceMap["agent_id"].(string); ok && agentID == "summarization" {
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
		if metrics, ok := after.Diagnostics.Performance.AgentMetrics["summarization"]; ok {
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
func (w *SummarizationOnlyWorkflow) buildSystemPrompt(ctx *models.AgentContext) string {
	prompt := "You are an AI assistant. Please provide a final response based on the following reasoning:\n\n"

	// Add conclusions (most important)
	if len(ctx.Reasoning.Conclusions) > 0 {
		prompt += "**Conclusions:**\n"
		for i, conclusion := range ctx.Reasoning.Conclusions {
			prompt += fmt.Sprintf("%d. %s (confidence: %.2f)\n", i+1, conclusion.Statement, conclusion.Confidence)
		}
		prompt += "\n"
	}

	// Add summary if available
	if ctx.Reasoning.Summary != "" {
		prompt += "**Executive Summary:**\n"
		prompt += ctx.Reasoning.Summary
		prompt += "\n\n"
	}

	// Add key facts
	if len(ctx.Enrichment.Facts) > 0 {
		prompt += "**Key Facts:**\n"
		for i, fact := range ctx.Enrichment.Facts {
			if i < 3 {
				prompt += fmt.Sprintf("- %s\n", fact.Statement)
			}
		}
		if len(ctx.Enrichment.Facts) > 3 {
			prompt += fmt.Sprintf("... and %d more facts\n", len(ctx.Enrichment.Facts)-3)
		}
		prompt += "\n"
	}

	// Add derived knowledge
	if len(ctx.Enrichment.DerivedKnowledge) > 0 {
		prompt += "**Insights:**\n"
		for i, knowledge := range ctx.Enrichment.DerivedKnowledge {
			if i < 2 {
				prompt += fmt.Sprintf("- %s\n", knowledge.Insight)
			}
		}
		if len(ctx.Enrichment.DerivedKnowledge) > 2 {
			prompt += fmt.Sprintf("... and %d more insights\n", len(ctx.Enrichment.DerivedKnowledge)-2)
		}
		prompt += "\n"
	}

	prompt += "Please provide a clear, concise, and helpful response to the user's question."

	return prompt
}

// splitIntoLines splits text into lines with maximum width
func splitIntoLines(text string, maxWidth int) []string {
	if text == "" {
		return []string{}
	}

	var lines []string
	currentLine := ""

	words := splitBySpaces(text)
	for _, word := range words {
		if len(currentLine)+len(word)+1 <= maxWidth {
			if currentLine != "" {
				currentLine += " "
			}
			currentLine += word
		} else {
			if currentLine != "" {
				lines = append(lines, currentLine)
			}
			currentLine = word
		}
	}

	if currentLine != "" {
		lines = append(lines, currentLine)
	}

	return lines
}

// splitBySpaces splits text by spaces and newlines
func splitBySpaces(text string) []string {
	var words []string
	currentWord := ""

	for _, char := range text {
		if char == ' ' || char == '\n' || char == '\t' {
			if currentWord != "" {
				words = append(words, currentWord)
				currentWord = ""
			}
			if char == '\n' {
				words = append(words, "\n")
			}
		} else {
			currentWord += string(char)
		}
	}

	if currentWord != "" {
		words = append(words, currentWord)
	}

	return words
}
