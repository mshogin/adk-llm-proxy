package workflows

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/mshogin/agents/internal/application/services"
	"github.com/mshogin/agents/internal/domain/models"
	"github.com/mshogin/agents/internal/domain/services/agents"
)

// DynamicWorkflow executes a custom sequence of agents
// This workflow reuses ALL the same agent logic as the advanced workflow
type DynamicWorkflow struct {
	llmOrchestrator *services.LLMOrchestrator
	agentIDs        []string
	agents          map[string]agents.ReasoningAgent
}

// NewDynamicWorkflow creates a workflow with specified agents (reuses server logic)
func NewDynamicWorkflow(llmOrchestrator *services.LLMOrchestrator, agentIDs []string) *DynamicWorkflow {
	// Create ALL agents (same as advanced workflow)
	agentRegistry := map[string]agents.ReasoningAgent{
		"intent_detection":    agents.NewIntentDetectionAgent(llmOrchestrator),
		"reasoning_structure": agents.NewReasoningStructureAgent(llmOrchestrator),
		"retrieval_planner":   agents.NewRetrievalPlannerAgent(llmOrchestrator),
		"retrieval_executor":  agents.NewRetrievalExecutorAgent(llmOrchestrator),
		"context_synthesizer": agents.NewContextSynthesizerAgent(llmOrchestrator),
		"inference":           agents.NewInferenceAgent(llmOrchestrator),
		"validation":          agents.NewValidationAgent(llmOrchestrator),
		"summarization":       agents.NewSummarizationAgent(llmOrchestrator),
	}

	return &DynamicWorkflow{
		llmOrchestrator: llmOrchestrator,
		agentIDs:        agentIDs,
		agents:          agentRegistry,
	}
}

// Name returns the workflow identifier
func (w *DynamicWorkflow) Name() string {
	return "dynamic"
}

// Execute runs the specified agents sequentially (same logic as advanced workflow)
func (w *DynamicWorkflow) Execute(ctx context.Context, input *models.ReasoningInput) (*models.ReasoningResult, error) {
	startTime := time.Now()

	// Create agent context (same as advanced workflow)
	now := time.Now()
	sessionID := fmt.Sprintf("dynamic-session-%d", now.Unix())
	traceID := fmt.Sprintf("dynamic-trace-%d", now.UnixNano())
	agentContext := models.NewAgentContext(sessionID, traceID)

	// Store user input in context (same as advanced workflow)
	if len(input.Messages) > 0 {
		lastMessage := input.Messages[len(input.Messages)-1]
		if agentContext.LLM == nil {
			agentContext.LLM = &models.LLMContext{
				Cache: make(map[string]interface{}),
			}
		}
		if agentContext.LLM.Cache == nil {
			agentContext.LLM.Cache = make(map[string]interface{})
		}
		agentContext.LLM.Cache["original_user_input"] = lastMessage.Content
		agentContext.LLM.Cache["all_messages"] = input.Messages
	}

	// Build detailed result message
	var message strings.Builder
	message.WriteString("=== DYNAMIC AGENT PIPELINE ===\n\n")
	message.WriteString(fmt.Sprintf("Pipeline: %s\n\n", strings.Join(w.agentIDs, " → ")))

	// Execute each agent sequentially (same pattern as advanced workflow)
	for i, agentID := range w.agentIDs {
		agent, exists := w.agents[agentID]
		if !exists {
			return nil, fmt.Errorf("unknown agent: %s", agentID)
		}

		message.WriteString(fmt.Sprintf("=== Agent %d/%d: %s ===\n", i+1, len(w.agentIDs), agentID))

		// Show input state
		message.WriteString(w.formatAgentInput(agentContext, agentID))

		// Execute agent (SAME EXECUTION LOGIC AS SERVER)
		agentStartTime := time.Now()
		var err error
		agentContext, err = agent.Execute(ctx, agentContext)
		agentDuration := time.Since(agentStartTime)

		if err != nil {
			message.WriteString(fmt.Sprintf("\n❌ Error: %v\n", err))
			return &models.ReasoningResult{
				Message: message.String(),
			}, fmt.Errorf("agent %s failed: %w", agentID, err)
		}

		// Show output state
		message.WriteString(w.formatAgentOutput(agentContext, agentID, agentDuration))
		message.WriteString("\n")
	}

	// Show final result
	message.WriteString("=== FINAL RESULT ===\n\n")
	if agentContext.Reasoning.Summary != "" {
		message.WriteString(fmt.Sprintf("Summary: %s\n\n", agentContext.Reasoning.Summary))
	}
	if agentContext.Reasoning.FinalResponse != "" {
		message.WriteString(fmt.Sprintf("Response: %s\n\n", agentContext.Reasoning.FinalResponse))
	}

	// Show overall metrics
	message.WriteString(fmt.Sprintf("⏱️  Total Pipeline Duration: %dms\n", time.Since(startTime).Milliseconds()))

	result := models.NewReasoningResult("dynamic", message.String())
	return result, nil
}

