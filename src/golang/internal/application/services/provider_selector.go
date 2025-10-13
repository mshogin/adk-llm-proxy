package services

import (
	"fmt"
	"strings"

	"github.com/mshogin/agents/internal/domain/models"
	domainServices "github.com/mshogin/agents/internal/domain/services"
)

// ProviderSelector selects the appropriate LLM provider based on the model name.
type ProviderSelector struct {
	providers map[string]domainServices.LLMProvider
}

// NewProviderSelector creates a new ProviderSelector instance.
func NewProviderSelector(providers map[string]domainServices.LLMProvider) *ProviderSelector {
	return &ProviderSelector{
		providers: providers,
	}
}

// SelectProvider returns the provider for the given model.
func (s *ProviderSelector) SelectProvider(model string) (domainServices.LLMProvider, error) {
	providerName := s.detectProvider(model)

	provider, ok := s.providers[providerName]
	if !ok {
		return nil, fmt.Errorf("%w: %s", models.ErrProviderNotFound, providerName)
	}

	return provider, nil
}

// detectProvider determines which provider to use based on the model name.
func (s *ProviderSelector) detectProvider(model string) string {
	modelLower := strings.ToLower(model)

	// OpenAI models
	if strings.HasPrefix(modelLower, "gpt-") || strings.HasPrefix(modelLower, "o1-") {
		return "openai"
	}

	// Anthropic models
	if strings.HasPrefix(modelLower, "claude-") {
		return "anthropic"
	}

	// DeepSeek models
	if strings.HasPrefix(modelLower, "deepseek-") {
		return "deepseek"
	}

	// Default to ollama for unknown models (local models)
	return "ollama"
}

// GetAvailableProviders returns the list of available provider names.
func (s *ProviderSelector) GetAvailableProviders() []string {
	providers := make([]string, 0, len(s.providers))
	for name := range s.providers {
		providers = append(providers, name)
	}
	return providers
}
