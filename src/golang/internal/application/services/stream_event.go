package services

import (
	"encoding/json"

	"github.com/mshogin/agents/internal/domain/models"
)

// StreamEvent represents an event in the streaming pipeline.
// Events are sent from the orchestrator to the HTTP handler for SSE streaming.
type StreamEvent struct {
	Type string      `json:"type"` // "reasoning", "completion", "error", "done"
	Data interface{} `json:"data"`
}

// NewReasoningEvent creates a reasoning event.
func NewReasoningEvent(result *models.ReasoningResult) *StreamEvent {
	return &StreamEvent{
		Type: "reasoning",
		Data: result,
	}
}

// NewCompletionEvent creates a completion chunk event.
func NewCompletionEvent(chunk *models.CompletionChunk) *StreamEvent {
	return &StreamEvent{
		Type: "completion",
		Data: chunk,
	}
}

// NewErrorEvent creates an error event.
func NewErrorEvent(message string) *StreamEvent {
	return &StreamEvent{
		Type: "error",
		Data: map[string]string{
			"message": message,
		},
	}
}

// NewDoneEvent creates a done event (signals end of stream).
func NewDoneEvent() *StreamEvent {
	return &StreamEvent{
		Type: "done",
		Data: map[string]string{
			"status": "complete",
		},
	}
}

// ToJSON converts the event to JSON string.
func (e *StreamEvent) ToJSON() (string, error) {
	data, err := json.Marshal(e)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// ToSSE formats the event as a Server-Sent Event.
func (e *StreamEvent) ToSSE() (string, error) {
	jsonData, err := e.ToJSON()
	if err != nil {
		return "", err
	}
	return "data: " + jsonData + "\n\n", nil
}
