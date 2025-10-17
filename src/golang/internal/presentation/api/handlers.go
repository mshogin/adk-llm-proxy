package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/mshogin/agents/internal/application/services"
	"github.com/mshogin/agents/internal/domain/models"
	"github.com/mshogin/agents/internal/infrastructure/config"
)

// Handler handles HTTP requests for the proxy API.
type Handler struct {
	orchestrator *services.Orchestrator
	config       *config.Config
}

// NewHandler creates a new Handler instance.
func NewHandler(orchestrator *services.Orchestrator, cfg *config.Config) *Handler {
	return &Handler{
		orchestrator: orchestrator,
		config:       cfg,
	}
}

// ChatCompletions handles POST /v1/chat/completions (OpenAI-compatible endpoint).
func (h *Handler) ChatCompletions(w http.ResponseWriter, r *http.Request) {
	// Parse request body
	var req models.CompletionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendErrorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Get workflow from header (or use default)
	workflow := r.Header.Get("X-Workflow")
	if workflow == "" {
		workflow = h.config.Workflows.Default
	}

	// Check if streaming is requested
	if req.Stream {
		h.streamResponse(w, r, &req, workflow)
	} else {
		h.bufferResponse(w, r, &req, workflow)
	}
}

// streamResponse handles streaming SSE responses.
func (h *Handler) streamResponse(w http.ResponseWriter, r *http.Request, req *models.CompletionRequest, workflow string) {
	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	// Get flusher
	flusher, ok := w.(http.Flusher)
	if !ok {
		h.sendErrorResponse(w, http.StatusInternalServerError, "Streaming not supported")
		return
	}

	// Process request
	eventChan, err := h.orchestrator.ProcessRequest(r.Context(), req, workflow)
	if err != nil {
		h.sendErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Stream events
	for event := range eventChan {
		switch event.Type {
		case "reasoning":
			// Send reasoning as visible content
			if result, ok := event.Data.(*models.ReasoningResult); ok {
				reasoningContent := h.formatReasoningResult(result)
				reasoningChunk := &models.CompletionChunk{
					ID:      "reasoning-chunk",
					Object:  "chat.completion.chunk",
					Created: 0,
					Model:   req.Model,
					Choices: []models.ChunkChoice{
						{
							Index: 0,
							Delta: models.ChunkDelta{
								Role:    "assistant",
								Content: reasoningContent,
							},
						},
					},
				}
				chunkJSON, err := json.Marshal(reasoningChunk)
				if err == nil {
					if _, err := w.Write([]byte("data: " + string(chunkJSON) + "\n\n")); err != nil {
						return
					}
				}
			}

		case "completion":
			// Send OpenAI chunk directly (unwrapped for compatibility)
			if chunk, ok := event.Data.(*models.CompletionChunk); ok {
				chunkJSON, err := json.Marshal(chunk)
				if err != nil {
					continue
				}
				if _, err := w.Write([]byte("data: " + string(chunkJSON) + "\n\n")); err != nil {
					return
				}
			}

		case "done":
			// Send OpenAI-compatible done marker
			if _, err := w.Write([]byte("data: [DONE]\n\n")); err != nil {
				return
			}

		case "error":
			// Extract and log error message
			errorMsg := "unknown error"
			if errMap, ok := event.Data.(map[string]string); ok {
				if msg, exists := errMap["message"]; exists {
					errorMsg = msg
				}
			} else if errStr, ok := event.Data.(string); ok {
				errorMsg = errStr
			}
			fmt.Printf("ERROR: %s\n", errorMsg)

			// Send error as SSE comment with actual error message
			if _, err := w.Write([]byte(fmt.Sprintf(": error: %s\n\n", errorMsg))); err != nil {
				return
			}
			return
		}

		// Flush immediately
		flusher.Flush()
	}
}

// bufferResponse handles non-streaming responses (buffers the entire response).
func (h *Handler) bufferResponse(w http.ResponseWriter, r *http.Request, req *models.CompletionRequest, workflow string) {
	// Process request
	eventChan, err := h.orchestrator.ProcessRequest(r.Context(), req, workflow)
	if err != nil {
		h.sendErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Collect all completion chunks
	var fullContent string
	var lastChunk *models.CompletionChunk

	for event := range eventChan {
		if event.Type == "completion" {
			if chunk, ok := event.Data.(*models.CompletionChunk); ok {
				fullContent += chunk.GetContent()
				lastChunk = chunk
			}
		} else if event.Type == "error" {
			errorMsg := "Processing error"
			if errMap, ok := event.Data.(map[string]string); ok {
				if msg, exists := errMap["message"]; exists {
					errorMsg = msg
				}
			} else if errStr, ok := event.Data.(string); ok {
				errorMsg = errStr
			}
			fmt.Printf("ERROR (buffered): %s\n", errorMsg)
			h.sendErrorResponse(w, http.StatusInternalServerError, errorMsg)
			return
		}
	}

	// Build complete response
	response := h.buildCompletionResponse(req, fullContent, lastChunk)

	// Send response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Health handles GET /health endpoint.
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
	})
}

// ListWorkflows handles GET /workflows endpoint.
func (h *Handler) ListWorkflows(w http.ResponseWriter, r *http.Request) {
	workflows := h.orchestrator.GetWorkflows()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"workflows":        workflows,
		"default_workflow": h.config.Workflows.Default,
	})
}

// sendErrorResponse sends an error response.
func (h *Handler) sendErrorResponse(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]string{
		"error": message,
	})
}

// buildCompletionResponse creates a complete CompletionResponse from buffered content.
func (h *Handler) buildCompletionResponse(req *models.CompletionRequest, content string, lastChunk *models.CompletionChunk) *models.CompletionResponse {
	id := "chatcmpl-buffered"
	if lastChunk != nil {
		id = lastChunk.ID
	}

	return &models.CompletionResponse{
		ID:      id,
		Object:  "chat.completion",
		Created: 0, // Would set to actual timestamp
		Model:   req.Model,
		Choices: []models.Choice{
			{
				Index: 0,
				Message: models.Message{
					Role:    "assistant",
					Content: content,
				},
				FinishReason: "stop",
			},
		},
	}
}

// formatReasoningResult formats a ReasoningResult into a string wrapped in <reasoning> tags.
func (h *Handler) formatReasoningResult(result *models.ReasoningResult) string {
	var output string

	output += "<reasoning>\n"
	output += "Workflow: " + result.WorkflowName + "\n"

	// Main message
	if result.Message != "" {
		output += "\n" + result.Message + "\n"
	}

	// Intent (for basic workflow)
	if result.Intent != "" {
		output += "\nIntent: " + result.Intent
		if result.Confidence > 0 {
			output += " (confidence: " + fmt.Sprintf("%.2f", result.Confidence) + ")"
		}
		output += "\n"
	}

	// Agent results (for advanced workflow)
	if len(result.AgentResults) > 0 {
		output += "\nAgent Results:\n"
		for name, agentResult := range result.AgentResults {
			if agentResult.Success {
				output += "- " + name + ": " + agentResult.Output + "\n"
			} else {
				output += "- " + name + ": ERROR - " + agentResult.Error + "\n"
			}
		}
	}

	// Duration
	if result.Duration > 0 {
		output += "\nDuration: " + fmt.Sprintf("%dms", result.Duration) + "\n"
	}

	output += "</reasoning>\n\n"
	return output
}
