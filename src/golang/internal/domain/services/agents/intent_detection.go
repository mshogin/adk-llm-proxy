package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/mshogin/agents/internal/application/services"
	"github.com/mshogin/agents/internal/domain/models"
	domainservices "github.com/mshogin/agents/internal/domain/services"
)

// IntentDetectionAgent detects user intent and extracts entities from input messages.
//
// Design Principles:
// - Rule-based classification for speed and determinism
// - Entity extraction for structured data (projects, dates, providers, statuses)
// - Confidence scoring for reliability tracking
// - Optional LLM fallback for low-confidence cases (< 0.8)
//
// Supported Intent Types:
// - query_commits: Ask about commits, code changes, or version history
// - query_issues: Ask about tasks, bugs, or issue tracking
// - query_analytics: Request statistics, metrics, or trends
// - query_status: Check project status, health, or progress
// - command_action: Execute an action (deploy, restart, update)
// - request_help: Ask for help or documentation
// - conversation: General conversation or unclear intent
//
// Extracted Entities:
// - projects: Project names (gitlab-repo, youtrack-project)
// - dates: Time references (today, last week, 2024-01-15)
// - providers: Service providers (gitlab, youtrack, openai)
// - statuses: Status indicators (open, closed, in-progress, failed)
type IntentDetectionAgent struct {
	// Agent configuration
	id                  string
	confidenceThreshold float64 // Threshold for LLM fallback (default 0.8)

	// Intent patterns (compiled at initialization)
	intentPatterns map[string]*intentPattern

	// LLM orchestrator for fallback (optional)
	llmOrchestrator *services.LLMOrchestrator
}

// intentPattern defines a pattern for detecting an intent.
type intentPattern struct {
	intentType string
	keywords   []string         // Keywords that indicate this intent
	patterns   []*regexp.Regexp // Regex patterns for matching
	weight     float64          // Weight for confidence calculation (0.0-1.0)
}

// NewIntentDetectionAgent creates a new intent detection agent.
// orchestrator is optional - if nil, only rule-based detection is used.
func NewIntentDetectionAgent(orchestrator *services.LLMOrchestrator) *IntentDetectionAgent {
	agent := &IntentDetectionAgent{
		id:                  "intent_detection",
		confidenceThreshold: 0.8,
		intentPatterns:      make(map[string]*intentPattern),
		llmOrchestrator:     orchestrator,
	}

	agent.initializePatterns()
	return agent
}

// AgentID returns the unique identifier for this agent.
func (a *IntentDetectionAgent) AgentID() string {
	return a.id
}

// Preconditions returns the list of context keys required before execution.
// Intent detection is the first agent, so it has no preconditions.
func (a *IntentDetectionAgent) Preconditions() []string {
	return []string{} // First agent - no preconditions
}

// Postconditions returns the list of context keys guaranteed after execution.
func (a *IntentDetectionAgent) Postconditions() []string {
	return []string{
		"reasoning.intents",
		"reasoning.entities",
		"reasoning.confidence_scores",
	}
}