// formatAgentInput shows the input state for an agent
func (w *DynamicWorkflow) formatAgentInput(ctx *models.AgentContext, agentID string) string {
	var msg strings.Builder
	msg.WriteString("\n📥 INPUT:\n")

	switch agentID {
	case "intent_detection":
		if ctx.LLM != nil && ctx.LLM.Cache != nil {
			if userInput, ok := ctx.LLM.Cache["original_user_input"].(string); ok {
				msg.WriteString(fmt.Sprintf("  User message: \"%s\"\n", userInput))
			}
		}

	case "reasoning_structure":
		msg.WriteString(fmt.Sprintf("  Intents: %d\n", len(ctx.Reasoning.Intents)))
		msg.WriteString(fmt.Sprintf("  Entities: %d\n", len(ctx.Reasoning.Entities)))

	case "retrieval_planner":
		msg.WriteString(fmt.Sprintf("  Intents: %d\n", len(ctx.Reasoning.Intents)))
		msg.WriteString(fmt.Sprintf("  Hypotheses: %d\n", len(ctx.Reasoning.Hypotheses)))

	case "retrieval_executor":
		msg.WriteString(fmt.Sprintf("  Retrieval Plans: %d\n", len(ctx.Retrieval.Plans)))

	case "context_synthesizer":
		msg.WriteString(fmt.Sprintf("  Artifacts: %d\n", len(ctx.Retrieval.Artifacts)))

	case "inference":
		msg.WriteString(fmt.Sprintf("  Hypotheses: %d\n", len(ctx.Reasoning.Hypotheses)))
		msg.WriteString(fmt.Sprintf("  Facts: %d\n", len(ctx.Enrichment.Facts)))

	case "validation":
		msg.WriteString(fmt.Sprintf("  Intents: %d\n", len(ctx.Reasoning.Intents)))
		msg.WriteString(fmt.Sprintf("  Conclusions: %d\n", len(ctx.Reasoning.Conclusions)))

	case "summarization":
		msg.WriteString(fmt.Sprintf("  Conclusions: %d\n", len(ctx.Reasoning.Conclusions)))
		msg.WriteString(fmt.Sprintf("  Facts: %d\n", len(ctx.Enrichment.Facts)))
	}

	return msg.String()
}

// formatAgentOutput shows the output state for an agent
func (w *DynamicWorkflow) formatAgentOutput(ctx *models.AgentContext, agentID string, duration time.Duration) string {
	var msg strings.Builder
	msg.WriteString("\n📤 OUTPUT:\n")

	switch agentID {
	case "intent_detection":
		for _, intent := range ctx.Reasoning.Intents {
			msg.WriteString(fmt.Sprintf("  • %s (confidence: %.2f)\n", intent.Type, intent.Confidence))
		}
		if len(ctx.Reasoning.Entities) > 0 {
			msg.WriteString(fmt.Sprintf("  Entities: %d extracted\n", len(ctx.Reasoning.Entities)))
		}

	case "reasoning_structure":
		msg.WriteString(fmt.Sprintf("  Hypotheses: %d generated\n", len(ctx.Reasoning.Hypotheses)))
		for i, h := range ctx.Reasoning.Hypotheses {
			if i < 3 {
				msg.WriteString(fmt.Sprintf("    %d. %s (%.2f)\n", i+1, h.Description, h.Confidence))
			}
		}

	case "retrieval_planner":
		msg.WriteString(fmt.Sprintf("  Retrieval Plans: %d created\n", len(ctx.Retrieval.Plans)))
		for _, plan := range ctx.Retrieval.Plans {
			msg.WriteString(fmt.Sprintf("    • %s: %s\n", plan.Source, plan.Description))
		}

	case "retrieval_executor":
		msg.WriteString(fmt.Sprintf("  Artifacts: %d retrieved\n", len(ctx.Retrieval.Artifacts)))
		sources := make(map[string]int)
		for _, artifact := range ctx.Retrieval.Artifacts {
			sources[artifact.Source]++
		}
		for source, count := range sources {
			msg.WriteString(fmt.Sprintf("    • %s: %d artifacts\n", source, count))
		}

	case "context_synthesizer":
		msg.WriteString(fmt.Sprintf("  Facts: %d synthesized\n", len(ctx.Enrichment.Facts)))
		msg.WriteString(fmt.Sprintf("  Derived Knowledge: %d items\n", len(ctx.Enrichment.DerivedKnowledge)))
		msg.WriteString(fmt.Sprintf("  Relationships: %d found\n", len(ctx.Enrichment.Relationships)))

	case "inference":
		msg.WriteString(fmt.Sprintf("  Conclusions: %d drawn\n", len(ctx.Reasoning.Conclusions)))
		for i, c := range ctx.Reasoning.Conclusions {
			if i < 3 {
				msg.WriteString(fmt.Sprintf("    %d. %s (%.2f)\n", i+1, c.Statement, c.Confidence))
			}
		}

	case "validation":
		if ctx.Diagnostics != nil {
			msg.WriteString(fmt.Sprintf("  Validation Reports: %d\n", len(ctx.Diagnostics.ValidationReports)))
			msg.WriteString(fmt.Sprintf("  Errors: %d\n", len(ctx.Diagnostics.Errors)))
			msg.WriteString(fmt.Sprintf("  Warnings: %d\n", len(ctx.Diagnostics.Warnings)))

			if len(ctx.Diagnostics.Errors) > 0 {
				msg.WriteString("  Status: ❌ FAILED\n")
			} else if len(ctx.Diagnostics.Warnings) > 0 {
				msg.WriteString("  Status: ⚠️  WARNING\n")
			} else {
				msg.WriteString("  Status: ✅ PASSED\n")
			}
		}

	case "summarization":
		if ctx.Reasoning.Summary != "" {
			msg.WriteString(fmt.Sprintf("  Summary generated (%d chars)\n", len(ctx.Reasoning.Summary)))
		}
		if ctx.Reasoning.FinalResponse != "" {
			msg.WriteString(fmt.Sprintf("  Final response ready (%d chars)\n", len(ctx.Reasoning.FinalResponse)))
		}
	}

	msg.WriteString(fmt.Sprintf("\n⏱️  Duration: %dms\n", duration.Milliseconds()))

	return msg.String()
}
