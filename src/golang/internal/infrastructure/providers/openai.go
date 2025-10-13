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

// OpenAIProvider implements the LLMProvider interface for OpenAI API.
// Supports streaming and non-streaming completions.
type OpenAIProvider struct {
	config     config.ProviderConfig
	httpClient *http.Client
}

// NewOpenAIProvider creates a new OpenAI provider instance.
func NewOpenAIProvider(cfg config.ProviderConfig) services.LLMProvider {
	return &OpenAIProvider{
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
func (p *OpenAIProvider) Name() string {
	return "openai"
}

// StreamCompletion sends a streaming completion request to OpenAI API.
// Returns a channel that streams completion chunks as they arrive.
func (p *OpenAIProvider) StreamCompletion(ctx context.Context, req *models.CompletionRequest) (<-chan *models.CompletionChunk, error) {
	// Validate request
	if err := req.Validate(); err != nil {
		return nil, err
	}

	// Create HTTP request
	httpReq, err := p.createHTTPRequest(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Send request
	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}

	// Check status code
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("OpenAI API error (status %d): %s", resp.StatusCode, string(body))
	}

	// Create chunk channel
	chunkChan := make(chan *models.CompletionChunk, 10)

	// Start streaming goroutine
	go p.streamResponse(ctx, resp, chunkChan)

	return chunkChan, nil
}

// CheckHealth verifies the provider is operational.
func (p *OpenAIProvider) CheckHealth(ctx context.Context) error {
	// Simple health check - try to create a request
	req, err := http.NewRequestWithContext(ctx, "GET", p.config.BaseURL+"/models", nil)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+p.config.APIKey)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check failed with status %d", resp.StatusCode)
	}

	return nil
}

// createHTTPRequest creates an HTTP request for the OpenAI API.
func (p *OpenAIProvider) createHTTPRequest(ctx context.Context, req *models.CompletionRequest) (*http.Request, error) {
	// Ensure streaming is enabled
	req.Stream = true

	// Marshal request body
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.config.BaseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.config.APIKey)

	return httpReq, nil
}

// streamResponse reads SSE events from the response and sends chunks to the channel.
func (p *OpenAIProvider) streamResponse(ctx context.Context, resp *http.Response, chunkChan chan<- *models.CompletionChunk) {
	defer close(chunkChan)
	defer resp.Body.Close()

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024) // Increase buffer size for large chunks

	for scanner.Scan() {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return
		default:
		}

		line := scanner.Text()

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, ":") {
			continue
		}

		// Parse SSE event
		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")

			// Check for stream end
			if data == "[DONE]" {
				return
			}

			// Parse JSON chunk
			var chunk models.CompletionChunk
			if err := json.Unmarshal([]byte(data), &chunk); err != nil {
				// Send error chunk (optional - could also just log and continue)
				continue
			}

			// Send chunk to channel
			select {
			case chunkChan <- &chunk:
			case <-ctx.Done():
				return
			}
		}
	}

	if err := scanner.Err(); err != nil {
		// Could send error through a separate error channel if needed
		return
	}
}
