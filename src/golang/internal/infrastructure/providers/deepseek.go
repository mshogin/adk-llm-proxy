package providers

import (
	"context"

	"github.com/mshogin/agents/internal/domain/models"
	"github.com/mshogin/agents/internal/domain/services"
	"github.com/mshogin/agents/internal/infrastructure/config"
)

// DeepSeekProvider implements the LLMProvider interface for DeepSeek API.
// DeepSeek uses OpenAI-compatible API, so we can reuse OpenAI implementation.
type DeepSeekProvider struct {
	openai *OpenAIProvider
}

// NewDeepSeekProvider creates a new DeepSeek provider instance.
func NewDeepSeekProvider(cfg config.ProviderConfig) services.LLMProvider {
	return &DeepSeekProvider{
		openai: NewOpenAIProvider(cfg).(*OpenAIProvider),
	}
}

// Name returns the provider identifier.
func (p *DeepSeekProvider) Name() string {
	return "deepseek"
}

// StreamCompletion delegates to OpenAI implementation (API-compatible).
func (p *DeepSeekProvider) StreamCompletion(ctx context.Context, req *models.CompletionRequest) (<-chan *models.CompletionChunk, error) {
	return p.openai.StreamCompletion(ctx, req)
}

// CheckHealth delegates to OpenAI implementation.
func (p *DeepSeekProvider) CheckHealth(ctx context.Context) error {
	return p.openai.CheckHealth(ctx)
}
