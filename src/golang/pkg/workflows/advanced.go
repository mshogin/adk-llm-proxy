package workflows

import (
	"context"
	"fmt"
	"time"

	appservices "github.com/mshogin/agents/internal/application/services"
	"github.com/mshogin/agents/internal/domain/models"
	domainservices "github.com/mshogin/agents/internal/domain/services"
	"github.com/mshogin/agents/internal/domain/services/agents"
	"github.com/mshogin/agents/internal/infrastructure/config"
)

// AdvancedWorkflow implements multi-agent orchestration using Phase 10 reasoning agents.
// Executes IntentDetectionAgent and InferenceAgent with LLM fallback support.
type AdvancedWorkflow struct {
	reasoningManager *appservices.ReasoningManager
	parallel         bool
}

// NewAdvancedWorkflow creates a new AdvancedWorkflow instance.
// It sets up the complete 8-agent reasoning pipeline (Phase 10 architecture).
func NewAdvancedWorkflow(cfg config.AdvancedConfig, providerRegistry map[string]domainservices.LLMProvider) domainservices.Workflow {
	// Create pipeline configuration with all 8 agents in dependency order
	pipelineConfig := appservices.PipelineConfig{
		Mode: appservices.SequentialMode,
		Agents: []appservices.AgentConfig{
			{
				ID:      "intent_detection",
				Enabled: true,
				Timeout: 5000, // 5 seconds
				Retry:   1,
			},
			{
				ID:        "reasoning_structure",
				Enabled:   true,
				DependsOn: []string{"intent_detection"},
				Timeout:   5000,
				Retry:     1,
			},
			{
				ID:        "retrieval_planner",
				Enabled:   true,
				DependsOn: []string{"intent_detection", "reasoning_structure"},
				Timeout:   5000,
				Retry:     1,
			},
			{
				ID:        "retrieval_executor",
				Enabled:   true,
				DependsOn: []string{"retrieval_planner"},
				Timeout:   10000, // 10 seconds for MCP calls
				Retry:     1,
			},
			{
				ID:        "context_synthesizer",
				Enabled:   true,
				DependsOn: []string{"retrieval_executor"},
				Timeout:   5000,
				Retry:     1,
			},
			{
				ID:        "inference",
				Enabled:   true,
				DependsOn: []string{"intent_detection", "reasoning_structure", "context_synthesizer"},
				Timeout:   10000,
				Retry:     1,
			},
			{
				ID:        "summarization",
				Enabled:   true,
				DependsOn: []string{"intent_detection", "inference"},
				Timeout:   5000,
				Retry:     1,
			},
			{
				ID:        "validation",
				Enabled:   true,
				DependsOn: []string{"intent_detection", "reasoning_structure", "inference"},
				Timeout:   5000,
				Retry:     1,
			},
		},
		Options: appservices.DefaultExecutionOptions(),
	}

	// Create reasoning manager
	manager := appservices.NewReasoningManager(pipelineConfig)

	// Create LLM orchestrator with real providers for intelligent reasoning
	var llmOrchestrator *appservices.LLMOrchestrator
	if len(providerRegistry) > 0 {
		llmOrchestrator = appservices.NewLLMOrchestrator()
		llmOrchestrator.RegisterProviders(providerRegistry)
	} else {
		// No providers available, agents will work in rule-based mode only
		llmOrchestrator = nil
	}

	// Register all 8 agents
	intentAgent := agents.NewIntentDetectionAgent(llmOrchestrator)
	reasoningAgent := agents.NewReasoningStructureAgent()
	retrievalPlannerAgent := agents.NewRetrievalPlannerAgent()
	retrievalExecutorAgent := agents.NewRetrievalExecutorAgent(nil, 5, 10*time.Second) // nil DataSource for now, 5 concurrent, 10s timeout
	contextSynthesizerAgent := agents.NewContextSynthesizerAgent()
	inferenceAgent := agents.NewInferenceAgent(llmOrchestrator)
	summarizationAgent := agents.NewSummarizationAgent()
	validationAgent := agents.NewValidationAgent()

	_ = manager.RegisterAgent(intentAgent)
	_ = manager.RegisterAgent(reasoningAgent)
	_ = manager.RegisterAgent(retrievalPlannerAgent)
	_ = manager.RegisterAgent(retrievalExecutorAgent)
	_ = manager.RegisterAgent(contextSynthesizerAgent)
	_ = manager.RegisterAgent(inferenceAgent)
	_ = manager.RegisterAgent(summarizationAgent)
	_ = manager.RegisterAgent(validationAgent)

	return &AdvancedWorkflow{
		reasoningManager: manager,
		parallel:         cfg.ParallelExecution,
	}
}