// Execute runs intent detection on the input context.
func (a *IntentDetectionAgent) Execute(ctx context.Context, agentContext *models.AgentContext) (*models.AgentContext, error) {
	startTime := time.Now()

	// Clone context to avoid modifying input
	newContext, err := agentContext.Clone()
	if err != nil {
		return nil, fmt.Errorf("failed to clone context: %w", err)
	}

	// Initialize reasoning context if needed
	if newContext.Reasoning == nil {
		newContext.Reasoning = &models.ReasoningContext{}
	}

	// Extract input text from metadata or use placeholder
	inputText := a.extractInputText(newContext)
	if inputText == "" {
		return nil, fmt.Errorf("no input text found in context")
	}

	// Store detailed agent trace in metadata
	agentTrace := map[string]interface{}{
		"agent_id": a.id,
		"input_received": inputText,
		"input_length": len(inputText),
	}

	// Detect intents using rule-based classification
	intents := a.detectIntents(inputText)
	agentTrace["rule_based_intents"] = intents

	// Extract entities from input text
	entities := a.extractEntities(inputText)
	agentTrace["extracted_entities"] = entities

	// Calculate confidence scores
	confidenceScores := a.calculateConfidenceScores(intents)
	agentTrace["confidence_scores"] = confidenceScores

	// LLM fallback for low-confidence cases
	llmCalls := 0
	llmPrompt := ""
	llmResponse := ""
	if a.shouldUseLLMFallback(confidenceScores) {
		// Build and store LLM prompt
		llmPrompt = a.buildIntentDetectionPrompt(inputText)
		agentTrace["llm_fallback_triggered"] = true
		agentTrace["llm_prompt"] = llmPrompt
		agentTrace["llm_trigger_reason"] = fmt.Sprintf("Primary confidence %.2f below threshold %.2f",
			confidenceScores["primary_intent"], a.confidenceThreshold)

		llmIntents, llmConfidence, calls, err := a.detectIntentsWithLLMDetailed(ctx, inputText, &llmResponse)
		llmCalls = calls
		agentTrace["llm_calls_made"] = llmCalls
		agentTrace["llm_response"] = llmResponse

		if err == nil && llmConfidence > confidenceScores["primary_intent"] {
			// LLM provided better confidence, use LLM results
			agentTrace["llm_result_used"] = true
			agentTrace["llm_intents"] = llmIntents
			agentTrace["llm_confidence"] = llmConfidence
			intents = llmIntents
			confidenceScores = a.calculateConfidenceScores(intents)
			confidenceScores["llm_used"] = 1.0
			confidenceScores["llm_confidence"] = llmConfidence
		} else {
			agentTrace["llm_result_used"] = false
			if err != nil {
				agentTrace["llm_error"] = err.Error()
			}
		}
	} else {
		agentTrace["llm_fallback_triggered"] = false
	}

	// Generate clarification questions for ambiguous intents
	clarificationQuestions := a.generateClarificationQuestions(intents, confidenceScores)
	agentTrace["clarification_questions"] = clarificationQuestions

	// Store final results in trace
	agentTrace["final_intents"] = intents
	agentTrace["final_entities"] = entities
	agentTrace["final_confidence_scores"] = confidenceScores

	// Write results to context
	newContext.Reasoning.Intents = intents
	newContext.Reasoning.Entities = entities
	newContext.Reasoning.ConfidenceScores = confidenceScores
	newContext.Reasoning.ClarificationQuestions = clarificationQuestions

	// Store agent trace in LLM cache
	if newContext.LLM == nil {
		newContext.LLM = &models.LLMContext{
			Cache: make(map[string]interface{}),
		}
	}
	if newContext.LLM.Cache == nil {
		newContext.LLM.Cache = make(map[string]interface{})
	}
	if traces, ok := newContext.LLM.Cache["agent_traces"].([]interface{}); ok {
		newContext.LLM.Cache["agent_traces"] = append(traces, agentTrace)
	} else {
		newContext.LLM.Cache["agent_traces"] = []interface{}{agentTrace}
	}

	// Track agent execution in audit
	duration := time.Since(startTime)
	a.recordAgentRun(newContext, duration, "success", nil, llmCalls)

	return newContext, nil
}

