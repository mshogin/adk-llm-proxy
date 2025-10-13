package models

// Message represents a single message in a chat completion request.
// This structure follows the OpenAI API schema for maximum compatibility.
type Message struct {
	Role    string `json:"role"`              // "system", "user", "assistant", "function"
	Content string `json:"content"`           // Message content
	Name    string `json:"name,omitempty"`    // Optional name for function/tool messages
}

// CompletionRequest represents a request for chat completion.
// This structure is compatible with OpenAI's /v1/chat/completions API.
//
// Design principles:
// - OpenAI API compatibility (works with gptel, curl, SDKs)
// - Immutable where possible (use pointers for optional fields)
// - JSON serialization ready
type CompletionRequest struct {
	// Model is the LLM model identifier (e.g., "gpt-4o-mini", "claude-3-5-sonnet")
	Model string `json:"model"`

	// Messages is the list of conversation messages
	Messages []Message `json:"messages"`

	// Stream controls whether to stream the response (SSE) or buffer it
	Stream bool `json:"stream,omitempty"`

	// Temperature controls randomness (0.0 = deterministic, 2.0 = very random)
	Temperature *float64 `json:"temperature,omitempty"`

	// MaxTokens limits the response length
	MaxTokens *int `json:"max_tokens,omitempty"`

	// TopP controls nucleus sampling (alternative to temperature)
	TopP *float64 `json:"top_p,omitempty"`

	// FrequencyPenalty reduces repetition (-2.0 to 2.0)
	FrequencyPenalty *float64 `json:"frequency_penalty,omitempty"`

	// PresencePenalty encourages new topics (-2.0 to 2.0)
	PresencePenalty *float64 `json:"presence_penalty,omitempty"`

	// Stop sequences that will halt generation
	Stop []string `json:"stop,omitempty"`

	// User identifier for tracking and abuse prevention
	User string `json:"user,omitempty"`
}

// Validate checks if the request is valid.
func (r *CompletionRequest) Validate() error {
	if r.Model == "" {
		return ErrMissingModel
	}
	if len(r.Messages) == 0 {
		return ErrEmptyMessages
	}
	return nil
}

// GetTemperature returns the temperature value or default (1.0).
func (r *CompletionRequest) GetTemperature() float64 {
	if r.Temperature == nil {
		return 1.0
	}
	return *r.Temperature
}

// GetMaxTokens returns the max tokens value or default (0 = unlimited).
func (r *CompletionRequest) GetMaxTokens() int {
	if r.MaxTokens == nil {
		return 0
	}
	return *r.MaxTokens
}
