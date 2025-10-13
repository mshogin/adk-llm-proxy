package workflows

import (
	"context"

	"github.com/mshogin/agents/internal/domain/models"
	"github.com/mshogin/agents/internal/domain/services"
)

// DefaultWorkflow is a simple pass-through workflow that returns "Hello World".
// Useful for testing and demonstration purposes.
type DefaultWorkflow struct{}

// NewDefaultWorkflow creates a new DefaultWorkflow instance.
func NewDefaultWorkflow() services.Workflow {
	return &DefaultWorkflow{}
}

// Name returns the workflow identifier.
func (w *DefaultWorkflow) Name() string {
	return "default"
}

// Execute processes the reasoning input and returns a simple message.
func (w *DefaultWorkflow) Execute(ctx context.Context, input *models.ReasoningInput) (*models.ReasoningResult, error) {
	// Check context cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Return simple result
	result := models.NewReasoningResult("default", "Hello World from Default Workflow")
	result.Confidence = 1.0

	return result, nil
}
