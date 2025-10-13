package models

// CompletionChunk represents a single chunk in a streaming completion response.
// This structure follows the OpenAI streaming response format (SSE).
//
// Example SSE event:
// data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1234567890,"model":"gpt-4o-mini","choices":[{"index":0,"delta":{"content":"Hello"},"finish_reason":null}]}
type CompletionChunk struct {
	// ID is a unique identifier for this completion
	ID string `json:"id"`

	// Object is always "chat.completion.chunk" for streaming
	Object string `json:"object"`

	// Created is the Unix timestamp of when the completion was created
	Created int64 `json:"created"`

	// Model is the model used for this completion
	Model string `json:"model"`

	// Choices contains the completion choices (usually just one for streaming)
	Choices []ChunkChoice `json:"choices"`
}

// ChunkChoice represents a single choice in a streaming completion.
type ChunkChoice struct {
	// Index is the choice index (usually 0)
	Index int `json:"index"`

	// Delta contains the incremental content for this chunk
	Delta ChunkDelta `json:"delta"`

	// FinishReason indicates why generation stopped (null during streaming)
	// Possible values: "stop", "length", "function_call", "content_filter", null
	FinishReason *string `json:"finish_reason"`
}

// ChunkDelta represents the incremental content in a streaming chunk.
type ChunkDelta struct {
	// Role is only present in the first chunk ("assistant")
	Role string `json:"role,omitempty"`

	// Content is the incremental text content
	Content string `json:"content,omitempty"`
}

// CompletionResponse represents a non-streaming completion response.
// This structure follows the OpenAI chat completion response format.
type CompletionResponse struct {
	// ID is a unique identifier for this completion
	ID string `json:"id"`

	// Object is always "chat.completion" for non-streaming
	Object string `json:"object"`

	// Created is the Unix timestamp of when the completion was created
	Created int64 `json:"created"`

	// Model is the model used for this completion
	Model string `json:"model"`

	// Choices contains the completion choices
	Choices []Choice `json:"choices"`

	// Usage contains token usage statistics
	Usage *Usage `json:"usage,omitempty"`
}

// Choice represents a single choice in a non-streaming completion.
type Choice struct {
	// Index is the choice index
	Index int `json:"index"`

	// Message contains the complete assistant message
	Message Message `json:"message"`

	// FinishReason indicates why generation stopped
	FinishReason string `json:"finish_reason"`
}

// Usage contains token usage statistics.
type Usage struct {
	// PromptTokens is the number of tokens in the prompt
	PromptTokens int `json:"prompt_tokens"`

	// CompletionTokens is the number of tokens in the completion
	CompletionTokens int `json:"completion_tokens"`

	// TotalTokens is the sum of prompt and completion tokens
	TotalTokens int `json:"total_tokens"`
}

// NewCompletionChunk creates a new completion chunk with defaults.
func NewCompletionChunk(id, model, content string, finishReason *string) *CompletionChunk {
	return &CompletionChunk{
		ID:      id,
		Object:  "chat.completion.chunk",
		Created: 0, // Will be set by provider
		Model:   model,
		Choices: []ChunkChoice{
			{
				Index: 0,
				Delta: ChunkDelta{
					Content: content,
				},
				FinishReason: finishReason,
			},
		},
	}
}

// IsComplete returns true if this chunk indicates the stream is complete.
func (c *CompletionChunk) IsComplete() bool {
	if len(c.Choices) == 0 {
		return false
	}
	return c.Choices[0].FinishReason != nil
}

// GetContent extracts the content from the first choice's delta.
func (c *CompletionChunk) GetContent() string {
	if len(c.Choices) == 0 {
		return ""
	}
	return c.Choices[0].Delta.Content
}
