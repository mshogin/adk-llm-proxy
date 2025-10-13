package workflows

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/mshogin/agents/internal/domain/models"
	"github.com/mshogin/agents/internal/domain/services"
)

// BasicWorkflow implements intent detection via regex and keywords.
// This workflow doesn't make any LLM calls - it uses simple pattern matching.
type BasicWorkflow struct {
	intents []intentPattern
}

// intentPattern represents an intent detection pattern.
type intentPattern struct {
	name       string
	keywords   []string
	regex      *regexp.Regexp
	confidence float64
}

// NewBasicWorkflow creates a new BasicWorkflow instance.
func NewBasicWorkflow() services.Workflow {
	return &BasicWorkflow{
		intents: []intentPattern{
			{
				name:       "question",
				keywords:   []string{"what", "why", "how", "when", "where", "who", "?"},
				regex:      regexp.MustCompile(`(?i)^(what|why|how|when|where|who|is|are|can|do|does)\b.*\?`),
				confidence: 0.9,
			},
			{
				name:       "greeting",
				keywords:   []string{"hello", "hi", "hey", "greetings"},
				regex:      regexp.MustCompile(`(?i)^(hello|hi|hey|greetings)\b`),
				confidence: 0.95,
			},
			{
				name:       "code_request",
				keywords:   []string{"code", "implement", "write", "create", "function", "class"},
				regex:      regexp.MustCompile(`(?i)\b(code|implement|write|create|function|class|method)\b`),
				confidence: 0.85,
			},
			{
				name:       "explanation",
				keywords:   []string{"explain", "describe", "tell me about", "what is"},
				regex:      regexp.MustCompile(`(?i)\b(explain|describe|tell me about|what is)\b`),
				confidence: 0.85,
			},
			{
				name:       "debug",
				keywords:   []string{"debug", "fix", "error", "bug", "issue", "problem"},
				regex:      regexp.MustCompile(`(?i)\b(debug|fix|error|bug|issue|problem|not working)\b`),
				confidence: 0.8,
			},
		},
	}
}

// Name returns the workflow identifier.
func (w *BasicWorkflow) Name() string {
	return "basic"
}

// Execute processes the reasoning input and detects intent.
func (w *BasicWorkflow) Execute(ctx context.Context, input *models.ReasoningInput) (*models.ReasoningResult, error) {
	// Check context cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Get user message
	userMessage := input.GetUserMessage()
	if userMessage == "" {
		result := models.NewReasoningResult("basic", "No user message found")
		result.Intent = "unknown"
		result.Confidence = 0.0
		return result, nil
	}

	// Detect intent
	intent, confidence := w.detectIntent(userMessage)

	// Create result
	result := models.NewReasoningResult("basic", fmt.Sprintf("Detected intent: %s (confidence: %.2f)", intent, confidence))
	result.Intent = intent
	result.Confidence = confidence

	// Add metadata
	if result.Metadata == nil {
		result.Metadata = make(map[string]interface{})
	}
	result.Metadata["user_message"] = userMessage
	result.Metadata["message_length"] = len(userMessage)

	return result, nil
}

// detectIntent analyzes the message and returns the detected intent and confidence.
func (w *BasicWorkflow) detectIntent(message string) (string, float64) {
	messageLower := strings.ToLower(message)
	bestIntent := "general"
	bestConfidence := 0.5

	// Check each intent pattern
	for _, pattern := range w.intents {
		score := 0.0

		// Check regex match
		if pattern.regex.MatchString(message) {
			score = pattern.confidence
		} else {
			// Check keyword match
			matchCount := 0
			for _, keyword := range pattern.keywords {
				if strings.Contains(messageLower, keyword) {
					matchCount++
				}
			}
			if matchCount > 0 {
				score = pattern.confidence * float64(matchCount) / float64(len(pattern.keywords))
			}
		}

		// Update best match
		if score > bestConfidence {
			bestIntent = pattern.name
			bestConfidence = score
		}
	}

	return bestIntent, bestConfidence
}