// initializePatterns sets up all intent detection patterns.
func (a *IntentDetectionAgent) initializePatterns() {
	// Query Commits Intent
	a.intentPatterns["query_commits"] = &intentPattern{
		intentType: "query_commits",
		keywords: []string{
			"commit", "commits", "changes", "changelog", "version history",
			"git", "repository", "code change", "pushed", "merged",
		},
		patterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)\b(latest|recent|last)\s+(commit|change)s?\b`),
			regexp.MustCompile(`(?i)\bwhat('s| is| are)?\s+(changed|new|updated)\b`),
			regexp.MustCompile(`(?i)\b(show|list|get)\s+commit`),
		},
		weight: 0.9,
	}

	// Query Issues Intent
	a.intentPatterns["query_issues"] = &intentPattern{
		intentType: "query_issues",
		keywords: []string{
			"issue", "issues", "task", "tasks", "bug", "bugs",
			"ticket", "tickets", "story", "stories", "epic",
		},
		patterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)\b(open|active|pending|closed)\s+(issue|task|bug)s?\b`),
			regexp.MustCompile(`(?i)\b(show|list|get)\s+(issue|task|bug)s?\b`),
			regexp.MustCompile(`(?i)\bwhat('s| is| are)?\s+(my|our)?\s+(task|issue|bug)s?\b`),
		},
		weight: 0.9,
	}

	// Query Analytics Intent
	a.intentPatterns["query_analytics"] = &intentPattern{
		intentType: "query_analytics",
		keywords: []string{
			"statistics", "stats", "metrics", "analytics", "report",
			"trend", "trends", "performance", "count", "total",
		},
		patterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)\b(how many|count|total|number of)\b`),
			regexp.MustCompile(`(?i)\b(statistics|stats|metrics|analytics)\b`),
			regexp.MustCompile(`(?i)\b(trend|pattern|over time)\b`),
		},
		weight: 0.85,
	}

	// Query Status Intent
	a.intentPatterns["query_status"] = &intentPattern{
		intentType: "query_status",
		keywords: []string{
			"status", "health", "progress", "state", "condition",
			"running", "stopped", "failed", "ok", "ready",
		},
		patterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)\bwhat('s| is)?\s+the\s+status\b`),
			regexp.MustCompile(`(?i)\bis\s+(it|everything|system)\s+(ok|running|working|healthy)\b`),
			regexp.MustCompile(`(?i)\bcheck\s+status\b`),
		},
		weight: 0.85,
	}

	// Command Action Intent
	a.intentPatterns["command_action"] = &intentPattern{
		intentType: "command_action",
		keywords: []string{
			"deploy", "restart", "start", "stop", "update", "upgrade",
			"create", "delete", "modify", "execute", "run",
		},
		patterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)\b(please|can you|could you)?\s*(deploy|restart|start|stop|update)\b`),
			regexp.MustCompile(`(?i)\b(create|delete|modify|execute|run)\s+`),
		},
		weight: 0.9,
	}

	// Request Help Intent
	a.intentPatterns["request_help"] = &intentPattern{
		intentType: "request_help",
		keywords: []string{
			"help", "how", "what", "explain", "documentation",
			"guide", "tutorial", "instructions", "support",
		},
		patterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)\b(help|how do|how to|can you help)\b`),
			regexp.MustCompile(`(?i)\b(what is|what are|explain|tell me about)\b`),
			regexp.MustCompile(`(?i)\b(documentation|guide|tutorial|instructions)\b`),
		},
		weight: 0.8,
	}

	// Conversation Intent (fallback)
	a.intentPatterns["conversation"] = &intentPattern{
		intentType: "conversation",
		keywords: []string{
			"hello", "hi", "thanks", "thank you", "bye", "goodbye",
		},
		patterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)\b(hello|hi|hey|greetings)\b`),
			regexp.MustCompile(`(?i)\b(thanks|thank you|bye|goodbye)\b`),
		},
		weight: 0.6,
	}
}

// detectIntents analyzes input text and returns detected intents with confidence scores.
func (a *IntentDetectionAgent) detectIntents(inputText string) []models.Intent {
	lowerInput := strings.ToLower(inputText)
	intentScores := make(map[string]float64)

	// Check each intent pattern
	for intentType, pattern := range a.intentPatterns {
		score := 0.0

		// Check keywords
		keywordMatches := 0
		for _, keyword := range pattern.keywords {
			if strings.Contains(lowerInput, keyword) {
				keywordMatches++
			}
		}
		if keywordMatches > 0 {
			score += float64(keywordMatches) * 0.3 // Each keyword adds 0.3
		}

		// Check regex patterns
		patternMatches := 0
		for _, regex := range pattern.patterns {
			if regex.MatchString(inputText) {
				patternMatches++
			}
		}
		if patternMatches > 0 {
			score += float64(patternMatches) * 0.5 // Each pattern match adds 0.5
		}

		// Apply pattern weight
		score *= pattern.weight

		// Normalize to 0.0-1.0 range
		if score > 1.0 {
			score = 1.0
		}

		if score > 0.0 {
			intentScores[intentType] = score
		}
	}

	// If no intents detected, default to conversation
	if len(intentScores) == 0 {
		intentScores["conversation"] = 0.5
	}

	// Convert to Intent objects, sorted by confidence
	intents := make([]models.Intent, 0, len(intentScores))
	for intentType, confidence := range intentScores {
		intents = append(intents, models.Intent{
			Type:       intentType,
			Confidence: confidence,
		})
	}

	// Sort by confidence (highest first)
	for i := 0; i < len(intents)-1; i++ {
		for j := i + 1; j < len(intents); j++ {
			if intents[j].Confidence > intents[i].Confidence {
				intents[i], intents[j] = intents[j], intents[i]
			}
		}
	}

	return intents
}

// extractEntities extracts structured entities from input text.
func (a *IntentDetectionAgent) extractEntities(inputText string) map[string]interface{} {
	entities := make(map[string]interface{})

	// Extract projects
	projects := a.extractProjects(inputText)
	if len(projects) > 0 {
		entities["projects"] = projects
	}

	// Extract dates
	dates := a.extractDates(inputText)
	if len(dates) > 0 {
		entities["dates"] = dates
	}

	// Extract providers
	providers := a.extractProviders(inputText)
	if len(providers) > 0 {
		entities["providers"] = providers
	}

	// Extract statuses
	statuses := a.extractStatuses(inputText)
	if len(statuses) > 0 {
		entities["statuses"] = statuses
	}

	return entities
}

// extractProjects extracts project names from input text.
func (a *IntentDetectionAgent) extractProjects(inputText string) []string {
	projects := []string{}
	lowerInput := strings.ToLower(inputText)

	// Common project name patterns
	projectPatterns := []string{
		"project", "repo", "repository", "module", "service",
	}

	for _, pattern := range projectPatterns {
		regex := regexp.MustCompile(fmt.Sprintf(`(?i)\b%s[:\s]+([a-zA-Z0-9\-_]+)\b`, pattern))
		matches := regex.FindAllStringSubmatch(inputText, -1)
		for _, match := range matches {
			if len(match) > 1 {
				projects = append(projects, match[1])
			}
		}
	}

	// Also check for explicit project names (gitlab-, youtrack-, etc.)
	explicitRegex := regexp.MustCompile(`\b([a-zA-Z0-9\-_]*(?:gitlab|youtrack|mcp)[a-zA-Z0-9\-_]*)\b`)
	matches := explicitRegex.FindAllString(lowerInput, -1)
	projects = append(projects, matches...)

	return a.deduplicateStrings(projects)
}

// extractDates extracts date references from input text.
func (a *IntentDetectionAgent) extractDates(inputText string) []string {
	dates := []string{}
	lowerInput := strings.ToLower(inputText)

	// Relative date keywords
	relativeDates := []string{
		"today", "yesterday", "tomorrow",
		"last week", "this week", "next week",
		"last month", "this month", "next month",
		"last year", "this year",
	}

	for _, dateRef := range relativeDates {
		if strings.Contains(lowerInput, dateRef) {
			dates = append(dates, dateRef)
		}
	}

	// Absolute date patterns (ISO format, US format)
	datePatterns := []*regexp.Regexp{
		regexp.MustCompile(`\b\d{4}-\d{2}-\d{2}\b`),               // 2024-01-15
		regexp.MustCompile(`\b\d{1,2}/\d{1,2}/\d{4}\b`),           // 01/15/2024
		regexp.MustCompile(`\b(Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)\s+\d{1,2},?\s+\d{4}\b`), // Jan 15, 2024
	}

	for _, regex := range datePatterns {
		matches := regex.FindAllString(inputText, -1)
		dates = append(dates, matches...)
	}

	return a.deduplicateStrings(dates)
}

// extractProviders extracts service provider names from input text.
func (a *IntentDetectionAgent) extractProviders(inputText string) []string {
	providers := []string{}
	lowerInput := strings.ToLower(inputText)

	// Known providers
	knownProviders := []string{
		"gitlab", "youtrack", "openai", "anthropic", "deepseek", "ollama",
		"github", "jira", "confluence",
	}

	for _, provider := range knownProviders {
		if strings.Contains(lowerInput, provider) {
			providers = append(providers, provider)
		}
	}

	return a.deduplicateStrings(providers)
}

// extractStatuses extracts status indicators from input text.
func (a *IntentDetectionAgent) extractStatuses(inputText string) []string {
	statuses := []string{}
	lowerInput := strings.ToLower(inputText)

	// Known status keywords
	knownStatuses := []string{
		"open", "closed", "in-progress", "in progress", "pending",
		"resolved", "done", "failed", "error", "success",
		"active", "inactive", "blocked", "ready", "draft",
	}

	for _, status := range knownStatuses {
		if strings.Contains(lowerInput, status) {
			statuses = append(statuses, status)
		}
	}

	return a.deduplicateStrings(statuses)
}

// calculateConfidenceScores calculates overall confidence scores.
func (a *IntentDetectionAgent) calculateConfidenceScores(intents []models.Intent) map[string]float64 {
	scores := make(map[string]float64)

	if len(intents) > 0 {
		// Primary intent confidence
		scores["primary_intent"] = intents[0].Confidence

		// Overall confidence (average of top 2 intents)
		if len(intents) > 1 {
			scores["overall"] = (intents[0].Confidence + intents[1].Confidence) / 2.0
		} else {
			scores["overall"] = intents[0].Confidence
		}
	} else {
		scores["primary_intent"] = 0.0
		scores["overall"] = 0.0
	}

	return scores
}

// extractInputText extracts input text from agent context.
func (a *IntentDetectionAgent) extractInputText(ctx *models.AgentContext) string {
	// Try to get input from metadata or retrieval context
	// For now, use a placeholder - in production this would come from the request

	// TODO: Define standard location for input text in AgentContext
	// Options:
	// 1. ctx.Metadata.InputText
	// 2. ctx.Retrieval.Query
	// 3. Passed as parameter to Execute()

	// For now, check if there's a query in retrieval context
	if ctx.Retrieval != nil && len(ctx.Retrieval.Queries) > 0 {
		return ctx.Retrieval.Queries[0].QueryString
	}

	// Return empty string if no input found
	return ""
}

// recordAgentRun records the agent execution in the audit trail.
func (a *IntentDetectionAgent) recordAgentRun(ctx *models.AgentContext, duration time.Duration, status string, err error, llmCalls int) {
	run := models.AgentRun{
		Timestamp:  time.Now(),
		AgentID:    a.id,
		Status:     status,
		DurationMS: duration.Milliseconds(),
		KeysWritten: []string{
			"reasoning.intents",
			"reasoning.entities",
			"reasoning.confidence_scores",
		},
	}

	if err != nil {
		run.Error = err.Error()
	}

	if ctx.Audit == nil {
		ctx.Audit = &models.AuditContext{}
	}

	ctx.Audit.AgentRuns = append(ctx.Audit.AgentRuns, run)

	// Also update performance metrics
	if ctx.Diagnostics == nil {
		ctx.Diagnostics = &models.DiagnosticsContext{
			Performance: &models.PerformanceData{},
		}
	}

	if ctx.Diagnostics.Performance.AgentMetrics == nil {
		ctx.Diagnostics.Performance.AgentMetrics = make(map[string]*models.AgentMetrics)
	}

	ctx.Diagnostics.Performance.AgentMetrics[a.id] = &models.AgentMetrics{
		DurationMS: duration.Milliseconds(),
		LLMCalls:   llmCalls,
		Status:     status,
	}
}

// deduplicateStrings removes duplicate strings from a slice.
func (a *IntentDetectionAgent) deduplicateStrings(input []string) []string {
	seen := make(map[string]bool)
	result := []string{}

	for _, item := range input {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}

	return result
}

// shouldUseLLMFallback determines if LLM fallback should be used based on confidence.
func (a *IntentDetectionAgent) shouldUseLLMFallback(confidenceScores map[string]float64) bool {
	// Only use LLM if orchestrator is available
	if a.llmOrchestrator == nil {
		return false
	}

	// Check if primary intent confidence is below threshold
	primaryConfidence, exists := confidenceScores["primary_intent"]
	if !exists {
		return true // No confidence score, use LLM
	}

	return primaryConfidence < a.confidenceThreshold
}

// detectIntentsWithLLM uses LLM to detect intents with fallback chain.
// Returns: intents, highest confidence score, number of LLM calls made, error
func (a *IntentDetectionAgent) detectIntentsWithLLM(ctx context.Context, inputText string) ([]models.Intent, float64, int, error) {
	response := ""
	return a.detectIntentsWithLLMDetailed(ctx, inputText, &response)
}

// detectIntentsWithLLMDetailed uses LLM to detect intents and captures the response.
// Returns: intents, highest confidence score, number of LLM calls made, error
func (a *IntentDetectionAgent) detectIntentsWithLLMDetailed(ctx context.Context, inputText string, llmResponse *string) ([]models.Intent, float64, int, error) {
	if a.llmOrchestrator == nil {
		return nil, 0.0, 0, fmt.Errorf("LLM orchestrator not available")
	}

	// Build prompt for intent detection
	prompt := a.buildIntentDetectionPrompt(inputText)

	// Try deepseek-chat first (cheapest option)
	intents, confidence, err := a.callLLMForIntentDetailed(ctx, prompt, models.TaskTypeIntentClassification, llmResponse)
	llmCalls := 1

	// If deepseek-chat confidence is still low, fallback to gpt-4o-mini
	if err == nil && confidence < a.confidenceThreshold {
		fallbackResponse := ""
		fallbackIntents, fallbackConfidence, fallbackErr := a.callLLMForIntentDetailed(ctx, prompt, models.TaskTypeIntentClassification, &fallbackResponse)
		llmCalls++

		if fallbackErr == nil && fallbackConfidence > confidence {
			*llmResponse = fallbackResponse // Use fallback response
			return fallbackIntents, fallbackConfidence, llmCalls, nil
		}
	}

	return intents, confidence, llmCalls, err
}

// buildIntentDetectionPrompt creates a prompt for LLM intent detection.
func (a *IntentDetectionAgent) buildIntentDetectionPrompt(inputText string) string {
	return fmt.Sprintf(`Analyze this user input and detect the primary intent. Return a JSON response with the intent type and confidence score.

