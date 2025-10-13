package providers

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/mshogin/agents/internal/domain/models"
	"github.com/mshogin/agents/internal/domain/services"
	"github.com/mshogin/agents/internal/infrastructure/config"
)

// AnthropicProvider implements the LLMProvider interface for Anthropic API.
// Converts between OpenAI format and Anthropic format.
type AnthropicProvider struct {
	config     config.ProviderConfig
	httpClient *http.Client
}

// NewAnthropicProvider creates a new Anthropic provider instance.
func NewAnthropicProvider(cfg config.ProviderConfig) services.LLMProvider {
	return &AnthropicProvider{
		config: cfg,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
			},
		},
	}
}

// Name returns the provider identifier.
func (p *AnthropicProvider) Name() string {
	return "anthropic"
}

// StreamCompletion sends a streaming completion request to Anthropic API.
// Converts OpenAI format to Anthropic format and back.
func (p *AnthropicProvider) StreamCompletion(ctx context.Context, req *models.CompletionRequest) (<-chan *models.CompletionChunk, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	// Convert OpenAI format to Anthropic format
	anthropicReq := p.convertToAnthropicFormat(req)

	// Create HTTP request
	body, err := json.Marshal(anthropicReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.config.BaseURL+"/messages", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	// Set Anthropic-specific headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", p.config.APIKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	// Send request
	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("Anthropic API error (status %d): %s", resp.StatusCode, string(body))
	}

	// Create chunk channel
	chunkChan := make(chan *models.CompletionChunk, 10)

	// Start streaming goroutine
	go p.streamResponse(ctx, resp, chunkChan, req.Model)

	return chunkChan, nil
}

// CheckHealth verifies the provider is operational.
func (p *AnthropicProvider) CheckHealth(ctx context.Context) error {
	// Anthropic doesn't have a dedicated health endpoint, so we skip this
	// In a real implementation, you might try a minimal request
	return nil
}

// convertToAnthropicFormat converts OpenAI request format to Anthropic format.
func (p *AnthropicProvider) convertToAnthropicFormat(req *models.CompletionRequest) map[string]interface{} {
	anthropicReq := map[string]interface{}{
		"model":  req.Model,
		"stream": true,
	}

	// Convert messages
	messages := make([]map[string]string, 0)
	for _, msg := range req.Messages {
		if msg.Role != "system" { // Anthropic handles system messages separately
			messages = append(messages, map[string]string{
				"role":    msg.Role,
				"content": msg.Content,
			})
		}
	}
	anthropicReq["messages"] = messages

	// Add system message if present
	for _, msg := range req.Messages {
		if msg.Role == "system" {
			anthropicReq["system"] = msg.Content
			break
		}
	}

	// Add optional parameters
	if req.MaxTokens != nil {
		anthropicReq["max_tokens"] = *req.MaxTokens
	} else {
		anthropicReq["max_tokens"] = 4096 // Anthropic requires max_tokens
	}

	if req.Temperature != nil {
		anthropicReq["temperature"] = *req.Temperature
	}

	return anthropicReq
}

// streamResponse reads SSE events from Anthropic API and converts to OpenAI format.
func (p *AnthropicProvider) streamResponse(ctx context.Context, resp *http.Response, chunkChan chan<- *models.CompletionChunk, model string) {
	defer close(chunkChan)
	defer resp.Body.Close()

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return
		default:
		}

		line := scanner.Text()

		if line == "" || strings.HasPrefix(line, ":") {
			continue
		}

		// Parse SSE event
		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")

			// Parse Anthropic event
			var event map[string]interface{}
			if err := json.Unmarshal([]byte(data), &event); err != nil {
				continue
			}

			// Convert to OpenAI format
			chunk := p.convertToOpenAIFormat(event, model)
			if chunk != nil {
				select {
				case chunkChan <- chunk:
				case <-ctx.Done():
					return
				}
			}

			// Check for stream end
			if eventType, ok := event["type"].(string); ok && eventType == "message_stop" {
				return
			}
		}
	}
}

// convertToOpenAIFormat converts Anthropic event to OpenAI chunk format.
func (p *AnthropicProvider) convertToOpenAIFormat(event map[string]interface{}, model string) *models.CompletionChunk {
	eventType, _ := event["type"].(string)

	chunk := &models.CompletionChunk{
		ID:      "anthropic-" + fmt.Sprint(time.Now().Unix()),
		Object:  "chat.completion.chunk",
		Created: time.Now().Unix(),
		Model:   model,
		Choices: []models.ChunkChoice{
			{
				Index: 0,
				Delta: models.ChunkDelta{},
			},
		},
	}

	switch eventType {
	case "content_block_delta":
		if delta, ok := event["delta"].(map[string]interface{}); ok {
			if text, ok := delta["text"].(string); ok {
				chunk.Choices[0].Delta.Content = text
			}
		}

	case "message_stop":
		finishReason := "stop"
		chunk.Choices[0].FinishReason = &finishReason

	default:
		return nil // Skip other event types
	}

	return chunk
}