// Name returns the workflow identifier.
func (w *AdvancedWorkflow) Name() string {
	return "advanced"
}

// Execute processes the reasoning input using Phase 10 reasoning agents.
// It creates an AgentContext, runs the reasoning pipeline, and converts results to ReasoningResult.
func (w *AdvancedWorkflow) Execute(ctx context.Context, input *models.ReasoningInput) (*models.ReasoningResult, error) {
	// Check context cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Create AgentContext from ReasoningInput
	agentContext := w.createAgentContext(input)

	// Execute reasoning pipeline
	resultContext, err := w.reasoningManager.Execute(ctx, agentContext)
	if err != nil {
		return nil, fmt.Errorf("reasoning pipeline failed: %w", err)
	}

	// Convert AgentContext to ReasoningResult
	result := w.convertToReasoningResult(resultContext)

	return result, nil
}

// createAgentContext converts ReasoningInput to AgentContext.
func (w *AdvancedWorkflow) createAgentContext(input *models.ReasoningInput) *models.AgentContext {
	// Generate session and trace IDs from current time
	now := time.Now()
	sessionID := fmt.Sprintf("session-%d", now.Unix())
	traceID := fmt.Sprintf("trace-%d", now.UnixNano())

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

		// IMPORTANT: Also store original user input in LLM cache so it doesn't get lost
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

// convertToReasoningResult converts AgentContext to ReasoningResult.
func (w *AdvancedWorkflow) convertToReasoningResult(ctx *models.AgentContext) *models.ReasoningResult {
	result := models.NewReasoningResult("advanced", "Multi-agent reasoning completed")

	// Build system prompt from reasoning context
	systemPrompt := w.buildSystemPrompt(ctx)

	// Populate EnrichedMessages with system prompt
	result.EnrichedMessages = []models.Message{
		{
			Role:    "system",
			Content: systemPrompt,
		},
	}

	// Build detailed summary message from reasoning results
	message := "=== Phase 10 Reasoning Pipeline ===\n\n"

	// 1. Intent Detection Agent
	message += "🎯 Intent Detection:\n"
	// Show original user input from LLM cache
	if ctx.LLM != nil && ctx.LLM.Cache != nil {
		if originalInput, ok := ctx.LLM.Cache["original_user_input"].(string); ok {
			message += fmt.Sprintf("  INPUT (user message): \"%s\"\n", originalInput)
		}
	}
	// Show output
	message += "  OUTPUT:\n"
	if len(ctx.Reasoning.Intents) > 0 {
		for _, intent := range ctx.Reasoning.Intents {
			message += fmt.Sprintf("    • %s (confidence: %.2f)\n", intent.Type, intent.Confidence)
		}
	} else {
		message += "    • No intents detected\n"
	}

	// Show entities
	if len(ctx.Reasoning.Entities) > 0 {
		message += "  Entities:\n"
		for key, values := range ctx.Reasoning.Entities {
			message += fmt.Sprintf("    • %s: %v\n", key, values)
		}
	}
	message += "\n"

	// 2. Reasoning Structure Agent
	message += "🧩 Reasoning Structure:\n"
	if len(ctx.Reasoning.Hypotheses) > 0 {
		message += fmt.Sprintf("  • Created %d hypotheses\n", len(ctx.Reasoning.Hypotheses))
		for i, h := range ctx.Reasoning.Hypotheses {
			if i < 3 { // Show first 3
				message += fmt.Sprintf("    %d. %s\n", i+1, h.Description)
				if len(h.Dependencies) > 0 {
					message += fmt.Sprintf("       Dependencies: %v\n", h.Dependencies)
				}
			}
		}
		if len(ctx.Reasoning.Hypotheses) > 3 {
			message += fmt.Sprintf("    ... and %d more\n", len(ctx.Reasoning.Hypotheses)-3)
		}
	} else {
		message += "  • No hypotheses created\n"
	}
	message += "\n"

	// 3. Retrieval Planner Agent
	message += "📋 Retrieval Planning:\n"
	if len(ctx.Retrieval.Plans) > 0 {
		message += fmt.Sprintf("  • Generated %d retrieval plan(s)\n", len(ctx.Retrieval.Plans))
		for _, plan := range ctx.Retrieval.Plans {
			sources := "unknown"
			if len(plan.Sources) > 0 {
				sources = plan.Sources[0]
			}
			message += fmt.Sprintf("    • Source: %s, Priority: %d\n", sources, plan.Priority)
		}
	} else {
		message += "  • No retrieval plans generated\n"
	}

	if len(ctx.Retrieval.Queries) > 0 {
		message += fmt.Sprintf("  Generated %d queries:\n", len(ctx.Retrieval.Queries))
		for i, q := range ctx.Retrieval.Queries {
			if i < 3 {
				message += fmt.Sprintf("    • [%s] %s\n", q.Source, q.QueryString)
			}
		}
		if len(ctx.Retrieval.Queries) > 3 {
			message += fmt.Sprintf("    ... and %d more\n", len(ctx.Retrieval.Queries)-3)
		}
	}
	message += "\n"

	// 4. Retrieval Executor Agent
	message += "🔍 Retrieval Execution:\n"
	if len(ctx.Retrieval.Artifacts) > 0 {
		message += fmt.Sprintf("  • Retrieved %d artifact(s)\n", len(ctx.Retrieval.Artifacts))
		for i, art := range ctx.Retrieval.Artifacts {
			if i < 3 {
				message += fmt.Sprintf("    • [%s] %s (type: %s)\n", art.Source, art.ID, art.Type)
			}
		}
		if len(ctx.Retrieval.Artifacts) > 3 {
			message += fmt.Sprintf("    ... and %d more\n", len(ctx.Retrieval.Artifacts)-3)
		}
	} else {
		message += "  • No artifacts retrieved\n"
	}
	message += "\n"

	// 5. Context Synthesizer Agent
	message += "🔬 Context Synthesis:\n"
	if len(ctx.Enrichment.Facts) > 0 {
		message += fmt.Sprintf("  • Extracted %d facts\n", len(ctx.Enrichment.Facts))
		for i, fact := range ctx.Enrichment.Facts {
			if i < 3 {
				message += fmt.Sprintf("    • %s (conf: %.2f)\n", fact.Content, fact.Confidence)
			}
		}
		if len(ctx.Enrichment.Facts) > 3 {
			message += fmt.Sprintf("    ... and %d more\n", len(ctx.Enrichment.Facts)-3)
		}
	} else {
		message += "  • No facts extracted\n"
	}
	message += "\n"

	// 6. Inference Agent
	message += "💡 Inference:\n"
	if len(ctx.Reasoning.Conclusions) > 0 {
		message += fmt.Sprintf("  • Drew %d conclusion(s)\n", len(ctx.Reasoning.Conclusions))
		for i, c := range ctx.Reasoning.Conclusions {
			if i < 3 {
				message += fmt.Sprintf("    %d. %s (conf: %.2f)\n", i+1, c.Description, c.Confidence)
				if len(c.Evidence) > 0 {
					message += fmt.Sprintf("       Evidence: %d item(s)\n", len(c.Evidence))
				}
			}
		}
		if len(ctx.Reasoning.Conclusions) > 3 {
			message += fmt.Sprintf("    ... and %d more\n", len(ctx.Reasoning.Conclusions)-3)
		}
	} else {
		message += "  • No conclusions drawn\n"
	}
	message += "\n"

	// 7. Summarization Agent
	message += "📝 Summarization:\n"
	if ctx.Reasoning.Summary != "" {
		message += fmt.Sprintf("  • %s\n", ctx.Reasoning.Summary)
	} else {
		message += "  • No summary generated\n"
	}

	// Add clarification questions if any
	if len(ctx.Reasoning.ClarificationQuestions) > 0 {
		message += "  Clarification needed:\n"
		for _, q := range ctx.Reasoning.ClarificationQuestions {
			message += fmt.Sprintf("    ? %s\n", q.Question)
		}
	}
	message += "\n"

	// 8. Validation Agent
	message += "✅ Validation:\n"
	if ctx.Diagnostics != nil && len(ctx.Diagnostics.ValidationReports) > 0 {
		passedCount := 0
		for _, report := range ctx.Diagnostics.ValidationReports {
			if report.Passed {
				passedCount++
			}
		}
		message += fmt.Sprintf("  • Passed %d/%d checks\n", passedCount, len(ctx.Diagnostics.ValidationReports))

		// Show failed validations
		for _, report := range ctx.Diagnostics.ValidationReports {
			if !report.Passed && len(report.Issues) > 0 {
				message += fmt.Sprintf("  ⚠ Issues found:\n")
				for _, issue := range report.Issues {
					message += fmt.Sprintf("    • %s\n", issue)
				}
			}
		}

		// Show errors and warnings
		if ctx.Diagnostics.Errors != nil && len(ctx.Diagnostics.Errors) > 0 {
			message += fmt.Sprintf("  ❌ Errors: %d\n", len(ctx.Diagnostics.Errors))
		}
		if ctx.Diagnostics.Warnings != nil && len(ctx.Diagnostics.Warnings) > 0 {
			message += fmt.Sprintf("  ⚠ Warnings: %d\n", len(ctx.Diagnostics.Warnings))
		}
	} else {
		message += "  • All checks passed\n"
	}
	message += "\n"

	// Add detailed agent traces if available
	if ctx.LLM != nil && ctx.LLM.Cache != nil {
		if traces, ok := ctx.LLM.Cache["agent_traces"].([]interface{}); ok && len(traces) > 0 {
		message += "\n=== 🔍 DETAILED AGENT TRACES ===\n\n"
		for i, trace := range traces {
			if traceMap, ok := trace.(map[string]interface{}); ok {
				agentID := ""
				if id, ok := traceMap["agent_id"].(string); ok {
					agentID = id
				}

				message += fmt.Sprintf("**Agent %d: %s**\n", i+1, agentID)

				// Format INPUT based on agent type
				switch agentID {
				case "intent_detection":
					if input, ok := traceMap["input_received"].(string); ok {
						message += fmt.Sprintf("📥 INPUT: \"%s\"\n", input)
					}
				case "reasoning_structure":
					if intents, ok := traceMap["input_intents"].([]models.Intent); ok {
						message += "📥 INPUT:\n"
						for _, intent := range intents {
							message += fmt.Sprintf("  • %s (confidence: %.2f)\n", intent.Type, intent.Confidence)
						}
					}
				case "retrieval_planner":
					if intents, ok := traceMap["input_intents"].([]models.Intent); ok {
						message += "📥 INPUT:\n"
						message += fmt.Sprintf("  • %d intent(s)\n", len(intents))
					}
					if hypotheses, ok := traceMap["input_hypotheses"].([]models.Hypothesis); ok {
						message += fmt.Sprintf("  • %d hypothese(s)\n", len(hypotheses))
					}
				case "retrieval_executor":
					if plans, ok := traceMap["input_plans"].([]models.RetrievalPlan); ok {
						message += "📥 INPUT:\n"
						message += fmt.Sprintf("  • %d retrieval plan(s)\n", len(plans))
					}
				case "context_synthesizer":
					if artifacts, ok := traceMap["input_artifacts"].([]models.Artifact); ok {
						message += "📥 INPUT:\n"
						message += fmt.Sprintf("  • %d artifact(s)\n", len(artifacts))
					}
				case "inference":
					message += "📥 INPUT:\n"
					if intents, ok := traceMap["input_intents"].([]models.Intent); ok {
						message += fmt.Sprintf("  • %d intent(s)\n", len(intents))
					}
					if facts, ok := traceMap["input_facts"].([]models.Fact); ok {
						message += fmt.Sprintf("  • %d fact(s)\n", len(facts))
					}
				case "summarization":
					message += "📥 INPUT:\n"
					if intents, ok := traceMap["input_intents"].([]models.Intent); ok {
						message += fmt.Sprintf("  • %d intent(s)\n", len(intents))
					}
					if conclusions, ok := traceMap["input_conclusions"].([]models.Conclusion); ok {
						message += fmt.Sprintf("  • %d conclusion(s)\n", len(conclusions))
					}
				case "validation":
					message += "📥 INPUT: Complete reasoning context\n"
				}

				// Show LLM interaction if triggered
				if triggered, ok := traceMap["llm_fallback_triggered"].(bool); ok && triggered {
					message += "\n💬 LLM Fallback:\n"
					if reason, ok := traceMap["llm_trigger_reason"].(string); ok {
						message += fmt.Sprintf("  Reason: %s\n", reason)
					}
					if prompt, ok := traceMap["llm_prompt"].(string); ok {
						// Show truncated prompt
						if len(prompt) > 200 {
							message += fmt.Sprintf("  Prompt: %s...\n", prompt[:200])
						} else {
							message += fmt.Sprintf("  Prompt: %s\n", prompt)
						}
					}
					if response, ok := traceMap["llm_response"].(string); ok && response != "" {
						// Show truncated response
						if len(response) > 200 {
							message += fmt.Sprintf("  Response: %s...\n", response[:200])
						} else {
							message += fmt.Sprintf("  Response: %s\n", response)
						}
					}
					if used, ok := traceMap["llm_result_used"].(bool); ok {
						message += fmt.Sprintf("  Result Used: %v\n", used)
					}
				}

				// Format OUTPUT based on agent type - read directly from context
				message += "📤 OUTPUT:\n"
				switch agentID {
				case "intent_detection":
					// Show intents from context
					if len(ctx.Reasoning.Intents) > 0 {
						for _, intent := range ctx.Reasoning.Intents {
							message += fmt.Sprintf("  • %s (confidence: %.2f)\n", intent.Type, intent.Confidence)
						}
					} else {
						message += "  • No intents detected\n"
					}
				case "reasoning_structure":
					// Show hypotheses from context
					if len(ctx.Reasoning.Hypotheses) > 0 {
						message += fmt.Sprintf("  • Generated %d hypotheses\n", len(ctx.Reasoning.Hypotheses))
						for j, h := range ctx.Reasoning.Hypotheses {
							if j < 3 {
								message += fmt.Sprintf("    %d. %s\n", j+1, h.Description)
							}
						}
						if len(ctx.Reasoning.Hypotheses) > 3 {
							message += fmt.Sprintf("    ... and %d more\n", len(ctx.Reasoning.Hypotheses)-3)
						}
					} else {
						message += "  • No hypotheses generated\n"
					}
				case "retrieval_planner":
					// Show plans and queries from context
					if len(ctx.Retrieval.Plans) > 0 {
						message += fmt.Sprintf("  • Generated %d retrieval plan(s)\n", len(ctx.Retrieval.Plans))
					} else {
						message += "  • No plans generated\n"
					}
					// Note: Query[0] is the original user input, so real queries start from index 1
					if len(ctx.Retrieval.Queries) > 1 {
						message += fmt.Sprintf("  • Generated %d quer(ies)\n", len(ctx.Retrieval.Queries)-1)
					}
				case "retrieval_executor":
					// Show artifacts from context
					if len(ctx.Retrieval.Artifacts) > 0 {
						message += fmt.Sprintf("  • Retrieved %d artifact(s)\n", len(ctx.Retrieval.Artifacts))
					} else {
						message += "  • No artifacts retrieved\n"
					}
				case "context_synthesizer":
					// Show facts from context
					if len(ctx.Enrichment.Facts) > 0 {
						message += fmt.Sprintf("  • Extracted %d fact(s)\n", len(ctx.Enrichment.Facts))
					} else {
						message += "  • No facts extracted\n"
					}
				case "inference":
					// Show conclusions from context
					if len(ctx.Reasoning.Conclusions) > 0 {
						message += fmt.Sprintf("  • Drew %d conclusion(s)\n", len(ctx.Reasoning.Conclusions))
						for j, c := range ctx.Reasoning.Conclusions {
							if j < 2 {
								message += fmt.Sprintf("    %d. %s (conf: %.2f)\n", j+1, c.Description, c.Confidence)
							}
						}
					} else {
						message += "  • No conclusions drawn\n"
					}
				case "summarization":
					// Show summary from context
					if ctx.Reasoning.Summary != "" {
						message += fmt.Sprintf("  • %s\n", ctx.Reasoning.Summary)
					} else {
						message += "  • No summary generated\n"
					}
				case "validation":
					// Show validation results from context
					if ctx.Diagnostics != nil && len(ctx.Diagnostics.ValidationReports) > 0 {
						passed := 0
						for _, report := range ctx.Diagnostics.ValidationReports {
							if report.Passed {
								passed++
							}
						}
						message += fmt.Sprintf("  • Passed %d/%d checks\n", passed, len(ctx.Diagnostics.ValidationReports))
					} else {
						message += "  • No validation performed\n"
					}
				}

				message += "\n"
			}
		}
		}
	}

	// Add agent execution summary from audit
	if len(ctx.Audit.AgentRuns) > 0 {
		message += "\nAgent Execution Summary:\n"
		for _, run := range ctx.Audit.AgentRuns {
			status := "✓"
			if run.Status != "success" {
				status = "✗"
			}
			message += fmt.Sprintf("  %s %s: %dms\n", status, run.AgentID, run.DurationMS)
		}
	}

	// Add performance metrics
	if ctx.Diagnostics != nil && ctx.Diagnostics.Performance != nil {
		message += fmt.Sprintf("\nTotal Duration: %dms\n", ctx.Diagnostics.Performance.TotalDurationMS)
	}

	// === Show what gets sent to LLM ===
	message += "\n=== 📤 LLM ENRICHMENT ===\n\n"
	message += "**System Prompt (sent to LLM):**\n"
	message += "```\n"
	message += systemPrompt
	message += "\n```\n\n"

	message += "**User Message (original):**\n"
	if ctx.LLM != nil && ctx.LLM.Cache != nil {
		if originalInput, ok := ctx.LLM.Cache["original_user_input"].(string); ok {
			message += fmt.Sprintf("\"%s\"\n", originalInput)
		} else {
			message += "[No user message]\n"
		}
	} else {
		message += "[No user message]\n"
	}
	message += "\n"

	message += "✅ Reasoning context is now enriching the LLM prompt!\n"
	message += fmt.Sprintf("   - %d intents detected\n", len(ctx.Reasoning.Intents))
	message += fmt.Sprintf("   - %d facts extracted\n", len(ctx.Enrichment.Facts))
	message += fmt.Sprintf("   - %d artifacts retrieved\n", len(ctx.Retrieval.Artifacts))
	message += fmt.Sprintf("   - %d conclusions drawn\n", len(ctx.Reasoning.Conclusions))
	message += "\n"

	result.Message = message

	// Convert agent runs to legacy AgentResult format for compatibility
	for _, run := range ctx.Audit.AgentRuns {
		agentResult := &models.AgentResult{
			AgentName: run.AgentID,
			Output:    fmt.Sprintf("%s completed", run.AgentID),
			Success:   run.Status == "success",
			Duration:  run.DurationMS,
		}
		if run.Error != "" {
			agentResult.Error = run.Error
			agentResult.Success = false
		}
		result.AddAgentResult(run.AgentID, agentResult)
	}

	return result
}

// buildSystemPrompt creates a concise system prompt from reasoning context.
func (w *AdvancedWorkflow) buildSystemPrompt(ctx *models.AgentContext) string {
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

	// Add extracted facts
	if len(ctx.Enrichment.Facts) > 0 {
		prompt += "**Extracted Facts:**\n"
		for i, fact := range ctx.Enrichment.Facts {
			if i < 5 { // Limit to top 5 facts
				prompt += fmt.Sprintf("- %s (confidence: %.2f)\n", fact.Content, fact.Confidence)
			}
		}
		if len(ctx.Enrichment.Facts) > 5 {
			prompt += fmt.Sprintf("... and %d more facts\n", len(ctx.Enrichment.Facts)-5)
		}
		prompt += "\n"
	}

	// Add retrieved artifacts
	if len(ctx.Retrieval.Artifacts) > 0 {
		prompt += "**Retrieved Artifacts:**\n"
		for i, artifact := range ctx.Retrieval.Artifacts {
			if i < 3 { // Limit to top 3 artifacts
				prompt += fmt.Sprintf("- [%s] %s (type: %s)\n", artifact.Source, artifact.ID, artifact.Type)
				if artifact.Content != nil {
					if contentMap, ok := artifact.Content.(map[string]interface{}); ok {
						if summary, ok := contentMap["summary"].(string); ok && summary != "" {
							prompt += fmt.Sprintf("  Summary: %s\n", summary)
						}
					}
				}
			}
		}
		if len(ctx.Retrieval.Artifacts) > 3 {
			prompt += fmt.Sprintf("... and %d more artifacts\n", len(ctx.Retrieval.Artifacts)-3)
		}
		prompt += "\n"
	}

	// Add reasoning conclusions
	if len(ctx.Reasoning.Conclusions) > 0 {
		prompt += "**Reasoning Conclusions:**\n"
		for i, conclusion := range ctx.Reasoning.Conclusions {
			if i < 3 { // Limit to top 3 conclusions
				prompt += fmt.Sprintf("- %s (confidence: %.2f)\n", conclusion.Description, conclusion.Confidence)
			}
		}
		if len(ctx.Reasoning.Conclusions) > 3 {
			prompt += fmt.Sprintf("... and %d more conclusions\n", len(ctx.Reasoning.Conclusions)-3)
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

	// Add summary if available
	if ctx.Reasoning.Summary != "" {
		prompt += "**Summary:**\n"
		prompt += ctx.Reasoning.Summary + "\n\n"
	}

	prompt += "Please use this context to provide an informed and accurate response to the user's question."

	return prompt
}
