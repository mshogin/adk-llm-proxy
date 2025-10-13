package services

import (
	"context"

	"github.com/mshogin/agents/internal/domain/models"
)

// Workflow defines the interface for reasoning workflow implementations.
// Workflows are responsible for processing user input before sending it to the LLM.
//
// This interface follows the Strategy Pattern - different workflows can be swapped
// at runtime based on configuration or user preferences.
//
// Available workflow implementations:
// - DefaultWorkflow: Simple pass-through (returns "Hello World")
// - BasicWorkflow: Intent detection via regex/keywords (no LLM calls)
// - AdvancedWorkflow: Multi-agent orchestration (ADK Python + OpenAI native)
type Workflow interface {
	// Name returns the workflow's identifier (e.g., "default", "basic", "advanced")
	Name() string

	// Execute processes the reasoning input and returns a reasoning result.
	// This is where the workflow's core logic is implemented.
	//
	// Parameters:
	//   ctx: Context for cancellation and timeout control
	//   input: The reasoning input containing user messages and metadata
	//
	// Returns:
	//   *ReasoningResult: The result of reasoning (insights, transformed messages, etc.)
	//   error: Any error that occurred during reasoning
	//
	// Usage:
	//   result, err := workflow.Execute(ctx, input)
	//   if err != nil { return err }
	//   // Use result.Message, result.EnrichedMessages, etc.
	Execute(ctx context.Context, input *models.ReasoningInput) (*models.ReasoningResult, error)
}
