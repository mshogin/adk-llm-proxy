package api

import (
	"encoding/json"
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
		// Convert event to SSE format
		sseData, err := event.ToSSE()
		if err != nil {
			continue
		}

		// Write event
		if _, err := w.Write([]byte(sseData)); err != nil {
			return // Client disconnected
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
			h.sendErrorResponse(w, http.StatusInternalServerError, "Processing error")
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