Supported intent types:
- query_commits: Questions about commits, code changes, or version history
- query_issues: Questions about tasks, bugs, or issue tracking
- query_analytics: Requests for statistics, metrics, or trends
- query_status: Questions about project status, health, or progress
- command_action: Execute an action (deploy, restart, update, etc.)
- request_help: Asking for help or documentation
- conversation: General conversation or unclear intent

User input: "%s"

Respond with JSON in this format:
{
  "intent_type": "query_commits",
  "confidence": 0.95
}`, inputText)
}

// callLLMForIntent calls the LLM orchestrator to detect intent.
func (a *IntentDetectionAgent) callLLMForIntent(ctx context.Context, prompt string, taskType models.TaskType) ([]models.Intent, float64, error) {
	response := ""
	return a.callLLMForIntentDetailed(ctx, prompt, taskType, &response)
}

// callLLMForIntentDetailed calls the LLM orchestrator and captures the response.
func (a *IntentDetectionAgent) callLLMForIntentDetailed(ctx context.Context, prompt string, taskType models.TaskType, llmResponse *string) ([]models.Intent, float64, error) {
	// Create LLM request
	req := &services.LLMRequest{
		Prompt:      prompt,
		TaskType:    taskType,
		AgentID:     a.id,
		MaxTokens:   100,
		Temperature: 0.0, // Deterministic for classification
		ContextSize: len(prompt),
		UseCach:     true, // Cache intent classifications
	}

	// Select model (will use deepseek-chat by default for intent classification)
	model, provider, err := a.llmOrchestrator.SelectModel(ctx, req)
	if err != nil {
		return nil, 0.0, fmt.Errorf("failed to select model: %w", err)
	}

	// Call LLM with the prompt
	llmResp, err := a.llmOrchestrator.Call(ctx, req)
	if err != nil {
		return nil, 0.0, fmt.Errorf("LLM call failed: %w", err)
	}

	_ = model    // Selected model logged in orchestrator
	_ = provider // Selected provider logged in orchestrator

	// Store the full response
	responseText := llmResp.Response
	if llmResponse != nil {
		*llmResponse = responseText
	}

	// Parse JSON response
	// Expected format: {"intent_type": "query_commits", "confidence": 0.95}
	var result struct {
		IntentType string  `json:"intent_type"`
		Confidence float64 `json:"confidence"`
	}
	startIdx := strings.Index(responseText, "{")
	endIdx := strings.LastIndex(responseText, "}")
	if startIdx >= 0 && endIdx > startIdx {
		jsonStr := responseText[startIdx : endIdx+1]
		if err := json.Unmarshal([]byte(jsonStr), &result); err == nil {
			// Successfully parsed JSON
			return []models.Intent{
				{
					Type:       result.IntentType,
					Confidence: result.Confidence,
				},
			}, result.Confidence, nil
		}
	}

	// JSON parsing failed, return default with lower confidence
	return []models.Intent{
		{
			Type:       "conversation",
			Confidence: 0.6,
		},
	}, 0.6, nil
}

// generateClarificationQuestions creates clarification questions for ambiguous intents.
func (a *IntentDetectionAgent) generateClarificationQuestions(intents []models.Intent, confidenceScores map[string]float64) []models.ClarificationQuestion {
	questions := []models.ClarificationQuestion{}

	// Check for ambiguity conditions
	if len(intents) == 0 {
		return questions // No intents, no questions
	}

	primaryConfidence := confidenceScores["primary_intent"]

	// Case 1: Very low confidence on primary intent (< 0.5)
	if primaryConfidence < 0.5 {
		question := models.ClarificationQuestion{
			Question:        "I'm not quite sure what you're asking for. Could you provide more details?",
			PossibleIntents: []string{intents[0].Type},
			Reason:          fmt.Sprintf("Low confidence (%.2f) in primary intent detection", primaryConfidence),
		}

		if len(intents) > 1 {
			question.PossibleIntents = append(question.PossibleIntents, intents[1].Type)
			question.Options = []string{
				a.getIntentDescription(intents[0].Type),
				a.getIntentDescription(intents[1].Type),
				"Something else",
			}
		}

		questions = append(questions, question)
		return questions
	}

	// Case 2: Multiple intents with similar confidence (difference < 0.2)
	if len(intents) >= 2 {
		confidenceDiff := intents[0].Confidence - intents[1].Confidence

		if confidenceDiff < 0.2 {
			intentTypes := []string{intents[0].Type, intents[1].Type}
			options := []string{
				a.getIntentDescription(intents[0].Type),
				a.getIntentDescription(intents[1].Type),
			}

			// Add third option if available
			if len(intents) >= 3 && (intents[1].Confidence-intents[2].Confidence) < 0.15 {
				intentTypes = append(intentTypes, intents[2].Type)
				options = append(options, a.getIntentDescription(intents[2].Type))
			}

			question := models.ClarificationQuestion{
				Question:        a.buildClarificationQuestion(intentTypes),
				Options:         options,
				PossibleIntents: intentTypes,
				Reason:          fmt.Sprintf("Multiple intents with similar confidence (diff: %.2f)", confidenceDiff),
			}

			questions = append(questions, question)
		}
	}

	return questions
}

// getIntentDescription returns a human-readable description of an intent type.
func (a *IntentDetectionAgent) getIntentDescription(intentType string) string {
	descriptions := map[string]string{
		"query_commits":   "You want to see recent code changes or commits",
		"query_issues":    "You want to check on tasks, bugs, or issues",
		"query_analytics": "You want to see statistics or metrics",
		"query_status":    "You want to check the status or health of a project",
		"command_action":  "You want to execute an action (deploy, restart, etc.)",
		"request_help":    "You need help or documentation",
		"conversation":    "General conversation",
	}

	if desc, ok := descriptions[intentType]; ok {
		return desc
	}

	return intentType
}

// buildClarificationQuestion creates a question based on possible intents.
func (a *IntentDetectionAgent) buildClarificationQuestion(intentTypes []string) string {
	if len(intentTypes) == 2 {
		return fmt.Sprintf("Are you asking about %s or %s?",
			a.getIntentShortName(intentTypes[0]),
			a.getIntentShortName(intentTypes[1]))
	}

	if len(intentTypes) == 3 {
		return fmt.Sprintf("Are you asking about %s, %s, or %s?",
			a.getIntentShortName(intentTypes[0]),
			a.getIntentShortName(intentTypes[1]),
			a.getIntentShortName(intentTypes[2]))
	}

	return "What specifically would you like to know?"
}

// getIntentShortName returns a short name for an intent type.
func (a *IntentDetectionAgent) getIntentShortName(intentType string) string {
	shortNames := map[string]string{
		"query_commits":   "code changes",
		"query_issues":    "tasks/issues",
		"query_analytics": "statistics",
		"query_status":    "project status",
		"command_action":  "executing an action",
		"request_help":    "help/documentation",
		"conversation":    "general info",
	}

	if name, ok := shortNames[intentType]; ok {
		return name
	}

	return strings.ReplaceAll(intentType, "_", " ")
}

// GetMetadata returns agent metadata (implements MetadataProvider).
func (a *IntentDetectionAgent) GetMetadata() domainservices.AgentMetadata {
	return domainservices.AgentMetadata{
		ID:          a.id,
		Name:        "Intent Detection Agent",
		Description: "Detects user intent and extracts entities using rule-based classification with LLM fallback for low-confidence cases",
		Version:     "1.1.0",
		Author:      "ADK LLM Proxy",
		Tags:        []string{"intent", "classification", "entity-extraction", "nlp", "llm-fallback"},
		Dependencies: []string{}, // First agent - no dependencies
	}
}

// GetCapabilities returns agent capabilities (implements CapabilitiesProvider).
func (a *IntentDetectionAgent) GetCapabilities() domainservices.AgentCapabilities {
	return domainservices.AgentCapabilities{
		SupportsParallelExecution: false, // First agent - must run first
		SupportsRetry:             true,
		RequiresLLM:               false, // Rule-based primary, LLM optional for low-confidence fallback
		IsDeterministic:           false, // With LLM fallback, output may vary
		EstimatedDuration:         100,   // ~50ms rule-based + up to 2s for LLM fallback
	}
}
