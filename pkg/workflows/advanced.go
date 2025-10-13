package workflows

import (
	"context"
	"fmt"

	"github.com/mshogin/agents/internal/domain/models"
	"github.com/mshogin/agents/internal/domain/services"
	"github.com/mshogin/agents/internal/infrastructure/agents"
	"github.com/mshogin/agents/internal/infrastructure/config"
)

// AdvancedWorkflow implements multi-agent orchestration.
// Executes ADK (Python) and OpenAI (native Go) agents in parallel.
type AdvancedWorkflow struct {
	adkAgent    *agents.ADKAgent
	openaiAgent *agents.OpenAIAgent
	parallel    bool
}

// NewAdvancedWorkflow creates a new AdvancedWorkflow instance.
func NewAdvancedWorkflow(cfg config.AdvancedConfig) services.Workflow {
	// Note: In a real implementation, you would pass the OpenAI provider here
	// For now, we'll create a placeholder
	return &AdvancedWorkflow{
		adkAgent: agents.NewADKAgent("python3", cfg.ADKAgentPath, cfg.ADKTimeout),
		// openaiAgent would be initialized with actual provider
		parallel: cfg.ParallelExecution,
	}
}

// Name returns the workflow identifier.
func (w *AdvancedWorkflow) Name() string {
	return "advanced"
}

// Execute processes the reasoning input using multiple agents.
func (w *AdvancedWorkflow) Execute(ctx context.Context, input *models.ReasoningInput) (*models.ReasoningResult, error) {
	// Check context cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	result := models.NewReasoningResult("advanced", "Multi-agent analysis completed")

	if w.parallel {
		// Execute agents in parallel
		w.executeParallel(ctx, input, result)
	} else {
		// Execute agents sequentially
		w.executeSequential(ctx, input, result)
	}

	// Aggregate results
	result.Message = w.aggregateResults(result)

	return result, nil
}

// executeParallel runs both agents concurrently.
func (w *AdvancedWorkflow) executeParallel(ctx context.Context, input *models.ReasoningInput, result *models.ReasoningResult) {
	adkChan := make(chan *models.AgentResult, 1)
	openaiChan := make(chan *models.AgentResult, 1)

	// Start ADK agent
	go func() {
		adkResult, _ := w.adkAgent.Execute(ctx, input)
		adkChan <- adkResult
	}()

	// Start OpenAI agent (placeholder - would use actual agent)
	go func() {
		// Placeholder result since we don't have OpenAI provider initialized yet
		openaiChan <- &models.AgentResult{
			AgentName: "openai_agent",
			Output:    "OpenAI agent analysis (placeholder)",
			Success:   true,
			Duration:  100,
		}
	}()

	// Wait for both agents
	adkResult := <-adkChan
	openaiResult := <-openaiChan

	// Add results
	result.AddAgentResult("adk_agent", adkResult)
	result.AddAgentResult("openai_agent", openaiResult)
}

// executeSequential runs agents one after another.
func (w *AdvancedWorkflow) executeSequential(ctx context.Context, input *models.ReasoningInput, result *models.ReasoningResult) {
	// Execute ADK agent
	adkResult, _ := w.adkAgent.Execute(ctx, input)
	result.AddAgentResult("adk_agent", adkResult)

	// Execute OpenAI agent (placeholder)
	openaiResult := &models.AgentResult{
		AgentName: "openai_agent",
		Output:    "OpenAI agent analysis (placeholder)",
		Success:   true,
		Duration:  100,
	}
	result.AddAgentResult("openai_agent", openaiResult)
}

// aggregateResults combines agent results into a final message.
func (w *AdvancedWorkflow) aggregateResults(result *models.ReasoningResult) string {
	if result.HasError() {
		return "Some agents failed during execution. Check AgentResults for details."
	}

	// Combine successful results
	message := "Advanced workflow analysis:\n"
	for name, agentResult := range result.AgentResults {
		if agentResult.Success {
			message += fmt.Sprintf("- %s: %s (%.0fms)\n", name, agentResult.Output, float64(agentResult.Duration))
		}
	}

	return message
}
