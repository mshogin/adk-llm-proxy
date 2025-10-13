package agents

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/mshogin/agents/internal/domain/models"
	"github.com/mshogin/agents/internal/domain/services"
)

// OpenAIAgent executes reasoning using OpenAI's API (native Go implementation).
// This agent makes a quick LLM call to analyze the user's intent.
type OpenAIAgent struct {
	provider services.LLMProvider
	model    string
	timeout  time.Duration
}

// NewOpenAIAgent creates a new OpenAI agent instance.
func NewOpenAIAgent(provider services.LLMProvider, model string, timeout time.Duration) *OpenAIAgent {
	return &OpenAIAgent{
		provider: provider,
		model:    model,
		timeout:  timeout,
	}
}

// Execute runs the OpenAI agent with the given input and returns the result.
func (a *OpenAIAgent) Execute(ctx context.Context, input *models.ReasoningInput) (*models.AgentResult, error) {
	start := time.Now()

	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, a.timeout)
	defer cancel()

	// Create a simple reasoning request
	req := &models.CompletionRequest{
		Model: a.model,
		Messages: []models.Message{
			{
				Role:    "system",
				Content: "You are a helpful assistant that analyzes user requests and provides brief insights.",
			},
			{
				Role:    "user",
				Content: fmt.Sprintf("Analyze this request and provide a brief insight: %s", input.GetUserMessage()),
			},
		},
		Stream: true,
	}

	// Stream completion
	chunkChan, err := a.provider.StreamCompletion(ctx, req)
	if err != nil {
		return &models.AgentResult{
			AgentName: "openai_agent",
			Success:   false,
			Error:     fmt.Sprintf("failed to start streaming: %v", err),
			Duration:  time.Since(start).Milliseconds(),
		}, nil
	}

	// Collect response
	var output strings.Builder
	for chunk := range chunkChan {
		content := chunk.GetContent()
		output.WriteString(content)
	}

	duration := time.Since(start).Milliseconds()

	return &models.AgentResult{
		AgentName: "openai_agent",
		Output:    output.String(),
		Success:   true,
		Duration:  duration,
		Metadata: map[string]interface{}{
			"model": a.model,
		},
	}, nil
}
