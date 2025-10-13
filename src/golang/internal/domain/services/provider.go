package services

import (
	"context"

	"github.com/mshogin/agents/internal/domain/models"
)

// LLMProvider defines the interface for all LLM provider implementations.
// This interface follows the Dependency Inversion Principle (DIP) - it's defined
// in the domain layer and implemented in the infrastructure layer.
//
// Key design principles:
// - Small, focused interface (Interface Segregation Principle)
// - Easy to mock for testing
// - Provider-agnostic (supports OpenAI, Anthropic, DeepSeek, Ollama, etc.)
type LLMProvider interface {
	// Name returns the provider's identifier (e.g., "openai", "anthropic")
	Name() string

	// StreamCompletion sends a completion request and returns a channel of streaming chunks.
	// The channel will be closed when the stream is complete or an error occurs.
	//
	// Parameters:
	//   ctx: Context for cancellation and timeout control
	//   req: The completion request containing messages, model, parameters, etc.
	//
	// Returns:
	//   <-chan: Receive-only channel of completion chunks (streamed response)
	//   error: Any error that occurred during request initialization
	//
	// Usage:
	//   chunkChan, err := provider.StreamCompletion(ctx, req)
	//   if err != nil { return err }
	//   for chunk := range chunkChan {
	//       // Process each chunk
	//   }
	StreamCompletion(ctx context.Context, req *models.CompletionRequest) (<-chan *models.CompletionChunk, error)

	// CheckHealth verifies the provider is operational and credentials are valid.
	// Returns nil if healthy, error otherwise.
	CheckHealth(ctx context.Context) error
}
