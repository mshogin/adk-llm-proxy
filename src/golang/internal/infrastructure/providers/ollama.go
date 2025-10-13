package providers

import (
	"context"

	"github.com/mshogin/agents/internal/domain/models"
	"github.com/mshogin/agents/internal/domain/services"
	"github.com/mshogin/agents/internal/infrastructure/config"
)

// OllamaProvider implements the LLMProvider interface for Ollama API.
// Ollama uses OpenAI-compatible API, so we can reuse OpenAI implementation.
type OllamaProvider struct {
	openai *OpenAIProvider
}

// NewOllamaProvider creates a new Ollama provider instance.
func NewOllamaProvider(cfg config.ProviderConfig) services.LLMProvider {
	return &OllamaProvider{
		openai: NewOpenAIProvider(cfg).(*OpenAIProvider),
	}
}

// Name returns the provider identifier.
func (p *OllamaProvider) Name() string {
	return "ollama"
}

// StreamCompletion delegates to OpenAI implementation (API-compatible).
func (p *OllamaProvider) StreamCompletion(ctx context.Context, req *models.CompletionRequest) (<-chan *models.CompletionChunk, error) {
	return p.openai.StreamCompletion(ctx, req)
}

// CheckHealth delegates to OpenAI implementation.
func (p *OllamaProvider) CheckHealth(ctx context.Context) error {
	return p.openai.CheckHealth(ctx)
}
