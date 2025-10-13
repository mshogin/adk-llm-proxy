package services

import (
	"context"

	"github.com/mshogin/agents/internal/domain/models"
)

// ReasoningService defines the interface for reasoning operations.
// This service is responsible for executing workflow logic and managing
// the reasoning phase of request processing.
//
// Design principles:
// - Single Responsibility: Only handles reasoning logic
// - Dependency Inversion: Defined in domain, implemented in application layer
// - Testable: Easy to mock for unit tests
type ReasoningService interface {
	// ProcessReasoning executes the reasoning workflow for a given input.
	//
	// Parameters:
	//   ctx: Context for cancellation and timeout control
	//   input: The reasoning input containing messages, metadata, workflow selection
	//
	// Returns:
	//   *ReasoningResult: The reasoning output (insights, transformed messages)
	//   error: Any error that occurred during reasoning
	ProcessReasoning(ctx context.Context, input *models.ReasoningInput) (*models.ReasoningResult, error)
}
